package main

/*
* CLI to control krd
 */

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/kryptco/kr"
	"github.com/kryptco/kr/krdclient"
	"github.com/urfave/cli"
)

func PrintFatal(stderr io.ReadWriter, msg string, args ...interface{}) {
	if len(args) == 0 {
		PrintErr(stderr, msg)
	} else {
		PrintErr(stderr, msg, args...)
	}
	os.Exit(1)
}

func PrintErr(stderr io.ReadWriter, msg string, args ...interface{}) {
	stderr.Write([]byte(fmt.Sprintf(msg, args...) + "\n"))
}

func confirmOrFatal(stderr io.ReadWriter, message string) {
	PrintErr(stderr, message+" [y/N] ")
	var c string
	fmt.Scan(&c)
	if len(c) == 0 || c[0] != 'y' {
		PrintFatal(stderr, "Aborting.")
	}
}

func pairCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "pair", nil, nil)
	}()
	return pairOver(kr.DaemonSocketOrFatal(), c.Bool("force"), os.Stdout, os.Stderr)
}

func pairOver(unixFile string, forceUnpair bool, stdout io.ReadWriter, stderr io.ReadWriter) (err error) {
	//	Listen for incompatible enclave notifications
	go func() {
		r, err := kr.OpenNotificationReader("")
		if err != nil {
			os.Stderr.WriteString("error connection to notificationr reader: " + err.Error())
			return
		}
		printedMessages := map[string]bool{}
		for {
			notification, err := r.Read()
			if err != nil {
				<-time.After(50 * time.Millisecond)
				continue
			}
			str := string(notification)
			if strings.HasPrefix(str, "[") {
				continue
			}
			if _, ok := printedMessages[str]; ok {
				continue
			}
			os.Stderr.WriteString(str)
			printedMessages[str] = true
		}
	}()
	if !forceUnpair {
		meConn, err := kr.DaemonDialWithTimeout(unixFile)
		if err != nil {
			PrintFatal(stderr, "Could not connect to Kryptonite daemon. Make sure it is running by typing \"kr restart\".")
		}
		_, err = krdclient.RequestMeOver(meConn)
		if err == nil {
			confirmOrFatal(stderr, "Already paired, unpair current session?")
		}
	}
	putConn, err := kr.DaemonDialWithTimeout(unixFile)
	if err != nil {
		PrintFatal(stderr, "Could not connect to Kryptonite daemon. Make sure it is running by typing \"kr restart\".")
	}
	defer putConn.Close()

	putPair, err := http.NewRequest("PUT", "/pair", nil)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}

	err = putPair.Write(putConn)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}

	putReader := bufio.NewReader(putConn)
	putPairResponse, err := http.ReadResponse(putReader, putPair)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}
	responseBytes, err := ioutil.ReadAll(putPairResponse.Body)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}
	if putPairResponse.StatusCode != http.StatusOK {
		PrintFatal(stderr, "Pairing failed, ensure your phone and workstation are connected to the internet and try again.")
	}

	qr, err := QREncode(responseBytes)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}

	stdout.Write([]byte("\r\n"))
	stdout.Write([]byte(qr.Terminal))
	stdout.Write([]byte("\r\n"))
	stdout.Write([]byte("Scan this QR Code with the Kryptonite mobile app to connect it with this workstation. Maximize the window and/or lower your font size if the QR code does not fit."))
	stdout.Write([]byte("\r\n"))

	getConn, err := kr.DaemonDialWithTimeout(unixFile)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}
	defer getConn.Close()

	getPair, err := http.NewRequest("GET", "/pair", nil)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}
	err = getPair.Write(getConn)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}

	//	Check/wait for pairing
	getReader := bufio.NewReader(getConn)
	getResponse, err := http.ReadResponse(getReader, getPair)

	clearCommand := exec.Command("clear")
	clearCommand.Stdout = stdout
	clearCommand.Run()

	if err != nil {
		PrintFatal(stderr, err.Error())
	}
	switch getResponse.StatusCode {
	case http.StatusNotFound, http.StatusInternalServerError:
		PrintFatal(stderr, "Pairing failed, ensure your phone and workstation are connected to the internet and try again.")
	case http.StatusOK:
	default:
		PrintFatal(stderr, "Pairing failed with error %d", getResponse.StatusCode)
	}
	defer getResponse.Body.Close()
	var me kr.Profile
	responseBody, err := ioutil.ReadAll(getResponse.Body)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}
	err = json.Unmarshal(responseBody, &me)

	stdout.Write([]byte("Paired successfully with identity\r\n"))
	authorizedKey, err := me.AuthorizedKeyString()
	if err != nil {
		PrintFatal(stderr, err.Error())
	}
	stdout.Write([]byte(authorizedKey))
	stdout.Write([]byte("\r\n"))
	return
}

func unpairCommand(c *cli.Context) (err error) {
	kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "unpair", nil, nil)
	return unpairOver(kr.DaemonSocketOrFatal(), os.Stdout, os.Stderr)
}

func unpairOver(unixFile string, stdout io.ReadWriter, stderr io.ReadWriter) (err error) {
	conn, err := kr.DaemonDialWithTimeout(unixFile)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}
	defer conn.Close()

	deletePair, err := http.NewRequest("DELETE", "/pair", nil)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}

	err = deletePair.Write(conn)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}

	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, deletePair)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}
	switch response.StatusCode {
	case http.StatusNotFound, http.StatusInternalServerError:
		PrintFatal(stderr, "Unpair failed, ensure the Kryptonite daemon is running with \"kr restart\".")
	case http.StatusOK:
	default:
		PrintFatal(stderr, "Unpair failed with error %d", response.StatusCode)
	}
	stdout.Write([]byte("Unpaired Kryptonite.\r\n"))
	return
}

func meCommand(c *cli.Context) (err error) {
	me, err := krdclient.RequestMe()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	authorizedKey, err := me.AuthorizedKeyString()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	fmt.Println(authorizedKey)
	PrintErr(os.Stderr, "\r\nCopy this key to your clipboard using \"kr copy\" or add it to a service like Github using \"kr github\". Type \"kr\" to see all available commands.")
	kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "me", nil, nil)
	return
}

func copyCommand(c *cli.Context) (err error) {
	copyKey()
	PrintErr(os.Stderr, "Public key copied to clipboard.")
	kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "copy", nil, nil)
	return
}

func copyKey() (me kr.Profile, err error) {
	me, err = krdclient.RequestMe()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	authorizedKey, err := me.AuthorizedKeyString()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	err = clipboard.WriteAll(authorizedKey)
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	return
}

func addCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "add", nil, nil)
	}()
	copyKey()
	if len(c.Args()) < 1 {
		PrintFatal(os.Stderr, "kr add <user@server or SSH alias>")
		return
	}
	server := c.Args()[0]

	portFlag := c.String("port")

	me, err := krdclient.RequestMe()
	if err != nil {
		PrintFatal(os.Stderr, "error retrieving your public key: ", err.Error())
	}

	authorizedKeyString, err := me.AuthorizedKeyString()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	authorizedKey := append([]byte(authorizedKeyString), []byte("\n")...)

	PrintErr(os.Stderr, "Adding your SSH public key to %s", server)

	authorizedKeyReader := bytes.NewReader(authorizedKey)
	args := []string{server}
	if portFlag != "" {
		args = append(args, "-p "+portFlag)
	}
	args = append(args, "sh -c 'read keys; mkdir -m 700 -p ~/.ssh && echo $keys >> ~/.ssh/authorized_keys; chmod 600 ~/.ssh/authorized_keys'")
	sshCommand := exec.Command("ssh", args...)
	sshCommand.Stdin = authorizedKeyReader
	sshCommand.Stdout = os.Stdout
	sshCommand.Stderr = os.Stderr
	sshCommand.Run()
	return
}

func githubCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "github", nil, nil)
	}()
	copyKey()
	PrintErr(os.Stderr, "Public key copied to clipboard.")
	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to GitHub. Then click \"New SSH Key\" and paste your public key.")
	os.Stdin.Read([]byte{0})
	openBrowser("https://github.com/settings/keys")
	return
}

func gitlabCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "gitlab", nil, nil)
	}()
	copyKey()
	PrintErr(os.Stderr, "Public key copied to clipboard.")
	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to GitLab. Then paste your public key and click \"Add key.\"")
	os.Stdin.Read([]byte{0})
	openBrowser("https://gitlab.com/profile/keys")
	return
}

func bitbucketCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "bitbucket", nil, nil)
	}()
	copyKey()
	PrintErr(os.Stderr, "Public key copied to clipboard.")
	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to BitBucket. Then click \"Add key\" and paste your public key.")
	os.Stdin.Read([]byte{0})
	openBrowser("https://bitbucket.org/account/ssh-keys/")
	return
}

func digitaloceanCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "digitalocean", nil, nil)
	}()
	copyKey()
	PrintErr(os.Stderr, "Public key copied to clipboard.")
	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to DigitalOcean. Then click \"Add SSH Key\" and paste your public key.")
	os.Stdin.Read([]byte{0})
	openBrowser("https://cloud.digitalocean.com/settings/security")
	return
}

func herokuCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "heroku", nil, nil)
	}()
	_, err = krdclient.RequestMe()
	if err != nil {
		PrintFatal(os.Stderr, "Failed to retrieve your public key:", err)
	}
	PrintErr(os.Stderr, "Adding your SSH public key using heroku toolbelt.")
	addKeyCmd := exec.Command("heroku", "keys:add", filepath.Join(os.Getenv("HOME"), ".ssh", "id_kryptonite.pub"))
	addKeyCmd.Stdin = os.Stdin
	addKeyCmd.Stdout = os.Stdout
	addKeyCmd.Stderr = os.Stderr
	addKeyCmd.Run()
	return
}

func gcloudCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "gcloud", nil, nil)
	}()
	copyKey()
	PrintErr(os.Stderr, "Public key copied to clipboard.")
	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to Google Cloud. Then click \"Edit\" and paste your public key.")
	os.Stdin.Read([]byte{0})
	openBrowser("https://console.cloud.google.com/compute/metadata/sshKeys")
	return
}

func awsCommand(c *cli.Context) (err error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "aws", nil, nil)
	}()
	me, err := copyKey()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	sshPk, err := me.SSHPublicKey()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	if sshPk.Type() != "ssh-rsa" {
		PrintFatal(os.Stderr, fmt.Sprintf("Unsupported key type: %s, AWS only supports ssh-rsa keys", sshPk.Type()))
	}

	PrintErr(os.Stderr, "Public key copied to clipboard.")
	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to Amazon Web Services. Then click \"Import Key Pair\" and paste your public key.")
	os.Stdin.Read([]byte{0})
	openBrowser("https://console.aws.amazon.com/ec2/v2/home?#KeyPairs:sort=keyName")
	return
}

func main() {
	app := cli.NewApp()
	app.Name = "kr"
	app.Usage = "communicate with Kryptonite and krd - the Kryptonite daemon"
	app.Version = kr.CURRENT_VERSION.String()
	app.Flags = []cli.Flag{}
	app.Commands = []cli.Command{
		cli.Command{
			Name:  "pair",
			Usage: "Initiate pairing of this workstation with a phone running Kryptonite.",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "force",
					Usage: "Do not ask for confirmation to unpair a currently paired device.",
				},
			},
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
			Name:  "aws,bitbucket,digitalocean,gcloud,github,gitlab,heroku",
			Usage: "Upload your public key to this site. Copies your public key to the clipboard and opens the site's settings page.",
		},
		cli.Command{
			Name:   "github",
			Usage:  "Upload your public key to GitHub. Copies your public key to the clipboard and opens GitHub settings.",
			Action: githubCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "gitlab",
			Usage:  "Upload your public key to GitLab. Copies your public key to the clipboard and opens your GitLab profile.",
			Action: gitlabCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "bitbucket",
			Usage:  "Upload your public key to BitBucket. Copies your public key to the clipboard and opens BitBucket settings.",
			Action: bitbucketCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "digitalocean",
			Usage:  "Upload your public key to DigitalOcean. Copies your public key to the clipboard and opens DigitalOcean settings.",
			Action: digitaloceanCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "digital-ocean",
			Usage:  "Upload your public key to DigitalOcean. Copies your public key to the clipboard and opens DigitalOcean settings.",
			Action: digitaloceanCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "heroku",
			Usage:  "Upload your public key to Heroku. Copies your public key to the clipboard and opens Heroku settings.",
			Action: herokuCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "aws",
			Usage:  "Upload your public key to Amazon Web Services. Copies your public key to the clipboard and opens the AWS Console.",
			Action: awsCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "gcloud",
			Usage:  "Upload your public key to Google Cloud. Copies your public key to the clipboard and opens the Google Cloud Console.",
			Action: gcloudCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "add",
			Usage:  "kr add <user@server or SSH alias> -- add your Kryptonite SSH public key to the server.",
			Action: addCommand,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "port, p",
					Usage: "Port of SSH server",
				},
			},
		},
		cli.Command{
			Name:   "restart",
			Usage:  "Restart the Kryptonite daemon.",
			Action: restartCommand,
		},
		cli.Command{
			Name:   "upgrade",
			Usage:  "Upgrade Kryptonite on this workstation.",
			Action: upgradeCommand,
		},
		cli.Command{
			Name:   "unpair",
			Usage:  "Unpair this workstation from a phone running Kryptonite.",
			Action: unpairCommand,
		},
		cli.Command{
			Name:   "uninstall",
			Usage:  "Uninstall Kryptonite from this workstation.",
			Action: uninstallCommand,
		},
		cli.Command{
			Name:   "debugaws",
			Usage:  "Check connectivity to AWS SQS.",
			Action: debugAWSCommand,
		},
	}
	app.Run(os.Args)
}
