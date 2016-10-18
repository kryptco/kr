package main

import (
	"os/exec"

	"github.com/urfave/cli"
)

func restartCommand(c *cli.Context) (err error) {
	exec.Command("systemctl", "--user", "disable", "kr").Run()
	exec.Command("systemctl", "--user", "stop", "kr").Run()
	exec.Command("systemctl", "--user", "enable", "kr").Run()
	exec.Command("systemctl", "--user", "start", "kr").Run()
	PrintErr("Restarted Kryptonite daemon.")
	return
}

func openBrowser(url string) {
	exec.Command("sensible-browser", url).Run()
}
