package client

import (
	"crypto/sha256"
	version2 "krypt.co/kr/common/version"
	"net"
	"os"
	"testing"

	. "krypt.co/kr/common/util"
	. "krypt.co/kr/daemon/control"
)

func TestVersion(t *testing.T) {
	ec, _, unixFile := NewLocalUnixServer(t)
	PairClient(t, ec)
	defer ec.Stop()

	conn, err := net.Dial("unix", unixFile)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(unixFile)

	version, err := RequestKrdVersionOver(conn)
	if err != nil {
		t.Fatal(err)
	}
	if version.Compare(version2.CURRENT_VERSION) != 0 {
		t.Fatal("wrong version")
	}
}

func TestMeRequest(t *testing.T) {
	ec, _, unixFile := NewLocalUnixServer(t)
	PairClient(t, ec)
	defer ec.Stop()

	conn, err := net.Dial("unix", unixFile)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(unixFile)

	me, err := RequestMeOver(conn)
	if err != nil {
		t.Fatal(err)
	}
	testMe, _, _ := TestMe(t)
	if !me.Equal(testMe) {
		t.Fatal("wrong profile")
	}
}

func TestSign(t *testing.T) {
	ec, _, unixFile := NewLocalUnixServer(t)
	PairClient(t, ec)
	defer ec.Stop()

	conn, err := net.Dial("unix", unixFile)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(unixFile)

	testMe, _, _ := TestMe(t)

	digest := sha256.Sum256([]byte{0})
	_, err = signOver(conn, testMe.PublicKeyFingerprint(), digest[:])
	if err != nil {
		t.Fatal(err)
	}
}

func TestNoOp(t *testing.T) {
	ec, _, unixFile := NewLocalUnixServer(t)
	PairClient(t, ec)
	defer ec.Stop()

	conn, err := net.Dial("unix", unixFile)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(unixFile)

	err = requestNoOpOver(conn)
	if err != nil {
		t.Fatal(err)
	}
}
