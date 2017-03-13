package kr

import (
	"fmt"
	"net"
	"os/exec"
	"time"
)

func DaemonDial(unixFile string) (conn net.Conn, err error) {
	conn, err = net.Dial("unix", unixFile)
	if err != nil {
		//	restart then try again
		exec.Command("systemctl", "--user", "disable", "kr").Run()
		exec.Command("systemctl", "--user", "enable", "kr").Run()
		exec.Command("systemctl", "--user", "stop", "kr").Run()
		exec.Command("systemctl", "--user", "start", "kr").Run()
		<-time.After(time.Second)
		conn, err = net.Dial("unix", unixFile)
	}
	if err != nil {
		err = fmt.Errorf("Failed to connect to Kryptonite daemon. Please make sure it is running by typing \"kr restart\".")
	}
	return
}
