package main

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/agrinman/kr"
)

func NewTestEnclaveClient(transport kr.Transport) EnclaveClientI {
	return UnpairedEnclaveClient(
		transport,
		&kr.MemoryPersister{},
	)
}

func TestPair(t *testing.T) {
	transport := &kr.ImmediatePairTransport{}
	ec := NewTestEnclaveClient(transport)
	ps := pairClient(t, ec)
	defer ec.Stop()

	if ps.SymmetricSecretKey == nil || !bytes.Equal(*ps.SymmetricSecretKey, transport.SymKey) {
		t.Fatal()
	}
}

func TestMultiPair(t *testing.T) {
	transport := &kr.MultiPairTransport{}
	ec := NewTestEnclaveClient(transport)
	ps := pairClient(t, ec)
	defer ec.Stop()

	if ps.SymmetricSecretKey == nil || !bytes.Equal(*ps.SymmetricSecretKey, transport.SymKey) {
		t.Fatal()
	}
}

func TestMe(t *testing.T) {
	transport := &kr.ResponseTransport{T: t}
	ec := NewTestEnclaveClient(transport)
	pairClient(t, ec)
	defer ec.Stop()

	me, err := ec.RequestMe(true)
	if err != nil {
		t.Fatal(err)
	}
	testMe, _, _ := kr.TestMe(t)
	if !me.Me.Equal(testMe) {
		t.Fatal("unexpected profile")
	}
}

func TestSignature(t *testing.T) {
	transport := &kr.ResponseTransport{T: t}
	ec := NewTestEnclaveClient(transport)
	pairClient(t, ec)
	defer ec.Stop()

	msg, err := kr.RandNBytes(32)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(msg)

	me, sk, _ := kr.TestMe(t)
	fp := me.PublicKeyFingerprint()

	signResponse, err := ec.RequestSignature(kr.SignRequest{
		PublicKeyFingerprint: fp[:],
		Digest:               digest[:],
	})
	if err != nil {
		t.Fatal(err)
	}
	if signResponse == nil || signResponse.Signature == nil || rsa.VerifyPKCS1v15(&sk.PublicKey, crypto.SHA256, digest[:], *signResponse.Signature) != nil {
		t.Fatal("invalid sign response")
	}
}

func pairClient(t *testing.T, client EnclaveClientI) (ps *kr.PairingSecret) {
	err := client.Start()
	if err != nil {
		t.Fatal(err)
	}
	ps, err = client.Pair()
	if err != nil {
		t.Fatal(err)
	}
	go client.RequestMe(true)
	kr.TrueBefore(t, client.IsPaired, time.Now().Add(time.Second))
	return
}
