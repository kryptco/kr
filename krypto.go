package krssh

import (
	"errors"
	"fmt"
	"github.com/GoKillers/libsodium-go/cryptosecretbox"
)

type SymmetricSecretKey struct {
	Bytes []byte
}

func GenSymmetricSecretKey() (key SymmetricSecretKey, err error) {
	keyBytes, err := RandNBytes(uint(secretbox.CryptoSecretBoxKeyBytes()))
	if err != nil {
		return
	}
	key = SymmetricSecretKey{
		Bytes: keyBytes,
	}
	return
}

func Seal(message []byte, key SymmetricSecretKey) (ciphertext []byte, err error) {
	iv, err := RandNBytes(uint(secretbox.CryptoSecretBoxNonceBytes()))
	if err != nil {
		err = errors.New("error generating IV: " + err.Error())
	}

	ciphertext, ret := secretbox.CryptoSecretBoxEasy(message, iv, key.Bytes)
	if ret != 0 {
		err = errors.New(fmt.Sprintf("error encrypting: %d", ret))
		return
	}

	ciphertext = append(iv, ciphertext...)
	return
}

func Open(ciphertext []byte, key SymmetricSecretKey) (message []byte, err error) {
	iv := ciphertext[0:secretbox.CryptoSecretBoxNonceBytes()]
	ciphertext = ciphertext[secretbox.CryptoSecretBoxNonceBytes():]

	message, ret := secretbox.CryptoSecretBoxOpenEasy(ciphertext, iv, key.Bytes)
	if ret != 0 {
		err = errors.New(fmt.Sprintf("error decrypting: %d", ret))
		return
	}
	return
}
