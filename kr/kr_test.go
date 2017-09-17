package main

import (
	"bytes"
	"testing"

	"github.com/kryptco/kr/krd"
)

func TestPair(t *testing.T) {
	ec, _ := krd.NewLocalUnixServer(t)
	ec.Start()
	defer ec.Stop()

	testPairSuccess(t, ec)
}

func testPairSuccess(t *testing.T, ec krd.EnclaveClientI) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := pairOver(true, nil, stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !ec.IsPaired() {
		t.Fatal("not paired")
	}
}

func TestUnpair(t *testing.T) {
	ec, _ := krd.NewLocalUnixServer(t)
	ec.Start()
	defer ec.Stop()

	testPairSuccess(t, ec)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := unpairOver(stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	if ec.IsPaired() {
		t.Fatal("paired")
	}
}
