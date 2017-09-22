package main

import (
	"encoding/base64"
	"errors"
	"os"
	"strconv"

	"github.com/kryptco/kr"
	"github.com/kryptco/kr/krdclient"
	"github.com/urfave/cli"
)

func createTeamCommand(c *cli.Context) (err error) {
	_, err = krdclient.RequestMe()
	if err != nil {
		PrintFatal(os.Stderr, kr.Red("Kryptonite ▶ "+err.Error()))
		return
	}
	request, err := kr.NewRequest()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
		return
	}
	name := c.String("name")
	if name == "" {
		PrintFatal(os.Stderr, "--name flag required")
	}
	request.CreateTeamRequest = &kr.CreateTeamRequest{
		Name: name,
	}
	os.Stderr.WriteString(kr.Yellow("Kryptonite ▶ Your team is almost ready, use the Kryptonite app on your phone to complete the setup.\r\n"))
	response, err := krdclient.Request(request)
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	createTeamResponse := response.CreateTeamResponse
	if createTeamResponse == nil {
		PrintFatal(os.Stderr, "Invalid response from Kryptonite app.")
	}
	if createTeamResponse.Error != nil {
		PrintFatal(os.Stderr, "Error creating team: "+*createTeamResponse.Error)
	}
	keyAndTeamCheckpoint := createTeamResponse.KeyAndTeamCheckpoint
	if keyAndTeamCheckpoint == nil {
		PrintFatal(os.Stderr, "No team admin private key returned from Kryptonite app.")
	}
	err = kr.SaveAdminSeedAndTeamCheckpoint(*keyAndTeamCheckpoint)
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	os.Stderr.WriteString(kr.Green("Success! Team " + name + " is ready to go ✔\r\n"))
	return
}

func ensureAdminKeyPresent() (err error) {
	if kr.AdminSeedAndTeamCheckpointExists() {
		return
	}

	adminRequest, err := kr.NewRequest()
	if err != nil {
		return
	}
	adminRequest.AdminKeyRequest = &kr.AdminKeyRequest{}
	response, err := krdclient.Request(adminRequest)
	if err != nil {
		return
	}

	adminResponse := response.AdminKeyResponse

	if adminResponse == nil {
		err = errors.New("no AdminKeyResponse returned from phone")
		return
	}

	if adminResponse.Error != nil {
		err = errors.New(*adminResponse.Error)
		return
	}

	if adminResponse.KeyAndTeamCheckpoint == nil {
		err = errors.New("no admin key returned from phone")
		return
	}
	keyAndTeamCheckpoint := *adminResponse.KeyAndTeamCheckpoint
	kr.SaveAdminSeedAndTeamCheckpoint(keyAndTeamCheckpoint)

	return
}

func createInviteCommand(c *cli.Context) (err error) {
	err = ensureAdminKeyPresent()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	kr.CreateInvite()
	return
}

func setPolicyCommand(c *cli.Context) (err error) {
	err = ensureAdminKeyPresent()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
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
	err = ensureAdminKeyPresent()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	var query *string
	if c.String("query") != "" {
		queryStr := c.String("query")
		query = &queryStr
	}
	kr.GetMembers(query, c.Bool("ssh"), c.Bool("pgp"))
	return
}

func addAdminCommand(c *cli.Context) (err error) {
	err = ensureAdminKeyPresent()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	email := c.String("email")
	if email == "" {
		PrintFatal(os.Stderr, "--email required")
	}
	kr.AddAdmin(email)
	return
}

func removeAdminCommand(c *cli.Context) (err error) {
	err = ensureAdminKeyPresent()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	email := c.String("email")
	if email == "" {
		PrintFatal(os.Stderr, "--email required")
	}
	kr.RemoveAdmin(email)
	return
}

func getAdminsCommand(c *cli.Context) (err error) {
	err = ensureAdminKeyPresent()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	kr.GetAdmins()
	return
}

func pinHostKeyCommand(c *cli.Context) (err error) {
	err = ensureAdminKeyPresent()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	if c.String("public-key") == "" {
		kr.PinKnownHostKeys(c.String("host"), c.Bool("update-from-server"))
		return
	}
	pk, err := base64.StdEncoding.DecodeString(c.String("public-key"))
	if err != nil {
		PrintFatal(os.Stderr, "error decoding public-key, make sure it is base64 encoded without the key type prefix (i.e. no 'ssh-rsa' or 'ssh-ed25519') "+err.Error())
	}
	kr.PinHostKey(c.String("host"), pk)
	return
}

func unpinHostKeyCommand(c *cli.Context) (err error) {
	err = ensureAdminKeyPresent()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	pk, err := base64.StdEncoding.DecodeString(c.String("public-key"))
	if err != nil {
		PrintFatal(os.Stderr, "error decoding public-key, make sure it is base64 encoded without the key type prefix (i.e. no 'ssh-rsa' or 'ssh-ed25519') "+err.Error())
	}
	kr.UnpinHostKey(c.String("host"), pk)
	return
}

func listPinnedKeysCommand(c *cli.Context) (err error) {
	err = ensureAdminKeyPresent()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	host := c.String("host")
	if host == "" {
		kr.GetAllPinnedHostKeys()
		return
	}
	kr.GetPinnedHostKeys(host, c.Bool("search"))
	return
}
