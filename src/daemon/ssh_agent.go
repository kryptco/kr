package daemon

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"krypt.co/kr/common/socket"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/keybase/saltpack/encoding/basex"
	"github.com/op/go-logging"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	. "krypt.co/kr/common/protocol"
	. "krypt.co/kr/common/util"
	. "krypt.co/kr/daemon/enclave"
)

type sessionIDSig struct {
	HostName  string
	PK        ssh.PublicKey
	Signature *ssh.Signature
}

type hostAuthCallback chan *HostAuth

func (a *Agent) withOriginalAgent(do func(extendedAgent agent.ExtendedAgent)) error {
	conn, err := getOriginalAgentConn()
	if conn == nil {
		return nil
	}
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
	mutex  sync.Mutex
	client EnclaveClientI

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
		a.notify("", Yellow("Krypton ▶ "+ErrNotPaired.Error()))
	}

	a.withOriginalAgent(func(fallbackAgent agent.ExtendedAgent) {
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
	return a.SignWithFlags(key, data, 0)
}

func (a *Agent) SignWithFlags(key ssh.PublicKey, data []byte, flags agent.SignatureFlags) (sshSignature *ssh.Signature, err error) {
	keyFingerprint := sha256.Sum256(key.Marshal())

	a.withOriginalAgent(func(fallbackAgent agent.ExtendedAgent) {
		fallbackKeys, fallbackErr := fallbackAgent.List()
		if fallbackErr == nil {
			for _, fallbackKey := range fallbackKeys {
				if bytes.Equal(fallbackKey.Marshal(), key.Marshal()) {
					sshSignature, err = fallbackAgent.SignWithFlags(key, data, flags)
					return
				}
			}
		}
	})
	if sshSignature != nil {
		return
	}

	session, algo, err := parseSessionAndAlgoFromSignaturePayload(data)

	var hostAuth *HostAuth
	notifyPrefix := ""
	if err != nil {
		a.log.Error("error parsing session from signature payload: " + err.Error())
	}

	hostAuth = a.awaitHostAuthFor(base64.StdEncoding.EncodeToString(session))
	if hostAuth != nil {
		sigHash := sha256.Sum256(hostAuth.Signature)
		notifyPrefix = "[" + basex.Base62StdEncoding.EncodeToString(sigHash[:]) + "]"
	} else {
		a.log.Warning("no hostname found for session")
	}

	switch key.Type() {
	case ssh.KeyAlgoRSA, ssh.KeyAlgoED25519, ssh.KeyAlgoECDSA256:
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

	a.notify(notifyPrefix, notifyPrefix+Cyan("Krypton ▶ Requesting SSH authentication from phone"))

	signRequest := SignRequest{
		PublicKeyFingerprint: keyFingerprint[:],
		Data:                 data,
		HostAuth:             hostAuth,
	}
	signResponse, enclaveVersion, err := a.client.RequestSignature(signRequest, func() {
		a.notify(notifyPrefix, notifyPrefix+Yellow("Krypton ▶ Phone approval required. Respond using the Krypton app"))
	})
	if err != nil {
		a.log.Error(err.Error())
		switch err {
		case ErrNotPaired:
			a.notify(notifyPrefix, notifyPrefix+Yellow("Krypton ▶ "+ErrNotPaired.Error()))
		case ErrTimeout:
			a.notify(notifyPrefix, notifyPrefix+Red("Krypton ▶ "+ErrTimedOut.Error()))
			a.notify(notifyPrefix, notifyPrefix+Yellow("Krypton ▶ Falling back to local keys."))
		}
		return
	}
	if signResponse == nil {
		err = errors.New("no signature response")
		a.log.Error(err.Error())
		if notifyPrefix != "" {
			a.notify(notifyPrefix, notifyPrefix+"STOP")
		}
		return
	}
	a.log.Notice(fmt.Sprintf("sign response: %+v", signResponse))
	if signResponse.Error != nil {
		err = errors.New(*signResponse.Error)
		a.log.Error(err.Error())
		if *signResponse.Error == "rejected" {
			//	signal krssh to kill session, allow 1 second to do so
			a.notify(notifyPrefix, notifyPrefix+"REJECTED")
			<-time.After(1 * time.Second)
		} else if strings.HasPrefix(*signResponse.Error, "host public key mismatched") {
			//	signal krssh to kill session, allow 1 second to do so
			a.notify(notifyPrefix, notifyPrefix+"HOST_KEY_MISMATCH")
			<-time.After(1 * time.Second)
		} else {
			a.notify(notifyPrefix, notifyPrefix+Red("Krypton ▶ "+ErrSigning.Error()))
		}
		if notifyPrefix != "" {
			a.notify(notifyPrefix, notifyPrefix+"STOP")
		}
		return
	}
	if signResponse.Signature == nil {
		err = errors.New("no signature in response")
		a.log.Error(err.Error())
		if notifyPrefix != "" {
			a.notify(notifyPrefix, notifyPrefix+"STOP")
		}
		return
	}
	a.notify(notifyPrefix, notifyPrefix+Green("Krypton ▶ Success. Request Allowed ✔"))
	signature := *signResponse.Signature
	format := algo
	//	FIXME: sunset backwards compatibility
	if enclaveVersion.LT(ENCLAVE_VERSION_SUPPORTS_RSA_SHA2_256_512) {
		format = key.Type()
	}
	a.log.Notice("Using Public Key Signature Digest Algorithm: " + format)

	sshSignature = &ssh.Signature{
		Format: format,
		Blob:   signature,
	}
	if notifyPrefix != "" {
		a.notify(notifyPrefix, notifyPrefix+"STOP")
	}
	return
}

// Add adds a private key to the agent.
func (a *Agent) Add(key agent.AddedKey) (err error) {
	connErr := a.withOriginalAgent(func(fallbackAgent agent.ExtendedAgent) {
		err = fallbackAgent.Add(key)
	})
	if connErr != nil {
		err = connErr
	}
	return
}

// Remove removes all identities with the given public key.
func (a *Agent) Remove(key ssh.PublicKey) (err error) {
	connErr := a.withOriginalAgent(func(fallbackAgent agent.ExtendedAgent) {
		err = fallbackAgent.Remove(key)
	})
	if connErr != nil {
		err = connErr
	}
	return
}

// RemoveAll removes all identities.
func (a *Agent) RemoveAll() (err error) {
	connErr := a.withOriginalAgent(func(fallbackAgent agent.ExtendedAgent) {
		err = fallbackAgent.RemoveAll()
	})
	if connErr != nil {
		err = connErr
	}
	return
}

// Lock locks the agent. Sign and Remove will fail, and List will empty an empty list.
func (a *Agent) Lock(passphrase []byte) (err error) {
	connErr := a.withOriginalAgent(func(fallbackAgent agent.ExtendedAgent) {
		err = fallbackAgent.Lock(passphrase)
	})
	if connErr != nil {
		err = connErr
	}
	return
}

// Unlock undoes the effect of Lock
func (a *Agent) Unlock(passphrase []byte) (err error) {
	connErr := a.withOriginalAgent(func(fallbackAgent agent.ExtendedAgent) {
		err = fallbackAgent.Unlock(passphrase)
	})
	if connErr != nil {
		err = connErr
	}
	return
}

// Signers returns signers for all the known keys.
func (a *Agent) Signers() (signers []ssh.Signer, err error) {
	connErr := a.withOriginalAgent(func(fallbackAgent agent.ExtendedAgent) {
		signers, err = fallbackAgent.Signers()
	})
	if connErr != nil {
		err = connErr
	}
	return
}

func (a *Agent) Extension(extensionType string, contents []byte) (ret []byte, err error) {
	connErr := a.withOriginalAgent(func(fallbackAgent agent.ExtendedAgent) {
		ret, err = fallbackAgent.Extension(extensionType, contents)
	})
	if connErr != nil {
		err = agent.ErrExtensionUnsupported
	}
	return
}

func (a *Agent) notify(prefix, body string) {
	n, err := socket.OpenNotifier(prefix)
	if err != nil {
		a.log.Error("error writing notification: " + err.Error())
		return
	}
	defer n.Close()
	err = n.Notify(append([]byte(body), '\r', '\n'))
	if err != nil {
		a.log.Error("error writing notification: " + err.Error())
		return
	}
}

func (a *Agent) checkForHostAuth(session string) (hostAuth *HostAuth) {
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

func (a *Agent) awaitHostAuthFor(session string) *HostAuth {
	if hostAuth := a.checkForHostAuth(session); hostAuth != nil {
		return hostAuth
	}

	a.mutex.Lock()
	cb := make(chan *HostAuth, 5)
	a.hostAuthCallbacksBySessionID.Add(session, cb)
	a.mutex.Unlock()

	select {
	case hostAuth := <-cb:
		return hostAuth
	case <-time.After(time.Second):
	}
	return nil
}

func (a *Agent) onHostAuth(hostAuth HostAuth) {
	sshPK, err := ssh.ParsePublicKey(hostAuth.HostKey)
	if err != nil {
		a.log.Error("error parsing hostAuth.HostKey: " + err.Error())
		return
	}

	// Server public key may be a certificate, for now extract the inner public key
	if sshCert, ok := sshPK.(*ssh.Certificate); ok {
		sshPK = sshCert.Key
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
	a.log.Debug("received hostAuth " + fmt.Sprintf("%+v", hostAuth))

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

func (a *Agent) tryHostAuth(sig *sessionIDSig, session []byte) *HostAuth {
	if err := sig.PK.Verify(session, sig.Signature); err == nil {
		hostAuth := &HostAuth{
			HostKey:   sig.PK.Marshal(),
			Signature: ssh.Marshal(sig.Signature),
			HostNames: []string{sig.HostName},
		}
		return hostAuth
	}
	return nil
}

func ServeKRAgent(enclaveClient EnclaveClientI, agentListener net.Listener, hostAuthListener net.Listener, log *logging.Logger) (err error) {
	hostAuthCallbacksBySessionID, err := lru.New(128)
	if err != nil {
		return
	}
	krAgent := &Agent{
		sync.Mutex{},
		enclaveClient,
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
				var hostAuth HostAuth
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
			continue
		}
		go func() {
			RecoverToLog(func() {
				defer conn.Close()
				agent.ServeAgent(krAgent, conn)
			}, log)
		}()
	}
}
