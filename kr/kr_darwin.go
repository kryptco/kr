package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/urfave/cli"
)

func restartCommand(c *cli.Context) (err error) {
	plist := os.Getenv("HOME") + "/Library/LaunchAgents/co.krypt.krd.plist"
	exec.Command("launchctl", "unload", plist).Run()
	err = exec.Command("launchctl", "load", plist).Run()
	if err != nil {
		PrintFatal("Failed to restart Kryptonite daemon.")
	}
	fmt.Println("Restarted Kryptonite daemon.")
	return
}

func githubCommand(c *cli.Context) (err error) {
	copyKey()
	PrintErr("Public key copied to clipboard.")
	exec.Command("open", "https://github.com/settings/keys").Run()
	return
}
