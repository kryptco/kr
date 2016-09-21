package krssh

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/GoKillers/libsodium-go/cryptobox"
	"github.com/satori/go.uuid"
)

const SQS_BASE_QUEUE_URL = "https://sqs.us-east-1.amazonaws.com/911777333295/"

type PairingSecret struct {
	SymmetricSecretKey   *[]byte `json:"-"`
	WorkstationPublicKey []byte  `json:"pk"`
	workstationSecretKey []byte
	WorkstationName      string `json:"n"`
}

func (ps PairingSecret) DeriveBluetoothServiceUUID() (btUUID uuid.UUID, err error) {
	keyDigest := sha256.Sum256(ps.WorkstationPublicKey)
	return uuid.FromBytes(keyDigest[0:16])
}

func (ps PairingSecret) SQSSendQueueURL() string {
	return SQS_BASE_QUEUE_URL + ps.SQSBaseQueueName()
}
func (ps PairingSecret) SQSRecvQueueURL() string {
	return SQS_BASE_QUEUE_URL + ps.SQSRecvQueueName()
}
func (ps PairingSecret) SQSSendQueueName() string {
	return ps.SQSBaseQueueName()
}
func (ps PairingSecret) SQSRecvQueueName() string {
	return ps.SQSBaseQueueName() + "-responder"
}

func (ps PairingSecret) SQSBaseQueueName() string {
	pkDigest := sha256.Sum256(ps.WorkstationPublicKey)
	return base64.URLEncoding.EncodeToString(pkDigest[0:16])
}

func GeneratePairingSecret() (ps PairingSecret, err error) {
	ret := 0
	ps.workstationSecretKey, ps.WorkstationPublicKey, ret = cryptobox.CryptoBoxKeyPair()
	if ret != 0 {
		err = fmt.Errorf("nonzero CryptoBoxKeyPair exit status: %d", ret)
		return
	}
	hostname, _ := os.Hostname()
	ps.WorkstationName = os.Getenv("USER") + "@" + hostname
	return
}

func (ps PairingSecret) CreateQueues() (err error) {
	_, err = CreateQueue(ps.SQSSendQueueName())
	if err != nil {
		return
	}
	_, err = CreateQueue(ps.SQSRecvQueueName())
	if err != nil {
		return
	}
	return
}

func GeneratePairingSecretAndCreateQueues() (ps PairingSecret, err error) {
	ps, err = GeneratePairingSecret()
	if err != nil {
		return
	}
	err = ps.CreateQueues()
	if err != nil {
		return
	}
	return
}

func (ps PairingSecret) EncryptMessage(message []byte) (ciphertext []byte, err error) {
	if ps.SymmetricSecretKey == nil {
		err = fmt.Errorf("SymmetricSecretKey not set")
		return
	}
	key, err := SymmetricSecretKeyFromBytes(*ps.SymmetricSecretKey)
	if err != nil {
		return
	}
	ciphertext, err = Seal(message, *key)
	if err != nil {
		return
	}
	return
}

func (ps PairingSecret) DecryptMessage(ciphertext []byte) (message []byte, err error) {
	if ps.SymmetricSecretKey == nil {
		err = fmt.Errorf("SymmetricSecretKey not set")
		return
	}
	key, err := SymmetricSecretKeyFromBytes(*ps.SymmetricSecretKey)
	if err != nil {
		return
	}
	message, err = Open(ciphertext, *key)
	if err != nil {
		return
	}
	return
}

func (ps PairingSecret) SendMessage(message []byte) (err error) {
	ctxt, err := ps.EncryptMessage(message)
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
	if ps.SymmetricSecretKey == nil {
		err = fmt.Errorf("SymmetricSecretKey not set")
		return
	}
	key, err := SymmetricSecretKeyFromBytes(*ps.SymmetricSecretKey)
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
	}
	log.Printf("received %d messages", len(messages))
	return
}
