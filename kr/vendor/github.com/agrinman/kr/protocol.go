package kr

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type Request struct {
	RequestID     string         `json:"request_id"`
	UnixSeconds   int64          `json:"unix_seconds"`
	SignRequest   *SignRequest   `json:"sign_request"`
	ListRequest   *ListRequest   `json:"list_request"`
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
	}
	return
}

type Response struct {
	RequestID      string          `json:"request_id"`
	SignResponse   *SignResponse   `json:"sign_response"`
	ListResponse   *ListResponse   `json:"list_response"`
	MeResponse     *MeResponse     `json:"me_response"`
	UnpairResponse *UnpairResponse `json:"unpair_response"`
	SNSEndpointARN *string         `json:"sns_endpoint_arn"`
	ApprovedUntil  *int64          `json:"approved_until"`
}

type SignRequest struct {
	//	N.B. []byte marshals to base64 encoding in JSON
	Digest []byte `json:"digest"`
	//	SHA256 hash of SSH wire format
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

type UnpairRequest struct{}

type UnpairResponse struct{}
