package main

import (
	"bitbucket.org/kryptco/krssh"
	"golang.org/x/crypto/ssh"
)

type mockedEnclaveClient struct{}

func (ec *mockedEnclaveClient) Pair(krssh.PairingSecret)
func (ec *mockedEnclaveClient) RequestMe() (*krssh.MeResponse, error)
func (ec *mockedEnclaveClient) RequestMeSigner() (ssh.Signer, error)
func (ec *mockedEnclaveClient) GetCachedMe() *krssh.Profile
func (ec *mockedEnclaveClient) GetCachedMeSigner() ssh.Signer
func (ec *mockedEnclaveClient) RequestSignature(krssh.SignRequest) (*krssh.SignResponse, error)
func (ec *mockedEnclaveClient) RequestList(krssh.ListRequest) (*krssh.ListResponse, error)
