package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/urfave/cli"

	"github.com/kryptco/kr"
)

const SSH_CONFIG_FORMAT = `# Added by Krypton
Host *
	PKCS11Provider %s/lib/kr-pkcs11.so
	ProxyCommand %s/bin/krssh %%h %%p
	IdentityFile ~/.ssh/id_krypton
	IdentityFile ~/.ssh/id_ed25519
	IdentityFile ~/.ssh/id_rsa
	IdentityFile ~/.ssh/id_ecdsa
	IdentityFile ~/.ssh/id_dsa`

func replaceKryptoniteWithKrypton(in []byte) []byte {
	commentReplaced := bytes.Replace(in, []byte("# Added by Kryptonite"), []byte("# Added by Krypton"), -1)
	identityFileReplaced := bytes.Replace(commentReplaced, []byte("~/.ssh/id_kryptonite"), []byte("~/.ssh/id_krypton"), -1)
	return identityFileReplaced
}

const KR_SKIP_SSH_CONFIG = "KR_SKIP_SSH_CONFIG"

func getKrSSHConfigBlock() string {
	prefix := getPrefix()
	var sshConfigWithPrefix = fmt.Sprintf(SSH_CONFIG_FORMAT, prefix, prefix)
	return sshConfigWithPrefix
}

func sshConfigCommand(c *cli.Context) (err error) {
	if c.Bool("print") {
		os.Stdout.WriteString(getKrSSHConfigBlock() + "\n")
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

func upgradeSSHConfig() (err error) {
	sshDirPath := os.Getenv("HOME") + "/.ssh"
	_ = os.MkdirAll(sshDirPath, 0700)
	sshConfigPath := sshDirPath + "/config"

	sshConfigFile, err := os.OpenFile(sshConfigPath, os.O_RDONLY|os.O_CREATE, 0700)
	if err != nil {
		return
	}
	defer sshConfigFile.Close()
	currentConfigContents, err := ioutil.ReadAll(sshConfigFile)
	if err != nil {
		return
	}

	//	update Kryptonite to Krypton without prompting
	updatedContents := replaceKryptoniteWithKrypton(currentConfigContents)
	if !bytes.Equal(updatedContents, currentConfigContents) {
		err = ioutil.WriteFile(sshConfigPath, updatedContents, 0700)
		if err != nil {
			return
		}
	}
	return
}

func editSSHConfig(prompt bool, forceAppend bool) (err error) {
	configBlock := []byte(getKrSSHConfigBlock())
	sshDirPath := os.Getenv("HOME") + "/.ssh"
	_ = os.MkdirAll(sshDirPath, 0700)
	sshConfigPath := sshDirPath + "/config"
	sshConfigBackupPath := sshConfigPath + ".bak.kr"

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
			PrintErr(os.Stderr, kr.Green("Krypton ▶ SSH already configured ✔"))
		}
		return
	}
	if bytes.Contains(currentConfigContents, []byte("krssh %h %p")) && !forceAppend {
		if prompt {
			PrintErr(os.Stderr, kr.Yellow("Krypton ▶ ~/.ssh/config already contains Krypton-related configuration. Please remove all Krypton-related lines from ~/.ssh/config or run with --force."))
		}
		return
	}

	if prompt {
		if !confirm(os.Stderr, kr.Yellow("Krypton ▶ SSH must be configured to use Krypton. Automatically configure SSH?")) {
			os.Stderr.WriteString(kr.Yellow("Please add the following to ~/.ssh/config:") + "\n\n" + string(configBlock) + "\n\n")
			os.Stderr.WriteString("Press " + kr.Cyan("ENTER") + " to continue")
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
			PrintErr(os.Stderr, kr.Green("Krypton ▶ ~/.ssh/config backed up to ~/.ssh/config.bak.kr ✔"))
		}
	}

	newConfigContents := bytes.Join([][]byte{currentConfigContents, configBlock}, []byte("\n\n"))
	err = ioutil.WriteFile(sshConfigPath, newConfigContents, 0700)
	if err != nil {
		return
	}
	if prompt {
		PrintErr(os.Stderr, kr.Green("Krypton ▶ SSH configured ✔"))
		<-time.After(time.Second)
	}
	return
}
