package krssh

import (
	"net"
	"os"
	"path/filepath"
)

const KRSSH_CTL_SOCK_ENV = "KRSSH_CTL_SOCK"

func krDirFile(file string) (fullPath string, err error) {
	krPath := filepath.Join(os.Getenv("HOME"), ".kr")
	err = os.MkdirAll(krPath, os.FileMode(0700))
	fullPath = filepath.Join(krPath, file)
	return
}

const DAEMON_SOCKET_FILENAME = "krsshd.sock"

func DaemonListen() (listener net.Listener, err error) {
	socketPath, err := krDirFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}
	listener, err = net.Listen("unix", socketPath)
	return
}

func DaemonDial() (conn net.Conn, err error) {
	socketPath, err := krDirFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}
	conn, err = net.Dial("unix", socketPath)
	return
}
