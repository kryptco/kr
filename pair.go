package krssh

import (
	"encoding/base64"
	"log"
	"os"
)

const SQS_BASE_QUEUE_URL = "https://sqs.us-east-1.amazonaws.com/911777333295/"

type PairingSecret struct {
	SQSBaseQueueName   string `json:"q"`
	SymmetricSecretKey []byte `json:"k"`
	WorkstationName    string `json:"n"`
}

func (ps PairingSecret) SQSSendQueueURL() string {
	return SQS_BASE_QUEUE_URL + ps.SQSBaseQueueName
}
func (ps PairingSecret) SQSRecvQueueURL() string {
	return SQS_BASE_QUEUE_URL + ps.SQSRecvQueueName()
}
func (ps PairingSecret) SQSSendQueueName() string {
	return ps.SQSBaseQueueName
}
func (ps PairingSecret) SQSRecvQueueName() string {
	return ps.SQSBaseQueueName + "-responder"
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

	_, err = CreateQueue(ps.SQSSendQueueName())
	if err != nil {
		return
	}
	_, err = CreateQueue(ps.SQSRecvQueueName())
	if err != nil {
		return
	}

	hostname, _ := os.Hostname()
	ps.WorkstationName = os.Getenv("USER") + "@" + hostname

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
			log.Println("base64 ciphertext decoding error")
			continue
		}

		message, err := Open(ctxt, *key)
		if err != nil {
			log.Println("open cipertext error")
			continue
		}
		messages = append(messages, message)
		log.Println("received message: ", string(message))
	}
	log.Printf("received %d messages", len(messages))
	return
}
