// +build windows

package socket

import (
	"bytes"
	"fmt"
	"github.com/Microsoft/go-winio"
	"krypt.co/kr/common/util"
	"net"
	"os"
	"os/exec"
	"time"
)

const AGENT_PIPE = "\\\\.\\pipe\\krd-agent"

func AgentListen() (listener net.Listener, err error) {
	listener, err = winio.ListenPipe(AGENT_PIPE, nil)
	return
}

func DaemonDial(unixFile string) (conn net.Conn, err error) {
	if !IsKrdRunning() {
		os.Stderr.WriteString(util.Yellow("Krypton ▶ Restarting krd...\r\n"))
		_ = exec.Command("cmd.exe", "/C", "start", "/b", `krd.exe`).Start()
		<-time.After(1 * time.Second)
	}
	conn, err = net.Dial("unix", unixFile)
	/*
	TODO
	if err != nil {
		//	restart then try again
		os.Stderr.WriteString(Yellow("Krypton ▶ Restarting krd...\r\n"))
		KillKrd()
		exec.Command("nohup", "krd").Start()
		<-time.After(1 * time.Second)
		conn, err = net.Dial("unix", unixFile)
	}
	 */
	if err != nil {
		err = fmt.Errorf("Failed to connect to Krypton daemon. Please make sure it is running by typing \"kr restart\".")
	}
	return
}

func KillKrd() {
	_ = exec.Command("taskkill", "/F", "/FI", `USERNAME eq ` + User(), "/IM", "krd.exe").Run()
	<-time.After(1*time.Second)
}

func IsKrdRunning() bool {
	cmd := exec.Command("tasklist", "/FI", `USERNAME eq ` + User(), "/FI", `IMAGENAME eq krd.exe`)
	if ret, err := cmd.CombinedOutput(); err == nil {
		return bytes.Contains(ret, []byte("krd.exe"))
	}
	return false
}
