package kr

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func GlobalGitUserId() (id string, err error) {
	oldPath := os.Getenv("PATH")
	if !strings.Contains(oldPath, "/usr/local/bin") || !strings.Contains(oldPath, "/usr/bin") {
		os.Setenv("PATH", "/usr/bin:/usr/local/bin:"+oldPath)
	}
	name, err := exec.Command("git", "config", "--global", "user.name").Output()
	if err != nil {
		return
	}
	email, err := exec.Command("git", "config", "--global", "user.email").Output()
	if err != nil {
		return
	}
	id = fmt.Sprintf("%s <%s>", strings.TrimSpace(string(name)), strings.TrimSpace(string(email)))
	return
}

func HasGPG() bool {
	err := exec.Command("gpg", "--help").Run()
	return err == nil
}
