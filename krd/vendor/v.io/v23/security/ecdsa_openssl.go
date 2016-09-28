// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build openssl

// OpenSSL's libcrypto may have faster implementations of ECDSA signing and
// verification on some architectures (not amd64 after Go 1.6 which includes
// https://go-review.googlesource.com/#/c/8968/). This file enables use
// of libcrypto's implementation of ECDSA operations in those situations.

package security

// #cgo pkg-config: libcrypto
// #include <openssl/bn.h>
// #include <openssl/ec.h>
// #include <openssl/ecdsa.h>
// #include <openssl/err.h>
// #include <openssl/objects.h>
// #include <openssl/opensslv.h>
// #include <openssl/x509.h>
//
// void openssl_init_locks();
// EC_KEY* openssl_d2i_EC_PUBKEY(const unsigned char* data, long len, unsigned long* e);
// EC_KEY* openssl_d2i_ECPrivateKey(const unsigned char* data, long len, unsigned long* e);
// ECDSA_SIG* openssl_ECDSA_do_sign(const unsigned char* data, int len, EC_KEY* key, unsigned long *e);
import "C"

import (
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"unsafe"

	"v.io/v23/verror"
)

var (
	errOpenSSL          = verror.Register(pkgPath+".errOpenSSL", verror.NoRetry, "{1:}{2:} OpenSSL error ({3}): {4} in {5}:{6}")
	errUnsupportedCurve = verror.Register(pkgPath+".errUnsupportedCurve", verror.NoRetry, "{1:}{2:} elliptic curve {3} is not supported")
	opensslLocks        []sync.RWMutex
)

func init() {
	C.ERR_load_crypto_strings()
	opensslLocks = make([]sync.RWMutex, C.CRYPTO_num_locks())
	C.openssl_init_locks()
}

//export openssl_lock
func openssl_lock(mode, n C.int) {
	l := &opensslLocks[int(n)]
	if (mode & C.CRYPTO_LOCK) == 0 {
		if (mode & C.CRYPTO_READ) == 0 {
			l.Unlock()
		} else {
			l.RUnlock()
		}
	} else {
		if (mode & C.CRYPTO_READ) == 0 {
			l.Lock()
		} else {
			l.RLock()
		}
	}
}

type opensslECPublicKey struct {
	k   *C.EC_KEY
	h   Hash
	der []byte
}

func (k *opensslECPublicKey) MarshalBinary() ([]byte, error) {
	cpy := make([]byte, len(k.der))
	copy(cpy, k.der)
	return cpy, nil
}
func (k *opensslECPublicKey) String() string { return publicKeyString(k) }
func (k *opensslECPublicKey) hash() Hash     { return k.h }
func (k *opensslECPublicKey) verify(digest []byte, signature *Signature) bool {
	sig := C.ECDSA_SIG_new()
	sig.r = C.BN_bin2bn(uchar(signature.R), C.int(len(signature.R)), sig.r)
	sig.s = C.BN_bin2bn(uchar(signature.S), C.int(len(signature.S)), sig.s)
	status := C.ECDSA_do_verify(uchar(digest), C.int(len(digest)), sig, k.k)
	C.ECDSA_SIG_free(sig)
	return status == 1
}

func newOpenSSLPublicKey(golang *ecdsa.PublicKey) (PublicKey, error) {
	der, err := x509.MarshalPKIXPublicKey(golang)
	if err != nil {
		return nil, err
	}
	return unmarshalPublicKeyImpl(der)
}

type opensslSigner struct {
	k *C.EC_KEY
}

func (k *opensslSigner) sign(data []byte) (r, s *big.Int, err error) {
	var errno C.ulong
	sig := C.openssl_ECDSA_do_sign(uchar(data), C.int(len(data)), k.k, &errno)
	if sig == nil {
		return nil, nil, opensslMakeError(errno)
	}
	var (
		rlen = (int(C.BN_num_bits(sig.r)) + 7) / 8
		slen = (int(C.BN_num_bits(sig.s)) + 7) / 8
		buf  []byte
	)
	if rlen > slen {
		buf = make([]byte, rlen)
	} else {
		buf = make([]byte, slen)
	}
	r = big.NewInt(0).SetBytes(buf[0:int(C.BN_bn2bin(sig.r, uchar(buf)))])
	s = big.NewInt(0).SetBytes(buf[0:int(C.BN_bn2bin(sig.s, uchar(buf)))])
	C.ECDSA_SIG_free(sig)
	return r, s, nil
}

func newOpenSSLSigner(golang *ecdsa.PrivateKey) (Signer, error) {
	der, err := x509.MarshalECPrivateKey(golang)
	if err != nil {
		return nil, err
	}
	pubkey, err := newOpenSSLPublicKey(&golang.PublicKey)
	if err != nil {
		return nil, err
	}
	var errno C.ulong
	k := C.openssl_d2i_ECPrivateKey(uchar(der), C.long(len(der)), &errno)
	if k == nil {
		return nil, opensslMakeError(errno)
	}
	impl := &opensslSigner{k}
	runtime.SetFinalizer(impl, func(k *opensslSigner) { C.EC_KEY_free(k.k) })
	return &ecdsaSigner{
		sign: func(data []byte) (r, s *big.Int, err error) {
			return impl.sign(data)
		},
		pubkey: pubkey,
		impl:   impl,
	}, nil
}

func opensslMakeError(errno C.ulong) error {
	return verror.New(errOpenSSL, nil, errno, C.GoString(C.ERR_func_error_string(errno)), C.GoString(C.ERR_lib_error_string(errno)), C.GoString(C.ERR_reason_error_string(errno)))
}

func uchar(b []byte) *C.uchar {
	if len(b) == 0 {
		return nil
	}
	return (*C.uchar)(unsafe.Pointer(&b[0]))
}

func openssl_version() string {
	return fmt.Sprintf("%v (CFLAGS:%v)", C.GoString(C.SSLeay_version(C.SSLEAY_VERSION)), C.GoString(C.SSLeay_version(C.SSLEAY_CFLAGS)))
}

func openssl_hash_for_key(k *C.EC_KEY) (Hash, error) {
	switch nid := C.EC_GROUP_get_curve_name(C.EC_KEY_get0_group(k)); nid {
	case C.NID_secp224r1, C.NID_X9_62_prime256v1:
		return SHA256Hash, nil
	case C.NID_secp384r1:
		return SHA384Hash, nil
	case C.NID_secp521r1:
		return SHA512Hash, nil
	default:
		var h Hash
		return h, verror.New(errUnsupportedCurve, nil, C.GoString(C.OBJ_nid2sn(C.int(nid))))
	}
}

func newInMemoryECDSASignerImpl(key *ecdsa.PrivateKey) (Signer, error) {
	return newOpenSSLSigner(key)
}

func newECDSAPublicKeyImpl(key *ecdsa.PublicKey) PublicKey {
	if key, err := newOpenSSLPublicKey(key); err == nil {
		return key
	}
	return newGoStdlibPublicKey(key)
}

func unmarshalPublicKeyImpl(der []byte) (PublicKey, error) {
	var errno C.ulong
	k := C.openssl_d2i_EC_PUBKEY(uchar(der), C.long(len(der)), &errno)
	if k == nil {
		return nil, opensslMakeError(errno)
	}
	h, err := openssl_hash_for_key(k)
	if err != nil {
		return nil, err
	}
	dercpy := make([]byte, len(der))
	copy(dercpy, der)
	ret := &opensslECPublicKey{k, h, dercpy}
	runtime.SetFinalizer(ret, func(k *opensslECPublicKey) { C.EC_KEY_free(k.k) })
	return ret, nil
}
