package main

import (
	"fmt"
	"os"
	"testing"

	krd "github.com/agrinman/kr/krd"
)

func TestPair(t *testing.T) {
	ec, _, unixFile := krd.NewLocalUnixServer(t)
	fmt.Println(unixFile)
	defer os.Remove(unixFile)
	ec.Start()
	defer ec.Stop()

	err := pairOver(unixFile, true)
	if err != nil {
		t.Fatal(err)
	}
}
