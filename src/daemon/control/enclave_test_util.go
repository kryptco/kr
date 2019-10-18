package control

import (
	"krypt.co/kr/common/log"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/op/go-logging"
	. "krypt.co/kr/common/persistance"
	. "krypt.co/kr/common/protocol"
	. "krypt.co/kr/common/transport"
	. "krypt.co/kr/common/util"
	. "krypt.co/kr/daemon/enclave"
)

func NewTestEnclaveClient(transport Transport) EnclaveClientI {
	return UnpairedEnclaveClient(
		transport,
		&MemoryPersister{},
		nil,
		log.SetupLogging("test", logging.INFO, false),
		nil,
	)
}

func NewTestEnclaveClientShortTimeouts(transport Transport) EnclaveClientI {
	shortTimeouts := Timeouts{
		Me: TimeoutPhases{
			Alert: 800 * time.Millisecond,
			Fail:  1600 * time.Millisecond,
		},
		Pair: TimeoutPhases{
			Alert: 800 * time.Millisecond,
			Fail:  1600 * time.Millisecond,
		},
		Sign: TimeoutPhases{
			Alert: 800 * time.Millisecond,
			Fail:  1600 * time.Millisecond,
		},
		ACKDelay: SHORT_ACK_DELAY,
	}

	ec := UnpairedEnclaveClient(
		transport,
		&MemoryPersister{},
		&shortTimeouts,
		log.SetupLogging("test", logging.INFO, false),
		nil,
	)
	return ec
}

func NewLocalUnixServer(t *testing.T) (ec EnclaveClientI, cs *ControlServer, unixFile string) {
	transport := &ResponseTransport{T: t}
	ec = NewTestEnclaveClient(transport)
	cs = &ControlServer{ec, log.SetupLogging("test", logging.INFO, false)}

	randFile, err := Rand128Base62()
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

func PairClient(t *testing.T, client EnclaveClientI) (ps *PairingSecret) {
	err := client.Start()
	if err != nil {
		t.Fatal(err)
	}
	var pairingOptions PairingOptions
	ps, err = client.Pair(pairingOptions)
	if err != nil {
		t.Fatal(err)
	}
	go client.RequestMe(MeRequest{}, true)
	TrueBefore(t, client.IsPaired, time.Now().Add(time.Second))
	return
}
