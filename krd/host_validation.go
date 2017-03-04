package krd

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

//	from https://github.com/golang/crypto/blob/master/ssh/common.go#L243-L264
type signaturePayload struct {
	Session []byte
	Type    byte
	User    string
	Service string
	Method  string
	Sign    bool
	Algo    []byte
	PubKey  []byte
}

type signaturePayloadWithoutPubkey struct {
	Session []byte
	Type    byte
	User    string
	Service string
	Method  string
	Sign    bool
	Algo    []byte
}

func (s signaturePayload) stripPubkey() signaturePayloadWithoutPubkey {
	return signaturePayloadWithoutPubkey{
		Session: s.Session,
		Type:    s.Type,
		User:    s.User,
		Service: s.Service,
		Method:  s.Method,
		Sign:    s.Sign,
		Algo:    s.Algo,
	}
}

func stripPubkeyFromSignaturePayload(data []byte) (stripped []byte, err error) {
	signedDataFormat := signaturePayload{}
	err = ssh.Unmarshal(data, &signedDataFormat)
	if err != nil {
		return
	}
	stripped = ssh.Marshal(signedDataFormat.stripPubkey())
	return
}

func parseSessionFromSignaturePayload(data []byte) (session []byte, err error) {
	signedDataFormat := signaturePayload{}
	err = ssh.Unmarshal(data, &signedDataFormat)
	if err != nil {
		return
	}
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
