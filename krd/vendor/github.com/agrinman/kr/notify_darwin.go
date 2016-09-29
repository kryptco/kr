package kr

import (
	"github.com/deckarep/gosx-notifier"
)

func DesktopNotify(message string) (err error) {
	note := gosxnotifier.NewNotification(message)
	note.Title = "kr"
	note.Group = "co.krypt.krd " + message
	note.AppIcon = "/usr/local/share/krssh/kr.png"
	err = note.Push()
	return
}
