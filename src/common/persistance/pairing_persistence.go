package persistance

import (
	. "krypt.co/kr/common/protocol"
)

const PAIRING_FILENAME = "pairing.json"
const ID_KRYPTON_FILENAME = "id_krypton.pub"

const PAIRING_TRANSFER_OLD_FILENAME = "pairing_transfer_old.json"
const PAIRING_TRANSFER_NEW_FILENAME = "pairing_transfer_new.json"

type persistedPairing struct {
	EnclavePublicKey     *[]byte
	WorkstationPublicKey []byte
	WorkstationSecretKey []byte
	WorkstationName      string
	SNSEndpointARN       *string
	TrackingID           *string
}

func pairingToPersisted(ps *PairingSecret) persistedPairing {
	return persistedPairing{
		EnclavePublicKey:     ps.EnclavePublicKey,
		WorkstationPublicKey: ps.WorkstationPublicKey,
		WorkstationSecretKey: ps.WorkstationSecretKey,
		WorkstationName:      ps.WorkstationName,
		SNSEndpointARN:       ps.SnsEndpointARN,
		TrackingID:           ps.TrackingID,
	}
}

func pairingFromPersisted(pp *persistedPairing) *PairingSecret {
	return &PairingSecret{
		EnclavePublicKey:     pp.EnclavePublicKey,
		WorkstationPublicKey: pp.WorkstationPublicKey,
		WorkstationSecretKey: pp.WorkstationSecretKey,
		WorkstationName:      pp.WorkstationName,
		SnsEndpointARN:       pp.SNSEndpointARN,
		TrackingID:           pp.TrackingID,
	}
}
