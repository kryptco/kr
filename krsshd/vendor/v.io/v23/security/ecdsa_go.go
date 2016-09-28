// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !openssl

package security

import (
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"

	"v.io/v23/verror"
)

func newInMemoryECDSASignerImpl(key *ecdsa.PrivateKey) (Signer, error) {
	return newGoStdlibSigner(key)
}

func newECDSAPublicKeyImpl(key *ecdsa.PublicKey) PublicKey {
	return newGoStdlibPublicKey(key)
}

func unmarshalPublicKeyImpl(bytes []byte) (PublicKey, error) {
	key, err := x509.ParsePKIXPublicKey(bytes)
	if err != nil {
		return nil, err
	}
	switch v := key.(type) {
	case *ecdsa.PublicKey:
		return newGoStdlibPublicKey(v), nil
	default:
		return nil, verror.New(errUnrecognizedKey, nil, fmt.Sprintf("%T", key))
	}
}
