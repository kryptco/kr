// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

import (
	"fmt"
)

// VDLWrite writes this vdl Value to the encoder.
func (vv *Value) VDLWrite(enc Encoder) error {
	// TODO(bprosnitz) Change this for new vv logic
	if vv.Kind() == Any {
		if vv.IsNil() {
			return enc.NilValue(vv.Type())
		}
		vv = vv.Elem()
	}
	if vv.Kind() == Optional {
		enc.SetNextStartValueIsOptional()
		if vv.IsNil() {
			return enc.NilValue(vv.Type())
		}
		vv = vv.Elem()
	}
	if err := enc.StartValue(vv.Type()); err != nil {
		return err
	}
	if err := vv.writeNonNilValue(enc); err != nil {
		return err
	}
	return enc.FinishValue()
}

func (vv *Value) writeNonNilValue(enc Encoder) error {
	if vv.Type().IsBytes() {
		return enc.EncodeBytes(vv.Bytes())
	}
	switch vv.Kind() {
	case Bool:
		return enc.EncodeBool(vv.Bool())
	case Byte, Uint16, Uint32, Uint64:
		return enc.EncodeUint(vv.Uint())
	case Int8, Int16, Int32, Int64:
		return enc.EncodeInt(vv.Int())
	case Float32, Float64:
		return enc.EncodeFloat(vv.Float())
	case String:
		return enc.EncodeString(vv.RawString())
	case TypeObject:
		return enc.EncodeTypeObject(vv.TypeObject())
	case Enum:
		return enc.EncodeString(vv.EnumLabel())
	case Array, List:
		return vv.writeArrayOrList(enc)
	case Set, Map:
		return vv.writeSetOrMap(enc)
	case Struct:
		return vv.writeStruct(enc)
	case Union:
		return vv.writeUnion(enc)
	}
	panic(fmt.Sprintf("unknown kind", vv.Kind()))
}

func (vv *Value) writeArrayOrList(enc Encoder) error {
	if vv.Kind() == List {
		if err := enc.SetLenHint(vv.Len()); err != nil {
			return err
		}
	}
	for i := 0; i < vv.Len(); i++ {
		if err := enc.NextEntry(false); err != nil {
			return err
		}
		if err := vv.Index(i).VDLWrite(enc); err != nil {
			return err
		}
	}
	return enc.NextEntry(true)
}

func (vv *Value) writeSetOrMap(enc Encoder) error {
	if err := enc.SetLenHint(vv.Len()); err != nil {
		return err
	}
	for _, key := range vv.Keys() {
		if err := enc.NextEntry(false); err != nil {
			return err
		}
		if err := key.VDLWrite(enc); err != nil {
			return err
		}
		if vv.Kind() == Map {
			if err := vv.MapIndex(key).VDLWrite(enc); err != nil {
				return err
			}
		}
	}
	return enc.NextEntry(true)
}

func (vv *Value) writeStruct(enc Encoder) error {
	for index := 0; index < vv.Type().NumField(); index++ {
		if vv.StructField(index).IsZero() {
			continue
		}
		if err := enc.NextField(index); err != nil {
			return err
		}
		if err := vv.StructField(index).VDLWrite(enc); err != nil {
			return err
		}
	}
	return enc.NextField(-1)
}

func (vv *Value) writeUnion(enc Encoder) error {
	index, field := vv.UnionField()
	if err := enc.NextField(index); err != nil {
		return err
	}
	if err := field.VDLWrite(enc); err != nil {
		return err
	}
	return enc.NextField(-1)
}
