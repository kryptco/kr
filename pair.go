package kr

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/GoKillers/libsodium-go/cryptobox"
	"github.com/satori/go.uuid"
)

const SQS_BASE_QUEUE_URL = "https://sqs.us-east-1.amazonaws.com/911777333295/"

var ErrWaitingForKey = fmt.Errorf("Pairing in progress, waiting for symmetric key")

//	TODO: Indicate whether bluetooth support enabled
type PairingSecret struct {
	SymmetricSecretKey   *[]byte `json:"-"`
	WorkstationPublicKey []byte  `json:"pk"`
	workstationSecretKey []byte
	WorkstationName      string `json:"n"`
	snsEndpointARN       *string
	ApprovedUntil        *int64 `json:"-"`
	trackingID           *string
	sync.Mutex
}

func (ps *PairingSecret) Equals(other *PairingSecret) bool {
	return bytes.Equal(ps.WorkstationPublicKey, other.WorkstationPublicKey)
}

func (ps *PairingSecret) DeriveUUID() (derivedUUID uuid.UUID, err error) {
	keyDigest := sha256.Sum256(ps.WorkstationPublicKey)
	return uuid.FromBytes(keyDigest[0:16])
}

func (ps *PairingSecret) SQSSendQueueURL() string {
	return SQS_BASE_QUEUE_URL + ps.SQSBaseQueueName()
}
func (ps *PairingSecret) SQSRecvQueueURL() string {
	return SQS_BASE_QUEUE_URL + ps.SQSRecvQueueName()
}
func (ps *PairingSecret) SQSSendQueueName() string {
	return ps.SQSBaseQueueName()
}
func (ps *PairingSecret) SQSRecvQueueName() string {
	return ps.SQSBaseQueueName() + "-responder"
}

func (ps *PairingSecret) SQSBaseQueueName() string {
	derivedUUID, err := ps.DeriveUUID()
	if err != nil {
		log.Error("error deriving UUID in PairingSecret:", err.Error())
		return ""
	}
	return strings.ToUpper(derivedUUID.String())
}

func GeneratePairingSecret() (ps *PairingSecret, err error) {
	ret := 0
	ps = new(PairingSecret)
	ps.workstationSecretKey, ps.WorkstationPublicKey, ret = cryptobox.CryptoBoxKeyPair()
	if ret != 0 {
		err = fmt.Errorf("nonzero CryptoBoxKeyPair exit status: %d", ret)
		return
	}
	hostname, _ := os.Hostname()
	ps.WorkstationName = os.Getenv("USER") + "@" + hostname
	return
}

func (ps *PairingSecret) CreateQueues() (err error) {
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

func GeneratePairingSecretAndCreateQueues() (ps *PairingSecret, err error) {
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

func (ps *PairingSecret) EncryptMessage(message []byte) (ciphertext []byte, err error) {
	ps.Lock()
	defer ps.Unlock()
	if ps.SymmetricSecretKey == nil {
		err = ErrWaitingForKey
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
	ciphertext = append([]byte{HEADER_CIPHERTEXT}, ciphertext...)
	return
}

func (ps *PairingSecret) UnwrapKeyIfPresent(ciphertext []byte) (remainingCiphertext *[]byte, didUnwrapKey bool, err error) {
	ps.Lock()
	defer ps.Unlock()
	if len(ciphertext) == 0 {
		err = fmt.Errorf("ciphertext empty")
		return
	}
	switch ciphertext[0] {
	case HEADER_CIPHERTEXT:
		ctxt := ciphertext[1:]
		remainingCiphertext = &ctxt
		return
	case HEADER_WRAPPED_KEY:
		if ps.SymmetricSecretKey != nil {
			return
		}
		wrappedKey := ciphertext[1:]
		key, unwrapErr := UnwrapKey(wrappedKey, ps.WorkstationPublicKey, ps.workstationSecretKey)
		if unwrapErr != nil {
			err = unwrapErr
			return
		}
		ps.SymmetricSecretKey = &key
		didUnwrapKey = true
		log.Notice("stored symmetric key")
		return
	}
	return
}

func (ps *PairingSecret) DecryptMessage(ciphertext []byte) (message *[]byte, err error) {
	ps.Lock()
	defer ps.Unlock()
	if ps.SymmetricSecretKey == nil {
		err = ErrWaitingForKey
		return
	}
	key, err := SymmetricSecretKeyFromBytes(*ps.SymmetricSecretKey)
	if err != nil {
		return
	}
	messageBytes, err := Open(ciphertext, *key)
	if err != nil {
		return
	}
	message = &messageBytes
	return
}

func (ps *PairingSecret) SetSNSEndpointARN(arn *string) {
	ps.Lock()
	defer ps.Unlock()
	ps.snsEndpointARN = arn
}

func (ps *PairingSecret) SetTrackingID(trackingID *string) {
	ps.Lock()
	defer ps.Unlock()
	ps.trackingID = trackingID
}

func (ps *PairingSecret) GetTrackingID() *string {
	ps.Lock()
	defer ps.Unlock()
	return ps.trackingID
}

func (ps *PairingSecret) PushAlert(alertText string, message []byte) (err error) {
	ctxt, err := ps.EncryptMessage(message)
	if err != nil {
		return
	}

	ctxtString := base64.StdEncoding.EncodeToString(ctxt)
	go func() {
		ps.Lock()
		arn := ps.snsEndpointARN
		ps.Unlock()
		if arn != nil {
			if pushErr := PushAlertToSNSEndpoint(alertText, ctxtString, *arn, ps.SQSSendQueueName()); pushErr != nil {
				log.Error("Push error:", pushErr)
			}
		}
	}()
	return
}

func (ps *PairingSecret) SendMessage(message []byte) (err error) {
	ctxt, err := ps.EncryptMessage(message)
	if err != nil {
		return
	}

	ctxtString := base64.StdEncoding.EncodeToString(ctxt)
	go func() {
		ps.Lock()
		arn := ps.snsEndpointARN
		ps.Unlock()
		if arn != nil {
			if pushErr := PushToSNSEndpoint(ctxtString, *arn, ps.SQSSendQueueName()); pushErr != nil {
				log.Error("Push error:", pushErr)
			}
		}
	}()

	err = SendToQueue(ps.SQSSendQueueURL(), ctxtString)
	if err != nil {
		return
	}
	return
}

func (ps *PairingSecret) ReadQueue() (ciphertexts [][]byte, err error) {
	ctxtStrings, err := ReceiveAndDeleteFromQueue(ps.SQSRecvQueueURL())
	if err != nil {
		return
	}

	for _, ctxtString := range ctxtStrings {
		ctxt, err := base64.StdEncoding.DecodeString(ctxtString)
		if err != nil {
			log.Error("base64 ciphertext decoding error")
			continue
		}
		ciphertexts = append(ciphertexts, ctxt)
	}
	return
}

func (ps *PairingSecret) IsPaired() bool {
	ps.Lock()
	defer ps.Unlock()
	return ps.SymmetricSecretKey != nil
}

func (ps *PairingSecret) RequiresApproval() bool {
	if ps.ApprovedUntil == nil {
		return true
	}
	return *ps.ApprovedUntil < time.Now().Unix()
}
