package main

/*
* CLI to control krd
 */

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/agrinman/kr"
	"github.com/urfave/cli"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
)

func PrintFatal(msg string, args ...interface{}) {
	os.Stderr.WriteString(fmt.Sprintf(msg, args...) + "\n")
	os.Exit(1)
}

func connectToAgent() (conn net.Conn, err error) {
	conn, err = kr.DaemonDial()
	return
}

func pairCommand(c *cli.Context) (err error) {
	putConn, err := connectToAgent()
	if err != nil {
		PrintFatal(err.Error())
	}
	defer putConn.Close()

	putPair, err := http.NewRequest("PUT", "/pair", nil)
	if err != nil {
		PrintFatal(err.Error())
	}

	err = putPair.Write(putConn)
	if err != nil {
		PrintFatal(err.Error())
	}

	putReader := bufio.NewReader(putConn)
	putPairResponse, err := http.ReadResponse(putReader, putPair)
	if err != nil {
		PrintFatal(err.Error())
	}
	responseBytes, err := ioutil.ReadAll(putPairResponse.Body)
	if err != nil {
		PrintFatal(err.Error())
	}
	if putPairResponse.StatusCode != http.StatusOK {
		PrintFatal(string(responseBytes))
	}

	qr, err := QREncode(responseBytes)
	if err != nil {
		PrintFatal(err.Error())
	}

	fmt.Println()
	fmt.Println(qr.Terminal)
	fmt.Println("Scan this QR Code with the kryptonite mobile app to connect it with this workstation. Try lowering your terminal font size if the QR code does not fit on the screen.")
	fmt.Println()

	getConn, err := connectToAgent()
	if err != nil {
		PrintFatal(err.Error())
	}
	defer getConn.Close()

	getPair, err := http.NewRequest("GET", "/pair", nil)
	if err != nil {
		PrintFatal(err.Error())
	}
	err = getPair.Write(getConn)
	if err != nil {
		PrintFatal(err.Error())
	}

	//	Check/wait for pairing
	getReader := bufio.NewReader(getConn)
	getResponse, err := http.ReadResponse(getReader, getPair)

	clearCommand := exec.Command("clear")
	clearCommand.Stdout = os.Stdout
	clearCommand.Run()

	if err != nil {
		PrintFatal(err.Error())
	}
	switch getResponse.StatusCode {
	case 404, 500:
		PrintFatal("Pairing failed %d, ensure your phone and workstation are connected to the internet and try again.", getResponse.StatusCode)
	default:
	}
	defer getResponse.Body.Close()
	var me kr.Profile
	responseBody, err := ioutil.ReadAll(getResponse.Body)
	if err != nil {
		PrintFatal(err.Error())
	}
	err = json.Unmarshal(responseBody, &me)

	fmt.Println("Paired successfully with identity")
	authorizedKey := me.AuthorizedKeyString()
	fmt.Println(authorizedKey)
	return
}

func meCommand(c *cli.Context) (err error) {
	agentConn, err := connectToAgent()
	if err != nil {
		PrintFatal(err.Error())
	}

	request, err := kr.NewRequest()
	if err != nil {
		PrintFatal(err.Error())
	}
	request.MeRequest = &kr.MeRequest{}
	httpRequest, err := request.HTTPRequest()
	if err != nil {
		PrintFatal(err.Error())
	}
	err = httpRequest.Write(agentConn)
	if err != nil {
		PrintFatal(err.Error())
	}

	bufReader := bufio.NewReader(agentConn)
	response, err := http.ReadResponse(bufReader, httpRequest)
	if err != nil {
		PrintFatal(err.Error())
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case 404:
		PrintFatal("Workstation not yet paired. Please run \"kr pair\" and scan the QRCode with the Kryptonite mobile app.")
	default:
	}

	var meResponse kr.Response
	err = json.NewDecoder(response.Body).Decode(&meResponse)
	if err != nil {
		PrintFatal(err.Error())
	}
	if meResponse.MeResponse == nil {
		PrintFatal("Response missing profile")
	}
	me := meResponse.MeResponse.Me
	authorizedKey := me.AuthorizedKeyString()
	if err != nil {
		PrintFatal(err.Error())
	}
	fmt.Println(authorizedKey)
	return
}

func listCommand(c *cli.Context) (err error) {
	agentConn, err := connectToAgent()
	if err != nil {
		PrintFatal(err.Error())
	}

	request, err := kr.NewRequest()
	if err != nil {
		PrintFatal(err.Error())
	}
	request.ListRequest = &kr.ListRequest{}
	httpRequest, err := request.HTTPRequest()
	if err != nil {
		PrintFatal(err.Error())
	}
	err = httpRequest.Write(agentConn)
	if err != nil {
		PrintFatal(err.Error())
	}

	bufReader := bufio.NewReader(agentConn)
	response, err := http.ReadResponse(bufReader, httpRequest)
	if err != nil {
		PrintFatal(err.Error())
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case 404:
		PrintFatal("Workstation not yet paired. Please run \"kr pair\" and scan the QRCode with the Kryptonite mobile app.")
	default:
	}

	var krResponse kr.Response
	err = json.NewDecoder(response.Body).Decode(&krResponse)
	if err != nil {
		PrintFatal(err.Error())
	}
	if krResponse.ListResponse == nil {
		PrintFatal("Response missing profiles")
	}
	profiles := krResponse.ListResponse.Profiles
	for _, profile := range profiles {
		fmt.Println(profile.AuthorizedKeyString())
	}
	return
}

func main() {
	app := cli.NewApp()
	app.Name = "kr"
	app.Usage = "communicate with Kryptonite and krd - the Kryptonite daemon"
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{}
	app.Commands = []cli.Command{
		cli.Command{
			Name:    "pair",
			Aliases: []string{"p"},
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "no-aws"},
			},
			Action: pairCommand,
		},
		cli.Command{
			Name:   "me",
			Action: meCommand,
		},
		cli.Command{
			Name:    "list",
			Aliases: []string{"ls"},
			Action:  listCommand,
		},
	}
	app.Run(os.Args)
}
