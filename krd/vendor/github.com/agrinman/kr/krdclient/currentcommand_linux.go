package krdclient

import (
	"os"
	"strings"
)

func currentCommand() *string {
	command := strings.TrimSpace(strings.Join(os.Args, " "))
	command = strings.Replace(command, "git-receive-pack", "push", 1)
	command = strings.Replace(command, "git-upload-pack", "pull", 1)
	if command == "ssh" {
		return nil
	}
	return &command
}
