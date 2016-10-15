package main

/*
* CLI to control krd
 */

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/agrinman/kr"
	"github.com/agrinman/kr/krdclient"
	"github.com/atotto/clipboard"
	"github.com/fatih/color"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh"
)

func PrintFatal(msg string, args ...interface{}) {
	PrintErr(msg, args...)
	os.Exit(1)
}

func PrintErr(msg string, args ...interface{}) {
	os.Stderr.WriteString(fmt.Sprintf(msg, args...) + "\n")
}

func pairCommand(c *cli.Context) (err error) {
	_, err = krdclient.RequestMe()
	if err == nil {
		PrintErr("Already paired, unpair current session? [y/N] ")
		var c string
		fmt.Scan(&c)
		if len(c) == 0 || c[0] != 'y' {
			PrintFatal("Aborting.")
		}
	}
	putConn, err := kr.DaemonDial()
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
	fmt.Println("Scan this QR Code with the Kryptonite mobile app to connect it with this workstation. Try lowering your terminal font size if the QR code does not fit on the screen.")
	fmt.Println()

	getConn, err := kr.DaemonDial()
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
	case http.StatusNotFound, http.StatusInternalServerError:
		PrintFatal("Pairing failed, ensure your phone and workstation are connected to the internet and try again.")
	case http.StatusOK:
	default:
		PrintFatal("Pairing failed with error %d", getResponse.StatusCode)
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

func unpairCommand(c *cli.Context) (err error) {
	conn, err := kr.DaemonDial()
	if err != nil {
		PrintFatal(err.Error())
	}
	defer conn.Close()

	deletePair, err := http.NewRequest("DELETE", "/pair", nil)
	if err != nil {
		PrintFatal(err.Error())
	}

	err = deletePair.Write(conn)
	if err != nil {
		PrintFatal(err.Error())
	}

	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, deletePair)
	if err != nil {
		PrintFatal(err.Error())
	}
	switch response.StatusCode {
	case http.StatusNotFound, http.StatusInternalServerError:
		PrintFatal("Unpair failed, ensure the Kryptonite daemon is running with \"kr reset\".")
	case http.StatusOK:
	default:
		PrintFatal("Unpair failed with error %d", response.StatusCode)
	}
	fmt.Println("Unpaired Kryptonite.")
	return
}

func meCommand(c *cli.Context) (err error) {
	me, err := krdclient.RequestMe()
	if err != nil {
		PrintFatal(err.Error())
	}
	authorizedKey := me.AuthorizedKeyString()
	if err != nil {
		PrintFatal(err.Error())
	}
	fmt.Println(authorizedKey)
	PrintErr("\r\nCopy this key to your clipboard using \"kr copy\" or add it to a service like Github using \"kr github\". Type \"kr\" to see all available commands.")
	return
}

func copyCommand(c *cli.Context) (err error) {
	copyKey()
	PrintErr("Public key copied to clipboard.")
	return
}

func copyKey() (err error) {
	me, err := krdclient.RequestMe()
	if err != nil {
		PrintFatal(err.Error())
	}
	authorizedKey := me.AuthorizedKeyString()
	err = clipboard.WriteAll(authorizedKey)
	if err != nil {
		PrintFatal(err.Error())
	}
	return
}

func addCommand(c *cli.Context) (err error) {
	if len(c.Args()) < 2 {
		PrintFatal("kr add <first email> <second email>... <user@server or SSH alias>")
		return
	}
	server := c.Args()[len(c.Args())-1]

	profiles := []kr.Profile{}
	me, err := krdclient.RequestMe()
	if err == nil {
		profiles = append(profiles, me)
	} else {
		PrintErr("error retrieving your key: ", err.Error())
	}
	peers, err := krdclient.RequestList()
	if err == nil {
		profiles = append(profiles, peers...)
	} else {
		PrintErr("error retrieving peer keys: ", err.Error())
	}

	filter := map[string]bool{}
	for _, email := range c.Args()[:len(c.Args())] {
		if email == "me" {
			filter[me.Email] = true
		}
		filter[email] = true
	}

	authorizedKeys := [][]byte{}
	for _, profile := range profiles {
		if _, ok := filter[profile.Email]; ok {
			authorizedKeys = append(authorizedKeys, []byte(profile.AuthorizedKeyString()))
		}
	}

	if len(authorizedKeys) == 0 {
		PrintFatal("No keys match specified emails")
	}

	PrintErr("Adding %d keys to %s", len(authorizedKeys), server)

	authorizedKeysReader := bytes.NewReader(append(bytes.Join(authorizedKeys, []byte("\n")), []byte("\n")...))
	sshCommand := exec.Command("ssh", server, "cat - >> ~/.ssh/authorized_keys")
	sshCommand.Stdin = authorizedKeysReader
	err = sshCommand.Run()
	if err != nil {
		PrintFatal(err.Error())
	}
	return
}

func listCommand(c *cli.Context) (err error) {
	if len(c.Args()) == 0 {
		PrintFatal("usage: kr list <user@server or SSH alias>")
	}
	server := c.Args()[0]

	peers, err := krdclient.RequestList()
	if err != nil {
		PrintFatal(err.Error())
	}
	me, err := krdclient.RequestMe()
	if err != nil {
		PrintFatal(err.Error())
	}

	profilesByWireB64 := map[string]kr.Profile{}
	for _, peer := range append(peers, me) {
		profilesByWireB64[base64.StdEncoding.EncodeToString(peer.SSHWirePublicKey)] = peer
	}

	authorizedKeysBuffer := bytes.Buffer{}
	sshCommand := exec.Command("ssh", server, "cat ~/.ssh/authorized_keys")
	sshCommand.Stdout = &authorizedKeysBuffer
	sshCommand.Stderr = os.Stderr
	err = sshCommand.Run()

	authorizedKeysBytes := authorizedKeysBuffer.Bytes()
	var key ssh.PublicKey
	var comment string
	nPeers := 0
	nUnknown := 0
	for {
		key, comment, _, authorizedKeysBytes, err = ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err == nil {
			wireB64 := base64.StdEncoding.EncodeToString(key.Marshal())
			if peer, ok := profilesByWireB64[wireB64]; ok {
				color.Green(peer.Email)
				nPeers++
			} else {
				if comment == "" {
					color.Yellow("Unknown Key")
				} else {
					color.Yellow("Unknown Key (" + comment + ")")
				}
				nUnknown++
			}
			fmt.Printf("%s %s\n\n", key.Type(), wireB64)
		} else if err != nil || len(authorizedKeysBytes) == 0 {
			break
		}
	}
	fmt.Printf("Found %s and %s\n", color.GreenString("%d Peer Keys", nPeers), color.YellowString("%d Unknown Keys", nUnknown))
	return
}

func peersCommand(c *cli.Context) (err error) {
	profiles, err := krdclient.RequestList()
	if err != nil {
		PrintFatal(err.Error())
	}
	if len(profiles) == 0 {
		PrintErr("You don't have any peers yet. Use the kryptonite app to request peers' public keys.")
	}
	filterEmails := map[string]bool{}
	for _, arg := range c.Args() {
		filterEmails[arg] = true
	}

	for _, profile := range profiles {
		if _, ok := filterEmails[profile.Email]; len(filterEmails) == 0 || ok {
			color.Green(profile.Email)
			fmt.Println(profile.AuthorizedKeyString())
			fmt.Println()
		}
	}
	return
}

func githubCommand(c *cli.Context) (err error) {
	copyKey()
	PrintErr("Public key copied to clipboard.")
	<-time.After(500 * time.Millisecond)
	PrintErr("Opening GitHub...")
	<-time.After(500 * time.Millisecond)
	openBrowser("https://github.com/settings/keys")
	return
}

func digitaloceanCommand(c *cli.Context) (err error) {
	copyKey()
	PrintErr("Public key copied to clipboard.")
	<-time.After(500 * time.Millisecond)
	PrintErr("Opening DigitalOcean...")
	<-time.After(500 * time.Millisecond)
	openBrowser("https://cloud.digitalocean.com/settings/security")
	return
}

func herokuCommand(c *cli.Context) (err error) {
	copyKey()
	PrintErr("Public key copied to clipboard.")
	<-time.After(500 * time.Millisecond)
	PrintErr("Opening Heroku...")
	<-time.After(500 * time.Millisecond)
	openBrowser("https://dashboard.heroku.com/account")
	return
}

func gcloudCommand(c *cli.Context) (err error) {
	copyKey()
	PrintErr("Public key copied to clipboard.")
	<-time.After(500 * time.Millisecond)
	PrintErr("Opening Google Cloud...")
	<-time.After(500 * time.Millisecond)
	openBrowser("https://console.cloud.google.com/compute/metadata/sshKeys")
	return
}

func awsCommand(c *cli.Context) (err error) {
	copyKey()
	PrintErr("Public key copied to clipboard.")
	<-time.After(500 * time.Millisecond)
	PrintErr("Opening AWS Console. Click 'Import Key Pair' to add your key.")
	<-time.After(1500 * time.Millisecond)
	openBrowser("https://console.aws.amazon.com/ec2/v2/home?#KeyPairs:sort=keyName")
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
			Name:   "pair",
			Usage:  "Initiate pairing of this workstation with a phone running Kryptonite.",
			Action: pairCommand,
		},
		cli.Command{
			Name:   "me",
			Usage:  "Print your SSH public key.",
			Action: meCommand,
		},
		cli.Command{
			Name:   "copy",
			Usage:  "Copy your SSH public key to the clipboard.",
			Action: copyCommand,
		},
		cli.Command{
			Name:   "github",
			Usage:  "Upload your public key to GitHub. Copies your public key to the clipboard and opens GitHub settings.",
			Action: githubCommand,
		},
		cli.Command{
			Name:   "digital-ocean",
			Usage:  "Upload your public key to Digital Ocean. Copies your public key to the clipboard and opens Digital Ocean settings.",
			Action: digitaloceanCommand,
		},
		cli.Command{
			Name:   "heroku",
			Usage:  "Upload your public key to Heroku. Copies your public key to the clipboard and opens Heroku settings.",
			Action: herokuCommand,
		},
		cli.Command{
			Name:   "aws",
			Usage:  "Upload your public key to Amazon Web Services. Copies your public key to the clipboard and opens the AWS Console.",
			Action: awsCommand,
		},
		cli.Command{
			Name:   "gcloud",
			Usage:  "Upload your public key to Google Cloud. Copies your public key to the clipboard and opens the Google Cloud Console.",
			Action: gcloudCommand,
		},
		cli.Command{
			Name:   "peers",
			Usage:  "peers <optional email> -- list your peers' public keys, filtering by email if present.",
			Action: peersCommand,
		},
		cli.Command{
			Name:   "list",
			Usage:  "kr list <user@server or SSH alias> -- List public keys authorized on the specified server.",
			Action: listCommand,
		},
		cli.Command{
			Name:   "add",
			Usage:  "kr add <first email> <second email>... <user@server or SSH alias> -- add the public key of the specified users to the server.",
			Action: addCommand,
		},
		cli.Command{
			Name:   "restart",
			Usage:  "Restart the Kryptonite daemon.",
			Action: restartCommand,
		},
		cli.Command{
			Name:   "unpair",
			Usage:  "Unpair this workstation from a phone running Kryptonite.",
			Action: unpairCommand,
		},
	}
	app.Run(os.Args)
}
