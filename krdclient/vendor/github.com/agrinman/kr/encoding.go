package kr

import (
	"crypto/rsa"
	"math/big"

	"golang.org/x/crypto/ssh"
)

func SSHWireRSAPublicKeyToRSAPublicKey(wire []byte) (pk *rsa.PublicKey, err error) {
	//	parse RSA SSH wire format
	//  https://github.com/golang/crypto/blob/077efaa604f994162e3307fafe5954640763fc08/ssh/keys.go#L302
	var w struct {
		//	assume type RSA
		Type string
		E    *big.Int
		N    *big.Int
	}
	if err = ssh.Unmarshal(wire, &w); err != nil {
		return
	}
	pk = &rsa.PublicKey{
		N: w.N,
		E: int(w.E.Int64()),
	}
	return
}
