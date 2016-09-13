package att

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/currantlabs/ble"
)

type conn struct {
	ble.Conn
	svr  *Server
	cccs map[uint16]uint16
	nn   map[uint16]ble.Notifier
	in   map[uint16]ble.Notifier
}

// Server implementas an ATT (Attribute Protocol) server.
type Server struct {
	conn *conn
	db   *DB

	// Refer to [Vol 3, Part F, 3.3.2 & 3.3.3] for the requirement of
	// sequential request-response protocol, and transactions.
	rxMTU     int
	txBuf     []byte
	chNotBuf  chan []byte
	chIndBuf  chan []byte
	chConfirm chan bool

	dummyRspWriter ble.ResponseWriter
}

// NewServer returns an ATT (Attribute Protocol) server.
func NewServer(db *DB, l2c ble.Conn) (*Server, error) {
	mtu := l2c.RxMTU()
	if mtu < ble.DefaultMTU || mtu > ble.MaxMTU {
		return nil, fmt.Errorf("invalid MTU")
	}
	// Although the rxBuf is initialized with the capacity of rxMTU, it is
	// not discovered, and only the default ATT_MTU (23 bytes) of it shall
	// be used until remote central request ExchangeMTU.
	s := &Server{
		conn: &conn{
			Conn: l2c,
			cccs: make(map[uint16]uint16),
			in:   make(map[uint16]ble.Notifier),
			nn:   make(map[uint16]ble.Notifier),
		},
		db: db,

		rxMTU:     mtu,
		txBuf:     make([]byte, ble.DefaultMTU, ble.DefaultMTU),
		chNotBuf:  make(chan []byte, 1),
		chIndBuf:  make(chan []byte, 1),
		chConfirm: make(chan bool),

		dummyRspWriter: ble.NewResponseWriter(nil),
	}
	s.conn.svr = s
	s.chNotBuf <- make([]byte, ble.DefaultMTU, ble.DefaultMTU)
	s.chIndBuf <- make([]byte, ble.DefaultMTU, ble.DefaultMTU)
	return s, nil
}

// notify sends notification to remote central.
func (s *Server) notify(h uint16, data []byte) (int, error) {
	// Acquire and reuse notifyBuffer. Release it after usage.
	nBuf := <-s.chNotBuf
	defer func() { s.chNotBuf <- nBuf }()

	rsp := HandleValueNotification(nBuf)
	rsp.SetAttributeOpcode()
	rsp.SetAttributeHandle(h)
	buf := bytes.NewBuffer(rsp.AttributeValue())
	buf.Reset()
	if len(data) > buf.Cap() {
		data = data[:buf.Cap()]
	}
	buf.Write(data)
	return s.conn.Write(rsp[:3+buf.Len()])
}

// indicate sends indication to remote central.
func (s *Server) indicate(h uint16, data []byte) (int, error) {
	// Acquire and reuse indicateBuffer. Release it after usage.
	iBuf := <-s.chIndBuf
	defer func() { s.chIndBuf <- iBuf }()

	rsp := HandleValueIndication(iBuf)
	rsp.SetAttributeOpcode()
	rsp.SetAttributeHandle(h)
	buf := bytes.NewBuffer(rsp.AttributeValue())
	buf.Reset()
	if len(data) > buf.Cap() {
		data = data[:buf.Cap()]
	}
	buf.Write(data)
	n, err := s.conn.Write(rsp[:3+buf.Len()])
	if err != nil {
		return n, err
	}
	select {
	case ok := <-s.chConfirm:
		if !ok {
			return 0, io.ErrClosedPipe
		}
		return n, nil
	case <-time.After(time.Second * 30):
		return 0, ErrSeqProtoTimeout
	}
}

// Loop accepts incoming ATT request, and respond response.
func (s *Server) Loop() {
	type sbuf struct {
		buf []byte
		len int
	}
	pool := make(chan *sbuf, 2)
	pool <- &sbuf{buf: make([]byte, s.rxMTU)}
	pool <- &sbuf{buf: make([]byte, s.rxMTU)}

	seq := make(chan *sbuf)
	go func() {
		b := <-pool
		for {
			n, err := s.conn.Read(b.buf)
			if n == 0 || err != nil {
				close(seq)
				s.close()
				return
			}
			if b.buf[0] == HandleValueConfirmationCode {
				select {
				case s.chConfirm <- true:
				default:
					logger.Error("server", "recieved a spurious confirmation", nil)
				}
				continue
			}
			b.len = n
			seq <- b   // Send the current request for handling
			b = <-pool // Swap the buffer for next incoming request.
		}
	}()
	for req := range seq {
		if rsp := s.handleRequest(req.buf[:req.len]); rsp != nil {
			if len(rsp) != 0 {
				s.conn.Write(rsp)
			}
		}
		pool <- req
	}
	for h, ccc := range s.conn.cccs {
		if ccc != 0 {
			logger.Info("cleanup", "ccc", fmt.Sprintf("0x%02X", ccc))
		}
		if ccc&cccIndicate != 0 {
			s.conn.in[h].Close()
		}
		if ccc&cccNotify != 0 {
			s.conn.nn[h].Close()
		}
	}
}

func (s *Server) close() error {
	s.chConfirm <- false
	return s.conn.Close()
}

func (s *Server) handleRequest(b []byte) []byte {
	var resp []byte
	logger.Debug("server", "req", fmt.Sprintf("% X", b))
	switch reqType := b[0]; reqType {
	case ExchangeMTURequestCode:
		resp = s.handleExchangeMTURequest(b)
	case FindInformationRequestCode:
		resp = s.handleFindInformationRequest(b)
	case FindByTypeValueRequestCode:
		resp = s.handleFindByTypeValueRequest(b)
	case ReadByTypeRequestCode:
		resp = s.handleReadByTypeRequest(b)
	case ReadRequestCode:
		resp = s.handleReadRequest(b)
	case ReadBlobRequestCode:
		resp = s.handleReadBlobRequest(b)
	case ReadByGroupTypeRequestCode:
		resp = s.handleReadByGroupRequest(b)
	case WriteRequestCode:
		resp = s.handleWriteRequest(b)
	case WriteCommandCode:
		s.handleWriteCommand(b)
	case ReadMultipleRequestCode,
		PrepareWriteRequestCode,
		ExecuteWriteRequestCode,
		SignedWriteCommandCode:
		fallthrough
	default:
		resp = newErrorResponse(reqType, 0x0000, ble.ErrReqNotSupp)
	}
	logger.Debug("server", "rsp", fmt.Sprintf("% X", resp))
	return resp
}

// handle MTU Exchange request. [Vol 3, Part F, 3.4.2]
func (s *Server) handleExchangeMTURequest(r ExchangeMTURequest) []byte {
	// Validate the request.
	switch {
	case len(r) != 3:
		fallthrough
	case r.ClientRxMTU() < 23:
		return newErrorResponse(r.AttributeOpcode(), 0x0000, ble.ErrInvalidPDU)
	}

	txMTU := int(r.ClientRxMTU())
	s.conn.SetTxMTU(txMTU)

	if txMTU != len(s.txBuf) {
		// Apply the txMTU afer this response has been sent and before
		// any other attribute protocol PDU is sent.
		defer func() {
			s.txBuf = make([]byte, txMTU, txMTU)
			<-s.chNotBuf
			s.chNotBuf <- make([]byte, txMTU, txMTU)
			<-s.chIndBuf
			s.chIndBuf <- make([]byte, txMTU, txMTU)
		}()
	}

	rsp := ExchangeMTUResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	rsp.SetServerRxMTU(uint16(s.rxMTU))
	return rsp[:3]
}

// handle Find Information request. [Vol 3, Part F, 3.4.3.1 & 3.4.3.2]
func (s *Server) handleFindInformationRequest(r FindInformationRequest) []byte {
	// Validate the request.
	switch {
	case len(r) != 5:
		return newErrorResponse(r.AttributeOpcode(), 0x0000, ble.ErrInvalidPDU)
	case r.StartingHandle() == 0 || r.StartingHandle() > r.EndingHandle():
		return newErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ble.ErrInvalidHandle)
	}

	rsp := FindInformationResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	rsp.SetFormat(0x00)
	buf := bytes.NewBuffer(rsp.InformationData())
	buf.Reset()

	// Each response shall contain Types of the same format.
	for _, a := range s.db.subrange(r.StartingHandle(), r.EndingHandle()) {
		if rsp.Format() == 0 {
			rsp.SetFormat(0x01)
			if a.typ.Len() == 16 {
				rsp.SetFormat(0x02)
			}
		}
		if rsp.Format() == 0x01 && a.typ.Len() != 2 {
			break
		}
		if rsp.Format() == 0x02 && a.typ.Len() != 16 {
			break
		}

		if buf.Len()+2+a.typ.Len() > buf.Cap() {
			break
		}
		binary.Write(buf, binary.LittleEndian, a.h)
		binary.Write(buf, binary.LittleEndian, a.typ)
	}

	// Nothing has been found.
	if rsp.Format() == 0 {
		return newErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ble.ErrAttrNotFound)
	}
	return rsp[:2+buf.Len()]
}

// handle Find By Type Value request. [Vol 3, Part F, 3.4.3.3 & 3.4.3.4]
func (s *Server) handleFindByTypeValueRequest(r FindByTypeValueRequest) []byte {
	// Validate the request.
	switch {
	case len(r) < 7:
		return newErrorResponse(r.AttributeOpcode(), 0x0000, ble.ErrInvalidPDU)
	case r.StartingHandle() == 0 || r.StartingHandle() > r.EndingHandle():
		return newErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ble.ErrInvalidHandle)
	}

	rsp := FindByTypeValueResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.HandleInformationList())
	buf.Reset()

	for _, a := range s.db.subrange(r.StartingHandle(), r.EndingHandle()) {
		v, starth, endh := a.v, a.h, a.endh
		if v == nil {
			// The value shall not exceed ATT_MTU - 7 bytes.
			// Since ResponseWriter caps the value at the capacity,
			// we allocate one extra byte, and the written length.
			buf2 := bytes.NewBuffer(make([]byte, 0, len(s.txBuf)-7+1))
			e := handleATT(a, s.conn, r, ble.NewResponseWriter(buf2))
			if e != ble.ErrSuccess || buf2.Len() > len(s.txBuf)-7 {
				return newErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ble.ErrInvalidHandle)
			}
			endh = a.h
		}
		if !(ble.UUID(v).Equal(ble.UUID(r.AttributeValue()))) {
			continue
		}

		if buf.Len()+4 > buf.Cap() {
			break
		}
		binary.Write(buf, binary.LittleEndian, starth)
		binary.Write(buf, binary.LittleEndian, endh)
	}
	if buf.Len() == 0 {
		return newErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ble.ErrAttrNotFound)
	}

	return rsp[:1+buf.Len()]
}

// handle Read By Type request. [Vol 3, Part F, 3.4.4.1 & 3.4.4.2]
func (s *Server) handleReadByTypeRequest(r ReadByTypeRequest) []byte {
	// Validate the request.
	switch {
	case len(r) != 7 && len(r) != 21:
		return newErrorResponse(r.AttributeOpcode(), 0x0000, ble.ErrInvalidPDU)
	case r.StartingHandle() == 0 || r.StartingHandle() > r.EndingHandle():
		return newErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ble.ErrInvalidHandle)
	}

	rsp := ReadByTypeResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.AttributeDataList())
	buf.Reset()

	// handle length (2 bytes) + value length.
	// Each response shall only contains values with the same size.
	dlen := 0
	for _, a := range s.db.subrange(r.StartingHandle(), r.EndingHandle()) {
		if !a.typ.Equal(ble.UUID(r.AttributeType())) {
			continue
		}
		v := a.v
		if v == nil {
			buf2 := bytes.NewBuffer(make([]byte, 0, len(s.txBuf)-2))
			if e := handleATT(a, s.conn, r, ble.NewResponseWriter(buf2)); e != ble.ErrSuccess {
				// Return if the first value read cause an error.
				if dlen == 0 {
					return newErrorResponse(r.AttributeOpcode(), r.StartingHandle(), e)
				}
				// Otherwise, skip to the next one.
				break
			}
			v = buf2.Bytes()
		}
		if dlen == 0 {
			// Found the first value.
			dlen = 2 + len(v)
			if dlen > 255 {
				dlen = 255
			}
			if dlen > buf.Cap() {
				dlen = buf.Cap()
			}
			rsp.SetLength(uint8(dlen))
		} else if 2+len(v) != dlen {
			break
		}

		if buf.Len()+dlen > buf.Cap() {
			break
		}
		binary.Write(buf, binary.LittleEndian, a.h)
		binary.Write(buf, binary.LittleEndian, v[:dlen-2])
	}
	if dlen == 0 {
		return newErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ble.ErrAttrNotFound)
	}
	return rsp[:2+buf.Len()]
}

// handle Read request. [Vol 3, Part F, 3.4.4.3 & 3.4.4.4]
func (s *Server) handleReadRequest(r ReadRequest) []byte {
	// Validate the request.
	switch {
	case len(r) != 3:
		return newErrorResponse(r.AttributeOpcode(), 0x0000, ble.ErrInvalidPDU)
	}

	rsp := ReadResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.AttributeValue())
	buf.Reset()

	a, ok := s.db.at(r.AttributeHandle())
	if !ok {
		return newErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ble.ErrInvalidHandle)
	}

	// Simple case. Read-only, no-authorization, no-authentication.
	if a.v != nil {
		binary.Write(buf, binary.LittleEndian, a.v)
		return rsp[:1+buf.Len()]
	}

	// Pass the request to upper layer with the ResponseWriter, which caps
	// the buffer to a valid length of payload.
	if e := handleATT(a, s.conn, r, ble.NewResponseWriter(buf)); e != ble.ErrSuccess {
		return newErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), e)
	}
	return rsp[:1+buf.Len()]
}

// handle Read Blob request. [Vol 3, Part F, 3.4.4.5 & 3.4.4.6]
func (s *Server) handleReadBlobRequest(r ReadBlobRequest) []byte {
	// Validate the request.
	switch {
	case len(r) != 5:
		return newErrorResponse(r.AttributeOpcode(), 0x0000, ble.ErrInvalidPDU)
	}

	a, ok := s.db.at(r.AttributeHandle())
	if !ok {
		return newErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ble.ErrInvalidHandle)
	}

	rsp := ReadBlobResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.PartAttributeValue())
	buf.Reset()

	// Simple case. Read-only, no-authorization, no-authentication.
	if a.v != nil {
		binary.Write(buf, binary.LittleEndian, a.v)
		return rsp[:1+buf.Len()]
	}

	// Pass the request to upper layer with the ResponseWriter, which caps
	// the buffer to a valid length of payload.
	if e := handleATT(a, s.conn, r, ble.NewResponseWriter(buf)); e != ble.ErrSuccess {
		return newErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), e)
	}
	return rsp[:1+buf.Len()]
}

// handle Read Blob request. [Vol 3, Part F, 3.4.4.9 & 3.4.4.10]
func (s *Server) handleReadByGroupRequest(r ReadByGroupTypeRequest) []byte {
	// Validate the request.
	switch {
	case len(r) != 7 && len(r) != 21:
		return newErrorResponse(r.AttributeOpcode(), 0x0000, ble.ErrInvalidPDU)
	case r.StartingHandle() == 0 || r.StartingHandle() > r.EndingHandle():
		return newErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ble.ErrInvalidHandle)
	}

	rsp := ReadByGroupTypeResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.AttributeDataList())
	buf.Reset()

	dlen := 0
	for _, a := range s.db.subrange(r.StartingHandle(), r.EndingHandle()) {
		v := a.v
		if v == nil {
			buf2 := bytes.NewBuffer(make([]byte, buf.Cap()-buf.Len()-4))
			if e := handleATT(a, s.conn, r, ble.NewResponseWriter(buf2)); e != ble.ErrSuccess {
				return newErrorResponse(r.AttributeOpcode(), r.StartingHandle(), e)
			}
			v = buf2.Bytes()
		}
		if dlen == 0 {
			dlen = 4 + len(v)
			if dlen > 255 {
				dlen = 255
			}
			if dlen > buf.Cap() {
				dlen = buf.Cap()
			}
			rsp.SetLength(uint8(dlen))
		} else if 4+len(v) != dlen {
			break
		}

		if buf.Len()+dlen > buf.Cap() {
			break
		}
		binary.Write(buf, binary.LittleEndian, a.h)
		binary.Write(buf, binary.LittleEndian, a.endh)
		binary.Write(buf, binary.LittleEndian, v[:dlen-4])
	}
	if dlen == 0 {
		return newErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ble.ErrAttrNotFound)
	}
	return rsp[:2+buf.Len()]
}

// handle Write request. [Vol 3, Part F, 3.4.5.1 & 3.4.5.2]
func (s *Server) handleWriteRequest(r WriteRequest) []byte {
	// Validate the request.
	switch {
	case len(r) < 3:
		return newErrorResponse(r.AttributeOpcode(), 0x0000, ble.ErrInvalidPDU)
	}

	a, ok := s.db.at(r.AttributeHandle())
	if !ok {
		return newErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ble.ErrInvalidHandle)
	}

	// We don't support write to static value. Pass the request to upper layer.
	if a == nil {
		return newErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ble.ErrWriteNotPerm)
	}
	if e := handleATT(a, s.conn, r, ble.NewResponseWriter(nil)); e != ble.ErrSuccess {
		return newErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), e)
	}
	return []byte{WriteResponseCode}
}

// handle Write command. [Vol 3, Part F, 3.4.5.3]
func (s *Server) handleWriteCommand(r WriteCommand) []byte {
	// Validate the request.
	switch {
	case len(r) <= 3:
		return nil
	}

	a, ok := s.db.at(r.AttributeHandle())
	if !ok {
		return nil
	}

	// We don't support write to static value. Pass the request to upper layer.
	if a == nil {
		return nil
	}
	if e := handleATT(a, s.conn, r, s.dummyRspWriter); e != ble.ErrSuccess {
		return nil
	}
	return nil
}

func newErrorResponse(op byte, h uint16, s ble.ATTError) []byte {
	r := ErrorResponse(make([]byte, 5))
	r.SetAttributeOpcode()
	r.SetRequestOpcodeInError(op)
	r.SetAttributeInError(h)
	r.SetErrorCode(uint8(s))
	return r
}

func handleATT(a *attr, conn ble.Conn, req []byte, rsp ble.ResponseWriter) ble.ATTError {
	rsp.SetStatus(ble.ErrSuccess)
	var offset int
	var data []byte
	switch req[0] {
	case ReadByTypeRequestCode:
		fallthrough
	case ReadRequestCode:
		if a.rh == nil {
			return ble.ErrReadNotPerm
		}
		a.rh.ServeRead(ble.NewRequest(conn, data, offset), rsp)
	case ReadBlobRequestCode:
		if a.rh == nil {
			return ble.ErrReadNotPerm
		}
		offset = int(ReadBlobRequest(req).ValueOffset())
		a.rh.ServeRead(ble.NewRequest(conn, data, offset), rsp)
	case WriteRequestCode:
		fallthrough
	case WriteCommandCode:
		if a.wh == nil {
			return ble.ErrWriteNotPerm
		}
		data = WriteRequest(req).AttributeValue()
		a.wh.ServeWrite(ble.NewRequest(conn, data, offset), rsp)
	// case PrepareWriteRequestCode:
	// case ExecuteWriteRequestCode:
	// case SignedWriteCommandCode:
	// case ReadByGroupTypeRequestCode:
	// case ReadMultipleRequestCode:
	default:
		return ble.ErrReqNotSupp
	}

	return rsp.Status()
}
