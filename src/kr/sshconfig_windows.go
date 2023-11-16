package main

import (
	"os"
	"path/filepath"

	. "krypt.co/kr/common/util"
)

func getPrefix() (string, error) {
	if ex, err := os.Executable(); err == nil {
		return filepath.Dir(ex), nil
	} else {
		PrintErr(os.Stderr, Red("Krypton ▶ Problem getting path of kr.exe"))
		return "", err
	}
}
