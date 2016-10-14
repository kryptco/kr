package main

import (
	"os/exec"
	"time"

	"github.com/urfave/cli"
)

func restartCommand(c *cli.Context) (err error) {
	PrintFatal("Not yet implemented on Linux.")
	return
}

func githubCommand(c *cli.Context) (err error) {
	copyKey()
	PrintErr("Public key copied to clipboard.")
	<-time.After(500 * time.Millisecond)
	PrintErr("Opening GitHub...")
	<-time.After(500 * time.Millisecond)
	exec.Command("sensible-browser", "https://github.com/settings/keys").Run()
	return
}
