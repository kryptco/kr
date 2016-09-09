package krssh

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
	return RandNBase62(32)
}

func Rand128Base62() (encodedRand string, err error) {
	return RandNBase62(16)
}

func RandNBase62(n uint) (encodedRand string, err error) {
	randBuf, err := RandNBytes(n)
	_, err = rand.Read(randBuf)
	if err != nil {
		return
	}
	encodedRand = basex.Base62StdEncoding.EncodeToString(randBuf)
	return
}
