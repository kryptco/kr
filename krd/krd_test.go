package krd

import(
	"testing"
	"runtime"
	"flag"
	"os"
)

func TestMain(m *testing.M) {
	if runtime.GOMAXPROCS(0) == 1 {
		runtime.GOMAXPROCS(4)
	}
	flag.Parse()
	os.Exit(m.Run())
}
