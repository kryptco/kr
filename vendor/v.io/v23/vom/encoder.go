// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vom

import (
	"errors"
	"fmt"
	"io"

	"v.io/v23/vdl"
	"v.io/v23/verror"
)

func (v Version) String() string {
	return fmt.Sprintf("Version%x", byte(v))
}

var (
	errEmptyEncoderStack       = errors.New("vom: empty encoder stack")
	errEntriesMustSetLenHint   = errors.New("vom: entries must set LenHint")
	errLabelNotInType          = verror.Register(pkgPath+".errLabelNotInType", verror.NoRetry, "{1:}{2:} enum label {3} doesn't exist in type {4}{:_}")
	errUnsupportedInVOMVersion = verror.Register(pkgPath+".errUnsupportedInVOMVersion", verror.NoRetry, "{1:}{2:} {3} unsupported in vom version {4}{:_}")
	errUnusedTypeIds           = verror.Register(pkgPath+".errUnusedTypeIds", verror.NoRetry, "{1:}{2:} vom: some type ids unused during encode {:_}")
	errUnusedAnys              = verror.Register(pkgPath+".errUnusedAnys", verror.NoRetry, "{1:}{2:} vom: some anys unused during encode {:_}")
)

const (
	typeIDListInitialSize = 16
	anyLenListInitialSize = 16
)

// paddingLen must be large enough to hold the header in writeMsg.
const paddingLen = maxEncodedUintBytes * 2

// Encoder manages the transmission and marshaling of typed values to the other
// side of a connection.
type Encoder struct {
	enc encoder81
}

// NewEncoder returns a new Encoder that writes to the given writer in the VOM
// binary format.  The binary format is compact and fast.
func NewEncoder(w io.Writer) *Encoder {
	return NewVersionedEncoder(DefaultVersion, w)
}

// NewVersionedEncoder returns a new Encoder that writes to the given writer with
// the specified version.
func NewVersionedEncoder(version Version, w io.Writer) *Encoder {
	typeEnc := newTypeEncoderInternal(version, newEncoderForTypes(version, w))
	return NewVersionedEncoderWithTypeEncoder(version, w, typeEnc)
}

// NewEncoderWithTypeEncoder returns a new Encoder that writes to the given
// writer, where types are encoded separately through the typeEnc.
func NewEncoderWithTypeEncoder(w io.Writer, typeEnc *TypeEncoder) *Encoder {
	return NewVersionedEncoderWithTypeEncoder(DefaultVersion, w, typeEnc)
}

// NewVersionedEncoderWithTypeEncoder returns a new Encoder that writes to the
// given writer with the specified version, where types are encoded separately
// through the typeEnc.
func NewVersionedEncoderWithTypeEncoder(version Version, w io.Writer, typeEnc *TypeEncoder) *Encoder {
	if !isAllowedVersion(version) {
		panic(fmt.Sprintf("unsupported VOM version: %x", version))
	}
	return &Encoder{encoder81{
		writer:          w,
		buf:             newEncbuf(),
		typeEnc:         typeEnc,
		sentVersionByte: false,
		version:         version,
	}}
}

func newEncoderForTypes(version Version, w io.Writer) *encoder81 {
	if !isAllowedVersion(version) {
		panic(fmt.Sprintf("unsupported VOM version: %x", version))
	}
	buf := newEncbuf()
	e := &encoder81{
		writer:          w,
		buf:             buf,
		sentVersionByte: true,
		version:         version,
		mode:            encoderForTypes,
	}
	return e
}

func newEncoderForRawBytes(w io.Writer) *encoder81 {
	// RawBytes doesn't need the types to be encoded, since it holds the in-memory
	// representation.  We still need a type encoder to collect the unique types,
	// but we give it a dummy encoder that doesn't have any state set up.
	typeEnc := newTypeEncoderInternal(DefaultVersion, &encoder81{
		sentVersionByte: true,
		version:         DefaultVersion,
		mode:            encoderForRawBytes,
	})
	return &encoder81{
		writer:          w,
		buf:             newEncbuf(),
		typeEnc:         typeEnc,
		sentVersionByte: true,
		version:         DefaultVersion,
		mode:            encoderForRawBytes,
	}
}

// Encoder returns e as a vdl.Encoder.
func (e *Encoder) Encoder() vdl.Encoder {
	return &e.enc
}

// Encode transmits the value v.  Values of type T are encodable as long as the
// T is a valid vdl type.
func (e *Encoder) Encode(v interface{}) error {
	return vdl.Write(&e.enc, v)
}

type encoderMode int

const (
	encoderRegular     encoderMode = iota
	encoderForTypes                // encoder81 is embedded in TypeEncoder
	encoderForRawBytes             // encoder81 is used to encode RawBytes
)

type encoder81 struct {
	stack []encoderStackEntry
	// We use buf to buffer up the encoded value. The buffering is necessary so
	// that we can compute the total message length.
	buf *encbuf
	// Buffer for the header of messages with any or typeobject.
	bufHeader *encbuf
	// Underlying writer.
	writer io.Writer
	// All types are sent through typeEnc.
	typeEnc         *TypeEncoder
	sentVersionByte bool
	version         Version

	tids    *typeIDList
	anyLens *anyLenList

	mid                           int64
	hasLen, hasAny, hasTypeObject bool
	typeIncomplete                bool

	isParentBytes            bool // parent type is []byte or [N]byte
	nextStartValueIsOptional bool // next StartValue is an optional type

	// msgType captures the type of the top-level value.  Unlike stack[0].Type, it
	// also captures optionality for non-nil types.
	msgType *vdl.Type

	mode encoderMode
}

type encoderStackEntry struct {
	Type       *vdl.Type
	Index      int
	LenHint    int
	NumStarted int
	AnyRef     anyStartRef
}

// We can only determine whether the next value is AnyType
// by checking the next type of the entry.
func (entry *encoderStackEntry) nextValueIsAny() bool {
	switch tt := entry.Type; tt.Kind() {
	case vdl.List, vdl.Array:
		return tt.Elem() == vdl.AnyType
	case vdl.Set:
		return tt.Key() == vdl.AnyType
	case vdl.Map:
		// NumStarted is already incremented by the time we check it.
		if entry.NumStarted%2 == 1 {
			return tt.Key() == vdl.AnyType
		} else {
			return tt.Elem() == vdl.AnyType
		}
	case vdl.Struct, vdl.Union:
		return tt.Field(entry.Index).Type == vdl.AnyType
	}
	return false
}

func (e *encoder81) top() *encoderStackEntry {
	if len(e.stack) == 0 {
		return nil
	}
	return &e.stack[len(e.stack)-1]
}

func (e *encoder81) encodeWireType(tid TypeId, wt wireType, typeIncomplete bool) error {
	// RawBytes doesn't need type messages to be encoded, since it holds the types
	// in-memory.
	if e.mode == encoderForRawBytes {
		return nil
	}
	// Set up the state that would normally be set in startMessage, and use
	// VDLWrite to encode wt as a regular value.
	e.mid = int64(-tid)
	e.typeIncomplete = typeIncomplete
	e.hasAny = false
	e.hasTypeObject = false
	e.hasLen = true
	e.tids = nil
	e.anyLens = nil
	return wt.VDLWrite(e)
}

func (e *encoder81) SetNextStartValueIsOptional() {
	e.nextStartValueIsOptional = true
}

func (e *encoder81) NilValue(tt *vdl.Type) error {
	switch tt.Kind() {
	case vdl.Any, vdl.Optional:
	default:
		return fmt.Errorf("concrete types disallowed for NilValue (type was %v)", tt)
	}
	// Handle StartValue logic.
	top, isOptionalInsideAny := e.top(), false
	if top == nil {
		if err := e.startMessage(tt); err != nil {
			return err
		}
	} else {
		isOptionalInsideAny = top.nextValueIsAny() && tt.Kind() == vdl.Optional
	}
	var anyRef anyStartRef
	if isOptionalInsideAny {
		tid, err := e.typeEnc.encode(tt)
		if err != nil {
			return err
		}
		binaryEncodeUint(e.buf, e.tids.ReferenceTypeID(tid))
		anyRef = e.anyLens.StartAny(e.buf.Len())
		binaryEncodeUint(e.buf, uint64(anyRef.index))
	}
	// Encode the nil control byte.
	binaryEncodeControl(e.buf, WireCtrlNil)
	// Handle FinishValue logic.
	if isOptionalInsideAny {
		e.anyLens.FinishAny(anyRef, e.buf.Len())
	}
	if top == nil {
		if err := e.finishMessage(); err != nil {
			return err
		}
	}
	e.nextStartValueIsOptional = false
	return nil
}

func (e *encoder81) StartValue(tt *vdl.Type) error {
	switch tt.Kind() {
	case vdl.Any, vdl.Optional:
		return fmt.Errorf("only concrete types allowed for StartValue (type was %v)", tt)
	}
	var anyRef anyStartRef
	top := e.top()
	if top == nil {
		e.isParentBytes = false
		msgType := tt
		if e.nextStartValueIsOptional {
			msgType = vdl.OptionalType(tt)
		}
		if err := e.startMessage(msgType); err != nil {
			return err
		}
	} else {
		e.isParentBytes = top.Type.IsBytes()
		top.NumStarted++
		// Handle Any types, which need the Any header to be written.
		if top.nextValueIsAny() {
			anyType := tt
			if e.nextStartValueIsOptional {
				anyType = vdl.OptionalType(tt)
			}
			tid, err := e.typeEnc.encode(anyType)
			if err != nil {
				return err
			}
			binaryEncodeUint(e.buf, e.tids.ReferenceTypeID(tid))
			anyRef = e.anyLens.StartAny(e.buf.Len())
			binaryEncodeUint(e.buf, uint64(anyRef.index))
		}
	}
	// Add entry to the stack.
	e.stack = append(e.stack, encoderStackEntry{
		Type:    tt,
		Index:   -1,
		LenHint: -1,
		AnyRef:  anyRef,
	})
	e.nextStartValueIsOptional = false
	return nil
}

func (e *encoder81) startMessage(tt *vdl.Type) error {
	e.buf.Reset()
	e.buf.Grow(paddingLen)
	e.msgType = tt
	if e.mode == encoderForTypes {
		// We've already set up the state in encodeWireType.
		return nil
	}
	if !e.sentVersionByte {
		if _, err := e.writer.Write([]byte{byte(e.version)}); err != nil {
			return err
		}
		e.sentVersionByte = true
	}
	tid, err := e.typeEnc.encode(tt)
	if err != nil {
		return err
	}
	e.hasLen = hasChunkLen(tt)
	e.hasAny = containsAny(tt)
	e.hasTypeObject = containsTypeObject(tt)
	e.typeIncomplete = false
	e.mid = int64(tid)
	if e.hasAny || e.hasTypeObject {
		e.tids = newTypeIDList()
	} else {
		e.tids = nil
	}
	if e.hasAny {
		e.anyLens = newAnyLenList()
	} else {
		e.anyLens = nil
	}
	return nil
}

func (e *encoder81) FinishValue() error {
	e.isParentBytes = false
	oldStackTop := len(e.stack) - 1
	if oldStackTop == -1 {
		return errEmptyDecoderStack
	}
	anyRef := e.stack[oldStackTop].AnyRef
	e.stack = e.stack[:oldStackTop]
	if oldStackTop == 0 {
		return e.finishMessage()
	}
	if newTop := e.stack[oldStackTop-1]; newTop.nextValueIsAny() {
		e.anyLens.FinishAny(anyRef, e.buf.Len())
	}
	return nil
}

func (e *encoder81) finishMessage() error {
	if e.mode == encoderForRawBytes {
		// Only encode the value portion for RawBytes.
		msg := e.buf.Bytes()
		_, err := e.writer.Write(msg[paddingLen:])
		return err
	}
	if e.typeIncomplete {
		if _, err := e.writer.Write([]byte{WireCtrlTypeIncomplete}); err != nil {
			return err
		}
	}
	if e.hasAny || e.hasTypeObject {
		ids := e.tids.NewIDs()
		var anyLens []int
		if e.hasAny {
			anyLens = e.anyLens.NewAnyLens()
		}
		if e.bufHeader == nil {
			e.bufHeader = newEncbuf()
		} else {
			e.bufHeader.Reset()
		}
		binaryEncodeInt(e.bufHeader, e.mid)
		binaryEncodeUint(e.bufHeader, uint64(len(ids)))
		for _, id := range ids {
			binaryEncodeUint(e.bufHeader, uint64(id))
		}
		if e.hasAny {
			binaryEncodeUint(e.bufHeader, uint64(len(anyLens)))
			for _, anyLen := range anyLens {
				binaryEncodeUint(e.bufHeader, uint64(anyLen))
			}
		}
		msg := e.buf.Bytes()
		if e.hasLen {
			binaryEncodeUint(e.bufHeader, uint64(len(msg)-paddingLen))
		}
		if _, err := e.writer.Write(e.bufHeader.Bytes()); err != nil {
			return err
		}
		_, err := e.writer.Write(msg[paddingLen:])
		return err
	}
	msg := e.buf.Bytes()
	header := msg[:paddingLen]
	if e.hasLen {
		start := binaryEncodeUintEnd(header, uint64(len(msg)-paddingLen))
		header = header[:start]
	}
	start := binaryEncodeIntEnd(header, e.mid)
	_, err := e.writer.Write(msg[start:])
	return err
}

func (e *encoder81) NextEntry(done bool) error {
	top := e.top()
	if top == nil {
		return errEmptyEncoderStack
	}
	top.Index++
	if top.Index == 0 {
		switch {
		case top.Type.Kind() == vdl.Array:
			binaryEncodeUint(e.buf, 0)
		case top.LenHint >= 0:
			binaryEncodeUint(e.buf, uint64(top.LenHint))
		}
	}
	if done && top.Type.Kind() != vdl.Array && top.LenHint == -1 {
		// TODO(toddw): Emit collection terminator.
		// binaryEncodeControl(e.buf, WireCtrlCollectionTerminator)
		return errEntriesMustSetLenHint
	}
	return nil
}

func (e *encoder81) NextField(index int) error {
	top := e.top()
	if top == nil {
		return errEmptyEncoderStack
	}
	if index < -1 || index >= top.Type.NumField() {
		return fmt.Errorf("vom: NextField called with invalid index %d", index)
	}
	if index == -1 {
		if top.Type.Kind() == vdl.Struct {
			binaryEncodeControl(e.buf, WireCtrlEnd)
		}
		return nil
	}
	binaryEncodeUint(e.buf, uint64(index))
	top.Index = index
	return nil
}

func (e *encoder81) SetLenHint(lenHint int) error {
	top := e.top()
	if top == nil {
		return errEmptyEncoderStack
	}
	switch top.Type.Kind() {
	case vdl.List, vdl.Set, vdl.Map:
	default:
		fmt.Errorf("SetLenHint illegal for type %v", top.Type)
	}
	top.LenHint = lenHint
	return nil
}

func (e *encoder81) EncodeBool(value bool) error {
	binaryEncodeBool(e.buf, value)
	return nil
}

func (e *encoder81) EncodeUint(value uint64) error {
	// Handle a special-case where normally single bytes are written out as
	// variable sized numbers, which use 2 bytes to encode bytes > 127.  But each
	// byte contained in a list or array is written out as one byte.  E.g.
	//   byte(0x81)         -> 0xFF81   : single byte with variable-size
	//   []byte("\x81\x82") -> 0x028182 : each elem byte encoded as one byte
	if e.isParentBytes {
		e.buf.WriteOneByte(byte(value))
	} else {
		binaryEncodeUint(e.buf, value)
	}
	return nil
}

func (e *encoder81) EncodeInt(value int64) error {
	binaryEncodeInt(e.buf, value)
	return nil
}

func (e *encoder81) EncodeFloat(value float64) error {
	binaryEncodeFloat(e.buf, value)
	return nil
}

func (e *encoder81) EncodeBytes(value []byte) error {
	top := e.top()
	if top == nil {
		return errEmptyEncoderStack
	}
	if top.Type.Kind() == vdl.Array {
		binaryEncodeUint(e.buf, 0)
	} else {
		binaryEncodeUint(e.buf, uint64(len(value)))
	}
	e.buf.Write(value)
	return nil
}

func (e *encoder81) EncodeString(value string) error {
	top := e.top()
	if top == nil {
		return errEmptyEncoderStack
	}
	if tt := top.Type; tt.Kind() == vdl.Enum {
		index := tt.EnumIndex(value)
		if index < 0 {
			return verror.New(errLabelNotInType, nil, value, tt)
		}
		binaryEncodeUint(e.buf, uint64(index))
	} else {
		binaryEncodeUint(e.buf, uint64(len(value)))
		e.buf.WriteString(value)
	}
	return nil
}

func (e *encoder81) EncodeTypeObject(value *vdl.Type) error {
	tid, err := e.typeEnc.encode(value)
	if err != nil {
		return err
	}
	binaryEncodeUint(e.buf, e.tids.ReferenceTypeID(tid))
	return nil
}

// writeRawBytes writes rb to e.  This only works if e at the top-level; if it
// has already encoded some values, rb.Data needs to be re-written with new
// indices for type ids and any lengths.
//
// REQUIRES: e.version == rb.Version && len(e.stack) == 0
//
// TODO(toddw): Code a variant of this that performs the re-writing.
func (e *encoder81) writeRawBytes(rb *RawBytes) error {
	if rb.IsNil() {
		return e.NilValue(rb.Type)
	}
	tt := rb.Type
	if tt.Kind() == vdl.Optional {
		e.SetNextStartValueIsOptional()
		tt = tt.Elem()
	}
	if err := e.StartValue(tt); err != nil {
		return err
	}
	if containsAny(tt) || containsTypeObject(tt) {
		for _, refType := range rb.RefTypes {
			tid, err := e.typeEnc.encode(refType)
			if err != nil {
				return err
			}
			e.tids.ReferenceTypeID(tid)
		}
	}
	if containsAny(tt) {
		e.anyLens.lens = rb.AnyLengths
	}
	e.buf.Write(rb.Data)
	return e.FinishValue()
}

func newTypeIDList() *typeIDList {
	return &typeIDList{
		tids: make([]TypeId, 0, typeIDListInitialSize),
	}
}

type typeIDList struct {
	tids      []TypeId
	totalSent int
}

func (l *typeIDList) ReferenceTypeID(tid TypeId) uint64 {
	for index, existingTid := range l.tids {
		if existingTid == tid {
			return uint64(index)
		}
	}

	l.tids = append(l.tids, tid)
	return uint64(len(l.tids) - 1)
}

func (l *typeIDList) Reset() error {
	if l.totalSent != len(l.tids) {
		return verror.New(errUnusedTypeIds, nil)
	}
	l.tids = l.tids[:0]
	l.totalSent = 0
	return nil
}

func (l *typeIDList) NewIDs() []TypeId {
	var newIDs []TypeId
	if l.totalSent < len(l.tids) {
		newIDs = l.tids[l.totalSent:]
	}
	l.totalSent = len(l.tids)
	return newIDs
}

func newAnyLenList() *anyLenList {
	return &anyLenList{
		lens: make([]int, 0, anyLenListInitialSize),
	}
}

type anyStartRef struct {
	index  int // index into the anyLen list
	marker int // position marker for the start of the any
}

type anyLenList struct {
	lens      []int
	totalSent int
}

func (l *anyLenList) StartAny(startMarker int) anyStartRef {
	l.lens = append(l.lens, 0)
	index := len(l.lens) - 1
	return anyStartRef{
		index:  index,
		marker: startMarker + lenUint(uint64(index)),
	}
}

func (l *anyLenList) FinishAny(start anyStartRef, endMarker int) {
	l.lens[start.index] = endMarker - start.marker
}

func (l *anyLenList) Reset() error {
	if l.totalSent != len(l.lens) {
		return verror.New(errUnusedAnys, nil)
	}
	l.lens = l.lens[:0]
	l.totalSent = 0
	return nil
}

func (l *anyLenList) NewAnyLens() []int {
	var newAnyLens []int
	if l.totalSent < len(l.lens) {
		newAnyLens = l.lens[l.totalSent:]
	}
	l.totalSent = len(l.lens)
	return newAnyLens
}
