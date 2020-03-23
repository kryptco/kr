// +build windows

package main

import (
	"fmt"
	"github.com/pkg/browser"
	"github.com/urfave/cli"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"

	. "krypt.co/kr/common/analytics"
	. "krypt.co/kr/common/socket"
)

func initTerminal() {
	var m uint32
	windows.GetConsoleMode(windows.Stdout, &m)
	windows.SetConsoleMode(windows.Stdout, m|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}

func restartCommandOptions(c *cli.Context, isUserInitiated bool) (err error) {
	if isUserInitiated {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "restart", nil, nil)
	}

	_ = migrateSSHConfig()

	KillKrd()
	startKrd()

	if isUserInitiated {
		PrintErr(os.Stderr, "Restarted Krypton daemon.")
	}
	return
}

func upgradeCommand(c *cli.Context) (err error) {
	return fmt.Errorf("Upgrade not supported")
}

func uninstallCommand(c *cli.Context) (err error) {
	go func() {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "uninstall", nil, nil)
	}()
	confirmOrFatal(os.Stderr, "Uninstall Krypton from this workstation?")

	cleanSSHConfig()

	KillKrd()

	//uninstallCodesigning()
	PrintErr(os.Stderr, "Krypton uninstalled. If you experience any issues, please refer to https://krypt.co/docs/start/installation.html#uninstalling-kr")
	return
}

func startKrd() (err error) {
	exe := "krd.exe"
	if pfx, err := getPrefix(); err == nil {
		exe = pfx + `\krd.exe`
	}
	cmd := exec.Command(exe)
	return cmd.Start()
}

func openBrowser(url string) {
	err := browser.OpenURL(url)
	if err != nil {
		os.Stderr.WriteString("Unable to open browser, please visit " + url + "\r\n")
	}
}

func killKrd() {
	KillKrd()
}
