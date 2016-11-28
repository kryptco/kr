// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vom

import (
	"strconv"

	"v.io/v23/vdl"
)

// This file contains the NextEntryValue* methods.  The semantics of these
// methods is the same as if NextEntry, StartValue, Decode*, FinishValue were
// called in sequence.  The implementation is faster than actually calling that
// sequence, because we minimize the work done in the inner loop.
//
// Here's what the regular sequence of calls looks like, without error handling:
//   for {
//     done, _ := dec.NextEntry()    // Increments index
//     if done {
//       return nil
//     }
//     dec.StartValue()              // Pushes type onto stack
//     value, _ := dec.DecodeBool()  // Decodes the value
//     dec.FinishValue()             // Pops type off of stack
//   }
//
// Here's what the NextEntryValue call looks like:
//   for {
//     done, value, _ := dec.NextEntryValueBool()
//     if done {
//       return nil
//     }
//   }
//
// So far, we've reduced the 4 calls through the Decoder interface into 1 call,
// but if we just implemented exactly the same logic it wouldn't be much faster.
// The real speedup comes because in the common case, the type of the entry is
// static (i.e. not Any), so we can skip pushing and popping the stack, and also
// skip checking compatibility.  Instead we only perform this once, the first
// time through the loop, and each subsequent call has a fastpath.
//
// Each method as the same pattern:
//
// NextEntry:
//   Increments the index, and check if we're done.
// Check fastpath:
//   The NextEntryType field on the containing collection tells us whether the
//   fastpath has been enabled.  E.g. if []int32 is at the top of the stack, its
//   NextEntryType field will be Int32Type.  We initialize this on the first
//   call, by running the equivalent of StartValue and checking compatibility.
//   There's no need to push the stack, since we have the NextEntryType field to
//   remind us that all this work has been done.  We can't use the fastpath for
//   Any types, since we'll always need to decode the Any header and check
//   compatibility
// Decode:
//   We implement common-case fastpaths; e.g. avoiding unnecessary conversions.

type nextEntryData int

const (
	// We store extra information about the next type in the decoder flags, so we
	// don't need to recompute it for each entry.
	//
	// For number types, bits [8,15] store the bitlen, and we have two special
	// sentry flags.  The sentries must be > any valid bitlen.
	nextEntryMustConvert nextEntryData = 0x0100 // must convert value
	nextEntryParentBytes nextEntryData = 0x0200 // parent is []byte

	// TODO(toddw): VDL currently doesn't support Optional scalars, but if we were
	// to add this support, we'd need to set an extra bit in nextEntryData to
	// remind us to decode the Optional header.
)

func (d *decoder81) NextEntryValueBool() (done bool, value bool, err error) {
	// NextEntry
	top := d.top()
	if top == nil {
		return false, false, errEmptyDecoderStack
	}
	top.Index++
	if top.Index == top.LenHint {
		return true, false, nil
	}
	// Check fastpath
	if top.NextEntryType != nil {
		if top.Type.Kind() == vdl.Map {
			top.Flag = top.Flag.FlipIsMapKey()
		}
	} else {
		// StartValue
		var ttNext *vdl.Type
		if ttNext, err = d.dfsNextType(); err != nil {
			return false, false, err
		}
		var flag decStackFlag
		if ttNext, _, flag, err = d.setupType(ttNext, nil); err != nil {
			return false, false, err
		}
		if !flag.IsAny() { // can't enable fastpath for Any types.
			top.NextEntryType = ttNext
		}
		// Check compatibility
		switch ttNext.Kind() {
		case vdl.Bool:
		default:
			return false, false, errIncompatibleDecode(ttNext, "bool")
		}
	}
	// Decode
	value, err = binaryDecodeBool(d.buf)
	return false, value, err
}

func (d *decoder81) NextEntryValueString() (done bool, value string, err error) {
	// NextEntry
	top := d.top()
	if top == nil {
		return false, "", errEmptyDecoderStack
	}
	top.Index++
	if top.Index == top.LenHint {
		return true, "", nil
	}
	// Check fastpath
	var ttNext *vdl.Type
	if top.NextEntryType != nil {
		ttNext = top.NextEntryType
		if top.Type.Kind() == vdl.Map {
			top.Flag = top.Flag.FlipIsMapKey()
		}
	} else {
		// StartValue
		if ttNext, err = d.dfsNextType(); err != nil {
			return false, "", err
		}
		var flag decStackFlag
		if ttNext, _, flag, err = d.setupType(ttNext, nil); err != nil {
			return false, "", err
		}
		if !flag.IsAny() { // can't enable fastpath for Any types.
			top.NextEntryType = ttNext
		}
		// Check compatibility
		switch ttNext.Kind() {
		case vdl.String, vdl.Enum:
		default:
			return false, "", errIncompatibleDecode(ttNext, "string")
		}
	}
	// Decode
	switch ttNext.Kind() {
	case vdl.String:
		value, err = binaryDecodeString(d.buf)
	case vdl.Enum:
		value, err = d.binaryDecodeEnum(ttNext)
	}
	return false, value, err
}

func (d *decoder81) NextEntryValueUint(bitlen int) (done bool, value uint64, err error) {
	// NextEntry
	top := d.top()
	if top == nil {
		return false, 0, errEmptyDecoderStack
	}
	top.Index++
	if top.Index == top.LenHint {
		return true, 0, nil
	}
	// Check fastpath
	var ttNext *vdl.Type
	if top.NextEntryType != nil {
		ttNext = top.NextEntryType
		if top.Type.Kind() == vdl.Map {
			top.Flag = top.Flag.FlipIsMapKey()
		}
	} else {
		// StartValue
		if ttNext, err = d.dfsNextType(); err != nil {
			return false, 0, err
		}
		var flag decStackFlag
		if ttNext, _, flag, err = d.setupType(ttNext, nil); err != nil {
			return false, 0, err
		}
		if !flag.IsAny() { // can't enable fastpath for Any types.
			top.NextEntryType = ttNext
		}
		// Check compatibility, and set NextEntryData.
		switch ttNext.Kind() {
		case vdl.Uint16, vdl.Uint32, vdl.Uint64:
			top.NextEntryData = nextEntryData(ttNext.Kind().BitLen())
		case vdl.Int8, vdl.Int16, vdl.Int32, vdl.Int64, vdl.Float32, vdl.Float64:
			top.NextEntryData = nextEntryMustConvert
		case vdl.Byte:
			if d.flag.IsParentBytes() {
				top.NextEntryData = nextEntryParentBytes
			} else {
				top.NextEntryData = 8 // byte is 8 bits
			}
		default:
			return false, 0, errIncompatibleDecode(ttNext, "uint"+strconv.Itoa(bitlen))
		}
	}
	// Decode, avoiding unnecessary number conversions.
	switch flag := top.NextEntryData; {
	case flag <= nextEntryData(bitlen):
		value, err = binaryDecodeUint(d.buf)
	case flag == nextEntryParentBytes:
		if d.buf.IsAvailable(1) {
			value = uint64(d.buf.ReadAvailableByte())
		} else if err = d.buf.Fill(1); err == nil {
			value = uint64(d.buf.ReadAvailableByte())
		}
	default: // must convert
		value, err = d.decodeUint(ttNext, uint(bitlen))
	}
	return false, value, err
}

func (d *decoder81) NextEntryValueInt(bitlen int) (done bool, value int64, err error) {
	// NextEntry
	top := d.top()
	if top == nil {
		return false, 0, errEmptyDecoderStack
	}
	top.Index++
	if top.Index == top.LenHint {
		return true, 0, nil
	}
	// Check fastpath
	var ttNext *vdl.Type
	if top.NextEntryType != nil {
		ttNext = top.NextEntryType
		if top.Type.Kind() == vdl.Map {
			top.Flag = top.Flag.FlipIsMapKey()
		}
	} else {
		// StartValue
		if ttNext, err = d.dfsNextType(); err != nil {
			return false, 0, err
		}
		var flag decStackFlag
		if ttNext, _, flag, err = d.setupType(ttNext, nil); err != nil {
			return false, 0, err
		}
		if !flag.IsAny() { // can't enable fastpath for Any types.
			top.NextEntryType = ttNext
		}
		// Check compatibility, and set NextEntryData.
		switch ttNext.Kind() {
		case vdl.Int8, vdl.Int16, vdl.Int32, vdl.Int64:
			top.NextEntryData = nextEntryData(ttNext.Kind().BitLen())
		case vdl.Byte, vdl.Uint16, vdl.Uint32, vdl.Uint64, vdl.Float32, vdl.Float64:
			top.NextEntryData = nextEntryMustConvert
		default:
			return false, 0, errIncompatibleDecode(ttNext, "int"+strconv.Itoa(bitlen))
		}
	}
	// Decode, avoiding unnecessary number conversions.
	switch flag := top.NextEntryData; {
	case flag <= nextEntryData(bitlen):
		value, err = binaryDecodeInt(d.buf)
	default: // must convert
		value, err = d.decodeInt(ttNext, uint(bitlen))
	}
	return false, value, err
}

func (d *decoder81) NextEntryValueFloat(bitlen int) (done bool, value float64, err error) {
	// NextEntry
	top := d.top()
	if top == nil {
		return false, 0, errEmptyDecoderStack
	}
	top.Index++
	if top.Index == top.LenHint {
		return true, 0, nil
	}
	// Check fastpath
	var ttNext *vdl.Type
	if top.NextEntryType != nil {
		ttNext = top.NextEntryType
		if top.Type.Kind() == vdl.Map {
			top.Flag = top.Flag.FlipIsMapKey()
		}
	} else {
		// StartValue
		if ttNext, err = d.dfsNextType(); err != nil {
			return false, 0, err
		}
		var flag decStackFlag
		if ttNext, _, flag, err = d.setupType(ttNext, nil); err != nil {
			return false, 0, err
		}
		if !flag.IsAny() { // can't enable fastpath for Any types.
			top.NextEntryType = ttNext
		}
		// Check compatibility, and set NextEntryData.
		switch ttNext.Kind() {
		case vdl.Float32, vdl.Float64:
			top.NextEntryData = nextEntryData(ttNext.Kind().BitLen())
		case vdl.Byte, vdl.Uint16, vdl.Uint32, vdl.Uint64, vdl.Int8, vdl.Int16, vdl.Int32, vdl.Int64:
			top.NextEntryData = nextEntryMustConvert
		default:
			return false, 0, errIncompatibleDecode(ttNext, "float"+strconv.Itoa(bitlen))
		}
	}
	// Decode, avoiding unnecessary number conversions.
	switch flag := top.NextEntryData; {
	case flag <= nextEntryData(bitlen):
		value, err = binaryDecodeFloat(d.buf)
	default: // must convert
		value, err = d.decodeFloat(ttNext, uint(bitlen))
	}
	return false, value, err
}

func (d *decoder81) NextEntryValueTypeObject() (done bool, value *vdl.Type, err error) {
	// NextEntry
	top := d.top()
	if top == nil {
		return false, nil, errEmptyDecoderStack
	}
	top.Index++
	if top.Index == top.LenHint {
		return true, nil, nil
	}
	// Check fastpath
	if top.NextEntryType != nil {
		if top.Type.Kind() == vdl.Map {
			top.Flag = top.Flag.FlipIsMapKey()
		}
	} else {
		// StartValue
		var ttNext *vdl.Type
		if ttNext, err = d.dfsNextType(); err != nil {
			return false, nil, err
		}
		var flag decStackFlag
		if ttNext, _, flag, err = d.setupType(ttNext, nil); err != nil {
			return false, nil, err
		}
		if !flag.IsAny() { // can't enable fastpath for Any types.
			top.NextEntryType = ttNext
		}
		// Check compatibility
		switch ttNext.Kind() {
		case vdl.TypeObject:
		default:
			return false, nil, errIncompatibleDecode(ttNext, "typeobject")
		}
	}
	// Decode
	value, err = d.binaryDecodeType()
	return false, value, err
}
