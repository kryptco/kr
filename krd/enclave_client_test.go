package main

import (
	"github.com/agrinman/kr"
	"golang.org/x/crypto/ssh"
)

type mockedEnclaveClient struct {
	pairingSecret *kr.PairingSecret
}

func (ec *mockedEnclaveClient) Pair() (ps kr.PairingSecret, err error) {
	ps, err = kr.GeneratePairingSecret()
	return
}

func (ec *mockedEnclaveClient) RequestMe() (response *kr.MeResponse, err error) {
	return
}
func (ec *mockedEnclaveClient) RequestMeSigner() (signer ssh.Signer, err error) {
	return
}
func (ec *mockedEnclaveClient) GetCachedMe() (me *kr.Profile) {
	return
}
func (ec *mockedEnclaveClient) GetCachedMeSigner() (signer ssh.Signer) {
	return
}
func (ec *mockedEnclaveClient) RequestSignature(kr.SignRequest) (response *kr.SignResponse, err error) {
	return
}
func (ec *mockedEnclaveClient) RequestList(kr.ListRequest) (response *kr.ListResponse, err error) {
	return
}
