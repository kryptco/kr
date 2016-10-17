package krdclient

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

//	parse current command if dynamically loaded with ssh
func currentCommand() *string {
	pid := fmt.Sprintf("%d", os.Getpid())
	cmdBytes, _ := exec.Command("ps", "h", "--format", "command", pid).Output()
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
