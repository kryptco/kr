package main

import (
	"github.com/kryptco/kr"
	sigchain "github.com/kryptco/kr/sigchaingobridge"

	"github.com/urfave/cli"
)

func addCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "add", nil, nil)
	}()
	sigchain.KrAdd()
	return
}

func listCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "ls", nil, nil)
	}()
	sigchain.KrList()
	return
}

func removeCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "rm", nil, nil)
	}()
	sigchain.KrRm()
	return
}
