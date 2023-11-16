// +build !windows

package daemon

import (
	"net"
	"os"
	"strings"
)

func getOriginalAgentConn() (net.Conn, error) {
	originalAgentSock := os.Getenv("SSH_AUTH_SOCK")
	if strings.HasSuffix(originalAgentSock, "krd-agent.sock") {
		return nil, nil
	}

	return net.Dial("unix", originalAgentSock)
}
