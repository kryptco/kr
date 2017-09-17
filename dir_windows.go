// +build windows

package kr

import (
	"os"
	"os/user"
	"path/filepath"
)

//	Find home directory of logged-in user
func UnsudoedHomeDir() (home string) {
	currentUser, err := user.Current()
	if err == nil && currentUser != nil {
		home = currentUser.HomeDir
	} else {
		log.Notice("falling back to $HOME")
		home = os.Getenv("HOME")
		err = nil
	}
	return
}

func KrDir() (krPath string, err error) {
	home := UnsudoedHomeDir()
	if err != nil {
		return
	}
	krPath = filepath.Join(home, "appdata", "local", "Kryptco", "kr")
	err = os.MkdirAll(krPath, os.FileMode(0700))
	return
}

func NotifyDir() (notifyPath string, err error) {
	krDir, err := KrDir()
	if err != nil {
		return
	}
	notifyPath = filepath.Join(krDir, "notify")
	err = os.MkdirAll(notifyPath, os.FileMode(0700))
	return
}
