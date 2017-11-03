// +build !nobluetooth

package krd

import (
	"github.com/op/go-logging"
	"github.com/satori/go.uuid"
)

type BluetoothDriver struct {
}

func NewBluetoothDriver() (bt *BluetoothDriver, err error) {
	bt = &BluetoothDriver{}
	return
}

func (bt *BluetoothDriver) AddService(serviceUUID uuid.UUID) (err error) {
	return
}
func (bt *BluetoothDriver) RemoveService(serviceUUID uuid.UUID) (err error) {
	return
}
func (bt *BluetoothDriver) ReadChan() (readChan chan []byte, err error) {
	readChan = make(chan []byte)
	close(readChan)
	return
}
func (bt *BluetoothDriver) Write(serviceUUID uuid.UUID, data []byte) (err error) {
	return
}

func (bt *BluetoothDriver) Stop() {}

func SetBTLogger(logger *logging.Logger) {
}
