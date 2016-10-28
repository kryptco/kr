package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli"
)

var plist = os.Getenv("HOME") + "/Library/LaunchAgents/co.krypt.krd.plist"

func restartCommand(c *cli.Context) (err error) {
	exec.Command("launchctl", "unload", plist).Run()
	err = exec.Command("launchctl", "load", plist).Run()
	if err != nil {
		PrintFatal("Failed to restart Kryptonite daemon.")
	}
	fmt.Println("Restarted Kryptonite daemon.")
	return
}

func openBrowser(url string) {
	exec.Command("open", url).Run()
}

func uninstallCommand(c *cli.Context) (err error) {
	confirmOrFatal("Uninstall Kryptonite from this workstation?")
	exec.Command("brew", "uninstall", "kr").Run()
	exec.Command("npm", "uninstall", "-g", "krd").Run()
	os.Remove("/usr/local/bin/kr")
	os.Remove("/usr/local/bin/krd")
	os.Remove("/usr/local/lib/kr-pkcs11.so")
	os.Remove("/usr/local/share/kr")
	exec.Command("launchctl", "unload", plist).Run()
	os.Remove(plist)
	exec.Command(cleanSSHConfigCommand[0], cleanSSHConfigCommand[1:]...).Run()
	PrintErr("Kryptonite uninstalled.")
	return
}

func installedWithBrew() bool {
	krLinkBytes, _ := exec.Command("sh", "-c", "ls -l `command -v kr`").CombinedOutput()
	krLink := string(krLinkBytes)
	return strings.Contains(krLink, "Cellar")
}

func installedWithNPM() bool {
	return exec.Command("npm", "list", "-g", "krd").Run() == nil
}

func upgradeCommand(c *cli.Context) (err error) {
	confirmOrFatal("Upgrade Kryptonite on this workstation?")
	var cmd *exec.Cmd
	if installedWithBrew() {
		cmd = exec.Command("brew", "upgrade", "kr")
	} else if installedWithNPM() {
		cmd = exec.Command("npm", "upgrade", "-g", "krd")
	} else {
		cmd = exec.Command("sh", "-c", "curl https://krypt.co/kr | sh")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Run()
	return
}
