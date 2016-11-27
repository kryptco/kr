package kr

import (
	"fmt"
	"sync"
)

type MemoryPersister struct {
	sync.Mutex
	me      *Profile
	pairing *PairingSecret
}

func (mp *MemoryPersister) SaveMe(me Profile) (err error) {
	mp.Lock()
	defer mp.Unlock()
	mp.me = &me
	return
}
func (mp *MemoryPersister) LoadMe() (me Profile, err error) {
	mp.Lock()
	defer mp.Unlock()
	if mp.me == nil {
		err = fmt.Errorf("no me saved")
		return
	}
	me = *mp.me
	return
}
func (mp *MemoryPersister) DeleteMe() (err error) {
	mp.Lock()
	defer mp.Unlock()
	mp.me = nil
	return
}
func (mp *MemoryPersister) SaveMySSHPubKey(me Profile) (err error) {
	return
}
func (mp *MemoryPersister) LoadPairing() (pairingSecret *PairingSecret, err error) {
	mp.Lock()
	defer mp.Unlock()
	if mp.pairing == nil {
		err = fmt.Errorf("no pairing saved")
		return
	}
	pairingSecret = mp.pairing
	return
}
func (mp *MemoryPersister) SavePairing(pairingSecret *PairingSecret) (err error) {
	mp.Lock()
	defer mp.Unlock()
	mp.pairing = pairingSecret
	return
}
func (mp *MemoryPersister) DeletePairing() (pairingSecret *PairingSecret, err error) {
	mp.Lock()
	defer mp.Unlock()
	mp.pairing = nil
	return
}
