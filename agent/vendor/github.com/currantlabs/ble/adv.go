package ble

// AdvHandler ...
type AdvHandler interface {
	Handle(a Advertisement)
}

// The AdvHandlerFunc type is an adapter to allow the use of ordinary functions as packet or event handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler object that calls f.
type AdvHandlerFunc func(a Advertisement)

// Handle handles an advertisement.
func (f AdvHandlerFunc) Handle(a Advertisement) {
	f(a)
}

// Advertisement ...
type Advertisement interface {
	LocalName() string
	ManufacturerData() []byte
	ServiceData() []ServiceData
	Services() []UUID
	OverflowService() []UUID
	TxPowerLevel() int
	Connectable() bool
	SolicitedService() []UUID

	RSSI() int
	Address() Addr
}

// ServiceData ...
type ServiceData struct {
	UUID UUID
	Data []byte
}
