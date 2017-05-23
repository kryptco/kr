package kr

import (
	"fmt"
	"os/exec"
	"strings"
)

func GlobalGitUserId() (id string, err error) {
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
