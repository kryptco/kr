package kr

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/ssh"
)

type Profile struct {
	SSHWirePublicKey []byte `json:"rsa_public_key_wire"`
	Email            string `json:"email"`
}

func (p Profile) AuthorizedKeyString() string {
	return "ssh-rsa " + base64.StdEncoding.EncodeToString(p.SSHWirePublicKey) + " " + p.Email
}

func (p Profile) SSHPublicKey() (pk ssh.PublicKey, err error) {
	return ssh.ParsePublicKey(p.SSHWirePublicKey)
}

func (p Profile) RSAPublicKey() (pk *rsa.PublicKey, err error) {
	return SSHWireRSAPublicKeyToRSAPublicKey(p.SSHWirePublicKey)
}

func (p Profile) PublicKeyFingerprint() []byte {
	digest := sha256.Sum256(p.SSHWirePublicKey)
	return digest[:]
}

func (p Profile) Equal(other Profile) bool {
	return bytes.Equal(p.SSHWirePublicKey, other.SSHWirePublicKey) && p.Email == other.Email
}

func PersistMe(me Profile) (err error) {
	path, err := KrDirFile("me")
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

func LoadMe() (me Profile, err error) {
	path, err := KrDirFile("me")
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

func DeleteMe() (err error) {
	path, err := KrDirFile("me")
	if err != nil {
		return
	}
	err = os.Remove(path)
	return
}
