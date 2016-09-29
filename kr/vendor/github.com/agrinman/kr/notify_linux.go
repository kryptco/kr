package kr

import (
	"github.com/0xAX/notificator"
)

func DesktopNotify(message string) (err error) {
	notify := notificator.New(notificator.Options{
		AppName: "kr",
	})

	notify.Push("kr", message, "", notificator.UR_CRITICAL)
	return
}
