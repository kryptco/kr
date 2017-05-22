package kr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/blang/semver"
	"golang.org/x/crypto/openpgp/armor"
)

//	Previous enclave versions assume SHA1 for all RSA keys regardless of the PubKeyAlgorithm specified in the signature payload
var ENCLAVE_VERSION_SUPPORTS_RSA_SHA2_256_512 = semver.MustParse("2.1.0")

type Request struct {
	RequestID      string          `json:"request_id"`
	UnixSeconds    int64           `json:"unix_seconds"`
	Version        semver.Version  `json:"v"`
	SendACK        bool            `json:"a"`
	SignRequest    *SignRequest    `json:"sign_request"`
	GitSignRequest *GitSignRequest `json:"git_sign_request"`
	MeRequest      *MeRequest      `json:"me_request"`
	UnpairRequest  *UnpairRequest  `json:"unpair_request"`
}

func NewRequest() (request Request, err error) {
	id, err := Rand128Base62()
	if err != nil {
		return
	}
	request = Request{
		RequestID:   id,
		UnixSeconds: time.Now().Unix(),
		Version:     CURRENT_VERSION,
		SendACK:     true,
	}
	return
}

type Response struct {
	RequestID       string           `json:"request_id"`
	Version         semver.Version   `json:"v"`
	SignResponse    *SignResponse    `json:"sign_response"`
	GitSignResponse *GitSignResponse `json:"git_sign_response"`
	MeResponse      *MeResponse      `json:"me_response"`
	UnpairResponse  *UnpairResponse  `json:"unpair_response"`
	AckResponse     *AckResponse     `json:"ack_response"`
	SNSEndpointARN  *string          `json:"sns_endpoint_arn"`
	ApprovedUntil   *int64           `json:"approved_until"`
	TrackingID      *string          `json:"tracking_id"`
}

type SignRequest struct {
	//	N.B. []byte marshals to base64 encoding in JSON
	Data []byte `json:"data"`
	//	SHA256 hash of SSH wire format
	PublicKeyFingerprint []byte    `json:"public_key_fingerprint"`
	Command              *string   `json:"command"`
	HostAuth             *HostAuth `json:"host_auth"`
}

type SignResponse struct {
	Signature *[]byte `json:"signature"`
	Error     *string `json:"error"`
}

type GitSignRequest struct {
	Commit CommitInfo `json:"commit"`
	UserId string     `json:"user_id"`
}

type GitSignResponse struct {
	Signature *[]byte `json:"signature"`
	Error     *string `json:"error"`
}

func (gsr GitSignResponse) AsciiArmorSignature() (s string, err error) {
	if gsr.Signature == nil {
		err = fmt.Errorf("no signature")
		return
	}
	output := &bytes.Buffer{}
	input, err := armor.Encode(output, "PGP SIGNATURE", map[string]string{"Comment": "Created With Kryptonite"})
	if err != nil {
		return
	}
	_, err = input.Write(*gsr.Signature)
	if err != nil {
		return
	}
	err = input.Close()
	if err != nil {
		return
	}
	s = string(output.Bytes())
	return
}

type CommitInfo struct {
	Tree      []byte  `json:"tree"`
	Parent    *[]byte `json:"parent"`
	Author    []byte  `json:"author"`
	Committer []byte  `json:"committer"`
	Message   []byte  `json:"message"`
}

type MeRequest struct {
	PGPUserId *string `json:"pgp_user_id"`
}

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

func (request Request) IsNoOp() bool {
	return request.SignRequest == nil && request.MeRequest == nil && request.UnpairRequest == nil
}

type UnpairRequest struct{}

type UnpairResponse struct{}

type AckResponse struct{}
