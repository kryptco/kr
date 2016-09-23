package main

import (
	"github.com/agrinman/krssh"
	"golang.org/x/crypto/ssh"
)

type mockedEnclaveClient struct {
	pairingSecret *krssh.PairingSecret
}

func (ec *mockedEnclaveClient) Pair(ps krssh.PairingSecret) {
	ec.pairingSecret = &ps
}

func (ec *mockedEnclaveClient) RequestMe() (response *krssh.MeResponse, err error) {
	return
}
func (ec *mockedEnclaveClient) RequestMeSigner() (signer ssh.Signer, err error) {
	return
}
func (ec *mockedEnclaveClient) GetCachedMe() (me *krssh.Profile) {
	return
}
func (ec *mockedEnclaveClient) GetCachedMeSigner() (signer ssh.Signer) {
	return
}
func (ec *mockedEnclaveClient) RequestSignature(krssh.SignRequest) (response *krssh.SignResponse, err error) {
	return
}
func (ec *mockedEnclaveClient) RequestList(krssh.ListRequest) (response *krssh.ListResponse, err error) {
	return
}
