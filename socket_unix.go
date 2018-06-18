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
	if !IsKrdRunning() {
		os.Stderr.WriteString(Yellow("Krypton ▶ Restarting krd...\r\n"))
		exec.Command("nohup", "krd").Start()
		<-time.After(250 * time.Millisecond)
	}
	conn, err = net.Dial("unix", unixFile)
	if err != nil {
		//	restart then try again
		os.Stderr.WriteString(Yellow("Krypton ▶ Restarting krd...\r\n"))
		KillKrd()
		exec.Command("nohup", "krd").Run()
		<-time.After(250 * time.Millisecond)
		conn, err = net.Dial("unix", unixFile)
	}
	if err != nil {
		err = fmt.Errorf("Failed to connect to Krypton daemon. Please make sure it is running by typing \"kr restart\".")
	}
	return
}

func KillKrd() {
	exec.Command("killall", "-u", os.Getenv("USER"), "krd").Run()
}
