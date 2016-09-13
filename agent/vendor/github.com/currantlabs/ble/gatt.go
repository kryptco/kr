package ble

// A Client is a GATT client.
type Client interface {
	// Address returns platform specific unique ID of the remote peripheral, e.g. MAC on Linux, Client UUID on OS X.
	Address() Addr

	// Name returns the name of the remote peripheral.
	// This can be the advertised name, if exists, or the GAP device name, which takes priority.
	Name() string

	// Profile returns discovered profile.
	Profile() *Profile

	// DiscoverProfile discovers the whole hierachy of a server.
	DiscoverProfile(force bool) (*Profile, error)
	// DiscoverServices finds all the primary services on a server. [Vol 3, Part G, 4.4.1]
	// If filter is specified, only filtered services are returned.
	DiscoverServices(filter []UUID) ([]*Service, error)

	// DiscoverIncludedServices finds the included services of a service. [Vol 3, Part G, 4.5.1]
	// If filter is specified, only filtered services are returned.
	DiscoverIncludedServices(filter []UUID, s *Service) ([]*Service, error)

	// DiscoverCharacteristics finds all the characteristics within a service. [Vol 3, Part G, 4.6.1]
	// If filter is specified, only filtered characteristics are returned.
	DiscoverCharacteristics(filter []UUID, s *Service) ([]*Characteristic, error)

	// DiscoverDescriptors finds all the descriptors within a characteristic. [Vol 3, Part G, 4.7.1]
	// If filter is specified, only filtered descriptors are returned.
	DiscoverDescriptors(filter []UUID, c *Characteristic) ([]*Descriptor, error)

	// ReadCharacteristic reads a characteristic value from a server. [Vol 3, Part G, 4.8.1]
	ReadCharacteristic(c *Characteristic) ([]byte, error)

	// ReadLongCharacteristic reads a characteristic value which is longer than the MTU. [Vol 3, Part G, 4.8.3]
	ReadLongCharacteristic(c *Characteristic) ([]byte, error)

	// WriteCharacteristic writes a characteristic value to a server. [Vol 3, Part G, 4.9.3]
	WriteCharacteristic(c *Characteristic, value []byte, noRsp bool) error

	// ReadDescriptor reads a characteristic descriptor from a server. [Vol 3, Part G, 4.12.1]
	ReadDescriptor(d *Descriptor) ([]byte, error)

	// WriteDescriptor writes a characteristic descriptor to a server. [Vol 3, Part G, 4.12.3]
	WriteDescriptor(d *Descriptor, v []byte) error

	// ReadRSSI retrieves the current RSSI value of remote peripheral. [Vol 2, Part E, 7.5.4]
	ReadRSSI() int

	// ExchangeMTU set the ATT_MTU to the maximum possible value that can be supported by both devices [Vol 3, Part G, 4.3.1]
	ExchangeMTU(rxMTU int) (txMTU int, err error)

	// Subscribe subscribes to indication (if ind is set true), or notification of a characteristic value. [Vol 3, Part G, 4.10 & 4.11]
	Subscribe(c *Characteristic, ind bool, h NotificationHandler) error

	// Unsubscribe unsubscribes to indication (if ind is set true), or notification of a specified characteristic value. [Vol 3, Part G, 4.10 & 4.11]
	Unsubscribe(c *Characteristic, ind bool) error

	// ClearSubscriptions clears all subscriptions to notifications and indications.
	ClearSubscriptions() error

	// CancelConnection disconnects the connection.
	CancelConnection() error
}

// A NotificationHandler handles notification or indication from a server.
type NotificationHandler func(req []byte)
