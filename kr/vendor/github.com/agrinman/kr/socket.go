package kr

import (
	"net"
	"os"
	"os/user"
	"path/filepath"
)

//	Find home directory of logged-in user even when run as sudo
func KrDirFile(file string) (fullPath string, err error) {
	userName := os.Getenv("SUDO_USER")
	if userName == "" {
		userName = os.Getenv("USER")
	}
	user, err := user.Lookup(userName)
	var userHome string
	if err == nil && user != nil {
		userHome = user.HomeDir
	} else {
		log.Notice("falling back to $HOME")
		userHome = os.Getenv("HOME")
		err = nil
	}

	krPath := filepath.Join(userHome, ".kr")
	err = os.MkdirAll(krPath, os.FileMode(0700))
	fullPath = filepath.Join(krPath, file)
	return
}

const DAEMON_SOCKET_FILENAME = "krd.sock"

func DaemonListen() (listener net.Listener, err error) {
	socketPath, err := KrDirFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}
	//	delete UNIX socket in case daemon was not killed cleanly
	_ = os.Remove(socketPath)
	listener, err = net.Listen("unix", socketPath)
	return
}
