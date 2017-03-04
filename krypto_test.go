package kr

import (
	"bytes"
	"testing"
)

func TestSealOpen(t *testing.T) {
	wsPk, wsSk, err := GenKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	ePk, eSk, err := GenKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	msg, err := RandNBytes(31)
	if err != nil {
		t.Fatal(err)
	}

	c, err := sodiumBox(append([]byte(nil), msg...), wsPk, eSk)
	if err != nil {
		t.Fatal(err)
	}

	openedMsg, err := sodiumBoxOpen(c, ePk, wsSk)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(openedMsg, msg) != 0 {
		t.Fatalf("decryption does not match, \n%v \n!= \n%v\n", openedMsg, msg)
	}
}

func TestSealChangeOpen(t *testing.T) {
	wsPk, wsSk, err := GenKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	ePk, eSk, err := GenKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	msg, err := RandNBytes(31)
	if err != nil {
		t.Fatal(err)
	}

	c, err := sodiumBox(msg, wsPk, eSk)
	if err != nil {
		t.Fatal(err)
	}

	c[len(c)-1] ^= byte(1)

	_, err = sodiumBoxOpen(c, ePk, wsSk)
	if err == nil {
		t.Fatal("decryption should fail")
	}
}
