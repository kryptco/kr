package kr

import (
	"bytes"
	"testing"
)

func TestGenWrapEncDec(t *testing.T) {
	var workstationName = "test.workstation.name"
	ps, err := GeneratePairingSecret(&workstationName)
	if err != nil {
		t.Fatal(err)
	}
	if ps.WorkstationName != "test.workstation.name" {
		t.Fatal("WorkstationName is wrong")
	}

	ps, err = GeneratePairingSecret(nil)
	if err != nil {
		t.Fatal(err)
	}
	if ps.WorkstationName != MachineName() {
		t.Fatal("WorkstationName is wrong")
	}
	sessionKey, err := RandNBytes(32)
	if err != nil {
		t.Fatal(err)
	}

	encryptedKey, err := WrapKey(sessionKey, ps.WorkstationPublicKey)
	if err != nil {
		t.Fatal(err)
	}

	remaining, didUnwrap, err := ps.UnwrapKeyIfPresent(encryptedKey)
	if err != nil {
		t.Fatal(err)
	}
	if remaining != nil {
		t.Fatal()
	}
	if !didUnwrap {
		t.Fatal()
	}
	if !bytes.Equal(sessionKey, *ps.EnclavePublicKey) {
		t.Fatal("SymmetricSecretKey wrong")
	}

	msg, err := RandNBytes(129)
	if err != nil {
		t.Fatal(err)
	}
	ctxt, err := ps.EncryptMessage(msg)
	if err != nil {
		t.Fatal(err)
	}

	remainingCtxt, didUnwrap, err := ps.UnwrapKeyIfPresent(ctxt)
	if remainingCtxt == nil {
		t.Fatal("should have remaining ciphertext")
	}
	if didUnwrap {
		t.Fatal("was not wrapped key")
	}
	if err != nil {
		t.Fatal(err)
	}

	ptxt, err := ps.DecryptMessage(*remainingCtxt)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(*ptxt, msg) {
		t.Fatal("decrypt failed")
	}
}
