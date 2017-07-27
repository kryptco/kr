package krd

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/kryptco/kr"
)

func TestPair(t *testing.T) {
	transport := &kr.ImmediatePairTransport{}
	ec := NewTestEnclaveClient(transport)
	ps := PairClient(t, ec)
	defer ec.Stop()

	if ps.EnclavePublicKey == nil || !bytes.Equal(*ps.EnclavePublicKey, transport.Keys[base64.StdEncoding.EncodeToString(ps.WorkstationPublicKey)]) {
		t.Fatal()
	}
}

func TestMultiPair(t *testing.T) {
	transport := &kr.MultiPairTransport{}
	ec := NewTestEnclaveClient(transport)
	ps := PairClient(t, ec)
	defer ec.Stop()

	if ps.EnclavePublicKey == nil || !bytes.Equal(*ps.EnclavePublicKey, transport.SymKey) {
		t.Fatal()
	}
}

func TestRemoteUnpair(t *testing.T) {
	transport := &kr.ResponseTransport{T: t}
	ec := NewTestEnclaveClient(transport)
	PairClient(t, ec)
	defer ec.Stop()

	transport.RemoteUnpair()

	go ec.RequestMe(kr.MeRequest{}, false)

	kr.TrueBefore(t, func() bool {
		return !ec.IsPaired()
	}, time.Now().Add(time.Second))
}

func TestMe(t *testing.T) {
	transport := &kr.ResponseTransport{T: t}
	ec := NewTestEnclaveClient(transport)
	testMeSuccess(t, ec)
}

func TestMeAlert(t *testing.T) {
	transport := &kr.ResponseTransport{T: t, RespondToAlertOnly: true}
	ec := NewTestEnclaveClientShortTimeouts(transport)
	testMeSuccess(t, ec)
}

func TestMeTimeout(t *testing.T) {
	transport := &kr.ResponseTransport{T: t, DoNotRespond: true}
	ec := NewTestEnclaveClientShortTimeouts(transport)

	PairClient(t, ec)
	defer ec.Stop()

	me, err := ec.RequestMe(kr.MeRequest{}, true)
	if me != nil && err != ErrTimeout {
		t.Fatal("expected nil response or timeout")
	}
}

func testMeSuccess(t *testing.T, ec EnclaveClientI) {
	PairClient(t, ec)
	defer ec.Stop()

	me, err := ec.RequestMe(kr.MeRequest{}, true)
	if err != nil {
		t.Fatal(err)
	}
	testMe, _, _ := kr.TestMe(t)
	if !me.Me.Equal(testMe) {
		t.Fatal("unexpected profile")
	}
	cachedMe := ec.GetCachedMe()
	if cachedMe == nil || !cachedMe.Equal(testMe) {
		t.Fatal("bad cached profile")
	}
}

func TestSignature(t *testing.T) {
	transport := &kr.ResponseTransport{T: t}
	ec := NewTestEnclaveClient(transport)
	testSignatureSuccess(t, ec)
}

func TestSignatureAlert(t *testing.T) {
	transport := &kr.ResponseTransport{T: t, RespondToAlertOnly: true}
	ec := NewTestEnclaveClientShortTimeouts(transport)
	testSignatureSuccess(t, ec)
}

func TestSignatureTimeout(t *testing.T) {
	transport := &kr.ResponseTransport{T: t, DoNotRespond: true}
	ec := NewTestEnclaveClientShortTimeouts(transport)
	resp, _, err := testSignature(t, ec)
	if resp != nil && err != ErrTimeout {
		t.Fatal("expected nil response or timeout")
	}
}

func TestSignatureAckDelayWithResponse(t *testing.T) {
	transport := &kr.ResponseTransport{T: t, Ack: true, SendAfterHalfAckDelay: true}
	ec := NewTestEnclaveClientShortTimeouts(transport)
	testSignatureSuccess(t, ec)
}

func TestSignatureAckDelayWithoutResponse(t *testing.T) {
	transport := &kr.ResponseTransport{T: t, Ack: true, SendAfterHalfAckDelay: false}
	ec := NewTestEnclaveClientShortTimeouts(transport)
	resp, _, err := testSignature(t, ec)
	if resp != nil && err != ErrTimeout {
		t.Fatal("expected nil response or timeout")
	}
}

func testSignatureSuccess(t *testing.T, ec EnclaveClientI) {
	_, sk, _ := kr.TestMe(t)
	signResponse, digest, err := testSignature(t, ec)
	if err != nil {
		t.Fatal(err)
	}
	if signResponse == nil || signResponse.Signature == nil || rsa.VerifyPKCS1v15(&sk.PublicKey, crypto.SHA256, digest[:], *signResponse.Signature) != nil {
		t.Fatal("invalid sign response")
	}
}

func testSignature(t *testing.T, ec EnclaveClientI) (resp *kr.SignResponse, digest [32]byte, err error) {
	PairClient(t, ec)
	defer ec.Stop()

	msg, err := kr.RandNBytes(32)
	if err != nil {
		t.Fatal(err)
	}
	digest = sha256.Sum256(msg)

	me, _, _ := kr.TestMe(t)
	fp := me.PublicKeyFingerprint()
	signResponse, _, err := ec.RequestSignature(kr.SignRequest{
		PublicKeyFingerprint: fp[:],
		Data:                 digest[:],
	},
		nil,
	)
	return signResponse, digest, err
}

func TestNoOp(t *testing.T) {
	transport := &kr.ResponseTransport{T: t}
	ec := NewTestEnclaveClient(transport)
	PairClient(t, ec)
	defer ec.Stop()

	go ec.RequestNoOp()

	kr.TrueBefore(t, func() bool {
		return transport.GetSentNoOps() > 0
	}, time.Now().Add(time.Second))
}
