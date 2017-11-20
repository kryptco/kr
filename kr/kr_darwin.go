package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/kryptco/kr"
	"github.com/urfave/cli"
)

const DEFAULT_PREFIX = "/usr/local"

func getPrefix() string {
	prefix := DEFAULT_PREFIX
	if os.Getenv("PREFIX") != "" {
		prefix = os.Getenv("PREFIX")
	} else if os.Getenv("HOMEBREW_PREFIX") != "" {
		prefix = os.Getenv("HOMEBREW_PREFIX")
	}
	return prefix
}

const PLIST_TEMPLATE = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>EnvironmentVariables</key>
	<dict>
		<key>GOTRACEBACK</key>
		<string>crash</string>
	</dict>
	<key>Label</key>
	<string>co.krypt.krd</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s/krd_stdout.log</string>
	<key>StandardErrorPath</key>
	<string>%s/krd_stderr.log</string>
</dict>
</plist>`

func copyPlist() (err error) {
	output, err := exec.Command("which", "krd").Output()
	if err != nil {
		PrintErr(os.Stderr, kr.Red("Kryptonite ▶ Could not find krd on PATH, make sure krd is installed"))
		return
	}
	krdir, err := kr.KrDir()
	if err != nil {
		PrintErr(os.Stderr, kr.Red("Kryptonite ▶ Error finding ~/.kr folder: "+err.Error()))
		return
	}
	plistContents := fmt.Sprintf(PLIST_TEMPLATE, strings.TrimSpace(string(output)), krdir, krdir)
	_ = os.MkdirAll(homePlistDir, 0700)
	err = ioutil.WriteFile(homePlist, []byte(plistContents), 0700)
	if err != nil {
		PrintErr(os.Stderr, kr.Red("Kryptonite ▶ Error writing krd plist: "+err.Error()))
		return
	}
	return
}

func runCommandTmuxFriendly(cmd string, args ...string) (output string, err error) {
	//	fixes tmux launchctl permissions
	var outputBytes []byte
	if os.Getenv("TMUX") != "" {
		subcommandArgs := strings.Join(append([]string{cmd}, args...), " ")
		outputBytes, err = exec.Command("reattach-to-user-namespace", "-l", "bash", "-c", subcommandArgs).CombinedOutput()
		if err != nil {
			if execErr, ok := err.(*exec.Error); ok && execErr.Err == exec.ErrNotFound {
				PrintFatal(os.Stderr, kr.Red("Kryptonite ▶ Running tmux-friendly command failed. Make sure \"reattach-to-user-namespace\" is installed with \"brew install reattach-to-user-namespace\"\r\n"))
			}
		}
	} else {
		outputBytes, err = exec.Command(cmd, args...).CombinedOutput()
	}
	output = string(outputBytes)
	return
}

func startKrd() (err error) {
	err = copyPlist()
	if err != nil {
		return
	}
	for _, proxyVar := range []string{"http_proxy", "HTTP_PROXY", "https_proxy", "HTTPS_PROXY"} {
		copyEnvToLaunchctl(proxyVar)
	}
	_, _ = runCommandTmuxFriendly("launchctl", "unload", homePlist)
	output, err := runCommandTmuxFriendly("launchctl", "load", homePlist)
	if len(output) > 0 || err != nil {
		err = fmt.Errorf(kr.Red("Kryptonite ▶ Error starting krd with launchctl: " + string(output)))
		PrintErr(os.Stderr, err.Error())
		return
	}
	return
}

func isKrdRunning() bool {
	return nil == exec.Command("pgrep", "krd").Run()
}

func killKrd() (err error) {
	_, _ = runCommandTmuxFriendly("launchctl", "unload", homePlist)
	_, _ = runCommandTmuxFriendly("killall", "krd")
	return
}

const PLIST = "co.krypt.krd.plist"

var homePlistDir = os.Getenv("HOME") + "/Library/LaunchAgents"
var homePlist = homePlistDir + "/" + PLIST

func copyEnvToLaunchctl(varName string) {
	_, _ = runCommandTmuxFriendly("launchctl", "setenv", varName, os.Getenv(varName))
}

func restartCommandOptions(c *cli.Context, isUserInitiated bool) (err error) {
	if isUserInitiated {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "restart", nil, nil)
	}

	err = copyPlist()
	if err != nil {
		return
	}
	err = killKrd()
	if err != nil {
		return
	}
	err = startKrd()
	if err != nil {
		return
	}

	if isUserInitiated {
		fmt.Println("Restarted Kryptonite daemon.")
	}
	return
}

func openBrowser(url string) {
	exec.Command("open", url).Run()
}

func cleanSSHConfig() (err error) {
	configBlock := []byte(getKryptoniteSSHConfigBlock())
	sshDirPath := os.Getenv("HOME") + "/.ssh"
	sshConfigPath := sshDirPath + "/config"
	sshConfigBackupPath := sshConfigPath + ".bak.kr.uninstall"

	sshConfigFile, err := os.Open(sshConfigPath)
	if err != nil {
		return
	}
	currentConfigContents, err := ioutil.ReadAll(sshConfigFile)
	if err != nil {
		return
	}
	if len(currentConfigContents) > 0 {
		err = ioutil.WriteFile(sshConfigBackupPath, currentConfigContents, 0700)
		if err != nil {
			return
		}
	}

	newConfigContents := bytes.Replace(currentConfigContents, configBlock, []byte{}, -1)
	err = ioutil.WriteFile(sshConfigPath, newConfigContents, 0700)
	if err != nil {
		return
	}
	return
}

func uninstallCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "uninstall", nil, nil)
	}()
	confirmOrFatal(os.Stderr, "Uninstall Kryptonite from this workstation?")
	_, _ = runCommandTmuxFriendly("brew", "uninstall", "kr")
	_, _ = runCommandTmuxFriendly("npm", "uninstall", "-g", "krd")
	prefix := getPrefix()
	for _, file := range []string{"/bin/kr", "/bin/krssh", "/bin/krd", "/bin/krgpg", "/lib/kr-pkcs11.so", "/share/kr", "/Frameworks/krbtle.framework"} {
		rmErr := exec.Command("rm", "-rf", prefix+file).Run()
		if rmErr != nil {
			if os.IsPermission(rmErr) {
				PrintErr(os.Stderr, "sudo rm -rf "+prefix+file)
				runCommandWithUserInteraction("sudo", "rm", "-rf", prefix+file)
			}
		}
	}
	runCommandTmuxFriendly("launchctl", "unload", homePlist)
	os.Remove(homePlist)
	cleanSSHConfig()
	uninstallCodesigning()
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
