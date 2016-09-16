package hci

import "errors"

// errors
var (
	ErrBusyScanning    = errors.New("busy scanning")
	ErrBusyAdvertising = errors.New("busy advertising")
	ErrBusyDialing     = errors.New("busy dialing")
	ErrBusyListening   = errors.New("busy listening")
	ErrInvalidAddr     = errors.New("invalid address")
)

// HCI Command Errors  [Vol2, Part D, 1.3 ]
// FIXME: Terrible shorthand. Name them properly.
const (
	ErrUnknownCommand       ErrCommand = 0x01 // Unknown HCI Command
	ErrConnID               ErrCommand = 0x02 // Unknown Connection Identifier
	ErrHardware             ErrCommand = 0x03 // Hardware Failure
	ErrPageTimeout          ErrCommand = 0x04 // Page Timeout
	ErrAuth                 ErrCommand = 0x05 // Authentication Failure
	ErrPINMissing           ErrCommand = 0x06 // PIN or Key Missing
	ErrMemoryCapacity       ErrCommand = 0x07 // Memory Capacity Exceeded
	ErrConnTimeout          ErrCommand = 0x08 // Connection Timeout
	ErrConnLimit            ErrCommand = 0x09 // Connection Limit Exceeded
	ErrSCOConnLimit         ErrCommand = 0x0A // Synchronous Connection Limit To A Device Exceeded
	ErrACLConnExists        ErrCommand = 0x0B // ACL Connection Already Exists
	ErrDisallowed           ErrCommand = 0x0C // Command Disallowed
	ErrLimitedResource      ErrCommand = 0x0D // Connection Rejected due to Limited Resources
	ErrSecurity             ErrCommand = 0x0E // Connection Rejected Due To Security Reasons
	ErrBDADDR               ErrCommand = 0x0F // Connection Rejected due to Unacceptable BD_ADDR
	ErrConnAcceptTimeout    ErrCommand = 0x10 // Connection Accept Timeout Exceeded
	ErrUnsupportedParams    ErrCommand = 0x11 // Unsupported Feature or Parameter Value
	ErrInvalidParams        ErrCommand = 0x12 // Invalid HCI Command Parameters
	ErrRemoteUser           ErrCommand = 0x13 // Remote User Terminated Connection
	ErrRemoteLowResources   ErrCommand = 0x14 // Remote Device Terminated Connection due to Low Resources
	ErrRemotePowerOff       ErrCommand = 0x15 // Remote Device Terminated Connection due to Power Off
	ErrLocalHost            ErrCommand = 0x16 // Connection Terminated By Local Host
	ErrRepeatedAttempts     ErrCommand = 0x17 // Repeated Attempts
	ErrPairingNotAllowed    ErrCommand = 0x18 // Pairing Not Allowed
	ErrUnknownLMP           ErrCommand = 0x19 // Unknown LMP PDU
	ErrUnsupportedLMP       ErrCommand = 0x1A // Unsupported Remote Feature / Unsupported LMP Feature
	ErrSCOOffset            ErrCommand = 0x1B // SCO Offset Rejected
	ErrSCOInterval          ErrCommand = 0x1C // SCO Interval Rejected
	ErrSCOAirMode           ErrCommand = 0x1D // SCO Air Mode Rejected
	ErrInvalidLLParams      ErrCommand = 0x1E // Invalid LMP Parameters / Invalid LL Parameters
	ErrUnspecified          ErrCommand = 0x1F // Unspecified Error
	ErrUnsupportedLLParams  ErrCommand = 0x20 // Unsupported LMP Parameter Value / Unsupported LL Parameter Value
	ErrRoleChangeNotAllowed ErrCommand = 0x21 // Role Change Not Allowed
	ErrLLResponseTimeout    ErrCommand = 0x22 // LMP Response Timeout / LL Response Timeout
	ErrLMPTransColl         ErrCommand = 0x23 // LMP Error Transaction Collision
	ErrLMPPDU               ErrCommand = 0x24 // LMP PDU Not Allowed
	ErrEncNotAccepted       ErrCommand = 0x25 // Encryption Mode Not Acceptable
	ErrLinkKey              ErrCommand = 0x26 // Link Key cannot be Changed
	ErrQoSNotSupported      ErrCommand = 0x27 // Requested QoS Not Supported
	ErrInstantPassed        ErrCommand = 0x28 // Instant Passed
	ErrUnitKeyNotSupported  ErrCommand = 0x29 // Pairing With Unit Key Not Supported
	ErrDifferentTransColl   ErrCommand = 0x2A // Different Transaction Collision
	ErrQOSParameter         ErrCommand = 0x2C // QoS Unacceptable Parameter
	ErrQOSReject            ErrCommand = 0x2D // QoS Rejected
	ErrChannelClass         ErrCommand = 0x2E // Channel Classification Not Supported
	ErrInsufficientSecurity ErrCommand = 0x2F // Insufficient Security
	ErrOutOfRange           ErrCommand = 0x30 // Parameter Out Of Mandatory Range
	ErrRoleSwitchPending    ErrCommand = 0x32 // Role Switch Pending
	ErrReservedSlot         ErrCommand = 0x34 // Reserved Slot Violation
	ErrRoleSwitch           ErrCommand = 0x35 // Role Switch Failed
	ErrEIRTooLarge          ErrCommand = 0x36 // Extended Inquiry Response Too Large
	ErrSecureSimplePairing  ErrCommand = 0x37 // Secure Simple Pairing Not Supported By Host
	ErrHostBusy             ErrCommand = 0x38 // Host Busy - Pairing
	ErrNoChannel            ErrCommand = 0x39 // Connection Rejected due to No Suitable Channel Found
	ErrControllerBusy       ErrCommand = 0x3A // Controller Busy
	ErrConnParams           ErrCommand = 0x3B // Unacceptable Connection Parameters
	ErrDirAdvTimeout        ErrCommand = 0x3C // Directed Advertising Timeout
	ErrMIC                  ErrCommand = 0x3D // Connection Terminated due to MIC Failure
	ErrEstablished          ErrCommand = 0x3E // Connection Failed to be Established
	ErrMACConn              ErrCommand = 0x3F // MAC Connection Failed
	ErrCoarseClock          ErrCommand = 0x40 // Coarse Clock Adjustment Rejected but Will Try to Adjust Using Clock Dragging
	// 0x2B // Reserved
	// 0x31 // Reserved
	// 0x33 // Reserved
)

// ErrCommand [Vol2, Part D, 1.3 ]
type ErrCommand byte

func (e ErrCommand) Error() string {
	if s, ok := errCmd[e]; ok {
		return s
	}
	// A Host shall consider any error code that it does not explicitly
	// understand equivalent to the “Unspecified Error (0x1F).”
	return errCmd[0x1F]
}

var errCmd = map[ErrCommand]string{
	0x00: "Success",
	0x01: "Unknown HCI Command",
	0x02: "Unknown Connection Identifier",
	0x03: "Hardware Failure",
	0x04: "Page Timeou",
	0x05: "Authentication Failure",
	0x06: "PIN or Key Missing",
	0x07: "Memory Capacity Exceeded",
	0x08: "Connection Timeout",
	0x09: "Connection Limit Exceeded",
	0x0A: "Synchronous Connection Limit To A Device Exceeded",
	0x0B: "ACL Connection Already Exists",
	0x0C: "Command Disallowed",
	0x0D: "Connection Rejected due to Limited Resources",
	0x0E: "Connection Rejected Due To Security Reasons",
	0x0F: "Connection Rejected due to Unacceptable BD_ADDR",
	0x10: "Connection Accept Timeout Exceeded",
	0x11: "Unsupported Feature or Parameter Value",
	0x12: "Invalid HCI Command Parameters",
	0x13: "Remote User Terminated Connection",
	0x14: "Remote Device Terminated Connection due to Low Resources",
	0x15: "Remote Device Terminated Connection due to Power Off",
	0x16: "Connection Terminated By Local Host",
	0x17: "Repeated Attempts",
	0x18: "Pairing Not Allowed",
	0x19: "Unknown LMP PDU",
	0x1A: "Unsupported Remote Feature / Unsupported LMP Feature",
	0x1B: "SCO Offset Rejected",
	0x1C: "SCO Interval Rejected",
	0x1D: "SCO Air Mode Rejected",
	0x1E: "Invalid LMP Parameters / Invalid LL Parameters",
	0x1F: "Unspecified Error",
	0x20: "Unsupported LMP Parameter Value / Unsupported LL Parameter Value",
	0x21: "Role Change Not Allowed",
	0x22: "LMP Response Timeout / LL Response Timeout",
	0x23: "LMP Error Transaction Collision",
	0x24: "LMP PDU Not Allowed",
	0x25: "Encryption Mode Not Acceptable",
	0x26: "Link Key cannot be Changed",
	0x27: "Requested QoS Not Supported",
	0x28: "Instant Passed",
	0x29: "Pairing With Unit Key Not Supported",
	0x2A: "Different Transaction Collision",
	0x2B: "Reserved",
	0x2C: "QoS Unacceptable Parameter",
	0x2D: "QoS Rejected",
	0x2E: "Channel Classification Not Supported",
	0x2F: "Insufficient Security",
	0x30: "Parameter Out Of Mandatory Range",
	0x31: "Reserved",
	0x32: "Role Switch Pending",
	0x33: "Reserved",
	0x34: "Reserved Slot Violation",
	0x35: "Role Switch Failed",
	0x36: "Extended Inquiry Response Too Large",
	0x37: "Secure Simple Pairing Not Supported By Host",
	0x38: "Host Busy - Pairing",
	0x39: "Connection Rejected due to No Suitable Channel Found",
	0x3A: "Controller Busy",
	0x3B: "Unacceptable Connection Parameters",
	0x3C: "Directed Advertising Timeout",
	0x3D: "Connection Terminated due to MIC Failure",
	0x3E: "Connection Failed to be Established",
	0x3F: "MAC Connection Failed",
	0x40: "Coarse Clock Adjustment Rejected but Will Try to Adjust Using Clock Dragging",
}
