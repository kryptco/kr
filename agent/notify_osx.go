//+build darwin
package main

import (
	"github.com/deckarep/gosx-notifier"
)

func DesktopNotify(message string) (err error) {
	note := gosxnotifier.NewNotification(message)
	note.Title = "krssh-agent"
	note.Group = "co.krypt.krssh " + message
	note.AppIcon = "kryptonite.png"
	err = note.Push()
	return
}
