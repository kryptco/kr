// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ble

import "v.io/v23/context"

type DriverFactory func(ctx *context.T, host string) (Driver, error)

var (
	driverFactory DriverFactory = newDummyDriver
)

type dummyDriver struct{}

func (dummyDriver) AddService(uuid string, characteristics map[string][]byte) error { return nil }
func (dummyDriver) RemoveService(uuid string)                                       {}
func (dummyDriver) StartScan(uuids []string, baseUuid, maskUUid string, handler ScanHandler) error {
	return nil
}
func (dummyDriver) StopScan()           {}
func (dummyDriver) DebugString() string { return "BLE not available" }

func newDummyDriver(ctx *context.T, host string) (Driver, error) { return dummyDriver{}, nil }

// SetPluginFactory sets the plugin factory with the given name.
// This should be called before v23.NewDiscovery() is called.
func SetDriverFactory(factory DriverFactory) {
	driverFactory = factory
}
