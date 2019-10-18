package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli"
	. "krypt.co/kr/common/analytics"
	. "krypt.co/kr/common/socket"
	. "krypt.co/kr/common/util"
)

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
		PrintErr(os.Stderr, Red("Krypton ▶ Could not find krd on PATH, make sure krd is installed"))
		return
	}
	krdir, err := KrDir()
	if err != nil {
		PrintErr(os.Stderr, Red("Krypton ▶ Error finding ~/.kr folder: "+err.Error()))
		return
	}
	plistContents := fmt.Sprintf(PLIST_TEMPLATE, strings.TrimSpace(string(output)), krdir, krdir)
	_ = os.MkdirAll(homePlistDir, 0700)
	err = ioutil.WriteFile(homePlist, []byte(plistContents), 0700)
	if err != nil {
		PrintErr(os.Stderr, Red("Krypton ▶ Error writing krd plist: "+err.Error()))
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
				PrintFatal(os.Stderr, Red("Krypton ▶ Running tmux-friendly command failed. Make sure \"reattach-to-user-namespace\" is installed with \"brew install reattach-to-user-namespace\"\r\n"))
			}
		}
	} else {
		outputBytes, err = exec.Command(cmd, args...).CombinedOutput()
	}
	output = string(outputBytes)
	return
}

func filterReattachToUserNamespaceOutput(output string) string {
	//	reattach-to-user-namespace prints a warning on 10.14, filter this
	//	output so that a false positive is not created for launchctl load error
	//	detection
	lines := strings.Split(output, "\n")
	filteredLines := []string{}
	for _, line := range lines {
		if strings.Contains(line, "reattach-to-user-namespace") {
			continue
		}
		filteredLines = append(filteredLines, line)
	}
	return strings.Join(filteredLines, "\n")
}

func startKrd() (err error) {
	err = copyPlist()
	if err != nil {
		return
	}
	for _, envVar := range []string{"http_proxy", "HTTP_PROXY", "https_proxy", "HTTPS_PROXY", "SSH_AUTH_SOCK"} {
		copyEnvToLaunchctl(envVar)
	}
	_, _ = runCommandTmuxFriendly("launchctl", "unload", homePlist)
	output, err := runCommandTmuxFriendly("launchctl", "load", homePlist)
	output = filterReattachToUserNamespaceOutput(output)
	//	launchctl load exit status only reflects syntax, not execution failure (always returns 0)
	if len(output) > 0 || err != nil {
		err = fmt.Errorf(Red("Krypton ▶ Error starting krd with launchctl: " + string(output)))
		PrintErr(os.Stderr, err.Error())
		return
	}
	return
}

func killKrd() (err error) {
	_, _ = runCommandTmuxFriendly("launchctl", "unload", homePlist)
	_, _ = runCommandTmuxFriendly("pkill", "-U", os.Getenv("USER"), "-x", "krd")
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
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "restart", nil, nil)
	}

	_ = migrateSSHConfig()

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
		fmt.Println("Restarted Krypton daemon.")
	}
	return
}

func openBrowser(url string) {
	exec.Command("open", url).Run()
}

func uninstallCommand(c *cli.Context) (err error) {
	go func() {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "uninstall", nil, nil)
	}()
	confirmOrFatal(os.Stderr, "Uninstall Krypton from this workstation?")
	_, _ = runCommandTmuxFriendly("brew", "uninstall", "kr")
	_, _ = runCommandTmuxFriendly("npm", "uninstall", "-g", "krd")
	cleanSSHConfig()
	prefix, err := getPrefix()
	if err != nil {
		PrintErr(os.Stderr, "Could not determine PREFIX: "+err.Error())
	} else {
		for _, file := range []string{"/bin/kr", "/bin/krssh", "/bin/krd", "/bin/krgpg", "/lib/kr-pkcs11.so", "/share/kr", "/Frameworks/krbtle.framework"} {
			rmErr := exec.Command("rm", "-rf", prefix+file).Run()
			if rmErr != nil {
				if os.IsPermission(rmErr) {
					PrintErr(os.Stderr, "sudo rm -rf "+prefix+file)
					runCommandWithUserInteraction("sudo", "rm", "-rf", prefix+file)
				}
			}
		}
	}
	runCommandTmuxFriendly("launchctl", "unload", homePlist)
	os.Remove(homePlist)
	uninstallCodesigning()
	PrintErr(os.Stderr, "Krypton uninstalled. If you experience any issues, please refer to https://krypt.co/docs/start/installation.html#uninstalling-kr")
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
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "upgrade", nil, nil)
	}()
	confirmOrFatal(os.Stderr, "Upgrade Krypton on this workstation?")
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
