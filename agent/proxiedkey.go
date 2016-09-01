package main

import (
	"crypto"
	"crypto/x509"
	"errors"
	"io"
	"log"
)

//	Implements crypto.Signer by requesting signatures from phone
type ProxiedKey struct {
	crypto.PublicKey
}

func (pk *ProxiedKey) Public() crypto.PublicKey {
	return pk.PublicKey
}

func (pk *ProxiedKey) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	log.Printf("trying to sign with %t\n", pk.PublicKey)
	err = errors.New("not yet implemented")
	return
}

func PKDERToProxiedKey(pkDER []byte) (proxiedKey crypto.Signer, err error) {
	pk, err := x509.ParsePKIXPublicKey(pkDER)
	if err != nil {
		return
	}

	proxiedKey = &ProxiedKey{
		PublicKey: pk,
	}
	return
}
