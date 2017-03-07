package krd

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/op/go-logging"
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

func hostForPublicKey(log *logging.Logger, pk ssh.PublicKey) (hosts []string, err error) {
	marshaledPk := pk.Marshal()
	knownHostsBytes, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		log.Error("error reading known hosts file: " + err.Error())
		return
	}
	var rest []byte
	rest = knownHostsBytes
	for {
		_, newHosts, hostPubkey, _, knownHostsBytes, err := ssh.ParseKnownHosts(rest)
		if hostPubkey != nil && bytes.Equal(hostPubkey.Marshal(), marshaledPk) {
			hosts = append(hosts, newHosts...)
		}
		rest = knownHostsBytes
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
	}

	//	prioritize first domain name over IP addresses
	var domainIdx *int
	for idx, host := range hosts {
		if strings.ContainsAny(strings.ToLower(host), "abcdefghijklmnopqrstuvwxyz") {
			domainIdx = &idx
			break
		}
	}
	if domainIdx != nil {
		hosts[0], hosts[*domainIdx] = hosts[*domainIdx], hosts[0]
	}
	return
}
