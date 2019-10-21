package socket

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

func User() string {
	user := os.Getenv("USER")
	if user == "" {
		whoami, err := exec.Command("whoami").Output()
		if err == nil {
			user = strings.TrimSpace(string(whoami))
			os.Setenv("USER", user)
		}
	}
	return user
}

func HomeDir() (home string) {
	user, err := user.Lookup(User())
	if err == nil && user != nil {
		home = user.HomeDir
	} else {
		home = os.Getenv("HOME")
		err = nil
	}
	if os.Getenv("HOME") != home {
		os.Setenv("HOME", home)
	}
	return
}

func KrDir() (krPath string, err error) {
	home := HomeDir()
	if err != nil {
		return
	}
	krPath = filepath.Join(home, ".kr")
	err = os.MkdirAll(krPath, os.FileMode(0700))
	return
}

func NotifyDir() (krPath string, err error) {
	home := HomeDir()
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

func AgentListenUnix() (listener net.Listener, err error) {
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
	case <-time.After(5*time.Second):
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
