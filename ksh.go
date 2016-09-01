package main

import (
	"github.com/urfave/cli"
	"os"
)

func pairCommand(c *cli.Context) (err error) {

	return
}

func main() {
	app := cli.NewApp()
	app.Name = "ksh"
	app.Usage = "communicate with krssh-agent and krssh-iOS"
	app.Flags = []cli.Flag{}
	app.Commands = []cli.Command{
		cli.Command{
			Name:    "pair",
			Aliases: []string{"p"},
			Action:  pairCommand,
		},
	}
	app.Run(os.Args)

}
