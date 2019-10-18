package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli"
	. "krypt.co/kr/common/persistance"
	. "krypt.co/kr/common/protocol"
	. "krypt.co/kr/common/socket"
	. "krypt.co/kr/common/util"
	. "krypt.co/kr/daemon/client"
)

/// Constants
var TransferSpecialServices = map[string]bool{
	"github.com":     true,
	"pgp-github.com": true,
	"bitbucket.org":  true,
	"gitlab.com":     true,
}

var TransferExcludeServices = map[string]bool{
	"me.krypt.co":      true,
	"pintest.krypt.co": true,
}

/// Custom Error Types

type BadHostInfoError struct {
	Message string
}

func (b BadHostInfoError) Error() string {
	return b.Message
}

/// Helper Functions

func getFilePersister() (files FilePersister, err error) {
	krdir, err := KrDir()
	if err != nil {
		return
	}

	files = FilePersister{
		PairingDir: krdir,
		SSHDir:     filepath.Join(HomeDir(), ".ssh"),
	}

	return
}

func doRequestHostInfo() (hostInfo HostInfo, err error) {
	os.Stderr.WriteString(Cyan("Krypton ▶ Requesting logs from phone") + "\r\n")

	response, err := RequestHosts()
	if err != nil {
		return
	}

	if response.Error != nil {
		err = BadHostInfoError{Message: *response.Error}
		return
	}

	if response.HostInfo == nil {
		err = BadHostInfoError{Message: "Empty HostInfo"}
		return
	}

	os.Stderr.WriteString(Green("Krypton ▶ Obtained host list successfully ✔") + "\r\n")
	hostInfo = *response.HostInfo

	return
}

func doManualRePairWithNewKrypton(c *cli.Context, newProfile Profile) (err error) {
	files, err := getFilePersister()
	if err != nil {
		return
	}

	pairingFilePath, err := KrDirFile(PAIRING_FILENAME)
	if err != nil {
		return
	}

	// the temporary new pairing filelocation
	pairingTransferNewFilePath, err := KrDirFile(PAIRING_TRANSFER_NEW_FILENAME)
	if err != nil {
		return
	}

	killKrd()

	// move the new pairing back
	err = os.Rename(pairingTransferNewFilePath, pairingFilePath)
	if err != nil {
		return
	}

	err = files.SaveMe(newProfile)
	if err != nil {
		return
	}

	// restart krd
	err = restartCommandOptions(c, false)
	if err != nil {
		return
	}

	err = files.SaveMySSHPubKey(newProfile)
	if err != nil {
		return
	}

	///TODO: fix workaround sleep
	<-time.After(time.Second)

	return
}

func doManualRePairWithOldKrypton(c *cli.Context, oldProfile Profile) (err error) {
	files, err := getFilePersister()
	if err != nil {
		return
	}

	pairingFilePath, err := KrDirFile(PAIRING_FILENAME)
	if err != nil {
		return
	}

	pairingTransferOldFilePath, err := KrDirFile(PAIRING_TRANSFER_OLD_FILENAME)
	if err != nil {
		return
	}

	// the temporary new pairing filelocation
	pairingTransferNewFilePath, err := KrDirFile(PAIRING_TRANSFER_NEW_FILENAME)
	if err != nil {
		return
	}

	killKrd()

	// move the new pairing temporarily
	err = os.Rename(pairingFilePath, pairingTransferNewFilePath)
	if err != nil {
		return
	}

	// move back the old pairing temporarily
	err = os.Rename(pairingTransferOldFilePath, pairingFilePath)
	if err != nil {
		return
	}

	err = files.SaveMe(oldProfile)
	if err != nil {
		return
	}

	// restart krd
	err = restartCommandOptions(c, false)
	if err != nil {
		return
	}

	err = files.SaveMySSHPubKey(oldProfile)
	if err != nil {
		return
	}

	///TODO: fix workaround sleep
	<-time.After(time.Second)

	return
}

/// pair with the new Krypton device
func doPairNewKryptonDevice(c *cli.Context) (newProfile Profile, err error) {
	pairingFilePath, err := KrDirFile(PAIRING_FILENAME)
	if err != nil {
		return
	}

	// the temporary pairing filelocation
	pairingTransferOldFilePath, err := KrDirFile(PAIRING_TRANSFER_OLD_FILENAME)
	if err != nil {
		return
	}

	killKrd()

	// move it temporarily
	err = os.Rename(pairingFilePath, pairingTransferOldFilePath)
	if err != nil {
		return
	}

	// restart krd
	err = restartCommandOptions(c, false)
	if err != nil {
		return
	}

	///TODO: fix workaround sleep
	<-time.After(time.Second)

	// 2. pair with the new Krypton device to get it's public key
	os.Stderr.WriteString(Magenta("\nNext, pair with your ") + Green("NEW") + Magenta(" Krypton device. ") + "\r\n")

	err = pairCommandForce()
	if err != nil {
		return
	}

	newProfile, err = RequestMe()
	if err != nil {
		return
	}

	return
}

/// Pair krd with the old Krypton device
func doPairOldKryptonDevice(c *cli.Context) (oldProfile Profile, err error) {
	/// pair with **old** device first
	err = pairCommandForce()
	if err != nil {
		return
	}

	oldProfile, err = RequestMe()
	if err != nil {
		return
	}

	return
}

func getAndPrintSummary(hosts []UserAndHost, pgpUserIDs []string) (specialCases []string) {
	specialCases = make([]string, 0)

	os.Stderr.WriteString("\n=== " + Yellow("SUMMARY") + " ===\r\n")

	os.Stderr.WriteString("\nHosts to transfer authority to\r\n")

	for _, host := range hosts {
		if TransferSpecialServices[host.Host] {
			specialCases = append(specialCases, host.Host)
			continue
		}

		if TransferExcludeServices[host.Host] {
			continue
		}

		os.Stdout.WriteString("- " + Green(host.User) + " @ " + Green(host.Host) + "\r\n")
	}

	os.Stdout.WriteString("\nAdditional actions\r\n")

	for _, host := range specialCases {
		os.Stdout.WriteString("- Upload SSH public-key to " + Green(host) + "\r\n")
	}

	if len(pgpUserIDs) > 0 {
		specialCases = append(specialCases, "pgp-github.com")
		idsString := "\r\n\t" + strings.Join(pgpUserIDs, "\r\n\t")
		os.Stdout.WriteString("- Upload PGP public key to GitHub user ids (emails):" + Green(idsString) + "\r\n")

	}

	os.Stderr.WriteString("\n=== " + Yellow("END OF SUMMARY") + " ===\r\n")
	return
}

/// for a given user@host add the `authorizedPublicKeyString` to the hosts authorized_key file
/// using SSH
func transferAuthorizePublicKey(userAndHost UserAndHost, authorizedPublicKeyString string) (err error) {
	authorizedKey := append([]byte(authorizedPublicKeyString), []byte("\n")...)

	hostAndPort := strings.Split(userAndHost.Host, ":")
	host := hostAndPort[0]

	var port = ""
	if len(hostAndPort) == 2 {
		port = hostAndPort[1]
	}

	server := userAndHost.User + "@" + host

	authorizedKeyReader := bytes.NewReader(authorizedKey)
	args := []string{server}
	if port != "" {
		args = append(args, "-p "+port)
	}
	//	disable host key checking in case this workstation does not have the target host in its known_hosts
	//	note that Krypton still validates the host key when granting access
	args = append(args, "-o StrictHostKeyChecking=no", "-o UserKnownHostsFile=/dev/null\"")

	// inspired by ssh-copy-id
	args = append(args, "exec sh -c 'cd ; umask 077 ; mkdir -p .ssh && cat >> .ssh/authorized_keys || exit 1 ; if type restorecon >/dev/null 2>&1 ; then restorecon -F .ssh .ssh/authorized_keys ; fi'")
	sshCommand := exec.Command("ssh", args...)
	sshCommand.Stdin = authorizedKeyReader
	sshCommand.Stdout = os.Stdout
	sshCommand.Stderr = os.Stderr
	err = sshCommand.Run()

	if err == nil {
		os.Stderr.WriteString(Green("Success, access granted to ") + userAndHost.User + " @ " + userAndHost.Host + "\r\n")
	}

	return
}

func transferAuthority(c *cli.Context) (err error) {
	err = transferAuthorityMain(c)
	if err != nil {
		os.Stderr.WriteString("\n" + Red("Error: "+err.Error()) + "\r\n")
	}

	return
}

func transferAuthorityMain(c *cli.Context) (err error) {

	isDryRun := c.Bool("d")

	os.Stderr.WriteString(Magenta("Preparing to transfer authority to a new Krypton public key. ") + "\r\n")
	if isDryRun {
		os.Stderr.WriteString(Yellow("WARNING: this is only a dry run.") + "\r\n\n")
	}

	os.Stderr.WriteString("\n" + Magenta("First, pair with your ") + Yellow("old") + Magenta(" Krypton device.") + "\r\n")

	<-time.After(2 * time.Second)

	/// pair with old Krypton
	oldProfile, err := doPairOldKryptonDevice(c)
	if err != nil {
		return
	}
	os.Stderr.WriteString(Green("Success, paired with OLD Krypton device ✔") + "\r\n")

	// pause
	<-time.After(time.Second)

	os.Stderr.WriteString("\n" + "Next, kr will request user@hostname access logs from Krypton to get a list of hosts that you will need to authorize for your new Krypton public-key." + "\r\n")

	// pause
	<-time.After(time.Second)

	/// request HostInfo
	hostInfo, err := doRequestHostInfo()
	if err != nil {
		return
	}

	hosts := hostInfo.Hosts
	pgpUserIDs := hostInfo.PGPUserIDs

	/// print summary, get special cases from summary
	specialCases := getAndPrintSummary(hosts, pgpUserIDs)

	if isDryRun {
		return
	}

	if !confirm(os.Stderr, "\nContinue?") {
		return
	}

	/// pair with new Krypton to get new session and new public key
	newProfile, err := doPairNewKryptonDevice(c)
	if err != nil {
		return
	}
	os.Stderr.WriteString("\n" + Green("Success, paired with NEW Krypton device ✔") + "\r\n")

	newAuthorizedPublicKey, err := newProfile.AuthorizedKeyString()
	if err != nil {
		return
	}

	/// manually re-pair with old session
	err = doManualRePairWithOldKrypton(c, oldProfile)
	if err != nil {
		return
	}

	failedHosts := make([]UserAndHost, 0)

	/// proceed to transfer authority to the new servers
	for _, host := range hosts {
		if TransferSpecialServices[host.Host] || TransferExcludeServices[host.Host] {
			continue
		}

		message := "\nAuthorize access to: " + Magenta(host.User+" @ "+host.Host) + "?"
		if !confirm(os.Stderr, message) {
			os.Stderr.WriteString(Yellow("Skipped") + "\r\n")
			continue
		}

		err = transferAuthorizePublicKey(host, newAuthorizedPublicKey)
		if err != nil {
			failedHosts = append(failedHosts, host)
			os.Stderr.WriteString(Red("× failed authorizing "+host.User+" @ "+host.Host) + "\r\n")
			continue
		}
	}

	// show failed hosts
	handleFailures(failedHosts)

	/// manually re-pair with new session
	err = doManualRePairWithNewKrypton(c, newProfile)
	if err != nil {
		return
	}

	// perform special cases (i.e. GitHub, etc)
	handleSpecialCases(c, specialCases, pgpUserIDs)

	os.Stderr.WriteString(Green("\nDone. Your new Krypton public-key is authorized and ready to use ✔") + "\r\n\n")

	return
}

/// print warning about failed hosts
func handleFailures(failures []UserAndHost) {
	if len(failures) == 0 {
		return
	}

	os.Stderr.WriteString("\nFailed authorizing the following hosts:\r\n")

	for _, host := range failures {
		os.Stderr.WriteString(Red("× "+host.User+" @ "+host.Host) + "\r\n")
	}
}

/// helper for special cases public key upload
func handleSpecialCases(c *cli.Context, specialCases []string, pgpUserIDs []string) {
	os.Stderr.WriteString("\n" + Magenta("Special cases") + "\r\n")

	for _, service := range specialCases {

		if service == "github.com" {
			if !confirm(os.Stderr, "Add SSH public-key to GitHub?") {
				continue
			}

			githubCommand(c)
		}

		if service == "pgp-github.com" {
			if !confirm(os.Stderr, "Add PGP public-key to GitHub?") {
				return
			}

			// workaround to mint pgp keys with multiple user_ids
			for _, userID := range pgpUserIDs {
				if confirm(os.Stderr, "include UserID: "+Green(userID)) {
					_, _ = RequestMeForceRefresh(&userID)
				}
			}

			githubPGPCommand(c)
		}

		if service == "bitbucket.org" {
			if !confirm(os.Stderr, "Add SSH public-key to Bitbucket?") {
				continue
			}

			bitbucketCommand(c)
		}

		if service == "gitlab.com" {
			if !confirm(os.Stderr, "Add SSH public-key to GitLab?") {
				continue
			}

			gitlabCommand(c)
		}
	}
}
