package adv

import (
	"encoding/binary"

	"github.com/currantlabs/ble"
)

// Packet is an implemntation of ble.AdvPacket for crafting or parsing an advertising packet or scan response.
// Refer to Supplement to Bluetooth Core Specification | CSSv6, Part A.
type Packet struct {
	b []byte
}

// Bytes returns the bytes of the packet.
func (p *Packet) Bytes() []byte {
	return p.b
}

// Len returns the length of the packet.
func (p *Packet) Len() int {
	return len(p.b)
}

// NewPacket returns a new advertising Packet.
func NewPacket(fields ...Field) (*Packet, error) {
	p := &Packet{b: make([]byte, 0, MaxEIRPacketLength)}
	for _, f := range fields {
		if err := f(p); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// NewRawPacket returns a new advertising Packet.
func NewRawPacket(bytes ...[]byte) *Packet {
	p := &Packet{b: make([]byte, 0, MaxEIRPacketLength)}
	for _, b := range bytes {
		p.b = append(p.b, b...)
	}
	return p
}

// Field is an advertising field which can be appended to a packet.
type Field func(p *Packet) error

// Append appends a field to the packet. It returns ErrNotFit if the field
// doesn't fit into the packet, and leaves the packet intact.
func (p *Packet) Append(f Field) error {
	return f(p)
}

// appends appends a field to the packet. It returns ErrNotFit if the field
// doesn't fit into the packet, and leaves the packet intact.
func (p *Packet) append(typ byte, b []byte) error {
	if p.Len()+1+1+len(b) > MaxEIRPacketLength {
		return ErrNotFit
	}
	p.b = append(p.b, byte(len(b)+1))
	p.b = append(p.b, typ)
	p.b = append(p.b, b...)
	return nil
}

// Raw appends the bytes to the current packet.
// This is helpful for creating new packet from existing packets.
func Raw(b []byte) Field {
	return func(p *Packet) error {
		if p.Len()+len(b) > MaxEIRPacketLength {
			return ErrNotFit
		}
		p.b = append(p.b, b...)
		return nil
	}
}

// IBeaconData returns an iBeacon advertising packet with specified parameters.
func IBeaconData(md []byte) Field {
	return func(p *Packet) error {
		return ManufacturerData(0x004C, md)(p)
	}
}

// IBeacon returns an iBeacon advertising packet with specified parameters.
func IBeacon(u ble.UUID, major, minor uint16, pwr int8) Field {
	return func(p *Packet) error {
		if u.Len() != 16 {
			return ErrInvalid
		}
		md := make([]byte, 23)
		md[0] = 0x02                               // Data type: iBeacon
		md[1] = 0x15                               // Data length: 21 bytes
		copy(md[2:], ble.Reverse(u))                // Big endian
		binary.BigEndian.PutUint16(md[18:], major) // Big endian
		binary.BigEndian.PutUint16(md[20:], minor) // Big endian
		md[22] = uint8(pwr)                        // Measured Tx Power
		return ManufacturerData(0x004C, md)(p)
	}
}

// Flags is a flags.
func Flags(f byte) Field {
	return func(p *Packet) error {
		return p.append(flags, []byte{f})
	}
}

// ShortName is a short local name.
func ShortName(n string) Field {
	return func(p *Packet) error {
		return p.append(shortName, []byte(n))
	}
}

// CompleteName is a compelete local name.
func CompleteName(n string) Field {
	return func(p *Packet) error {
		return p.append(completeName, []byte(n))
	}
}

// ManufacturerData is manufacturer specific data.
func ManufacturerData(id uint16, b []byte) Field {
	return func(p *Packet) error {
		d := append([]byte{uint8(id), uint8(id >> 8)}, b...)
		return p.append(manufacturerData, d)
	}
}

// AllUUID is one of the complete service UUID list.
func AllUUID(u ble.UUID) Field {
	return func(p *Packet) error {
		if u.Len() == 2 {
			return p.append(allUUID16, u)
		}
		if u.Len() == 4 {
			return p.append(allUUID32, u)
		}
		return p.append(allUUID128, u)
	}
}

// SomeUUID is one of the incomplete service UUID list.
func SomeUUID(u ble.UUID) Field {
	return func(p *Packet) error {
		if u.Len() == 2 {
			return p.append(someUUID16, u)
		}
		if u.Len() == 4 {
			return p.append(someUUID32, u)
		}
		return p.append(someUUID128, u)
	}
}

// Field returns the field data (excluding the initial length and typ byte).
// It returns nil, if the specified field is not found.
func (p *Packet) Field(typ byte) []byte {
	b := p.b
	for len(b) > 0 {
		if len(b) < 2 {
			return nil
		}
		l, t := b[0], b[1]
		if len(b) < int(1+l) {
			return nil
		}
		if t == typ {
			return b[2 : 2+l-1]
		}
		b = b[1+l:]
	}
	return nil
}

// Flags returns the flags of the packet.
func (p *Packet) Flags() (flags byte, present bool) {
	b := p.Field(flags)
	if len(b) < 2 {
		return 0, false
	}
	return b[2], true
}

// LocalName returns the ShortName or CompleteName if it presents.
func (p *Packet) LocalName() string {
	if b := p.Field(shortName); b != nil {
		return string(b)
	}
	return string(p.Field(completeName))
}

// TxPower returns the TxPower, if it presents.
func (p *Packet) TxPower() (power int, present bool) {
	b := p.Field(txPower)
	if len(b) < 3 {
		return 0, false
	}
	return int(int8(b[2])), true
}

// UUIDs returns a list of service UUIDs.
func (p *Packet) UUIDs() []ble.UUID {
	var u []ble.UUID
	if b := p.Field(someUUID16); b != nil {
		u = uuidList(u, b, 2)
	}
	if b := p.Field(allUUID16); b != nil {
		u = uuidList(u, b, 2)
	}
	if b := p.Field(someUUID32); b != nil {
		u = uuidList(u, b, 4)
	}
	if b := p.Field(allUUID32); b != nil {
		u = uuidList(u, b, 4)
	}
	if b := p.Field(someUUID128); b != nil {
		u = uuidList(u, b, 16)
	}
	if b := p.Field(allUUID128); b != nil {
		u = uuidList(u, b, 16)
	}
	return u
}

// ServiceSol ...
func (p *Packet) ServiceSol() []ble.UUID {
	var u []ble.UUID
	if b := p.Field(serviceSol16); b != nil {
		u = uuidList(u, b, 2)
	}
	if b := p.Field(serviceSol32); b != nil {
		u = uuidList(u, b, 16)
	}
	if b := p.Field(serviceSol128); b != nil {
		u = uuidList(u, b, 16)
	}
	return u
}

// ServiceData ...
func (p *Packet) ServiceData() []ble.ServiceData {
	var s []ble.ServiceData
	if b := p.Field(serviceData16); b != nil {
		s = serviceDataList(s, b, 2)
	}
	if b := p.Field(serviceData32); b != nil {
		s = serviceDataList(s, b, 4)
	}
	if b := p.Field(serviceData128); b != nil {
		s = serviceDataList(s, b, 16)
	}
	return s
}

// ManufacturerData returns the ManufacturerData field if it presents.
func (p *Packet) ManufacturerData() []byte {
	return p.Field(manufacturerData)
}

// Utility function for creating a list of uuids.
func uuidList(u []ble.UUID, d []byte, w int) []ble.UUID {
	for len(d) > 0 {
		u = append(u, ble.UUID(d[:w]))
		d = d[w:]
	}
	return u
}

func serviceDataList(sd []ble.ServiceData, d []byte, w int) []ble.ServiceData {
	serviceData := ble.ServiceData{
		UUID: ble.UUID(d[:w]),
		Data: make([]byte, len(d)-w),
	}
	copy(serviceData.Data, d[2:])
	return append(sd, serviceData)
}
