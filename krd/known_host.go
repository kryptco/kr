package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"encoding/base64"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

func parseHostPubkeyFromSignaturePayload(data []byte) (pubkey ssh.PublicKey, err error) {
	//	from https://github.com/golang/crypto/blob/master/ssh/common.go#L243-L264
	signedDataFormat := struct {
		Session []byte
		Type    byte
		User    string
		Service string
		Method  string
		Sign    bool
		Algo    []byte
		PubKey  []byte
	}{}
	err = ssh.Unmarshal(data, &signedDataFormat)
	if err != nil {
		return
	}
	log.Notice(fmt.Sprintf("%+v", signedDataFormat))
	pubkey, err = ssh.ParsePublicKey(signedDataFormat.PubKey)
	return
}

func hostForPublicKey(pk ssh.PublicKey) (hosts []string, err error) {
	marshaledPk := pk.Marshal()
	log.Notice(fmt.Sprintf("marshaled key: %s", base64.StdEncoding.EncodeToString(marshaledPk)))
	knownHostsBytes, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		return
	}
	var rest []byte
	rest = knownHostsBytes
	for {
		_, hosts, hostPubkey, comment, knownHostsBytes, err := ssh.ParseKnownHosts(rest)
		if hostPubkey != nil && bytes.Equal(hostPubkey.Marshal(), marshaledPk) {
			return hosts, nil
		}
		rest = knownHostsBytes
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		log.Notice(fmt.Sprintf("marshaled host key: %s", base64.StdEncoding.EncodeToString(hostPubkey.Marshal())))
		log.Notice(fmt.Sprintf("%v %v %s", hosts, hostPubkey, comment))
	}
	return
}
