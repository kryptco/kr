package kr

import (
	"crypto/rand"
	"crypto/rsa"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"
)

var testSK *rsa.PrivateKey
var testPK ssh.PublicKey
var testMe *Profile
var testMeMutex sync.Mutex

func TestMe(t *testing.T) (profile Profile, sk *rsa.PrivateKey, pk ssh.PublicKey) {
	testMeMutex.Lock()
	defer testMeMutex.Unlock()
	var err error
	if testMe == nil {
		testSK, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		testPK, err = ssh.NewPublicKey(&testSK.PublicKey)
		if err != nil {
			t.Fatal(err)
		}
		testMe = &Profile{
			SSHWirePublicKey: testPK.Marshal(),
			Email:            "kevin@krypt.co",
		}
	}
	return *testMe, testSK, testPK
}
