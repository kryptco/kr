package main

import (
	"crypto/rand"
	"github.com/keybase/saltpack/encoding/basex"
)

func RandNBytes(n uint) (randBytes []byte, err error) {
	randBytes = make([]byte, n)
	_, err = rand.Read(randBytes)
	return
}

func Rand256Base62() (encodedRand string, err error) {
	randBuf, err := RandNBytes(32)
	_, err = rand.Read(randBuf)
	if err != nil {
		return
	}
	encodedRand = basex.Base62StdEncoding.EncodeToString(randBuf)
	return
}
