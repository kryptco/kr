package main

import (
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
