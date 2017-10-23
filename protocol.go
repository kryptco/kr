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
	SignRequest    *SignRequest    `json:"sign_request,omitempty"`
	GitSignRequest *GitSignRequest `json:"git_sign_request,omitempty"`
	MeRequest      *MeRequest      `json:"me_request,omitempty"`
	UnpairRequest  *UnpairRequest  `json:"unpair_request,omitempty"`
}

func NewRequest() (request Request, err error) {
	err = request.Prepare()
	return
}

func (r *Request) Prepare() (err error) {
	id, err := Rand128Base62()
	if err != nil {
		return
	}
	r.RequestID = id
	r.UnixSeconds = time.Now().Unix()
	r.Version = CURRENT_VERSION
	r.SendACK = true
	return
}

func (r Request) NotifyPrefix() string {
	return fmt.Sprintf("[%s]", r.RequestID)
}

type RequestParameters struct {
	AlertText string
	Timeout   TimeoutPhases
}

func (r Request) RequestParameters(timeouts Timeouts) RequestParameters {
	if r.SignRequest != nil {
		return RequestParameters{
			AlertText: "Incoming SSH request. Open Kryptonite to continue.",
			Timeout:   timeouts.Sign,
		}
	}
	if r.GitSignRequest != nil {
		return RequestParameters{
			AlertText: "Incoming Git request. Open Kryptonite to continue.",
			Timeout:   timeouts.Sign,
		}
	}
	return RequestParameters{
		AlertText: "Incoming Kryptonite request. ",
		Timeout:   timeouts.Sign,
	}
}

type Response struct {
	RequestID       string           `json:"request_id"`
	Version         semver.Version   `json:"v"`
	SignResponse    *SignResponse    `json:"sign_response,omitempty"`
	GitSignResponse *GitSignResponse `json:"git_sign_response,omitempty"`
	MeResponse      *MeResponse      `json:"me_response,omitempty"`
	UnpairResponse  *UnpairResponse  `json:"unpair_response,omitempty"`
	AckResponse     *AckResponse     `json:"ack_response,omitempty"`
	SNSEndpointARN  *string          `json:"sns_endpoint_arn,omitempty"`
	ApprovedUntil   *int64           `json:"approved_until,omitempty"`
	TrackingID      *string          `json:"tracking_id,omitempty"`
}

type SignRequest struct {
	//	N.B. []byte marshals to base64 encoding in JSON
	Data []byte `json:"data"`
	//	SHA256 hash of SSH wire format
	PublicKeyFingerprint []byte    `json:"public_key_fingerprint"`
	Command              *string   `json:"command,omitempty"`
	HostAuth             *HostAuth `json:"host_auth,omitempty"`
}

type SignResponse struct {
	Signature *[]byte `json:"signature,omitempty"`
	Error     *string `json:"error,omitempty"`
}

type GitSignRequest struct {
	Commit *CommitInfo `json:"commit,omitempty"`
	Tag    *TagInfo    `json:"tag,omitempty"`
	UserId string      `json:"user_id"`
}

type GitSignResponse struct {
	Signature *[]byte `json:"signature,omitempty"`
	Error     *string `json:"error,omitempty"`
}

func (gsr GitSignResponse) AsciiArmorSignature() (s string, err error) {
	if gsr.Signature == nil {
		err = fmt.Errorf("no signature")
		return
	}
	output := &bytes.Buffer{}
	input, err := armor.Encode(output, "PGP SIGNATURE", KRYPTONITE_ASCII_ARMOR_HEADERS)
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
	Tree         string    `json:"tree"`
	Parent       *string   `json:"parent,omitempty"`
	MergeParents *[]string `json:"merge_parents,omitempty"`
	Author       string    `json:"author"`
	Committer    string    `json:"committer"`
	Message      []byte    `json:"message"`
}

type TagInfo struct {
	Object  string `json:"object"`
	Type    string `json:"type"`
	Tag     string `json:"tag"`
	Tagger  string `json:"tagger"`
	Message []byte `json:"message"`
}

type MeRequest struct {
	PGPUserId *string `json:"pgp_user_id,omitempty"`
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

func (r Response) Error() *string {
	if r.GitSignResponse != nil {
		return r.GitSignResponse.Error
	}
	if r.SignResponse != nil {
		return r.SignResponse.Error
	}
	return nil
}
