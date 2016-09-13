package ble

import (
	"bytes"
	"context"
	"io"
)

// A ReadHandler handles GATT requests.
type ReadHandler interface {
	ServeRead(req Request, rsp ResponseWriter)
}

// ReadHandlerFunc is an adapter to allow the use of ordinary functions as Handlers.
type ReadHandlerFunc func(req Request, rsp ResponseWriter)

// ServeRead returns f(r, maxlen, offset).
func (f ReadHandlerFunc) ServeRead(req Request, rsp ResponseWriter) {
	f(req, rsp)
}

// A WriteHandler handles GATT requests.
type WriteHandler interface {
	ServeWrite(req Request, rsp ResponseWriter)
}

// WriteHandlerFunc is an adapter to allow the use of ordinary functions as Handlers.
type WriteHandlerFunc func(req Request, rsp ResponseWriter)

// ServeWrite returns f(r, maxlen, offset).
func (f WriteHandlerFunc) ServeWrite(req Request, rsp ResponseWriter) {
	f(req, rsp)
}

// A NotifyHandler handles GATT requests.
type NotifyHandler interface {
	ServeNotify(req Request, n Notifier)
}

// NotifyHandlerFunc is an adapter to allow the use of ordinary functions as Handlers.
type NotifyHandlerFunc func(req Request, n Notifier)

// ServeNotify returns f(r, maxlen, offset).
func (f NotifyHandlerFunc) ServeNotify(req Request, n Notifier) {
	f(req, n)
}

// Request ...
type Request interface {
	Conn() Conn
	Data() []byte
	Offset() int
}

// NewRequest returns a default implementation of Request.
func NewRequest(conn Conn, data []byte, offset int) Request {
	return &request{conn: conn, data: data, offset: offset}
}

// Default implementation of request.
type request struct {
	conn   Conn
	data   []byte
	offset int
}

func (r *request) Conn() Conn   { return r.conn }
func (r *request) Data() []byte { return r.data }
func (r *request) Offset() int  { return r.offset }

// ResponseWriter ...
type ResponseWriter interface {
	// Write writes data to return as the characteristic value.
	Write(b []byte) (int, error)

	// Status reports the result of the request.
	Status() ATTError

	// SetStatus reports the result of the request.
	SetStatus(status ATTError)

	// Len ...
	Len() int

	// Cap ...
	Cap() int
}

// NewResponseWriter ...
func NewResponseWriter(buf *bytes.Buffer) ResponseWriter {
	return &responseWriter{buf: buf}
}

// responseWriter implements Response
type responseWriter struct {
	buf    *bytes.Buffer
	status ATTError
}

// Status reports the result of the request.
func (r *responseWriter) Status() ATTError {
	return r.status
}

// SetStatus reports the result of the request.
func (r *responseWriter) SetStatus(status ATTError) {
	r.status = status
}

// Len returns length of the buffer.
// Len returns 0 if it is a dummy write response for WriteCommand.
func (r *responseWriter) Len() int {
	if r.buf == nil {
		return 0
	}
	return r.buf.Len()
}

// Cap returns capacity of the buffer.
// Cap returns 0 if it is a dummy write response for WriteCommand.
func (r *responseWriter) Cap() int {
	if r.buf == nil {
		return 0
	}
	return r.buf.Cap()
}

// Write writes data to return as the characteristic value.
// Cap returns 0 with error set to ErrReqNotSupp if it is a dummy write response for WriteCommand.
func (r *responseWriter) Write(b []byte) (int, error) {
	if r.buf == nil {
		return 0, ErrReqNotSupp
	}
	if len(b) > r.buf.Cap()-r.buf.Len() {
		return 0, io.ErrShortWrite
	}

	return r.buf.Write(b)
}

// Notifier ...
type Notifier interface {
	// Context sends data to the central.
	Context() context.Context

	// Write sends data to the central.
	Write(b []byte) (int, error)

	// Close ...
	Close() error

	// Cap returns the maximum number of bytes that may be sent in a single notification.
	Cap() int
}

type notifier struct {
	ctx    context.Context
	maxlen int
	cancel func()
	send   func([]byte) (int, error)
}

// NewNotifier ...
func NewNotifier(send func([]byte) (int, error)) Notifier {
	n := &notifier{}
	n.ctx, n.cancel = context.WithCancel(context.Background())
	n.send = send
	// n.maxlen = cap
	return n
}

func (n *notifier) Context() context.Context {
	return n.ctx
}

func (n *notifier) Write(b []byte) (int, error) {
	return n.send(b)
}

func (n *notifier) Close() error {
	n.cancel()
	return nil
}

func (n *notifier) Cap() int {
	return n.maxlen
}
