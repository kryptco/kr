// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package security

import (
	"crypto/md5"
	"encoding"
	"fmt"
	"v.io/v23/verror"
)

var (
	errUnrecognizedKey = verror.Register(pkgPath+".errUnrecognizedKey", verror.NoRetry, "{1:}{2:}unrecognized PublicKey type({3}){:_}")
)

// PublicKey represents a public key using an unspecified algorithm.
//
// MarshalBinary returns the DER-encoded PKIX representation of the public key,
// while UnmarshalPublicKey creates a PublicKey object from the marshaled bytes.
//
// String returns a human-readable representation of the public key.
type PublicKey interface {
	encoding.BinaryMarshaler
	fmt.Stringer

	// hash returns a cryptographic hash function whose security strength is
	// appropriate for creating message digests to sign with this public key.
	// For example, an ECDSA public key with a 512-bit curve would require a
	// 512-bit hash function, whilst a key with a 256-bit curve would be
	// happy with a 256-bit hash function.
	hash() Hash

	// verify returns true iff signature was created by the corresponding
	// private key when signing the provided message digest (obtained by
	// the messageDigest function).
	verify(digest []byte, signature *Signature) bool
}

func publicKeyString(pk PublicKey) string {
	bytes, err := pk.MarshalBinary()
	if err != nil {
		return fmt.Sprintf("<invalid public key: %v>", err)
	}
	const hextable = "0123456789abcdef"
	hash := md5.Sum(bytes)
	var repr [md5.Size * 3]byte
	for i, v := range hash {
		repr[i*3] = hextable[v>>4]
		repr[i*3+1] = hextable[v&0x0f]
		repr[i*3+2] = ':'
	}
	return string(repr[:len(repr)-1])
}

// UnmarshalPublicKey returns a PublicKey object from the DER-encoded PKIX represntation of it
// (typically obtianed via PublicKey.MarshalBinary).
func UnmarshalPublicKey(bytes []byte) (PublicKey, error) {
	return unmarshalPublicKeyImpl(bytes)
}
