package main

import (
	"os"
	"strconv"

	"github.com/kryptco/kr"
	"github.com/urfave/cli"
)

func createTeamCommand(c *cli.Context) (err error) {
	kr.CreateTeam(c.String("name"))
	return
}

func createInviteCommand(c *cli.Context) (err error) {
	kr.CreateInvite()
	return
}

func setPolicyCommand(c *cli.Context) (err error) {
	var window *int64
	if c.String("window") != "" {
		windowInt, err := strconv.ParseInt(c.String("window"), 10, 64)
		if err != nil {
			PrintFatal(os.Stderr, "Could not parse window as integer seconds: "+err.Error())
		}
		window = &windowInt
	}
	kr.SetApprovalWindow(window)
	return
}
