// +build !darwin,!windows

package socket

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	util "krypt.co/kr/common/util"
)

func DaemonDial(unixFile string) (conn net.Conn, err error) {
	if !IsKrdRunning() {
		os.Stderr.WriteString(util.Yellow("Krypton ▶ Restarting krd...\r\n"))
		exec.Command("nohup", "krd").Start()
		<-time.After(1 * time.Second)
	}
	conn, err = net.Dial("unix", unixFile)
	if err != nil {
		//	restart then try again
		os.Stderr.WriteString(util.Yellow("Krypton ▶ Restarting krd...\r\n"))
		KillKrd()
		exec.Command("nohup", "krd").Start()
		<-time.After(1 * time.Second)
		conn, err = net.Dial("unix", unixFile)
	}
	if err != nil {
		err = fmt.Errorf("Failed to connect to Krypton daemon. Please make sure it is running by typing \"kr restart\".")
	}
	return
}

func KillKrd() {
	exec.Command("pkill", "-U", User(), "-x", "krd").Run()
	<-time.After(1*time.Second)
}

func IsKrdRunning() bool {
	err := exec.Command("pgrep", "-U", User(), "krd").Run()
	return nil == err
}

func AgentListen() (listener net.Listener, err error) {
	return AgentListenUnix()
}
