package kr

const PAIRING_FILENAME = "pairing.json"

type persistedPairing struct {
	SymmetricSecretKey   *[]byte
	WorkstationPublicKey []byte
	WorkstationSecretKey []byte
	WorkstationName      string
	SNSEndpointARN       *string
	ApprovedUntil        *int64
	TrackingID           *string
}

func pairingToPersisted(ps *PairingSecret) persistedPairing {
	return persistedPairing{
		SymmetricSecretKey:   ps.SymmetricSecretKey,
		WorkstationPublicKey: ps.WorkstationPublicKey,
		WorkstationSecretKey: ps.workstationSecretKey,
		WorkstationName:      ps.WorkstationName,
		SNSEndpointARN:       ps.snsEndpointARN,
		ApprovedUntil:        ps.ApprovedUntil,
		TrackingID:           ps.trackingID,
	}
}

func pairingFromPersisted(pp *persistedPairing) *PairingSecret {
	return &PairingSecret{
		SymmetricSecretKey:   pp.SymmetricSecretKey,
		WorkstationPublicKey: pp.WorkstationPublicKey,
		workstationSecretKey: pp.WorkstationSecretKey,
		WorkstationName:      pp.WorkstationName,
		snsEndpointARN:       pp.SNSEndpointARN,
		ApprovedUntil:        pp.ApprovedUntil,
		trackingID:           pp.TrackingID,
	}
}
