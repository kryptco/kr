// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin,cgo

// Package corebluetooth provides an implementation of ble.Driver using the CoreBluetooth Objective-C API
//
// The bridge rules between the two are as follows:
//   THREADS:
//      Everything in obj-c runs single threaded on a dedicated Grand Central Dispatch queue (read: thread).
//      Obj-C is responsible for getting itself on that thread in calls from Go.
//      Go is responsible for getting _off_ the obj-c queue via goroutines when calling out to the rest of
//      the stack.
//   MEMORY:
//      Callers retain ownership of their memory -- callee must copy right away.
//      The exception to this is when memory is returned either via the function or via a double pointer
//      (in the case of errorOut following Obj-C semantics of returning BOOL matched with passing of NSError **).
//      In this case ownership is transfered and callee must free.
package corebluetooth

import (
	"errors"
	"strings"
	"sync"
	"unsafe"

	"github.com/op/go-logging"
	"github.com/vanadium/go.ref/lib/discovery/plugins/ble"
	"github.com/vanadium/go.v23/context"
)

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -DCBLOG_LEVEL=CBLOG_LEVEL_ERROR
#cgo LDFLAGS: -framework Foundation -framework CoreBluetooth
#import <CoreBluetooth/CoreBluetooth.h>
#import "CBDriver.h"

static int objcBOOL2int(BOOL b) {
	return (int)b;
}
*/
import "C"

var log *logging.Logger

func SetLogger(logger *logging.Logger) {
	driverMu.Lock()
	defer driverMu.Unlock()
	log = logger
}

func logPrintln(args ...interface{}) {
	driverMu.Lock()
	defer driverMu.Unlock()
	if log != nil {
		log.Info([]interface{}{"CoreBluetooth:", args}...)
	}
}
func logError(args ...interface{}) {
	driverMu.Lock()
	defer driverMu.Unlock()
	if log != nil {
		log.Error([]interface{}{"CoreBluetooth:", args}...)
	}
}

func logNotice(args ...interface{}) {
	driverMu.Lock()
	defer driverMu.Unlock()
	if log != nil {
		log.Notice([]interface{}{"CoreBluetooth:", args}...)
	}
}

func logInfo(args ...interface{}) {
	driverMu.Lock()
	defer driverMu.Unlock()
	if log != nil {
		log.Info([]interface{}{"CoreBluetooth:", args}...)
	}
}

type (
	// CoreBluetoothDriver provides an abstraction for an underlying mechanism to discover
	// near-by Vanadium services through Bluetooth Low Energy (BLE) with CoreBluetooth.
	//
	// See Driver for more documentation.
	CoreBluetoothDriver struct {
		ctx          *context.T
		scanHandler  ble.ScanHandler
		mu           sync.Mutex
		Read         chan []byte
		splitMessage []byte
	}

	OnDiscovered struct {
		UUID            string
		Characteristics map[string][]byte
		RSSI            int
	}
)

var (
	driverMu sync.Mutex
	driver   *CoreBluetoothDriver
)

func NewContextAndDriver() (*CoreBluetoothDriver, error) {
	ctx, _ := context.RootContext()
	return New(ctx)
}
func New(ctx *context.T) (*CoreBluetoothDriver, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}
	driverMu.Lock()
	defer driverMu.Unlock()
	if driver != nil {
		return nil, errors.New("only one corebluetooth driver can be created at a time; call .Clean() instead")
	}
	driver = &CoreBluetoothDriver{ctx: ctx, Read: make(chan []byte, 2048)}
	// Clean everything when the context is done.
	go func() {
		<-ctx.Done()
		Clean()
	}()
	return driver, nil
}

// Clean shuts down any existing scans/advertisements, releases the BLE hardware, removes the
// objective-c singleton from memory, and releases the global driver in this package. It is
// necessary before New may be called.
func Clean() {
	driverMu.Lock()
	if driver != nil {
		driver.StopScan()
		close(driver.Read)
	}
	// This is thread safe in objc
	C.v23_cbdriver_clean()
	driver = nil
	driverMu.Unlock()
}

func (d *CoreBluetoothDriver) NumServicesAdvertising() int {
	return int(C.v23_cbdriver_advertisingServiceCount())
}

// AddService implements v.io/x/lib/discovery/plugins/ble.Driver.AddService
func (d *CoreBluetoothDriver) AddService(uuid string, characteristics map[string][]byte) error {
	// Convert args to C
	entries := C.malloc(C.size_t(len(characteristics)) * C.sizeof_CBDriverCharacteristicMapEntry)
	// See CGO Wiki on how we can use this weird looking technique to get a go slice out of a c array
	// https://github.com/golang/go/wiki/cgo
	entriesSlice := (*[1 << 30]C.CBDriverCharacteristicMapEntry)(unsafe.Pointer(entries))[:len(characteristics):len(characteristics)]
	i := 0
	for characteristicUuid, data := range characteristics {
		var entry C.CBDriverCharacteristicMapEntry
		entry.uuid = C.CString(characteristicUuid)
		entry.data = unsafe.Pointer(&data[0])
		entry.dataLength = C.int(len(data))
		entriesSlice[i] = entry
		i++
	}
	defer func() {
		for _, entry := range entriesSlice {
			C.free(unsafe.Pointer(entry.uuid))
		}
		C.free(unsafe.Pointer(entries))
	}()
	// Call objective-c
	var errorOut *C.char = nil
	// This is thread-safe in obj-c
	if err := objcBOOL2Error(C.v23_cbdriver_addService(C.CString(uuid), (*C.CBDriverCharacteristicMapEntry)(entries), C.int(len(characteristics)), &errorOut), &errorOut); err != nil {
		return err
	}
	// Success
	logNotice("added service ", uuid)
	return nil
}

func (d *CoreBluetoothDriver) WriteData(data []byte) error {
	if len(data) == 0 {
		return errors.New("Cannot write empty data")
	}
	// See CGO Wiki on how we can use this weird looking technique to get a go slice out of a c array
	// https://github.com/golang/go/wiki/cgo
	// Call objective-c
	var errorOut *C.char = nil
	// This is thread-safe in obj-c
	if err := objcBOOL2Error(C.v23_cbdriver_writeData((*C.char)(unsafe.Pointer(&data[0])), C.int(len(data)), &errorOut), &errorOut); err != nil {
		return err
	}
	// Success
	logInfo("wrote", len(data), "bytes")
	return nil
}

// RemoveService implements v.io/x/lib/discovery/plugins/ble.Driver.RemoveService
func (d *CoreBluetoothDriver) RemoveService(uuid string) {
	cUuid := C.CString(uuid)
	// This is thread-safe in obj-c
	C.v23_cbdriver_removeService(cUuid)
	C.free(unsafe.Pointer(cUuid))
	logNotice("removed service ", uuid)
}

// StartScan implements v.io/x/lib/discovery/plugins/ble.Driver.StartService
func (d *CoreBluetoothDriver) StartScan(uuids []string, baseUuid, maskUuid string, handler ble.ScanHandler) error {
	// Convert args to C
	cUuids := C.malloc(C.sizeof_size_t * C.size_t(len(uuids)))
	// See CGO Wiki on how we can use this weird looking technique to get a go slice out of a c array
	// https://github.com/golang/go/wiki/cgo
	cUuidsSlice := (*[1 << 30]*C.char)(unsafe.Pointer(cUuids))[:len(uuids):len(uuids)]
	for i, uuid := range uuids {
		cUuidsSlice[i] = C.CString(uuid)
	}
	cBaseUuid := C.CString(baseUuid)
	cMaskUuid := C.CString(maskUuid)
	defer func() {
		for _, cUuid := range cUuidsSlice {
			C.free(unsafe.Pointer(cUuid))
		}
		C.free(unsafe.Pointer(cUuids))
		C.free(unsafe.Pointer(cBaseUuid))
		C.free(unsafe.Pointer(cMaskUuid))
	}()

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.scanHandler != nil {
		return errors.New("scan already in progress")
	}
	// Kick start handler
	d.scanHandler = handler
	// Call Objective-C
	var errorOut *C.char = nil
	if err := objcBOOL2Error(C.v23_cbdriver_startScan((**C.char)(cUuids), C.int(len(uuids)), cBaseUuid, cMaskUuid, &errorOut), &errorOut); err != nil {
		d.scanHandler = nil
		return err
	}
	// Success
	return nil
}

//export v23_corebluetooth_scan_handler_on_discovered
func v23_corebluetooth_scan_handler_on_discovered(cUuid *C.char, cEntries *C.CBDriverCharacteristicMapEntry, entriesLength C.int, rssi C.int) {
	uuid := strings.ToLower(C.GoString(cUuid))
	characteristics := map[string][]byte{}
	if cEntries != nil && entriesLength > 0 {
		// See CGO Wiki on how we can use this weird looking technique to get a go slice out of a c array
		// https://github.com/golang/go/wiki/cgo
		entries := (*[1 << 30]C.CBDriverCharacteristicMapEntry)(unsafe.Pointer(cEntries))[:int(entriesLength):int(entriesLength)]
		for _, entry := range entries {
			characteristicUuid := strings.ToLower(C.GoString(entry.uuid))
			data := C.GoBytes(entry.data, entry.dataLength)
			characteristics[characteristicUuid] = data
		}
	}
	driverMu.Lock()
	defer driverMu.Unlock()
	if driver == nil {
		logError("got onDiscovered event from CoreBluetooth but missing driver -- dropping")
		return
	}
	driver.mu.Lock()
	// Callbacks should happen off Swift threads and instead on a go routine.
	// We use a local variable to avoid closure on driver itself since we have it currently locked.
	if sh := driver.scanHandler; sh != nil {
		go func() {
			sh.OnDiscovered(uuid, characteristics, int(rssi))
		}()
	}
	driver.mu.Unlock()
}

// StopScan implements v.io/x/lib/discovery/plugins/ble.Driver.StopScan
func (d *CoreBluetoothDriver) StopScan() {
	// This call is thread-safe in obj-c
	C.v23_cbdriver_stopScan()
	d.mu.Lock()
	d.scanHandler = nil
	d.mu.Unlock()
}

// DebugString implements v.io/x/lib/discovery/plugins/ble.Driver.DebugString by
// returning the current state of the CoreBluetooth driver in a string description
func (d *CoreBluetoothDriver) DebugString() string {
	cstr := C.v23_cbdriver_debug_string()
	str := C.GoString(cstr)
	C.free(unsafe.Pointer(cstr))
	return str
}

// Callback from Obj-C
//export v23_corebluetooth_go_log
func v23_corebluetooth_go_log(message *C.char) {
	msg := C.GoString(message)
	// Run asynchronously to prevent deadlocks where us calling functions like stopScan log
	// while already retaining this lock.
	go func() {
		logInfo(msg)
	}()
}

// Callback from Obj-C
//export v23_corebluetooth_go_log_error
func v23_corebluetooth_go_log_error(message *C.char) {
	msg := C.GoString(message)
	// Run asynchronously to prevent deadlocks where us calling functions like stopScan log
	// while already retaining this lock.
	go func() {
		logError(msg)
	}()
}

//export v23_corebluetooth_go_data_received
func v23_corebluetooth_go_data_received(data unsafe.Pointer, dataLength C.int) {
	borrowedBytes := C.GoBytes(data, dataLength)
	copiedBytes := make([]byte, len(borrowedBytes))
	if len(copiedBytes) < 2 {
		return
	}
	copy(copiedBytes, borrowedBytes)
	n := copiedBytes[0]
	msg := copiedBytes[1:]
	if n == 0 {
		message := append(driver.splitMessage, msg...)
		driver.splitMessage = []byte{}
		select {
		case driver.Read <- message:
			logPrintln("received", len(message), "byte message over BT")
		default:
			logPrintln("receive queue unavailable")
		}
	} else {
		driver.splitMessage = append(driver.splitMessage, msg...)
		logPrintln(" Received", len(copiedBytes), "byte message split over BT")
	}
}

func objcBOOL2Error(b C.BOOL, errStr **C.char) error {
	// Any non-zero means true for Obj-C BOOL
	if int(C.objcBOOL2int(b)) != 0 {
		return nil
	}
	err := C.GoString(*errStr)
	C.free(unsafe.Pointer(*errStr))
	return errors.New(err)
}
