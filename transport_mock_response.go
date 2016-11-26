package kr

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"encoding/json"
	"sync"
	"testing"
)

type ResponseTransport struct {
	ImmediatePairTransport
	*testing.T
	sync.Mutex
	responses          [][]byte
	sentNoOps          int
	RespondToAlertOnly bool
}

func (t *ResponseTransport) respondToMessage(ps *PairingSecret, m []byte) (err error) {
	me, sk, _ := TestMe(t.T)
	var request Request
	err = json.Unmarshal(m, &request)
	if err != nil {
		t.T.Fatal(err)
	}
	if request.IsNoOp() {
		t.sentNoOps += 1
		return
	}
	response := Response{
		RequestID: request.RequestID,
	}
	if request.MeRequest != nil {
		response.MeResponse = &MeResponse{
			Me: me,
		}
	}
	if request.SignRequest != nil {
		fp := me.PublicKeyFingerprint()
		if !bytes.Equal(request.SignRequest.PublicKeyFingerprint, fp[:]) {
			t.Fatal("wrong public key")
		}
		sig, err := sk.Sign(rand.Reader, request.SignRequest.Digest, crypto.SHA256)
		if err != nil {
			t.T.Fatal(err)
		}
		response.SignResponse = &SignResponse{
			Signature: &sig,
		}
	}
	respJson, err := json.Marshal(response)
	if err != nil {
		t.T.Fatal(err)
	}
	t.responses = append(t.responses, respJson)
	return
}

func (t *ResponseTransport) SendMessage(ps *PairingSecret, m []byte) (err error) {
	t.Lock()
	defer t.Unlock()
	if t.RespondToAlertOnly {
		return
	}
	err = t.respondToMessage(ps, m)
	return
}

func (t *ResponseTransport) PushAlert(ps *PairingSecret, alertText string, message []byte) (err error) {
	t.Lock()
	defer t.Unlock()
	err = t.respondToMessage(ps, message)
	return
}

func (t *ResponseTransport) Read(ps *PairingSecret) (ciphertexts [][]byte, err error) {
	pairCiphertexts, err := t.ImmediatePairTransport.Read(ps)
	ciphertexts = append(ciphertexts, pairCiphertexts...)
	t.Lock()
	defer t.Unlock()
	for _, responseBytes := range t.responses {
		ctxt, err := ps.EncryptMessage(responseBytes)
		if err != nil {
			t.T.Fatal(err)
		}
		ciphertexts = append(ciphertexts, ctxt)
	}
	t.responses = [][]byte{}
	return
}

func (t *ResponseTransport) GetSentNoOps() int {
	t.Lock()
	defer t.Unlock()
	return t.sentNoOps
}
