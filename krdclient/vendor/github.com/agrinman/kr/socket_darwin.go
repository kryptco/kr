package kr

import (
	"fmt"
	"net"
)

func DaemonDial(unixFile string) (conn net.Conn, err error) {
	socketPath, err := KrDirFile(DAEMON_SOCKET_FILENAME)
	if err == nil {
		conn, err = net.Dial("unix", socketPath)
	}
	if err != nil {
		err = fmt.Errorf("Failed to connect to Kryptonite daemon. Please make sure it is running by typing \"kr restart\".")
	}
	return
}
