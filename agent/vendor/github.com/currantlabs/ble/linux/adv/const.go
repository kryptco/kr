package adv

import "errors"

// MaxEIRPacketLength is the maximum allowed AdvertisingPacket
// and ScanResponsePacket length.
const MaxEIRPacketLength = 31

// ErrNotFit ...
var (
	ErrInvalid = errors.New("invalid argument")
	ErrNotFit  = errors.New("data not fit")
)

// Advertising flags
const (
	FlagLimitedDiscoverable = 0x01 // LE Limited Discoverable Mode
	FlagGeneralDiscoverable = 0x02 // LE General Discoverable Mode
	FlagLEOnly              = 0x04 // BR/EDR Not Supported. Bit 37 of LMP Feature Mask Definitions (Page 0)
	FlagBothController      = 0x08 // Simultaneous LE and BR/EDR to Same Device Capable (Controller).
	FlagBothHost            = 0x10 // Simultaneous LE and BR/EDR to Same Device Capable (Host).
)

// Advertising data field s
const (
	flags             = 0x01 // Flags
	someUUID16        = 0x02 // Incomplete List of 16-bit Service Class UUIDs
	allUUID16         = 0x03 // Complete List of 16-bit Service Class UUIDs
	someUUID32        = 0x04 // Incomplete List of 32-bit Service Class UUIDs
	allUUID32         = 0x05 // Complete List of 32-bit Service Class UUIDs
	someUUID128       = 0x06 // Incomplete List of 128-bit Service Class UUIDs
	allUUID128        = 0x07 // Complete List of 128-bit Service Class UUIDs
	shortName         = 0x08 // Shortened Local Name
	completeName      = 0x09 // Complete Local Name
	txPower           = 0x0A // Tx Power Level
	classOfDevice     = 0x0D // Class of Device
	simplePairingC192 = 0x0E // Simple Pairing Hash C-192
	simplePairingR192 = 0x0F // Simple Pairing Randomizer R-192
	secManagerTK      = 0x10 // Security Manager TK Value
	secManagerOOB     = 0x11 // Security Manager Out of Band Flags
	slaveConnInt      = 0x12 // Slave Connection Interval Range
	serviceSol16      = 0x14 // List of 16-bit Service Solicitation UUIDs
	serviceSol128     = 0x15 // List of 128-bit Service Solicitation UUIDs
	serviceData16     = 0x16 // Service Data - 16-bit UUID
	pubTargetAddr     = 0x17 // Public Target Address
	randTargetAddr    = 0x18 // Random Target Address
	appearance        = 0x19 // Appearance
	advInterval       = 0x1A // Advertising Interval
	leDeviceAddr      = 0x1B // LE Bluetooth Device Address
	leRole            = 0x1C // LE Role
	serviceSol32      = 0x1F // List of 32-bit Service Solicitation UUIDs
	serviceData32     = 0x20 // Service Data - 32-bit UUID
	serviceData128    = 0x21 // Service Data - 128-bit UUID
	leSecConfirm      = 0x22 // LE Secure Connections Confirmation Value
	leSecRandom       = 0x23 // LE Secure Connections Random Value
	manufacturerData  = 0xFF // Manufacturer Specific Data
)
