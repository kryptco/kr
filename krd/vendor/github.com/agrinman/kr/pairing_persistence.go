package kr

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

const PAIRING_FILENAME = "pairing.json"

type persistedPairing struct {
	SymmetricSecretKey   *[]byte
	WorkstationPublicKey []byte
	WorkstationSecretKey []byte
	WorkstationName      string
	SNSEndpointARN       *string
	ApprovedUntil        *int64 `json:"approved_until"`
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

func LoadPairing() (pairingSecret *PairingSecret, err error) {
	path, err := KrDirFile(PAIRING_FILENAME)
	if err != nil {
		return
	}
	pairingJson, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	var pp persistedPairing
	err = json.Unmarshal(pairingJson, &pp)
	if err != nil {
		return
	}
	ps := pairingFromPersisted(&pp)
	pairingSecret = ps
	return
}

func SavePairing(pairingSecret *PairingSecret) (err error) {
	path, err := KrDirFile(PAIRING_FILENAME)
	if err != nil {
		return
	}
	pairingJson, err := json.Marshal(pairingToPersisted(pairingSecret))
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path, pairingJson, os.FileMode(0700))
	return
}

func DeletePairing() (pairingSecret *PairingSecret, err error) {
	path, err := KrDirFile(PAIRING_FILENAME)
	if err != nil {
		return
	}
	err = os.Remove(path)
	return
}
