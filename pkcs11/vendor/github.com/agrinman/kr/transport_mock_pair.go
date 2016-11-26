package kr

import (
	"sync"
)

type ImmediatePairTransport struct {
	NoopTransport
	sync.Mutex
	paired bool
	SymKey []byte
}

func (t *ImmediatePairTransport) Read(ps *PairingSecret) (ciphertexts [][]byte, err error) {
	t.Lock()
	defer t.Unlock()
	if !t.paired {
		t.paired = true
		t.SymKey, err = RandNBytes(32)
		if err != nil {
			return
		}
		wrappedKey, wrapErr := WrapKey(t.SymKey, ps.WorkstationPublicKey)
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

func (t *MultiPairTransport) Read(ps *PairingSecret) (ciphertexts [][]byte, err error) {
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
