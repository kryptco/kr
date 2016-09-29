package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestControlServerPair(t *testing.T) {
	cs := ControlServer{&mockedEnclaveClient{}}
	pairRequest, err := http.NewRequest("PUT", "/pair", nil)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	cs.handlePair(recorder, pairRequest)
	recorder.Result()
}
