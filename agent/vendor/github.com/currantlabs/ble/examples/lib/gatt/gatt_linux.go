package gatt

import (
	"context"
	"fmt"
	"log"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/linux/att"
	"github.com/currantlabs/ble/linux/gatt"
	"github.com/currantlabs/ble/linux/hci"
)

// DefaultDevice returns the default HCI device.
func DefaultDevice() Device {
	return dev()
}

// DefaultServer returns the default GATT server.
func DefaultServer() *gatt.Server {
	return server()
}

type manager struct {
	server *gatt.Server
	dev    *hci.HCI
}

var m manager

func server() *gatt.Server {
	if m.server == nil {
		s, err := gatt.NewServer()
		if err != nil {
			log.Fatalf("create server failed: %s", err)
		}
		m.server = s
		start(dev(), s)
	}
	return m.server
}

func dev() *hci.HCI {
	if m.dev != nil {
		return m.dev
	}
	dev, err := hci.NewHCI()
	if err != nil {
		log.Fatalf("create hci failed: %s", err)
	}
	if err = dev.Init(); err != nil {
		log.Fatalf("init hci failed: %s", err)
	}
	m.dev = dev
	return dev
}

func start(dev *hci.HCI, s *gatt.Server) error {
	mtu := ble.DefaultMTU
	mtu = ble.MaxMTU // TODO: get this from user using Option.
	if mtu > ble.MaxMTU {
		return fmt.Errorf("maximum ATT_MTU is %d", ble.MaxMTU)
	}
	go func() {
		for {
			l2c, err := dev.Accept()
			if err != nil {
				log.Printf("can't accept: %s", err)
				return
			}

			// Initialize the per-connection cccd values.
			l2c.SetContext(context.WithValue(l2c.Context(), "ccc", make(map[uint16]uint16)))
			l2c.SetRxMTU(mtu)

			s.Lock()
			as, err := att.NewServer(s.DB(), l2c)
			s.Unlock()
			if err != nil {
				log.Printf("can't create ATT server: %s", err)
				continue

			}
			go as.Loop()
		}
	}()
	return nil
}

// AddService adds a service to database.
func AddService(svc *ble.Service) error {
	return server().AddService(svc)
}

// RemoveAllServices removes all services that are currently in the database.
func RemoveAllServices() error {
	return server().RemoveAllServices()
}

// SetServices set the specified service to the database.
// It removes all currently added services, if any.
func SetServices(svcs []*ble.Service) error {
	return server().SetServices(svcs)
}

// Stop detatch the GATT server from a peripheral device.
func Stop() error {
	return nil
}

// AdvertiseNameAndServices advertises device name, and specified service UUIDs.
// It tres to fit the UUIDs in the advertising packet as much as possible.
// If name doesn't fit in the advertising packet, it will be put in scan response.
func AdvertiseNameAndServices(name string, uuids ...ble.UUID) error {
	return dev().AdvertiseNameAndServices(name, uuids...)
}

// AdvertiseIBeaconData advertise iBeacon with given manufacturer data.
func AdvertiseIBeaconData(b []byte) error {
	return dev().AdvertiseIBeaconData(b)
}

// AdvertiseIBeacon advertises iBeacon with specified parameters.
func AdvertiseIBeacon(u ble.UUID, major, minor uint16, pwr int8) error {
	return dev().AdvertiseIBeacon(u, major, minor, pwr)
}

// StopAdvertising stops advertising.
func StopAdvertising() error {
	return dev().StopAdvertising()
}

// SetAdvHandler sets filter, handler.
func SetAdvHandler(h ble.AdvHandler) error {
	return dev().SetAdvHandler(h)
}

// Scan starts scanning. Duplicated advertisements will be filtered out if allowDup is set to false.
func Scan(allowDup bool) error {
	return dev().Scan(allowDup)
}

// StopScanning stops scanning.
func StopScanning() error {
	return dev().StopScanning()
}

// Close closes the listner.
// Any blocked Accept operations will be unblocked and return errors.
func Close() error {
	return dev().Close()
}

// Addr returns the listener's device address.
func Addr() ble.Addr {
	return dev().Addr()
}

// Dial ...
func Dial(a ble.Addr) (ble.Client, error) {
	return dev().Dial(a)
}
