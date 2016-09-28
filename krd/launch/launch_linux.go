package launch

import (
	"errors"
	"fmt"
	"github.com/coreos/go-systemd/activation"
	"net"
)

func OpenAuthAndCtlSockets() (authSocket net.Listener, ctlSocket net.Listener, err error) {
	listeners, err := activation.Listeners(true)
	if err != nil {
		return
	}
	if len(listeners) != 2 {
		err = errors.New(fmt.Sprintf("Expected 2 systemd listeners, found %d", len(listeners)))
		return
	}
	authSocket = listeners[0]
	ctlSocket = listeners[1]
	if authSocket == nil || ctlSocket == nil {
		err = errors.New("found nil socket")
		return
	}
	return
}
