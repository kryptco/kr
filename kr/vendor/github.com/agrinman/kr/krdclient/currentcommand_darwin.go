package krdclient

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

//	parse current command if dynamically loaded with ssh
func currentCommand() *string {
	pid := fmt.Sprintf("%d", os.Getpid())
	psBytes, _ := exec.Command("ps", "-o", "command", "-p", pid).Output()
	tailCmd := exec.Command("tail", "-1")
	tailCmd.Stdin = bytes.NewReader(psBytes)
	cmdBytes, _ := tailCmd.Output()
	cmd := string(cmdBytes)
	toks := strings.Fields(cmd)
	if cmd == "" || len(toks) == 0 {
		return nil
	}
	if !strings.HasSuffix(toks[0], "ssh") {
		return nil
	}
	cmd = strings.Join(toks, " ")
	cmd = strings.Replace(cmd, "git-receive-pack", "push", 1)
	cmd = strings.Replace(cmd, "git-upload-pack", "pull", 1)
	return &cmd
}
