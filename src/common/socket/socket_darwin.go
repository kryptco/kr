package socket

import (
	"fmt"
	"net"
	"os/exec"
)

func DaemonDial(unixFile string) (conn net.Conn, err error) {
	conn, err = net.Dial("unix", unixFile)
	if err != nil {
		err = fmt.Errorf("Failed to connect to Krypton daemon. Please make sure it is running by typing \"kr restart\".")
	}
	return
}

func IsKrdRunning() bool {
	err := exec.Command("pgrep", "-U", User(), "krd").Run()
	return nil == err
}

func AgentListen() (listener net.Listener, err error) {
	return AgentListenUnix()
}
