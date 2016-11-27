package kr

import (
	"os"
	"os/exec"
	"strings"
)

func MachineName() (name string) {
	nameBytes, err := exec.Command("scutil", "--get", "ComputerName").Output()
	if err == nil {
		name = strings.TrimSpace(string(nameBytes))
	} else {
		name, _ = os.Hostname()
	}
	return
}
