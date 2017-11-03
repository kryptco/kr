package krd

import (
	"github.com/satori/go.uuid"
)

const krsshCharUUIDString = "20F53E48-C08D-423A-B2C2-1C797889AF24"

type BluetoothDriverI interface {
	AddService(uuid.UUID) (err error)
	RemoveService(uuid.UUID) (err error)
	Write(uuid.UUID, []byte) (err error)
	ReadChan() (readChan chan []byte, err error)
	Stop()
}
