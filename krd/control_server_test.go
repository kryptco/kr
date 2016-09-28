package main

import (
	"github.com/agrinman/kr"
	"net/http/httptest"
	"testing"
)

func TestControlServerPair(t *testing.T) {
	pairingSecret, err := kr.GeneratePairingSecret()
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
