package kr

import (
	"fmt"
	"gopkg.in/natefinch/npipe.v2"
	"net"
)

func PipePath(file string) (pipePath string, err error) {
	pipePath = `\\.\pipe\` + file
	return
}

const AGENT_SOCKET_FILENAME = `krdagent`

func AgentListen() (listener net.Listener, err error) {
	socketPath, err := PipePath(AGENT_SOCKET_FILENAME)
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
	socketPath, err := PipePath(DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}

	listener, err = npipe.Listen(socketPath)
	return
}

const HOST_AUTH_FILENAME = `krdhost`

func HostAuthListen() (listener net.Listener, err error) {
	socketPath, err := PipePath(HOST_AUTH_FILENAME)
	if err != nil {
		return
	}

	listener, err = npipe.Listen(socketPath)
	return
}

func HostAuthDial() (conn net.Conn, err error) {
	socketPath, err := PipePath(HOST_AUTH_FILENAME)
	if err != nil {
		return
	}

	conn, err = Dial(socketPath)
	return
}

func DaemonDial() (conn net.Conn, err error) {
	socketPath, err := PipePath(DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}

	conn, err = Dial(socketPath)
	return
}

func Dial(unixFile string) (conn net.Conn, err error) {
	conn, err = npipe.Dial(unixFile)
	if err != nil {
		err = fmt.Errorf("Failed to connect to Kryptonite daemon. Please make sure it is running.")
	}
	return
}
