package kr

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/blang/semver"
)

//	Previous enclave versions assume SHA1 for all RSA keys regardless of the PubKeyAlgorithm specified in the signature payload
var ENCLAVE_VERSION_SUPPORTS_RSA_SHA2_256_512 = semver.MustParse("2.1.0")

type Request struct {
	RequestID     string         `json:"request_id"`
	UnixSeconds   int64          `json:"unix_seconds"`
	Version       semver.Version `json:"v"`
	SendACK       bool           `json:"a"`
	SignRequest   *SignRequest   `json:"sign_request"`
	MeRequest     *MeRequest     `json:"me_request"`
	UnpairRequest *UnpairRequest `json:"unpair_request"`
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
	RequestID      string          `json:"request_id"`
	Version        semver.Version  `json:"v"`
	SignResponse   *SignResponse   `json:"sign_response"`
	MeResponse     *MeResponse     `json:"me_response"`
	UnpairResponse *UnpairResponse `json:"unpair_response"`
	AckResponse    *AckResponse    `json:"ack_response"`
	SNSEndpointARN *string         `json:"sns_endpoint_arn"`
	ApprovedUntil  *int64          `json:"approved_until"`
	TrackingID     *string         `json:"tracking_id"`
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

func (request Request) IsNoOp() bool {
	return request.SignRequest == nil && request.MeRequest == nil && request.UnpairRequest == nil
}

type UnpairRequest struct{}

type UnpairResponse struct{}

type AckResponse struct{}
