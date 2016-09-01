package main

import (
	"bytes"
	"testing"
)

func TestSealOpen(t *testing.T) {
	key, err := GenSymmetricSecretKey()
	if err != nil {
		t.Fatal(err)
	}
	msg, err := RandNBytes(32)
	if err != nil {
		t.Fatal(err)
	}
	c, err := Seal(msg, key)
	if err != nil {
		t.Fatal(err)
	}

	openedMsg, err := Open(c, key)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(openedMsg, msg) != 0 {
		t.Fatal("decryption does not match")
	}
}

func TestSealChangeOpen(t *testing.T) {
	key, err := GenSymmetricSecretKey()
	if err != nil {
		t.Fatal(err)
	}
	msg, err := RandNBytes(32)
	if err != nil {
		t.Fatal(err)
	}
	c, err := Seal(msg, key)
	if err != nil {
		t.Fatal(err)
	}

	c[0] ^= byte(1)

	_, err = Open(c, key)
	if err == nil {
		t.Fatal("decryption should fail")
	}
}
