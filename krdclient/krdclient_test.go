package krdclient

import (
	"crypto/sha256"
	"testing"

	"github.com/kryptco/kr"
	"github.com/kryptco/kr/krd"
)

func TestVersion(t *testing.T) {
	ec, _ := krd.NewLocalUnixServer(t)
	krd.PairClient(t, ec)
	defer ec.Stop()

	conn, err := kr.DaemonDial()
	if err != nil {
		t.Fatal(err)
	}

	version, err := RequestKrdVersionOver(conn)
	if err != nil {
		t.Fatal(err)
	}
	if version.Compare(kr.CURRENT_VERSION) != 0 {
		t.Fatal("wrong version")
	}
}

func TestMe(t *testing.T) {
	ec, _ := krd.NewLocalUnixServer(t)
	krd.PairClient(t, ec)
	defer ec.Stop()

	conn, err := kr.DaemonDial()
	if err != nil {
		t.Fatal(err)
	}

	me, err := RequestMeOver(conn)
	if err != nil {
		t.Fatal(err)
	}
	testMe, _, _ := kr.TestMe(t)
	if !me.Equal(testMe) {
		t.Fatal("wrong profile")
	}
}

func TestSign(t *testing.T) {
	ec, _ := krd.NewLocalUnixServer(t)
	krd.PairClient(t, ec)
	defer ec.Stop()

	conn, err := kr.DaemonDial()
	if err != nil {
		t.Fatal(err)
	}

	testMe, _, _ := kr.TestMe(t)

	digest := sha256.Sum256([]byte{0})
	_, err = signOver(conn, testMe.PublicKeyFingerprint(), digest[:])
	if err != nil {
		t.Fatal(err)
	}
}

func TestNoOp(t *testing.T) {
	ec, _ := krd.NewLocalUnixServer(t)
	krd.PairClient(t, ec)
	defer ec.Stop()

	conn, err := kr.DaemonDial()
	if err != nil {
		t.Fatal(err)
	}

	err = requestNoOpOver(conn)
	if err != nil {
		t.Fatal(err)
	}
}
