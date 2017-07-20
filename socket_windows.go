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

	"gopkg.in/natefinch/npipe.v2"
)

//	Find home directory of logged-in user even when run as sudo
func UnsudoedHomeDir() (home string) {
	user, err := user.Current()
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
	krPath = filepath.Join(home, "appdata", "local", "kryptco", "kr")
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

func KrPipeFile(file string) (fullPath string, err error) {
	fullPath = `\\.\pipe\` + file
	return
}

const AGENT_SOCKET_FILENAME = `krdagent`

func AgentListen() (listener net.Listener, err error) {
	socketPath, err := KrPipeFile(AGENT_SOCKET_FILENAME)
	if err != nil {
		return
	}
	//	delete UNIX socket in case daemon was not killed cleanly
	// _ = os.Remove(socketPath)
	listener, err = npipe.Listen(socketPath)
	return
}

const DAEMON_SOCKET_FILENAME = `krd`

func DaemonListen() (listener net.Listener, err error) {
	socketPath, err := KrPipeFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}

	listener, err = npipe.Listen(socketPath)
	return
}

const HOST_AUTH_FILENAME = `krdhost`

func HostAuthListen() (listener net.Listener, err error) {
	socketPath, err := KrPipeFile(HOST_AUTH_FILENAME)
	if err != nil {
		return
	}

	listener, err = npipe.Listen(socketPath)
	return
}

func HostAuthDial() (conn net.Conn, err error) {
	socketPath, err := KrPipeFile(HOST_AUTH_FILENAME)
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

func DaemonDial(unixFile string) (conn net.Conn, err error) {
	conn, err = npipe.Dial(unixFile)
	if err != nil {
		err = fmt.Errorf("Failed to connect to Kryptonite daemon. Please make sure it is running.")
	}
	return
}

func DaemonSocket() (unixFile string, err error) {
	return KrPipeFile(DAEMON_SOCKET_FILENAME)
}

func DaemonSocketOrFatal() (unixFile string) {
	unixFile, err := DaemonSocket()
	if err != nil {
		log.Fatal("Could not open connection to daemon. Make sure it is running.")
	}
	return
}
