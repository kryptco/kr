// +build !windows

package kr

import (
	"os"
	"os/user"
	"path/filepath"
)

//	Find home directory of logged-in user even when run as sudo
func UnsudoedHomeDir() (home string) {
	userName := os.Getenv("SUDO_USER")
	if userName == "" {
		userName = os.Getenv("USER")
	}
	currentUser, err := user.Lookup(userName)
	if err == nil && currentUser != nil {
		home = currentUser.HomeDir
	} else {
		log.Notice("falling back to $HOME")
		home = os.Getenv("HOME")
		err = nil
	}
	if os.Getenv("HOME") != home {
		os.Setenv("HOME", home)
	}
	return
}

func KrDir() (krPath string, err error) {
	home := UnsudoedHomeDir()
	if err != nil {
		return
	}
	krPath = filepath.Join(home, ".kr")
	err = os.MkdirAll(krPath, os.FileMode(0700))
	return
}

func NotifyDir() (krPath string, err error) {
	home := UnsudoedHomeDir()
	if err != nil {
		return
	}
	krPath = filepath.Join(home, ".kr", "notify")
	err = os.MkdirAll(krPath, os.FileMode(0700))
	return
}
