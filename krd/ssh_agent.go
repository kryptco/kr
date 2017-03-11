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
	"time"

	"github.com/hashicorp/golang-lru"
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
	HostName  string
	PK        ssh.PublicKey
	Signature *ssh.Signature
}

type hostAuthCallback chan *kr.HostAuth

func getOriginalAgent() (originalAgent agent.Agent, err error) {
	originalAgentSock, err := kr.KrDirFile("original-agent.sock")
	if err != nil {
		return
	}
	conn, err := net.Dial("unix", originalAgentSock)
	if err != nil {
		return
	}
	return agent.NewClient(conn), nil
}

func (a *Agent) withOriginalAgent(do func(agent.Agent)) error {
	originalAgentSock, err := kr.KrDirFile("original-agent.sock")
	if err != nil {
		a.log.Error("error connecting to fallbackAgent: " + err.Error())
		return err
	}
	conn, err := net.Dial("unix", originalAgentSock)
	if err != nil {
		a.log.Error("error connecting to fallbackAgent: " + err.Error())
		return err
	}
	defer conn.Close()
	originalAgent := agent.NewClient(conn)
	do(originalAgent)
	return nil
}

type Agent struct {
	mutex    sync.Mutex
	client   EnclaveClientI
	notifier kr.Notifier

	recentSessionIDSignatures    []sessionIDSig
	hostAuthCallbacksBySessionID *lru.Cache

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

	a.withOriginalAgent(func(fallbackAgent agent.Agent) {
		fallbackKeys, err := fallbackAgent.List()
		if err == nil {
			keys = append(keys, fallbackKeys...)
		}
	})

	return
}

// Sign has the agent sign the data using a protocol 2 key as defined
// in [PROTOCOL.agent] section 2.6.2.
func (a *Agent) Sign(key ssh.PublicKey, data []byte) (sshSignature *ssh.Signature, err error) {
	keyFingerprint := sha256.Sum256(key.Marshal())

	a.withOriginalAgent(func(fallbackAgent agent.Agent) {
		fallbackKeys, fallbackErr := fallbackAgent.List()
		if fallbackErr == nil {
			for _, fallbackKey := range fallbackKeys {
				if bytes.Equal(fallbackKey.Marshal(), key.Marshal()) {
					sshSignature, err = fallbackAgent.Sign(key, data)
					return
				}
			}
		}
	})
	if sshSignature != nil {
		return
	}

	session, err := parseSessionFromSignaturePayload(data)
	var hostAuth *kr.HostAuth
	notifyPrefix := ""
	if err == nil {
		a.log.Notice("session: " + base64.StdEncoding.EncodeToString(session))
	} else {
		a.log.Error("error parsing session from signature payload: " + err.Error())
	}

	hostAuth = a.awaitHostAuthFor(base64.StdEncoding.EncodeToString(session))
	if hostAuth != nil {
		notifyPrefix = "[" + base64.StdEncoding.EncodeToString(hostAuth.Signature) + "]"
	} else {
		a.log.Warning("no hostname found for session " + base64.StdEncoding.EncodeToString(session))
	}

	switch key.Type() {
	case ssh.KeyAlgoRSA, ssh.KeyAlgoED25519:
		var dataWithoutPubkey []byte
		//	strip pubkey since it is redundant
		dataWithoutPubkey, err = stripPubkeyFromSignaturePayload(data)
		if err != nil {
			a.log.Error(err.Error())
			return
		}
		data = dataWithoutPubkey
	default:
		err = errors.New("unsupported key type: " + key.Type())
		a.log.Error(err.Error())
		return
	}

	a.notify(notifyPrefix + kr.Cyan("Kryptonite ▶ Requesting SSH authentication from phone"))

	signRequest := kr.SignRequest{
		PublicKeyFingerprint: keyFingerprint[:],
		Data:                 data,
		HostAuth:             hostAuth,
	}
	signResponse, err := a.client.RequestSignature(signRequest, func() {
		a.notify(notifyPrefix + kr.Yellow("Kryptonite ▶ Phone approval required. Respond using the Kryptonite app"))
	})
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
	if signResponse == nil {
		err = errors.New("no signature response")
		a.log.Error(err.Error())
		if notifyPrefix != "" {
			a.notify(notifyPrefix + "STOP")
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
		if notifyPrefix != "" {
			a.notify(notifyPrefix + "STOP")
		}
		return
	}
	if signResponse.Signature == nil {
		err = errors.New("no signature in response")
		a.log.Error(err.Error())
		if notifyPrefix != "" {
			a.notify(notifyPrefix + "STOP")
		}
		return
	}
	a.notify(notifyPrefix + kr.Green("Kryptonite ▶ Success. Request Allowed ✔"))
	signature := *signResponse.Signature
	sshSignature = &ssh.Signature{
		Format: key.Type(),
		Blob:   signature,
	}
	if notifyPrefix != "" {
		a.notify(notifyPrefix + "STOP")
	}
	return
}

// Add adds a private key to the agent.
func (a *Agent) Add(key agent.AddedKey) (err error) {
	connErr := a.withOriginalAgent(func(fallbackAgent agent.Agent) {
		err = fallbackAgent.Add(key)
	})
	if connErr != nil {
		err = connErr
	}
	return
}

// Remove removes all identities with the given public key.
func (a *Agent) Remove(key ssh.PublicKey) (err error) {
	connErr := a.withOriginalAgent(func(fallbackAgent agent.Agent) {
		err = fallbackAgent.Remove(key)
	})
	if connErr != nil {
		err = connErr
	}
	return
}

// RemoveAll removes all identities.
func (a *Agent) RemoveAll() (err error) {
	connErr := a.withOriginalAgent(func(fallbackAgent agent.Agent) {
		err = fallbackAgent.RemoveAll()
	})
	if connErr != nil {
		err = connErr
	}
	return
}

// Lock locks the agent. Sign and Remove will fail, and List will empty an empty list.
func (a *Agent) Lock(passphrase []byte) (err error) {
	connErr := a.withOriginalAgent(func(fallbackAgent agent.Agent) {
		err = fallbackAgent.Lock(passphrase)
	})
	if connErr != nil {
		err = connErr
	}
	return
}

// Unlock undoes the effect of Lock
func (a *Agent) Unlock(passphrase []byte) (err error) {
	connErr := a.withOriginalAgent(func(fallbackAgent agent.Agent) {
		err = fallbackAgent.Unlock(passphrase)
	})
	if connErr != nil {
		err = connErr
	}
	return
}

// Signers returns signers for all the known keys.
func (a *Agent) Signers() (signers []ssh.Signer, err error) {
	connErr := a.withOriginalAgent(func(fallbackAgent agent.Agent) {
		signers, err = fallbackAgent.Signers()
	})
	if connErr != nil {
		err = connErr
	}
	return
}

func (a *Agent) notify(body string) {
	if err := a.notifier.Notify(append([]byte(body), '\r', '\n')); err != nil {
		a.log.Error("error writing notification: " + err.Error())
	}
}

func (a *Agent) checkForHostAuth(session string) (hostAuth *kr.HostAuth) {
	a.mutex.Lock()
	sessionBytes, err := base64.StdEncoding.DecodeString(session)
	if err != nil {
		return nil
	}
	for _, sig := range a.recentSessionIDSignatures {
		hostAuth = a.tryHostAuth(&sig, sessionBytes)
		if hostAuth != nil {
			break
		}
	}
	a.mutex.Unlock()
	return
}

func (a *Agent) awaitHostAuthFor(session string) *kr.HostAuth {
	if hostAuth := a.checkForHostAuth(session); hostAuth != nil {
		return hostAuth
	}

	a.mutex.Lock()
	cb := make(chan *kr.HostAuth, 5)
	a.hostAuthCallbacksBySessionID.Add(session, cb)
	a.mutex.Unlock()

	select {
	case hostAuth := <-cb:
		return hostAuth
	case <-time.After(5 * time.Second):
	}
	return nil
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

	if len(hostAuth.HostNames) > 0 {
		sig.HostName = hostAuth.HostNames[0]
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.recentSessionIDSignatures = append([]sessionIDSig{sig}, a.recentSessionIDSignatures...)
	if len(a.recentSessionIDSignatures) > 50 {
		a.recentSessionIDSignatures = a.recentSessionIDSignatures[:50]
	}
	a.log.Notice("received hostAuth " + fmt.Sprintf("%+v", hostAuth))

	for _, session := range a.hostAuthCallbacksBySessionID.Keys() {
		sessionBytes, err := base64.StdEncoding.DecodeString(session.(string))
		if err != nil {
			continue
		}
		if hostAuth := a.tryHostAuth(&sig, sessionBytes); hostAuth != nil {
			if cb, found := a.hostAuthCallbacksBySessionID.Get(session); found {
				cb.(hostAuthCallback) <- hostAuth
				a.hostAuthCallbacksBySessionID.Remove(session)
			}
			break
		}
	}
}

func (a *Agent) tryHostAuth(sig *sessionIDSig, session []byte) *kr.HostAuth {
	if err := sig.PK.Verify(session, sig.Signature); err == nil {
		hostAuth := &kr.HostAuth{
			HostKey:   sig.PK.Marshal(),
			Signature: ssh.Marshal(sig.Signature),
			HostNames: []string{sig.HostName},
		}
		return hostAuth
	}
	return nil
}

func ServeKRAgent(enclaveClient EnclaveClientI, n kr.Notifier, agentListener net.Listener, hostAuthListener net.Listener, log *logging.Logger) (err error) {
	hostAuthCallbacksBySessionID, err := lru.New(128)
	if err != nil {
		return
	}
	krAgent := &Agent{
		sync.Mutex{},
		enclaveClient,
		n,
		[]sessionIDSig{},
		hostAuthCallbacksBySessionID,
		log,
	}
	go func() {
		for {
			conn, err := hostAuthListener.Accept()
			if err != nil {
				log.Error("hostAuth accept error: ", err.Error())
				continue
			}
			go func() {
				defer conn.Close()
				var hostAuth kr.HostAuth
				err = json.NewDecoder(conn).Decode(&hostAuth)
				if err != nil {
					log.Error("hostAuth decode error: ", err.Error())
					return
				}
				krAgent.onHostAuth(hostAuth)
			}()
		}
	}()

	for {
		conn, err := agentListener.Accept()
		if err != nil {
			log.Error("accept error: ", err.Error())
		}
		go func() {
			defer conn.Close()
			agent.ServeAgent(krAgent, conn)
		}()
	}
	return
}
