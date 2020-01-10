package main

import (
	"bytes"
	"io/ioutil"
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

func isSSHConfigEdited(t *testing.T) bool {
	sshConfigPath, _ := getSSHConfigAndBakPaths()
	currentConfigContents, err := ioutil.ReadFile(sshConfigPath)
	if err != nil {
		t.Fatal("failed to open SSH config", err)
	}
	return bytes.Contains(currentConfigContents, []byte(getKrSSHConfigBlockOrFatal()))
}

func TestUnpair(t *testing.T) {
	ec, _, unixFile := NewLocalUnixServer(t)
	defer os.Remove(unixFile)
	ec.Start()
	defer ec.Stop()

	testPairSuccess(t, unixFile, ec)

	// manually make call to edit SSH config since test util doesn't call full
	// pair command which is normally responsible for modifying SSH config
	err := os.Setenv("HOME", os.TempDir())
	if err != nil {
		t.Fatal("failed to override HOME env var to create test SSH config")
	}
	err = autoEditSSHConfig()
	if err != nil {
		t.Fatal("failed to create SSH config for test")
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = unpairOver(unixFile, stdout, stderr)

	if err != nil {
		t.Fatal(err)
	}
	if ec.IsPaired() {
		t.Fatal("paired")
	}
	if isSSHConfigEdited(t) {
		t.Fatal("failed to reset SSH config after unpairing")
	}
}
