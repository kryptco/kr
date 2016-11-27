// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vom

import (
	"strconv"

	"v.io/v23/vdl"
)

// This file contains the ReadValue* methods.  The semantics of these methods is
// the same as if StartValue, Decode*, FinishValue were called in sequence.  The
// implementation is faster than actually calling that sequence, because we can
// avoid pushing and popping the decoder stack and also avoid unnecessary
// compatibility checks.  We also get a minor improvement by avoiding extra
// method calls indirected through the Decoder interface.
//
// Each method has the same pattern:
//
// Check fastpath:
//   If we've already determined from the parent type that we can use the
//   fastpath, we simply decode the value, skipping both the StartValue logic as
//   well as the conversion logic.
// StartValue:
//   If IgnoreNextStartValue is set, the type is already on the stack.
//   Otherwise setup the type to process Any and Optional headers.  We pass nil
//   to d.setupType to avoid the compatibility check, since the decode step will
//   naturally let us perform that check.
// Decode:
//   We implement common-case fastpaths; e.g. avoiding unnecessary conversions.
// FinishValue:
//   Mirrors StartValue, only pop the stack if necessary.

// isFastReadParent returns true iff subtypes of tt can use the fastpath for the
// ReadValue* methods.  By using the fastpath we can skip the expensive
// dfsNextType and setupType calls.  We can't use the fastpath for:
//   Any:  since we always need to process the any header
//   Enum: since ReadValueString won't know whether to decode a string or enum
//   Byte: since ReadValueUint won't know whether to decode a uint or full byte
//
// REQUIRES: tt is identical to the want type that the user is decoding into,
// which ensures that we don't need to perform conversions.
func isFastReadParent(tt *vdl.Type) bool {
	switch tt.Kind() {
	case vdl.Array, vdl.List:
		elem := tt.Elem().Kind()
		return elem != vdl.Any && elem != vdl.Enum && elem != vdl.Byte
	case vdl.Set:
		key := tt.Key().Kind()
		return key != vdl.Any && key != vdl.Enum && key != vdl.Byte
	case vdl.Map:
		key := tt.Key().Kind()
		elem := tt.Elem().Kind()
		return key != vdl.Any && key != vdl.Enum && key != vdl.Byte &&
			elem != vdl.Any && elem != vdl.Enum && elem != vdl.Byte
	case vdl.Struct, vdl.Union:
		if !tt.ContainsKind(vdl.WalkAll, kkAnyEnumByte...) {
			return true
		}
		for f := 0; f < tt.NumField(); f++ {
			if k := tt.Field(f).Type.Kind(); k == vdl.Any || k == vdl.Enum || k == vdl.Byte {
				return false
			}
		}
		return true
	default:
		return false
	}
}

var kkAnyEnumByte = []vdl.Kind{vdl.Any, vdl.Enum, vdl.Byte}

func (d *decoder81) ReadValueBool() (value bool, err error) {
	top, isOnStack := d.top(), d.flag.IgnoreNextStartValue()
	// Check fastpath
	if top != nil && top.Flag.FastRead() {
		value, err = binaryDecodeBool(d.buf)
	} else {
		// StartValue
		var tt *vdl.Type
		if isOnStack {
			tt = top.Type
		} else {
			if tt, err = d.dfsNextType(); err != nil {
				return false, err
			}
			if tt, _, _, err = d.setupType(tt, nil); err != nil {
				return false, err
			}
		}
		// Decode
		switch tt.Kind() {
		case vdl.Bool:
			value, err = binaryDecodeBool(d.buf)
		default:
			return false, errIncompatibleDecode(tt, "bool")
		}
	}
	// FinishValue
	if isOnStack {
		if err := d.FinishValue(); err != nil {
			return false, err
		}
	} else {
		d.flag = d.flag.Clear(decFlagFinishValue)
		if top == nil {
			if err := d.endMessage(); err != nil {
				return false, err
			}
		}
	}
	return value, err
}

func (d *decoder81) ReadValueString() (value string, err error) {
	top, isOnStack := d.top(), d.flag.IgnoreNextStartValue()
	// Check fastpath
	if top != nil && top.Flag.FastRead() {
		value, err = binaryDecodeString(d.buf)
	} else {
		// StartValue
		var tt *vdl.Type
		if isOnStack {
			tt = top.Type
		} else {
			if tt, err = d.dfsNextType(); err != nil {
				return "", err
			}
			if tt, _, _, err = d.setupType(tt, nil); err != nil {
				return "", err
			}
		}
		// Decode
		switch tt.Kind() {
		case vdl.String:
			value, err = binaryDecodeString(d.buf)
		case vdl.Enum:
			value, err = d.binaryDecodeEnum(tt)
		default:
			return "", errIncompatibleDecode(tt, "string")
		}
	}
	// FinishValue
	if isOnStack {
		if err := d.FinishValue(); err != nil {
			return "", err
		}
	} else {
		d.flag = d.flag.Clear(decFlagFinishValue)
		if top == nil {
			if err := d.endMessage(); err != nil {
				return "", err
			}
		}
	}
	return value, err
}

func (d *decoder81) ReadValueUint(bitlen int) (value uint64, err error) {
	top, isOnStack := d.top(), d.flag.IgnoreNextStartValue()
	// Check fastpath
	if top != nil && top.Flag.FastRead() {
		value, err = binaryDecodeUint(d.buf)
	} else {
		// StartValue
		var tt *vdl.Type
		if isOnStack {
			tt = top.Type
		} else {
			if tt, err = d.dfsNextType(); err != nil {
				return 0, err
			}
			if tt, _, _, err = d.setupType(tt, nil); err != nil {
				return 0, err
			}
		}
		// Decode, avoiding unnecessary number conversions.
		switch kind := tt.Kind(); kind {
		case vdl.Uint16, vdl.Uint32, vdl.Uint64:
			if kind.BitLen() <= bitlen {
				value, err = binaryDecodeUint(d.buf)
			} else {
				value, err = d.decodeUint(tt, uint(bitlen))
			}
		case vdl.Int8, vdl.Int16, vdl.Int32, vdl.Int64, vdl.Float32, vdl.Float64:
			value, err = d.decodeUint(tt, uint(bitlen))
		case vdl.Byte:
			var b byte
			b, err = d.binaryDecodeByte()
			value = uint64(b)
		default:
			return 0, errIncompatibleDecode(tt, "uint"+strconv.Itoa(bitlen))
		}
	}
	// FinishValue
	if isOnStack {
		if err := d.FinishValue(); err != nil {
			return 0, err
		}
	} else {
		d.flag = d.flag.Clear(decFlagFinishValue)
		if top == nil {
			if err := d.endMessage(); err != nil {
				return 0, err
			}
		}
	}
	return value, err
}

func (d *decoder81) ReadValueInt(bitlen int) (value int64, err error) {
	top, isOnStack := d.top(), d.flag.IgnoreNextStartValue()
	// Check fastpath
	if top != nil && top.Flag.FastRead() {
		value, err = binaryDecodeInt(d.buf)
	} else {
		// StartValue
		var tt *vdl.Type
		if isOnStack {
			tt = top.Type
		} else {
			if tt, err = d.dfsNextType(); err != nil {
				return 0, err
			}
			if tt, _, _, err = d.setupType(tt, nil); err != nil {
				return 0, err
			}
		}
		// Decode, avoiding unnecessary number conversions.
		switch kind := tt.Kind(); kind {
		case vdl.Int8, vdl.Int16, vdl.Int32, vdl.Int64:
			if kind.BitLen() <= bitlen {
				value, err = binaryDecodeInt(d.buf)
			} else {
				value, err = d.decodeInt(tt, uint(bitlen))
			}
		case vdl.Byte, vdl.Uint16, vdl.Uint32, vdl.Uint64, vdl.Float32, vdl.Float64:
			value, err = d.decodeInt(tt, uint(bitlen))
		default:
			return 0, errIncompatibleDecode(tt, "int"+strconv.Itoa(bitlen))
		}
	}
	// FinishValue
	if isOnStack {
		if err := d.FinishValue(); err != nil {
			return 0, err
		}
	} else {
		d.flag = d.flag.Clear(decFlagFinishValue)
		if top == nil {
			if err := d.endMessage(); err != nil {
				return 0, err
			}
		}
	}
	return value, err
}

func (d *decoder81) ReadValueFloat(bitlen int) (value float64, err error) {
	top, isOnStack := d.top(), d.flag.IgnoreNextStartValue()
	// Check fastpath
	if top != nil && top.Flag.FastRead() {
		value, err = binaryDecodeFloat(d.buf)
	} else {
		// StartValue
		var tt *vdl.Type
		if isOnStack {
			tt = top.Type
		} else {
			if tt, err = d.dfsNextType(); err != nil {
				return 0, err
			}
			if tt, _, _, err = d.setupType(tt, nil); err != nil {
				return 0, err
			}
		}
		// Decode, avoiding unnecessary number conversions.
		switch kind := tt.Kind(); kind {
		case vdl.Float32, vdl.Float64:
			if kind.BitLen() <= bitlen {
				value, err = binaryDecodeFloat(d.buf)
			} else {
				value, err = d.decodeFloat(tt, uint(bitlen))
			}
		case vdl.Byte, vdl.Uint16, vdl.Uint32, vdl.Uint64, vdl.Int8, vdl.Int16, vdl.Int32, vdl.Int64:
			value, err = d.decodeFloat(tt, uint(bitlen))
		default:
			return 0, errIncompatibleDecode(tt, "float"+strconv.Itoa(bitlen))
		}
	}
	// FinishValue
	if isOnStack {
		if err := d.FinishValue(); err != nil {
			return 0, err
		}
	} else {
		d.flag = d.flag.Clear(decFlagFinishValue)
		if top == nil {
			if err := d.endMessage(); err != nil {
				return 0, err
			}
		}
	}
	return value, err
}

func (d *decoder81) ReadValueTypeObject() (value *vdl.Type, err error) {
	top, isOnStack := d.top(), d.flag.IgnoreNextStartValue()
	// Check fastpath
	if top != nil && top.Flag.FastRead() {
		value, err = d.binaryDecodeType()
	} else {
		// StartValue
		var tt *vdl.Type
		if isOnStack {
			tt = top.Type
		} else {
			if tt, err = d.dfsNextType(); err != nil {
				return nil, err
			}
			if tt, _, _, err = d.setupType(tt, nil); err != nil {
				return nil, err
			}
		}
		// Decode
		switch tt.Kind() {
		case vdl.TypeObject:
			value, err = d.binaryDecodeType()
		default:
			return nil, errIncompatibleDecode(tt, "typeobject")
		}
	}
	// FinishValue
	if isOnStack {
		if err := d.FinishValue(); err != nil {
			return nil, err
		}
	} else {
		d.flag = d.flag.Clear(decFlagFinishValue)
		if top == nil {
			if err := d.endMessage(); err != nil {
				return nil, err
			}
		}
	}
	return value, err
}

// ReadValueBytes is more complicated than the other ReadValue* methods, since
// []byte lists and [n]byte arrays aren't scalar, and may need more complicated
// conversions
//
// TODO(toddw): Implement fastpath for this?
func (d *decoder81) ReadValueBytes(fixedLen int, x *[]byte) (err error) {
	// StartValue.  Initialize tt and lenHint, and track whether the []byte type
	// is already on the stack via isOnStack.
	isOnStack := d.flag.IgnoreNextStartValue()
	d.flag = d.flag.Clear(decFlagIgnoreNextStartValue)
	var tt *vdl.Type
	var lenHint int
	if isOnStack {
		top := d.top()
		tt, lenHint = top.Type, top.LenHint
	} else {
		if tt, err = d.dfsNextType(); err != nil {
			return err
		}
		var flag decStackFlag
		if tt, lenHint, flag, err = d.setupType(tt, nil); err != nil {
			return err
		}
		// If tt isn't []byte or [n]byte (or a named variant of these), we need to
		// perform conversion byte-by-byte.  This is complicated, and can't be
		// really fast, so we just push an entry onto the stack and handle this via
		// DecodeConvertedBytes below.
		//
		// We also need to perform the compatibility check, to make sure tt is
		// compatible with []byte.  The check is fairly expensive, so skipping it
		// when tt is actually a bytes type makes the the common case faster.
		if !tt.IsBytes() {
			if !vdl.Compatible(tt, ttByteList) {
				return errIncompatibleDecode(tt, "bytes")
			}
			d.stack = append(d.stack, decStackEntry{
				Type:    tt,
				Index:   -1,
				LenHint: lenHint,
				Flag:    flag,
			})
			isOnStack = true
		}
	}
	// Decode.  The common-case fastpath reads directly from the buffer.
	if tt.IsBytes() {
		if err := d.decodeBytes(tt, lenHint, fixedLen, x); err != nil {
			return err
		}
	} else {
		if err := vdl.DecodeConvertedBytes(d, fixedLen, x); err != nil {
			return err
		}
	}
	// FinishValue
	if isOnStack {
		if err := d.FinishValue(); err != nil {
			return err
		}
	} else {
		d.flag = d.flag.Clear(decFlagIsParentBytes)
		if len(d.stack) == 0 {
			if err := d.endMessage(); err != nil {
				return err
			}
		}
	}
	return nil
}

var ttByteList = vdl.ListType(vdl.ByteType)
