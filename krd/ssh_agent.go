package main

import (
	"crypto"
	//"crypto/dsa"
	//"crypto/ecdsa"
	"crypto/sha1"
	"crypto/sha256"
	//"encoding/asn1"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/kryptco/kr"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io"
	//"math/big"
	"net"
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

type Agent struct {
	client EnclaveClientI
}

// List returns the identities known to the agent.
func (a Agent) List() (keys []*agent.Key, err error) {
	cachedProfile := a.client.GetCachedMe()
	if cachedProfile != nil {
		pk, parseErr := ssh.ParsePublicKey(cachedProfile.SSHWirePublicKey)
		if parseErr != nil {
			log.Error("list: parseKey error: " + parseErr.Error())
			return
		}
		keys = []*agent.Key{
			&agent.Key{
				Format:  pk.Type(),
				Blob:    pk.Marshal(),
				Comment: cachedProfile.Email,
			},
		}
	}
	return
}

// Sign has the agent sign the data using a protocol 2 key as defined
// in [PROTOCOL.agent] section 2.6.2.
func (a Agent) Sign(key ssh.PublicKey, data []byte) (sshSignature *ssh.Signature, err error) {
	log.Notice("Sign: " + base64.StdEncoding.EncodeToString(data))
	log.Notice("Sign: " + string(data))
	keyFingerprint := sha256.Sum256(key.Marshal())

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

	signRequest := kr.SignRequest{
		PublicKeyFingerprint: keyFingerprint[:],
		Digest:               digest,
		Command:              getLastCommand(),
	}
	signResponse, err := a.client.RequestSignature(signRequest)
	signature := *signResponse.Signature
	//switch key.Type() {
	//case ssh.KeyAlgoRSA
	//ssh.KeyAlgoDSA
	//ssh.KeyAlgoECDSA256
	//ssh.KeyAlgoECDSA384
	//ssh.KeyAlgoECDSA521
	//ssh.KeyAlgoED25519
	//type asn1Signature struct {
	//R, S *big.Int
	//}
	//asn1Sig := new(asn1Signature)
	//_, err := asn1.Unmarshal(signature, asn1Sig)
	//if err != nil {
	//return nil, err
	//}

	//switch key.(type) {
	//case *ecdsa.PublicKey:
	//signature = ssh.Marshal(asn1Sig)

	//case *dsa.PublicKey:
	//signature = make([]byte, 40)
	//r := asn1Sig.R.Bytes()
	//s := asn1Sig.S.Bytes()
	//copy(signature[20-len(r):20], r)
	//copy(signature[40-len(s):40], s)
	//}
	//}
	log.Notice(fmt.Sprintf("sign response: %+v", signResponse))
	sshSignature = &ssh.Signature{
		Format: key.Type(),
		Blob:   signature,
	}
	return
}

// Add adds a private key to the agent.
func (a Agent) Add(key agent.AddedKey) (err error) {
	return
}

// Remove removes all identities with the given public key.
func (a Agent) Remove(key ssh.PublicKey) (err error) {
	return
}

// RemoveAll removes all identities.
func (a Agent) RemoveAll() (err error) {
	return
}

// Lock locks the agent. Sign and Remove will fail, and List will empty an empty list.
func (a Agent) Lock(passphrase []byte) (err error) {
	return
}

// Unlock undoes the effect of Lock
func (a Agent) Unlock(passphrase []byte) (err error) {
	return
}

// Signers returns signers for all the known keys.
func (a Agent) Signers() (signers []ssh.Signer, err error) {
	return
}

func ServeKRAgent(enclaveClient EnclaveClientI, l net.Listener) (err error) {
	for {
		conn, err := l.Accept()
		if err != nil {
			// handle error
			log.Error("accept error: ", err.Error())
		}
		go agent.ServeAgent(Agent{enclaveClient}, conn)
	}
	return
}

//	Implements crypto.Signer by requesting signatures from phone
type ProxiedKey struct {
	crypto.PublicKey
	publicKeyFingerprint []byte
	enclaveClient        EnclaveClientI
}

func (pk *ProxiedKey) Public() crypto.PublicKey {
	return pk.PublicKey
}

func (pk *ProxiedKey) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	command := getLastCommand()
	request := kr.SignRequest{
		PublicKeyFingerprint: pk.publicKeyFingerprint,
		Digest:               digest,
		Command:              command,
	}
	response, err := pk.enclaveClient.RequestSignature(request)
	if err != nil {
		log.Error("error requesting signature:", err)
		return
	}
	if response != nil {
		if response.Error != nil {
			err = errors.New("Enclave signature error: " + *response.Error)
			return
		}
		if response.Signature != nil {
			signature = *response.Signature
			return
		}
		err = errors.New("No enclave signature in response")
		return
	} else {
		err = errors.New("No response from enclave")
		return
	}

	err = errors.New("not yet implemented")
	return
}
