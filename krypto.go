package kr

import (
	"crypto/rand"
	"fmt"
	"github.com/kryptco/go-crypto/blake2b"
	"golang.org/x/crypto/nacl/box"
)

const (
	HEADER_CIPHERTEXT = iota
	HEADER_WRAPPED_KEY
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
	c := nonceAndCiphertext[24:]
	m, ret = box.Open(m, c, &n, &pkArr, &skArr)
	if !ret {
		err = fmt.Errorf("Open failed")
		return
	}
	return
}

func sodiumBoxSealOpen(c, pk, sk []byte) (m []byte, err error) {
	if len(c) < 32 || len(pk) != 32 || len(sk) != 32 {
		err = fmt.Errorf("invalid argument passed to sodium")
		return
	}
	var ephemeralPk [32]byte
	copy(ephemeralPk[:], c[:32])

	var skArr [32]byte
	copy(skArr[:], sk)

	noncePreimage := append(ephemeralPk[:], pk...)
	n := blake2b.Sum192(noncePreimage)

	m, ok := box.Open(m, c[32:], &n, &ephemeralPk, &skArr)
	if !ok {
		err = fmt.Errorf("verify failed")
		return
	}
	return
}

//	https://download.libsodium.org/doc/public-key_cryptography/sealed_boxes.html
func sodiumBoxSeal(m, pk []byte) (c []byte, err error) {
	//	protect against bindings panicking
	if len(m) == 0 || len(pk) == 0 {
		err = fmt.Errorf("empty argument passed to sodium")
		return
	}
	ephemeralPk, ephemeralSk, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return
	}
	noncePreimage := append(ephemeralPk[:], pk...)
	n := blake2b.Sum192(noncePreimage)

	var pkArr [32]byte
	copy(pkArr[:], pk)

	c = box.Seal(c, m, &n, &pkArr, ephemeralSk)
	c = append(ephemeralPk[:], c...)
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
	pkArr, skArr, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return
	}
	pk = pkArr[:]
	sk = skArr[:]
	return
}
