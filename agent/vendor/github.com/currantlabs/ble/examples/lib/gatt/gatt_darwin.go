package gatt

import (
	"fmt"

	"github.com/currantlabs/ble/darwin"

	"github.com/currantlabs/ble"
)

// DefaultDevice returns the default device.
func DefaultDevice() Device {
	return nil
}

type manager struct {
	central    *darwin.Device
	peripheral *darwin.Device
}

var m manager

func central() (dev *darwin.Device, err error) {
	if m.central == nil {
		m.central, err = newDev(darwin.OptCentralRole())
	}
	if m.central == nil {
		err = fmt.Errorf("nil central")
		return
	}
	dev = m.central
	return
}

func peripheral() (dev *darwin.Device, err error) {
	if m.peripheral == nil {
		m.peripheral, err = newDev(darwin.OptPeripheralRole())
	}
	if m.peripheral == nil {
		err = fmt.Errorf("nil peripheral")
		return
	}
	dev = m.peripheral
	return
}

func Reset() {
	m.central = nil
	m.peripheral = nil
}

func newDev(opts ...darwin.Option) (dev *darwin.Device, err error) {
	dev, err = darwin.NewDevice(opts...)
	if err != nil {
		err = fmt.Errorf("create device failed: %s", err)
		return
	}
	if err = dev.Init(); err != nil {
		err = fmt.Errorf("init device failed: %s", err)
		return
	}
	return
}

// AddService adds a service to database.
func AddService(svc *ble.Service) (err error) {
	p, err := peripheral()
	if err != nil {
		return
	}
	return p.AddService(svc)
}

// RemoveAllServices removes all services that are currently in the database.
func RemoveAllServices() (err error) {
	p, err := peripheral()
	if err != nil {
		return
	}
	return p.RemoveAllServices()
}

// SetServices set the specified service to the database.
// It removes all currently added services, if any.
func SetServices(svcs []*ble.Service) (err error) {
	p, err := peripheral()
	if err != nil {
		return
	}
	return p.SetServices(svcs)
}

// Stop detatch the GATT peripheral from a peripheral device.
func Stop() (err error) {
	p, err := peripheral()
	if err != nil {
		return
	}
	return p.Stop()
}

// AdvertiseNameAndServices advertises device name, and specified service UUIDs.
// It tres to fit the UUIDs in the advertising packet as much as possible.
// If name doesn't fit in the advertising packet, it will be put in scan response.
func AdvertiseNameAndServices(name string, uuids ...ble.UUID) (err error) {
	p, err := peripheral()
	if err != nil {
		return
	}
	return p.AdvertiseNameAndServices(name, uuids...)
}

// AdvertiseIBeaconData advertise iBeacon with given manufacturer data.
func AdvertiseIBeaconData(b []byte) (err error) {
	p, err := peripheral()
	if err != nil {
		return
	}
	return p.AdvertiseIBeaconData(b)
}

// AdvertiseIBeacon advertises iBeacon with specified parameters.
func AdvertiseIBeacon(u ble.UUID, major, minor uint16, pwr int8) (err error) {
	p, err := peripheral()
	if err != nil {
		return
	}
	return p.AdvertiseIBeacon(u, major, minor, pwr)
}

// StopAdvertising stops advertising.
func StopAdvertising() (err error) {
	p, err := peripheral()
	if err != nil {
		return
	}
	return p.StopAdvertising()
}

// SetAdvHandler sets filter, handler.
func SetAdvHandler(h ble.AdvHandler) (err error) {
	c, err := central()
	if err != nil {
		return
	}
	return c.SetAdvHandler(h)
}

// Scan starts scanning. Duplicated advertisements will be filtered out if allowDup is set to false.
func Scan(allowDup bool) (err error) {
	c, err := central()
	if err != nil {
		return
	}
	return c.Scan(allowDup)
}

// StopScanning stops scanning.
func StopScanning() (err error) {
	c, err := central()
	if err != nil {
		return
	}
	return c.StopScanning()
}

// Addr returns the listener's device address.
func Addr() (addr ble.Addr, err error) {
	c, err := central()
	if err != nil {
		return
	}
	addr = c.Addr()
	return
}

// Dial ...
func Dial(a ble.Addr) (cli ble.Client, err error) {
	c, err := central()
	if err != nil {
		return
	}
	return c.Dial(a)
}
