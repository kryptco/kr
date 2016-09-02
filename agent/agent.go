package main

import (
	"encoding/base64"
	"errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io/ioutil"
	"log"
	"net"
	"os"
)

type Agent struct{}

var signers []ssh.Signer
var paired bool

func (a *Agent) List() (keys []*agent.Key, err error) {
	log.Println("list")
	idrsaBytes, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_rsa.pub")
	if err != nil {
		log.Fatal(err)
	}
	idrsaPk, comment, _, _, err := ssh.ParseAuthorizedKey(idrsaBytes)
	if err != nil {
		log.Fatal(err)
	}

	keys = append(keys, &agent.Key{
		Format:  idrsaPk.Type(),
		Blob:    idrsaPk.Marshal(),
		Comment: comment,
	})

	for _, signer := range signers {
		log.Println(signer.PublicKey().Type() + " " +
			base64.StdEncoding.EncodeToString(signer.PublicKey().Marshal()))
		keys = append(keys, &agent.Key{
			Format: signer.PublicKey().Type(),
			Blob:   signer.PublicKey().Marshal(),
		})
	}

	return
}

func (a *Agent) Sign(key ssh.PublicKey, data []byte) (signature *ssh.Signature, err error) {
	log.Println("sign")
	log.Println(key)
	log.Println(string(data))
	log.Println(base64.StdEncoding.EncodeToString(data))
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
	pkDER, err := base64.StdEncoding.DecodeString("MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEHD0yLU4UBhXwUZg7LbN5qdrBerbw/WvcP88xc5csWZVoVFDIbZTr0fk1fruV6zOlzk98C9ojHcM0df5yfSd6VA==")
	if err != nil {
		log.Fatal(err)
	}
	pk, err := PKDERToProxiedKey(pkDER)
	if err != nil {
		log.Fatal(err)
	}
	pkSigner, err := ssh.NewSignerFromSigner(pk)
	if err != nil {
		log.Fatal(err)
	}

	signers = append(signers, pkSigner)

	krAgent := &Agent{}
	os.Remove("/tmp/krsshagent")
	l, err := net.Listen("unix", "/tmp/krsshagent")
	if err != nil {
		log.Fatal(err)
	}
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go agent.ServeAgent(krAgent, c)
	}
}
