package evt

import "encoding/binary"

func (e CommandComplete) NumHCICommandPackets() uint8 { return e[0] }
func (e CommandComplete) CommandOpcode() uint16       { return binary.LittleEndian.Uint16(e[1:]) }
func (e CommandComplete) ReturnParameters() []byte    { return e[3:] }

// Per-spec [Vol 2, Part E, 7.7.19], the packet structure should be:
//
//     NumOfHandle, HandleA, HandleB, CompPktNumA, CompPktNumB
//
// But we got the actual packet from BCM20702A1 with the following structure instead.
//
//     NumOfHandle, HandleA, CompPktNumA, HandleB, CompPktNumB
//              02,   40 00,       01 00,   41 00,       01 00

func (e NumberOfCompletedPackets) NumberOfHandles() uint8 { return e[0] }
func (e NumberOfCompletedPackets) ConnectionHandle(i int) uint16 {
	// return binary.LittleEndian.Uint16(e[1+i*2:])
	return binary.LittleEndian.Uint16(e[1+i*4:])
}
func (e NumberOfCompletedPackets) HCNumOfCompletedPackets(i int) uint16 {
	// return binary.LittleEndian.Uint16(e[1+int(e.NumberOfHandles())*2:])
	return binary.LittleEndian.Uint16(e[1+i*4+2:])
}
func (e LEAdvertisingReport) SubeventCode() uint8     { return e[0] }
func (e LEAdvertisingReport) NumReports() uint8       { return e[1] }
func (e LEAdvertisingReport) EventType(i int) uint8   { return e[2+i] }
func (e LEAdvertisingReport) AddressType(i int) uint8 { return e[2+int(e.NumReports())*1+i] }
func (e LEAdvertisingReport) Address(i int) [6]byte {
	e = e[2+int(e.NumReports())*2:]
	b := [6]byte{}
	copy(b[:], e[6*i:])
	return b
}

func (e LEAdvertisingReport) LengthData(i int) uint8 { return e[2+int(e.NumReports())*8+i] }

func (e LEAdvertisingReport) Data(i int) []byte {
	l := 0
	for j := 0; j < i; j++ {
		l += int(e.LengthData(j))
	}
	b := e[2+int(e.NumReports())*9+l:]
	return b[:e.LengthData(i)]
}

func (e LEAdvertisingReport) RSSI(i int) int8 {
	l := 0
	for j := 0; j < int(e.NumReports()); j++ {
		l += int(e.LengthData(j))
	}
	return int8(e[2+int(e.NumReports())*9+l+i])
}
