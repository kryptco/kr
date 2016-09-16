package ble

import (
	"errors"
	"fmt"
)

// ErrEIRPacketTooLong is the error returned when an AdvertisingPacket
// or ScanResponsePacket is too long.
var ErrEIRPacketTooLong = errors.New("max packet length is 31")

// ErrNotImplemented means the functionality is not implemented.
var ErrNotImplemented = errors.New("not implemented")

// ATTError is the error code of Attribute Protocol [Vol 3, Part F, 3.4.1.1].
type ATTError byte

// ATTError is the error code of Attribute Protocol [Vol 3, Part F, 3.4.1.1].
const (
	ErrSuccess           ATTError = 0x00 // ErrSuccess measn the operation is success.
	ErrInvalidHandle     ATTError = 0x01 // ErrInvalidHandle means the attribute handle given was not valid on this server.
	ErrReadNotPerm       ATTError = 0x02 // ErrReadNotPerm eans the attribute cannot be read.
	ErrWriteNotPerm      ATTError = 0x03 // ErrWriteNotPerm eans the attribute cannot be written.
	ErrInvalidPDU        ATTError = 0x04 // ErrInvalidPDU means the attribute PDU was invalid.
	ErrAuthentication    ATTError = 0x05 // ErrAuthentication means the attribute requires authentication before it can be read or written.
	ErrReqNotSupp        ATTError = 0x06 // ErrReqNotSupp means the attribute server does not support the request received from the client.
	ErrInvalidOffset     ATTError = 0x07 // ErrInvalidOffset means the specified was past the end of the attribute.
	ErrAuthorization     ATTError = 0x08 // ErrAuthorization means the attribute requires authorization before it can be read or written.
	ErrPrepQueueFull     ATTError = 0x09 // ErrPrepQueueFull means too many prepare writes have been queued.
	ErrAttrNotFound      ATTError = 0x0a // ErrAttrNotFound means no attribute found within the given attribute handle range.
	ErrAttrNotLong       ATTError = 0x0b // ErrAttrNotLong means the attribute cannot be read or written using the Read Blob Request.
	ErrInsuffEncrKeySize ATTError = 0x0c // ErrInsuffEncrKeySize means the Encryption Key Size used for encrypting this link is insufficient.
	ErrInvalAttrValueLen ATTError = 0x0d // ErrInvalAttrValueLen means the attribute value length is invalid for the operation.
	ErrUnlikely          ATTError = 0x0e // ErrUnlikely means the attribute request that was requested has encountered an error that was unlikely, and therefore could not be completed as requested.
	ErrInsuffEnc         ATTError = 0x0f // ErrInsuffEnc means the attribute requires encryption before it can be read or written.
	ErrUnsuppGrpType     ATTError = 0x10 // ErrUnsuppGrpType means the attribute type is not a supported grouping attribute as defined by a higher layer specification.
	ErrInsuffResources   ATTError = 0x11 // ErrInsuffResources means insufficient resources to complete the request.
)

func (e ATTError) Error() string {
	switch i := int(e); {
	case i < 0x11:
		return errName[e]
	case i >= 0x12 && i <= 0x7F: // Reserved for future use.
		return fmt.Sprintf("reserved error code (0x%02X)", i)
	case i >= 0x80 && i <= 0x9F: // Application error, defined by higher level.
		return fmt.Sprintf("application error code (0x%02X)", i)
	case i >= 0xA0 && i <= 0xDF: // Reserved for future use.
		return fmt.Sprintf("reserved error code (0x%02X)", i)
	case i >= 0xE0 && i <= 0xFF: // Common profile and service error codes.
		return "profile or service error"
	}
	return "unkown error"
}

var errName = map[ATTError]string{
	ErrSuccess:           "success",
	ErrInvalidHandle:     "invalid handle",
	ErrReadNotPerm:       "read not permitted",
	ErrWriteNotPerm:      "write not permitted",
	ErrInvalidPDU:        "invalid PDU",
	ErrAuthentication:    "insufficient authentication",
	ErrReqNotSupp:        "request not supported",
	ErrInvalidOffset:     "invalid offset",
	ErrAuthorization:     "insufficient authorization",
	ErrPrepQueueFull:     "prepare queue full",
	ErrAttrNotFound:      "attribute not found",
	ErrAttrNotLong:       "attribute not long",
	ErrInsuffEncrKeySize: "insufficient encryption key size",
	ErrInvalAttrValueLen: "invalid attribute value length",
	ErrUnlikely:          "unlikely error",
	ErrInsuffEnc:         "insufficient encryption",
	ErrUnsuppGrpType:     "unsupported group type",
	ErrInsuffResources:   "insufficient resources",
}
