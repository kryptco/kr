package main

import (
	"os/exec"

	"github.com/urfave/cli"
)

func restartCommand(c *cli.Context) (err error) {
	PrintFatal("Not yet implemented on Linux.")
	return
}

func githubCommand(c *cli.Context) (err error) {
	copyKey()
	PrintErr("Public key copied to clipboard.")
	exec.Command("sensible-browser", "https://github.com/settings/keys").Run()
	return
}
