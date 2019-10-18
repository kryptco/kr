package main

import (
	"bytes"
	"os"
	"testing"

	. "krypt.co/kr/daemon/enclave"
	. "krypt.co/kr/daemon/control"

)

func TestPair(t *testing.T) {
	ec, _, unixFile := NewLocalUnixServer(t)
	defer os.Remove(unixFile)
	ec.Start()
	defer ec.Stop()

	testPairSuccess(t, unixFile, ec)
}

func testPairSuccess(t *testing.T, unixFile string, ec EnclaveClientI) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := pairOver(unixFile, true, nil, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !ec.IsPaired() {
		t.Fatal("not paired")
	}
}

func TestUnpair(t *testing.T) {
	ec, _, unixFile := NewLocalUnixServer(t)
	defer os.Remove(unixFile)
	ec.Start()
	defer ec.Stop()

	testPairSuccess(t, unixFile, ec)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := unpairOver(unixFile, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	if ec.IsPaired() {
		t.Fatal("paired")
	}
}
