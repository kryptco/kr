package kr

const PAIRING_FILENAME = "pairing.json"
const ID_KRYPTONITE_FILENAME = "id_kryptonite.pub"

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
		WorkstationSecretKey: ps.workstationSecretKey,
		WorkstationName:      ps.WorkstationName,
		SNSEndpointARN:       ps.snsEndpointARN,
		TrackingID:           ps.trackingID,
	}
}

func pairingFromPersisted(pp *persistedPairing) *PairingSecret {
	return &PairingSecret{
		EnclavePublicKey:     pp.EnclavePublicKey,
		WorkstationPublicKey: pp.WorkstationPublicKey,
		workstationSecretKey: pp.WorkstationSecretKey,
		WorkstationName:      pp.WorkstationName,
		snsEndpointARN:       pp.SNSEndpointARN,
		trackingID:           pp.TrackingID,
	}
}
