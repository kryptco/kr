package daemon

import (
	"github.com/Microsoft/go-winio"
	"net"
	"os"
	"strings"
)

func getOriginalAgentConn() (net.Conn, error) {
	originalAgentSock := os.Getenv("SSH_AUTH_SOCK")
	if strings.HasSuffix(originalAgentSock, "krd-agent") {
		return nil, nil
	}

	return winio.DialPipe(`\\.\pipe\openssh-ssh-agent`, nil)
}
