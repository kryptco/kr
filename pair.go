package kr

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"

	"github.com/satori/go.uuid"
)

var ErrWaitingForKey = fmt.Errorf("Pairing in progress, waiting for symmetric key")
var ErrWrappedKeyUnsupported = fmt.Errorf("WRAPPED_KEY unsupported")

//	TODO: Indicate whether bluetooth support enabled
type PairingSecret struct {
	EnclavePublicKey     *[]byte `json:"-"`
	WorkstationPublicKey []byte  `json:"pk"`
	workstationSecretKey []byte
	WorkstationName      string `json:"n"`
	snsEndpointARN       *string
	trackingID           *string
	Version              string `json:"v"`
	sync.Mutex
}

type PairingOptions struct {
	WorkstationName *string `json:"name"`
}

func (ps *PairingSecret) Equals(other *PairingSecret) bool {
	return bytes.Equal(ps.WorkstationPublicKey, other.WorkstationPublicKey)
}

func (ps *PairingSecret) DeriveUUID() (derivedUUID uuid.UUID, err error) {
	keyDigest := sha256.Sum256(ps.WorkstationPublicKey)
	return uuid.FromBytes(keyDigest[0:16])
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

func GeneratePairingSecret(workstationName *string) (ps *PairingSecret, err error) {
	ps = new(PairingSecret)
	ps.WorkstationPublicKey, ps.workstationSecretKey, err = GenKeyPair()
	if err != nil {
		return
	}
	if workstationName == nil {
		ps.WorkstationName = MachineName()
	} else {
		ps.WorkstationName = *workstationName
	}
	ps.Version = CURRENT_VERSION.String()
	return
}

func (ps *PairingSecret) EncryptMessage(message []byte) (ciphertext []byte, err error) {
	ps.Lock()
	defer ps.Unlock()
	if ps.EnclavePublicKey == nil {
		err = ErrWaitingForKey
		return
	}
	ciphertext, err = sodiumBox(message, *ps.EnclavePublicKey, ps.workstationSecretKey)
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
		err = ErrWrappedKeyUnsupported
		return
	case HEADER_WRAPPED_PUBLIC_KEY:
		if ps.EnclavePublicKey != nil {
			return
		}
		wrappedKey := ciphertext[1:]
		key, unwrapErr := UnwrapKey(wrappedKey, ps.WorkstationPublicKey, ps.workstationSecretKey)
		if unwrapErr != nil {
			err = unwrapErr
			return
		}
		ps.EnclavePublicKey = &key
		didUnwrapKey = true
		log.Notice("stored symmetric key")
		return
	default:
		err = fmt.Errorf("unknown header")
		return
	}
}

func (ps *PairingSecret) DecryptMessage(ciphertext []byte) (message *[]byte, err error) {
	ps.Lock()
	defer ps.Unlock()
	if ps.EnclavePublicKey == nil {
		err = ErrWaitingForKey
		return
	}
	messageBytes, err := sodiumBoxOpen(ciphertext, *ps.EnclavePublicKey, ps.workstationSecretKey)
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

func (ps *PairingSecret) GetSNSEndpointARN() (arn *string) {
	ps.Lock()
	defer ps.Unlock()
	return ps.snsEndpointARN
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

func (ps *PairingSecret) IsPaired() bool {
	ps.Lock()
	defer ps.Unlock()
	return ps.EnclavePublicKey != nil
}

func (ps *PairingSecret) DisplayName() string {
	return strings.TrimSuffix(ps.WorkstationName, ".local")
}
