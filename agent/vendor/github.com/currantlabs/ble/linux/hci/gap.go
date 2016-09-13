package hci

import (
	"fmt"
	"net"
	"time"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/linux/adv"
	"github.com/currantlabs/ble/linux/gatt"
	"github.com/pkg/errors"
)

// Addr ...
func (h *HCI) Addr() ble.Addr { return h.addr }

// SetAdvHandler ...
func (h *HCI) SetAdvHandler(ah ble.AdvHandler) error {
	h.advHandler = ah
	return nil
}

// Scan starts scanning.
func (h *HCI) Scan(allowDup bool) error {
	h.params.scanEnable.FilterDuplicates = 1
	if allowDup {
		h.params.scanEnable.FilterDuplicates = 0
	}
	h.params.scanEnable.LEScanEnable = 1
	h.adHist = make([]*Advertisement, 128)
	h.adLast = 0
	return h.Send(&h.params.scanEnable, nil)
}

// StopScanning stops scanning.
func (h *HCI) StopScanning() error {
	h.params.scanEnable.LEScanEnable = 0
	return h.Send(&h.params.scanEnable, nil)
}

// AdvertiseNameAndServices advertises device name, and specified service UUIDs.
// It tries to fit the UUIDs in the advertising data as much as possible.
// If name doesn't fit in the advertising data, it will be put in scan response.
func (h *HCI) AdvertiseNameAndServices(name string, uuids ...ble.UUID) error {
	ad, err := adv.NewPacket(adv.Flags(adv.FlagGeneralDiscoverable | adv.FlagLEOnly))
	if err != nil {
		return err
	}
	f := adv.AllUUID

	// Current length of ad packet plus two bytes of length and tag.
	l := ad.Len() + 1 + 1
	for _, u := range uuids {
		l += u.Len()
	}
	if l > adv.MaxEIRPacketLength {
		f = adv.SomeUUID
	}
	for _, u := range uuids {
		if err := ad.Append(f(u)); err != nil {
			if err == adv.ErrNotFit {
				break
			}
			return err
		}
	}
	sr, _ := adv.NewPacket()
	switch {
	case ad.Append(adv.CompleteName(name)) == nil:
	case sr.Append(adv.CompleteName(name)) == nil:
	case sr.Append(adv.ShortName(name)) == nil:
	}
	if err := h.SetAdvertisement(ad.Bytes(), sr.Bytes()); err != nil {
		return nil
	}
	return h.Advertise()
}

// AdvertiseIBeaconData advertise iBeacon with given manufacturer data.
func (h *HCI) AdvertiseIBeaconData(md []byte) error {
	ad, err := adv.NewPacket(adv.IBeaconData(md))
	if err != nil {
		return err
	}
	if err := h.SetAdvertisement(ad.Bytes(), nil); err != nil {
		return nil
	}
	return h.Advertise()
}

// AdvertiseIBeacon advertises iBeacon with specified parameters.
func (h *HCI) AdvertiseIBeacon(u ble.UUID, major, minor uint16, pwr int8) error {
	ad, err := adv.NewPacket(adv.IBeacon(u, major, minor, pwr))
	if err != nil {
		return err
	}
	if err := h.SetAdvertisement(ad.Bytes(), nil); err != nil {
		return nil
	}
	return h.Advertise()
}

// StopAdvertising stops advertising.
func (h *HCI) StopAdvertising() error {
	h.params.advEnable.AdvertisingEnable = 0
	return h.Send(&h.params.advEnable, nil)
}

// Accept starts advertising and accepts connection.
func (h *HCI) Accept() (ble.Conn, error) {
	var tmo <-chan time.Time
	if h.listenerTmo != time.Duration(0) {
		tmo = time.After(h.listenerTmo)
	}
	select {
	case <-h.done:
		return nil, h.err
	case c := <-h.chSlaveConn:
		return c, nil
	case <-tmo:
		return nil, fmt.Errorf("listner timed out")
	}
}

// Dial ...
func (h *HCI) Dial(a ble.Addr) (ble.Client, error) {
	b, err := net.ParseMAC(a.String())
	if err != nil {
		return nil, ErrInvalidAddr
	}
	h.params.connParams.PeerAddress = [6]byte{b[5], b[4], b[3], b[2], b[1], b[0]}
	if _, ok := a.(RandomAddress); ok {
		h.params.connParams.PeerAddressType = 1
	}
	if err = h.Send(&h.params.connParams, nil); err != nil {
		return nil, err
	}
	var tmo <-chan time.Time
	if h.dialerTmo != time.Duration(0) {
		tmo = time.After(h.dialerTmo)
	}
	select {
	case <-h.done:
		return nil, h.err
	case c := <-h.chMasterConn:
		return gatt.NewClient(c)
	case <-tmo:
		err := h.Send(&h.params.connCancel, nil)
		if err == nil {
			// The pending connection was canceled successfully.
			return nil, fmt.Errorf("connection timed out")
		}
		// The connection has been established, the cancel command
		// failed with ErrDisallowed.
		if err == ErrDisallowed {
			return gatt.NewClient(<-h.chMasterConn)
		}
		return nil, errors.Wrap(err, "cancel connection failed")
	}
}

// Close ...
func (h *HCI) Close() error {
	if h.err != nil {
		return h.err
	}
	return nil
}

// Advertise starts advertising.
func (h *HCI) Advertise() error {
	h.params.advEnable.AdvertisingEnable = 1
	return h.Send(&h.params.advEnable, nil)
}

// SetAdvertisement sets advertising data and scanResp.
func (h *HCI) SetAdvertisement(ad []byte, sr []byte) error {
	if len(ad) > adv.MaxEIRPacketLength || len(sr) > adv.MaxEIRPacketLength {
		return ble.ErrEIRPacketTooLong
	}

	h.params.advData.AdvertisingDataLength = uint8(len(ad))
	copy(h.params.advData.AdvertisingData[:], ad)
	if err := h.Send(&h.params.advData, nil); err != nil {
		return err
	}

	h.params.scanResp.ScanResponseDataLength = uint8(len(sr))
	copy(h.params.scanResp.ScanResponseData[:], sr)
	if err := h.Send(&h.params.scanResp, nil); err != nil {
		return err
	}
	return nil
}
