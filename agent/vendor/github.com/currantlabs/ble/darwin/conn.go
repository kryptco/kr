package darwin

import (
	"context"
	"sync"

	"github.com/currantlabs/ble"
	"github.com/raff/goble/xpc"
)

func newConn(d *Device, a ble.Addr) *conn {
	return &conn{
		dev:   d,
		rxMTU: 23,
		txMTU: 23,
		addr:  a,

		notifiers: make(map[uint16]ble.Notifier),
		subs:      make(map[uint16]*sub),

		rspc: make(chan msg),
	}
}

type conn struct {
	sync.RWMutex

	dev   *Device
	role  int
	ctx   context.Context
	rxMTU int
	txMTU int
	addr  ble.Addr

	rspc chan msg

	connInterval       int
	connLatency        int
	supervisionTimeout int

	notifiers map[uint16]ble.Notifier // central connection only

	subs map[uint16]*sub
}

func (c *conn) Context() context.Context {
	return c.ctx
}

func (c *conn) SetContext(ctx context.Context) {
	c.ctx = ctx
}

func (c *conn) LocalAddr() ble.Addr {
	return c.dev.Addr()
}

func (c *conn) RemoteAddr() ble.Addr {
	return c.addr
}

func (c *conn) RxMTU() int {
	return c.rxMTU
}

func (c *conn) SetRxMTU(mtu int) {
	c.rxMTU = mtu
}

func (c *conn) TxMTU() int {
	return c.txMTU
}

func (c *conn) SetTxMTU(mtu int) {
	c.txMTU = mtu
}

func (c *conn) Read(b []byte) (int, error) {
	return 0, nil
}

func (c *conn) Write(b []byte) (int, error) {
	return 0, nil
}

func (c *conn) Close() error {
	return nil
}

// server (peripheral)
func (c *conn) subscribed(char *ble.Characteristic) {
	h := char.Handle
	if _, found := c.notifiers[h]; found {
		return
	}
	send := func(b []byte) (int, error) {
		c.dev.sendCmd(15, xpc.Dict{
			"kCBMsgArgUUIDs":       [][]byte{},
			"kCBMsgArgAttributeID": h,
			"kCBMsgArgData":        b,
		})
		return len(b), nil
	}
	n := ble.NewNotifier(send)
	c.notifiers[h] = n
	req := ble.NewRequest(c, nil, 0) // convey *conn to user handler.
	go char.NotifyHandler.ServeNotify(req, n)
}

// server (peripheral)
func (c *conn) unsubscribed(char *ble.Characteristic) {
	if n, found := c.notifiers[char.Handle]; found {
		n.Close()
		delete(c.notifiers, char.Handle)
	}
}

func (c *conn) sendReq(id int, args xpc.Dict) msg {
	c.dev.sendCmd(id, args)
	m := <-c.rspc
	return msg(m.args())
}

func (c *conn) sendCmd(id int, args xpc.Dict) {
	c.dev.sendCmd(id, args)
}
