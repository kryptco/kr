package darwin

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/currantlabs/ble"
	"github.com/raff/goble/xpc"
)

const (
	evtStateChanged               = 6
	evtAdvertisingStarted         = 16
	evtAdvertisingStopped         = 17
	evtServiceAdded               = 18
	evtReadRequest                = 19
	evtWriteRequest               = 20
	evtSubscribe                  = 21
	evtUnubscribe                 = 22
	evtConfirmation               = 23
	evtPeripheralDiscovered       = 37
	evtPeripheralConnected        = 38
	evtPeripheralDisconnected     = 40
	evtATTMTU                     = 53
	evtRSSIRead                   = 55
	evtServiceDiscovered          = 56
	evtIncludedServicesDiscovered = 63
	evtCharacteristicsDiscovered  = 64
	evtCharacteristicRead         = 71
	evtCharacteristicWritten      = 72
	evtNotificationValueSet       = 74
	evtDescriptorsDiscovered      = 76
	evtDescriptorRead             = 79
	evtDescriptorWritten          = 80
	evtSleveConnectionComplete    = 81
	evtMasterConnectionComplete   = 82
)

// Device is either a Peripheral or Central device.
type Device struct {
	xpc  xpc.XPC
	role int // 1: peripheralManager (server), 0: centralManager (client)

	rspc chan msg

	conns map[string]*conn

	// Only used in client/centralManager implementation
	advHandler ble.AdvHandler
	chConn     chan *conn

	// Only used in server/peripheralManager implementation
	chars map[int]*ble.Characteristic
	base  int
}

// NewDevice returns a BLE device.
func NewDevice(opts ...Option) (*Device, error) {
	d := &Device{
		rspc:   make(chan msg),
		conns:  make(map[string]*conn),
		chConn: make(chan *conn),
		chars:  make(map[int]*ble.Characteristic),
		base:   1,
	}
	if err := d.Option(opts...); err != nil {
		return nil, err
	}

	d.xpc = xpc.XpcConnect("com.apple.blued", d)

	return d, nil
}

// Option sets the options specified.
func (d *Device) Option(opts ...Option) error {
	var err error
	for _, opt := range opts {
		err = opt(d)
	}
	return err
}

// Init ...
func (d *Device) Init() error {
	rsp := d.sendReq(1, xpc.Dict{
		"kCBMsgArgName": fmt.Sprintf("gopher-%v", time.Now().Unix()),
		"kCBMsgArgOptions": xpc.Dict{
			"kCBInitOptionShowPowerAlert": 1,
		},
		"kCBMsgArgType": d.role,
	})
	if s := State(rsp.state()); s != StatePoweredOn {
		return fmt.Errorf("state: %s", s)
	}
	return nil
}

// AdvertiseMfgData ...
func (d *Device) AdvertiseMfgData(b []byte) error {
	return d.sendReq(8, xpc.Dict{
		"kCBAdvDataAppleMfgData": b,
	}).err()
}

// AdvertiseNameAndServices advertises name and specifid service UUIDs.
func (d *Device) AdvertiseNameAndServices(name string, ss ...ble.UUID) error {
	return d.sendReq(8, xpc.Dict{
		"kCBAdvDataLocalName":    name,
		"kCBAdvDataServiceUUIDs": uuidSlice(ss)},
	).err()
}

// AdvertiseIBeaconData advertises iBeacon packet with specified manufacturer data.
func (d *Device) AdvertiseIBeaconData(md []byte) error {
	var utsname xpc.Utsname
	xpc.Uname(&utsname)

	if utsname.Release >= "14." {
		l := len(md)
		b := []byte{byte(l + 5), 0xFF, 0x4C, 0x00, 0x02, byte(l)}
		return d.AdvertiseMfgData(append(b, md...))
	}
	return d.sendReq(8, xpc.Dict{"kCBAdvDataAppleBeaconKey": md}).err()
}

// AdvertiseIBeacon advertises iBeacon packet.
func (d *Device) AdvertiseIBeacon(u ble.UUID, major, minor uint16, pwr int8) error {
	b := make([]byte, 21)
	copy(b, ble.Reverse(u))                   // Big endian
	binary.BigEndian.PutUint16(b[16:], major) // Big endian
	binary.BigEndian.PutUint16(b[18:], minor) // Big endian
	b[20] = uint8(pwr)                        // Measured Tx Power
	return d.AdvertiseIBeaconData(b)
}

// StopAdvertising stops advertising.
func (d *Device) StopAdvertising() error {
	return d.sendReq(9, nil).err()
}

// SetAdvHandler ...
func (d *Device) SetAdvHandler(ah ble.AdvHandler) error {
	d.advHandler = ah
	return nil
}

// Scan ...
func (d *Device) Scan(allowDup bool) error {
	return d.sendCmd(29, xpc.Dict{
		// "kCBMsgArgUUIDs": uuidSlice(ss),
		"kCBMsgArgOptions": xpc.Dict{
			"kCBScanOptionAllowDuplicates": map[bool]int{true: 1, false: 0}[allowDup],
		},
	})
}

// StopScanning stops scanning
func (d *Device) StopScanning() error {
	return d.sendCmd(30, nil)
}

// RemoveAllServices removes all services of device's
func (d *Device) RemoveAllServices() error {
	return d.sendCmd(12, nil)
}

// AddService adds a service to device's database.
// The following services are ignored as they are provided by OS X.
//
// 0x1800 (Generic Access)
// 0x1801 (Generic Attribute)
// 0x1805 (Current Time Service)
// 0x180A (Device Information)
// 0x180F (Battery Service)
// 0x1812 (Human Interface Device)
func (d *Device) AddService(s *ble.Service) error {
	if s.UUID.Equal(ble.GAPUUID) ||
		s.UUID.Equal(ble.GATTUUID) ||
		s.UUID.Equal(ble.CurrentTimeUUID) ||
		s.UUID.Equal(ble.DeviceInfoUUID) ||
		s.UUID.Equal(ble.BatteryUUID) ||
		s.UUID.Equal(ble.HIDUUID) {
		return nil
	}
	xs := xpc.Dict{
		"kCBMsgArgAttributeID":     d.base,
		"kCBMsgArgAttributeIDs":    []int{},
		"kCBMsgArgCharacteristics": nil,
		"kCBMsgArgType":            1, // 1 => primary, 0 => excluded
		"kCBMsgArgUUID":            ble.Reverse(s.UUID),
	}
	d.base++

	xcs := xpc.Array{}
	for _, c := range s.Characteristics {
		props := 0
		perm := 0
		if c.Property&ble.CharRead != 0 {
			props |= 0x02
			if ble.CharRead&c.Secure != 0 {
				perm |= 0x04
			} else {
				perm |= 0x01
			}
		}
		if c.Property&ble.CharWriteNR != 0 {
			props |= 0x04
			if c.Secure&ble.CharWriteNR != 0 {
				perm |= 0x08
			} else {
				perm |= 0x02
			}
		}
		if c.Property&ble.CharWrite != 0 {
			props |= 0x08
			if c.Secure&ble.CharWrite != 0 {
				perm |= 0x08
			} else {
				perm |= 0x02
			}
		}
		if c.Property&ble.CharNotify != 0 {
			if c.Secure&ble.CharNotify != 0 {
				props |= 0x100
			} else {
				props |= 0x10
			}
		}
		if c.Property&ble.CharIndicate != 0 {
			if c.Secure&ble.CharIndicate != 0 {
				props |= 0x200
			} else {
				props |= 0x20
			}
		}

		xc := xpc.Dict{
			"kCBMsgArgAttributeID":              d.base,
			"kCBMsgArgUUID":                     ble.Reverse(c.UUID),
			"kCBMsgArgAttributePermissions":     perm,
			"kCBMsgArgCharacteristicProperties": props,
			"kCBMsgArgData":                     c.Value,
		}
		c.Handle = uint16(d.base)
		d.chars[d.base] = c
		d.base++

		xds := xpc.Array{}
		for _, d := range c.Descriptors {
			if d.UUID.Equal(ble.ClientCharacteristicConfigUUID) {
				// skip CCCD
				continue
			}
			xd := xpc.Dict{
				"kCBMsgArgData": d.Value,
				"kCBMsgArgUUID": ble.Reverse(d.UUID),
			}
			xds = append(xds, xd)
		}
		xc["kCBMsgArgDescriptors"] = xds
		xcs = append(xcs, xc)
	}
	xs["kCBMsgArgCharacteristics"] = xcs

	return d.sendReq(10, xs).err()
}

// SetServices ...
func (d *Device) SetServices(ss []*ble.Service) error {
	if err := d.RemoveAllServices(); err != nil {
		return nil
	}
	for _, s := range ss {
		if err := d.AddService(s); err != nil {
			return err
		}
	}
	return nil
}

// Dial ...
func (d *Device) Dial(a ble.Addr) (ble.Client, error) {
	d.sendCmd(31, xpc.Dict{
		"kCBMsgArgDeviceUUID": xpc.MakeUUID(a.String()),
		"kCBMsgArgOptions": xpc.Dict{
			"kCBConnectOptionNotifyOnDisconnection": 1,
		},
	})
	return NewClient(<-d.chConn)
}

// HandleXpcEvent process Device events and asynchronous errors.
func (d *Device) HandleXpcEvent(event xpc.Dict, err error) {
	if err != nil {
		log.Println("error:", err)
		return
	}
	m := msg(event)
	args := msg(msg(event).args())
	logger.Info("recv", "id", m.id(), "args", fmt.Sprintf("%v", m.args()))

	switch m.id() {
	case // Device event
		evtStateChanged,
		evtAdvertisingStarted,
		evtAdvertisingStopped,
		evtServiceAdded:
		d.rspc <- args

	case evtPeripheralDiscovered:
		if d.advHandler == nil {
			break
		}
		a := &adv{args: m.args(), ad: args.advertisementData()}
		go d.advHandler.Handle(a)

	case evtConfirmation:
		// log.Printf("confirmed: %d", args.attributeID())

	case evtATTMTU:
		d.conn(args).SetTxMTU(args.attMTU())

	case evtSleveConnectionComplete:
		// remote peripheral is connected.
		fallthrough
	case evtMasterConnectionComplete:
		// remote central is connected.

		// Could be LEConnectionComplete or LEConnectionUpdateComplete.
		c := d.conn(args)
		c.connInterval = args.connectionInterval()
		c.connLatency = args.connectionLatency()
		c.supervisionTimeout = args.supervisionTimeout()

	case evtReadRequest:
		aid := args.attributeID()
		char := d.chars[aid]
		v := char.Value
		if v == nil {
			c := d.conn(args)
			req := ble.NewRequest(c, nil, args.offset())
			buf := bytes.NewBuffer(make([]byte, 0, c.txMTU-1))
			rsp := ble.NewResponseWriter(buf)
			char.ReadHandler.ServeRead(req, rsp)
			v = buf.Bytes()
		}

		d.sendCmd(13, xpc.Dict{
			"kCBMsgArgAttributeID":   aid,
			"kCBMsgArgData":          v,
			"kCBMsgArgTransactionID": args.transactionID(),
			"kCBMsgArgResult":        0,
		})

	case evtWriteRequest:
		for _, xxw := range args.attWrites() {
			xw := msg(xxw.(xpc.Dict))
			aid := xw.attributeID()
			char := d.chars[aid]
			req := ble.NewRequest(d.conn(args), xw.data(), xw.offset())
			char.WriteHandler.ServeWrite(req, nil)
			if xw.ignoreResponse() == 1 {
				continue
			}
			d.sendCmd(13, xpc.Dict{
				"kCBMsgArgAttributeID":   aid,
				"kCBMsgArgData":          nil,
				"kCBMsgArgTransactionID": args.transactionID(),
				"kCBMsgArgResult":        0,
			})
		}

	case evtSubscribe:
		// characteristic is subscribed by remote central.
		d.conn(args).subscribed(d.chars[args.attributeID()])

	case evtUnubscribe:
		// characteristic is unsubscribed by remote central.
		d.conn(args).unsubscribed(d.chars[args.attributeID()])

	case evtPeripheralConnected:
		d.chConn <- d.conn(args)

	case evtPeripheralDisconnected:
		d.conn(args).rspc <- m
		delete(d.conns, d.conn(args).RemoteAddr().String())

	case evtCharacteristicRead:
		// Notification
		c := d.conn(args)
		if args.isNotification() != 0 {
			sub := c.subs[uint16(args.characteristicHandle())]
			if sub == nil {
				log.Printf("notified by unsubscribed handle")
				// FIXME: should terminate the connection?
			} else {
				go sub.fn(args.data())
			}
			break
		}
		c.rspc <- m

	case // Peripheral events
		evtRSSIRead,
		evtServiceDiscovered,
		evtIncludedServicesDiscovered,
		evtCharacteristicsDiscovered,
		evtCharacteristicWritten,
		evtNotificationValueSet,
		evtDescriptorsDiscovered,
		evtDescriptorRead,
		evtDescriptorWritten:

		d.conn(args).rspc <- m

	default:
		log.Printf("Unhandled event: %#v", event)
	}
}

// Addr ...
func (d *Device) Addr() ble.Addr {
	return nil
}

// Stop ...
func (d *Device) Stop() error {
	return nil
}

func (d *Device) conn(m msg) *conn {
	// Convert xpc.UUID to ble.UUID.
	a := ble.MustParse(m.deviceUUID().String())
	c, ok := d.conns[a.String()]
	if !ok {
		c = newConn(d, a)
		d.conns[a.String()] = c
	}
	return c
}

// sendReq sends a message and waits for its reply.
func (d *Device) sendReq(id int, args xpc.Dict) msg {
	d.sendCmd(id, args)
	return <-d.rspc
}

func (d *Device) sendCmd(id int, args xpc.Dict) error {
	logger.Info("send", "id", id, "args", fmt.Sprintf("%v", args))
	d.xpc.Send(xpc.Dict{"kCBMsgId": id, "kCBMsgArgs": args}, false)
	return nil
}
