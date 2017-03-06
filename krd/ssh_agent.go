package krd

import (
	"bytes"
	"crypto"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/kryptco/kr"
	"github.com/op/go-logging"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

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

type sessionIDSig struct {
	PK        ssh.PublicKey
	Signature *ssh.Signature
}

type Agent struct {
	mutex    sync.Mutex
	client   EnclaveClientI
	notifier kr.Notifier
	fallback agent.Agent

	recentSessionIDSignatures []sessionIDSig

	log *logging.Logger
}

// List returns the identities known to the agent.
func (a *Agent) List() (keys []*agent.Key, err error) {
	cachedProfile := a.client.GetCachedMe()
	keys = []*agent.Key{}

	if cachedProfile != nil {
		pk, parseErr := ssh.ParsePublicKey(cachedProfile.SSHWirePublicKey)
		if parseErr != nil {
			a.log.Error("list: parseKey error: " + parseErr.Error())
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
		a.notify(kr.Yellow("Kryptonite ▶ " + kr.ErrNotPaired.Error()))
	}
	fallbackKeys, err := a.fallback.List()
	if err == nil {
		keys = append(keys, fallbackKeys...)
	}
	return
}

// Sign has the agent sign the data using a protocol 2 key as defined
// in [PROTOCOL.agent] section 2.6.2.
func (a *Agent) Sign(key ssh.PublicKey, data []byte) (sshSignature *ssh.Signature, err error) {
	keyFingerprint := sha256.Sum256(key.Marshal())

	keyringKeys, err := a.fallback.List()
	if err == nil {
		for _, keyringKey := range keyringKeys {
			if bytes.Equal(keyringKey.Marshal(), key.Marshal()) {
				return a.fallback.Sign(key, data)
			}
		}
	}

	session, err := parseSessionFromSignaturePayload(data)
	var hostAuth *kr.HostAuth
	notifyPrefix := ""
	if err == nil {
		a.log.Notice("session: " + base64.StdEncoding.EncodeToString(session))
		a.mutex.Lock()
		for _, sig := range a.recentSessionIDSignatures {
			if err := sig.PK.Verify(session, sig.Signature); err == nil {
				hostAuth = &kr.HostAuth{
					HostKey:   sig.PK.Marshal(),
					Signature: ssh.Marshal(sig.Signature),
				}
				notifyPrefix = "[" + base64.StdEncoding.EncodeToString(hostAuth.Signature) + "]"
				hostNames, err := hostForPublicKey(sig.PK)
				if err == nil {
					hostAuth.HostNames = hostNames
				} else {
					a.log.Error("error looking up hostname for public key: " + err.Error())
				}
				a.log.Notice("found remote signature for session, host auth: " + fmt.Sprintf("%+v", hostAuth))
				a.log.Notice("pubkey " + base64.StdEncoding.EncodeToString(hostAuth.HostKey))
				a.log.Notice("sig " + base64.StdEncoding.EncodeToString(hostAuth.Signature))
				break
			}
		}
		a.mutex.Unlock()
	} else {
		a.log.Error("error parsing session from signature payload: " + err.Error())
	}

	var digest []byte
	switch key.Type() {
	case ssh.KeyAlgoRSA, ssh.KeyAlgoED25519:
		var dataWithoutPubkey []byte
		//	strip pubkey since it is redundant
		dataWithoutPubkey, err = stripPubkeyFromSignaturePayload(data)
		if err != nil {
			a.log.Error(err.Error())
			return
		}
		digest = dataWithoutPubkey
	default:
		err = errors.New("unsupported key type: " + key.Type())
		a.log.Error(err.Error())
		return
	}

	a.notify(notifyPrefix + kr.Cyan("Kryptonite ▶ Requesting SSH authentication from phone"))

	signRequest := kr.SignRequest{
		PublicKeyFingerprint: keyFingerprint[:],
		Digest:               digest,
		Command:              getLastCommand(a.log),
		HostAuth:             hostAuth,
	}
	signResponse, err := a.client.RequestSignature(signRequest)
	if err != nil {
		a.log.Error(err.Error())
		switch err {
		case ErrNotPaired:
			a.notify(notifyPrefix + kr.Yellow("Kryptonite ▶ "+kr.ErrNotPaired.Error()))
		case ErrTimeout:
			a.notify(notifyPrefix + kr.Red("Kryptonite ▶ "+kr.ErrTimedOut.Error()))
			a.notify(notifyPrefix + kr.Yellow("Kryptonite ▶ Falling back to local keys."))
		}
		return
	}
	a.log.Notice(fmt.Sprintf("sign response: %+v", signResponse))
	if signResponse.Error != nil {
		err = errors.New(*signResponse.Error)
		a.log.Error(err.Error())
		if *signResponse.Error == "rejected" {
			a.notify(notifyPrefix + kr.Red("Kryptonite ▶ "+kr.ErrRejected.Error()))
		} else {
			a.notify(notifyPrefix + kr.Red("Kryptonite ▶ "+kr.ErrSigning.Error()))
		}
		a.notify(notifyPrefix + "STOP")
		return
	}
	if signResponse == nil {
		err = errors.New("nil response")
		a.log.Error(err.Error())
		a.notify(notifyPrefix + "STOP")
		return
	}
	if signResponse.Signature == nil {
		err = errors.New("no signature in response")
		a.log.Error(err.Error())
		a.notify(notifyPrefix + "STOP")
		return
	}
	a.notify(notifyPrefix + kr.Green("Kryptonite ▶ Success. Request Allowed ✔"))
	signature := *signResponse.Signature
	sshSignature = &ssh.Signature{
		Format: key.Type(),
		Blob:   signature,
	}
	a.notify(notifyPrefix + "STOP")
	return
}

// Add adds a private key to the agent.
func (a *Agent) Add(key agent.AddedKey) (err error) {
	return a.fallback.Add(key)
}

// Remove removes all identities with the given public key.
func (a *Agent) Remove(key ssh.PublicKey) (err error) {
	return a.fallback.Remove(key)
}

// RemoveAll removes all identities.
func (a *Agent) RemoveAll() (err error) {
	return a.fallback.RemoveAll()
}

// Lock locks the agent. Sign and Remove will fail, and List will empty an empty list.
func (a *Agent) Lock(passphrase []byte) (err error) {
	return a.fallback.Lock(passphrase)
}

// Unlock undoes the effect of Lock
func (a *Agent) Unlock(passphrase []byte) (err error) {
	return a.fallback.Unlock(passphrase)
}

// Signers returns signers for all the known keys.
func (a *Agent) Signers() (signers []ssh.Signer, err error) {
	return a.fallback.Signers()
}

func (a *Agent) notify(body string) {
	if err := a.notifier.Notify(append([]byte(body), '\r', '\n')); err != nil {
		a.log.Error("error writing notification: " + err.Error())
	}
}

func (a *Agent) onHostAuth(hostAuth kr.HostAuth) {
	sshPK, err := ssh.ParsePublicKey(hostAuth.HostKey)
	if err != nil {
		a.log.Error("error parsing hostAuth.HostKey: " + err.Error())
		return
	}

	var sshSig ssh.Signature
	err = ssh.Unmarshal(hostAuth.Signature, &sshSig)
	if err != nil {
		a.log.Error("error parsing hostAuth.Signature: " + err.Error())
		return
	}

	sig := sessionIDSig{
		PK:        sshPK,
		Signature: &sshSig,
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.recentSessionIDSignatures = append([]sessionIDSig{sig}, a.recentSessionIDSignatures...)
	if len(a.recentSessionIDSignatures) > 16 {
		a.recentSessionIDSignatures = a.recentSessionIDSignatures[:16]
	}
	a.log.Notice("received hostAuth " + fmt.Sprintf("%+v", hostAuth))
}

func ServeKRAgent(enclaveClient EnclaveClientI, n kr.Notifier, agentListener net.Listener, hostAuthListener net.Listener, log *logging.Logger) (err error) {
	krAgent := &Agent{sync.Mutex{}, enclaveClient, n, agent.NewKeyring(), []sessionIDSig{}, log}
	go func() {
		for {
			conn, err := hostAuthListener.Accept()
			if err != nil {
				log.Error("hostAuth accept error: ", err.Error())
				continue
			}
			defer conn.Close()
			var hostAuth kr.HostAuth
			err = json.NewDecoder(conn).Decode(&hostAuth)
			if err != nil {
				log.Error("hostAuth decode error: ", err.Error())
				continue
			}
			krAgent.onHostAuth(hostAuth)
		}
	}()

	for {
		conn, err := agentListener.Accept()
		if err != nil {
			log.Error("accept error: ", err.Error())
		}
		go agent.ServeAgent(krAgent, conn)
	}
	return
}
