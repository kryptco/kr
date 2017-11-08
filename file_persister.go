package kr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type FilePersister struct {
	PairingDir string
	SSHDir     string
}

func (fp FilePersister) SaveMe(me Profile) (err error) {
	path := filepath.Join(fp.PairingDir, "me")
	if err != nil {
		return
	}
	profileJson, err := json.Marshal(me)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(path, profileJson, 0700)
	return
}

func (fp FilePersister) LoadMe() (me Profile, err error) {
	path := filepath.Join(fp.PairingDir, "me")
	if err != nil {
		return
	}

	profileJson, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	err = json.Unmarshal(profileJson, &me)
	if err != nil {
		return
	}
	if len(me.SSHWirePublicKey) == 0 {
		err = fmt.Errorf("missing public key")
		return
	}
	return
}

func (fp FilePersister) DeleteMe() (err error) {
	path := filepath.Join(fp.PairingDir, "me")
	if err != nil {
		return
	}
	err = os.Remove(path)
	return
}

func (fp FilePersister) SaveMySSHPubKey(me Profile) (err error) {
	authString, err := me.AuthorizedKeyString()
	if err != nil {
		return
	}
	err = ioutil.WriteFile(filepath.Join(fp.SSHDir, ID_KRYPTONITE_FILENAME), []byte(authString), 0700)
	return
}

func (fp FilePersister) LoadPairing() (pairingSecret *PairingSecret, err error) {
	path := filepath.Join(fp.PairingDir, PAIRING_FILENAME)
	if err != nil {
		return
	}
	pairingJson, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	var pp persistedPairing
	err = json.Unmarshal(pairingJson, &pp)
	if err != nil {
		return
	}
	ps := pairingFromPersisted(&pp)
	pairingSecret = ps
	return
}
func (fp FilePersister) SavePairing(pairingSecret *PairingSecret) (err error) {
	path := filepath.Join(fp.PairingDir, PAIRING_FILENAME)
	if err != nil {
		return
	}
	pairingJson, err := json.Marshal(pairingToPersisted(pairingSecret))
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path, pairingJson, os.FileMode(0700))
	return
}
func (fp FilePersister) DeletePairing() (pairingSecret *PairingSecret, err error) {
	path := filepath.Join(fp.PairingDir, PAIRING_FILENAME)
	if err != nil {
		return
	}
	err = os.Remove(path)
	return
}
