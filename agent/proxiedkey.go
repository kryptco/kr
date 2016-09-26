package main

import (
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"github.com/agrinman/krssh"
	"io"
	"log"
)

//	Implements crypto.Signer by requesting signatures from phone
type ProxiedKey struct {
	crypto.PublicKey
	publicKeyFingerprint []byte
	enclaveClient        EnclaveClientI
}

func (pk *ProxiedKey) Public() crypto.PublicKey {
	return pk.PublicKey
}

func (pk *ProxiedKey) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	command := getLastCommand()
	if command != nil {
		log.Println("command:", *command)
	}
	request := krssh.SignRequest{
		PublicKeyFingerprint: pk.publicKeyFingerprint,
		Digest:               digest,
		Command:              command,
	}
	response, err := pk.enclaveClient.RequestSignature(request)
	if err != nil {
		log.Println("error requesting signature:", err)
		return
	}
	if response != nil {
		if response.Error != nil {
			err = errors.New("Enclave signature error: " + *response.Error)
			return
		}
		if response.Signature != nil {
			signature = *response.Signature
			return
		}
		err = errors.New("No enclave signature in response")
		return
	} else {
		err = errors.New("No response from enclave")
		return
	}

	err = errors.New("not yet implemented")
	return
}

func PKDERToProxiedKey(enclaveClient EnclaveClientI, pkDER []byte) (proxiedKey crypto.Signer, err error) {
	pk, err := krssh.ParseRsaAsn1(pkDER)
	if err != nil {
		return
	}

	publicKeyFingerprint := sha256.Sum256(pkDER)

	proxiedKey = &ProxiedKey{
		publicKeyFingerprint: publicKeyFingerprint[:],
		enclaveClient:        enclaveClient,
		PublicKey:            pk,
	}
	return
}
