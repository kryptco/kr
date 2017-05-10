package krd

import "golang.org/x/crypto/ssh"

//	from https://github.com/golang/crypto/blob/master/ssh/common.go#L243-L264
type signaturePayload struct {
	Session []byte
	Type    byte
	User    string
	Service string
	Method  string
	Sign    bool
	Algo    []byte
	PubKey  []byte
}

type signaturePayloadWithoutPubkey struct {
	Session []byte
	Type    byte
	User    string
	Service string
	Method  string
	Sign    bool
	Algo    []byte
}

func (s signaturePayload) stripPubkey() signaturePayloadWithoutPubkey {
	return signaturePayloadWithoutPubkey{
		Session: s.Session,
		Type:    s.Type,
		User:    s.User,
		Service: s.Service,
		Method:  s.Method,
		Sign:    s.Sign,
		Algo:    s.Algo,
	}
}

func stripPubkeyFromSignaturePayload(data []byte) (stripped []byte, err error) {
	signedDataFormat := signaturePayload{}
	err = ssh.Unmarshal(data, &signedDataFormat)
	if err != nil {
		return
	}
	stripped = ssh.Marshal(signedDataFormat.stripPubkey())
	return
}

func parseSessionAndAlgoFromSignaturePayload(data []byte) (session []byte, algo string, err error) {
	signedDataFormat := signaturePayload{}
	err = ssh.Unmarshal(data, &signedDataFormat)
	if err != nil {
		return
	}

	session = signedDataFormat.Session
	algo = string(signedDataFormat.Algo)

	return
}
