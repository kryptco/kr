// +build !windows

package main

import (
	"os"
	"os/exec"
	"strings"

	. "krypt.co/kr/common/util"
)

func getPrefix() (string, error) {
	krAbsPath, err := exec.Command("which", "kr").Output()
	if err != nil {
		PrintErr(os.Stderr, Red("Krypton ▶ Could not find kr on PATH"))
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSpace(string(krAbsPath)), "/bin/kr"), nil
}
