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
func UnsudoedHomeDir() (home string) {
	userName := os.Getenv("SUDO_USER")
	if userName == "" {
		userName = os.Getenv("USER")
	}
	user, err := user.Lookup(userName)
	if err == nil && user != nil {
		home = user.HomeDir
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

func NotifyDirFile(file string) (fullPath string, err error) {
	notifyPath, err := NotifyDir()
	if err != nil {
		return
	}
	fullPath = filepath.Join(notifyPath, file)
	return
}

func KrDirFile(file string) (fullPath string, err error) {
	krPath, err := KrDir()
	if err != nil {
		return
	}
	fullPath = filepath.Join(krPath, file)
	return
}

const AGENT_SOCKET_FILENAME = "krd-agent.sock"

func AgentListen() (listener net.Listener, err error) {
	socketPath, err := KrDirFile(AGENT_SOCKET_FILENAME)
	if err != nil {
		return
	}
	//	delete UNIX socket in case daemon was not killed cleanly
	_ = os.Remove(socketPath)
	listener, err = net.Listen("unix", socketPath)
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

const HOST_AUTH_FILENAME = "krd-hostauth.sock"

func HostAuthListen() (listener net.Listener, err error) {
	socketPath, err := KrDirFile(HOST_AUTH_FILENAME)
	if err != nil {
		return
	}
	//	delete UNIX socket in case daemon was not killed cleanly
	_ = os.Remove(socketPath)
	listener, err = net.Listen("unix", socketPath)
	return
}

func HostAuthDial() (conn net.Conn, err error) {
	socketPath, err := KrDirFile(HOST_AUTH_FILENAME)
	if err != nil {
		return
	}
	conn, err = DaemonDial(socketPath)
	return
}

func pingDaemon(unixFile string) (err error) {
	conn, err := DaemonDial(unixFile)
	if err != nil {
		return
	}
	defer conn.Close()

	pingRequest, err := http.NewRequest("GET", "/ping", nil)
	if err != nil {
		return
	}
	err = pingRequest.Write(conn)
	if err != nil {
		return
	}
	responseReader := bufio.NewReader(conn)
	_, err = http.ReadResponse(responseReader, pingRequest)
	if err != nil {
		err = fmt.Errorf("Daemon Read error: %s", err.Error())
		return
	}
	return
}

func DaemonDialWithTimeout(unixFile string) (conn net.Conn, err error) {
	done := make(chan error, 1)
	go func() {
		done <- pingDaemon(unixFile)
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

	conn, err = DaemonDial(unixFile)
	return
}

func DaemonSocketOrFatal() (unixFile string) {
	unixFile, err := KrDirFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		log.Fatal("Could not open connection to daemon. Make sure it is running by typing \"kr restart\".")
	}
	return
}
