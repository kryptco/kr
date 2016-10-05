package main

import (
	"bytes"
	"os/exec"
	"strings"
)

func getLastCommand() (lastCommand *string) {
	psWithHeader, err := exec.Command("ps", "-o", "lstart", "-f").Output()
	if err != nil {
		return
	}
	skipHeaderCmd := exec.Command("tail", "-n", "+2")
	skipHeaderCmd.Stdin = bytes.NewReader(psWithHeader)
	ps, err := skipHeaderCmd.Output()
	if err != nil {
		log.Error("tailCmd error", err)
		return
	}
	trimDayCmd := exec.Command("awk", "{$1=\"\";print}")
	trimDayCmd.Stdin = bytes.NewReader(ps)
	unsortedPs, err := trimDayCmd.Output()
	if err != nil {
		log.Error("awkCmd error", err)
		return
	}
	sortCmd := exec.Command("sort")
	sortCmd.Stdin = bytes.NewReader(unsortedPs)
	sortedPs, err := sortCmd.Output()
	if err != nil {
		log.Error("sortCmd error", err)
		return
	}
	tailCmd := exec.Command("tail", "-1")
	tailCmd.Stdin = bytes.NewReader(sortedPs)
	psLine, err := tailCmd.Output()
	if err != nil {
		log.Error("tailcmd error", err)
		return
	}

	psTokens := strings.Fields(string(psLine))
	if len(psTokens) <= 11 {
		return
	}
	command := strings.Join(psTokens[11:], " ")
	command = strings.Replace(command, "git-receive-pack", "push", 1)
	command = strings.Replace(command, "git-upload-pack", "pull", 1)
	return &command
}
