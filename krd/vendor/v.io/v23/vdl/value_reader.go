// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

import (
	"fmt"
)

// VDLRead uses dec to decode a value into vv.  If vv isn't valid (i.e. has no
// type), it will be filled in the exact type of value read from the decoder.
// Otherwise the type of vv must be compatible with the type of the value read
// from the decoder.
func (vv *Value) VDLRead(dec Decoder) error {
	if vv == nil {
		return errReadIntoNilValue
	}
	ttWant := AnyType
	if vv.IsValid() {
		ttWant = vv.Type()
	}
	if err := dec.StartValue(ttWant); err != nil {
		return err
	}
	if !vv.IsValid() {
		// Initialize vv to the zero value of the exact type read from the decoder.
		// It is as if vv were initialized with that type to begin with.  The
		// top-level any type is dropped.
		switch {
		case dec.IsOptional() && !dec.IsNil():
			*vv = *ZeroValue(OptionalType(dec.Type()))
		default:
			*vv = *ZeroValue(dec.Type())
		}
	}
	dec.IgnoreNextStartValue()
	return vv.read(dec)
}

func (vv *Value) read(dec Decoder) error {
	if err := dec.StartValue(vv.Type()); err != nil {
		return err
	}
	// Handle nil decoded values first, to simplify the rest of the cases.
	if dec.IsNil() {
		return vv.readFromNil(dec)
	}
	// Handle non-nil values.  If vv is any or optional we need to treat it
	// specially, since it needs to be assigned after the value is read.
	vvFill, makeOptional := vv, false
	switch {
	case vv.Kind() == Any:
		// Fill in a value of the type read from the decoder.
		vvFill = ZeroValue(dec.Type())
		makeOptional = dec.IsOptional()
	case vv.Kind() == Optional:
		// Fill in a value of our elem type.
		vvFill = ZeroValue(vv.Type().Elem())
		makeOptional = true
	}
	if err := vvFill.readNonNilValue(dec); err != nil {
		return err
	}
	// Finished reading, handle any and optional cases.
	if makeOptional {
		vvFill = OptionalValue(vvFill)
	}
	if vv.Kind() == Any || vv.Kind() == Optional {
		vv.Assign(vvFill)
	}
	return dec.FinishValue()
}

func (vv *Value) readFromNil(dec Decoder) error {
	// We've already checked for compatibility above, so we know that we're
	// allowed to set the nil value.
	switch {
	case vv.Kind() == Any:
		// Make sure that any(nil) and ?T(nil) are retained correctly in the any.
		vv.Assign(ZeroValue(dec.Type()))
	case vv.Kind() == Optional:
		// Just set the optional to nil.
		vv.Assign(nil)
	default:
		return fmt.Errorf("vdl: can't decode nil into non-any non-optional %v", vv.Type())
	}
	return dec.FinishValue()
}

func (vv *Value) readNonNilValue(dec Decoder) error {
	if vv.Type().IsBytes() {
		var val []byte
		fixedLen := -1
		if vv.Kind() == Array {
			fixedLen = vv.Type().Len()
			val = make([]byte, fixedLen)
		}
		if err := dec.DecodeBytes(fixedLen, &val); err != nil {
			return err
		}
		vv.AssignBytes(val)
		return nil
	}
	switch vv.Kind() {
	case Bool:
		val, err := dec.DecodeBool()
		if err != nil {
			return err
		}
		vv.AssignBool(val)
		return nil
	case Byte, Uint16, Uint32, Uint64:
		val, err := dec.DecodeUint(vv.Kind().BitLen())
		if err != nil {
			return err
		}
		vv.AssignUint(val)
		return nil
	case Int8, Int16, Int32, Int64:
		val, err := dec.DecodeInt(vv.Kind().BitLen())
		if err != nil {
			return err
		}
		vv.AssignInt(val)
		return nil
	case Float32, Float64:
		val, err := dec.DecodeFloat(vv.Kind().BitLen())
		if err != nil {
			return err
		}
		vv.AssignFloat(val)
		return nil
	case String:
		val, err := dec.DecodeString()
		if err != nil {
			return err
		}
		vv.AssignString(val)
		return nil
	case TypeObject:
		val, err := dec.DecodeTypeObject()
		if err != nil {
			return err
		}
		vv.AssignTypeObject(val)
		return nil
	case Enum:
		val, err := dec.DecodeString()
		if err != nil {
			return err
		}
		index := vv.Type().EnumIndex(val)
		if index == -1 {
			return fmt.Errorf("vdl: %v invalid enum label %q", vv.Type(), val)
		}
		vv.AssignEnumIndex(index)
		return nil
	case Array:
		return vv.readArray(dec)
	case List:
		return vv.readList(dec)
	case Set:
		return vv.readSet(dec)
	case Map:
		return vv.readMap(dec)
	case Struct:
		return vv.readStruct(dec)
	case Union:
		return vv.readUnion(dec)
	}
	panic(fmt.Errorf("vdl: unhandled type %v in VDLRead", vv.Type()))
}

func (vv *Value) readArray(dec Decoder) error {
	index := 0
	for {
		switch done, err := dec.NextEntry(); {
		case err != nil:
			return err
		case done != (index >= vv.Type().Len()):
			return fmt.Errorf("array len mismatch, done:%v index:%d len:%d %v", done, index, vv.Type().Len(), vv.Type())
		case done:
			return nil
		}
		if err := vv.Index(index).read(dec); err != nil {
			return err
		}
		index++
	}
}

func (vv *Value) readList(dec Decoder) error {
	switch len := dec.LenHint(); {
	case len >= 0:
		vv.AssignLen(len)
	default:
		// Assign 0 length when we don't have a hint.
		vv.AssignLen(0)
	}
	index := 0
	for {
		switch done, err := dec.NextEntry(); {
		case err != nil:
			return err
		case done:
			return nil
		}
		if needLen := index + 1; needLen > vv.Len() {
			var cap int
			if needLen <= 1024 {
				cap = needLen * 2
			} else {
				cap = needLen + needLen/4
			}
			// Grow the underlying buffer.  The first AssignLen grows the buffer to
			// the capacity, while the second AssignLen sets the actual length.
			//
			// TODO(toddw): Consider changing the Value API to either add an Append
			// method, or to allow the user to explicitly manage the capacity.
			vv.AssignLen(cap)
			vv.AssignLen(needLen)
		}
		if err := vv.Index(index).read(dec); err != nil {
			return err
		}
		index++
	}
}

func (vv *Value) readSet(dec Decoder) error {
	for {
		switch done, err := dec.NextEntry(); {
		case err != nil:
			return err
		case done:
			return nil
		}
		key := ZeroValue(vv.Type().Key())
		if err := key.read(dec); err != nil {
			return err
		}
		vv.AssignSetKey(key)
	}
}

func (vv *Value) readMap(dec Decoder) error {
	for {
		switch done, err := dec.NextEntry(); {
		case err != nil:
			return err
		case done:
			return nil
		}
		key := ZeroValue(vv.Type().Key())
		if err := key.read(dec); err != nil {
			return err
		}
		elem := ZeroValue(vv.Type().Elem())
		if err := elem.read(dec); err != nil {
			return err
		}
		vv.AssignMapIndex(key, elem)
	}
}

func (vv *Value) readStruct(dec Decoder) error {
	// Reset to zero struct, since fields may be missing.
	vv.Assign(nil)
	tt, decType := vv.Type(), dec.Type()
	for {
		index, err := dec.NextField()
		switch {
		case err != nil:
			return err
		case index == -1:
			return nil
		}
		if decType != tt {
			index = tt.FieldIndexByName(decType.Field(index).Name)
			if index == -1 {
				if err := dec.SkipValue(); err != nil {
					return err
				}
				continue
			}
		}
		if err := vv.StructField(index).read(dec); err != nil {
			return err
		}
	}
}

func (vv *Value) readUnion(dec Decoder) error {
	tt, decType := vv.Type(), dec.Type()
	index, err := dec.NextField()
	switch {
	case err != nil:
		return err
	case index == -1:
		return fmt.Errorf("missing field in union %v, from %v", tt, decType)
	}
	var ttField Field
	if decType == tt {
		ttField = tt.Field(index)
	} else {
		name := decType.Field(index).Name
		ttField, index = tt.FieldByName(name)
		if index == -1 {
			return fmt.Errorf("field %q not in union %v, from %v", name, tt, decType)
		}
	}
	vvElem := ZeroValue(ttField.Type)
	if err := vvElem.read(dec); err != nil {
		return err
	}
	vv.AssignField(index, vvElem)
	switch index, err := dec.NextField(); {
	case err != nil:
		return err
	case index != -1:
		return fmt.Errorf("extra field %d in union %v, from %v", index, tt, decType)
	}
	return nil
}
