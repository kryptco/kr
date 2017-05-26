package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kryptco/kr"
)

func onboardGithub(pk string) {
	os.Stderr.WriteString("Would you like to add this key to " + kr.Cyan("GitHub") + "? [y/n]")
	in := []byte{0, 0}
	os.Stdin.Read(in)
	if in[0] == 'y' {
		_, err := copyPGPKeyNonFatalClipboard()
		if err == nil {
			os.Stderr.WriteString("Your PGP public key has been " + kr.Cyan("copied to your clipboard.") + "\r\n")
			<-time.After(1000 * time.Millisecond)
			os.Stderr.WriteString("Press " + kr.Cyan("ENTER") + " to open your browser to GitHub settings. Then click " + kr.Cyan("New GPG key") + " and paste your Kryptonite PGP public key.\r\n")
			os.Stdin.Read([]byte{0})
			openBrowser("https://github.com/settings/keys")
		} else {
			os.Stderr.WriteString(kr.Cyan("Press ENTER to print your Kryptonite PGP public key\r\n"))
			os.Stdin.Read([]byte{0})
			os.Stdout.WriteString(pk)
			os.Stdout.WriteString("\r\n\r\n")
			<-time.After(500 * time.Millisecond)
			os.Stderr.WriteString("Copy and paste your PGP public key in GitHub at " + kr.Yellow("https://github.com/settings/keys\r\n"))
		}
	}
}

func onboardAutoCommitSign(interactive bool) {
	var autoSign bool
	if interactive {
		os.Stderr.WriteString("Would you like to enable " + kr.Cyan("automatic commit signing") + "? [y/n]")
		in := []byte{0, 0}
		os.Stdin.Read(in)
		if in[0] == 'y' {
			autoSign = true
		}
	}
	if autoSign || !interactive {
		err := exec.Command("git", "config", "--global", "commit.gpgSign", "true").Run()
		if err != nil {
			PrintErr(os.Stderr, err.Error()+"\r\n")
		}
		os.Stderr.WriteString(kr.Green("Automatic commit signing enabled ✔ ") + " disable by running " + kr.Cyan("git config --global --unset commit.gpgSign") + "\r\n")
	} else {
		os.Stderr.WriteString("You can manually create a signed git commit by running " + kr.Cyan("git commit -S") + "\r\n")
	}
	<-time.After(500 * time.Millisecond)
}

func shellRCFileAndGPG_TTYExport() (file string, export string) {
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return filepath.Join(os.Getenv("HOME"), ".zshrc"), "export GPG_TTY=$(tty)"
	} else if strings.Contains(shell, "bash") {
		return filepath.Join(os.Getenv("HOME"), ".bashrc"), "export GPG_TTY=$(tty)"
	} else if strings.Contains(shell, "ksh") {
		return filepath.Join(os.Getenv("HOME"), ".kshrc"), "export GPG_TTY=$(tty)"
	} else if strings.Contains(shell, "csh") {
		return filepath.Join(os.Getenv("HOME"), ".cshrc"), "setenv GPG_TTY `tty`"
	} else if strings.Contains(shell, "fish") {
		return filepath.Join(os.Getenv("HOME"), ".config", "fish", "config.fish"), "set -x GPG_TTY (tty)"
	} else {
		return filepath.Join(os.Getenv("HOME"), "/.profile"), "export GPG_TTY=$(tty)"
	}
}

func addGPG_TTYExportToCurrentShellIfNotPresent() {
	path, cmd := shellRCFileAndGPG_TTYExport()
	rcContents, err := ioutil.ReadFile(path)
	if err == nil {
		if strings.Contains(string(rcContents), cmd) {
			return
		}
	}
	rcFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return
	}
	//	seek to end
	rcFile.Seek(0, 2)
	rcFile.WriteString(cmd + "\n")
	rcFile.Close()
}

func onboardGPG_TTY(interactive bool) {
	if os.Getenv("GPG_TTY") != "" {
		return
	}
	if interactive {
		os.Stderr.WriteString("\r\n" + kr.Red("WARNING:") + " In order to see Kryptonite log messages when requesting a git signature, add " + kr.Yellow("export GPG_TTY=$(tty)") + " to your shell startup (~/.bashrc, ~/.bash_profile, ~/.zshrc, etc.) and restart your terminal.\r\n")
		os.Stderr.WriteString("Press " + kr.Cyan("ENTER") + " to continue")
		os.Stdin.Read([]byte{0})
		os.Stderr.WriteString("\r\n")
	} else {
		addGPG_TTYExportToCurrentShellIfNotPresent()
	}
}

func onboardKeyServerUpload(interactive bool, pk string) {
	var uploadKey bool
	if interactive {
		os.Stderr.WriteString("In order for other people to verify your commits, they need to be able to download your public key. Would you like to " + kr.Cyan("upload your public key to the MIT keyserver") + "? [y/n]")
		in := []byte{0, 0}
		os.Stdin.Read(in)
		if in[0] == 'y' {
			uploadKey = true
		}
	}
	if uploadKey || !interactive {
		cmd := exec.Command("curl", "https://pgp.mit.edu/pks/add", "-f", "--data-urlencode", "keytext="+pk)
		output, err := cmd.CombinedOutput()
		if err == nil {
			os.Stderr.WriteString(kr.Green("Key uploaded ✔\r\n"))
		} else {
			os.Stderr.WriteString(kr.Red("Failed to upload key, curl output:\r\n" + string(output) + "\r\n"))
		}
	}
}

func hasGPG() bool {
	err := exec.Command("gpg", "--help").Run()
	return err == nil
}

func onboardLocalGPG(interactive bool, me kr.Profile) {
	if !hasGPG() {
		return
	}
	var importKey bool
	if interactive {
		os.Stderr.WriteString("In order to verify your own commits, you must add your key to gpg locally. Would you like to " + kr.Cyan("add your public key to gpg") + "? [y/n]")
		in := []byte{0, 0}
		os.Stdin.Read(in)
		if in[0] == 'y' {
			importKey = true
		}
	}
	if importKey || !interactive {
		pkFp, err := me.PGPPublicKeySHA1Fingerprint()
		if err != nil {
			os.Stderr.WriteString(kr.Red("Failed to create key fingerprint:\r\n" + err.Error() + "\r\n"))
			return
		}
		pk, _ := me.AsciiArmorPGPPublicKey()

		cmdImport := exec.Command("gpg", "--import", "--armor")
		cmdImport.Stdin = bytes.NewReader([]byte(pk))
		importOutput, err := cmdImport.CombinedOutput()
		if err != nil {
			os.Stderr.WriteString(kr.Red("Failed to import key, gpg output:\r\n" + string(importOutput) + "\r\n"))
			return
		}

		cmdTrust := exec.Command("gpg", "--import-ownertrust")
		cmdTrust.Stdin = bytes.NewReader([]byte(pkFp + ":6:\r\n"))
		output, err := cmdTrust.CombinedOutput()
		if err == nil {
			os.Stderr.WriteString(kr.Green("Key imported ✔\r\n"))
		} else {
			os.Stderr.WriteString(kr.Red("Failed to import key, gpg output:\r\n" + string(output) + "\r\n"))
		}
	}
}
