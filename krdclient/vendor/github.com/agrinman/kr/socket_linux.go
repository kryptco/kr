package kr

import(
	"net"
	"fmt"
	"time"
	"os/exec"
)

func DaemonDial() (conn net.Conn, err error) {
	socketPath, err := KrDirFile(DAEMON_SOCKET_FILENAME)
	if err == nil {
		conn, err = net.Dial("unix", socketPath)
	}
	if err != nil {
		//	restart then try again
		exec.Command("systemctl", "--user", "enable", "kr").Run()
		exec.Command("systemctl", "--user", "stop", "kr").Run()
		exec.Command("systemctl", "--user", "start", "kr").Run()
		<-time.After(time.Second)
		socketPath, err = KrDirFile(DAEMON_SOCKET_FILENAME)
		if err == nil {
			conn, err = net.Dial("unix", socketPath)
		} else {
			err = fmt.Errorf("Failed to connect to Kryptonite daemon. Please make sure it is running by typing \"kr restart\".")
		}
	}
	return
}
