package kr

import (
	"fmt"
	"net"
	"os/exec"
	"time"
	"os"
)

func DaemonDial(unixFile string) (conn net.Conn, err error) {
	if runningErr := exec.Command("pgrep", "krd").Run(); runningErr != nil {
		os.Stderr.WriteString(Yellow("Kryptonite ▶ Restarting krd...\r\n"))
		exec.Command("nohup", "/usr/bin/krd", "&").Start()
		<-time.After(time.Second)
	}
	conn, err = net.Dial("unix", unixFile)
	if err != nil {
		//	restart then try again
		os.Stderr.WriteString(Yellow("Kryptonite ▶ Restarting krd...\r\n"))
		exec.Command("pkill", "krd").Start()
		exec.Command("nohup", "/usr/bin/krd", "&").Run()
		<-time.After(time.Second)
		conn, err = net.Dial("unix", unixFile)
	}
	if err != nil {
		err = fmt.Errorf("Failed to connect to Kryptonite daemon. Please make sure it is running by typing \"kr restart\".")
	}
	return
}
