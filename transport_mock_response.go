package kr

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

var SHORT_ACK_DELAY = 500 * time.Millisecond

type ResponseTransport struct {
	ImmediatePairTransport
	*testing.T
	sync.Mutex
	responses             [][]byte
	sentNoOps             int
	RespondToAlertOnly    bool
	DoNotRespond          bool
	Ack                   bool
	SendAfterHalfAckDelay bool
}

func (t *ResponseTransport) respondToMessage(ps *PairingSecret, m []byte, ackSent bool) (err error) {
	if t.DoNotRespond {
		return
	}
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
	if request.SendACK && !ackSent && t.Ack {
		response.AckResponse = &AckResponse{}
		if t.SendAfterHalfAckDelay {
			go func() {
				<-time.After(SHORT_ACK_DELAY / 2)
				t.Lock()
				defer t.Unlock()
				t.respondToMessage(ps, m, true)
			}()
		}
	} else {
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
			sig, err := sk.Sign(rand.Reader, request.SignRequest.Data, crypto.SHA256)
			if err != nil {
				t.T.Fatal(err)
			}
			response.SignResponse = &SignResponse{
				Signature: &sig,
			}
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
	err = t.respondToMessage(ps, m, false)
	return
}

func (t *ResponseTransport) PushAlert(ps *PairingSecret, alertText string, message []byte) (err error) {
	t.Lock()
	defer t.Unlock()
	err = t.respondToMessage(ps, message, false)
	return
}

func (t *ResponseTransport) Read(notifier *Notifier, ps *PairingSecret) (ciphertexts [][]byte, err error) {
	pairCiphertexts, err := t.ImmediatePairTransport.Read(notifier, ps)
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

func (t *ResponseTransport) RemoteUnpair() {
	t.Lock()
	defer t.Unlock()
	unpairResponse := Response{
		UnpairResponse: &UnpairResponse{},
	}
	respJson, err := json.Marshal(unpairResponse)
	if err != nil {
		t.Fatal(err)
	}
	t.responses = append(t.responses, respJson)
}
