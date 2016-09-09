package main

/*
* CLI to control krssh-agent
 */

import (
	"bitbucket.org/kryptco/krssh"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
)

func connectToAgent() (conn net.Conn, err error) {
	agentSockName := os.Getenv(krssh.KRSSH_CTL_SOCK_ENV)
	conn, err = net.Dial("unix", agentSockName)
	return
}

func pairCommand(c *cli.Context) (err error) {
	agentConn, err := connectToAgent()
	if err != nil {
		log.Fatal(err)
	}

	pairingSecret, err := krssh.GeneratePairingSecret()
	if err != nil {
		log.Fatal(err)
	}

	pairingSecretJson, err := json.Marshal(pairingSecret)
	if err != nil {
		log.Fatal(err)
	}

	pairRequest, err := http.NewRequest("PUT", "/pair", bytes.NewReader(pairingSecretJson))
	if err != nil {
		log.Fatal(err)
	}

	err = pairRequest.Write(agentConn)
	if err != nil {
		log.Fatal(err)
	}

	qr, err := QREncode(pairingSecretJson)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Scan this QR Code with the krSSH Mobile App to connect it with this workstation.")
	fmt.Println()
	fmt.Println(qr.Terminal)

	bufReader := bufio.NewReader(agentConn)
	response, err := http.ReadResponse(bufReader, pairRequest)
	if err != nil {
		log.Fatal(err)
	}
	if response.StatusCode != 200 {
		log.Fatalf("Pairing failed with error code %d", response.StatusCode)
	}
	defer response.Body.Close()
	var me krssh.Profile
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(responseBody, &me)

	clearCommand := exec.Command("clear")
	clearCommand.Stdout = os.Stdout
	clearCommand.Run()

	fmt.Println("Paired successfully with identity", me.DisplayString())
	return
}

func meCommand(c *cli.Context) (err error) {
	agentConn, err := connectToAgent()
	if err != nil {
		log.Fatal(err)
	}

	request, err := krssh.NewRequest()
	if err != nil {
		log.Fatal(err)
	}
	request.MeRequest = &krssh.MeRequest{}
	requestJson, err := json.Marshal(request)
	if err != nil {
		log.Fatal(err)
	}
	httpRequest, err := http.NewRequest("PUT", "/enclave", bytes.NewReader(requestJson))
	if err != nil {
		log.Fatal(err)
	}
	err = httpRequest.Write(agentConn)
	if err != nil {
		log.Fatal(err)
	}

	bufReader := bufio.NewReader(agentConn)
	response, err := http.ReadResponse(bufReader, httpRequest)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		log.Printf("Error retrieving paired identity: %d", response.StatusCode)
	}

	var me krssh.Profile
	err = json.NewDecoder(response.Body).Decode(&me)
	if err != nil {
		log.Fatal(err)
	}
	wireString, err := me.SSHWireString()
	if err != nil {
		log.Fatal(err)
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
