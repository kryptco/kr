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
	"krypt.co/kr/common/version"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/urfave/cli"

	. "krypt.co/kr/common/analytics"
	. "krypt.co/kr/common/persistance"
	. "krypt.co/kr/common/protocol"
	. "krypt.co/kr/common/socket"
	. "krypt.co/kr/common/util"

	krdclient "krypt.co/kr/daemon/client"
)

func PrintFatal(stderr io.ReadWriter, msg string, args ...interface{}) {
	if len(args) == 0 {
		PrintErr(stderr, msg)
	} else {
		PrintErr(stderr, msg, args...)
	}
	os.Exit(1)
}

func runCommandWithUserInteraction(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func PrintErr(stderr io.ReadWriter, msg string, args ...interface{}) {
	stderr.Write([]byte(fmt.Sprintf(msg, args...) + "\n"))
}

func confirmOrFatal(stderr io.ReadWriter, message string) {
	if !confirm(stderr, message) {
		PrintFatal(stderr, "Aborting.")
	}
}

func confirm(stderr io.ReadWriter, message string) bool {
	stderr.Write([]byte(fmt.Sprintf(message + " [y/N] ")))
	in := []byte{0, 0}
	os.Stdin.Read(in)
	return in[0] == 'y'
}

func pairCommand(c *cli.Context) (err error) {
	go func() {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "pair", nil, nil)
	}()
	if !IsKrdRunning() {
		err = startKrd()
		if err != nil {
			return
		}
		<-time.After(time.Second)
	}
	err = autoEditSSHConfig()
	if err != nil {
		PrintErr(os.Stderr, Red("Krypton ▶ Error verifying SSH config: "+err.Error()))
		<-time.After(2 * time.Second)
		PrintErr(os.Stderr, Red("Krypton ▶ Continuing with pairing..."))
		<-time.After(2 * time.Second)
	}
	name := c.String("name")
	nameOpt := &name
	if *nameOpt == "" {
		nameOpt = nil
	}
	return pairOver(DaemonSocketOrFatal(), c.Bool("force"), nameOpt, os.Stdout, os.Stderr)
}

func pairCommandForce() (err error) {
	if !IsKrdRunning() {
		err = startKrd()
		if err != nil {
			return
		}
		<-time.After(time.Second)
	}
	err = autoEditSSHConfig()
	if err != nil {
		PrintErr(os.Stderr, Red("Krypton ▶ Error verifying SSH config: "+err.Error()))
		<-time.After(2 * time.Second)
		PrintErr(os.Stderr, Red("Krypton ▶ Continuing with pairing..."))
		<-time.After(2 * time.Second)
	}

	return pairOver(DaemonSocketOrFatal(), true, nil, os.Stdout, os.Stderr)
}

func pairOver(unixFile string, forceUnpair bool, name *string, stdout io.ReadWriter, stderr io.ReadWriter) (err error) {
	//	Listen for incompatible enclave notifications
	go func() {
		r, err := OpenNotificationReader("")
		if err != nil {
			os.Stderr.WriteString("error connecting to notification reader: " + err.Error())
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
		meConn, err := DaemonDialWithTimeout(unixFile)
		if err != nil {
			PrintFatal(stderr, "Could not connect to Krypton daemon. Make sure it is running by typing \"kr restart\".")
		}
		_, err = krdclient.RequestMeOver(meConn)
		if err == nil {
			confirmOrFatal(stderr, "Already paired, unpair current session?")
		}
	}
	putConn, err := DaemonDialWithTimeout(unixFile)
	if err != nil {
		PrintFatal(stderr, "Could not connect to Krypton daemon. Make sure it is running by typing \"kr restart\".")
	}
	defer putConn.Close()

	var pairingOptions PairingOptions
	pairingOptions.WorkstationName = name
	body, err := json.Marshal(pairingOptions)
	if err != nil {
		PrintFatal(stderr, err.Error())
	}

	putPair, err := http.NewRequest("PUT", "/pair", bytes.NewBuffer(body))
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
	stdout.Write([]byte("Scan this QR Code with the Krypton mobile app to connect it with this workstation. Maximize the window and/or lower your font size if the QR code does not fit."))
	stdout.Write([]byte("\r\n"))

	//	Check/wait for pairing
	getConn, err := DaemonDialWithTimeout(unixFile)
	if err != nil {
		PrintFatal(stderr, "Could not connect to Krypton daemon. Make sure it is running by typing \"kr restart\".")
	}
	defer putConn.Close()
	me, err := krdclient.RequestMeForceRefreshOver(getConn, nil)

	clearCommand := exec.Command("clear")
	clearCommand.Stdout = stdout
	clearCommand.Run()

	if err != nil {
		PrintFatal(stderr, err.Error())
	}

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
	Analytics{}.PostEventUsingPersistedTrackingID("kr", "unpair", nil, nil)
	return unpairOver(DaemonSocketOrFatal(), os.Stdout, os.Stderr)
}

func unpairOver(unixFile string, stdout io.ReadWriter, stderr io.ReadWriter) (err error) {
	conn, err := DaemonDialWithTimeout(unixFile)
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
		PrintFatal(stderr, "Unpair failed, ensure the Krypton daemon is running with \"kr restart\".")
	case http.StatusOK:
	default:
		PrintFatal(stderr, "Unpair failed with error %d", response.StatusCode)
	}
	stdout.Write([]byte("Unpaired Krypton.\r\n"))
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
	Analytics{}.PostEventUsingPersistedTrackingID("kr", "me", nil, nil)
	return
}

func mePGPCommand(c *cli.Context) (err error) {
	userID := globalGitUserIDOrFatal()
	me, err := krdclient.RequestMeForceRefresh(&userID)
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}

	pgp, err := me.AsciiArmorPGPPublicKey()
	if err != nil {
		PrintFatal(os.Stderr, "You do not yet have a PGP public key. Make sure you have the latest version of the Krypton app and that you have run "+Cyan("kr codesign")+" successfully.")
	}
	fmt.Println(pgp)

	PrintErr(os.Stderr, "\r\nCopy this key to your clipboard using "+Cyan("kr copy pgp")+" or add it to Github using "+Cyan("kr github pgp")+". Type "+Cyan("kr")+" to see all available commands.")
	Analytics{}.PostEventUsingPersistedTrackingID("kr", "me pgp", nil, nil)
	return
}

func copyCommand(c *cli.Context) (err error) {
	copyKey()
	Analytics{}.PostEventUsingPersistedTrackingID("kr", "copy", nil, nil)
	return
}

func copyPGPCommand(c *cli.Context) (err error) {
	copyPGPKey()
	Analytics{}.PostEventUsingPersistedTrackingID("kr", "copy pgp", nil, nil)
	return
}

func copyKey() (me Profile, err error) {
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
		PrintErr(os.Stderr, err.Error())
		PrintErr(os.Stderr, "Or copy the following lines to your clipboard:\n")
		PrintErr(os.Stderr, authorizedKey)
		err = nil
	} else {
		PrintErr(os.Stderr, "SSH public key "+Cyan("copied to clipboard")+".")
	}
	return
}

func copyPGPKey() (me Profile, err error) {
	me, pk, err := copyPGPKeyNonFatalOnClipboardError()
	if err != nil {
		PrintErr(os.Stderr, err.Error())
		PrintErr(os.Stderr, "Or copy the following lines to your clipboard:\n")
		PrintErr(os.Stderr, pk)
		err = nil
	} else {
		PrintErr(os.Stderr, "PGP public key "+Cyan("copied to clipboard")+".")
	}
	return
}

func copyPGPKeyNonFatalOnClipboardError() (me Profile, pk string, err error) {
	userID := globalGitUserIDOrFatal()
	me, err = krdclient.RequestMeForceRefresh(&userID)
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	pk, err = me.AsciiArmorPGPPublicKey()
	if err != nil {
		PrintFatal(os.Stderr, "You do not yet have a PGP public key. Make sure you have the latest version of the Krypton app and that you have run "+Cyan("kr codesign")+" successfully.")
	}
	err = clipboard.WriteAll(pk)
	return
}

func githubCommand(c *cli.Context) (err error) {
	go func() {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "github", nil, nil)
	}()
	copyKey()
	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to GitHub. Then click \"New SSH Key\" and paste your public key.")
	os.Stdin.Read([]byte{0})
	openBrowser("https://github.com/settings/keys")
	return
}

func githubPGPCommand(c *cli.Context) (err error) {
	go func() {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "github pgp", nil, nil)
	}()
	copyPGPKey()
	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to GitHub. Then click "+Cyan("New GPG key")+" and paste your public key.")
	os.Stdin.Read([]byte{0})
	openBrowser("https://github.com/settings/keys")
	return
}

func getGheUrlOrFatal(c *cli.Context) string {
	if c.String("url") != "" {
		return c.String("url")
	}
	PrintErr(os.Stderr, "Please enter your GitHub Enterprise URL, i.e. github.mit.edu")
	buf := make([]byte, 1024)
	n, _ := os.Stdin.Read(buf)
	return strings.TrimSpace(string(buf[:n]))
}

func gheCommand(c *cli.Context) (err error) {
	go func() {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "ghe", nil, nil)
	}()
	copyKey()

	gheURL := getGheUrlOrFatal(c)

	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to GitHub Enterprise. Then click \"New SSH Key\" and paste your public key.")
	os.Stdin.Read([]byte{0})
	openBrowser("https://" + gheURL + "/settings/keys")
	return
}

func ghePGPCommand(c *cli.Context) (err error) {
	go func() {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "ghe pgp", nil, nil)
	}()
	copyPGPKey()

	gheURL := getGheUrlOrFatal(c)

	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to GitHub Enterprise. Then click \"New GPG Key\" and paste your public key.")
	os.Stdin.Read([]byte{0})
	openBrowser("https://" + gheURL + "/settings/keys")
	return
}

func gitlabCommand(c *cli.Context) (err error) {
	go func() {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "gitlab", nil, nil)
	}()
	copyKey()
	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to GitLab. Then paste your public key and click \"Add key.\"")
	os.Stdin.Read([]byte{0})
	openBrowser("https://gitlab.com/profile/keys")
	return
}

func bitbucketCommand(c *cli.Context) (err error) {
	go func() {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "bitbucket", nil, nil)
	}()
	copyKey()
	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to BitBucket. Then click \"Add key\" and paste your public key.")
	os.Stdin.Read([]byte{0})
	openBrowser("https://bitbucket.org/account/ssh-keys/")
	return
}

func digitaloceanCommand(c *cli.Context) (err error) {
	go func() {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "digitalocean", nil, nil)
	}()
	copyKey()
	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to DigitalOcean. Then click \"Add SSH Key\" and paste your public key.")
	os.Stdin.Read([]byte{0})
	openBrowser("https://cloud.digitalocean.com/settings/security")
	return
}

func herokuCommand(c *cli.Context) (err error) {
	go func() {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "heroku", nil, nil)
	}()
	_, err = krdclient.RequestMe()
	if err != nil {
		PrintFatal(os.Stderr, "Failed to retrieve your public key: %s", err.Error())
	}
	PrintErr(os.Stderr, "Adding your SSH public key using heroku toolbelt.")
	addKeyCmd := exec.Command("heroku", "keys:add", filepath.Join(os.Getenv("HOME"), ".ssh", ID_KRYPTON_FILENAME))
	addKeyCmd.Stdin = os.Stdin
	addKeyCmd.Stdout = os.Stdout
	addKeyCmd.Stderr = os.Stderr
	addKeyCmd.Run()
	return
}

func gcloudCommand(c *cli.Context) (err error) {
	go func() {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "gcloud", nil, nil)
	}()
	copyKey()
	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to Google Cloud. Then click \"Edit\" and paste your public key.")
	os.Stdin.Read([]byte{0})
	openBrowser("https://console.cloud.google.com/compute/metadata/sshKeys")
	return
}

func awsCommand(c *cli.Context) (err error) {
	go func() {
		Analytics{}.PostEventUsingPersistedTrackingID("kr", "aws", nil, nil)
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

	<-time.After(500 * time.Millisecond)
	PrintErr(os.Stderr, "Press ENTER to open your web browser to Amazon Web Services. Then click \"Import Key Pair\" and paste your public key.")
	os.Stdin.Read([]byte{0})
	openBrowser("https://console.aws.amazon.com/ec2/v2/home?#KeyPairs:sort=keyName")
	return
}

func envCommand(c *cli.Context) (err error) {
	const ENV_VAR_USAGE = `Useful environment variables:
	KR_SKIP_SSH_CONFIG=1		Do not automatically configure ~/.ssh/config (see 'kr sshconfig --help')
	KR_SILENCE_WARNINGS=1		Do not print warnings about not being paired or a newer version of kr being available
	KR_NO_STDERR=1			Do not log anything to the terminal (useful for scripts that parse stderr)
	KR_LOG_LEVEL=<log level>	Set log level of kr/krssh
	KR_LOG_SYSLOG=true		Force krssh to log to system log`
	os.Stderr.WriteString(ENV_VAR_USAGE + "\n")
	return
}

func transferCommand(c *cli.Context) (err error) {
	return transferAuthority(c)
}

func restartCommand(c *cli.Context) (err error) {
	return restartCommandOptions(c, true)
}

func main() {
	initTerminal()
	app := cli.NewApp()
	app.Name = "kr"
	app.Usage = "communicate with Krypton and krd - the Krypton daemon"
	app.Version = version.CURRENT_VERSION.String()
	app.Flags = []cli.Flag{}
	app.Commands = []cli.Command{
		cli.Command{
			Name:  "pair",
			Usage: "Initiate pairing of this workstation with a phone running Krypton",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "force",
					Usage: "Do not ask for confirmation to unpair a currently paired device",
				},
				cli.StringFlag{
					Name:  "name, n",
					Usage: "WorkstationName for this computer",
				},
			},
			Action: pairCommand,
		},
		cli.Command{
			Name:   "me",
			Usage:  "Print your SSH public key",
			Action: meCommand,
			Subcommands: []cli.Command{
				cli.Command{
					Name:   "pgp",
					Usage:  "Print your PGP public key",
					Action: mePGPCommand,
				},
			},
		},
		cli.Command{
			Name:  "codesign",
			Usage: "Setup Krypton to sign git commits",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "interactive, i",
					Usage: "Prompt before each step",
				},
			},
			Action: codesignCommand,
			Subcommands: []cli.Command{
				cli.Command{
					Name:   "uninstall",
					Usage:  "Uninstall Krypton codesigning",
					Action: codesignUninstallCommand,
				},
				cli.Command{
					Name:   "test",
					Usage:  "Test Krypton codesigning",
					Action: codesignTestCommand,
				},
				cli.Command{
					Name:   "on",
					Usage:  "Turn on auto commit signing (requires git v2.0+)",
					Action: codesignOnCommand,
				},
				cli.Command{
					Name:   "off",
					Usage:  "Turn off auto commit signing",
					Action: codesignOffCommand,
				},
			},
		},
		cli.Command{
			Name:   "copy",
			Usage:  "Copy your SSH public key to the clipboard",
			Action: copyCommand,
			Subcommands: []cli.Command{
				cli.Command{
					Name:   "pgp",
					Usage:  "Copy your PGP public key to the clipboard.",
					Action: copyPGPCommand,
				},
			},
		},
		cli.Command{
			Name:   "env",
			Usage:  "Print useful environment variables for configuring kr/krd",
			Action: envCommand,
		},
		cli.Command{
			Name:   "sshconfig",
			Usage:  "Verify SSH is configured to use Krypton",
			Action: sshConfigCommand,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "print",
					Usage: "Print Krypton SSH config block",
				},
				cli.BoolFlag{
					Name:  "force",
					Usage: "Force append the Krypton SSH config block even if other Krypton-related lines are present",
				},
			},
		},
		cli.Command{
			Name:   "transfer",
			Usage:  "Authorize a new Krypton device to access of your servers",
			Action: transferCommand,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "dry-run, d",
					Usage: "Do a dry-run and preview all servers that kr will try to add the new public key too",
				},
			},
		},
		cli.Command{
			Name:  "aws,bitbucket,digitalocean,gcp,github,ghe,gitlab,heroku",
			Usage: "Upload your public key this service",
		},
		cli.Command{
			Name:   "github",
			Usage:  "Upload your public key to GitHub. Copies your public key to the clipboard and opens GitHub settings",
			Action: githubCommand,
			Hidden: true,
			Subcommands: []cli.Command{
				cli.Command{
					Name:   "pgp",
					Usage:  "Upload your PGP public key to GitHub. Copies your public key to the clipboard and opens GitHub settings",
					Action: githubPGPCommand,
				},
			},
		},
		cli.Command{
			Name:   "ghe",
			Usage:  "Upload your public key to GitHub Enterprise. Copies your public key to the clipboard and opens GitHub Enterprise settings",
			Action: gheCommand,
			Hidden: true,
			Subcommands: []cli.Command{
				cli.Command{
					Name:   "pgp",
					Usage:  "Upload your PGP public key to GitHub Enterprise. Copies your public key to the clipboard and opens GitHub Enterprise settings",
					Action: ghePGPCommand,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "url",
							Usage: "GitHub Enterprise URL, i.e. github.mit.edu",
						},
					},
				},
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "url",
					Usage: "GitHub Enterprise URL, i.e. github.mit.edu",
				},
			},
		},
		cli.Command{
			Name:   "gitlab",
			Usage:  "Upload your public key to GitLab. Copies your public key to the clipboard and opens your GitLab profile",
			Action: gitlabCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "bitbucket",
			Usage:  "Upload your public key to BitBucket. Copies your public key to the clipboard and opens BitBucket settings",
			Action: bitbucketCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "digitalocean",
			Usage:  "Upload your public key to DigitalOcean. Copies your public key to the clipboard and opens DigitalOcean settings",
			Action: digitaloceanCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "digital-ocean",
			Usage:  "Upload your public key to DigitalOcean. Copies your public key to the clipboard and opens DigitalOcean settings",
			Action: digitaloceanCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "heroku",
			Usage:  "Upload your public key to Heroku. Copies your public key to the clipboard and opens Heroku settings",
			Action: herokuCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "aws",
			Usage:  "Upload your public key to Amazon Web Services. Copies your public key to the clipboard and opens the AWS Console",
			Action: awsCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "gcp",
			Usage:  "Upload your public key to Google Cloud. Copies your public key to the clipboard and opens the Google Cloud Console",
			Action: gcloudCommand,
			Hidden: true,
		},
		cli.Command{
			Name:   "restart",
			Usage:  "Restart the Krypton daemon",
			Action: restartCommand,
		},
		cli.Command{
			Name:   "upgrade",
			Usage:  "Upgrade Krypton on this workstation",
			Action: upgradeCommand,
		},
		cli.Command{
			Name:   "unpair",
			Usage:  "Unpair this workstation from a phone running Krypton",
			Action: unpairCommand,
		},
		cli.Command{
			Name:   "uninstall",
			Usage:  "Uninstall Krypton from this workstation",
			Action: uninstallCommand,
		},
		cli.Command{
			Name:   "debugaws",
			Usage:  "Check connectivity to AWS SQS",
			Action: debugAWSCommand,
		},
	}
	app.Run(os.Args)
}
