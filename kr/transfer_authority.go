package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kryptco/kr"
	"github.com/kryptco/kr/krdclient"
	"github.com/urfave/cli"
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

func getFilePersister() (files kr.FilePersister, err error) {
	krdir, err := kr.KrDir()
	if err != nil {
		return
	}

	files = kr.FilePersister{
		PairingDir: krdir,
		SSHDir:     filepath.Join(kr.UnsudoedHomeDir(), ".ssh"),
	}

	return
}

func doRequestHostInfo() (hostInfo kr.HostInfo, err error) {
	os.Stderr.WriteString(kr.Cyan("Kryptonite ▶ Requesting logs from phone") + "\r\n")

	response, err := krdclient.RequestHosts()
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

	os.Stderr.WriteString(kr.Green("Kryptonite ▶ Obtained host list successfully ✔") + "\r\n")
	hostInfo = *response.HostInfo

	return
}

func doManualRePairWithNewKryptonite(c *cli.Context, newProfile kr.Profile) (err error) {
	files, err := getFilePersister()
	if err != nil {
		return
	}

	pairingFilePath, err := kr.KrDirFile(kr.PAIRING_FILENAME)
	if err != nil {
		return
	}

	// the temporary new pairing filelocation
	pairingTransferNewFilePath, err := kr.KrDirFile(kr.PAIRING_TRANSFER_NEW_FILENAME)
	if err != nil {
		return
	}

	err = exec.Command("killall", "krd").Run()
	if err != nil {
		return
	}

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

func doManualRePairWithOldKryptonite(c *cli.Context, oldProfile kr.Profile) (err error) {
	files, err := getFilePersister()
	if err != nil {
		return
	}

	pairingFilePath, err := kr.KrDirFile(kr.PAIRING_FILENAME)
	if err != nil {
		return
	}

	pairingTransferOldFilePath, err := kr.KrDirFile(kr.PAIRING_TRANSFER_OLD_FILENAME)
	if err != nil {
		return
	}

	// the temporary new pairing filelocation
	pairingTransferNewFilePath, err := kr.KrDirFile(kr.PAIRING_TRANSFER_NEW_FILENAME)
	if err != nil {
		return
	}

	err = exec.Command("killall", "krd").Run()
	if err != nil {
		return
	}

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

/// pair with the new Kryptonite device
func doPairNewKryptoniteDevice(c *cli.Context) (newProfile kr.Profile, err error) {
	pairingFilePath, err := kr.KrDirFile(kr.PAIRING_FILENAME)
	if err != nil {
		return
	}

	// the temporary pairing filelocation
	pairingTransferOldFilePath, err := kr.KrDirFile(kr.PAIRING_TRANSFER_OLD_FILENAME)
	if err != nil {
		return
	}

	err = exec.Command("killall", "krd").Run()
	if err != nil {
		return
	}

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

	// 2. pair with the new kryptonite device to get it's public key
	os.Stderr.WriteString(kr.Magenta("\nNext, pair with your ") + kr.Green("NEW") + kr.Magenta(" Kryptonite device. ") + "\r\n")

	err = pairCommandForce()
	if err != nil {
		return
	}

	newProfile, err = krdclient.RequestMe()
	if err != nil {
		return
	}

	return
}

/// Pair krd with the old Kryptonite device
func doPairOldKryptoniteDevice(c *cli.Context) (oldProfile kr.Profile, err error) {
	/// pair with **old** device first
	err = pairCommandForce()
	if err != nil {
		return
	}

	oldProfile, err = krdclient.RequestMe()
	if err != nil {
		return
	}

	return
}

func getAndPrintSummary(hosts []kr.UserAndHost, pgpUserIDs []string) (specialCases []string) {
	specialCases = make([]string, 0)

	os.Stderr.WriteString("\n=== " + kr.Yellow("SUMMARY") + " ===\r\n")

	os.Stderr.WriteString("\nHosts to transfer authority to\r\n")

	for _, host := range hosts {
		if TransferSpecialServices[host.Host] {
			specialCases = append(specialCases, host.Host)
			continue
		}

		if TransferExcludeServices[host.Host] {
			continue
		}

		os.Stdout.WriteString("- " + kr.Green(host.User) + " @ " + kr.Green(host.Host) + "\r\n")
	}

	os.Stdout.WriteString("\nAdditional actions\r\n")

	for _, host := range specialCases {
		os.Stdout.WriteString("- Upload SSH public-key to " + kr.Green(host) + "\r\n")
	}

	if len(pgpUserIDs) > 0 {
		specialCases = append(specialCases, "pgp-github.com")
		idsString := "\r\n\t" + strings.Join(pgpUserIDs, "\r\n\t")
		os.Stdout.WriteString("- Upload PGP public key to GitHub user ids (emails):" + kr.Green(idsString) + "\r\n")

	}

	os.Stderr.WriteString("\n=== " + kr.Yellow("END OF SUMMARY") + " ===\r\n")
	return
}

/// for a given user@host add the `authorizedPublicKeyString` to the hosts authorized_key file
/// using SSH
func transferAuthorizePublicKey(userAndHost kr.UserAndHost, authorizedPublicKeyString string) (err error) {
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
	//	note that Kryptonite still validates the host key when granting access
	args = append(args, "-o StrictHostKeyChecking=no", "-o UserKnownHostsFile=/dev/null\"")

	// inspired by ssh-copy-id
	args = append(args, "exec sh -c 'cd ; umask 077 ; mkdir -p .ssh && cat >> .ssh/authorized_keys || exit 1 ; if type restorecon >/dev/null 2>&1 ; then restorecon -F .ssh .ssh/authorized_keys ; fi'")
	sshCommand := exec.Command("ssh", args...)
	sshCommand.Stdin = authorizedKeyReader
	sshCommand.Stdout = os.Stdout
	sshCommand.Stderr = os.Stderr
	err = sshCommand.Run()

	if err == nil {
		os.Stderr.WriteString(kr.Green("Success, access granted to ") + userAndHost.User + " @ " + userAndHost.Host + "\r\n")
	}

	return
}

func transferAuthority(c *cli.Context) (err error) {
	err = transferAuthorityMain(c)
	if err != nil {
		os.Stderr.WriteString("\n" + kr.Red("Error: "+err.Error()) + "\r\n")
	}

	return
}

func transferAuthorityMain(c *cli.Context) (err error) {

	isDryRun := c.Bool("d")

	os.Stderr.WriteString(kr.Magenta("Preparing to transfer authority to a new Kryptonite public key. ") + "\r\n")
	if isDryRun {
		os.Stderr.WriteString(kr.Yellow("WARNING: this is only a dry run.") + "\r\n\n")
	}

	os.Stderr.WriteString("\n" + kr.Magenta("First, pair with your ") + kr.Yellow("old") + kr.Magenta(" Kryptonite device.") + "\r\n")

	<-time.After(2 * time.Second)

	/// pair with old kryptonite
	oldProfile, err := doPairOldKryptoniteDevice(c)
	if err != nil {
		return
	}
	os.Stderr.WriteString(kr.Green("Success, paired with OLD Kryptonite device ✔") + "\r\n")

	// pause
	<-time.After(time.Second)

	os.Stderr.WriteString("\n" + "Next, kr will request user@hostname access logs from Kryptonite to get a list of hosts that you will need to authorize for your new Kryptonite public-key." + "\r\n")

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

	/// pair with new Kryptonite to get new session and new public key
	newProfile, err := doPairNewKryptoniteDevice(c)
	if err != nil {
		return
	}
	os.Stderr.WriteString("\n" + kr.Green("Success, paired with NEW Kryptonite device ✔") + "\r\n")

	newAuthorizedPublicKey, err := newProfile.AuthorizedKeyString()
	if err != nil {
		return
	}

	/// manually re-pair with old session
	err = doManualRePairWithOldKryptonite(c, oldProfile)
	if err != nil {
		return
	}

	failedHosts := make([]kr.UserAndHost, 0)

	/// proceed to transfer authority to the new servers
	for _, host := range hosts {
		if TransferSpecialServices[host.Host] || TransferExcludeServices[host.Host] {
			continue
		}

		message := "\nAuthorize access to: " + kr.Magenta(host.User+" @ "+host.Host) + "?"
		if !confirm(os.Stderr, message) {
			os.Stderr.WriteString(kr.Yellow("Skipped") + "\r\n")
			continue
		}

		err = transferAuthorizePublicKey(host, newAuthorizedPublicKey)
		if err != nil {
			failedHosts = append(failedHosts, host)
			os.Stderr.WriteString(kr.Red("× failed authorizing "+host.User+" @ "+host.Host) + "\r\n")
			continue
		}
	}

	// show failed hosts
	handleFailures(failedHosts)

	/// manually re-pair with new session
	err = doManualRePairWithNewKryptonite(c, newProfile)
	if err != nil {
		return
	}

	// perform special cases (i.e. GitHub, etc)
	handleSpecialCases(c, specialCases, pgpUserIDs)

	os.Stderr.WriteString(kr.Green("\nDone. Your new Kryptonite public-key is authorized and ready to use ✔") + "\r\n\n")

	return
}

/// print warning about failed hosts
func handleFailures(failures []kr.UserAndHost) {
	if len(failures) == 0 {
		return
	}

	os.Stderr.WriteString("\nFailed authorizing the following hosts:\r\n")

	for _, host := range failures {
		os.Stderr.WriteString(kr.Red("× "+host.User+" @ "+host.Host) + "\r\n")
	}
}

/// helper for special cases public key upload
func handleSpecialCases(c *cli.Context, specialCases []string, pgpUserIDs []string) {
	os.Stderr.WriteString("\n" + kr.Magenta("Special cases") + "\r\n")

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
				if confirm(os.Stderr, "include UserID: "+kr.Green(userID)) {
					_, _ = krdclient.RequestMeForceRefresh(&userID)
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
