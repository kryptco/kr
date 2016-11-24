package main

import (
	"encoding/base64"

	"github.com/agrinman/kr"
)

var testWire, _ = base64.StdEncoding.DecodeString("AAAAB3NzaC1yc2EAAAADAQABAAACAQCz5NxDjQgHtjnI4ilK7drJyZEzZyDCRrQgQWkKTTgKm/zH1K/3ygs63UW4zJB2sGR/UVhTJ8f11jyiRvSdEMzb47ERCxlVwA96C5i2Ha5JzxcV+ERY4uqkxIjfbsDdvdwewb3kMRYqVcPeBXWnwZ7VAkGWNZI3KP2CtSh/fsJ3xDSztzxtqZOWlPfRO4W0ClQvZpHkYRAoQH+7XHZFh1B/lw6hlSQmT5+q+WBkG1YGQUuFCyIZmnJat8YJAQkXOWBuOqxkWRQsPxd8LZrP87Ut32Lmz3oy0nxlRI1H56ebzj/vw/xpwntISg1XlsniXols75CLjs/N5DCM+KcxhE7Y49dui53/TQgc8SRYRIUy00c6Wll7QrqT5OvcGDi8kKGGWiWz1hquyT4Yb3ULWxf7sTTeVt+Ldrbxf3J3orFVaHkgI5HTduTbu45y96yPutJncX8CwoPI/l3pZ2684EXGwltHeUN1REqJwRMzaDc0A0ok3vFN5epoaBixhygWW1kK4CkzZ7UQ9XWz99ba7EVArz79tJZPLG7M4y8OIPSyoRZDcaCDBNyRIofiyAJlfi8zV7MAN6f2xjj1w8jfzp9FgX79K6DTp4tBJDIkum4YfUlKne7KHINLY2xMggdi6dkDucaEX0n1e1TrMe8CpCPzak1dDf99q3XNwVJaThZkpw==")
var testMe kr.Profile = kr.Profile{
	SSHWirePublicKey: testWire,
	Email:            "kevin@krypt.co",
}

type mockedEnclaveClient struct {
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
	response = &kr.MeResponse{
		Me: testMe,
	}
	return
}
func (ec *mockedEnclaveClient) GetCachedMe() (me *kr.Profile) {
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
