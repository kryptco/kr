package main

import (
	"os"
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

func uninstallCommand(c *cli.Context) (err error) {
	confirmOrFatal("Uninstall Kryptonite from this workstation? (same as sudo apt-get remove kr)")

	exec.Command("systemctl", "--user", "disable", "kr").Run()
	exec.Command("systemctl", "--user", "stop", "kr").Run()

	uninstallCmd := exec.Command("sudo", "apt-get", "remove", "kr", "-y")
	uninstallCmd.Stdout = os.Stdout
	uninstallCmd.Stderr = os.Stderr
	uninstallCmd.Run()

	PrintErr("Kryptonite uninstalled.")
	return
}
