package darwin

import (
	"github.com/currantlabs/ble"
	"github.com/raff/goble/xpc"
)

type adv struct {
	args xpc.Dict
	ad   xpc.Dict
}

func (a *adv) LocalName() string {
	return a.ad.GetString("kCBAdvDataLocalName", a.args.GetString("kCBMsgArgName", ""))
}

func (a *adv) ManufacturerData() []byte {
	return a.ad.GetBytes("kCBAdvDataManufacturerData", nil)
}

func (a *adv) ServiceData() []ble.ServiceData {
	xSDs, ok := a.ad["kCBAdvDataServiceData"]
	if !ok {
		return nil
	}

	xSD := xSDs.(xpc.Array)
	var sd []ble.ServiceData
	for i := 0; i < len(xSD); i += 2 {
		sd = append(
			sd, ble.ServiceData{
				UUID: ble.UUID(xSD[i].([]byte)),
				Data: xSD[i+1].([]byte),
			})
	}
	return sd
}

func (a *adv) Services() []ble.UUID {
	xUUIDs, ok := a.ad["kCBAdvDataServiceUUIDs"]
	if !ok {
		return nil
	}
	var uuids []ble.UUID
	for _, xUUID := range xUUIDs.(xpc.Array) {
		uuids = append(uuids, ble.UUID(ble.Reverse(xUUID.([]byte))))
	}
	return uuids
}

func (a *adv) OverflowService() []ble.UUID {
	return nil // TODO
}

func (a *adv) TxPowerLevel() int {
	return a.ad.GetInt("kCBAdvDataTxPowerLevel", 0)
}

func (a *adv) SolicitedService() []ble.UUID {
	return nil // TODO
}

func (a *adv) Connectable() bool {
	return a.ad.GetInt("kCBAdvDataIsConnectable", 0) > 0
}

func (a *adv) RSSI() int {
	return a.args.GetInt("kCBMsgArgRssi", 0)
}

func (a *adv) Address() ble.Addr {
	return xpc.UUID(a.args.MustGetUUID("kCBMsgArgDeviceUUID"))
}
