// +build !darwin

package kr

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
)

func DaemonDial(unixFile string) (conn net.Conn, err error) {
	if runningErr := exec.Command("pgrep", "krd").Run(); runningErr != nil {
		os.Stderr.WriteString(Yellow("Kryptonite ▶ Restarting krd...\r\n"))
		exec.Command("nohup", "krd").Start()
		<-time.After(250 * time.Millisecond)
	}
	conn, err = net.Dial("unix", unixFile)
	if err != nil {
		//	restart then try again
		os.Stderr.WriteString(Yellow("Kryptonite ▶ Restarting krd...\r\n"))
		exec.Command("killall", "krd").Start()
		exec.Command("nohup", "krd").Run()
		<-time.After(250 * time.Millisecond)
		conn, err = net.Dial("unix", unixFile)
	}
	if err != nil {
		err = fmt.Errorf("Failed to connect to Kryptonite daemon. Please make sure it is running by typing \"kr restart\".")
	}
	return
}
