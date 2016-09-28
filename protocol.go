package kr

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	"golang.org/x/crypto/ssh"
)

type Request struct {
	RequestID   string       `json:"request_id"`
	UnixSeconds int64        `json:"unix_seconds"`
	SignRequest *SignRequest `json:"sign_request"`
	ListRequest *ListRequest `json:"list_request"`
	MeRequest   *MeRequest   `json:"me_request"`
}

func NewRequest() (request Request, err error) {
	id, err := Rand128Base62()
	if err != nil {
		return
	}
	request = Request{
		RequestID:   id,
		UnixSeconds: time.Now().Unix(),
	}
	return
}

type Response struct {
	RequestID      string        `json:"request_id"`
	SignResponse   *SignResponse `json:"sign_response"`
	ListResponse   *ListResponse `json:"list_response"`
	MeResponse     *MeResponse   `json:"me_response"`
	SNSEndpointARN *string       `json:"sns_endpoint_arn"`
}

type SignRequest struct {
	//	N.B. []byte marshals to base64 encoding in JSON
	Digest []byte `json:"digest"`
	//	SHA256 hash of public key DER
	PublicKeyFingerprint []byte  `json:"public_key_fingerprint"`
	Command              *string `json:"command"`
}

type SignResponse struct {
	Signature *[]byte `json:"signature"`
	Error     *string `json:"error"`
}

type ListRequest struct {
	EmailFilter *string `json:"email_filter"`
}

type ListResponse struct {
	Profiles []Profile `json:"profiles"`
}

type Profile struct {
	SSHWirePublicKey []byte `json:"rsa_public_key_wire"`
	Email            string `json:"email"`
}

func (p Profile) AuthorizedKeyString() string {
	return "ssh-rsa " + base64.StdEncoding.EncodeToString(p.SSHWirePublicKey) + " " + p.Email
}

func (p Profile) SSHPublicKey() (pk ssh.PublicKey, err error) {
	return ssh.ParsePublicKey(p.SSHWirePublicKey)
}

func (p Profile) RSAPublicKey() (pk *rsa.PublicKey, err error) {
	return SSHWireRSAPublicKeyToRSAPublicKey(p.SSHWirePublicKey)
}

func (p Profile) PublicKeyFingerprint() []byte {
	digest := sha256.Sum256(p.SSHWirePublicKey)
	return digest[:]
}

type MeRequest struct{}

type MeResponse struct {
	Me Profile `json:"me"`
}

func (request Request) HTTPRequest() (httpRequest *http.Request, err error) {
	requestJson, err := json.Marshal(request)
	if err != nil {
		return
	}
	httpRequest, err = http.NewRequest("PUT", "/enclave", bytes.NewReader(requestJson))
	if err != nil {
		return
	}
	return
}

func SSHWireRSAPublicKeyToRSAPublicKey(wire []byte) (pk *rsa.PublicKey, err error) {
	//	parse RSA SSH wire format
	//  https://github.com/golang/crypto/blob/077efaa604f994162e3307fafe5954640763fc08/ssh/keys.go#L302
	var w struct {
		//	assume type RSA
		Type string
		E    *big.Int
		N    *big.Int
	}
	if err = ssh.Unmarshal(wire, &w); err != nil {
		return
	}
	pk = &rsa.PublicKey{
		N: w.N,
		E: int(w.E.Int64()),
	}
	return
}
