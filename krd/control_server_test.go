package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agrinman/kr"
)

func TestControlServerPair(t *testing.T) {
	cs := ControlServer{&mockedEnclaveClient{}}
	pairRequest, err := http.NewRequest("PUT", "/pair", nil)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	cs.handlePair(recorder, pairRequest)
	resp := recorder.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatal("non-200 status")
	}
	var pairingSecret kr.PairingSecret
	err = json.NewDecoder(resp.Body).Decode(&pairingSecret)
	if err != nil {
		t.Fatal(err)
	}
}

func TestControlServerMe(t *testing.T) {
	ec := &mockedEnclaveClient{}
	cs := ControlServer{ec}
	request, err := kr.NewRequest()
	if err != nil {
		t.Fatal(err)
	}
	request.MeRequest = &kr.MeRequest{}

	meRequest, err := request.HTTPRequest()
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	cs.handleEnclave(recorder, meRequest)
	resp := recorder.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatal("expected 404, not paired")
	}

	ec.pairedOverride = true
	recorder = httptest.NewRecorder()
	cs.handleEnclave(recorder, meRequest)
	resp = recorder.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatal("non-200 status")
	}

	var meResponse kr.Response
	err = json.NewDecoder(resp.Body).Decode(&meResponse)
	if err != nil {
		t.Fatal(err)
	}
	if !meResponse.MeResponse.Me.Equal(testMe) {
		t.Fatal("profiles unequal")
	}
}

func TestControlServerSign(t *testing.T) {
	ec := &mockedEnclaveClient{}
	cs := ControlServer{ec}
	request, err := kr.NewRequest()
	if err != nil {
		t.Fatal(err)
	}
	request.SignRequest = &kr.SignRequest{}

	signRequest, err := request.HTTPRequest()
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	cs.handleEnclave(recorder, signRequest)
	resp := recorder.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatal("expected 404, not paired")
	}

	ec.pairedOverride = true
	recorder = httptest.NewRecorder()
	cs.handleEnclave(recorder, signRequest)
	resp = recorder.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatal("non-200 status")
	}

	var signResponse kr.Response
	err = json.NewDecoder(resp.Body).Decode(&signResponse)
	if err != nil {
		t.Fatal(err)
	}
	if *signResponse.SignResponse.Error != "unimplemented" {
		t.Fatal("unexpected mocked response")
	}
}

func TestControlServerPing(t *testing.T) {
	cs := ControlServer{&mockedEnclaveClient{}}
	pingRequest, err := http.NewRequest("GET", "/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	cs.handlePair(recorder, pingRequest)
	resp := recorder.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatal("non-200 status")
	}
}
