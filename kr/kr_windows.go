package main

import (
	"os"
	"os/exec"

	"github.com/urfave/cli"
)

func restartCommand(c *cli.Context) (err error) {
	PrintErr(os.Stderr, "Kryptonite daemon restart from cmdline not supported on Windows.")
	return
}

func openBrowser(url string) {
	exec.Command("cmd", "start", url).Run()
}

func hasAptGet() bool {
	return false
}

func hasYum() bool {
	return false
}

func uninstallCommand(c *cli.Context) (err error) {
	PrintErr(os.Stderr, "Kryptonite uninstall from cmdline not supported on Windows.")
	return
}

func upgradeCommand(c *cli.Context) (err error) {
	PrintErr(os.Stderr, "Kryptonite upgrade from cmdline not supported on Windows.")
	return
}
