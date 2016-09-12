package main

import (
	"bitbucket.org/kryptco/krssh"
	"net/http/httptest"
	"testing"
)

func TestControlServerPair(t *testing.T) {
	pairingSecret, err := krssh.GeneratePairingSecret()
	if err != nil {
		t.Fatal(err)
	}

	pairRequest, err := pairingSecret.HTTPRequest()
	if err != nil {
		t.Fatal(err)
	}

	cs := ControlServer{&mockedEnclaveClient{}}
	recorder := httptest.NewRecorder()
	cs.handlePair(recorder, pairRequest)
	recorder.Result()
}
