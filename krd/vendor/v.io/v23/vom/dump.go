// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vom

import (
	"bytes"
	"fmt"
	"io"

	"v.io/v23/vdl"
	"v.io/v23/verror"
)

var (
	errDumperClosed  = verror.Register(pkgPath+".errDumperClosed", verror.NoRetry, "{1:}{2:} vom: Dumper closed{:_}")
	errDumperFlushed = verror.Register(pkgPath+".errDumperFlushed", verror.NoRetry, "{1:}{2:} vom: Dumper flushed{:_}")
)

// Dump returns a human-readable dump of the given vom data, in the default
// string format.
func Dump(data []byte) (string, error) {
	var buf bytes.Buffer
	d := NewDumper(NewDumpWriter(&buf))
	_, err := d.Write(data)
	d.Close()
	return string(buf.Bytes()), err
}

// DumpWriter is the interface that describes how to write out dumps produced by
// the Dumper.  Implement this interface to customize dump output behavior.
type DumpWriter interface {
	// WriteAtom is called by the Dumper for each atom it decodes.
	WriteAtom(atom DumpAtom)
	// WriteStatus is called by the Dumper to indicate the status of the dumper.
	WriteStatus(status DumpStatus)
}

// NewDumpWriter returns a DumpWriter that outputs dumps to w, writing each atom
// and status on its own line, in their default string format.
func NewDumpWriter(w io.Writer) DumpWriter {
	return dumpWriter{w}
}

type dumpWriter struct {
	w io.Writer
}

func (w dumpWriter) WriteAtom(atom DumpAtom) {
	fmt.Fprintln(w.w, atom)
}

func (w dumpWriter) WriteStatus(status DumpStatus) {
	id := verror.ErrorID(status.Err)
	if status.MsgLen == 0 && status.MsgN == 0 && (id == errDumperFlushed.ID || id == errDumperClosed.ID) {
		// Don't output status when we're waiting to decode the next message, and
		// we're either flushed or closed, to avoid cluttering the output.
		return
	}
	fmt.Fprintln(w.w, status)
}

// Dumper produces dumps of vom data.  It implements the io.WriteCloser
// interface; Data is fed to the dumper via Write, and Close must be called at
// the end of usage to release resources.
//
// Dumps of vom data consist of a single stream of DumpAtom and DumpStatus.
// Each atom describes a single piece of the vom encoding; the vom encoding is
// composed of a stream of atoms.  The status describes the state of the dumper
// at that point in the stream.
type Dumper struct {
	// The Dumper only contains channels to communicate with the dumpWorker, which
	// does all the actual work.
	cmdChan   chan<- dumpCmd
	closeChan <-chan struct{}
}

var _ io.WriteCloser = (*Dumper)(nil)

// NewDumper returns a new Dumper, which writes dumps of vom data to w.
//
// Close must be called on the returned Dumper to release resources.
func NewDumper(w DumpWriter) *Dumper {
	cmd, close := make(chan dumpCmd), make(chan struct{})
	startDumpWorker(cmd, close, w)
	return &Dumper{cmd, close}
}

// Close flushes buffered data and releases resources.  Close must be called
// exactly once, when the dumper is no longer needed.
func (d *Dumper) Close() error {
	d.Flush()
	close(d.cmdChan)
	<-d.closeChan
	return nil
}

// Flush flushes buffered data, and causes the dumper to restart decoding at the
// start of a new message.  This is useful if the previous data in the stream
// was corrupt, and subsequent data will be for new vom messages.  Previously
// buffered type information remains intact.
func (d *Dumper) Flush() error {
	done := make(chan error)
	d.cmdChan <- dumpCmd{nil, done}
	err := <-done
	return err
}

// Status triggers an explicit dump of the current status of the dumper to the
// DumpWriter.  Status is normally generated at the end of each each decoded
// message; call Status to get extra information for partial dumps and errors.
func (d *Dumper) Status() {
	done := make(chan error)
	d.cmdChan <- dumpCmd{[]byte{}, done}
	<-done
}

// Write implements the io.Writer interface method.  This is the mechanism by
// which data is fed into the dumper.
func (d *Dumper) Write(data []byte) (int, error) {
	if len(data) == 0 {
		// Nil data means Flush, and non-nil empty data means Status, so we must
		// ensure that normal writes never send 0-length data.
		return 0, nil
	}
	done := make(chan error)
	d.cmdChan <- dumpCmd{data, done}
	err := <-done
	return len(data), err
}

type dumpCmd struct {
	// data holds Write data, except nil means Flush, and empty means Status.
	data []byte
	// done is closed when the worker has finished the command.
	done chan error
}

// dumpWorker does all the actual work, in its own goroutine.  Commands are sent
// from the Dumper to the worker via the cmdChan, and the closeChan is closed
// when the worker has exited its goroutine.
//
// We run the worker in a separate goroutine to keep the dumping logic simple
// and synchronous; the worker essentially runs the regular vom decoder logic,
// annotated with extra dump information.  In theory we could implement the
// dumper without any extra goroutines, but that would require implementing a
// vom decoder that explicitly maintained the decoding stack (rather than simple
// recursive calls), which doesn't seem worth it.
//
// The reason we re-implement the vom decoding logic in the work rather than
// just adding the appropriate annotations to the regular vom decoder is for
// performance; we don't want to bloat the regular decoder with lots of dump
// annotations.
type dumpWorker struct {
	cmdChan   <-chan dumpCmd
	closeChan chan<- struct{}

	// We hold regular decoding state, and output dump information to w.
	w       DumpWriter
	buf     *decbuf
	typeDec *TypeDecoder
	status  DumpStatus
	version Version

	recReader     *recordingReader
	recDataReader *recordedDataReader
	redDataDec    *decoder81

	// Each Write call on the Dumper is passed to us on the cmdChan.  When we get
	// around to processing the Write data, we buffer any extra data, and hold on
	// to the done channel so that we can close it when all the data is processed.
	data      bytes.Buffer
	lastWrite chan<- error

	// Hold on to the done channel for Flush commands, so that we can close it
	// when the worker actually finishes decoding the current message.
	lastFlush chan<- error
}

func startDumpWorker(cmd <-chan dumpCmd, close chan<- struct{}, w DumpWriter) {
	worker := &dumpWorker{
		cmdChan:   cmd,
		closeChan: close,
		w:         w,
		typeDec:   newTypeDecoderInternal(nil),
	}
	worker.recReader = &recordingReader{r: worker}
	worker.recDataReader = &recordedDataReader{reader: worker.recReader}
	worker.redDataDec = &NewDecoder(worker.recDataReader).dec
	worker.buf = newDecbuf(worker.recReader)
	go worker.decodeLoop()
}

// Read implements the io.Reader method, and is our synchronization strategy.
// The worker decodeLoop is running in its own goroutine, and keeps trying to
// decode vom messages.  When the decoder runs out of data, it will trigger a
// Read call.
//
// Thus we're guaranteed that when Read is called, the worker decodeLoop is
// blocked waiting on the results.  This gives us a natural place to process all
// commands, and consume more data from Write calls.
func (d *dumpWorker) Read(data []byte) (int, error) {
	// If we have any data buffered up, just return it.
	if n, _ := d.data.Read(data); n > 0 || len(data) == 0 {
		return n, nil
	}
	// Otherwise we're done with all the buffered data.  Signal the last Write
	// call that all data has been processed.
	d.lastWriteDone(nil)
	// Wait for commands on the the cmd channel.
	for {
		select {
		case cmd, ok := <-d.cmdChan:
			if !ok {
				// Close called, return our special closed error.
				return 0, verror.New(errDumperClosed, nil)
			}
			switch {
			case cmd.data == nil:
				// Flush called, return our special flushed error.  The Flush is done
				// when the decoderLoop starts with a new message.
				d.lastFlush = cmd.done
				return 0, verror.New(errDumperFlushed, nil)
			case len(cmd.data) == 0:
				// Status called.
				d.writeStatus(nil, false)
				cmd.done <- nil
				close(cmd.done)
			default:
				// Write called.  Copy as much as we can into data, writing leftover
				// into our buffer.  Hold on to the cmd.done channel, so we can close it
				// when the data has all been read.
				n := copy(data, cmd.data)
				if n < len(cmd.data) {
					d.data.Write(cmd.data[n:])
				}
				d.lastWrite = cmd.done
				return n, nil
			}
		}
	}
}

// decodeLoop runs a loop synchronously decoding messages.  Calls to read from
// d.buf will eventually result in a call to d.Read, which allows us to handle
// special commands like Close, Flush and Status synchronously.
func (d *dumpWorker) decodeLoop() {
	for {
		err := d.decodeNextValue()
		d.writeStatus(err, true)
		switch {
		case verror.ErrorID(err) == errDumperClosed.ID:
			d.lastWriteDone(err)
			d.lastFlushDone(err)
			close(d.closeChan)
			return
		case err != nil:
			// Any error causes us to flush our buffers; otherwise we run the risk of
			// an infinite loop.
			d.buf.Reset()
			d.data.Reset()
			d.lastWriteDone(err)
			d.lastFlushDone(err)
		}
	}
}

func (d *dumpWorker) lastWriteDone(err error) {
	if d.lastWrite != nil {
		d.lastWrite <- err
		close(d.lastWrite)
		d.lastWrite = nil
	}
}

func (d *dumpWorker) lastFlushDone(err error) {
	if d.lastFlush != nil {
		d.lastFlush <- err
		close(d.lastFlush)
		d.lastFlush = nil
	}
}

// DumpStatus represents the state of the dumper.  It is written to the
// DumpWriter at the end of decoding each value, and may also be triggered
// explicitly via Dumper.Status calls to get information for partial dumps.
type DumpStatus struct {
	MsgId      int64
	MsgLen     int
	MsgN       int
	Buf        []byte
	Debug      string
	RefTypes   []*vdl.Type
	RefAnyLens []uint64
	Value      *vdl.Value
	Err        error
}

func (s DumpStatus) String() string {
	ret := fmt.Sprintf("DumpStatus{MsgId: %d", s.MsgId)
	if s.MsgLen != 0 {
		ret += fmt.Sprintf(", MsgLen: %d", s.MsgLen)
	}
	if s.MsgN != 0 {
		ret += fmt.Sprintf(", MsgN: %d", s.MsgN)
	}
	if len := len(s.Buf); len > 0 {
		ret += fmt.Sprintf(`, Buf(%d): "%x"`, len, s.Buf)
	}
	if s.Debug != "" {
		ret += fmt.Sprintf(", Debug: %q", s.Debug)
	}
	if s.Value.IsValid() {
		ret += fmt.Sprintf(", Value: %v", s.Value)
	}
	if s.Err != nil {
		ret += fmt.Sprintf(", Err: %v", s.Err)
	}
	return ret + "}"
}

func (a DumpAtom) String() string {
	dataFmt := "%20v"
	if _, isString := a.Data.Interface().(string); isString {
		dataFmt = "%20q"
	}
	ret := fmt.Sprintf("%-20x %-15v "+dataFmt, a.Bytes, a.Kind, a.Data.Interface())
	if a.Debug != "" {
		ret += fmt.Sprintf(" [%s]", a.Debug)
	}
	return ret
}

// writeStatus writes the current decoding status to the the DumpWriter.  It is
// called automatically after every message is decoded, and also on every error
// encountered during decoding.  It is also triggered by manual calls to
// Dumper.Status.
func (d *dumpWorker) writeStatus(err error, doneDecoding bool) {
	if doneDecoding {
		d.status.Err = err
		if err == nil {
			// Successful decoding, don't include the last "waiting..." debug message.
			d.status.Debug = ""
		}
	}
	// If we're stuck in the middle of a Read, the data we have so far is in the
	// decbuf.  Grab the data here for debugging.
	if buflen := d.buf.end - d.buf.beg; buflen > 0 {
		d.status.Buf = make([]byte, buflen)
		copy(d.status.Buf, d.buf.buf[d.buf.beg:d.buf.end])
	} else {
		d.status.Buf = nil
	}
	err = NewDecoder(bytes.NewReader(d.recReader.bytes)).Decode(&d.status.Value)
	d.w.WriteStatus(d.status)
	if doneDecoding {
		d.status = DumpStatus{}
	}
}

// prepareAtom sets the status.Debug message, and prepares the decbuf so that
// subsequent writeAtom calls can easily capture all data that's been read.
func (d *dumpWorker) prepareAtom(format string, v ...interface{}) {
	d.status.Debug = fmt.Sprintf(format, v...)
	d.buf.moveDataToFront()
}

// writeAtom writes an atom describing the chunk of data we just decoded.  In
// order to capture the data that was read, we rely on prepareAtom being called
// before the writeAtom call.
//
// The mechanism to capture the data is subtle.  In prepareAtom we moved all
// decbuf data to the front, setting decbuf.beg to 0.  Here we assume that all
// data in the decbuf up to the new value of decbuf.beg is what was read.
//
// This is tricky, and somewhat error-prone.  We're using this strategy so that
// we can share the raw decoding logic with the real decoder, while still
// keeping the raw decoding logic reasonably compact and fast.
func (d *dumpWorker) writeAtom(kind DumpKind, data Primitive, format string, v ...interface{}) {
	var bytes []byte
	if len := d.buf.beg; len > 0 {
		bytes = make([]byte, len)
		copy(bytes, d.buf.buf[:len])
	}
	d.w.WriteAtom(DumpAtom{
		Kind:  kind,
		Bytes: bytes,
		Data:  data,
		Debug: fmt.Sprintf(format, v...),
	})
	d.status.MsgN += len(bytes)
	d.buf.moveDataToFront()
}

func (d *dumpWorker) decodeNextValue() error {
	// Decode type messages until we get to the type of the next value.
	valType, err := d.decodeValueType()
	if err != nil {
		return err
	}
	// Decode value message.
	err = d.decodeValueMsg(valType)
	d.recDataReader.End(d.buf.beg)
	return err
}

func (d *dumpWorker) decodeValueType() (*vdl.Type, error) {
	for {
		// Decode the version byte. To make the dumper easier to use on partial
		// data, the version byte is optional, and is allowed to appear before any
		// type or value message. Note that this relies on 0x80 not being a valid
		// first byte of regular messages.
		if d.version == 0 {
			d.prepareAtom("waiting for version byte")
			if !d.buf.IsAvailable(1) {
				if err := d.buf.Fill(1); err != nil {
					return nil, err
				}
			}
			switch version := Version(d.buf.PeekAvailableByte()); version {
			case Version81:
				d.version = version
				d.buf.SkipAvailable(1)
				d.writeAtom(DumpKindVersion, PrimitivePByte{byte(version)}, version.String())
				d.writeStatus(nil, true)
			}
		}
		d.prepareAtom("waiting for message ID or control code")
		incomplete, err := binaryDecodeControlOnly(d.buf, WireCtrlTypeIncomplete)
		if err != nil {
			return nil, err
		}
		if incomplete {
			d.writeAtom(DumpKindControl, PrimitivePControl{ControlKindIncompleteType}, "incomplete type")
			d.prepareAtom("waiting for message ID")
		}
		id, err := binaryDecodeInt(d.buf)
		if err != nil {
			return nil, err
		}
		d.writeAtom(DumpKindMsgId, PrimitivePInt{id}, "")
		d.status.MsgId = id
		switch {
		case id == 0:
			return nil, verror.New(errDecodeZeroTypeID, nil)
		case id > 0:
			// This is a value message, the typeID is +id.
			tid := TypeId(+id)
			tt, err := d.typeDec.lookupType(tid)
			if err != nil {
				d.writeAtom(DumpKindValueMsg, PrimitivePUint{uint64(tid)}, "%v", err)
				return nil, err
			}
			d.writeAtom(DumpKindValueMsg, PrimitivePUint{uint64(tid)}, "%v", tt)
			return tt, nil
		}
		// This is a type message, the typeID is -id.
		tid := TypeId(-id)
		d.writeAtom(DumpKindTypeMsg, PrimitivePUint{uint64(tid)}, "")
		// Decode the wireType like a regular value, and store it in typeDec.  The
		// type will actually be built when a value message arrives using this tid.
		if err := d.decodeValueMsg(wireTypeType); err != nil {
			return nil, err
		}
		var wt wireType
		if _, err := d.redDataDec.decodeWireType(&wt); err != nil {
			return nil, err
		}
		d.recDataReader.End(d.buf.beg)
		if err := d.typeDec.addWireType(tid, wt); err != nil {
			return nil, err
		}
		if !incomplete {
			if err := d.typeDec.buildType(tid); d.version >= Version81 && err != nil {
				return nil, err
			}
		}
		d.writeStatus(nil, true)
	}
}

// decodeValueMsg decodes the rest of the message assuming type t, handling the
// optional message length.
func (d *dumpWorker) decodeValueMsg(tt *vdl.Type) error {
	if d.version >= Version81 && (containsAny(tt) || containsTypeObject(tt)) {
		d.prepareAtom("waiting for reference type ids")
		tidsLen, err := binaryDecodeLen(d.buf)
		if err != nil {
			return err
		}
		d.writeAtom(DumpKindTypeIdsLen, PrimitivePUint{uint64(tidsLen)}, "")
		d.status.RefTypes = make([]*vdl.Type, tidsLen)
		for i := range d.status.RefTypes {
			d.prepareAtom("waiting for type id")
			tid, err := binaryDecodeUint(d.buf)
			if err != nil {
				return err
			}
			d.status.RefTypes[i], err = d.typeDec.lookupType(TypeId(tid))
			d.writeAtom(DumpKindTypeId, PrimitivePUint{tid}, "")
		}
		if containsAny(tt) {
			d.prepareAtom("waiting for any length list length")
			anyLensLen, err := binaryDecodeLen(d.buf)
			if err != nil {
				return err
			}
			d.writeAtom(DumpKindAnyLensLen, PrimitivePUint{uint64(anyLensLen)}, "")
			d.status.RefAnyLens = make([]uint64, anyLensLen)
			for i := 0; i < anyLensLen; i++ {
				d.prepareAtom("waiting for any len")
				anyMsgLen, err := binaryDecodeUint(d.buf)
				if err != nil {
					return err
				}
				d.status.RefAnyLens[i] = anyMsgLen
				d.writeAtom(DumpKindAnyMsgLen, PrimitivePUint{anyMsgLen}, "")
			}
		}
	}
	if hasChunkLen(tt) {
		d.prepareAtom("waiting for message len")
		msgLen, err := binaryDecodeLen(d.buf)
		if err != nil {
			return err
		}
		d.writeAtom(DumpKindMsgLen, PrimitivePUint{uint64(msgLen)}, "")
		d.status.MsgLen = msgLen
		d.status.MsgN = 0 // Make MsgN match up with MsgLen when successful.
		d.buf.SetLimit(msgLen)
	}
	err := d.decodeValue(tt)
	leftover := d.buf.RemoveLimit()
	switch {
	case err != nil:
		return err
	case leftover > 0:
		return verror.New(errLeftOverBytes, nil, leftover)
	}
	return nil
}

// decodeValue decodes the rest of the message assuming type tt.
func (d *dumpWorker) decodeValue(tt *vdl.Type) error {
	ttFrom := tt
	if tt.Kind() == vdl.Optional {
		d.prepareAtom("waiting for optional control byte")
		// If the type is optional, we expect to see either WireCtrlNil or the actual
		// value, but not both.  And thus, we can just peek for the WireCtrlNil here.
		switch ctrl, err := binaryPeekControl(d.buf); {
		case err != nil:
			return err
		case ctrl == WireCtrlNil:
			d.buf.SkipAvailable(1)
			d.writeAtom(DumpKindControl, PrimitivePControl{ControlKindNil}, "%v is nil", ttFrom)
			return nil
		}
		tt = tt.Elem()
	}
	if tt.IsBytes() {
		d.prepareAtom("waiting for bytes len")
		len, err := binaryDecodeLenOrArrayLen(d.buf, ttFrom)
		if err != nil {
			return err
		}
		d.writeAtom(DumpKindByteLen, PrimitivePUint{uint64(len)}, "bytes len")
		d.prepareAtom("waiting for bytes data")
		data := make([]byte, len)
		if err := d.buf.ReadIntoBuf(data); err != nil {
			return err
		}
		d.writeAtom(DumpKindPrimValue, PrimitivePString{string(data)}, "bytes")
		return nil
	}
	switch kind := tt.Kind(); kind {
	case vdl.Bool:
		d.prepareAtom("waiting for bool value")
		var v bool
		var err error
		switch d.version {
		case Version80:
			v, err = binaryDecodeBool(d.buf)
		default:
			var uv uint64
			uv, err = binaryDecodeUint(d.buf)
			v = (uv == 1)
		}
		if err != nil {
			return err
		}
		d.writeAtom(DumpKindPrimValue, PrimitivePBool{v}, "bool")
		return nil
	case vdl.Byte:
		d.prepareAtom("waiting for byte value")
		v, err := binaryDecodeUint(d.buf)
		if err != nil {
			return err
		}
		d.writeAtom(DumpKindPrimValue, PrimitivePByte{byte(v)}, "byte")
		return nil
	case vdl.Uint16, vdl.Uint32, vdl.Uint64:
		d.prepareAtom("waiting for uint value")
		v, err := binaryDecodeUint(d.buf)
		if err != nil {
			return err
		}
		d.writeAtom(DumpKindPrimValue, PrimitivePUint{v}, "uint")
		return nil
	case vdl.Int8, vdl.Int16, vdl.Int32, vdl.Int64:
		d.prepareAtom("waiting for int value")
		v, err := binaryDecodeInt(d.buf)
		if err != nil {
			return err
		}
		d.writeAtom(DumpKindPrimValue, PrimitivePInt{v}, "int")
		return nil
	case vdl.Float32, vdl.Float64:
		d.prepareAtom("waiting for float value")
		v, err := binaryDecodeFloat(d.buf)
		if err != nil {
			return err
		}
		d.writeAtom(DumpKindPrimValue, PrimitivePFloat{v}, "float")
		return nil
	case vdl.String:
		d.prepareAtom("waiting for string len")
		len, err := binaryDecodeLen(d.buf)
		if err != nil {
			return err
		}
		d.writeAtom(DumpKindByteLen, PrimitivePUint{uint64(len)}, "string len")
		d.prepareAtom("waiting for string data")
		data := make([]byte, len)
		if err := d.buf.ReadIntoBuf(data); err != nil {
			return err
		}
		d.writeAtom(DumpKindPrimValue, PrimitivePString{string(data)}, "string")
		return nil
	case vdl.Enum:
		d.prepareAtom("waiting for enum index")
		index, err := binaryDecodeUint(d.buf)
		if err != nil {
			return err
		}
		if index >= uint64(tt.NumEnumLabel()) {
			d.writeAtom(DumpKindIndex, PrimitivePUint{index}, "out of range for %v", tt)
			return verror.New(errIndexOutOfRange, nil)
		}
		label := tt.EnumLabel(int(index))
		d.writeAtom(DumpKindIndex, PrimitivePUint{index}, "%v.%v", tt.Name(), label)
		return nil
	case vdl.TypeObject:
		d.prepareAtom("waiting for typeobject ID")
		id, err := binaryDecodeUint(d.buf)
		if err != nil {
			return err
		}
		var typeobj *vdl.Type
		switch d.version {
		case Version80:
			typeobj, err = d.typeDec.lookupType(TypeId(id))
		default:
			if id >= uint64(len(d.status.RefTypes)) {
				return fmt.Errorf("type index %d out of bounds", id)
			}
			typeobj = d.status.RefTypes[id]
		}
		if err != nil {
			d.writeAtom(DumpKindTypeId, PrimitivePUint{id}, "%v", err)
			return err
		}
		d.writeAtom(DumpKindTypeId, PrimitivePUint{id}, "%v", typeobj)
		return nil
	case vdl.Array, vdl.List:
		d.prepareAtom("waiting for list len")
		len, err := binaryDecodeLenOrArrayLen(d.buf, tt)
		if err != nil {
			return err
		}
		d.writeAtom(DumpKindValueLen, PrimitivePUint{uint64(len)}, "list len")
		for ix := 0; ix < len; ix++ {
			if err := d.decodeValue(tt.Elem()); err != nil {
				return err
			}
		}
		return nil
	case vdl.Set:
		d.prepareAtom("waiting for set len")
		len, err := binaryDecodeLen(d.buf)
		if err != nil {
			return err
		}
		d.writeAtom(DumpKindValueLen, PrimitivePUint{uint64(len)}, "set len")
		for ix := 0; ix < len; ix++ {
			if err := d.decodeValue(tt.Key()); err != nil {
				return err
			}
		}
		return nil
	case vdl.Map:
		d.prepareAtom("waiting for map len")
		len, err := binaryDecodeLen(d.buf)
		if err != nil {
			return err
		}
		d.writeAtom(DumpKindValueLen, PrimitivePUint{uint64(len)}, "map len")
		for ix := 0; ix < len; ix++ {
			if err := d.decodeValue(tt.Key()); err != nil {
				return err
			}
			if err := d.decodeValue(tt.Elem()); err != nil {
				return err
			}
		}
		return nil
	case vdl.Struct:
		// Loop through decoding the 0-based field index and corresponding field.
		for {
			d.prepareAtom("waiting for struct field index")
			switch ok, err := binaryDecodeControlOnly(d.buf, WireCtrlEnd); {
			case err != nil:
				return err
			case ok:
				d.writeAtom(DumpKindControl, PrimitivePControl{ControlKindEnd}, "%v END", tt.Name())
				return nil
			}
			index, err := binaryDecodeUint(d.buf)
			switch {
			case err != nil:
				return err
			case index >= uint64(tt.NumField()):
				d.writeAtom(DumpKindIndex, PrimitivePUint{index}, "out of range for %v", tt)
				return verror.New(errIndexOutOfRange, nil)
			}
			ttfield := tt.Field(int(index))
			d.writeAtom(DumpKindIndex, PrimitivePUint{index}, "%v.%v", tt.Name(), ttfield.Name)
			if err := d.decodeValue(ttfield.Type); err != nil {
				return err
			}
		}
	case vdl.Union:
		d.prepareAtom("waiting for union field index")
		index, err := binaryDecodeUint(d.buf)
		switch {
		case err != nil:
			return err
		case index >= uint64(tt.NumField()):
			d.writeAtom(DumpKindIndex, PrimitivePUint{index}, "out of range for %v", tt)
			return verror.New(errIndexOutOfRange, nil)
		}
		ttfield := tt.Field(int(index))
		if tt == wireTypeType {
			// Pretty-print for wire type definition messages.
			d.writeAtom(DumpKindWireTypeIndex, PrimitivePUint{index}, "%v", ttfield.Type.Name())
		} else {
			d.writeAtom(DumpKindIndex, PrimitivePUint{index}, "%v.%v", tt.Name(), ttfield.Name)
		}
		if err := d.decodeValue(ttfield.Type); err != nil {
			return err
		}
		return nil
	case vdl.Any:
		d.prepareAtom("waiting for any typeID")
		switch ok, err := binaryDecodeControlOnly(d.buf, WireCtrlNil); {
		case err != nil:
			return err
		case ok:
			d.writeAtom(DumpKindControl, PrimitivePControl{ControlKindNil}, "any(nil)")
			return nil
		}
		switch id, err := binaryDecodeUint(d.buf); {
		case err != nil:
			return err
		default:
			var err error
			var elemType *vdl.Type
			switch d.version {
			case Version80:
				elemType, err = d.typeDec.lookupType(TypeId(id))
			default:
				if id >= uint64(len(d.status.RefTypes)) {
					return fmt.Errorf("type index %d out of bounds", id)
				}
				elemType = d.status.RefTypes[id]
			}
			if err != nil {
				d.writeAtom(DumpKindTypeId, PrimitivePUint{id}, "%v", err)
				return err
			}
			d.writeAtom(DumpKindTypeId, PrimitivePUint{id}, "%v", elemType)
			if d.version >= Version81 {
				d.prepareAtom("waiting for any message length index")
				switch index, err := binaryDecodeUint(d.buf); {
				case err != nil:
					return err
				default:
					if index >= uint64(len(d.status.RefAnyLens)) {
						return fmt.Errorf("any len index %d out of bounds", index)
					}
					d.writeAtom(DumpKindAnyMsgLen, PrimitivePUint{index}, "len %v", d.status.RefAnyLens[index])
					return d.decodeValue(elemType)
				}
			}
			return d.decodeValue(elemType)
		}
	default:
		panic(verror.New(errDecodeValueUnhandledType, nil, tt))
	}
}

// recordingReader delegates reads to the underlying reader, but stores
// the resulting bytes.
type recordingReader struct {
	bytes []byte // TODO(toddw) don't accumulate bytes forever
	r     io.Reader
}

func (r *recordingReader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if n > 0 {
		r.bytes = append(r.bytes, p[:n]...)
	}
	return
}

// recordedDataReader reads from the buffer of a recordingReader.
// Recording can continue while this reader is being used.
type recordedDataReader struct {
	reader *recordingReader
	pos    int
}

func (r *recordedDataReader) Read(p []byte) (n int, err error) {
	if r.pos == len(r.reader.bytes) && len(p) > 0 {
		return 0, io.EOF
	}
	n = copy(p, r.reader.bytes[r.pos:])
	r.pos += n
	return
}

func (r *recordedDataReader) End(readInBuf int) {
	r.pos += readInBuf
}
