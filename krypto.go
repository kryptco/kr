package kr

import (
	"crypto/rand"
	"fmt"
	//	anonymous box
	"github.com/GoKillers/libsodium-go/cryptobox"
	//	authenticated box
	"golang.org/x/crypto/nacl/box"
)

const (
	HEADER_CIPHERTEXT = iota
	HEADER_WRAPPED_KEY
	//	TODO: verify that encrypting with a sodium public key does not reveal the public key
	HEADER_WRAPPED_PUBLIC_KEY
)

func sodiumBox(m, pk, sk []byte) (c []byte, err error) {
	var n [24]byte
	_, err = rand.Read(n[:])
	if err != nil {
		return
	}
	if len(pk) != 32 || len(sk) != 32 {
		err = fmt.Errorf("incorrect key length")
		return
	}
	var pkArr [32]byte
	copy(pkArr[:], pk)

	var skArr [32]byte
	copy(skArr[:], sk)

	c = box.Seal(c, m, &n, &pkArr, &skArr)
	c = append(n[:], c...)
	return
}

func sodiumBoxOpen(nonceAndCiphertext, pk, sk []byte) (m []byte, err error) {
	if len(nonceAndCiphertext) < 24 {
		err = fmt.Errorf("CryptoBox nonce too small")
		return
	}
	var n [24]byte
	copy(n[:], nonceAndCiphertext[:24])

	if len(pk) != 32 || len(sk) != 32 {
		err = fmt.Errorf("incorrect key length")
		return
	}
	var pkArr [32]byte
	copy(pkArr[:], pk)

	var skArr [32]byte
	copy(skArr[:], sk)

	var ret bool
	c := nonceAndCiphertext[cryptobox.CryptoBoxNonceBytes():]
	m, ret = box.Open(m, c, &n, &pkArr, &skArr)
	if !ret {
		err = fmt.Errorf("Open failed")
		return
	}
	return
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

func sodiumBoxSeal(m, pk []byte) (c []byte, err error) {
	//	protect against bindings panicking
	if len(m) == 0 || len(pk) == 0 {
		err = fmt.Errorf("empty argument passed to sodium")
		return
	}
	c, ret := cryptobox.CryptoBoxSeal(m, pk)
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
	return
}

func WrapKey(pkToWrap, pk []byte) (c []byte, err error) {
	encryptedKey, err := sodiumBoxSeal(pkToWrap, pk)
	if err != nil {
		return
	}

	c = append([]byte{HEADER_WRAPPED_PUBLIC_KEY}, encryptedKey...)
	return
}

func GenKeyPair() (pk []byte, sk []byte, err error) {
	var ret int
	sk, pk, ret = cryptobox.CryptoBoxKeyPair()
	if ret != 0 {
		err = fmt.Errorf("nonzero sodium return status: %d", ret)
		return
	}
	return
}
