package kr

import (
	"fmt"
	"net"
	"os"
)

const AGENT_SOCKET_FILENAME = "krd-agent.sock"

func Listen(socketPath string) (listener net.Listener, err error) {
	//	delete UNIX socket in case daemon was not killed cleanly
	_ = os.Remove(socketPath)
	listener, err = net.Listen("unix", socketPath)
	return
}

func AgentListen() (listener net.Listener, err error) {
	socketPath, err := KrDirFile(AGENT_SOCKET_FILENAME)
	if err != nil {
		return
	}

	return Listen(socketPath)
}

const DAEMON_SOCKET_FILENAME = "krd.sock"

func SocketPath(fileName string) (socketPath string, err error) {
	return KrDirFile(DAEMON_SOCKET_FILENAME)
}

func DaemonListen() (listener net.Listener, err error) {
	socketPath, err := SocketPath(DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}
	return Listen(socketPath)
}

const HOST_AUTH_FILENAME = "krd-hostauth.sock"

func HostAuthListen() (listener net.Listener, err error) {
	socketPath, err := SocketPath(HOST_AUTH_FILENAME)
	if err != nil {
		return
	}
	//	delete UNIX socket in case daemon was not killed cleanly
	_ = os.Remove(socketPath)
	listener, err = net.Listen("unix", socketPath)
	return
}

func HostAuthDial() (conn net.Conn, err error) {
	socketPath, err := SocketPath(HOST_AUTH_FILENAME)
	if err != nil {
		return
	}
	conn, err = Dial(socketPath)
	return
}

func DaemonDial() (conn net.Conn, err error) {
	socketPath, err := SocketPath(DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}

	return Dial(socketPath)
}

func Dial(unixFile string) (conn net.Conn, err error) {
	conn, err = net.Dial("unix", unixFile)
	if err != nil {
		err = fmt.Errorf("Failed to connect to Kryptonite daemon. Please make sure it is running by typing \"kr restart\".")
	}
	return
}
