package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"log"
	"log/syslog"
	"sync"
	"syscall"

	"github.com/agrinman/krssh/agent/launch"
)

type Agent struct {
	enclaveClient EnclaveClientI
	signers       []ssh.Signer
	mutex         sync.Mutex
}

func (a *Agent) List() (keys []*agent.Key, err error) {
	log.Println("list")
	signer := a.enclaveClient.GetCachedMeSigner()
	if signer == nil {
		log.Println("no keys associated with this agent")
		DesktopNotify("Not paired, please run \"kr pair\" and scan the QR code with kryptonite.")
		return
	}

	log.Println(signer.PublicKey().Type() + " " +
		base64.StdEncoding.EncodeToString(signer.PublicKey().Marshal()))
	keys = append(keys, &agent.Key{
		Format: signer.PublicKey().Type(),
		Blob:   signer.PublicKey().Marshal(),
	})

	return
}

func (a *Agent) Sign(key ssh.PublicKey, data []byte) (signature *ssh.Signature, err error) {
	log.Println("sign")
	log.Println(key)
	log.Printf("%v\n", data)
	log.Printf("%q\n", string(data))
	log.Println(base64.StdEncoding.EncodeToString(data))
	me := a.enclaveClient.GetCachedMeSigner()
	if bytes.Equal(me.PublicKey().Marshal(), key.Marshal()) {
		return me.Sign(rand.Reader, data)
	}
	err = errors.New("not yet implemented")
	return
}

func (a *Agent) Add(key agent.AddedKey) (err error) {
	return
}

func (a *Agent) Remove(key ssh.PublicKey) (err error) {
	return
}

func (a *Agent) RemoveAll() (err error) {
	return
}

func (a *Agent) Lock(passphrase []byte) (err error) {
	return
}

func (a *Agent) Unlock(passphrase []byte) (err error) {
	return
}

func (a *Agent) Signers() (signers []ssh.Signer, err error) {
	log.Println("signers")
	return
}

func main() {
	syscall.Dup2(2, 1)
	logwriter, e := syslog.New(syslog.LOG_NOTICE, "krssh-agent")
	if e == nil {
		log.SetOutput(logwriter)
	}

	authSocket, ctlSocket, err := launch.OpenAuthAndCtlSockets()
	if err != nil {
		log.Fatal(err)
	}
	pkDER, err := base64.StdEncoding.DecodeString("MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEHD0yLU4UBhXwUZg7LbN5qdrBerbw/WvcP88xc5csWZVoVFDIbZTr0fk1fruV6zOlzk98C9ojHcM0df5yfSd6VA==")
	if err != nil {
		log.Fatal(err)
	}
	pk, err := PKDERToProxiedKey(nil, pkDER)
	if err != nil {
		log.Fatal(err)
	}
	pkSigner, err := ssh.NewSignerFromSigner(pk)
	if err != nil {
		log.Fatal(err)
	}

	signers := []ssh.Signer{pkSigner}

	controlServer := NewControlServer()
	go controlServer.HandleControlHTTP(ctlSocket)

	krAgent := &Agent{
		enclaveClient: controlServer.enclaveClient,
		signers:       signers,
	}
	for {
		c, err := authSocket.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go agent.ServeAgent(krAgent, c)
	}
}
