package main

import (
	"os"
	"os/exec"

	"github.com/kryptco/kr"
	"github.com/urfave/cli"
)

func runCommandWithUserInteraction(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func restartCommand(c *cli.Context) (err error) {
	kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "restart", nil, nil)
	exec.Command("pkill", "krd").Run()
	exec.Command("nohup", "krd").Start()
	PrintErr(os.Stderr, "Restarted Kryptonite daemon.")
	return
}

func openBrowser(url string) {
	exec.Command("sensible-browser", url).Run()
}

func hasAptGet() bool {
	return exec.Command("which", "apt-get").Run() == nil
}

func hasYum() bool {
	return exec.Command("which", "yum").Run() == nil
}

func uninstallCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "uninstall", nil, nil)
	}()
	confirmOrFatal(os.Stderr, "Uninstall Kryptonite from this workstation? (same as sudo apt-get/yum remove kr)")

	exec.Command("pkill", "krd").Run()

	if hasAptGet() {
		uninstallCmd := exec.Command("sudo", "apt-get", "remove", "kr", "-y")
		uninstallCmd.Stdout = os.Stdout
		uninstallCmd.Stderr = os.Stderr
		uninstallCmd.Run()
	}

	if hasYum() {
		uninstallCmd := exec.Command("sudo", "yum", "remove", "kr", "-y")
		uninstallCmd.Stdout = os.Stdout
		uninstallCmd.Stderr = os.Stderr
		uninstallCmd.Run()
	}

	PrintErr(os.Stderr, "Kryptonite uninstalled.")
	return
}

func upgradeCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "upgrade", nil, nil)
	}()
	confirmOrFatal(os.Stderr, "Upgrade Kryptonite on this workstation?")
	if hasAptGet() {
		update := exec.Command("sudo", "apt-get", "update")
		update.Stdout = os.Stdout
		update.Stderr = os.Stderr
		update.Stdin = os.Stdin
		update.Run()
		cmd := exec.Command("sudo", "apt-get", "install", "kr")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Run()
	}
	if hasYum() {
		runCommandWithUserInteraction("sudo", "yum", "clean", "expire-cache")
		runCommandWithUserInteraction("sudo", "yum", "upgrade", "kr", "-y")
	}
	return
}
