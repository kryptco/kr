package krd

import (
	"flag"
	"os"
	"runtime"
	"testing"
)

func TestMain(m *testing.M) {
	if runtime.GOMAXPROCS(0) == 1 {
		runtime.GOMAXPROCS(4)
	}
	flag.Parse()
	os.Exit(m.Run())
}
