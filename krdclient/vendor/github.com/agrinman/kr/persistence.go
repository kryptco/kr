package kr

type Persister interface {
	SaveMe(me Profile) (err error)
	LoadMe() (me Profile, err error)
	DeleteMe() (err error)
	SaveMySSHPubKey(me Profile) (err error)

	LoadPairing() (pairingSecret *PairingSecret, err error)
	SavePairing(pairingSecret *PairingSecret) (err error)
	DeletePairing() (pairingSecret *PairingSecret, err error)
}
