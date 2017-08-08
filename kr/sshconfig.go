package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/kryptco/kr"
)

const SSH_CONFIG_FORMAT = `# Added by Kryptonite
Host *
	PKCS11Provider %s/lib/kr-pkcs11.so
	ProxyCommand %s/bin/krssh %%h %%p
	IdentityFile ~/.ssh/id_kryptonite
	IdentityFile ~/.ssh/id_ed25519
	IdentityFile ~/.ssh/id_rsa
	IdentityFile ~/.ssh/id_ecdsa
	IdentityFile ~/.ssh/id_dsa`

const KR_SKIP_SSH_CONFIG = "KR_SKIP_SSH_CONFIG"

func getKryptoniteSSHConfigBlock() string {
	prefix := getPrefix()
	var sshConfigWithPrefix = fmt.Sprintf(SSH_CONFIG_FORMAT, prefix, prefix)
	return sshConfigWithPrefix
}

func promptEditSSHConfig() (err error) {
	if os.Getenv(KR_SKIP_SSH_CONFIG) != "" {
		return
	}
	configBlock := []byte(getKryptoniteSSHConfigBlock())
	sshDirPath := os.Getenv("HOME") + "/.ssh"
	_ = os.MkdirAll(sshDirPath, 0700)
	sshConfigPath := sshDirPath + "/config"
	sshConfigBackupPath := sshConfigPath + ".bak.kr"

	sshConfigFile, err := os.OpenFile(sshConfigPath, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return
	}
	currentConfigContents, err := ioutil.ReadAll(sshConfigFile)
	if err != nil {
		return
	}

	if bytes.Contains(currentConfigContents, configBlock) {
		return
	}

	if !confirm(os.Stderr, kr.Yellow("Kryptonite ▶ SSH must be configured to use Kryptonite. Automatically configure SSH?")) {
		os.Stderr.WriteString(kr.Yellow("Please add the following to ~/.ssh/config:") + "\n\n" + string(configBlock) + "\n\n" + "You can mute this message by adding \"export KR_SKIP_SSH_CONFIG=1\" to your ~/.bashrc file.\n\n")
		os.Stderr.WriteString("Press " + kr.Cyan("ENTER") + " to continue")
		os.Stdin.Read([]byte{0})
		return
	}
	if len(currentConfigContents) > 0 {
		err = ioutil.WriteFile(sshConfigBackupPath, currentConfigContents, 0700)
		if err != nil {
			return
		}
		PrintErr(os.Stderr, kr.Green("Kryptonite ▶ ~/.ssh/config backed up to ~/.ssh/config.bak.kr ✔"))
	}

	newConfigContents := bytes.Join([][]byte{currentConfigContents, configBlock}, []byte("\n"))
	err = ioutil.WriteFile(sshConfigPath, newConfigContents, 0700)
	if err != nil {
		return
	}
	PrintErr(os.Stderr, kr.Green("Kryptonite ▶ SSH configured ✔"))
	<-time.After(time.Second)
	return
}