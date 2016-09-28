// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"sync"
)

var (
	errEmptyPipeStack        = errors.New("vdl: empty pipe stack")
	errEncCallDuringDecPhase = errors.New("vdl: pipe encoder method called during decode phase")
	errInvalidPipeState      = errors.New("vdl: invalid pipe state")
)

func convertPipe(dst, src interface{}) error {
	enc, dec := newPipe()
	go func() {
		enc.Close(Write(enc, src))
	}()
	return dec.Close(Read(dec, dst))
}

func convertPipeReflect(dst, src reflect.Value) error {
	enc, dec := newPipe()
	go func() {
		enc.Close(WriteReflect(enc, src))
	}()
	return dec.Close(ReadReflect(dec, dst))
}

// newPipe returns a pipeEncoder and pipeDecoder connected via the same "pipe".
// Conceptually this is similar to an os.Pipe between a io.Writer and io.Reader.
// It is used to pair together vdl.Write and vdl.Read calls during conversion.
//
// Note that vdl.Write calls a sequence of pipeEncoder methods, while vdl.Read
// calls a sequence of pipeDecoder methods.  The concept behind the pipe is that
// we match each {Start,Finish}Value on the pipeEncoder with a corresponding
// call on the pipeDecoder.  Only one call is allowed to be in-flight at a time.
//
// We bootstrap with the first enc.StartValue call.  After this first call,
// every subsequent call to enc.{Start,Finish}Value blocks, waiting for the
// previous call to be matched by a call on the decoder.  Similarly every call
// to dec.{Start,Finish}Value blocks, waiting to match the encoder call.
// Control alternates between the Encoder and Decoder until completion.
//
// The Close method shuts the pipe down, causing everything to unblock.
func newPipe() (*pipeEncoder, *pipeDecoder) {
	d := &pipeDecoder{Enc: pipeEncoder{}}
	d.Enc.Cond.L = &d.Enc.Mutex
	return &d.Enc, d
}

type pipeEncoder struct {
	sync.Mutex
	Cond sync.Cond

	Stack                    []pipeStackEntry
	NextEntryDone            bool
	NextFieldIndex           int
	NextStartValueIsOptional bool       // The StartValue refers to an optional type.
	NumberType               numberType // The number type X in EncodeX.

	// EncIsBytes, DecBytesAsEntries and DecByteStartValue deal with a subtlety
	// when handling bytes.  Normally the {Start,Finish}Value calls on the Encoder
	// are perfectly matched by the same calls on the Decoder; the pipe's blocking
	// mechanism depends on this.  But since we allow value conversions, it's
	// valid for the following mismatched sequences to occur:
	//    # Encode bytes, decode as an array or list
	//    Encoder: StartValue,EncodeBytes,                           FinishValue
	//    Decoder: StartValue,(NextEntry,StartValue,FinishValue,...),FinishValue
	//
	//    # Encode array or list, decode as bytes
	//    Encoder: StartValue,(NextEntry,StartValue,FinishValue,...),FinishValue
	//    Decoder: StartValue,DecodeBytes,                           FinishValue
	//
	// EncIsBytes is set in enc.EncodeBytes, and cleared in enc.FinishValue.
	// DecBytesAsEntries is set in dec.NextEntry, and cleared in dec.FinishValue.
	// DecByteStartValue is set in dec.StartValue, and cleared in dec.FinishValue.
	// The concept is EncIsBytes tracks the encoder bytes state.  If it is set,
	// and dec.NextEntry is called, it means the caller is decoding the bytes as a
	// sequence of entries.  Thereafter pairs of dec.{Start,FinishValue} calls are
	// allowed to proceed without blocking.  The final dec.FinishValue resumes the
	// regular pattern of blocking on matching calls.
	EncIsBytes        bool
	DecBytesAsEntries bool
	DecByteStartValue bool

	DecStarted bool // Decoding has started
	Err        error

	State pipeState

	// Arguments from Encode* to be passed to Decode*:
	ArgBool   bool
	ArgUint   uint64
	ArgInt    int64
	ArgFloat  float64
	ArgString string
	ArgBytes  []byte
	ArgType   *Type
}

type pipeStackEntry struct {
	Type       *Type
	NextOp     pipeOp
	LenHint    int
	Index      int
	NumStarted int
	IsOptional bool
	IsNil      bool
}

type pipeDecoder struct {
	Enc                  pipeEncoder
	ignoreNextStartValue bool
}

// pipeState represents the blocking state of the pipe.  Control is transferred
// alternately between the encoder and decoder; while one side is running, the
// other side blocks.  Once the closed state is entered, everything unblocks.
type pipeState int

const (
	pipeStateEncoder pipeState = iota
	pipeStateDecoder
	pipeStateClosed
)

func (x pipeState) String() string {
	switch x {
	case pipeStateEncoder:
		return "Encoder"
	case pipeStateDecoder:
		return "Decoder"
	case pipeStateClosed:
		return "Closed"
	default:
		panic(fmt.Errorf("vdl: unknown pipeState %d", x))
	}
}

// pipeOp is used to check our invariants for state transitions.
type pipeOp int

const (
	pipeStartEnc pipeOp = iota
	pipeStartDec
	pipeFinishEnc
	pipeFinishDec
)

func (op pipeOp) String() string {
	switch op {
	case pipeStartEnc:
		return "StartEnc"
	case pipeStartDec:
		return "StartDec"
	case pipeFinishEnc:
		return "FinishEnc"
	case pipeFinishDec:
		return "FinishDec"
	default:
		panic("bad op")
	}
}

func (op pipeOp) Next() pipeOp {
	if op == pipeFinishDec {
		op = pipeStartEnc
	} else {
		op++
	}
	return op
}

// We can only determine whether the next value is AnyType
// by checking the next type of the entry.
func (entry *pipeStackEntry) nextValueIsAny() bool {
	switch entry.Type.Kind() {
	case List, Array:
		return entry.Type.Elem() == AnyType
	case Set:
		return entry.Type.Key() == AnyType
	case Map:
		switch entry.NumStarted % 2 {
		case 1:
			return entry.Type.Key() == AnyType
		case 0:
			return entry.Type.Elem() == AnyType
		}
	case Struct, Union:
		return entry.Type.Field(entry.Index).Type == AnyType
	}
	return false
}

type numberType int

const (
	numberUint numberType = iota
	numberInt
	numberFloat
)

func (e *pipeEncoder) top() *pipeStackEntry {
	if len(e.Stack) == 0 {
		return nil
	}
	return &e.Stack[len(e.Stack)-1]
}

func (d *pipeDecoder) top() *pipeStackEntry {
	if len(d.Enc.Stack) == 0 {
		return nil
	}
	return &d.Enc.Stack[len(d.Enc.Stack)-1]
}

func (d *pipeDecoder) Lock()   { d.Enc.Mutex.Lock() }
func (d *pipeDecoder) Unlock() { d.Enc.Mutex.Unlock() }

func (e *pipeEncoder) closeLocked(err error) error {
	if err != nil && e.Err == nil {
		e.Err = err
	}
	e.State = pipeStateClosed
	e.Cond.Broadcast()
	return e.Err
}

func (e *pipeEncoder) Close(err error) error {
	e.Lock()
	defer e.Unlock()
	return e.closeLocked(err)
}

func (d *pipeDecoder) Close(err error) error {
	return d.Enc.Close(err)
}

func (e *pipeEncoder) SetNextStartValueIsOptional() {
	e.NextStartValueIsOptional = true
}

func (e *pipeEncoder) NilValue(tt *Type) error {
	switch tt.Kind() {
	case Any:
	case Optional:
		e.SetNextStartValueIsOptional()
	default:
		return fmt.Errorf("concrete types disallowed for NilValue (type was %v)", tt)
	}
	if err := e.StartValue(tt); err != nil {
		return err
	}
	top := e.top()
	if top == nil {
		return e.Close(errEmptyPipeStack)
	}
	top.IsNil = true
	return e.FinishValue()
}

func (e *pipeEncoder) StartValue(tt *Type) error {
	e.Lock()
	defer e.Unlock()
	if e.State != pipeStateEncoder {
		return e.closeLocked(errInvalidPipeState)
	}
	if err := e.wait(); err != nil {
		return err
	}
	top := e.top()
	if top != nil {
		top.NumStarted++
	}
	e.Stack = append(e.Stack, pipeStackEntry{
		Type:       tt,
		NextOp:     pipeStartDec,
		Index:      -1,
		LenHint:    -1,
		IsOptional: e.NextStartValueIsOptional,
	})
	e.NextStartValueIsOptional = false
	return e.Err
}

func (e *pipeEncoder) FinishValue() error {
	e.Lock()
	defer e.Unlock()
	if e.State != pipeStateEncoder {
		return e.closeLocked(errInvalidPipeState)
	}
	if err := e.wait(); err != nil {
		return err
	}
	top := e.top()
	if top == nil {
		return e.closeLocked(errEmptyPipeStack)
	}
	if got, want := top.NextOp, pipeFinishEnc; got != want {
		return e.closeLocked(fmt.Errorf("vdl: pipe got state %v, want %v", got, want))
	}
	top.NextOp = top.NextOp.Next()
	// We're finished with the enc.EncodeBytes special case once we've unblocked,
	// regardless of what sequence of Decoder calls were made.
	e.EncIsBytes = false
	return e.Err
}

func (e *pipeEncoder) wait() error {
	top := e.top()
	if e.State == pipeStateClosed {
		return e.Err
	}
	if top != nil {
		e.State = pipeStateDecoder
		e.Cond.Broadcast()
		for e.State == pipeStateDecoder {
			e.Cond.Wait()
		}
		if e.State == pipeStateClosed {
			return e.Err
		}
	}
	return nil
}

func (d *pipeDecoder) StartValue(want *Type) error {
	d.Lock()
	defer d.Unlock()
	if d.Enc.DecStarted && d.Enc.State != pipeStateDecoder {
		return d.Enc.closeLocked(errInvalidPipeState)
	}
	if d.ignoreNextStartValue {
		d.ignoreNextStartValue = false
		return d.Enc.Err
	}
	// Don't block if enc.EncodeBytes was called, but the caller is decoding as a
	// sequence of entries.  See the matching logic in dec.FinishValue.
	if d.Enc.DecBytesAsEntries {
		if d.Enc.DecByteStartValue {
			return d.Enc.closeLocked(fmt.Errorf("vdl: can't StartValue on byte"))
		}
		d.Enc.DecByteStartValue = true
		return d.Enc.Err
	}
	if err := d.wait(false); err != nil {
		return err
	}
	top := d.top()
	if top == nil {
		return d.Enc.closeLocked(errEmptyPipeStack)
	}
	// Check compatibility between the actual type and the want type.  Since
	// compatibility applies to the entire static type, we only need to perform
	// this check for top-level decoded values, and subsequently for decoded any
	// values.
	if len(d.Enc.Stack) == 1 || d.IsAny() {
		if tt := d.Type(); !Compatible(tt, want) {
			return d.Enc.closeLocked(fmt.Errorf("vdl: pipe incompatible decode from %v into %v", tt, want))
		}
	}
	if got, want := top.NextOp, pipeStartDec; got != want {
		return d.Enc.closeLocked(fmt.Errorf("vdl: pipe got state %v, want %v", got, want))
	}
	top.NextOp = top.NextOp.Next()
	return d.Enc.Err
}

func (d *pipeDecoder) FinishValue() error {
	d.Lock()
	defer d.Unlock()
	switch {
	case d.Enc.State == pipeStateEncoder:
		return d.Enc.closeLocked(errInvalidPipeState)
	case d.Enc.State == pipeStateClosed:
		return d.Enc.Err
	}
	// Don't block if enc.EncodeBytes was called, but the caller is decoding as a
	// sequence of entries.  See the matching logic in dec.StartValue.
	if d.Enc.DecBytesAsEntries {
		if d.Enc.DecByteStartValue {
			d.Enc.DecByteStartValue = false
			return d.Enc.Err
		}
		//
		d.Enc.DecBytesAsEntries = false
	}
	if err := d.wait(true); err != nil {
		return err
	}
	top := d.top()
	if top == nil {
		return d.Enc.closeLocked(errEmptyPipeStack)
	}
	if got, want := top.NextOp, pipeFinishDec; got != want {
		return d.Enc.closeLocked(fmt.Errorf("vdl: pipe got state %v, want %v", got, want))
	}
	d.Enc.Stack = d.Enc.Stack[:len(d.Enc.Stack)-1]
	return d.Enc.Err
}

func (d *pipeDecoder) wait(isFinish bool) error {
	if d.Enc.State == pipeStateClosed {
		return d.Enc.Err
	}
	if isFinish || d.Enc.DecStarted {
		d.Enc.State = pipeStateEncoder
		d.Enc.Cond.Broadcast()
	}
	d.Enc.DecStarted = true
	for d.Enc.State == pipeStateEncoder {
		d.Enc.Cond.Wait()
	}
	if d.Enc.State == pipeStateClosed {
		return d.Enc.Err
	}
	return nil
}

func (d *pipeDecoder) SkipValue() error {
	if err := d.StartValue(AnyType); err != nil {
		return err
	}
	return d.FinishValue()
}

func (d *pipeDecoder) IgnoreNextStartValue() {
	d.ignoreNextStartValue = true
}

func (e *pipeEncoder) SetLenHint(lenHint int) error {
	top := e.top()
	if top == nil {
		return e.Close(errEmptyPipeStack)
	}
	top.LenHint = lenHint
	return e.Err
}

func (e *pipeEncoder) NextEntry(done bool) error {
	if e.State == pipeStateDecoder {
		return e.Close(errEncCallDuringDecPhase)
	}
	e.NextEntryDone = done
	return e.Err
}

func (d *pipeDecoder) NextEntry() (bool, error) {
	top := d.top()
	if top == nil {
		return false, d.Close(errEmptyPipeStack)
	}
	top.Index++
	var done bool
	if d.Enc.EncIsBytes {
		d.Enc.DecBytesAsEntries = true
		done = top.Index >= len(d.Enc.ArgBytes)
	} else {
		done = d.Enc.NextEntryDone
	}
	d.Enc.NextEntryDone = false
	return done, d.Enc.Err
}

func (e *pipeEncoder) NextField(index int) error {
	if e.State == pipeStateDecoder {
		return e.Close(errEncCallDuringDecPhase)
	}
	e.NextFieldIndex = index
	return e.Err
}

func (d *pipeDecoder) NextField() (int, error) {
	top := d.top()
	if top == nil {
		return -1, d.Close(errEmptyPipeStack)
	}
	top.Index = d.Enc.NextFieldIndex
	d.Enc.NextFieldIndex = -1
	return top.Index, d.Enc.Err
}

func (d *pipeDecoder) Type() *Type {
	top := d.top()
	if top == nil {
		return nil
	}
	if d.Enc.DecBytesAsEntries {
		return top.Type.Elem()
	}
	return top.Type
}

func (d *pipeDecoder) IsAny() bool {
	if stackTop2 := len(d.Enc.Stack) - 2; stackTop2 >= 0 {
		return d.Enc.Stack[stackTop2].nextValueIsAny()
	}
	return false
}

func (d *pipeDecoder) IsOptional() bool {
	if top := d.top(); top != nil {
		return top.IsOptional
	}
	return false
}

func (d *pipeDecoder) IsNil() bool {
	if top := d.top(); top != nil {
		return top.IsNil
	}
	return false
}

func (d *pipeDecoder) Index() int {
	if top := d.top(); top != nil {
		return top.Index
	}
	return -1
}

func (d *pipeDecoder) LenHint() int {
	if top := d.top(); top != nil {
		return top.LenHint
	}
	return -1
}

func (e *pipeEncoder) EncodeBool(v bool) error {
	if e.State == pipeStateDecoder {
		return e.Close(errEncCallDuringDecPhase)
	}
	e.ArgBool = v
	return e.Err
}

func (d *pipeDecoder) DecodeBool() (bool, error) {
	return d.Enc.ArgBool, d.Enc.Err
}

func (e *pipeEncoder) EncodeString(v string) error {
	if e.State == pipeStateDecoder {
		return e.Close(errEncCallDuringDecPhase)
	}
	e.ArgString = v
	return e.Err
}

func (d *pipeDecoder) DecodeString() (string, error) {
	return d.Enc.ArgString, d.Enc.Err
}

func (e *pipeEncoder) EncodeTypeObject(v *Type) error {
	if e.State == pipeStateDecoder {
		return e.Close(errEncCallDuringDecPhase)
	}
	e.ArgType = v
	return e.Err
}

func (d *pipeDecoder) DecodeTypeObject() (*Type, error) {
	return d.Enc.ArgType, d.Enc.Err
}

func (e *pipeEncoder) EncodeUint(v uint64) error {
	if e.State == pipeStateDecoder {
		return e.Close(errEncCallDuringDecPhase)
	}
	e.ArgUint = v
	e.NumberType = numberUint
	return e.Err
}

func (d *pipeDecoder) DecodeUint(bitlen int) (uint64, error) {
	const errFmt = "vdl: conversion from %v into uint%d loses precision: %v"
	top, tt := d.top(), d.Type()
	if top == nil {
		return 0, d.Close(errEmptyPipeStack)
	}
	switch d.Enc.NumberType {
	case numberUint:
		x := d.Enc.ArgUint
		if d.Enc.DecBytesAsEntries {
			x = uint64(d.Enc.ArgBytes[top.Index])
		}
		if shift := 64 - uint(bitlen); x != (x<<shift)>>shift {
			return 0, d.Close(fmt.Errorf(errFmt, tt, bitlen, x))
		}
		return x, d.Enc.Err
	case numberInt:
		x := d.Enc.ArgInt
		ux := uint64(x)
		if shift := 64 - uint(bitlen); x < 0 || ux != (ux<<shift)>>shift {
			return 0, d.Close(fmt.Errorf(errFmt, tt, bitlen, x))
		}
		return ux, d.Enc.Err
	case numberFloat:
		x := d.Enc.ArgFloat
		ux := uint64(x)
		if shift := 64 - uint(bitlen); x != float64(ux) || ux != (ux<<shift)>>shift {
			return 0, d.Close(fmt.Errorf(errFmt, tt, bitlen, x))
		}
		return ux, d.Enc.Err
	}
	return 0, d.Close(fmt.Errorf("vdl: incompatible decode from %v into uint%d", tt, bitlen))
}

func (e *pipeEncoder) EncodeInt(v int64) error {
	if e.State == pipeStateDecoder {
		return e.Close(errEncCallDuringDecPhase)
	}
	e.ArgInt = v
	e.NumberType = numberInt
	return e.Err
}

func (d *pipeDecoder) DecodeInt(bitlen int) (int64, error) {
	const errFmt = "vdl: conversion from %v into int%d loses precision: %v"
	top, tt := d.top(), d.Type()
	if top == nil {
		return 0, d.Close(errEmptyPipeStack)
	}
	switch d.Enc.NumberType {
	case numberUint:
		x := d.Enc.ArgUint
		if d.Enc.DecBytesAsEntries {
			x = uint64(d.Enc.ArgBytes[top.Index])
		}
		ix := int64(x)
		// The shift uses 65 since the topmost bit is the sign bit.  I.e. 32 bit
		// numbers should be shifted by 33 rather than 32.
		if shift := 65 - uint(bitlen); ix < 0 || x != (x<<shift)>>shift {
			return 0, d.Close(fmt.Errorf(errFmt, tt, bitlen, x))
		}
		return ix, d.Enc.Err
	case numberInt:
		x := d.Enc.ArgInt
		if shift := 64 - uint(bitlen); x != (x<<shift)>>shift {
			return 0, d.Close(fmt.Errorf(errFmt, tt, bitlen, x))
		}
		return x, d.Enc.Err
	case numberFloat:
		x := d.Enc.ArgFloat
		ix := int64(x)
		if shift := 64 - uint(bitlen); x != float64(ix) || ix != (ix<<shift)>>shift {
			return 0, d.Close(fmt.Errorf(errFmt, tt, bitlen, x))
		}
		return ix, d.Enc.Err
	}
	return 0, d.Close(fmt.Errorf("vdl: incompatible decode from %v into int%d", tt, bitlen))
}

func (e *pipeEncoder) EncodeFloat(v float64) error {
	if e.State == pipeStateDecoder {
		return e.Close(errEncCallDuringDecPhase)
	}
	e.ArgFloat = v
	e.NumberType = numberFloat
	return e.Err
}

func (d *pipeDecoder) DecodeFloat(bitlen int) (float64, error) {
	const errFmt = "vdl: conversion from %v into float%d loses precision: %v"
	top, tt := d.top(), d.Type()
	if top == nil {
		return 0, d.Close(errEmptyPipeStack)
	}
	switch d.Enc.NumberType {
	case numberUint:
		x := d.Enc.ArgUint
		if d.Enc.DecBytesAsEntries {
			x = uint64(d.Enc.ArgBytes[top.Index])
		}
		var max uint64
		if bitlen > 32 {
			max = float64MaxInt
		} else {
			max = float32MaxInt
		}
		if x > max {
			return 0, d.Close(fmt.Errorf(errFmt, tt, bitlen, x))
		}
		return float64(x), d.Enc.Err
	case numberInt:
		x := d.Enc.ArgInt
		var min, max int64
		if bitlen > 32 {
			min, max = float64MinInt, float64MaxInt
		} else {
			min, max = float32MinInt, float32MaxInt
		}
		if x < min || x > max {
			return 0, d.Close(fmt.Errorf(errFmt, tt, bitlen, x))
		}
		return float64(x), d.Enc.Err
	case numberFloat:
		x := d.Enc.ArgFloat
		if bitlen <= 32 && (x < -math.MaxFloat32 || x > math.MaxFloat32) {
			return 0, d.Close(fmt.Errorf(errFmt, tt, bitlen, x))
		}
		return x, d.Enc.Err
	}
	return 0, d.Close(fmt.Errorf("vdl: incompatible decode from %v into float%d", tt, bitlen))
}

func (e *pipeEncoder) EncodeBytes(v []byte) error {
	if e.State == pipeStateDecoder {
		return e.Close(errEncCallDuringDecPhase)
	}
	e.EncIsBytes = true
	e.ArgBytes = v
	e.NumberType = numberUint
	return e.Err
}

func (d *pipeDecoder) DecodeBytes(fixedLen int, value *[]byte) error {
	top := d.top()
	if top == nil {
		return d.Close(errEmptyPipeStack)
	}
	if !d.Enc.EncIsBytes {
		if err := DecodeConvertedBytes(d, fixedLen, value); err != nil {
			return d.Close(err)
		}
		return nil
	}
	len := len(d.Enc.ArgBytes)
	switch {
	case fixedLen >= 0 && fixedLen != len:
		return d.Close(fmt.Errorf("vdl: got %d bytes, want fixed len %d, %v", len, fixedLen, d.Type()))
	case len == 0:
		*value = nil
		return nil
	case fixedLen >= 0:
		// Only re-use the existing buffer if we're filling in an array.  This
		// sacrifices some performance, but also avoids bugs when repeatedly
		// decoding into the same value.
		*value = (*value)[:len]
	default:
		*value = make([]byte, len)
	}
	copy(*value, d.Enc.ArgBytes)
	return d.Enc.Err
}
