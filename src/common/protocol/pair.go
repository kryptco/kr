package protocol

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"krypt.co/kr/common/log"
	"krypt.co/kr/common/version"
	"strings"
	"sync"

	"github.com/satori/go.uuid"

	. "krypt.co/kr/common/util"
)

var ErrWaitingForKey = fmt.Errorf("pairing in progress, waiting for symmetric key")
var ErrWrappedKeyUnsupported = fmt.Errorf("WRAPPED_KEY unsupported")

//	TODO: Indicate whether bluetooth support enabled
type PairingSecret struct {
	EnclavePublicKey     *[]byte `json:"-"`
	WorkstationPublicKey []byte  `json:"pk"`
	WorkstationSecretKey []byte
	WorkstationName      string `json:"n"`
	SnsEndpointARN       *string
	TrackingID           *string
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
		log.Log.Error("error deriving UUID in PairingSecret:", err.Error())
		return ""
	}
	return strings.ToUpper(derivedUUID.String())
}

func GeneratePairingSecret(workstationName *string) (ps *PairingSecret, err error) {
	ps = new(PairingSecret)
	ps.WorkstationPublicKey, ps.WorkstationSecretKey, err = GenKeyPair()
	if err != nil {
		return
	}
	if workstationName == nil {
		ps.WorkstationName = MachineName()
	} else {
		ps.WorkstationName = *workstationName
	}
	ps.Version = version.CURRENT_VERSION.String()
	return
}

func (ps *PairingSecret) EncryptMessage(message []byte) (ciphertext []byte, err error) {
	ps.Lock()
	defer ps.Unlock()
	if ps.EnclavePublicKey == nil {
		err = ErrWaitingForKey
		return
	}
	ciphertext, err = sodiumBox(message, *ps.EnclavePublicKey, ps.WorkstationSecretKey)
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
		key, unwrapErr := UnwrapKey(wrappedKey, ps.WorkstationPublicKey, ps.WorkstationSecretKey)
		if unwrapErr != nil {
			err = unwrapErr
			return
		}
		ps.EnclavePublicKey = &key
		didUnwrapKey = true
		log.Log.Notice("stored symmetric key")
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
	messageBytes, err := sodiumBoxOpen(ciphertext, *ps.EnclavePublicKey, ps.WorkstationSecretKey)
	if err != nil {
		return
	}
	message = &messageBytes
	return
}

func (ps *PairingSecret) SetSNSEndpointARN(arn *string) {
	ps.Lock()
	defer ps.Unlock()
	ps.SnsEndpointARN = arn
}

func (ps *PairingSecret) GetSNSEndpointARN() (arn *string) {
	ps.Lock()
	defer ps.Unlock()
	return ps.SnsEndpointARN
}

func (ps *PairingSecret) SetTrackingID(trackingID *string) {
	ps.Lock()
	defer ps.Unlock()
	ps.TrackingID = trackingID
}

func (ps *PairingSecret) GetTrackingID() *string {
	ps.Lock()
	defer ps.Unlock()
	return ps.TrackingID
}

func (ps *PairingSecret) IsPaired() bool {
	ps.Lock()
	defer ps.Unlock()
	return ps.EnclavePublicKey != nil
}

func (ps *PairingSecret) DisplayName() string {
	return strings.TrimSuffix(ps.WorkstationName, ".local")
}
