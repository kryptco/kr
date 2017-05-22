package kr

import (
	"fmt"
	"os/exec"
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
	id = fmt.Sprintf("%s <%s>", name, email)
	return
}
