package att

import "encoding/binary"

// ErrorResponseCode ...
const ErrorResponseCode = 0x01

// ErrorResponse implements Error Response (0x01) [Vol 3, Part E, 3.4.1.1].
type ErrorResponse []byte

// AttributeOpcode ...
func (r ErrorResponse) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ErrorResponse) SetAttributeOpcode() { r[0] = 0x01 }

// RequestOpcodeInError ...
func (r ErrorResponse) RequestOpcodeInError() uint8 { return r[1] }

// SetRequestOpcodeInError ...
func (r ErrorResponse) SetRequestOpcodeInError(v uint8) { r[1] = v }

// AttributeInError ...
func (r ErrorResponse) AttributeInError() uint16 { return binary.LittleEndian.Uint16(r[2:]) }

// SetAttributeInError ...
func (r ErrorResponse) SetAttributeInError(v uint16) { binary.LittleEndian.PutUint16(r[2:], v) }

// ErrorCode ...
func (r ErrorResponse) ErrorCode() uint8 { return r[4] }

// SetErrorCode ...
func (r ErrorResponse) SetErrorCode(v uint8) { r[4] = v }

// ExchangeMTURequestCode ...
const ExchangeMTURequestCode = 0x02

// ExchangeMTURequest implements Exchange MTU Request (0x02) [Vol 3, Part E, 3.4.2.1].
type ExchangeMTURequest []byte

// AttributeOpcode ...
func (r ExchangeMTURequest) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ExchangeMTURequest) SetAttributeOpcode() { r[0] = 0x02 }

// ClientRxMTU ...
func (r ExchangeMTURequest) ClientRxMTU() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetClientRxMTU ...
func (r ExchangeMTURequest) SetClientRxMTU(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// ExchangeMTUResponseCode ...
const ExchangeMTUResponseCode = 0x03

// ExchangeMTUResponse implements Exchange MTU Response (0x03) [Vol 3, Part E, 3.4.2.2].
type ExchangeMTUResponse []byte

// AttributeOpcode ...
func (r ExchangeMTUResponse) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ExchangeMTUResponse) SetAttributeOpcode() { r[0] = 0x03 }

// ServerRxMTU ...
func (r ExchangeMTUResponse) ServerRxMTU() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetServerRxMTU ...
func (r ExchangeMTUResponse) SetServerRxMTU(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// FindInformationRequestCode ...
const FindInformationRequestCode = 0x04

// FindInformationRequest implements Find Information Request (0x04) [Vol 3, Part E, 3.4.3.1].
type FindInformationRequest []byte

// AttributeOpcode ...
func (r FindInformationRequest) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r FindInformationRequest) SetAttributeOpcode() { r[0] = 0x04 }

// StartingHandle ...
func (r FindInformationRequest) StartingHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetStartingHandle ...
func (r FindInformationRequest) SetStartingHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// EndingHandle ...
func (r FindInformationRequest) EndingHandle() uint16 { return binary.LittleEndian.Uint16(r[3:]) }

// SetEndingHandle ...
func (r FindInformationRequest) SetEndingHandle(v uint16) { binary.LittleEndian.PutUint16(r[3:], v) }

// FindInformationResponseCode ...
const FindInformationResponseCode = 0x05

// FindInformationResponse implements Find Information Response (0x05) [Vol 3, Part E, 3.4.3.2].
type FindInformationResponse []byte

// AttributeOpcode ...
func (r FindInformationResponse) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r FindInformationResponse) SetAttributeOpcode() { r[0] = 0x05 }

// Format ...
func (r FindInformationResponse) Format() uint8 { return r[1] }

// SetFormat ...
func (r FindInformationResponse) SetFormat(v uint8) { r[1] = v }

// InformationData ...
func (r FindInformationResponse) InformationData() []byte { return r[2:] }

// SetInformationData ...
func (r FindInformationResponse) SetInformationData(v []byte) { copy(r[2:], v) }

// FindByTypeValueRequestCode ...
const FindByTypeValueRequestCode = 0x06

// FindByTypeValueRequest implements Find By Type Value Request (0x06) [Vol 3, Part E, 3.4.3.3].
type FindByTypeValueRequest []byte

// AttributeOpcode ...
func (r FindByTypeValueRequest) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r FindByTypeValueRequest) SetAttributeOpcode() { r[0] = 0x06 }

// StartingHandle ...
func (r FindByTypeValueRequest) StartingHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetStartingHandle ...
func (r FindByTypeValueRequest) SetStartingHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// EndingHandle ...
func (r FindByTypeValueRequest) EndingHandle() uint16 { return binary.LittleEndian.Uint16(r[3:]) }

// SetEndingHandle ...
func (r FindByTypeValueRequest) SetEndingHandle(v uint16) { binary.LittleEndian.PutUint16(r[3:], v) }

// AttributeType ...
func (r FindByTypeValueRequest) AttributeType() uint16 { return binary.LittleEndian.Uint16(r[5:]) }

// SetAttributeType ...
func (r FindByTypeValueRequest) SetAttributeType(v uint16) { binary.LittleEndian.PutUint16(r[5:], v) }

// AttributeValue ...
func (r FindByTypeValueRequest) AttributeValue() []byte { return r[7:] }

// SetAttributeValue ...
func (r FindByTypeValueRequest) SetAttributeValue(v []byte) { copy(r[7:], v) }

// FindByTypeValueResponseCode ...
const FindByTypeValueResponseCode = 0x07

// FindByTypeValueResponse implements Find By Type Value Response (0x07) [Vol 3, Part E, 3.4.3.4].
type FindByTypeValueResponse []byte

// AttributeOpcode ...
func (r FindByTypeValueResponse) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r FindByTypeValueResponse) SetAttributeOpcode() { r[0] = 0x07 }

// HandleInformationList ...
func (r FindByTypeValueResponse) HandleInformationList() []byte { return r[1:] }

// SetHandleInformationList ...
func (r FindByTypeValueResponse) SetHandleInformationList(v []byte) { copy(r[1:], v) }

// ReadByTypeRequestCode ...
const ReadByTypeRequestCode = 0x08

// ReadByTypeRequest implements Read By Type Request (0x08) [Vol 3, Part E, 3.4.4.1].
type ReadByTypeRequest []byte

// AttributeOpcode ...
func (r ReadByTypeRequest) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ReadByTypeRequest) SetAttributeOpcode() { r[0] = 0x08 }

// StartingHandle ...
func (r ReadByTypeRequest) StartingHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetStartingHandle ...
func (r ReadByTypeRequest) SetStartingHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// EndingHandle ...
func (r ReadByTypeRequest) EndingHandle() uint16 { return binary.LittleEndian.Uint16(r[3:]) }

// SetEndingHandle ...
func (r ReadByTypeRequest) SetEndingHandle(v uint16) { binary.LittleEndian.PutUint16(r[3:], v) }

// AttributeType ...
func (r ReadByTypeRequest) AttributeType() []byte { return r[5:] }

// SetAttributeType ...
func (r ReadByTypeRequest) SetAttributeType(v []byte) { copy(r[5:], v) }

// ReadByTypeResponseCode ...
const ReadByTypeResponseCode = 0x09

// ReadByTypeResponse implements Read By Type Response (0x09) [Vol 3, Part E, 3.4.4.2].
type ReadByTypeResponse []byte

// AttributeOpcode ...
func (r ReadByTypeResponse) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ReadByTypeResponse) SetAttributeOpcode() { r[0] = 0x09 }

// Length ...
func (r ReadByTypeResponse) Length() uint8 { return r[1] }

// SetLength ...
func (r ReadByTypeResponse) SetLength(v uint8) { r[1] = v }

// AttributeDataList ...
func (r ReadByTypeResponse) AttributeDataList() []byte { return r[2:] }

// SetAttributeDataList ...
func (r ReadByTypeResponse) SetAttributeDataList(v []byte) { copy(r[2:], v) }

// ReadRequestCode ...
const ReadRequestCode = 0x0A

// ReadRequest implements Read Request (0x0A) [Vol 3, Part E, 3.4.4.3].
type ReadRequest []byte

// AttributeOpcode ...
func (r ReadRequest) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ReadRequest) SetAttributeOpcode() { r[0] = 0x0A }

// AttributeHandle ...
func (r ReadRequest) AttributeHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetAttributeHandle ...
func (r ReadRequest) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// ReadResponseCode ...
const ReadResponseCode = 0x0B

// ReadResponse implements Read Response (0x0B) [Vol 3, Part E, 3.4.4.4].
type ReadResponse []byte

// AttributeOpcode ...
func (r ReadResponse) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ReadResponse) SetAttributeOpcode() { r[0] = 0x0B }

// AttributeValue ...
func (r ReadResponse) AttributeValue() []byte { return r[1:] }

// SetAttributeValue ...
func (r ReadResponse) SetAttributeValue(v []byte) { copy(r[1:], v) }

// ReadBlobRequestCode ...
const ReadBlobRequestCode = 0x0C

// ReadBlobRequest implements Read Blob Request (0x0C) [Vol 3, Part E, 3.4.4.5].
type ReadBlobRequest []byte

// AttributeOpcode ...
func (r ReadBlobRequest) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ReadBlobRequest) SetAttributeOpcode() { r[0] = 0x0C }

// AttributeHandle ...
func (r ReadBlobRequest) AttributeHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetAttributeHandle ...
func (r ReadBlobRequest) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// ValueOffset ...
func (r ReadBlobRequest) ValueOffset() uint16 { return binary.LittleEndian.Uint16(r[3:]) }

// SetValueOffset ...
func (r ReadBlobRequest) SetValueOffset(v uint16) { binary.LittleEndian.PutUint16(r[3:], v) }

// ReadBlobResponseCode ...
const ReadBlobResponseCode = 0x0D

// ReadBlobResponse implements Read Blob Response (0x0D) [Vol 3, Part E, 3.4.4.6].
type ReadBlobResponse []byte

// AttributeOpcode ...
func (r ReadBlobResponse) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ReadBlobResponse) SetAttributeOpcode() { r[0] = 0x0D }

// PartAttributeValue ...
func (r ReadBlobResponse) PartAttributeValue() []byte { return r[1:] }

// SetPartAttributeValue ...
func (r ReadBlobResponse) SetPartAttributeValue(v []byte) { copy(r[1:], v) }

// ReadMultipleRequestCode ...
const ReadMultipleRequestCode = 0x0E

// ReadMultipleRequest implements Read Multiple Request (0x0E) [Vol 3, Part E, 3.4.4.7].
type ReadMultipleRequest []byte

// AttributeOpcode ...
func (r ReadMultipleRequest) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ReadMultipleRequest) SetAttributeOpcode() { r[0] = 0x0E }

// SetOfHandles ...
func (r ReadMultipleRequest) SetOfHandles() []byte { return r[1:] }

// SetSetOfHandles ...
func (r ReadMultipleRequest) SetSetOfHandles(v []byte) { copy(r[1:], v) }

// ReadMultipleResponseCode ...
const ReadMultipleResponseCode = 0x0F

// ReadMultipleResponse implements Read Multiple Response (0x0F) [Vol 3, Part E, 3.4.4.8].
type ReadMultipleResponse []byte

// AttributeOpcode ...
func (r ReadMultipleResponse) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ReadMultipleResponse) SetAttributeOpcode() { r[0] = 0x0F }

// SetOfValues ...
func (r ReadMultipleResponse) SetOfValues() []byte { return r[1:] }

// SetSetOfValues ...
func (r ReadMultipleResponse) SetSetOfValues(v []byte) { copy(r[1:], v) }

// ReadByGroupTypeRequestCode ...
const ReadByGroupTypeRequestCode = 0x10

// ReadByGroupTypeRequest implements Read By Group Type Request (0x10) [Vol 3, Part E, 3.4.4.9].
type ReadByGroupTypeRequest []byte

// AttributeOpcode ...
func (r ReadByGroupTypeRequest) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ReadByGroupTypeRequest) SetAttributeOpcode() { r[0] = 0x10 }

// StartingHandle ...
func (r ReadByGroupTypeRequest) StartingHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetStartingHandle ...
func (r ReadByGroupTypeRequest) SetStartingHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// EndingHandle ...
func (r ReadByGroupTypeRequest) EndingHandle() uint16 { return binary.LittleEndian.Uint16(r[3:]) }

// SetEndingHandle ...
func (r ReadByGroupTypeRequest) SetEndingHandle(v uint16) { binary.LittleEndian.PutUint16(r[3:], v) }

// AttributeGroupType ...
func (r ReadByGroupTypeRequest) AttributeGroupType() []byte { return r[5:] }

// SetAttributeGroupType ...
func (r ReadByGroupTypeRequest) SetAttributeGroupType(v []byte) { copy(r[5:], v) }

// ReadByGroupTypeResponseCode ...
const ReadByGroupTypeResponseCode = 0x11

// ReadByGroupTypeResponse implements Read By Group Type Response (0x11) [Vol 3, Part E, 3.4.4.10].
type ReadByGroupTypeResponse []byte

// AttributeOpcode ...
func (r ReadByGroupTypeResponse) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ReadByGroupTypeResponse) SetAttributeOpcode() { r[0] = 0x11 }

// Length ...
func (r ReadByGroupTypeResponse) Length() uint8 { return r[1] }

// SetLength ...
func (r ReadByGroupTypeResponse) SetLength(v uint8) { r[1] = v }

// AttributeDataList ...
func (r ReadByGroupTypeResponse) AttributeDataList() []byte { return r[2:] }

// SetAttributeDataList ...
func (r ReadByGroupTypeResponse) SetAttributeDataList(v []byte) { copy(r[2:], v) }

// WriteRequestCode ...
const WriteRequestCode = 0x12

// WriteRequest implements Write Request (0x12) [Vol 3, Part E, 3.4.5.1].
type WriteRequest []byte

// AttributeOpcode ...
func (r WriteRequest) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r WriteRequest) SetAttributeOpcode() { r[0] = 0x12 }

// AttributeHandle ...
func (r WriteRequest) AttributeHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetAttributeHandle ...
func (r WriteRequest) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// AttributeValue ...
func (r WriteRequest) AttributeValue() []byte { return r[3:] }

// SetAttributeValue ...
func (r WriteRequest) SetAttributeValue(v []byte) { copy(r[3:], v) }

// WriteResponseCode ...
const WriteResponseCode = 0x13

// WriteResponse implements Write Response (0x13) [Vol 3, Part E, 3.4.5.2].
type WriteResponse []byte

// AttributeOpcode ...
func (r WriteResponse) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r WriteResponse) SetAttributeOpcode() { r[0] = 0x13 }

// WriteCommandCode ...
const WriteCommandCode = 0x52

// WriteCommand implements Write Command (0x52) [Vol 3, Part E, 3.4.5.3].
type WriteCommand []byte

// AttributeOpcode ...
func (r WriteCommand) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r WriteCommand) SetAttributeOpcode() { r[0] = 0x52 }

// AttributeHandle ...
func (r WriteCommand) AttributeHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetAttributeHandle ...
func (r WriteCommand) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// AttributeValue ...
func (r WriteCommand) AttributeValue() []byte { return r[3:] }

// SetAttributeValue ...
func (r WriteCommand) SetAttributeValue(v []byte) { copy(r[3:], v) }

// SignedWriteCommandCode ...
const SignedWriteCommandCode = 0xD2

// SignedWriteCommand implements Signed Write Command (0xD2) [Vol 3, Part E, 3.4.5.4].
type SignedWriteCommand []byte

// AttributeOpcode ...
func (r SignedWriteCommand) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r SignedWriteCommand) SetAttributeOpcode() { r[0] = 0xD2 }

// AttributeHandle ...
func (r SignedWriteCommand) AttributeHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetAttributeHandle ...
func (r SignedWriteCommand) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// AttributeValue ...
func (r SignedWriteCommand) AttributeValue() []byte { return r[3:] }

// SetAttributeValue ...
func (r SignedWriteCommand) SetAttributeValue(v []byte) { copy(r[3:], v) }

// AuthenticationSignature ...
func (r SignedWriteCommand) AuthenticationSignature() [12]byte {
	b := [12]byte{}
	copy(b[:], r[3:])
	return b
}

// SetAuthenticationSignature ...
func (r SignedWriteCommand) SetAuthenticationSignature(v [12]byte) { copy(r[3:3+12], v[:]) }

// PrepareWriteRequestCode ...
const PrepareWriteRequestCode = 0x16

// PrepareWriteRequest implements Prepare Write Request (0x16) [Vol 3, Part E, 3.4.6.1].
type PrepareWriteRequest []byte

// AttributeOpcode ...
func (r PrepareWriteRequest) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r PrepareWriteRequest) SetAttributeOpcode() { r[0] = 0x16 }

// AttributeHandle ...
func (r PrepareWriteRequest) AttributeHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetAttributeHandle ...
func (r PrepareWriteRequest) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// ValueOffset ...
func (r PrepareWriteRequest) ValueOffset() uint16 { return binary.LittleEndian.Uint16(r[3:]) }

// SetValueOffset ...
func (r PrepareWriteRequest) SetValueOffset(v uint16) { binary.LittleEndian.PutUint16(r[3:], v) }

// PartAttributeValue ...
func (r PrepareWriteRequest) PartAttributeValue() []byte { return r[5:] }

// SetPartAttributeValue ...
func (r PrepareWriteRequest) SetPartAttributeValue(v []byte) { copy(r[5:], v) }

// PrepareWriteResponseCode ...
const PrepareWriteResponseCode = 0x17

// PrepareWriteResponse implements Prepare Write Response (0x17) [Vol 3, Part E, 3.4.6.2].
type PrepareWriteResponse []byte

// AttributeOpcode ...
func (r PrepareWriteResponse) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r PrepareWriteResponse) SetAttributeOpcode() { r[0] = 0x17 }

// AttributeHandle ...
func (r PrepareWriteResponse) AttributeHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetAttributeHandle ...
func (r PrepareWriteResponse) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// ValueOffset ...
func (r PrepareWriteResponse) ValueOffset() uint16 { return binary.LittleEndian.Uint16(r[3:]) }

// SetValueOffset ...
func (r PrepareWriteResponse) SetValueOffset(v uint16) { binary.LittleEndian.PutUint16(r[3:], v) }

// PartAttributeValue ...
func (r PrepareWriteResponse) PartAttributeValue() []byte { return r[5:] }

// SetPartAttributeValue ...
func (r PrepareWriteResponse) SetPartAttributeValue(v []byte) { copy(r[5:], v) }

// ExecuteWriteRequestCode ...
const ExecuteWriteRequestCode = 0x18

// ExecuteWriteRequest implements Execute Write Request (0x18) [Vol 3, Part E, 3.4.6.3].
type ExecuteWriteRequest []byte

// AttributeOpcode ...
func (r ExecuteWriteRequest) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ExecuteWriteRequest) SetAttributeOpcode() { r[0] = 0x18 }

// Flags ...
func (r ExecuteWriteRequest) Flags() uint8 { return r[1] }

// SetFlags ...
func (r ExecuteWriteRequest) SetFlags(v uint8) { r[1] = v }

// ExecuteWriteResponseCode ...
const ExecuteWriteResponseCode = 0x19

// ExecuteWriteResponse implements Execute Write Response (0x19) [Vol 3, Part E, 3.4.6.4].
type ExecuteWriteResponse []byte

// AttributeOpcode ...
func (r ExecuteWriteResponse) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r ExecuteWriteResponse) SetAttributeOpcode() { r[0] = 0x19 }

// HandleValueNotificationCode ...
const HandleValueNotificationCode = 0x1B

// HandleValueNotification implements Handle Value Notification (0x1B) [Vol 3, Part E, 3.4.7.1].
type HandleValueNotification []byte

// AttributeOpcode ...
func (r HandleValueNotification) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r HandleValueNotification) SetAttributeOpcode() { r[0] = 0x1B }

// AttributeHandle ...
func (r HandleValueNotification) AttributeHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetAttributeHandle ...
func (r HandleValueNotification) SetAttributeHandle(v uint16) {
	binary.LittleEndian.PutUint16(r[1:], v)
}

// AttributeValue ...
func (r HandleValueNotification) AttributeValue() []byte { return r[3:] }

// SetAttributeValue ...
func (r HandleValueNotification) SetAttributeValue(v []byte) { copy(r[3:], v) }

// HandleValueIndicationCode ...
const HandleValueIndicationCode = 0x1D

// HandleValueIndication implements Handle Value Indication (0x1D) [Vol 3, Part E, 3.4.7.2].
type HandleValueIndication []byte

// AttributeOpcode ...
func (r HandleValueIndication) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r HandleValueIndication) SetAttributeOpcode() { r[0] = 0x1D }

// AttributeHandle ...
func (r HandleValueIndication) AttributeHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }

// SetAttributeHandle ...
func (r HandleValueIndication) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

// AttributeValue ...
func (r HandleValueIndication) AttributeValue() []byte { return r[3:] }

// SetAttributeValue ...
func (r HandleValueIndication) SetAttributeValue(v []byte) { copy(r[3:], v) }

// HandleValueConfirmationCode ...
const HandleValueConfirmationCode = 0x1E

// HandleValueConfirmation implements Handle Value Confirmation (0x1E) [Vol 3, Part E, 3.4.7.3].
type HandleValueConfirmation []byte

// AttributeOpcode ...
func (r HandleValueConfirmation) AttributeOpcode() uint8 { return r[0] }

// SetAttributeOpcode ...
func (r HandleValueConfirmation) SetAttributeOpcode() { r[0] = 0x1E }
