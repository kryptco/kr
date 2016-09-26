package krssh

import (
	"github.com/0xAX/notificator"
)

func DesktopNotify(message string) (err error) {
	notify := notificator.New(notificator.Options{
		AppName: "krssh-agent",
	})

	notify.Push("krssh-agent", message, "", notificator.UR_CRITICAL)
	return
}
