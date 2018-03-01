package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kryptco/kr"
	"github.com/kryptco/kr/krdclient"
	"github.com/urfave/cli"

	"net/mail"
)

func exitIfNotOnTeam() {
	me, err := krdclient.RequestMe()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}

	if !me.IsOnTeam() {
		fmt.Println(kr.Red("This is a Krypton for Teams feature, but you're not on a team yet."))
		PrintFatal(os.Stderr, kr.Yellow("To get started with Krypton for Teams, go to https://www.krypt.co/docs/teams/getting-started.html"))
	}
}

func createTeamCommand(c *cli.Context) (err error) {
	fmt.Println(kr.Yellow("Creating a team is not yet supported from the command line."))
	fmt.Println()
	fmt.Println(kr.Yellow("Open the Krypton app and tap 'Create my team` on the Teams tab."))
	fmt.Println()
	fmt.Println(kr.Magenta("Learn more here to get started: https://www.krypt.co/docs/teams/getting-started.html"))
	return
}

func setTeamNameCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	_, err = krdclient.RequestMe()
	if err != nil {
		PrintFatal(os.Stderr, kr.Red("Kryptonite ▶ "+err.Error()))
		return
	}

	name := c.String("name")
	if name == "" {
		PrintFatal(os.Stderr, "--name flag required")
	}
	kr.SetTeamName(name)
	return
}

func createInviteCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	emails := strings.Split(c.String("emails"), ",")
	domain := c.String("domain")
	if domain != "" {

		domain := c.String("domain")
		if domain == "" {
			PrintFatal(os.Stderr, "Supply a valid email domain")
			return
		}
		_, err := mail.ParseAddress("x@" + domain)
		if err != nil {
			PrintFatal(os.Stderr, "Email domain "+domain+" is invalid.")
			return err
		}

		kr.InviteDomain(domain)

	} else if len(emails) > 0 {
		var verifed_addresses []string

		for _, email := range emails {
			address, err := mail.ParseAddress(email)
			if err != nil {
				PrintFatal(os.Stderr, "Email "+email+" is invalid.")
				return err
			}
			verifed_addresses = append(verifed_addresses, address.Address)
		}

		kr.InviteEmails(verifed_addresses)

	} else {
		PrintFatal(os.Stderr, "--emails or --domain are required")
	}

	return
}

func closeInvitationsCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	kr.CancelInvite()
	return
}

func viewLogs(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	query := c.String("q")
	kr.ViewLogs(query)
	return
}

func setPolicyCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

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

func getMembersCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	var query *string
	if c.String("query") != "" {
		queryStr := c.String("query")
		query = &queryStr
	}
	kr.GetMembers(query, c.Bool("ssh"), c.Bool("pgp"))
	return
}

func addAdminCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	email := c.String("email")
	if email == "" {
		PrintFatal(os.Stderr, "--email required")
	}
	kr.AddAdmin(email)
	return
}

func removeAdminCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	email := c.String("email")
	if email == "" {
		PrintFatal(os.Stderr, "--email required")
	}
	kr.RemoveAdmin(email)
	return
}

func getAdminsCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	kr.GetAdmins()
	return
}

func pinHostKeyCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	var publicKey string

	// check if input is from stdin
	fi, _ := os.Stdin.Stat()
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		reader := bufio.NewReader(os.Stdin)
		publicKeyStdin, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		publicKey = publicKeyStdin
	} else if c.String("public-key") != "" {
		publicKey = c.String("public-key")
	} else {
		kr.PinKnownHostKeys(c.String("host"), c.Bool("update-from-server"))
		return
	}

	pk, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		PrintFatal(os.Stderr, "error decoding public-key, make sure it is base64 encoded without the key type prefix (i.e. no 'ssh-rsa' or 'ssh-ed25519') "+err.Error())
	}
	kr.PinHostKey(c.String("host"), pk)
	return
}

func unpinHostKeyCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	var publicKey string

	// check if input is from stdin
	fi, _ := os.Stdin.Stat()
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		reader := bufio.NewReader(os.Stdin)
		publicKeyStdin, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		publicKey = publicKeyStdin
	} else if c.String("public-key") != "" {
		publicKey = c.String("public-key")
	} else {
		PrintFatal(os.Stderr, "you must supply a public-key")
		return
	}

	pk, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		PrintFatal(os.Stderr, "error decoding public-key, make sure it is base64 encoded without the key type prefix (i.e. no 'ssh-rsa' or 'ssh-ed25519') "+err.Error())
	}
	kr.UnpinHostKey(c.String("host"), pk)
	return
}

func listPinnedKeysCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	host := c.String("host")
	if host == "" {
		kr.GetAllPinnedHostKeys()
		return
	}
	kr.GetPinnedHostKeys(host, c.Bool("search"))
	return
}

func enableLoggingCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	kr.EnableLogging()
	return
}

func logsCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	kr.UpdateTeamLogs()
	return
}

func teamBillingCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	kr.OpenBilling()
	return
}
