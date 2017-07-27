package kr

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
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
	if runningErr := exec.Command("pgrep", "krd").Run(); runningErr != nil {
		os.Stderr.WriteString(Yellow("Kryptonite ▶ Restarting krd...\r\n"))
		exec.Command("nohup", "krd").Start()
		<-time.After(250 * time.Millisecond)
	}
	conn, err = net.Dial("unix", unixFile)
	if err != nil {
		//	restart then try again
		os.Stderr.WriteString(Yellow("Kryptonite ▶ Restarting krd...\r\n"))
		exec.Command("pkill", "krd").Start()
		exec.Command("nohup", "krd").Run()
		<-time.After(250 * time.Millisecond)
		conn, err = net.Dial("unix", unixFile)
	}
	if err != nil {
		err = fmt.Errorf("Failed to connect to Kryptonite daemon. Please make sure it is running by typing \"kr restart\".")
	}
	return
}
