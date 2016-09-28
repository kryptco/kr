// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package security

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"math/big"

	"v.io/v23/verror"
)

// NewECDSAPublicKey creates a PublicKey object that uses the ECDSA algorithm and the provided ECDSA public key.
func NewECDSAPublicKey(key *ecdsa.PublicKey) PublicKey {
	return newECDSAPublicKeyImpl(key)
}

func newGoStdlibPublicKey(key *ecdsa.PublicKey) PublicKey {
	return &ecdsaPublicKey{key}
}

type ecdsaPublicKey struct {
	key *ecdsa.PublicKey
}

func (pk *ecdsaPublicKey) MarshalBinary() ([]byte, error) { return x509.MarshalPKIXPublicKey(pk.key) }
func (pk *ecdsaPublicKey) String() string                 { return publicKeyString(pk) }
func (pk *ecdsaPublicKey) verify(digest []byte, sig *Signature) bool {
	var r, s big.Int
	return ecdsa.Verify(pk.key, digest, r.SetBytes(sig.R), s.SetBytes(sig.S))
}

func (pk *ecdsaPublicKey) hash() Hash {
	if nbits := pk.key.Curve.Params().BitSize; nbits <= 160 {
		return SHA1Hash
	} else if nbits <= 256 {
		return SHA256Hash
	} else if nbits <= 384 {
		return SHA384Hash
	} else {
		return SHA512Hash
	}
}

// NewInMemoryECDSASigner creates a Signer that uses the provided ECDSA private
// key to sign messages.  This private key is kept in the clear in the memory
// of the running process.
func NewInMemoryECDSASigner(key *ecdsa.PrivateKey) Signer {
	// TODO(ashankar): Change this function to return an error
	// and not panic.
	signer, err := newInMemoryECDSASignerImpl(key)
	if err != nil {
		panic(err)
	}
	return signer
}

func newGoStdlibSigner(key *ecdsa.PrivateKey) (Signer, error) {
	sign := func(data []byte) (r, s *big.Int, err error) {
		return ecdsa.Sign(rand.Reader, key, data)
	}
	return &ecdsaSigner{sign: sign, pubkey: newGoStdlibPublicKey(&key.PublicKey)}, nil
}

// NewECDSASigner creates a Signer that uses the provided function to sign
// messages.
func NewECDSASigner(key *ecdsa.PublicKey, sign func(data []byte) (r, s *big.Int, err error)) Signer {
	return &ecdsaSigner{sign: sign, pubkey: NewECDSAPublicKey(key)}
}

type ecdsaSigner struct {
	sign   func(data []byte) (r, s *big.Int, err error)
	pubkey PublicKey
	impl   interface{} // Object to hold on to for garbage collection
}

func (c *ecdsaSigner) Sign(purpose, message []byte) (Signature, error) {
	hash := c.pubkey.hash()
	if message = messageDigest(hash, purpose, message, c.pubkey); message == nil {
		return Signature{}, verror.New(errSignCantHash, nil, hash)
	}
	r, s, err := c.sign(message)
	if err != nil {
		return Signature{}, err
	}
	return Signature{
		Purpose: purpose,
		Hash:    hash,
		R:       r.Bytes(),
		S:       s.Bytes(),
	}, nil
}

func (c *ecdsaSigner) PublicKey() PublicKey {
	return c.pubkey
}
