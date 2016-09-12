package main

/*
* CLI to control krssh-agent
 */

import (
	"bitbucket.org/kryptco/krssh"
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
)

func PrintFatal(msg string) {
	os.Stderr.WriteString(msg + "\n")
	os.Exit(1)
}

func connectToAgent() (conn net.Conn, err error) {
	agentSockName := os.Getenv(krssh.KRSSH_CTL_SOCK_ENV)
	conn, err = net.Dial("unix", agentSockName)
	return
}

func pairCommand(c *cli.Context) (err error) {
	agentConn, err := connectToAgent()
	if err != nil {
		PrintFatal(err.Error())
	}

	pairingSecret, err := krssh.GeneratePairingSecret()
	if err != nil {
		PrintFatal(err.Error())
	}

	pairRequest, err := pairingSecret.HTTPRequest()
	if err != nil {
		PrintFatal(err.Error())
	}

	err = pairRequest.Write(agentConn)
	if err != nil {
		PrintFatal(err.Error())
	}

	pairingSecretJson, err := json.Marshal(pairingSecret)
	if err != nil {
		PrintFatal(err.Error())
	}

	qr, err := QREncode(pairingSecretJson)
	if err != nil {
		PrintFatal(err.Error())
	}

	fmt.Println("Scan this QR Code with the krSSH Mobile App to connect it with this workstation.")
	fmt.Println()
	fmt.Println(qr.Terminal)

	bufReader := bufio.NewReader(agentConn)
	response, err := http.ReadResponse(bufReader, pairRequest)

	clearCommand := exec.Command("clear")
	clearCommand.Stdout = os.Stdout
	clearCommand.Run()

	if err != nil {
		PrintFatal(err.Error())
	}
	switch response.StatusCode {
	case 404:
		PrintFatal("Pairing failed, ensure your phone and workstation are connected to the internet and try again.")
	case 500:
		PrintFatal("Pairing failed, ensure your phone and workstation are connected to the internet and try again.")
	default:
	}
	defer response.Body.Close()
	var me krssh.Profile
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		PrintFatal(err.Error())
	}
	err = json.Unmarshal(responseBody, &me)

	fmt.Println("Paired successfully with identity", me.DisplayString())
	return
}

func meCommand(c *cli.Context) (err error) {
	agentConn, err := connectToAgent()
	if err != nil {
		PrintFatal(err.Error())
	}

	request, err := krssh.NewRequest()
	if err != nil {
		PrintFatal(err.Error())
	}
	request.MeRequest = &krssh.MeRequest{}
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
		PrintFatal("Workstation not yet paired. Please run \"kr pair\" and scan the QRCode with the krSSH mobile app.")
	default:
	}

	var me krssh.Profile
	err = json.NewDecoder(response.Body).Decode(&me)
	if err != nil {
		PrintFatal(err.Error())
	}
	wireString, err := me.SSHWireString()
	if err != nil {
		PrintFatal(err.Error())
	}
	fmt.Println(wireString)
	return
}

func main() {
	app := cli.NewApp()
	app.Name = "kr"
	app.Usage = "communicate with krssh-agent and krssh-iOS"
	app.Flags = []cli.Flag{}
	app.Commands = []cli.Command{
		cli.Command{
			Name:    "pair",
			Aliases: []string{"p"},
			Action:  pairCommand,
		},
		cli.Command{
			Name:   "me",
			Action: meCommand,
		},
		cli.Command{
			Name:    "list",
			Aliases: []string{"ls"},
			Action:  pairCommand,
		},
	}
	app.Run(os.Args)
}
