package main

import (
	"bytes"
	"crypto"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"

	"github.com/fatih/color"
	"github.com/kryptco/kr"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func cyan(s string) string {
	cyan := color.New(color.FgHiCyan)
	cyan.EnableColor()
	return cyan.SprintFunc()(s)
}

func green(s string) string {
	green := color.New(color.FgHiGreen)
	green.EnableColor()
	return green.SprintFunc()(s)
}

func yellow(s string) string {
	yellow := color.New(color.FgHiYellow)
	yellow.EnableColor()
	return yellow.SprintFunc()(s)
}

func red(s string) string {
	red := color.New(color.FgHiRed)
	red.EnableColor()
	return red.SprintFunc()(s)
}

// 	from https://golang.org/src/crypto/rsa/pkcs1v15.go
var hashPrefixes = map[crypto.Hash][]byte{
	crypto.MD5:       {0x30, 0x20, 0x30, 0x0c, 0x06, 0x08, 0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x02, 0x05, 0x05, 0x00, 0x04, 0x10},
	crypto.SHA1:      {0x30, 0x21, 0x30, 0x09, 0x06, 0x05, 0x2b, 0x0e, 0x03, 0x02, 0x1a, 0x05, 0x00, 0x04, 0x14},
	crypto.SHA224:    {0x30, 0x2d, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x04, 0x05, 0x00, 0x04, 0x1c},
	crypto.SHA256:    {0x30, 0x31, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01, 0x05, 0x00, 0x04, 0x20},
	crypto.SHA384:    {0x30, 0x41, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x02, 0x05, 0x00, 0x04, 0x30},
	crypto.SHA512:    {0x30, 0x51, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x03, 0x05, 0x00, 0x04, 0x40},
	crypto.MD5SHA1:   {}, // A special TLS case which doesn't use an ASN1 prefix.
	crypto.RIPEMD160: {0x30, 0x20, 0x30, 0x08, 0x06, 0x06, 0x28, 0xcf, 0x06, 0x03, 0x00, 0x31, 0x04, 0x14},
}

type Agent struct {
	client   EnclaveClientI
	notifier kr.Notifier
	fallback agent.Agent
}

// List returns the identities known to the agent.
func (a Agent) List() (keys []*agent.Key, err error) {
	cachedProfile := a.client.GetCachedMe()
	keys = []*agent.Key{}

	if CheckIfUpdateAvailable() {
		a.notify(yellow("Kryptonite ▶ A new version of Kryptonite is available. Run \"kr upgrade\" to install it."))
	}

	if cachedProfile != nil {
		pk, parseErr := ssh.ParsePublicKey(cachedProfile.SSHWirePublicKey)
		if parseErr != nil {
			log.Error("list: parseKey error: " + parseErr.Error())
			err = parseErr
			return
		}
		keys = append(keys,
			&agent.Key{
				Format:  pk.Type(),
				Blob:    pk.Marshal(),
				Comment: cachedProfile.Email,
			})
	} else {
		a.notify(yellow("Kryptonite ▶ " + kr.ErrNotPaired.Error()))
	}
	fallbackKeys, err := a.fallback.List()
	if err == nil {
		keys = append(keys, fallbackKeys...)
	}
	return
}

// Sign has the agent sign the data using a protocol 2 key as defined
// in [PROTOCOL.agent] section 2.6.2.
func (a Agent) Sign(key ssh.PublicKey, data []byte) (sshSignature *ssh.Signature, err error) {
	keyFingerprint := sha256.Sum256(key.Marshal())

	keyringKeys, err := a.fallback.List()
	if err == nil {
		for _, keyringKey := range keyringKeys {
			if bytes.Equal(keyringKey.Marshal(), key.Marshal()) {
				return a.fallback.Sign(key, data)
			}
		}
	}

	var digest []byte
	switch key.Type() {
	case ssh.KeyAlgoRSA:
		sha1Digest := sha1.Sum(data)
		digest = append(hashPrefixes[crypto.SHA1], sha1Digest[:]...)
	case ssh.KeyAlgoED25519:
		digest = data
	default:
		err = errors.New("unsupported key type: " + key.Type())
		log.Error(err.Error())
		return
	}

	a.notify(cyan("Kryptonite ▶ Requesting SSH authentication from phone"))

	signRequest := kr.SignRequest{
		PublicKeyFingerprint: keyFingerprint[:],
		Digest:               digest,
		Command:              getLastCommand(),
	}
	signResponse, err := a.client.RequestSignature(signRequest)
	if err != nil {
		log.Error(err.Error())
		switch err {
		case ErrNotPaired:
			a.notify(yellow("Kryptonite ▶ " + kr.ErrNotPaired.Error()))
		case ErrTimeout:
			a.notify(red("Kryptonite ▶ " + kr.ErrTimedOut.Error()))
			a.notify(yellow("Kryptonite ▶ Falling back to local keys."))
		}
		return
	}
	log.Notice(fmt.Sprintf("sign response: %+v", signResponse))
	if signResponse.Error != nil {
		err = errors.New(*signResponse.Error)
		log.Error(err.Error())
		if *signResponse.Error == "rejected" {
			a.notify(red("Kryptonite ▶ " + kr.ErrRejected.Error()))
		} else {
			a.notify(red("Kryptonite ▶ " + kr.ErrSigning.Error()))
		}
		return
	}
	if signResponse == nil {
		err = errors.New("nil response")
		log.Error(err.Error())
		return
	}
	if signResponse.Signature == nil {
		err = errors.New("no signature in response")
		log.Error(err.Error())
		return
	}
	a.notify(green("Kryptonite ▶ Success. Request Allowed ✔"))
	signature := *signResponse.Signature
	sshSignature = &ssh.Signature{
		Format: key.Type(),
		Blob:   signature,
	}
	return
}

// Add adds a private key to the agent.
func (a Agent) Add(key agent.AddedKey) (err error) {
	return a.fallback.Add(key)
}

// Remove removes all identities with the given public key.
func (a Agent) Remove(key ssh.PublicKey) (err error) {
	return a.fallback.Remove(key)
}

// RemoveAll removes all identities.
func (a Agent) RemoveAll() (err error) {
	return a.fallback.RemoveAll()
}

// Lock locks the agent. Sign and Remove will fail, and List will empty an empty list.
func (a Agent) Lock(passphrase []byte) (err error) {
	return a.fallback.Lock(passphrase)
}

// Unlock undoes the effect of Lock
func (a Agent) Unlock(passphrase []byte) (err error) {
	return a.fallback.Unlock(passphrase)
}

// Signers returns signers for all the known keys.
func (a Agent) Signers() (signers []ssh.Signer, err error) {
	return a.fallback.Signers()
}

func (a Agent) notify(body string) {
	if err := a.notifier.Notify(append([]byte(body), '\n')); err != nil {
		log.Error("error writing notification: " + err.Error())
	}
}

func ServeKRAgent(enclaveClient EnclaveClientI, n kr.Notifier, l net.Listener) (err error) {
	for {
		conn, err := l.Accept()
		if err != nil {
			// handle error
			log.Error("accept error: ", err.Error())
		}
		go agent.ServeAgent(Agent{enclaveClient, n, agent.NewKeyring()}, conn)
	}
	return
}
