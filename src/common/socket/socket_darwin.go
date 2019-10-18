package socket

import (
	"fmt"
	"net"
)

func DaemonDial(unixFile string) (conn net.Conn, err error) {
	conn, err = net.Dial("unix", unixFile)
	if err != nil {
		err = fmt.Errorf("Failed to connect to Krypton daemon. Please make sure it is running by typing \"kr restart\".")
	}
	return
}
