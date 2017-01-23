package kr

import (
	"bytes"
	"testing"

	"golang.org/x/crypto/ssh"
)

var rsaProfile = Profile{
	SSHWirePublicKey: pubWire,
	Email:            "hello@krypt.co",
}

func TestRSAProfile(t *testing.T) {
	authKey, err := rsaProfile.AuthorizedKeyString()
	if err != nil {
		t.Fatal(err)
	}
	key1, email, _, _, err := ssh.ParseAuthorizedKey([]byte(authKey))
	if err != nil {
		t.Fatal(err)
	}

	if email != rsaProfile.Email {
		t.Fatal("email wrong")
	}

	pk, err := rsaProfile.RSAPublicKey()
	if err != nil {
		t.Fatal(err)
	}
	key2, err := ssh.NewPublicKey(pk)
	if err != nil {
		t.Fatal(err)
	}

	key3, err := rsaProfile.SSHPublicKey()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(key1.Marshal(), key2.Marshal()) {
		t.Fatal("authorized key != public key")
	}
	if !bytes.Equal(key1.Marshal(), key3.Marshal()) {
		t.Fatal("authorized key != ssh public key")
	}
}
