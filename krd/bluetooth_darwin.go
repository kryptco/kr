// +build !nobluetooth

package main

import (
	"github.com/kryptco/go.corebluetooth"
	"github.com/op/go-logging"
	"github.com/satori/go.uuid"
)

type BluetoothDriver struct {
	*corebluetooth.CoreBluetoothDriver
}

func NewBluetoothDriver() (bt *BluetoothDriver, err error) {
	coreBT, err := corebluetooth.NewContextAndDriver()
	if err != nil {
		return
	}
	bt = &BluetoothDriver{coreBT}
	return
}

func (bt *BluetoothDriver) AddService(serviceUUID uuid.UUID) (err error) {
	err = bt.CoreBluetoothDriver.AddService(serviceUUID.String(), map[string][]byte{})
	return
}
func (bt *BluetoothDriver) RemoveService(serviceUUID uuid.UUID) (err error) {
	bt.CoreBluetoothDriver.RemoveService(serviceUUID.String())
	return
}
func (bt *BluetoothDriver) ReadChan() (readChan chan []byte, err error) {
	readChan = bt.CoreBluetoothDriver.Read
	return
}
func (bt *BluetoothDriver) Write(data []byte) (err error) {
	err = bt.CoreBluetoothDriver.WriteData(data)
	return
}

func (bt *BluetoothDriver) Stop() {
	corebluetooth.Clean()
}

func SetBTLogger(logger *logging.Logger) {
	corebluetooth.SetLogger(logger)
}
