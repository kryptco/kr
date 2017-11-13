package kr

const PAIRING_FILENAME = "pairing.json"

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
