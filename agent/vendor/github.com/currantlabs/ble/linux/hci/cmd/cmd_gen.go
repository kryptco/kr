package cmd

// Disconnect implements Disconnect (0x01|0x0006) [Vol 2, Part E, 7.1.6]
type Disconnect struct {
	ConnectionHandle uint16
	Reason           uint8
}

func (c *Disconnect) String() string {
	return "Disconnect (0x01|0x0006)"
}

// OpCode returns the opcode of the command.
func (c *Disconnect) OpCode() int { return 0x01<<10 | 0x0006 }

// Len returns the length of the command.
func (c *Disconnect) Len() int { return 3 }

// Marshal serializes the command parameters into binary form.
func (c *Disconnect) Marshal(b []byte) error {
	return marshal(c, b)
}

// ReadRemoteVersionInformation implements Read Remote Version Information (0x01|0x001D) [Vol 2, Part E, 7.1.23]
type ReadRemoteVersionInformation struct {
	ConnectionHandle uint16
}

func (c *ReadRemoteVersionInformation) String() string {
	return "Read Remote Version Information (0x01|0x001D)"
}

// OpCode returns the opcode of the command.
func (c *ReadRemoteVersionInformation) OpCode() int { return 0x01<<10 | 0x001D }

// Len returns the length of the command.
func (c *ReadRemoteVersionInformation) Len() int { return 2 }

// Marshal serializes the command parameters into binary form.
func (c *ReadRemoteVersionInformation) Marshal(b []byte) error {
	return marshal(c, b)
}

// WriteDefaultLinkPolicySettings implements Write Default Link Policy Settings (0x02|0x000D) [Vol 2, Part E, 7.2.12]
type WriteDefaultLinkPolicySettings struct {
	DefaultLinkPolicySettings uint16
}

func (c *WriteDefaultLinkPolicySettings) String() string {
	return "Write Default Link Policy Settings (0x02|0x000D)"
}

// OpCode returns the opcode of the command.
func (c *WriteDefaultLinkPolicySettings) OpCode() int { return 0x02<<10 | 0x000D }

// Len returns the length of the command.
func (c *WriteDefaultLinkPolicySettings) Len() int { return 2 }

// Marshal serializes the command parameters into binary form.
func (c *WriteDefaultLinkPolicySettings) Marshal(b []byte) error {
	return marshal(c, b)
}

// WriteDefaultLinkPolicySettingsRP returns the return parameter of Write Default Link Policy Settings
type WriteDefaultLinkPolicySettingsRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *WriteDefaultLinkPolicySettingsRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// SetEventMask implements Set Event Mask (0x03|0x0001) [Vol 2, Part E, 7.3.1]
type SetEventMask struct {
	EventMask uint64
}

func (c *SetEventMask) String() string {
	return "Set Event Mask (0x03|0x0001)"
}

// OpCode returns the opcode of the command.
func (c *SetEventMask) OpCode() int { return 0x03<<10 | 0x0001 }

// Len returns the length of the command.
func (c *SetEventMask) Len() int { return 8 }

// Marshal serializes the command parameters into binary form.
func (c *SetEventMask) Marshal(b []byte) error {
	return marshal(c, b)
}

// SetEventMaskRP returns the return parameter of Set Event Mask
type SetEventMaskRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *SetEventMaskRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// Reset implements Reset (0x03|0x003) [Vol 2, Part E, 7.3.2]
type Reset struct {
}

func (c *Reset) String() string {
	return "Reset (0x03|0x003)"
}

// OpCode returns the opcode of the command.
func (c *Reset) OpCode() int { return 0x03<<10 | 0x003 }

// Len returns the length of the command.
func (c *Reset) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *Reset) Marshal(b []byte) error {
	return marshal(c, b)
}

// ResetRP returns the return parameter of Reset
type ResetRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *ResetRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// WritePageTimeout implements Write Page Timeout (0x03|0x0018) [Vol 2, Part E, 7.3.16]
type WritePageTimeout struct {
	PageTimeout uint16
}

func (c *WritePageTimeout) String() string {
	return "Write Page Timeout (0x03|0x0018)"
}

// OpCode returns the opcode of the command.
func (c *WritePageTimeout) OpCode() int { return 0x03<<10 | 0x0018 }

// Len returns the length of the command.
func (c *WritePageTimeout) Len() int { return 2 }

// Marshal serializes the command parameters into binary form.
func (c *WritePageTimeout) Marshal(b []byte) error {
	return marshal(c, b)
}

// WritePageTimeoutRP returns the return parameter of Write Page Timeout
type WritePageTimeoutRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *WritePageTimeoutRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// WriteClassOfDevice implements Write Class Of Device (0x03|0x0024) [Vol 2, Part E, 7.3.26]
type WriteClassOfDevice struct {
	ClassOfDevice [3]byte
}

func (c *WriteClassOfDevice) String() string {
	return "Write Class Of Device (0x03|0x0024)"
}

// OpCode returns the opcode of the command.
func (c *WriteClassOfDevice) OpCode() int { return 0x03<<10 | 0x0024 }

// Len returns the length of the command.
func (c *WriteClassOfDevice) Len() int { return 3 }

// Marshal serializes the command parameters into binary form.
func (c *WriteClassOfDevice) Marshal(b []byte) error {
	return marshal(c, b)
}

// WriteClassOfDeviceRP returns the return parameter of Write Class Of Device
type WriteClassOfDeviceRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *WriteClassOfDeviceRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// ReadTransmitPowerLevel implements Read Transmit Power Level (0x03|0x002D) [Vol 2, Part E, 7.3.35]
type ReadTransmitPowerLevel struct {
	ConnectionHandle uint16
	Type             uint8
}

func (c *ReadTransmitPowerLevel) String() string {
	return "Read Transmit Power Level (0x03|0x002D)"
}

// OpCode returns the opcode of the command.
func (c *ReadTransmitPowerLevel) OpCode() int { return 0x03<<10 | 0x002D }

// Len returns the length of the command.
func (c *ReadTransmitPowerLevel) Len() int { return 3 }

// Marshal serializes the command parameters into binary form.
func (c *ReadTransmitPowerLevel) Marshal(b []byte) error {
	return marshal(c, b)
}

// ReadTransmitPowerLevelRP returns the return parameter of Read Transmit Power Level
type ReadTransmitPowerLevelRP struct {
	Status           uint8
	ConnectionHandle uint16
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *ReadTransmitPowerLevelRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// HostBufferSize implements Host Buffer Size (0x03|0x0033) [Vol 2, Part E, 7.3.39]
type HostBufferSize struct {
	HostACLDataPacketLength            uint16
	HostSynchronousDataPacketLength    uint8
	HostTotalNumACLDataPackets         uint16
	HostTotalNumSynchronousDataPackets uint16
}

func (c *HostBufferSize) String() string {
	return "Host Buffer Size (0x03|0x0033)"
}

// OpCode returns the opcode of the command.
func (c *HostBufferSize) OpCode() int { return 0x03<<10 | 0x0033 }

// Len returns the length of the command.
func (c *HostBufferSize) Len() int { return 7 }

// Marshal serializes the command parameters into binary form.
func (c *HostBufferSize) Marshal(b []byte) error {
	return marshal(c, b)
}

// HostBufferSizeRP returns the return parameter of Host Buffer Size
type HostBufferSizeRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *HostBufferSizeRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// HostNumberOfCompletedPackets implements Host Number Of Completed Packets (0x03|0x0035) [Vol 2, Part E, 7.3.40]
type HostNumberOfCompletedPackets struct {
	NumberOfHandles           uint8
	ConnectionHandle          []uint16
	HostNumOfCompletedPackets []uint16
}

func (c *HostNumberOfCompletedPackets) String() string {
	return "Host Number Of Completed Packets (0x03|0x0035)"
}

// OpCode returns the opcode of the command.
func (c *HostNumberOfCompletedPackets) OpCode() int { return 0x03<<10 | 0x0035 }

// Len returns the length of the command.
func (c *HostNumberOfCompletedPackets) Len() int { return -1 }

// SetEventMaskPage2 implements Set Event Mask Page 2 (0x03|0x0063) [Vol 2, Part E, 7.3.69]
type SetEventMaskPage2 struct {
	EventMaskPage2 uint64
}

func (c *SetEventMaskPage2) String() string {
	return "Set Event Mask Page 2 (0x03|0x0063)"
}

// OpCode returns the opcode of the command.
func (c *SetEventMaskPage2) OpCode() int { return 0x03<<10 | 0x0063 }

// Len returns the length of the command.
func (c *SetEventMaskPage2) Len() int { return 8 }

// Marshal serializes the command parameters into binary form.
func (c *SetEventMaskPage2) Marshal(b []byte) error {
	return marshal(c, b)
}

// SetEventMaskPage2RP returns the return parameter of Set Event Mask Page 2
type SetEventMaskPage2RP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *SetEventMaskPage2RP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// WriteLEHostSupport implements Write LE Host Support (0x03|0x006D) [Vol 2, Part E, 7.3.79]
type WriteLEHostSupport struct {
	LESupportedHost    uint8
	SimultaneousLEHost uint8
}

func (c *WriteLEHostSupport) String() string {
	return "Write LE Host Support (0x03|0x006D)"
}

// OpCode returns the opcode of the command.
func (c *WriteLEHostSupport) OpCode() int { return 0x03<<10 | 0x006D }

// Len returns the length of the command.
func (c *WriteLEHostSupport) Len() int { return 2 }

// Marshal serializes the command parameters into binary form.
func (c *WriteLEHostSupport) Marshal(b []byte) error {
	return marshal(c, b)
}

// WriteLEHostSupportRP returns the return parameter of Write LE Host Support
type WriteLEHostSupportRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *WriteLEHostSupportRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// ReadAuthenticatedPayloadTimeout implements Read Authenticated Payload Timeout (0x03|0x007B) [Vol 2, Part E, 7.3.93]
type ReadAuthenticatedPayloadTimeout struct {
	ConnectionHandle uint16
}

func (c *ReadAuthenticatedPayloadTimeout) String() string {
	return "Read Authenticated Payload Timeout (0x03|0x007B)"
}

// OpCode returns the opcode of the command.
func (c *ReadAuthenticatedPayloadTimeout) OpCode() int { return 0x03<<10 | 0x007B }

// Len returns the length of the command.
func (c *ReadAuthenticatedPayloadTimeout) Len() int { return 2 }

// Marshal serializes the command parameters into binary form.
func (c *ReadAuthenticatedPayloadTimeout) Marshal(b []byte) error {
	return marshal(c, b)
}

// ReadAuthenticatedPayloadTimeoutRP returns the return parameter of Read Authenticated Payload Timeout
type ReadAuthenticatedPayloadTimeoutRP struct {
	Status                      uint8
	ConnectionHandle            uint16
	AuthenticatedPayloadTimeout uint16
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *ReadAuthenticatedPayloadTimeoutRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// WriteAuthenticatedPayloadTimeout implements Write Authenticated Payload Timeout (0x01|0x007C) [Vol 2, Part E, 7.3.94]
type WriteAuthenticatedPayloadTimeout struct {
	ConnectionHandle            uint16
	AuthenticatedPayloadTimeout uint16
}

func (c *WriteAuthenticatedPayloadTimeout) String() string {
	return "Write Authenticated Payload Timeout (0x01|0x007C)"
}

// OpCode returns the opcode of the command.
func (c *WriteAuthenticatedPayloadTimeout) OpCode() int { return 0x01<<10 | 0x007C }

// Len returns the length of the command.
func (c *WriteAuthenticatedPayloadTimeout) Len() int { return 4 }

// Marshal serializes the command parameters into binary form.
func (c *WriteAuthenticatedPayloadTimeout) Marshal(b []byte) error {
	return marshal(c, b)
}

// WriteAuthenticatedPayloadTimeoutRP returns the return parameter of Write Authenticated Payload Timeout
type WriteAuthenticatedPayloadTimeoutRP struct {
	Status           uint8
	ConnectionHandle uint16
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *WriteAuthenticatedPayloadTimeoutRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// ReadLocalVersionInformation implements Read Local Version Information (0x04|0x0001) [Vol 2, Part E, 7.4.1]
type ReadLocalVersionInformation struct {
}

func (c *ReadLocalVersionInformation) String() string {
	return "Read Local Version Information (0x04|0x0001)"
}

// OpCode returns the opcode of the command.
func (c *ReadLocalVersionInformation) OpCode() int { return 0x04<<10 | 0x0001 }

// Len returns the length of the command.
func (c *ReadLocalVersionInformation) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *ReadLocalVersionInformation) Marshal(b []byte) error {
	return marshal(c, b)
}

// ReadLocalVersionInformationRP returns the return parameter of Read Local Version Information
type ReadLocalVersionInformationRP struct {
	Status           uint8
	HCIVersion       uint8
	HCIRevision      uint16
	LMPPAMVersion    uint8
	ManufacturerName uint16
	LMPPAMSubversion uint16
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *ReadLocalVersionInformationRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// ReadLocalSupportedCommands implements Read Local Supported Commands (0x04|0x0002) [Vol 2, Part E, 7.4.2]
type ReadLocalSupportedCommands struct {
}

func (c *ReadLocalSupportedCommands) String() string {
	return "Read Local Supported Commands (0x04|0x0002)"
}

// OpCode returns the opcode of the command.
func (c *ReadLocalSupportedCommands) OpCode() int { return 0x04<<10 | 0x0002 }

// Len returns the length of the command.
func (c *ReadLocalSupportedCommands) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *ReadLocalSupportedCommands) Marshal(b []byte) error {
	return marshal(c, b)
}

// ReadLocalSupportedCommandsRP returns the return parameter of Read Local Supported Commands
type ReadLocalSupportedCommandsRP struct {
	Status     uint8
	Supporteds uint64
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *ReadLocalSupportedCommandsRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// ReadLocalSupportedFeatures implements Read Local Supported Features (0x04|0x0003) [Vol 2, Part E, 7.4.3]
type ReadLocalSupportedFeatures struct {
}

func (c *ReadLocalSupportedFeatures) String() string {
	return "Read Local Supported Features (0x04|0x0003)"
}

// OpCode returns the opcode of the command.
func (c *ReadLocalSupportedFeatures) OpCode() int { return 0x04<<10 | 0x0003 }

// Len returns the length of the command.
func (c *ReadLocalSupportedFeatures) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *ReadLocalSupportedFeatures) Marshal(b []byte) error {
	return marshal(c, b)
}

// ReadLocalSupportedFeaturesRP returns the return parameter of Read Local Supported Features
type ReadLocalSupportedFeaturesRP struct {
	Status      uint8
	LMPFeatures uint64
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *ReadLocalSupportedFeaturesRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// ReadBufferSize implements Read Buffer Size (0x04|0x0005) [Vol 2, Part E, 7.4.5]
type ReadBufferSize struct {
}

func (c *ReadBufferSize) String() string {
	return "Read Buffer Size (0x04|0x0005)"
}

// OpCode returns the opcode of the command.
func (c *ReadBufferSize) OpCode() int { return 0x04<<10 | 0x0005 }

// Len returns the length of the command.
func (c *ReadBufferSize) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *ReadBufferSize) Marshal(b []byte) error {
	return marshal(c, b)
}

// ReadBufferSizeRP returns the return parameter of Read Buffer Size
type ReadBufferSizeRP struct {
	Status                           uint8
	HCACLDataPacketLength            uint16
	HCSynchronousDataPacketLength    uint8
	HCTotalNumACLDataPackets         uint16
	HCTotalNumSynchronousDataPackets uint16
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *ReadBufferSizeRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// ReadBDADDR implements Read BD_ADDR (0x04|0x0009) [Vol 2, Part E, 7.4.6]
type ReadBDADDR struct {
}

func (c *ReadBDADDR) String() string {
	return "Read BD_ADDR (0x04|0x0009)"
}

// OpCode returns the opcode of the command.
func (c *ReadBDADDR) OpCode() int { return 0x04<<10 | 0x0009 }

// Len returns the length of the command.
func (c *ReadBDADDR) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *ReadBDADDR) Marshal(b []byte) error {
	return marshal(c, b)
}

// ReadBDADDRRP returns the return parameter of Read BD_ADDR
type ReadBDADDRRP struct {
	Status uint8
	BDADDR [6]byte
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *ReadBDADDRRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// ReadRSSI implements Read RSSI (0x05|0x0005) [Vol 2, Part E, 7.5.4]
type ReadRSSI struct {
	Handle uint16
}

func (c *ReadRSSI) String() string {
	return "Read RSSI (0x05|0x0005)"
}

// OpCode returns the opcode of the command.
func (c *ReadRSSI) OpCode() int { return 0x05<<10 | 0x0005 }

// Len returns the length of the command.
func (c *ReadRSSI) Len() int { return 2 }

// Marshal serializes the command parameters into binary form.
func (c *ReadRSSI) Marshal(b []byte) error {
	return marshal(c, b)
}

// ReadRSSIRP returns the return parameter of Read RSSI
type ReadRSSIRP struct {
	Status           uint8
	ConnectionHandle uint16
	RSSI             int8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *ReadRSSIRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LESetEventMask implements LE Set Event Mask (0x08|0x0001) [Vol 2, Part E, 7.8.1]
type LESetEventMask struct {
	LEEventMask uint64
}

func (c *LESetEventMask) String() string {
	return "LE Set Event Mask (0x08|0x0001)"
}

// OpCode returns the opcode of the command.
func (c *LESetEventMask) OpCode() int { return 0x08<<10 | 0x0001 }

// Len returns the length of the command.
func (c *LESetEventMask) Len() int { return 8 }

// Marshal serializes the command parameters into binary form.
func (c *LESetEventMask) Marshal(b []byte) error {
	return marshal(c, b)
}

// LESetEventMaskRP returns the return parameter of LE Set Event Mask
type LESetEventMaskRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LESetEventMaskRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LEReadBufferSize implements LE Read Buffer Size (0x08|0x0002) [Vol 2, Part E, 7.8.2]
type LEReadBufferSize struct {
}

func (c *LEReadBufferSize) String() string {
	return "LE Read Buffer Size (0x08|0x0002)"
}

// OpCode returns the opcode of the command.
func (c *LEReadBufferSize) OpCode() int { return 0x08<<10 | 0x0002 }

// Len returns the length of the command.
func (c *LEReadBufferSize) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *LEReadBufferSize) Marshal(b []byte) error {
	return marshal(c, b)
}

// LEReadBufferSizeRP returns the return parameter of LE Read Buffer Size
type LEReadBufferSizeRP struct {
	Status                  uint8
	HCLEDataPacketLength    uint16
	HCTotalNumLEDataPackets uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LEReadBufferSizeRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LEReadLocalSupportedFeatures implements LE Read Local Supported Features (0x08|0x0003) [Vol 2, Part E, 7.8.3]
type LEReadLocalSupportedFeatures struct {
}

func (c *LEReadLocalSupportedFeatures) String() string {
	return "LE Read Local Supported Features (0x08|0x0003)"
}

// OpCode returns the opcode of the command.
func (c *LEReadLocalSupportedFeatures) OpCode() int { return 0x08<<10 | 0x0003 }

// Len returns the length of the command.
func (c *LEReadLocalSupportedFeatures) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *LEReadLocalSupportedFeatures) Marshal(b []byte) error {
	return marshal(c, b)
}

// LEReadLocalSupportedFeaturesRP returns the return parameter of LE Read Local Supported Features
type LEReadLocalSupportedFeaturesRP struct {
	Status     uint8
	LEFeatures uint64
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LEReadLocalSupportedFeaturesRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LESetRandomAddress implements LE Set Random Address (0x08|0x0005) [Vol 2, Part E, 7.8.4]
type LESetRandomAddress struct {
	RandomAddress [6]byte
}

func (c *LESetRandomAddress) String() string {
	return "LE Set Random Address (0x08|0x0005)"
}

// OpCode returns the opcode of the command.
func (c *LESetRandomAddress) OpCode() int { return 0x08<<10 | 0x0005 }

// Len returns the length of the command.
func (c *LESetRandomAddress) Len() int { return 6 }

// Marshal serializes the command parameters into binary form.
func (c *LESetRandomAddress) Marshal(b []byte) error {
	return marshal(c, b)
}

// LESetRandomAddressRP returns the return parameter of LE Set Random Address
type LESetRandomAddressRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LESetRandomAddressRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LESetAdvertisingParameters implements LE Set Advertising Parameters (0x08|0x0006) [Vol 2, Part E, 7.8.5]
type LESetAdvertisingParameters struct {
	AdvertisingIntervalMin  uint16
	AdvertisingIntervalMax  uint16
	AdvertisingType         uint8
	OwnAddressType          uint8
	DirectAddressType       uint8
	DirectAddress           [6]byte
	AdvertisingChannelMap   uint8
	AdvertisingFilterPolicy uint8
}

func (c *LESetAdvertisingParameters) String() string {
	return "LE Set Advertising Parameters (0x08|0x0006)"
}

// OpCode returns the opcode of the command.
func (c *LESetAdvertisingParameters) OpCode() int { return 0x08<<10 | 0x0006 }

// Len returns the length of the command.
func (c *LESetAdvertisingParameters) Len() int { return 15 }

// Marshal serializes the command parameters into binary form.
func (c *LESetAdvertisingParameters) Marshal(b []byte) error {
	return marshal(c, b)
}

// LESetAdvertisingParametersRP returns the return parameter of LE Set Advertising Parameters
type LESetAdvertisingParametersRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LESetAdvertisingParametersRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LEReadAdvertisingChannelTxPower implements LE Read Advertising Channel Tx Power (0x08|0x0007) [Vol 2, Part E, 7.8.6]
type LEReadAdvertisingChannelTxPower struct {
}

func (c *LEReadAdvertisingChannelTxPower) String() string {
	return "LE Read Advertising Channel Tx Power (0x08|0x0007)"
}

// OpCode returns the opcode of the command.
func (c *LEReadAdvertisingChannelTxPower) OpCode() int { return 0x08<<10 | 0x0007 }

// Len returns the length of the command.
func (c *LEReadAdvertisingChannelTxPower) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *LEReadAdvertisingChannelTxPower) Marshal(b []byte) error {
	return marshal(c, b)
}

// LEReadAdvertisingChannelTxPowerRP returns the return parameter of LE Read Advertising Channel Tx Power
type LEReadAdvertisingChannelTxPowerRP struct {
	Status             uint8
	TransmitPowerLevel uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LEReadAdvertisingChannelTxPowerRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LESetAdvertisingData implements LE Set Advertising Data (0x08|0x0008) [Vol 2, Part E, 7.8.7]
type LESetAdvertisingData struct {
	AdvertisingDataLength uint8
	AdvertisingData       [31]byte
}

func (c *LESetAdvertisingData) String() string {
	return "LE Set Advertising Data (0x08|0x0008)"
}

// OpCode returns the opcode of the command.
func (c *LESetAdvertisingData) OpCode() int { return 0x08<<10 | 0x0008 }

// Len returns the length of the command.
func (c *LESetAdvertisingData) Len() int { return 32 }

// Marshal serializes the command parameters into binary form.
func (c *LESetAdvertisingData) Marshal(b []byte) error {
	return marshal(c, b)
}

// LESetAdvertisingDataRP returns the return parameter of LE Set Advertising Data
type LESetAdvertisingDataRP struct {
	Status                  uint8
	HCLEDataPacketLength    uint16
	HCTotalNumLEDataPackets uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LESetAdvertisingDataRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LESetScanResponseData implements LE Set Scan Response Data (0x08|0x0009) [Vol 2, Part E, 7.8.8]
type LESetScanResponseData struct {
	ScanResponseDataLength uint8
	ScanResponseData       [31]byte
}

func (c *LESetScanResponseData) String() string {
	return "LE Set Scan Response Data (0x08|0x0009)"
}

// OpCode returns the opcode of the command.
func (c *LESetScanResponseData) OpCode() int { return 0x08<<10 | 0x0009 }

// Len returns the length of the command.
func (c *LESetScanResponseData) Len() int { return 32 }

// Marshal serializes the command parameters into binary form.
func (c *LESetScanResponseData) Marshal(b []byte) error {
	return marshal(c, b)
}

// LESetScanResponseDataRP returns the return parameter of LE Set Scan Response Data
type LESetScanResponseDataRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LESetScanResponseDataRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LESetAdvertiseEnable implements LE Set Advertise Enable (0x08|0x000A) [Vol 2, Part E, 7.8.9]
type LESetAdvertiseEnable struct {
	AdvertisingEnable uint8
}

func (c *LESetAdvertiseEnable) String() string {
	return "LE Set Advertise Enable (0x08|0x000A)"
}

// OpCode returns the opcode of the command.
func (c *LESetAdvertiseEnable) OpCode() int { return 0x08<<10 | 0x000A }

// Len returns the length of the command.
func (c *LESetAdvertiseEnable) Len() int { return 1 }

// Marshal serializes the command parameters into binary form.
func (c *LESetAdvertiseEnable) Marshal(b []byte) error {
	return marshal(c, b)
}

// LESetAdvertiseEnableRP returns the return parameter of LE Set Advertise Enable
type LESetAdvertiseEnableRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LESetAdvertiseEnableRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LESetScanParameters implements LE Set Scan Parameters (0x08|0x000B) [Vol 2, Part E, 7.8.10]
type LESetScanParameters struct {
	LEScanType           uint8
	LEScanInterval       uint16
	LEScanWindow         uint16
	OwnAddressType       uint8
	ScanningFilterPolicy uint8
}

func (c *LESetScanParameters) String() string {
	return "LE Set Scan Parameters (0x08|0x000B)"
}

// OpCode returns the opcode of the command.
func (c *LESetScanParameters) OpCode() int { return 0x08<<10 | 0x000B }

// Len returns the length of the command.
func (c *LESetScanParameters) Len() int { return 7 }

// Marshal serializes the command parameters into binary form.
func (c *LESetScanParameters) Marshal(b []byte) error {
	return marshal(c, b)
}

// LESetScanParametersRP returns the return parameter of LE Set Scan Parameters
type LESetScanParametersRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LESetScanParametersRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LESetScanEnable implements LE Set Scan Enable (0x08|0x000C) [Vol 2, Part E, 7.8.11]
type LESetScanEnable struct {
	LEScanEnable     uint8
	FilterDuplicates uint8
}

func (c *LESetScanEnable) String() string {
	return "LE Set Scan Enable (0x08|0x000C)"
}

// OpCode returns the opcode of the command.
func (c *LESetScanEnable) OpCode() int { return 0x08<<10 | 0x000C }

// Len returns the length of the command.
func (c *LESetScanEnable) Len() int { return 2 }

// Marshal serializes the command parameters into binary form.
func (c *LESetScanEnable) Marshal(b []byte) error {
	return marshal(c, b)
}

// LESetScanEnableRP returns the return parameter of LE Set Scan Enable
type LESetScanEnableRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LESetScanEnableRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LECreateConnection implements LE Create Connection (0x08|0x000D) [Vol 2, Part E, 7.8.12]
type LECreateConnection struct {
	LEScanInterval        uint16
	LEScanWindow          uint16
	InitiatorFilterPolicy uint8
	PeerAddressType       uint8
	PeerAddress           [6]byte
	OwnAddressType        uint8
	ConnIntervalMin       uint16
	ConnIntervalMax       uint16
	ConnLatency           uint16
	SupervisionTimeout    uint16
	MinimumCELength       uint16
	MaximumCELength       uint16
}

func (c *LECreateConnection) String() string {
	return "LE Create Connection (0x08|0x000D)"
}

// OpCode returns the opcode of the command.
func (c *LECreateConnection) OpCode() int { return 0x08<<10 | 0x000D }

// Len returns the length of the command.
func (c *LECreateConnection) Len() int { return 25 }

// Marshal serializes the command parameters into binary form.
func (c *LECreateConnection) Marshal(b []byte) error {
	return marshal(c, b)
}

// LECreateConnectionCancel implements LE Create Connection Cancel (0x08|0x000E) [Vol 2, Part E, 7.8.13]
type LECreateConnectionCancel struct {
}

func (c *LECreateConnectionCancel) String() string {
	return "LE Create Connection Cancel (0x08|0x000E)"
}

// OpCode returns the opcode of the command.
func (c *LECreateConnectionCancel) OpCode() int { return 0x08<<10 | 0x000E }

// Len returns the length of the command.
func (c *LECreateConnectionCancel) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *LECreateConnectionCancel) Marshal(b []byte) error {
	return marshal(c, b)
}

// LECreateConnectionCancelRP returns the return parameter of LE Create Connection Cancel
type LECreateConnectionCancelRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LECreateConnectionCancelRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LEReadWhiteListSize implements LE Read White List Size (0x08|0x000F) [Vol 2, Part E, 7.8.14]
type LEReadWhiteListSize struct {
}

func (c *LEReadWhiteListSize) String() string {
	return "LE Read White List Size (0x08|0x000F)"
}

// OpCode returns the opcode of the command.
func (c *LEReadWhiteListSize) OpCode() int { return 0x08<<10 | 0x000F }

// Len returns the length of the command.
func (c *LEReadWhiteListSize) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *LEReadWhiteListSize) Marshal(b []byte) error {
	return marshal(c, b)
}

// LEReadWhiteListSizeRP returns the return parameter of LE Read White List Size
type LEReadWhiteListSizeRP struct {
	Status        uint8
	WhiteListSize uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LEReadWhiteListSizeRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LEClearWhiteList implements LE Clear White List (0x08|0x0010) [Vol 2, Part E, 7.8.15]
type LEClearWhiteList struct {
}

func (c *LEClearWhiteList) String() string {
	return "LE Clear White List (0x08|0x0010)"
}

// OpCode returns the opcode of the command.
func (c *LEClearWhiteList) OpCode() int { return 0x08<<10 | 0x0010 }

// Len returns the length of the command.
func (c *LEClearWhiteList) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *LEClearWhiteList) Marshal(b []byte) error {
	return marshal(c, b)
}

// LEClearWhiteListRP returns the return parameter of LE Clear White List
type LEClearWhiteListRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LEClearWhiteListRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LEAddDeviceToWhiteList implements LE Add Device To White List (0x08|0x0011) [Vol 2, Part E, 7.8.16]
type LEAddDeviceToWhiteList struct {
	AddressType uint8
	Address     [6]byte
}

func (c *LEAddDeviceToWhiteList) String() string {
	return "LE Add Device To White List (0x08|0x0011)"
}

// OpCode returns the opcode of the command.
func (c *LEAddDeviceToWhiteList) OpCode() int { return 0x08<<10 | 0x0011 }

// Len returns the length of the command.
func (c *LEAddDeviceToWhiteList) Len() int { return 7 }

// Marshal serializes the command parameters into binary form.
func (c *LEAddDeviceToWhiteList) Marshal(b []byte) error {
	return marshal(c, b)
}

// LEAddDeviceToWhiteListRP returns the return parameter of LE Add Device To White List
type LEAddDeviceToWhiteListRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LEAddDeviceToWhiteListRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LERemoveDeviceFromWhiteList implements LE Remove Device From White List (0x08|0x0012) [Vol 2, Part E, 7.8.17]
type LERemoveDeviceFromWhiteList struct {
	AddressType uint8
	Address     [6]byte
}

func (c *LERemoveDeviceFromWhiteList) String() string {
	return "LE Remove Device From White List (0x08|0x0012)"
}

// OpCode returns the opcode of the command.
func (c *LERemoveDeviceFromWhiteList) OpCode() int { return 0x08<<10 | 0x0012 }

// Len returns the length of the command.
func (c *LERemoveDeviceFromWhiteList) Len() int { return 7 }

// Marshal serializes the command parameters into binary form.
func (c *LERemoveDeviceFromWhiteList) Marshal(b []byte) error {
	return marshal(c, b)
}

// LERemoveDeviceFromWhiteListRP returns the return parameter of LE Remove Device From White List
type LERemoveDeviceFromWhiteListRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LERemoveDeviceFromWhiteListRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LEConnectionUpdate implements LE Connection Update (0x08|0x0013) [Vol 2, Part E, 7.8.18]
type LEConnectionUpdate struct {
	ConnectionHandle   uint16
	ConnIntervalMin    uint16
	ConnIntervalMax    uint16
	ConnLatency        uint16
	SupervisionTimeout uint16
	MinimumCELength    uint16
	MaximumCELength    uint16
}

func (c *LEConnectionUpdate) String() string {
	return "LE Connection Update (0x08|0x0013)"
}

// OpCode returns the opcode of the command.
func (c *LEConnectionUpdate) OpCode() int { return 0x08<<10 | 0x0013 }

// Len returns the length of the command.
func (c *LEConnectionUpdate) Len() int { return 14 }

// Marshal serializes the command parameters into binary form.
func (c *LEConnectionUpdate) Marshal(b []byte) error {
	return marshal(c, b)
}

// LESetHostChannelClassification implements LE Set Host Channel Classification (0x08|0x0014) [Vol 2, Part E, 7.8.19]
type LESetHostChannelClassification struct {
	ChannelMap [5]byte
}

func (c *LESetHostChannelClassification) String() string {
	return "LE Set Host Channel Classification (0x08|0x0014)"
}

// OpCode returns the opcode of the command.
func (c *LESetHostChannelClassification) OpCode() int { return 0x08<<10 | 0x0014 }

// Len returns the length of the command.
func (c *LESetHostChannelClassification) Len() int { return 5 }

// Marshal serializes the command parameters into binary form.
func (c *LESetHostChannelClassification) Marshal(b []byte) error {
	return marshal(c, b)
}

// LESetHostChannelClassificationRP returns the return parameter of LE Set Host Channel Classification
type LESetHostChannelClassificationRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LESetHostChannelClassificationRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LEReadChannelMap implements LE Read Channel Map (0x08|0x0015) [Vol 2, Part E, 7.8.20]
type LEReadChannelMap struct {
	ConnectionHandle uint16
}

func (c *LEReadChannelMap) String() string {
	return "LE Read Channel Map (0x08|0x0015)"
}

// OpCode returns the opcode of the command.
func (c *LEReadChannelMap) OpCode() int { return 0x08<<10 | 0x0015 }

// Len returns the length of the command.
func (c *LEReadChannelMap) Len() int { return 2 }

// Marshal serializes the command parameters into binary form.
func (c *LEReadChannelMap) Marshal(b []byte) error {
	return marshal(c, b)
}

// LEReadChannelMapRP returns the return parameter of LE Read Channel Map
type LEReadChannelMapRP struct {
	Status           uint8
	ConnectionHandle uint16
	ChannelMap       [5]byte
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LEReadChannelMapRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LEReadRemoteUsedFeatures implements LE Read Remote Used Features (0x08|0x0016) [Vol 2, Part E, 7.8.21]
type LEReadRemoteUsedFeatures struct {
	ConnectionHandle uint16
}

func (c *LEReadRemoteUsedFeatures) String() string {
	return "LE Read Remote Used Features (0x08|0x0016)"
}

// OpCode returns the opcode of the command.
func (c *LEReadRemoteUsedFeatures) OpCode() int { return 0x08<<10 | 0x0016 }

// Len returns the length of the command.
func (c *LEReadRemoteUsedFeatures) Len() int { return 2 }

// Marshal serializes the command parameters into binary form.
func (c *LEReadRemoteUsedFeatures) Marshal(b []byte) error {
	return marshal(c, b)
}

// LEEncrypt implements LE Encrypt (0x08|0x0017) [Vol 2, Part E, 7.8.22]
type LEEncrypt struct {
	Key           [16]byte
	PlaintextData [16]byte
}

func (c *LEEncrypt) String() string {
	return "LE Encrypt (0x08|0x0017)"
}

// OpCode returns the opcode of the command.
func (c *LEEncrypt) OpCode() int { return 0x08<<10 | 0x0017 }

// Len returns the length of the command.
func (c *LEEncrypt) Len() int { return 32 }

// Marshal serializes the command parameters into binary form.
func (c *LEEncrypt) Marshal(b []byte) error {
	return marshal(c, b)
}

// LEEncryptRP returns the return parameter of LE Encrypt
type LEEncryptRP struct {
	Status        uint8
	EncryptedData [16]byte
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LEEncryptRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LERand implements LE Rand (0x08|0x0018) [Vol 2, Part E, 7.8.23]
type LERand struct {
}

func (c *LERand) String() string {
	return "LE Rand (0x08|0x0018)"
}

// OpCode returns the opcode of the command.
func (c *LERand) OpCode() int { return 0x08<<10 | 0x0018 }

// Len returns the length of the command.
func (c *LERand) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *LERand) Marshal(b []byte) error {
	return marshal(c, b)
}

// LERandRP returns the return parameter of LE Rand
type LERandRP struct {
	Status       uint8
	RandomNumber uint64
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LERandRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LEStartEncryption implements LE Start Encryption (0x08|0x0019) [Vol 2, Part E, 7.8.24]
type LEStartEncryption struct {
	ConnectionHandle     uint16
	RandomNumber         uint64
	EncryptedDiversifier uint16
	LongTermKey          [16]byte
}

func (c *LEStartEncryption) String() string {
	return "LE Start Encryption (0x08|0x0019)"
}

// OpCode returns the opcode of the command.
func (c *LEStartEncryption) OpCode() int { return 0x08<<10 | 0x0019 }

// Len returns the length of the command.
func (c *LEStartEncryption) Len() int { return 28 }

// Marshal serializes the command parameters into binary form.
func (c *LEStartEncryption) Marshal(b []byte) error {
	return marshal(c, b)
}

// LELongTermKeyRequestReply implements LE Long Term Key Request Reply (0x08|0x001A) [Vol 2, Part E, 7.8.25]
type LELongTermKeyRequestReply struct {
	ConnectionHandle uint16
	LongTermKey      [16]byte
}

func (c *LELongTermKeyRequestReply) String() string {
	return "LE Long Term Key Request Reply (0x08|0x001A)"
}

// OpCode returns the opcode of the command.
func (c *LELongTermKeyRequestReply) OpCode() int { return 0x08<<10 | 0x001A }

// Len returns the length of the command.
func (c *LELongTermKeyRequestReply) Len() int { return 18 }

// Marshal serializes the command parameters into binary form.
func (c *LELongTermKeyRequestReply) Marshal(b []byte) error {
	return marshal(c, b)
}

// LELongTermKeyRequestReplyRP returns the return parameter of LE Long Term Key Request Reply
type LELongTermKeyRequestReplyRP struct {
	Status           uint8
	ConnectionHandle uint16
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LELongTermKeyRequestReplyRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LELongTermKeyRequestNegativeReply implements LE Long Term Key Request Negative Reply (0x08|0x001B) [Vol 2, Part E, 7.8.26]
type LELongTermKeyRequestNegativeReply struct {
	ConnectionHandle uint16
}

func (c *LELongTermKeyRequestNegativeReply) String() string {
	return "LE Long Term Key Request Negative Reply (0x08|0x001B)"
}

// OpCode returns the opcode of the command.
func (c *LELongTermKeyRequestNegativeReply) OpCode() int { return 0x08<<10 | 0x001B }

// Len returns the length of the command.
func (c *LELongTermKeyRequestNegativeReply) Len() int { return 2 }

// Marshal serializes the command parameters into binary form.
func (c *LELongTermKeyRequestNegativeReply) Marshal(b []byte) error {
	return marshal(c, b)
}

// LELongTermKeyRequestNegativeReplyRP returns the return parameter of LE Long Term Key Request Negative Reply
type LELongTermKeyRequestNegativeReplyRP struct {
	Status           uint8
	ConnectionHandle uint16
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LELongTermKeyRequestNegativeReplyRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LEReadSupportedStates implements LE Read Supported States (0x08|0x001C) [Vol 2, Part E, 7.8.27]
type LEReadSupportedStates struct {
}

func (c *LEReadSupportedStates) String() string {
	return "LE Read Supported States (0x08|0x001C)"
}

// OpCode returns the opcode of the command.
func (c *LEReadSupportedStates) OpCode() int { return 0x08<<10 | 0x001C }

// Len returns the length of the command.
func (c *LEReadSupportedStates) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *LEReadSupportedStates) Marshal(b []byte) error {
	return marshal(c, b)
}

// LEReadSupportedStatesRP returns the return parameter of LE Read Supported States
type LEReadSupportedStatesRP struct {
	Status   uint8
	LEStates uint64
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LEReadSupportedStatesRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LEReceiverTest implements LE Receiver Test (0x08|0x001D) [Vol 2, Part E, 7.8.28]
type LEReceiverTest struct {
	RXChannel uint8
}

func (c *LEReceiverTest) String() string {
	return "LE Receiver Test (0x08|0x001D)"
}

// OpCode returns the opcode of the command.
func (c *LEReceiverTest) OpCode() int { return 0x08<<10 | 0x001D }

// Len returns the length of the command.
func (c *LEReceiverTest) Len() int { return 1 }

// Marshal serializes the command parameters into binary form.
func (c *LEReceiverTest) Marshal(b []byte) error {
	return marshal(c, b)
}

// LEReceiverTestRP returns the return parameter of LE Receiver Test
type LEReceiverTestRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LEReceiverTestRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LETransmitterTest implements LE Transmitter Test (0x08|0x001E) [Vol 2, Part E, 7.8.29]
type LETransmitterTest struct {
	TXChannel        uint8
	LengthOfTestData uint8
	PacketPayload    uint8
}

func (c *LETransmitterTest) String() string {
	return "LE Transmitter Test (0x08|0x001E)"
}

// OpCode returns the opcode of the command.
func (c *LETransmitterTest) OpCode() int { return 0x08<<10 | 0x001E }

// Len returns the length of the command.
func (c *LETransmitterTest) Len() int { return 3 }

// Marshal serializes the command parameters into binary form.
func (c *LETransmitterTest) Marshal(b []byte) error {
	return marshal(c, b)
}

// LETransmitterTestRP returns the return parameter of LE Transmitter Test
type LETransmitterTestRP struct {
	Status uint8
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LETransmitterTestRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LETestEnd implements LE Test End (0x08|0x001F) [Vol 2, Part E, 7.8.30]
type LETestEnd struct {
}

func (c *LETestEnd) String() string {
	return "LE Test End (0x08|0x001F)"
}

// OpCode returns the opcode of the command.
func (c *LETestEnd) OpCode() int { return 0x08<<10 | 0x001F }

// Len returns the length of the command.
func (c *LETestEnd) Len() int { return 0 }

// Marshal serializes the command parameters into binary form.
func (c *LETestEnd) Marshal(b []byte) error {
	return marshal(c, b)
}

// LETestEndRP returns the return parameter of LE Test End
type LETestEndRP struct {
	Status          uint8
	NumberOfPackats uint16
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LETestEndRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LERemoteConnectionParameterRequestReply implements LE Remote Connection Parameter Request Reply (0x08|0x0020) [Vol 2, Part E, 7.8.31]
type LERemoteConnectionParameterRequestReply struct {
	ConnectionHandle uint16
	IntervalMin      uint16
	IntervalMax      uint16
	Latency          uint16
	Timeout          uint16
	MinimumCELength  uint16
	MaximumCELength  uint16
}

func (c *LERemoteConnectionParameterRequestReply) String() string {
	return "LE Remote Connection Parameter Request Reply (0x08|0x0020)"
}

// OpCode returns the opcode of the command.
func (c *LERemoteConnectionParameterRequestReply) OpCode() int { return 0x08<<10 | 0x0020 }

// Len returns the length of the command.
func (c *LERemoteConnectionParameterRequestReply) Len() int { return 14 }

// Marshal serializes the command parameters into binary form.
func (c *LERemoteConnectionParameterRequestReply) Marshal(b []byte) error {
	return marshal(c, b)
}

// LERemoteConnectionParameterRequestReplyRP returns the return parameter of LE Remote Connection Parameter Request Reply
type LERemoteConnectionParameterRequestReplyRP struct {
	Status           uint8
	ConnectionHandle uint16
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LERemoteConnectionParameterRequestReplyRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}

// LERemoteConnectionParameterRequestNegativeReply implements LE Remote Connection Parameter Request Negative Reply (0x08|0x0021) [Vol 2, Part E, 7.8.32]
type LERemoteConnectionParameterRequestNegativeReply struct {
	ConnectionHandle uint16
	Reason           uint8
}

func (c *LERemoteConnectionParameterRequestNegativeReply) String() string {
	return "LE Remote Connection Parameter Request Negative Reply (0x08|0x0021)"
}

// OpCode returns the opcode of the command.
func (c *LERemoteConnectionParameterRequestNegativeReply) OpCode() int { return 0x08<<10 | 0x0021 }

// Len returns the length of the command.
func (c *LERemoteConnectionParameterRequestNegativeReply) Len() int { return 3 }

// Marshal serializes the command parameters into binary form.
func (c *LERemoteConnectionParameterRequestNegativeReply) Marshal(b []byte) error {
	return marshal(c, b)
}

// LERemoteConnectionParameterRequestNegativeReplyRP returns the return parameter of LE Remote Connection Parameter Request Negative Reply
type LERemoteConnectionParameterRequestNegativeReplyRP struct {
	Status           uint8
	ConnectionHandle uint16
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (c *LERemoteConnectionParameterRequestNegativeReplyRP) Unmarshal(b []byte) error {
	return unmarshal(c, b)
}
