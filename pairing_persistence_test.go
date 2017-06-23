package kr

import (
	"testing"
)

func TestPairingPersistence(t *testing.T) {
	pairing, err := GeneratePairingSecret(nil)
	if err != nil {
		t.Fatal(err)
	}
	persisted := pairingToPersisted(pairing)
	pairing2 := pairingFromPersisted(&persisted)

	if !pairing.Equals(pairing2) {
		t.Fatal()
	}
}
