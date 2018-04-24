package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kryptco/kr"
	"github.com/kryptco/kr/krdclient"
	sigchain "github.com/kryptco/kr/sigchaingobridge"

	"github.com/urfave/cli"

	"net/mail"
)

func exitIfNotOnTeam() {
	latestKrdRunning, err := krdclient.IsLatestKrdRunning()
	if err != nil || !latestKrdRunning {
		PrintFatal(os.Stderr, krdclient.ErrOldKrdRunning.Error())
	}

	me, err := krdclient.RequestMe()
	if err != nil {
		PrintFatal(os.Stderr, kr.Red("Krypton ▶ "+err.Error()))
	}

	if me.TeamCheckpoint == nil {
		me, err = krdclient.RequestMeForceRefresh(nil)
		if err != nil {
			PrintFatal(os.Stderr, kr.Red("Krypton ▶ "+err.Error()))
		}
	}

	if !me.IsOnTeam() {
		fmt.Println(kr.Red("This is a Krypton for Teams feature, but you're not on a team yet."))
		PrintFatal(os.Stderr, kr.Yellow("To get started with Krypton for Teams, go to https://www.krypt.co/docs/teams/getting-started.html"))
	}
}

func exitIfNotAdmin() {
	latestKrdRunning, err := krdclient.IsLatestKrdRunning()
	if err != nil || !latestKrdRunning {
		PrintFatal(os.Stderr, krdclient.ErrOldKrdRunning.Error())
	}

	if !sigchain.IsAdmin() {
		PrintFatal(os.Stderr, kr.Red("This is a Krypton for Teams admin feature, but you're not an admin."))
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
	exitIfNotAdmin()

	_, err = krdclient.RequestMe()
	if err != nil {
		PrintFatal(os.Stderr, kr.Red("Kryptonite ▶ "+err.Error()))
		return
	}

	name := c.String("name")
	if name == "" {
		PrintFatal(os.Stderr, "--name flag required")
	}
	sigchain.SetTeamName(name)
	return
}

func createInviteCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()
	exitIfNotAdmin()

	emails := strings.Split(c.String("emails"), ",")
	domain := c.String("domain")
	if domain != "" {
		_, err := mail.ParseAddress("x@" + domain)
		if err != nil {
			PrintFatal(os.Stderr, "Email domain "+domain+" is invalid.")
			return err
		}

		sigchain.InviteDomain(domain)

	} else if len(emails) > 0 {
		var verifedAddresses []string

		for _, email := range emails {
			address, err := mail.ParseAddress(email)
			if err != nil {
				PrintFatal(os.Stderr, "Email "+email+" is invalid.")
				return err
			}
			verifedAddresses = append(verifedAddresses, address.Address)
		}

		sigchain.InviteEmails(verifedAddresses)

	} else {
		PrintFatal(os.Stderr, "--emails or --domain are required")
	}

	return
}

func closeInvitationsCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()
	exitIfNotAdmin()

	sigchain.CancelInvite()
	return
}

func removeMemberCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()
	exitIfNotAdmin()

	email := c.String("email")
	if email == "" {
		PrintFatal(os.Stderr, "--email is required")
	}

	address, err := mail.ParseAddress(email)
	if err != nil {
		PrintFatal(os.Stderr, "Email "+email+" is invalid.")
		return err
	}
	sigchain.RemoveMemberCommand(address.Address)
	return
}

func viewLogs(c *cli.Context) (err error) {
	exitIfNotOnTeam()
	exitIfNotAdmin()

	sigchain.ViewLogs()
	return
}

func getPolicyCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	sigchain.GetPolicy()
	return
}

func setPolicyCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()
	exitIfNotAdmin()

	var window *int64
	if !c.Bool("unset") {
		if c.String("window") == "" {
			PrintFatal(os.Stderr, "Please provide an auto-approval window value")
		}
		windowInt, err := strconv.ParseInt(c.String("window"), 10, 64)
		if err != nil {
			PrintFatal(os.Stderr, "Could not parse window as integer minutes: "+err.Error())
		}
		windowInt *= 60
		window = &windowInt
	}
	sigchain.SetApprovalWindow(window)
	return
}

func getMembersCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	var email *string
	if c.String("email") != "" {
		emailStr := c.String("email")
		address, err := mail.ParseAddress(emailStr)
		if err != nil {
			PrintFatal(os.Stderr, "Email "+emailStr+" is invalid.")
			return err
		}
		email = &address.Address
	}
	sigchain.GetMembers(email, c.Bool("ssh"), c.Bool("pgp"), c.Bool("admin"))
	return
}

func addAdminCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()
	exitIfNotAdmin()

	email := c.String("email")
	if email == "" {
		PrintFatal(os.Stderr, "--email required")
	}
	sigchain.AddAdmin(email)
	return
}

func removeAdminCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()
	exitIfNotAdmin()

	email := c.String("email")
	if email == "" {
		PrintFatal(os.Stderr, "--email required")
	}
	sigchain.RemoveAdmin(email)
	return
}

func pinHostKeyCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()
	exitIfNotAdmin()

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
		sigchain.PinKnownHostKeys(c.String("host"), c.Bool("update-from-server"))
		return
	}

	pk, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		PrintFatal(os.Stderr, "error decoding public-key, make sure it is base64 encoded without the key type prefix (i.e. no 'ssh-rsa' or 'ssh-ed25519') "+err.Error())
	}
	sigchain.PinHostKey(c.String("host"), pk)
	return
}

func unpinHostKeyCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()
	exitIfNotAdmin()

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
	sigchain.UnpinHostKey(c.String("host"), pk)
	return
}

func listPinnedKeysCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	host := c.String("host")
	if host == "" {
		sigchain.GetAllPinnedHostKeys()
		return
	}
	sigchain.GetPinnedHostKeys(host, c.Bool("search"))
	return
}

func enableLoggingCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()
	exitIfNotAdmin()

	sigchain.EnableLogging()
	return
}

func logsCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()
	exitIfNotAdmin()

	sigchain.UpdateTeamLogs()
	return
}

func teamBillingCommand(c *cli.Context) (err error) {
	exitIfNotOnTeam()

	sigchain.OpenBilling()
	return
}

type DashboardParams struct {
	Port  uint16 `json:"port"`
	Token []byte `json:"token"`
}

func teamDashboardCommand(c *cli.Context) (err error) {
	if c.Bool("clean") {
		teamDbFile, err := kr.KrDirFile("team.db")
		if err != nil {
			PrintFatal(os.Stderr, "Failed to find ~/.kr "+err.Error())
		}
		_ = os.Remove(teamDbFile)

		err = restartCommandOptions(c, false)
		if err != nil {
			PrintFatal(os.Stderr, "Failed to restart krd: "+err.Error())
		}
		<-time.After(time.Second)
	}

	exitIfNotOnTeam()
	exitIfNotAdmin()

	err = krdclient.RequestDashboard()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}

	f, err := kr.KrDirFile("dashboard_params")
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}

	paramsJson, err := ioutil.ReadFile(f)
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	var params DashboardParams
	err = json.Unmarshal(paramsJson, &params)
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}

	link := fmt.Sprintf("http://localhost:%d/#%s", params.Port, base64.URLEncoding.EncodeToString(params.Token))
	PrintErr(os.Stderr, kr.Cyan(fmt.Sprintf("Krypton ▶ Dashboard running at %s", link)))
	<-time.After(time.Millisecond * 750)
	PrintErr(os.Stderr, kr.Cyan("Krypton ▶ Opening web browser..."))
	<-time.After(time.Millisecond * 750)
	openBrowser(link)
	return
}
