// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

import "fmt"

// Reader is the interface that wraps the VDLRead method.
//
// VDLRead fills in the the receiver that implements this method from the
// Decoder.  This method is auto-generated for all types defined in vdl.  It may
// be implemented for regular Go types not defined in vdl, to customize the
// decoding.
type Reader interface {
	VDLRead(dec Decoder) error
}

// Writer is the interface that wraps the VDLWrite method.
//
// VDLWrite writes out the receiver that implements this method to the Encoder.
// This method is auto-generated for all types defined in vdl.  It may be
// implemented for regular Go types not defined in vdl, to customize the
// encoding.
type Writer interface {
	VDLWrite(enc Encoder) error
}

// ReadWriter is the interface that groups the VDLRead and VDLWrite methods.
type ReadWriter interface {
	Reader
	Writer
}

// Decoder defines the interface for a decoder of vdl values.  The Decoder is
// passed as the argument to VDLRead.  An example of an implementation of this
// interface is vom.Decoder.
//
// The Decoder provides an API to read vdl values of all types in depth-first
// order.  The ordering is based on the type of the value being read.
// E.g. given the following value:
//    type MyStruct struct {
//      A []string
//      B map[int64]bool
//      C any
//    }
//    value := MyStruct{
//      A: {"abc", "def"},
//      B: {123: true, 456: false},
//      C: float32(1.5),
//    }
// The values will be read in the following order:
//    "abc"
//    "def"
//    (123, true)
//    (456, false)
//    1.5
type Decoder interface {
	// StartValue must be called before decoding each value, for both scalar and
	// composite values.  The want type is the type of value being decoded into,
	// used to check compatibility with the value in the decoder; use AnyType if
	// you don't know, or want to decode any type of value.  Each call pushes the
	// type of the next value on to the stack.
	StartValue(want *Type) error
	// FinishValue must be called after decoding each value, for both scalar and
	// composite values.  Each call pops the type of the top value off of the
	// stack.
	FinishValue() error
	// SkipValue skips the next value; logically it behaves as if a full sequence
	// of StartValue / ...Decode*... / FinishValue were called.  It enables
	// optimizations when the caller doesn't care about the next value.
	SkipValue() error
	// IgnoreNextStartValue instructs the Decoder to ignore the next call to
	// StartValue.  It is used to simplify implementations of VDLRead; e.g. a
	// caller might call StartValue to check for nil values, and subsequently call
	// StartValue again to read non-nil values.  IgnoreNextStartValue is used to
	// ignore the second StartValue call.
	IgnoreNextStartValue()

	// NextEntry instructs the Decoder to move to the next element of an Array or
	// List, the next key of a Set, or the next (key,elem) pair of a Map.  Returns
	// done=true when there are no remaining entries.
	NextEntry() (done bool, _ error)
	// NextField instructs the Decoder to move to the next field of a Struct or
	// Union.  Returns the index of the next field, or -1 when there are no
	// remaining fields.  You may call Decoder.Type().Field(index).Name to
	// retrieve the name of the struct or union field.
	NextField() (index int, _ error)

	// Type returns the type of the top value on the stack.  Returns nil when the
	// stack is empty.  The returned type is only Any or Optional iff the value is
	// nil; non-nil values are "auto-dereferenced" to their underlying elem value.
	Type() *Type
	// IsAny returns true iff the type of the top value on the stack was Any,
	// despite the "auto-dereference" behavior of non-nil values.
	IsAny() bool
	// IsOptional returns true iff the type of the top value on the stack was
	// Optional, despite the "auto-dereference" behavior of non-nil values.
	IsOptional() bool
	// IsNil returns true iff the top value on the stack is nil.  It is equivalent
	// to Type() == AnyType || Type().Kind() == Optional.
	IsNil() bool
	// Index returns the index of the current entry or field of the top value on
	// the stack.  Returns -1 if the top value is a scalar, or if NextEntry /
	// NextField has not been called.
	Index() int
	// LenHint returns the length of the top value on the stack, if it is
	// available.  Returns -1 if the top value is a scalar, or if the length is
	// not available.
	LenHint() int

	// DecodeBool returns the top value on the stack as a bool.
	DecodeBool() (bool, error)
	// DecodeString returns the top value on the stack as a string.
	DecodeString() (string, error)
	// DecodeUint returns the top value on the stack as a uint, where the result
	// has bitlen bits.  Errors are returned on loss of precision.
	DecodeUint(bitlen int) (uint64, error)
	// DecodeInt returns the top value on the stack as an int, where the result
	// has bitlen bits.  Errors are returned on loss of precision.
	DecodeInt(bitlen int) (int64, error)
	// DecodeFloat returns the top value on the stack as a float, where the result
	// has bitlen bits.  Errors are returned on loss of precision.
	DecodeFloat(bitlen int) (float64, error)
	// DecodeTypeObject returns the top value on the stack as a type.
	DecodeTypeObject() (*Type, error)
	// DecodeBytes decodes the top value on the stack as bytes, into x.  If
	// fixedLen >= 0 the decoded bytes must be exactly that length, otherwise
	// there is no restriction on the number of decoded bytes.  If cap(*x) is not
	// large enough to fit the decoded bytes, a new byte slice is assigned to *x.
	DecodeBytes(fixedLen int, x *[]byte) error

	// ReadValueBool behaves as if StartValue, DecodeBool, FinishValue were
	// called in sequence.  Some decoders optimize this codepath.
	ReadValueBool() (bool, error)
	// ReadValueString behaves as if StartValue, DecodeString, FinishValue were
	// called in sequence.  Some decoders optimize this codepath.
	ReadValueString() (string, error)
	// ReadValueUint behaves as if StartValue, DecodeUint, FinishValue were called
	// in sequence.  Some decoders optimize this codepath.
	ReadValueUint(bitlen int) (uint64, error)
	// ReadValueInt behaves as if StartValue, DecodeInt, FinishValue were called
	// in sequence.  Some decoders optimize this codepath.
	ReadValueInt(bitlen int) (int64, error)
	// ReadValueFloat behaves as if StartValue, DecodeFloat, FinishValue were
	// called in sequence.  Some decoders optimize this codepath.
	ReadValueFloat(bitlen int) (float64, error)
	// ReadValueTypeObject behaves as if StartValue, DecodeTypeObject, FinishValue
	// were called in sequence.  Some decoders optimize this codepath.
	ReadValueTypeObject() (*Type, error)
	// ReadValueBytes behaves as if StartValue, DecodeBytes, FinishValue were
	// called in sequence.  Some decoders optimize this codepath.
	ReadValueBytes(fixedLen int, x *[]byte) error

	// NextEntryValueBool behaves as if NextEntry, StartValue, DecodeBool,
	// FinishValue were called in sequence.  Some decoders optimize this codepath.
	NextEntryValueBool() (done bool, _ bool, _ error)
	// NextEntryValueString behaves as if NextEntry, StartValue, DecodeString,
	// FinishValue were called in sequence.  Some decoders optimize this codepath.
	NextEntryValueString() (done bool, _ string, _ error)
	// NextEntryValueUint behaves as if NextEntry, StartValue, DecodeUint,
	// FinishValue were called in sequence.  Some decoders optimize this codepath.
	NextEntryValueUint(bitlen int) (done bool, _ uint64, _ error)
	// NextEntryValueInt behaves as if NextEntry, StartValue, DecodeInt,
	// FinishValue were called in sequence.  Some decoders optimize this codepath.
	NextEntryValueInt(bitlen int) (done bool, _ int64, _ error)
	// NextEntryValueFloat behaves as if NextEntry, StartValue, DecodeFloat,
	// FinishValue were called in sequence.  Some decoders optimize this codepath.
	NextEntryValueFloat(bitlen int) (done bool, _ float64, _ error)
	// NextEntryValueTypeObject behaves as if NextEntry, StartValue,
	// DecodeTypeObject, FinishValue were called in sequence.  Some decoders
	// optimize this codepath.
	NextEntryValueTypeObject() (done bool, _ *Type, _ error)
}

// Encoder defines the interface for an encoder of vdl values.  The Encoder is
// passed as the argument to VDLWrite.  An example of an implementation of this
// interface is vom.Encoder.
//
// The Encoder provides an API to write vdl values of all types in depth-first
// order.  The ordering is based on the type of the value being written; see
// Decoder for examples.
type Encoder interface {
	// StartValue must be called before encoding each non-nil value, for both
	// scalar and composite values.  The tt type cannot be Any or Optional; use
	// NilValue to encode nil values.
	StartValue(tt *Type) error
	// FinishValue must be called after encoding each non-nil value, for both
	// scalar and composite values.
	FinishValue() error
	// NilValue encodes a nil value.  The tt type must be Any or Optional.
	NilValue(tt *Type) error
	// SetNextStartValueIsOptional instructs the encoder that the next call to
	// StartValue represents a value with an Optional type.
	SetNextStartValueIsOptional()

	// NextEntry instructs the Encoder to move to the next element of an Array or
	// List, the next key of a Set, or the next (key,elem) pair of a Map.  Set
	// done=true when there are no remaining entries.
	NextEntry(done bool) error
	// NextField instructs the Encoder to move to the next field of a Struct or
	// Union.  Set index to the index of the next field, or -1 when there are no
	// remaining fields.
	NextField(index int) error

	// SetLenHint sets the length of the List, Set or Map value.  It may only be
	// called immediately after StartValue, before NextEntry has been called.  Do
	// not call this method if the length is not known.
	SetLenHint(lenHint int) error

	// EncodeBool encodes a bool value.
	EncodeBool(value bool) error
	// EncodeString encodes a string value.
	EncodeString(value string) error
	// EncodeUint encodes a uint value.
	EncodeUint(value uint64) error
	// EncodeInt encodes an int value.
	EncodeInt(value int64) error
	// EncodeFloat encodes a float value.
	EncodeFloat(value float64) error
	// EncodeTypeObject encodes a type.
	EncodeTypeObject(value *Type) error
	// EncodeBytes encodes a bytes value; either an array or list of bytes.
	EncodeBytes(value []byte) error

	// WriteValueBool behaves as if StartValue, EncodeBool, FinishValue were
	// called in sequence.  Some encoders optimize this codepath.
	WriteValueBool(tt *Type, value bool) error
	// WriteValueString behaves as if StartValue, EncodeString, FinishValue were
	// called in sequence.  Some encoders optimize this codepath.
	WriteValueString(tt *Type, value string) error
	// WriteValueUint behaves as if StartValue, EncodeUint, FinishValue were
	// called in sequence.  Some encoders optimize this codepath.
	WriteValueUint(tt *Type, value uint64) error
	// WriteValueInt behaves as if StartValue, EncodeInt, FinishValue were called
	// in sequence.  Some encoders optimize this codepath.
	WriteValueInt(tt *Type, value int64) error
	// WriteValueFloat behaves as if StartValue, EncodeFloat, FinishValue were
	// called in sequence.  Some encoders optimize this codepath.
	WriteValueFloat(tt *Type, value float64) error
	// WriteValueTypeObject behaves as if StartValue, EncodeTypeObject,
	// FinishValue were called in sequence.  Some encoders optimize this codepath.
	WriteValueTypeObject(value *Type) error
	// WriteValueBytes behaves as if StartValue, EncodeBytes, FinishValue were
	// called in sequence.  Some encoders optimize this codepath.
	WriteValueBytes(tt *Type, value []byte) error

	// NextEntryValueBool behaves as if NextEntry, StartValue, EncodeBool,
	// FinishValue were called in sequence.  Some encoders optimize this codepath.
	NextEntryValueBool(tt *Type, value bool) error
	// NextEntryValueString behaves as if NextEntry, StartValue, EncodeString,
	// FinishValue were called in sequence.  Some encoders optimize this codepath.
	NextEntryValueString(tt *Type, value string) error
	// NextEntryValueUint behaves as if NextEntry, StartValue, EncodeUint,
	// FinishValue were called in sequence.  Some encoders optimize this codepath.
	NextEntryValueUint(tt *Type, value uint64) error
	// NextEntryValueInt behaves as if NextEntry, StartValue, EncodeInt,
	// FinishValue were called in sequence.  Some encoders optimize this codepath.
	NextEntryValueInt(tt *Type, value int64) error
	// NextEntryValueFloat behaves as if NextEntry, StartValue, EncodeFloat,
	// FinishValue were called in sequence.  Some encoders optimize this codepath.
	NextEntryValueFloat(tt *Type, value float64) error
	// NextEntryValueTypeObject behaves as if NextEntry, StartValue,
	// EncodeTypeObject, FinishValue were called in sequence.  Some encoders
	// optimize this codepath.
	NextEntryValueTypeObject(value *Type) error
	// NextEntryValueBytes behaves as if NextEntry, StartValue, EncodeBytes,
	// FinishValue were called in sequence.  Some encoders optimize this codepath.
	NextEntryValueBytes(tt *Type, value []byte) error

	// NextFieldValueBool behaves as if NextEntry, StartValue, EncodeBool,
	// FinishValue were called in sequence.  Some encoders optimize this codepath.
	NextFieldValueBool(index int, tt *Type, value bool) error
	// NextFieldValueString behaves as if NextEntry, StartValue, EncodeString,
	// FinishValue were called in sequence.  Some encoders optimize this codepath.
	NextFieldValueString(index int, tt *Type, value string) error
	// NextFieldValueUint behaves as if NextEntry, StartValue, EncodeUint,
	// FinishValue were called in sequence.  Some encoders optimize this codepath.
	NextFieldValueUint(index int, tt *Type, value uint64) error
	// NextFieldValueInt behaves as if NextEntry, StartValue, EncodeInt,
	// FinishValue were called in sequence.  Some encoders optimize this codepath.
	NextFieldValueInt(index int, tt *Type, value int64) error
	// NextFieldValueFloat behaves as if NextEntry, StartValue, EncodeFloat,
	// FinishValue were called in sequence.  Some encoders optimize this codepath.
	NextFieldValueFloat(index int, tt *Type, value float64) error
	// NextFieldValueTypeObject behaves as if NextEntry, StartValue,
	// EncodeTypeObject, FinishValue were called in sequence.  Some encoders
	// optimize this codepath.
	NextFieldValueTypeObject(index int, value *Type) error
	// NextFieldValueBytes behaves as if NextEntry, StartValue, EncodeBytes,
	// FinishValue were called in sequence.  Some encoders optimize this codepath.
	NextFieldValueBytes(index int, tt *Type, value []byte) error
}

// DecodeConvertedBytes is a helper function for implementations of
// Decoder.DecodeBytes, to deal with cases where the decoder value is
// convertible to []byte.  E.g. if the decoder value is []float64, we need to
// decode each element as a uint8, performing conversion checks.
//
// Since this is meant to be used in the implementation of DecodeBytes, there is
// no outer call to StartValue/FinishValue.
func DecodeConvertedBytes(dec Decoder, fixedLen int, buf *[]byte) error {
	// Only re-use the existing buffer if we're filling in an array.  This
	// sacrifices some performance, but also avoids bugs when repeatedly decoding
	// into the same value.
	switch len := dec.LenHint(); {
	case fixedLen >= 0:
		*buf = (*buf)[:0]
	case len > 0:
		*buf = make([]byte, 0, len)
	default:
		*buf = nil
	}
	index := 0
	for {
		switch done, elem, err := dec.NextEntryValueUint(8); {
		case err != nil:
			return err
		case fixedLen >= 0 && done != (index >= fixedLen):
			return fmt.Errorf("array len mismatch, done:%v index:%d len:%d", done, index, fixedLen)
		case done:
			return nil
		default:
			*buf = append(*buf, byte(elem))
		}
		index++
	}
}
