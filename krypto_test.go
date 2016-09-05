package krssh

import (
	"bytes"
	"testing"
)

func TestSealOpen(t *testing.T) {
	key, err := GenSymmetricSecretKey()
	if err != nil {
		t.Fatal(err)
	}
	msg, err := RandNBytes(31)
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
		t.Fatalf("decryption does not match, \n%v \n!= \n%v\n", openedMsg, msg)
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

func TestPKCS7PadUnpad(t *testing.T) {
	for blockSize := 1; blockSize < 128; blockSize++ {
		msg, err := RandNBytes(32)
		if err != nil {
			t.Fatal(err)
		}
		unpadded, err := PKCS7Unpad(PKCS7Pad(blockSize, msg))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(msg, unpadded) {
			t.Fatalf("unpadded does not match, \n%v \n!= \n%v\n", msg, unpadded)
		}
	}
}

func TestPKCS7Pad(t *testing.T) {
	blockSize := 16
	msg, err := RandNBytes(15)
	if err != nil {
		t.Fatal(err)
	}
	padded := PKCS7Pad(blockSize, msg)
	if !bytes.Equal(padded, append(msg, 0x01)) {
		t.Fatal("padding incorrect")
	}
}
