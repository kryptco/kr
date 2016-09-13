package notificator

import (
	"os/exec"
	"runtime"
)

type Options struct {
	DefaultIcon string
	AppName     string
}

const (
	UR_NORMAL   =	"normal"
	UR_CRITICAL	=	"critical"
)

type notifier interface {
	push(title string, text string, iconPath string) *exec.Cmd
	pushCritical(title string, text string, iconPath string) *exec.Cmd
}

type Notificator struct {
	notifier    notifier
	defaultIcon string
}

func (n Notificator) Push(title string, text string, iconPath string, urgency string) error {
	icon := n.defaultIcon

	if iconPath != "" {
		icon = iconPath
	}

	if urgency == UR_CRITICAL {
		return n.notifier.pushCritical(title, text, icon).Run()
	}
	
	return n.notifier.push(title, text, icon).Run()
	
}

type osxNotificator struct {
	AppName string
}

func (o osxNotificator) push(title string, text string, iconPath string) *exec.Cmd {
	return exec.Command("growlnotify", "-n", o.AppName, "--image", iconPath, "-m", title)
}

// Causes the notification to stick around until clicked.
func (o osxNotificator) pushCritical(title string, text string, iconPath string) *exec.Cmd {
	return exec.Command("notify-send", "-i", iconPath, title, text, "--sticky", "-p", "2")	
}

type linuxNotificator struct{}

func (l linuxNotificator) push(title string, text string, iconPath string) *exec.Cmd {
	return exec.Command("notify-send", "-i", iconPath, title, text)
}

// Causes the notification to stick around until clicked.
func (l linuxNotificator) pushCritical(title string, text string, iconPath string) *exec.Cmd {
	return exec.Command("notify-send", "-i", iconPath, title, text, "-u", "critical")	
}

type windowsNotificator struct{}

func (w windowsNotificator) push(title string, text string, iconPath string) *exec.Cmd {
	return exec.Command("growlnotify", "/i:", iconPath, "/t:", title, text)
}

// Causes the notification to stick around until clicked.
func (w windowsNotificator) pushCritical(title string, text string, iconPath string) *exec.Cmd {
	return exec.Command("notify-send", "-i", iconPath, title, text, "/s", "true", "/p", "2")	
}


func New(o Options) *Notificator {

	var notifier notifier

	switch runtime.GOOS {

	case "darwin":
		notifier = osxNotificator{AppName: o.AppName}
	case "linux":
		notifier = linuxNotificator{}
	case "windows":
		notifier = windowsNotificator{}

	}

	return &Notificator{notifier: notifier, defaultIcon: o.DefaultIcon}
}
