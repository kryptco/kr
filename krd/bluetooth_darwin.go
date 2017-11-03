// +build !nobluetooth

package krd

/*
#cgo LDFLAGS: -framework CoreFoundation -framework CoreBluetooth
#cgo LDFLAGS: -L /Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/lib/swift/macosx

#include <stdlib.h>
#include "../krbtle/krbtle/krbtle.h"

extern void KrbtleGoOnBluetoothData(void*, unsigned long long);

*/
import "C"

import (
	"sync"
	"unsafe"

	"github.com/op/go-logging"
	"github.com/satori/go.uuid"
)

var initOnce sync.Once
var globalReadChan = make(chan []byte, 2048)

func initFn() {
	C.krbtle_set_on_bluetooth_data((*C.KRBTLE_ON_BLUETOOTH_DATA_T)(unsafe.Pointer(C.KrbtleGoOnBluetoothData)))
}

//export KrbtleGoOnBluetoothData
func KrbtleGoOnBluetoothData(data unsafe.Pointer, dataLen C.ulonglong) {
	bytes := C.GoBytes(data, C.int(dataLen))

	//	optimistic send
	select {
	case globalReadChan <- bytes:
		break
	default:
		break
	}
}

type BluetoothDriver struct {
}

func NewBluetoothDriver() (bt *BluetoothDriver, err error) {
	initOnce.Do(initFn)
	bt = &BluetoothDriver{}
	return
}

func (bt *BluetoothDriver) AddService(serviceUUID uuid.UUID) (err error) {
	uuidString := []byte(serviceUUID.String())
	bytes := C.CBytes(uuidString)
	C.krbtle_add_service((*C.char)(bytes), C.ulonglong(len(uuidString)))
	return
}
func (bt *BluetoothDriver) RemoveService(serviceUUID uuid.UUID) (err error) {
	uuidString := []byte(serviceUUID.String())
	bytes := C.CBytes(uuidString)
	C.krbtle_remove_service((*C.char)(bytes), C.ulonglong(len(uuidString)))
	return
}
func (bt *BluetoothDriver) ReadChan() (readChan chan []byte, err error) {
	return globalReadChan, nil
}
func (bt *BluetoothDriver) Write(serviceUUID uuid.UUID, data []byte) (err error) {
	uuidString := []byte(serviceUUID.String())
	uuidBytes := C.CBytes(uuidString)
	dataBytes := C.CBytes(data)
	C.krbtle_write_data((*C.char)(uuidBytes), C.ulonglong(len(uuidString)),
		(*C.uint8_t)(dataBytes), C.ulonglong(len(data)))
	return
}

func (bt *BluetoothDriver) Stop() {
	C.krbtle_stop()
}

func SetBTLogger(logger *logging.Logger) {
}
