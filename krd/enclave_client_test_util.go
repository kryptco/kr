package krd

import (
	"testing"
	"time"

	"fmt"
	"github.com/kryptco/kr"
	"github.com/op/go-logging"
	"net"
)

func NewTestEnclaveClient(transport kr.Transport) EnclaveClientI {
	return UnpairedEnclaveClient(
		transport,
		&kr.MemoryPersister{},
		nil,
		kr.SetupLogging("test", logging.INFO, false),
		nil,
	)
}

func NewTestEnclaveClientShortTimeouts(transport kr.Transport) EnclaveClientI {
	shortTimeouts := Timeouts{
		Me: TimeoutPhases{
			Alert: 100 * time.Millisecond,
			Fail:  200 * time.Millisecond,
		},
		Pair: TimeoutPhases{
			Alert: 100 * time.Millisecond,
			Fail:  200 * time.Millisecond,
		},
		Sign: TimeoutPhases{
			Alert: 100 * time.Millisecond,
			Fail:  200 * time.Millisecond,
		},
		ACKDelay: kr.SHORT_ACK_DELAY,
	}

	ec := UnpairedEnclaveClient(
		transport,
		&kr.MemoryPersister{},
		&shortTimeouts,
		kr.SetupLogging("test", logging.INFO, false),
		nil,
	)
	return ec
}

var listener net.Listener

func NewLocalUnixServer(t *testing.T) (ec EnclaveClientI, cs *ControlServer) {
	transport := &kr.ResponseTransport{T: t}
	ec = NewTestEnclaveClient(transport)
	cs = &ControlServer{ec, kr.SetupLogging("test", logging.INFO, false)}

	if listener == nil {
		l, err := kr.DaemonListen()
		if err != nil {
			t.Fatal(fmt.Errorf("DaemonListen() failure: %s.  Try stopping krd and running tests again", "doofis"))
		}
		listener = l
	}

	go func() {
		err := cs.HandleControlHTTP(listener)
		if err != nil {
			t.Fatal(err)
		}
	}()
	return
}

func NewLocalUnixServerWithListener(t *testing.T, listener net.Listener) (ec EnclaveClientI, cs *ControlServer) {
	transport := &kr.ResponseTransport{T: t}
	ec = NewTestEnclaveClient(transport)
	cs = &ControlServer{ec, kr.SetupLogging("test", logging.INFO, false)}

	go func() {
		err := cs.HandleControlHTTP(listener)
		if err != nil {
			t.Fatal(err)
		}
	}()
	return
}

func PairClient(t *testing.T, client EnclaveClientI) (ps *kr.PairingSecret) {
	err := client.Start()
	if err != nil {
		t.Fatal(err)
	}
	var pairingOptions kr.PairingOptions
	ps, err = client.Pair(pairingOptions)
	if err != nil {
		t.Fatal(err)
	}
	go client.RequestMe(kr.MeRequest{}, true)
	kr.TrueBefore(t, client.IsPaired, time.Now().Add(time.Second))
	return
}
