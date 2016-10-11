package main

import (
	"bytes"
	"os/exec"
	"strings"
	"time"
)

//	fallback using ps
func getLastCommand() (lastCommand *string) {
	psWithHeader, err := exec.Command("ps", "-o", "lstart", "-f").Output()
	if err != nil {
		return
	}
	skipHeaderCmd := exec.Command("tail", "-n", "+2")
	skipHeaderCmd.Stdin = bytes.NewReader(psWithHeader)
	unsortedPs, err := skipHeaderCmd.Output()
	if err != nil {
		log.Error("tailCmd error", err)
		return
	}
	var latestTime *time.Time
	var latestCommand *string
	for _, line := range strings.Split(string(unsortedPs), "\n") {
		toks := strings.Fields(line)
		if len(toks) < 13 {
			continue
		}
		t, err := time.Parse("Mon Jan 2 15:04:05 2006", strings.Join(toks[:5], " "))
		if err != nil {
			log.Warning("error parsing time:", err)
			continue
		}
		if latestTime == nil || t.After(*latestTime) || t.Equal(*latestTime) {
			latestTime = &t
			command := strings.Join(toks[12:], " ")
			latestCommand = &command
		}
	}

	if latestCommand != nil {
		command := strings.Replace(*latestCommand, "git-receive-pack", "push", 1)
		command = strings.Replace(command, "git-upload-pack", "pull", 1)
		latestCommand = &command
	}
	return latestCommand
}
