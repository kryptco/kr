package darwin

import (
	"github.com/currantlabs/ble"
	"github.com/raff/goble/xpc"
)

type msg xpc.Dict

func (m msg) id() int        { return xpc.Dict(m).MustGetInt("kCBMsgId") }
func (m msg) args() xpc.Dict { return xpc.Dict(m).MustGetDict("kCBMsgArgs") }
func (m msg) advertisementData() xpc.Dict {
	return xpc.Dict(m).MustGetDict("kCBMsgArgAdvertisementData")
}
func (m msg) attMTU() int          { return xpc.Dict(m).MustGetInt("kCBMsgArgATTMTU") }
func (m msg) attWrites() xpc.Array { return xpc.Dict(m).MustGetArray("kCBMsgArgATTWrites") }
func (m msg) attributeID() int     { return xpc.Dict(m).MustGetInt("kCBMsgArgAttributeID") }
func (m msg) characteristicHandle() int {
	return xpc.Dict(m).MustGetInt("kCBMsgArgCharacteristicHandle")
}
func (m msg) data() []byte {
	// return xpc.Dict(m).MustGetBytes("kCBMsgArgData")
	v := m["kCBMsgArgData"]
	switch v.(type) {
	case string:
		return []byte(v.(string))
	case []byte:
		return v.([]byte)
	default:
		return nil
	}
}

func (m msg) deviceUUID() xpc.UUID       { return xpc.Dict(m).MustGetUUID("kCBMsgArgDeviceUUID") }
func (m msg) ignoreResponse() int        { return xpc.Dict(m).MustGetInt("kCBMsgArgIgnoreResponse") }
func (m msg) offset() int                { return xpc.Dict(m).MustGetInt("kCBMsgArgOffset") }
func (m msg) isNotification() int        { return xpc.Dict(m).GetInt("kCBMsgArgIsNotification", 0) }
func (m msg) result() int                { return xpc.Dict(m).MustGetInt("kCBMsgArgResult") }
func (m msg) state() int                 { return xpc.Dict(m).MustGetInt("kCBMsgArgState") }
func (m msg) rssi() int                  { return xpc.Dict(m).MustGetInt("kCBMsgArgData") }
func (m msg) transactionID() int         { return xpc.Dict(m).MustGetInt("kCBMsgArgTransactionID") }
func (m msg) uuid() string               { return xpc.Dict(m).MustGetHexBytes("kCBMsgArgUUID") }
func (m msg) serviceStartHandle() int    { return xpc.Dict(m).MustGetInt("kCBMsgArgServiceStartHandle") }
func (m msg) serviceEndHandle() int      { return xpc.Dict(m).MustGetInt("kCBMsgArgServiceEndHandle") }
func (m msg) services() xpc.Array        { return xpc.Dict(m).MustGetArray("kCBMsgArgServices") }
func (m msg) characteristics() xpc.Array { return xpc.Dict(m).MustGetArray("kCBMsgArgCharacteristics") }
func (m msg) characteristicProperties() int {
	return xpc.Dict(m).MustGetInt("kCBMsgArgCharacteristicProperties")
}
func (m msg) characteristicValueHandle() int {
	return xpc.Dict(m).MustGetInt("kCBMsgArgCharacteristicValueHandle")
}
func (m msg) descriptors() xpc.Array  { return xpc.Dict(m).MustGetArray("kCBMsgArgDescriptors") }
func (m msg) descriptorHandle() int   { return xpc.Dict(m).MustGetInt("kCBMsgArgDescriptorHandle") }
func (m msg) connectionInterval() int { return xpc.Dict(m).MustGetInt("kCBMsgArgConnectionInterval") }
func (m msg) connectionLatency() int  { return xpc.Dict(m).MustGetInt("kCBMsgArgConnectionLatency") }
func (m msg) supervisionTimeout() int { return xpc.Dict(m).MustGetInt("kCBMsgArgSupervisionTimeout") }

func (m msg) err() error {
	if code := m.result(); code != 0 {
		return ble.ATTError(code)
	}
	return nil
}
