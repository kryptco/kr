package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kryptco/kr"
	"github.com/urfave/cli"
)

var plist = os.Getenv("HOME") + "/Library/LaunchAgents/co.krypt.krd.plist"

func copyEnvToLaunchctl(varName string) {
	exec.Command("launchctl", "setenv", varName, os.Getenv(varName)).Run()
}

func restartCommand(c *cli.Context) (err error) {
	kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "restart", nil, nil)
	for _, proxyVar := range []string{"http_proxy", "HTTP_PROXY", "https_proxy", "HTTPS_PROXY"} {
		copyEnvToLaunchctl(proxyVar)
	}
	exec.Command("launchctl", "unload", plist).Run()
	err = exec.Command("launchctl", "load", plist).Run()
	if err != nil {
		PrintFatal(os.Stderr, "Failed to restart Kryptonite daemon.")
	}
	fmt.Println("Restarted Kryptonite daemon.")
	return
}

func openBrowser(url string) {
	exec.Command("open", url).Run()
}

var oldSSHConfigString = "# Added by Kryptonite\\nHost \\*\\n\\tPKCS11Provider \\/usr\\/local\\/lib\\/kr-pkcs11.so\\n\\tProxyCommand \\`find \\/usr\\/local\\/bin\\/krssh 2\\>\\/dev\\/null \\|\\| which nc\\` \\%h \\%p\\n\\tIdentityFile ~\\/.ssh\\/id_kryptonite\\n\\tIdentityFile ~\\/.ssh\\/id_ed25519\\n\\tIdentityFile ~\\/.ssh\\/id_rsa\\n\\tIdentityFile ~\\/.ssh\\/id_ecdsa\\n\\tIdentityFile ~\\/.ssh\\/id_dsa"
var sshConfigString = "# Added by Kryptonite\\nHost \\*\\n\\tPKCS11Provider \\/usr\\/local\\/lib\\/kr-pkcs11.so\\n\\tProxyCommand \\/usr\\/local\\/bin\\/krssh \\%h \\%p\\n\\tIdentityFile ~\\/.ssh\\/id_kryptonite\\n\\tIdentityFile ~\\/.ssh\\/id_ed25519\\n\\tIdentityFile ~\\/.ssh\\/id_rsa\\n\\tIdentityFile ~\\/.ssh\\/id_ecdsa\\n\\tIdentityFile ~\\/.ssh\\/id_dsa"

func cleanSSHConfigString(sshConfig string) string {
	return "s/\\s*" + sshConfig + "//g"
}

func cleanSSHConfigCommand(sshConfig string, backupExtension string) []string {
	return []string{"perl", "-0777", "-p", "-i" + backupExtension, "-e", cleanSSHConfigString(sshConfig), os.Getenv("HOME") + "/.ssh/config"}
}

func cleanSSHConfig(sshConfig string, backupExtension string) {
	command := cleanSSHConfigCommand(sshConfig, backupExtension)
	exec.Command(command[0], command[1:]...).Run()
}

func uninstallCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "uninstall", nil, nil)
	}()
	confirmOrFatal(os.Stderr, "Uninstall Kryptonite from this workstation?")
	exec.Command("brew", "uninstall", "kr").Run()
	exec.Command("npm", "uninstall", "-g", "krd").Run()
	os.Remove("/usr/local/bin/kr")
	os.Remove("/usr/local/bin/krssh")
	os.Remove("/usr/local/bin/krd")
	os.Remove("/usr/local/lib/kr-pkcs11.so")
	os.Remove("/usr/local/share/kr")
	exec.Command("launchctl", "unload", plist).Run()
	os.Remove(plist)
	cleanSSHConfig(sshConfigString, ".bak3")
	cleanSSHConfig(oldSSHConfigString, ".bak4")
	PrintErr(os.Stderr, "Kryptonite uninstalled.")
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
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "upgrade", nil, nil)
	}()
	confirmOrFatal(os.Stderr, "Upgrade Kryptonite on this workstation?")
	var cmd *exec.Cmd
	if installedWithBrew() {
		cmd = exec.Command("brew", "upgrade", "kryptco/tap/kr")
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
