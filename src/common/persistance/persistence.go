package persistance

import (
	. "krypt.co/kr/common/util"
	. "krypt.co/kr/common/protocol"
)

type Persister interface {
	SaveMe(me Profile) (err error)
	LoadMe() (me Profile, err error)
	DeleteMe() (err error)
	SaveMySSHPubKey(me Profile) (err error)

	LoadPairing() (pairingSecret *PairingSecret, err error)
	SavePairing(pairingSecret *PairingSecret) (err error)
	DeletePairing() (pairingSecret *PairingSecret, err error)
}
