package evt

import "encoding/binary"

const DisconnectionCompleteCode = 0x05

// DisconnectionComplete implements Disconnection Complete (0x05) [Vol 2, Part E, 7.7.5].
type DisconnectionComplete []byte

func (r DisconnectionComplete) Status() uint8 { return r[0] }

func (r DisconnectionComplete) ConnectionHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

func (r DisconnectionComplete) Reason() uint8 { return r[3] }

const EncryptionChangeCode = 0x08

// EncryptionChange implements Encryption Change (0x08) [Vol 2, Part E, 7.7.8].
type EncryptionChange []byte

func (r EncryptionChange) Status() uint8 { return r[0] }

func (r EncryptionChange) ConnectionHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

func (r EncryptionChange) EncryptionEnabled() uint8 { return r[3] }

const ReadRemoteVersionInformationCompleteCode = 0x0C

// ReadRemoteVersionInformationComplete implements Read Remote Version Information Complete (0x0C) [Vol 2, Part E, 7.7.12].
type ReadRemoteVersionInformationComplete []byte

func (r ReadRemoteVersionInformationComplete) Status() uint8 { return r[0] }

func (r ReadRemoteVersionInformationComplete) ConnectionHandle() uint16 {
	return binary.LittleEndian.Uint16(r[1:])
}

func (r ReadRemoteVersionInformationComplete) Version() uint8 { return r[3] }

func (r ReadRemoteVersionInformationComplete) ManufacturerName() uint16 {
	return binary.LittleEndian.Uint16(r[4:])
}

func (r ReadRemoteVersionInformationComplete) Subversion() uint16 {
	return binary.LittleEndian.Uint16(r[6:])
}

const CommandCompleteCode = 0x0E

// CommandComplete implements Command Complete (0x0E) [Vol 2, Part E, 7.7.14].
type CommandComplete []byte

const CommandStatusCode = 0x0F

// CommandStatus implements Command Status (0x0F) [Vol 2, Part E, 7.7.15].
type CommandStatus []byte

func (r CommandStatus) Status() uint8 { return r[0] }

func (r CommandStatus) NumHCICommandPackets() uint8 { return r[1] }

func (r CommandStatus) CommandOpcode() uint16 { return binary.LittleEndian.Uint16(r[2:]) }

const HardwareErrorCode = 0x10

// HardwareError implements Hardware Error (0x10) [Vol 2, Part E, 7.7.16].
type HardwareError []byte

func (r HardwareError) HardwareCode() uint8 { return r[0] }

const NumberOfCompletedPacketsCode = 0x13

// NumberOfCompletedPackets implements Number Of Completed Packets (0x13) [Vol 2, Part E, 7.7.19].
type NumberOfCompletedPackets []byte

const DataBufferOverflowCode = 0x1A

// DataBufferOverflow implements Data Buffer Overflow (0x1A) [Vol 2, Part E, 7.7.26].
type DataBufferOverflow []byte

func (r DataBufferOverflow) LinkType() uint8 { return r[0] }

const EncryptionKeyRefreshCompleteCode = 0x30

// EncryptionKeyRefreshComplete implements Encryption Key Refresh Complete (0x30) [Vol 2, Part E, 7.7.39].
type EncryptionKeyRefreshComplete []byte

func (r EncryptionKeyRefreshComplete) Status() uint8 { return r[0] }

func (r EncryptionKeyRefreshComplete) ConnectionHandle() uint16 {
	return binary.LittleEndian.Uint16(r[1:])
}

const LEConnectionCompleteCode = 0x3E

const LEConnectionCompleteSubCode = 0x01

// LEConnectionComplete implements LE Connection Complete (0x3E:0x01) [Vol 2, Part E, 7.7.65.1].
type LEConnectionComplete []byte

func (r LEConnectionComplete) SubeventCode() uint8 { return r[0] }

func (r LEConnectionComplete) Status() uint8 { return r[1] }

func (r LEConnectionComplete) ConnectionHandle() uint16 { return binary.LittleEndian.Uint16(r[2:]) }

func (r LEConnectionComplete) Role() uint8 { return r[4] }

func (r LEConnectionComplete) PeerAddressType() uint8 { return r[5] }

func (r LEConnectionComplete) PeerAddress() [6]byte {
	b := [6]byte{}
	copy(b[:], r[6:])
	return b
}

func (r LEConnectionComplete) ConnInterval() uint16 { return binary.LittleEndian.Uint16(r[12:]) }

func (r LEConnectionComplete) ConnLatency() uint16 { return binary.LittleEndian.Uint16(r[14:]) }

func (r LEConnectionComplete) SupervisionTimeout() uint16 { return binary.LittleEndian.Uint16(r[16:]) }

func (r LEConnectionComplete) MasterClockAccuracy() uint8 { return r[18] }

const LEAdvertisingReportCode = 0x3E

const LEAdvertisingReportSubCode = 0x02

// LEAdvertisingReport implements LE Advertising Report (0x3E:0x02) [Vol 2, Part E, 7.7.65.2].
type LEAdvertisingReport []byte

const LEConnectionUpdateCompleteCode = 0x0E

const LEConnectionUpdateCompleteSubCode = 0x03

// LEConnectionUpdateComplete implements LE Connection Update Complete (0x0E:0x03) [Vol 2, Part E, 7.7.65.3].
type LEConnectionUpdateComplete []byte

func (r LEConnectionUpdateComplete) SubeventCode() uint8 { return r[0] }

func (r LEConnectionUpdateComplete) Status() uint8 { return r[1] }

func (r LEConnectionUpdateComplete) ConnectionHandle() uint16 {
	return binary.LittleEndian.Uint16(r[2:])
}

func (r LEConnectionUpdateComplete) ConnInterval() uint16 { return binary.LittleEndian.Uint16(r[4:]) }

func (r LEConnectionUpdateComplete) ConnLatency() uint16 { return binary.LittleEndian.Uint16(r[6:]) }

func (r LEConnectionUpdateComplete) SupervisionTimeout() uint16 {
	return binary.LittleEndian.Uint16(r[8:])
}

const LEReadRemoteUsedFeaturesCompleteCode = 0x3E

const LEReadRemoteUsedFeaturesCompleteSubCode = 0x04

// LEReadRemoteUsedFeaturesComplete implements LE Read Remote Used Features Complete (0x3E:0x04) [Vol 2, Part E, 7.7.65.4].
type LEReadRemoteUsedFeaturesComplete []byte

func (r LEReadRemoteUsedFeaturesComplete) SubeventCode() uint8 { return r[0] }

func (r LEReadRemoteUsedFeaturesComplete) Status() uint8 { return r[1] }

func (r LEReadRemoteUsedFeaturesComplete) ConnectionHandle() uint16 {
	return binary.LittleEndian.Uint16(r[2:])
}

func (r LEReadRemoteUsedFeaturesComplete) LEFeatures() uint64 {
	return binary.LittleEndian.Uint64(r[4:])
}

const LELongTermKeyRequestCode = 0x3E

const LELongTermKeyRequestSubCode = 0x05

// LELongTermKeyRequest implements LE Long Term Key Request (0x3E:0x05) [Vol 2, Part E, 7.7.65.5].
type LELongTermKeyRequest []byte

func (r LELongTermKeyRequest) SubeventCode() uint8 { return r[0] }

func (r LELongTermKeyRequest) ConnectionHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

func (r LELongTermKeyRequest) RandomNumber() uint64 { return binary.LittleEndian.Uint64(r[3:]) }

func (r LELongTermKeyRequest) EncryptionDiversifier() uint16 {
	return binary.LittleEndian.Uint16(r[11:])
}

const LERemoteConnectionParameterRequestCode = 0x3E

const LERemoteConnectionParameterRequestSubCode = 0x06

// LERemoteConnectionParameterRequest implements LE Remote Connection Parameter Request (0x3E:0x06) [Vol 2, Part E, 7.7.65.6].
type LERemoteConnectionParameterRequest []byte

func (r LERemoteConnectionParameterRequest) SubeventCode() uint8 { return r[0] }

func (r LERemoteConnectionParameterRequest) ConnectionHandle() uint16 {
	return binary.LittleEndian.Uint16(r[1:])
}

func (r LERemoteConnectionParameterRequest) IntervalMin() uint16 {
	return binary.LittleEndian.Uint16(r[3:])
}

func (r LERemoteConnectionParameterRequest) IntervalMax() uint16 {
	return binary.LittleEndian.Uint16(r[5:])
}

func (r LERemoteConnectionParameterRequest) Latency() uint16 {
	return binary.LittleEndian.Uint16(r[7:])
}

func (r LERemoteConnectionParameterRequest) Timeout() uint16 {
	return binary.LittleEndian.Uint16(r[9:])
}

const AuthenticatedPayloadTimeoutExpiredCode = 0x57

// AuthenticatedPayloadTimeoutExpired implements Authenticated Payload Timeout Expired (0x57) [Vol 2, Part E, 7.7.75].
type AuthenticatedPayloadTimeoutExpired []byte

func (r AuthenticatedPayloadTimeoutExpired) ConnectionHandle() uint16 {
	return binary.LittleEndian.Uint16(r[0:])
}
