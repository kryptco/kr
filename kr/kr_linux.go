package main

import (
	"os/exec"

	"github.com/urfave/cli"
)

func restartCommand(c *cli.Context) (err error) {
	PrintFatal("Not yet implemented on Linux.")
	return
}
func openBrowser(url string) {
	exec.Command("sensible-browser", url).Run()
}
