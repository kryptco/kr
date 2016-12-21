// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ble

// Driver provides an abstraction for an underlying mechanism to discover
// near-by Vanadium services through Bluetooth Low Energy (BLE).
//
// We publish a Vanadium service as a Gatt service with characteristics that contains
// the encoded service informations. Each characteristics value is up to 512 bytes as
// the Bluetooth specification limited. Every service is advertised with 128-bit service
// uuid that are generated from the interface name and are toggled on every update.
//
// The driver should ignore all operations while BLE is not available, but
// the operations should be automatically resumed when BLE become available.
type Driver interface {
	// AddService adds a new service to the GATT server with the given service uuid
	// and characteristics and starts advertising the service uuid.
	//
	// The characteristics will not be changed while it is being advertised.
	//
	// There can be multiple services at any given time and it is the driver's
	// responsibility to handle multiple advertisements in a compatible way.
	AddService(uuid string, characteristics map[string][]byte) error

	// RemoveService removes the service from the GATT server and stops advertising
	// the service uuid.
	RemoveService(uuid string)

	// StartScan starts BLE scanning for the specified uuids and the scan results will be
	// delivered through the scan handler.
	//
	// An empty uuids means all Vanadium services. The driver may use baseUuid and maskUuid
	// to filter Vanadium services.
	//
	// It is guarantted that there is at most one active scan at any given time. That is,
	// StopScan() will be called before starting a new scan.
	StartScan(uuids []string, baseUuid, maskUuid string, handler ScanHandler) error

	// StopScan stops BLE scanning.
	StopScan()

	// DebugString return a human-readable string description of the driver.
	DebugString() string
}

// A ScanHandler is used to deliver scan results.
type ScanHandler interface {
	// OnDiscovered is called when a target Vanadium service has been discovered.
	//
	// Optionally the received signal strength in dBm can be passed to rssi.
	// The valid range is [-127, 0).
	OnDiscovered(uuid string, characteristics map[string][]byte, rssi int)
}
