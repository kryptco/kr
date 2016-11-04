package kr

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"time"
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

func pingDaemon() (err error) {
	conn, err := DaemonDial()
	if err != nil {
		return
	}

	pingRequest, err := http.NewRequest("GET", "/ping", nil)
	if err != nil {
		return
	}
	err = pingRequest.Write(conn)
	if err != nil {
		return
	}
	responseReader := bufio.NewReader(conn)
	httpResponse, err := http.ReadResponse(responseReader, pingRequest)
	if err != nil {
		err = fmt.Errorf("Daemon Read error: %s", err.Error())
		return
	}

	if httpResponse.StatusCode != http.StatusOK {
		err = fmt.Errorf("ping error: non-200 status code from daemon")
		return
	}
	return
}

func DaemonDialWithTimeout() (conn net.Conn, err error) {
	done := make(chan error, 1)
	go func() {
		done <- pingDaemon()
	}()

	select {
	case <-time.After(time.Second):
		err = fmt.Errorf("ping timed out")
		return
	case err = <-done:
	}
	if err != nil {
		return
	}

	conn, err = DaemonDial()
	return
}
