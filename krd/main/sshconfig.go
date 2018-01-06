package main

import (
	"bytes"
	"io/ioutil"
	"os"
)

func replaceKryptoniteWithKrypton(in []byte) []byte {
	commentReplaced := bytes.Replace(in, []byte("# Added by Kryptonite"), []byte("# Added by Krypton"), -1)
	identityFileReplaced := bytes.Replace(commentReplaced, []byte("~/.ssh/id_kryptonite"), []byte("~/.ssh/id_krypton"), -1)
	return identityFileReplaced
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
