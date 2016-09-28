package kr

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/GoKillers/libsodium-go/cryptobox"
)

const (
	HEADER_CIPHERTEXT = iota
	HEADER_WRAPPED_KEY
)
const AES_KEY_NUM_BYTES = 32

type SymmetricSecretKey struct {
	Bytes []byte
}

func sodiumBoxSealOpen(c, pk, sk []byte) (m []byte, err error) {
	//	protect against bindings panicking
	if len(c) == 0 || len(pk) == 0 || len(sk) == 0 {
		err = fmt.Errorf("empty argument passed to sodium")
		return
	}
	m, ret := cryptobox.CryptoBoxSealOpen(c, pk, sk)
	if ret != 0 {
		err = fmt.Errorf("nonzero sodium return status: %d", ret)
		return
	}
	return
}

func UnwrapKey(c, pk, sk []byte) (key []byte, err error) {
	key, err = sodiumBoxSealOpen(c, pk, sk)
	if err != nil {
		return
	}
	//TODO: check key length here
	if len(key) != AES_KEY_NUM_BYTES {
		err = fmt.Errorf("incorrect key length of %d expected %d", len(key), AES_KEY_NUM_BYTES)
		return
	}
	return
}

func GenSymmetricSecretKey() (key SymmetricSecretKey, err error) {
	keyBytes, err := RandNBytes(AES_KEY_NUM_BYTES)
	if err != nil {
		return
	}
	key = SymmetricSecretKey{
		Bytes: keyBytes,
	}
	return
}

func SymmetricSecretKeyFromBytes(bytes []byte) (key *SymmetricSecretKey, err error) {
	if len(bytes) != AES_KEY_NUM_BYTES {
		err = errors.New(fmt.Sprintf("aes key must have %d bytes, %d provided", AES_KEY_NUM_BYTES, len(bytes)))
		return
	}
	key = &SymmetricSecretKey{bytes}
	return
}

func Seal(message []byte, key SymmetricSecretKey) (ciphertext []byte, err error) {
	aesCipher, err := aes.NewCipher(key.Bytes)
	if err != nil {
		err = errors.New("error creating AES cipher: " + err.Error())
		return
	}
	message = PKCS7Pad(aesCipher.BlockSize(), message)

	iv, err := RandNBytes(uint(aesCipher.BlockSize()))
	if err != nil {
		err = errors.New("error generating IV: " + err.Error())
		return
	}

	cbcEncryptor := cipher.NewCBCEncrypter(aesCipher, iv)

	ciphertext = make([]byte, len(message))
	cbcEncryptor.CryptBlocks(ciphertext, message)
	ciphertext = append(iv, ciphertext...)

	macFunc := hmac.New(sha256.New, key.Bytes)
	macFunc.Write(ciphertext)
	computedMAC := macFunc.Sum(nil)

	ciphertext = append(ciphertext, computedMAC...)

	return
}

func Open(ciphertext []byte, key SymmetricSecretKey) (message []byte, err error) {
	aesCipher, err := aes.NewCipher(key.Bytes)
	if err != nil {
		err = errors.New("error creating AES cipher: " + err.Error())
		return
	}

	macFunc := hmac.New(sha256.New, key.Bytes)

	encryptedData := ciphertext[:len(ciphertext)-macFunc.Size()]
	mac := ciphertext[len(ciphertext)-macFunc.Size():]

	macFunc.Write(encryptedData)
	computedMAC := macFunc.Sum(nil)

	if !hmac.Equal(computedMAC, mac) {
		err = errors.New("invalid HMAC")
		return
	}

	iv := encryptedData[0:aesCipher.BlockSize()]
	cipherBlocks := encryptedData[aesCipher.BlockSize():]

	message = make([]byte, len(cipherBlocks))

	cbcDecryptor := cipher.NewCBCDecrypter(aesCipher, iv)
	cbcDecryptor.CryptBlocks(message, cipherBlocks)

	message, err = PKCS7Unpad(message)
	if err != nil {
		err = errors.New("error PKCS7Unpad: " + err.Error())
		return
	}
	return
}

func PKCS7Pad(blockSize int, message []byte) []byte {
	numPadding := blockSize - len(message)%blockSize
	padding := bytes.Repeat([]byte{byte(numPadding)}, numPadding)
	return append(message, padding...)
}

func PKCS7Unpad(paddedMessage []byte) (message []byte, err error) {
	if len(paddedMessage) == 0 {
		err = errors.New("Empty message is not padded")
		return
	}

	numPadding := int(paddedMessage[len(paddedMessage)-1])
	if numPadding > len(paddedMessage) {
		err = errors.New("Invalid padding, larger than total message")
		return
	}

	message = paddedMessage[:len(paddedMessage)-numPadding]
	return
}
