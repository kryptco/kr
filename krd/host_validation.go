package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

func parseSessionFromSignaturePayload(data []byte) (session []byte, err error) {
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
	session = signedDataFormat.Session
	return
}

func hostForPublicKey(pk ssh.PublicKey) (hosts []string, err error) {
	marshaledPk := pk.Marshal()
	knownHostsBytes, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		return
	}
	var rest []byte
	rest = knownHostsBytes
	for {
		_, hosts, hostPubkey, _, knownHostsBytes, err := ssh.ParseKnownHosts(rest)
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
	}
	return
}
