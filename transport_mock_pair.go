package kr

import (
	"encoding/base64"
	"sync"
)

type ImmediatePairTransport struct {
	NoopTransport
	sync.Mutex
	Keys map[string][]byte
}

func (t *ImmediatePairTransport) Setup(ps *PairingSecret) (err error) {
	return
}

func (t *ImmediatePairTransport) Read(notifier *Notifier, ps *PairingSecret) (ciphertexts [][]byte, err error) {
	t.Lock()
	defer t.Unlock()
	if t.Keys == nil {
		t.Keys = map[string][]byte{}
	}
	if _, ok := t.Keys[base64.StdEncoding.EncodeToString(ps.WorkstationPublicKey)]; !ok {
		var key []byte
		key, err = RandNBytes(32)
		if err != nil {
			return
		}
		t.Keys[base64.StdEncoding.EncodeToString(ps.WorkstationPublicKey)] = key
		wrappedKey, wrapErr := WrapKey(t.Keys[base64.StdEncoding.EncodeToString(ps.WorkstationPublicKey)], ps.WorkstationPublicKey)
		if wrapErr != nil {
			err = wrapErr
			return
		}
		ciphertexts = [][]byte{wrappedKey}
	}
	return
}

//	store first key, but send multiple wrapped keys
type MultiPairTransport struct {
	NoopTransport
	sync.Mutex
	paired bool
	SymKey []byte
}

func (t *MultiPairTransport) Read(notifier *Notifier, ps *PairingSecret) (ciphertexts [][]byte, err error) {
	t.Lock()
	defer t.Unlock()
	for _ = range []int{1, 2, 3} {
		if !t.paired {
			t.SymKey, err = RandNBytes(32)
			if err != nil {
				return
			}
			t.paired = true
		}
		wrappedKey, wrapErr := WrapKey(t.SymKey, ps.WorkstationPublicKey)
		if wrapErr != nil {
			err = wrapErr
			return
		}
		ciphertexts = append(ciphertexts, wrappedKey)
	}
	return
}
