package krssh

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	PublicKeyDER []byte `json:"public_key_der"`
	Email        string `json:"email"`
}

func (p Profile) DisplayString() string {
	pkFingerprint := sha256.Sum256(p.PublicKeyDER)
	return base64.StdEncoding.EncodeToString(pkFingerprint[:]) + " <" + p.Email + ">"
}
func (p Profile) SSHWireString() (wireString string, err error) {
	rsaPk, err := ParseRsaAsn1(p.PublicKeyDER)
	if err != nil {
		return
	}
	sshPk, err := ssh.NewPublicKey(rsaPk)
	if err != nil {
		return
	}
	wireString = sshPk.Type() + " " + base64.StdEncoding.EncodeToString(sshPk.Marshal()) + " " + p.Email
	return
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

func ParseRsaAsn1(der []byte) (pk *rsa.PublicKey, err error) {
	pk = new(rsa.PublicKey)
	rest, err := asn1.Unmarshal(p.PublicKeyDER, pk)
	if err != nil {
		return
	}
}
