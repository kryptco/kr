package main

import (
	"crypto"
	"errors"
	"fmt"
)

const (
	HashFunctionSHA256 = "SHA256"
	HashFunctionSHA1   = "SHA1"
)

func HashToName(hash crypto.Hash) (name string, err error) {
	switch hash {
	case crypto.SHA1:
		name = HashFunctionSHA1
	case crypto.SHA256:
		name = HashFunctionSHA256
	default:
		err = errors.New(fmt.Sprintf("Unsupported hash function %d"))
	}
	return
}

type Request struct {
	RequestID   string       `json:"request_id"`
	SignRequest *SignRequest `json:"sign_request"`
	ListRequest *ListRequest `json:"list_request"`
	MeRequest   *MeRequest   `json:"me_request"`
}

type Response struct {
	RequestID    string        `json:"request_id"`
	SignResponse *SignResponse `json:"sign_response"`
	ListResponse *ListResponse `json:"list_response"`
	MeResponse   *MeResponse   `json:"me_response"`
}

type SignRequest struct {
	//	N.B. []byte marshals to base64 encoding in JSON
	Message []byte `json:"message"`
	//	SHA256 hash of public key DER
	PublicKeyFingerprint []byte `json:"public_key_fingerprint"`
	HashName             string `json:"hash_name"`
}

type SignResponse struct {
	Signature []byte `json:"signature"`
}

type ListRequest struct {
	EmailFilter *string `json:"email_filter"`
}

type ListResponse struct {
	Profiles []Profile `json:"profiles"`
}

type Profile struct {
	PublicKeyPEM string `json:"public_key_pem"`
	Email        string `json:"email"`
}

type MeRequest struct{}

type MeResponse struct {
	Me Profile `json:"me"`
}
