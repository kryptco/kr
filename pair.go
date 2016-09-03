package krssh

import (
	"encoding/base64"
)

const SQS_BASE_QUEUE_URL = "https://sqs.us-east-1.amazonaws.com/911777333295/"

type PairingSecret struct {
	SQSBaseQueueName   string `json:"q"`
	SymmetricSecretKey []byte `json:"k"`
}

func (ps PairingSecret) SQSSendQueueURL() string {
	return SQS_BASE_QUEUE_URL + ps.SQSBaseQueueName
}
func (ps PairingSecret) SQSRecvQueueURL() string {
	return SQS_BASE_QUEUE_URL + ps.SQSBaseQueueName + "-recv"
}

func GeneratePairingSecret() (ps PairingSecret, err error) {
	symmetricSecretKey, err := GenSymmetricSecretKey()
	if err != nil {
		return
	}
	ps.SymmetricSecretKey = symmetricSecretKey.Bytes

	ps.SQSBaseQueueName, err = Rand128Base62()
	if err != nil {
		return
	}

	_, err = CreateSendAndReceiveQueues(ps.SQSBaseQueueName)
	if err != nil {
		return
	}

	return
}

func (ps PairingSecret) SendMessage(message []byte) (err error) {
	key, err := SymmetricSecretKeyFromBytes(ps.SymmetricSecretKey)
	if err != nil {
		return
	}
	ctxt, err := Seal(message, *key)
	if err != nil {
		return
	}
	ctxtString := base64.StdEncoding.EncodeToString(ctxt)

	err = SendToQueue(ps.SQSSendQueueURL(), ctxtString)
	if err != nil {
		return
	}
	return
}

func (ps PairingSecret) ReceiveMessages() (messages [][]byte, err error) {
	key, err := SymmetricSecretKeyFromBytes(ps.SymmetricSecretKey)
	if err != nil {
		return
	}
	ctxtStrings, err := ReceiveAndDeleteFromQueue(ps.SQSRecvQueueURL())
	if err != nil {
		return
	}

	for _, ctxtString := range ctxtStrings {
		ctxt, err := base64.StdEncoding.DecodeString(ctxtString)
		if err != nil {
			continue
		}

		message, err := Open(ctxt, *key)
		if err != nil {
			continue
		}
		messages = append(messages, message)
	}
	return
}
