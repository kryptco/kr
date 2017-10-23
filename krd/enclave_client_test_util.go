package krd

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kryptco/kr"
	"github.com/op/go-logging"
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
	shortTimeouts := kr.Timeouts{
		Me: kr.TimeoutPhases{
			Alert: 100 * time.Millisecond,
			Fail:  200 * time.Millisecond,
		},
		Pair: kr.TimeoutPhases{
			Alert: 100 * time.Millisecond,
			Fail:  200 * time.Millisecond,
		},
		Sign: kr.TimeoutPhases{
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

func NewLocalUnixServer(t *testing.T) (ec EnclaveClientI, cs *ControlServer, unixFile string) {
	transport := &kr.ResponseTransport{T: t}
	ec = NewTestEnclaveClient(transport)
	cs = &ControlServer{ec, kr.SetupLogging("test", logging.INFO, false)}

	randFile, err := kr.Rand128Base62()
	if err != nil {
		t.Fatal(err)
	}
	unixFile = filepath.Join(os.TempDir(), randFile)
	l, err := net.Listen("unix", unixFile)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err := cs.HandleControlHTTP(l)
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
