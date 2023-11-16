package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli"

	. "krypt.co/kr/common/socket"
	. "krypt.co/kr/common/util"
)

const OLD_SSH_CONFIG_FORMAT = `# Added by Krypton
Host *
	PKCS11Provider %s/lib/kr-pkcs11.so
	ProxyCommand %s/bin/krssh %%h %%p
	IdentityFile ~/.ssh/id_krypton
	IdentityFile ~/.ssh/id_ed25519
	IdentityFile ~/.ssh/id_rsa
	IdentityFile ~/.ssh/id_ecdsa
	IdentityFile ~/.ssh/id_dsa`

const SSH_CONFIG_FORMAT = `# Added by Krypton
Host *
	IdentityAgent ~/.kr/krd-agent.sock
	ProxyCommand %s/bin/krssh %%h %%p
	IdentityFile ~/.ssh/id_krypton
	IdentityFile ~/.ssh/id_ed25519
	IdentityFile ~/.ssh/id_rsa
	IdentityFile ~/.ssh/id_ecdsa
	IdentityFile ~/.ssh/id_dsa`

const SSH_CONFIG_FORMAT_WIN = `# Added by Krypton
Host *
	IdentityAgent \\.\pipe\krd-agent
	ProxyCommand %s\krssh.exe %%h %%p`
/*
	IdentityFile ~/.ssh/id_krypton
	IdentityFile ~/.ssh/id_ed25519
	IdentityFile ~/.ssh/id_rsa
	IdentityFile ~/.ssh/id_ecdsa
	IdentityFile ~/.ssh/id_dsa`
*/

const OLD_PKCS11_PROVIDER_FORMAT = `PKCS11Provider %s/lib/kr-pkcs11.so`
const NEW_IDENTITY_AGENT = `IdentityAgent ~/.kr/krd-agent.sock`

const KR_SKIP_SSH_CONFIG = "KR_SKIP_SSH_CONFIG"

func getKrSSHConfigBlockOrFatal() string {
	prefix, err := getPrefix()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	var sshConfigWithPrefix string

	if localSSHSupportsIdentityAgent(){
		if runtime.GOOS == "windows" {
			sshConfigWithPrefix = fmt.Sprintf(SSH_CONFIG_FORMAT_WIN, prefix)
		} else {
			sshConfigWithPrefix = fmt.Sprintf(SSH_CONFIG_FORMAT, prefix)
		}
	} else {
		sshConfigWithPrefix = fmt.Sprintf(OLD_SSH_CONFIG_FORMAT, prefix, prefix)
	}

	return sshConfigWithPrefix
}

func sshConfigCommand(c *cli.Context) (err error) {
	if c.Bool("print") {
		os.Stdout.WriteString(getKrSSHConfigBlockOrFatal() + "\n")
		return
	}
	return editSSHConfig(true, c.Bool("force"))
}

func autoEditSSHConfig() (err error) {
	if os.Getenv(KR_SKIP_SSH_CONFIG) != "" {
		return
	}
	return editSSHConfig(false, false)
}

func getSSHConfigAndBakPaths() (string, string) {
	sshDirPath := HomeDir() + "/.ssh"
	_ = os.MkdirAll(sshDirPath, 0700)
	sshConfigPath := sshDirPath + "/config"
	sshConfigBackupPath := sshConfigPath + ".bak.kr"
	return sshConfigPath, sshConfigBackupPath
}

func editSSHConfig(prompt bool, forceAppend bool) (err error) {
	configBlock := []byte(getKrSSHConfigBlockOrFatal())

	sshConfigPath, sshConfigBackupPath := getSSHConfigAndBakPaths()

	sshConfigFile, err := os.OpenFile(sshConfigPath, os.O_RDONLY|os.O_CREATE, 0700)
	if err != nil {
		return
	}
	defer sshConfigFile.Close()
	currentConfigContents, err := ioutil.ReadAll(sshConfigFile)
	if err != nil {
		return
	}

	if bytes.Contains(currentConfigContents, configBlock) {
		if prompt {
			PrintErr(os.Stderr, Green("Krypton ▶ SSH already configured ✔"))
		}
		return
	}
	if bytes.Contains(currentConfigContents, []byte("krssh %h %p")) && !forceAppend {
		if prompt {
			PrintErr(os.Stderr, Yellow("Krypton ▶ ~/.ssh/config already contains Krypton-related configuration. Please remove all Krypton-related lines from ~/.ssh/config or run with --force."))
		}
		return
	}

	if prompt {
		if !confirm(os.Stderr, Yellow("Krypton ▶ SSH must be configured to use Krypton. Automatically configure SSH?")) {
			os.Stderr.WriteString(Yellow("Please add the following to ~/.ssh/config:") + "\n\n" + string(configBlock) + "\n\n")
			os.Stderr.WriteString("Press " + Cyan("ENTER") + " to continue")
			os.Stdin.Read([]byte{0})
			return
		}
	}
	if len(currentConfigContents) > 0 {
		err = ioutil.WriteFile(sshConfigBackupPath, currentConfigContents, 0700)
		if err != nil {
			return
		}
		if prompt {
			PrintErr(os.Stderr, Green("Krypton ▶ ~/.ssh/config backed up to ~/.ssh/config.bak.kr ✔"))
		}
	}

	newConfigContents := bytes.Join([][]byte{currentConfigContents, configBlock}, []byte("\n\n"))
	err = ioutil.WriteFile(sshConfigPath, newConfigContents, 0700)
	if err != nil {
		return
	}
	if prompt {
		PrintErr(os.Stderr, Green("Krypton ▶ SSH configured ✔"))
		<-time.After(time.Second)
	}
	return
}

func localSSHSupportsIdentityAgent() bool {
	//	Valid OpenSSH version strings:
	//	OpenSSH_6.7p1 Debian-5+deb8u4, OpenSSL 1.0.1t  3 May 2016
	//	OpenSSH_7.7p1, OpenSSL 1.0.2o  27 Mar 2018
	//  OpenSSH_for_Windows_7.7p1, LibreSSL 2.6.5
	versionOutput, err := exec.Command("ssh", "-V").CombinedOutput()
	if err != nil {
		return false
	}
	versionString := string(versionOutput)
	versionString = strings.TrimPrefix(versionString, "OpenSSH_for_Windows_")
	versionString = strings.TrimPrefix(versionString, "OpenSSH_")
	for _, suffixDelim := range []string{" ", ",", "p"} {
		versionString = strings.Split(versionString, suffixDelim)[0]
	}
	versionStringToks := strings.Split(versionString, ".")
	if len(versionStringToks) != 2 {
		return false
	}
	major, err := strconv.ParseUint(versionStringToks[0], 10, 64)
	if err != nil {
		return false
	}
	minor, err := strconv.ParseUint(versionStringToks[1], 10, 64)
	if err != nil {
		return false
	}
	return major > 7 || (major >= 7 && minor >= 3)
}

func migrateSSHConfig() (err error) {
	if !localSSHSupportsIdentityAgent() {
		return
	}
	prefix, err := getPrefix()
	if err != nil {
		PrintErr(os.Stderr, err.Error())
		return err
	}
	sshConfigPath, _ := getSSHConfigAndBakPaths()

	sshConfigFile, err := os.OpenFile(sshConfigPath, os.O_RDONLY|os.O_CREATE, 0700)
	if err != nil {
		return
	}
	defer sshConfigFile.Close()
	currentConfigContents, err := ioutil.ReadAll(sshConfigFile)
	if err != nil {
		return
	}

	oldPKCS11Provider := fmt.Sprintf(OLD_PKCS11_PROVIDER_FORMAT, prefix)

	if !bytes.Contains(currentConfigContents, []byte(oldPKCS11Provider)) {
		return nil
	}

	newConfigContents := bytes.Replace(currentConfigContents, []byte(oldPKCS11Provider), []byte(NEW_IDENTITY_AGENT), -1)

	err = ioutil.WriteFile(sshConfigPath, newConfigContents, 0700)
	if err != nil {
		return
	}

	return nil
}

func cleanSSHConfig() (err error) {
	configBlock := []byte(getKrSSHConfigBlockOrFatal())
	sshDirPath := HomeDir() + "/.ssh"
	sshConfigPath := sshDirPath + "/config"
	sshConfigBackupPath := sshConfigPath + ".bak.kr.uninstall"

	sshConfigFile, err := os.Open(sshConfigPath)
	if err != nil {
		return
	}
	defer sshConfigFile.Close()
	currentConfigContents, err := ioutil.ReadAll(sshConfigFile)
	if err != nil {
		return
	}
	if len(currentConfigContents) > 0 {
		err = ioutil.WriteFile(sshConfigBackupPath, currentConfigContents, 0700)
		if err != nil {
			return
		}
	}

	newConfigContents := bytes.Replace(currentConfigContents, configBlock, []byte{}, -1)
	err = ioutil.WriteFile(sshConfigPath, newConfigContents, 0700)
	if err != nil {
		return
	}
	return
}
