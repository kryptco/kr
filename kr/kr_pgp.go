package main

import (
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
		os.Stderr.WriteString("Automatic commit signing enabled. Disable by running " + kr.Cyan("git config --global --unset commit.gpgSign") + "\r\n")
	} else {
		os.Stderr.WriteString("You can manually create a signed git commit by running " + kr.Cyan("git commit -S") + "\r\n")
	}
}

func onboardGPG_TTY() {
	if os.Getenv("GPG_TTY") == "" {
		os.Stderr.WriteString("In order to see Kryptonite log messages when requesting a git signature, add " + kr.Yellow("export GPG_TTY=$(tty)") + " to your shell startup (~/.bashrc, ~/.bash_profile, ~/.zshrc, etc.) and restart your terminal.\r\n")
	}
}
