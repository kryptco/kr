package main

import (
	"testing"

	"github.com/agrinman/kr"
)

type mockedEnclaveClient struct {
	kr.Transport
	kr.Persister
	*testing.T
	pairingSecret  *kr.PairingSecret
	timeout        bool
	pairedOverride bool
}

func (ec *mockedEnclaveClient) Pair() (ps *kr.PairingSecret, err error) {
	ps, err = kr.GeneratePairingSecret()
	ec.pairingSecret = ps
	return
}

func (ec *mockedEnclaveClient) RequestMe(longTimeout bool) (response *kr.MeResponse, err error) {
	if ec.timeout {
		err = ErrTimeout
		return
	}
	me, _, _ := kr.TestMe(ec.T)
	response = &kr.MeResponse{
		Me: me,
	}
	return
}
func (ec *mockedEnclaveClient) GetCachedMe() (me *kr.Profile) {
	testMe, _, _ := kr.TestMe(ec.T)
	me = &testMe
	return
}
func (ec *mockedEnclaveClient) RequestSignature(kr.SignRequest) (response *kr.SignResponse, err error) {
	if ec.timeout {
		err = ErrTimeout
		return
	}
	errStr := "unimplemented"
	response = &kr.SignResponse{
		Error: &errStr,
	}
	return
}
func (ec *mockedEnclaveClient) RequestList(kr.ListRequest) (response *kr.ListResponse, err error) {
	if ec.timeout {
		err = ErrTimeout
		return
	}
	response = &kr.ListResponse{}
	return
}
func (ec *mockedEnclaveClient) RequestNoOp() (err error) {
	if ec.timeout {
		err = ErrTimeout
		return
	}
	return
}
func (ec *mockedEnclaveClient) IsPaired() (paired bool) {
	return ec.pairedOverride || (ec.pairingSecret != nil && ec.pairingSecret.IsPaired())
}
func (ec *mockedEnclaveClient) Start() (err error) {
	return
}
func (ec *mockedEnclaveClient) Stop() (err error) {
	return
}
func (ec *mockedEnclaveClient) Unpair() {
	ec.pairingSecret = nil
	return
}
