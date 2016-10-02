package kr

import (
	"net"
	"os"
	"path/filepath"
)

func KrDirFile(file string) (fullPath string, err error) {
	krPath := filepath.Join(os.Getenv("HOME"), ".kr")
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

func DaemonDial() (conn net.Conn, err error) {
	socketPath, err := KrDirFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}
	conn, err = net.Dial("unix", socketPath)
	return
}
