// +build !darwin

package main

import (
	"os"
	"os/exec"

	"github.com/kryptco/kr"
	"github.com/urfave/cli"
)

const DEFAULT_PREFIX = "/usr"

func getPrefix() string {
	prefix := DEFAULT_PREFIX
	if os.Getenv("PREFIX") != "" {
		prefix = os.Getenv("PREFIX")
	}
	return prefix
}

func restartCommandOptions(c *cli.Context, isUserInitiated bool) (err error) {
	if isUserInitiated {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "restart", nil, nil)
	}

	exec.Command("killall", "krd").Run()
	startKrd()

	if isUserInitiated {
		PrintErr(os.Stderr, "Restarted Kryptonite daemon.")
	}
	return
}

func startKrd() (err error) {
	exec.Command("nohup", "krd").Start()
	return
}

func isKrdRunning() bool {
	return nil == exec.Command("pgrep", "krd").Run()
}

func openBrowser(url string) {
	err := exec.Command("sensible-browser", url).Run()
	if err != nil {
		os.Stderr.WriteString("Unable to open browser, please visit " + url + "\r\n")
	}
}

func hasAptGet() bool {
	return exec.Command("which", "apt-get").Run() == nil
}

func hasYum() bool {
	return exec.Command("which", "yum").Run() == nil
}

func hasYaourt() bool {
	return exec.Command("which", "yaourt").Run() == nil
}

func uninstallCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "uninstall", nil, nil)
	}()
	confirmOrFatal(os.Stderr, "Uninstall Kryptonite from this workstation? (same as sudo apt-get/yum remove kr)")

	exec.Command("killall", "krd").Run()

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

	if hasYaourt() {
		runCommandWithUserInteraction("sudo", "yaourt", "-R", "kr")
	}

	uninstallCodesigning()
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
	if hasYaourt() {
		runCommandWithUserInteraction("sudo", "yaourt", "-Sy", "kr")
	}

	return
}
