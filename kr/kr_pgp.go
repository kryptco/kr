package main

import (
	"bytes"
	"os"
	"os/exec"
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

func onboardAutoCommitSign() {
	os.Stderr.WriteString("Would you like to enable " + kr.Cyan("automatic commit signing") + "? [y/n]")
	in := []byte{0, 0}
	os.Stdin.Read(in)
	if in[0] == 'y' {
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

func onboardGPG_TTY() {
	if os.Getenv("GPG_TTY") == "" {
		os.Stderr.WriteString("\r\n" + kr.Red("WARNING:") + " In order to see Kryptonite log messages when requesting a git signature, add " + kr.Yellow("export GPG_TTY=$(tty)") + " to your shell startup (~/.bashrc, ~/.bash_profile, ~/.zshrc, etc.) and restart your terminal.\r\n")
		os.Stderr.WriteString("Press " + kr.Cyan("ENTER") + " to continue")
		os.Stdin.Read([]byte{0})
		os.Stderr.WriteString("\r\n")
	}
}

func onboardKeyServerUpload(pk string) {
	os.Stderr.WriteString("In order for other people to verify your commits, they need to be able to download your public key. Would you like to " + kr.Cyan("upload your public key to the MIT keyserver") + "? [y/n]")
	in := []byte{0, 0}
	os.Stdin.Read(in)
	if in[0] == 'y' {
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

func onboardLocalGPG(me kr.Profile) {
	if !hasGPG() {
		return
	}
	pkFp, err := me.PGPPublicKeySHA1Fingerprint()
	if err != nil {
		os.Stderr.WriteString(kr.Red("Failed to create key fingerprint:\r\n" + err.Error() + "\r\n"))
		return
	}
	pk, _ := me.AsciiArmorPGPPublicKey()

	os.Stderr.WriteString("In order to verify your own commits, you must add your key to gpg locally. Would you like to " + kr.Cyan("add your public key to gpg") + "? [y/n]")
	in := []byte{0, 0}
	os.Stdin.Read(in)
	if in[0] == 'y' {
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
