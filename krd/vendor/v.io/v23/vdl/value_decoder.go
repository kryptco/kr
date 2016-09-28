// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

import (
	"errors"
	"fmt"
	"math"
)

var (
	errEmptyDecoderStack = errors.New("vdl: empty decoder stack")
)

// Decoder returns a decoder that traverses vv.
func (vv *Value) Decoder() Decoder {
	return &valueDecoder{initial: vv}
}

// ValueDecoder is an implementation of Decoder for vdl Value.
type valueDecoder struct {
	initial    *Value
	ignoreNext bool
	stack      []vdStackEntry
}

type vdStackEntry struct {
	Value      *Value   // the vdl Value
	Index      int      // next index or field (index into Keys for map/set)
	NumStarted int      // hold state for multiple StartValue() calls
	IsAny      bool     // true iff this value is within an any value
	IsOptional bool     // true iff this value is within an optional value
	Keys       []*Value // keys for set/map
}

func (d *valueDecoder) StartValue(want *Type) error {
	if d.ignoreNext {
		d.ignoreNext = false
		return nil
	}
	var vv *Value
	if top := d.top(); top == nil {
		vv = d.initial
	} else {
		switch top.Value.Kind() {
		case Array, List:
			vv = top.Value.Index(top.Index)
		case Set:
			vv = top.Keys[top.Index]
		case Map:
			switch top.NumStarted % 2 {
			case 0:
				vv = top.Keys[top.Index]
			case 1:
				vv = top.Value.MapIndex(top.Keys[top.Index])
			}
		case Struct:
			vv = top.Value.StructField(top.Index)
		case Union:
			_, vv = top.Value.UnionField()
		default:
			return fmt.Errorf("vdl: can't StartValue on %v", top.Value.Type())
		}
		top.NumStarted++
	}
	var isAny, isOptional bool
	if vv.Kind() == Any {
		isAny = true
		if !vv.IsNil() {
			vv = vv.Elem()
		}
	}
	if vv.Kind() == Optional {
		isOptional = true
		if !vv.IsNil() {
			vv = vv.Elem()
		}
	}
	// Check compatibility between the actual type and the want type.  Since
	// compatibility applies to the entire static type, we only need to perform
	// this check for top-level decoded values, and subsequently for decoded any
	// values.  We skip checking non-composite want types, since those will be
	// naturally caught by the Decode* calls anyways.
	if len(d.stack) == 0 || isAny {
		switch want.Kind() {
		case Optional, Array, List, Set, Map, Struct, Union:
			if !Compatible(vv.Type(), want) {
				return fmt.Errorf("vdl: incompatible decode from %v into %v", vv.Type(), want)
			}
		}
	}
	entry := vdStackEntry{
		Value:      vv,
		IsAny:      isAny,
		IsOptional: isOptional,
		Index:      -1,
	}
	if vv.Kind() == Map || vv.Kind() == Set {
		entry.Keys = vv.Keys()
	}
	d.stack = append(d.stack, entry)
	return nil
}

func (d *valueDecoder) IgnoreNextStartValue() {
	d.ignoreNext = true
}

func (d *valueDecoder) FinishValue() error {
	if len(d.stack) == 0 {
		return errEmptyDecoderStack
	}
	d.stack = d.stack[:len(d.stack)-1]
	d.ignoreNext = false
	return nil
}

func (d *valueDecoder) SkipValue() error {
	d.ignoreNext = false
	return nil
}

func (d *valueDecoder) NextEntry() (bool, error) {
	top := d.top()
	if top == nil {
		return false, errEmptyDecoderStack
	}
	top.Index++
	switch top.Value.Kind() {
	case Array, List, Set, Map:
		switch {
		case top.Index == top.Value.Len():
			return true, nil
		case top.Index > top.Value.Len():
			return false, fmt.Errorf("vdl: NextEntry called after done, stack: %+v", d.stack)
		}
	}
	return false, nil
}

func (d *valueDecoder) NextField() (int, error) {
	top := d.top()
	if top == nil {
		return -1, errEmptyDecoderStack
	}
	switch top.Value.Kind() {
	case Union:
		if top.Index != -1 {
			return -1, nil
		}
		top.Index, _ = top.Value.UnionField()
	case Struct:
		top.Index++
		if top.Index >= top.Value.Type().NumField() {
			return -1, nil
		}
	}
	return top.Index, nil
}

func (d *valueDecoder) topValue() *Value {
	if top := d.top(); top != nil {
		return top.Value
	}
	return nil
}

func (d *valueDecoder) Type() *Type {
	if top := d.top(); top != nil {
		return top.Value.Type()
	}
	return nil
}

func (d *valueDecoder) IsAny() bool {
	if top := d.top(); top != nil {
		return top.IsAny
	}
	return false
}

func (d *valueDecoder) IsOptional() bool {
	if top := d.top(); top != nil {
		return top.IsOptional
	}
	return false
}

func (d *valueDecoder) IsNil() bool {
	if top := d.top(); top != nil {
		return top.Value.IsNil()
	}
	return false
}

func (d *valueDecoder) Index() int {
	if top := d.top(); top != nil {
		return top.Index
	}
	return -1
}

func (d *valueDecoder) LenHint() int {
	if top := d.top(); top != nil {
		switch top.Value.Kind() {
		case Array, List, Set, Map:
			return top.Value.Len()
		}
	}
	return -1
}

func (d *valueDecoder) DecodeBool() (bool, error) {
	topV := d.topValue()
	if topV == nil {
		return false, errEmptyDecoderStack
	}
	if topV.Kind() == Bool {
		return topV.Bool(), nil
	}
	return false, fmt.Errorf("vdl: incompatible decode from %v into bool", topV.Type())
}

func (d *valueDecoder) DecodeUint(bitlen int) (uint64, error) {
	const errFmt = "vdl: conversion from %v into uint%d loses precision: %v"
	topV, ubitlen := d.topValue(), uint(bitlen)
	if topV == nil {
		return 0, errEmptyDecoderStack
	}
	switch topV.Kind() {
	case Byte, Uint16, Uint32, Uint64:
		x := topV.Uint()
		if shift := 64 - ubitlen; x != (x<<shift)>>shift {
			return 0, fmt.Errorf(errFmt, topV.Type(), bitlen, x)
		}
		return x, nil
	case Int8, Int16, Int32, Int64:
		x := topV.Int()
		ux := uint64(x)
		if shift := 64 - ubitlen; x < 0 || ux != (ux<<shift)>>shift {
			return 0, fmt.Errorf(errFmt, topV.Type(), bitlen, x)
		}
		return ux, nil
	case Float32, Float64:
		x := topV.Float()
		ux := uint64(x)
		if shift := 64 - ubitlen; x != float64(ux) || ux != (ux<<shift)>>shift {
			return 0, fmt.Errorf(errFmt, topV.Type(), bitlen, x)
		}
		return ux, nil
	default:
		return 0, fmt.Errorf("vdl: incompatible decode from %v into uint%d", topV.Type(), bitlen)
	}
}

func (d *valueDecoder) DecodeInt(bitlen int) (int64, error) {
	const errFmt = "vdl: conversion from %v into int%d loses precision: %v"
	topV, ubitlen := d.topValue(), uint(bitlen)
	if topV == nil {
		return 0, errEmptyDecoderStack
	}
	switch topV.Kind() {
	case Byte, Uint16, Uint32, Uint64:
		x := topV.Uint()
		ix := int64(x)
		// The shift uses 65 since the topmost bit is the sign bit.  I.e. 32 bit
		// numbers should be shifted by 33 rather than 32.
		if shift := 65 - ubitlen; ix < 0 || x != (x<<shift)>>shift {
			return 0, fmt.Errorf(errFmt, topV.Type(), bitlen, x)
		}
		return ix, nil
	case Int8, Int16, Int32, Int64:
		x := topV.Int()
		if shift := 64 - ubitlen; x != (x<<shift)>>shift {
			return 0, fmt.Errorf(errFmt, topV.Type(), bitlen, x)
		}
		return x, nil
	case Float32, Float64:
		x := topV.Float()
		ix := int64(x)
		if shift := 64 - ubitlen; x != float64(ix) || ix != (ix<<shift)>>shift {
			return 0, fmt.Errorf(errFmt, topV.Type(), bitlen, x)
		}
		return ix, nil
	default:
		return 0, fmt.Errorf("vdl: incompatible decode from %v into int%d", topV.Type(), bitlen)
	}
}

func (d *valueDecoder) DecodeFloat(bitlen int) (float64, error) {
	const errFmt = "vdl: conversion from %v into float%d loses precision: %v"
	topV := d.topValue()
	if topV == nil {
		return 0, errEmptyDecoderStack
	}
	switch topV.Kind() {
	case Byte, Uint16, Uint32, Uint64:
		x := topV.Uint()
		var max uint64
		if bitlen > 32 {
			max = float64MaxInt
		} else {
			max = float32MaxInt
		}
		if x > max {
			return 0, fmt.Errorf(errFmt, topV.Type(), bitlen, x)
		}
		return float64(x), nil
	case Int8, Int16, Int32, Int64:
		x := topV.Int()
		var min, max int64
		if bitlen > 32 {
			min, max = float64MinInt, float64MaxInt
		} else {
			min, max = float32MinInt, float32MaxInt
		}
		if x < min || x > max {
			return 0, fmt.Errorf(errFmt, topV.Type(), bitlen, x)
		}
		return float64(x), nil
	case Float32, Float64:
		x := topV.Float()
		if bitlen <= 32 && (x < -math.MaxFloat32 || x > math.MaxFloat32) {
			return 0, fmt.Errorf(errFmt, topV.Type(), bitlen, x)
		}
		return x, nil
	default:
		return 0, fmt.Errorf("vdl: incompatible decode from %v into float%d", topV.Type(), bitlen)
	}
}

func (d *valueDecoder) DecodeBytes(fixedlen int, v *[]byte) error {
	topV := d.topValue()
	if topV == nil {
		return errEmptyDecoderStack
	}
	if !topV.Type().IsBytes() {
		return DecodeConvertedBytes(d, fixedlen, v)
	}
	if fixedlen >= 0 && fixedlen != topV.Len() {
		return fmt.Errorf("vdl: %v got %d bytes, want fixed len %d", topV.Type(), topV.Len(), fixedlen)
	}
	if cap(*v) >= topV.Len() {
		*v = (*v)[:topV.Len()]
	} else {
		*v = make([]byte, topV.Len())
	}
	copy(*v, topV.Bytes())
	return nil
}

func (d *valueDecoder) DecodeString() (string, error) {
	topV := d.topValue()
	if topV == nil {
		return "", errEmptyDecoderStack
	}
	switch topV.Kind() {
	case String:
		return topV.RawString(), nil
	case Enum:
		return topV.EnumLabel(), nil
	}
	return "", fmt.Errorf("vdl: incompatible decode from %v into string", topV.Type())
}

func (d *valueDecoder) DecodeTypeObject() (*Type, error) {
	topV := d.topValue()
	if topV == nil {
		return nil, errEmptyDecoderStack
	}
	if topV.Type() == TypeObjectType {
		return topV.TypeObject(), nil
	}
	return nil, fmt.Errorf("vdl: incompatible decode from %v into typeobject", topV.Type())
}

func (d *valueDecoder) top() *vdStackEntry {
	if len(d.stack) > 0 {
		return &d.stack[len(d.stack)-1]
	}
	return nil
}

// The ReadValue* and NextEntryValue* methods just call methods in sequence.

func (d *valueDecoder) ReadValueBool() (bool, error) {
	if err := d.StartValue(BoolType); err != nil {
		return false, err
	}
	value, err := d.DecodeBool()
	if err != nil {
		return false, err
	}
	return value, d.FinishValue()
}

func (d *valueDecoder) ReadValueString() (string, error) {
	if err := d.StartValue(StringType); err != nil {
		return "", err
	}
	value, err := d.DecodeString()
	if err != nil {
		return "", err
	}
	return value, d.FinishValue()
}

func (d *valueDecoder) ReadValueUint(bitlen int) (uint64, error) {
	if err := d.StartValue(Uint64Type); err != nil {
		return 0, err
	}
	value, err := d.DecodeUint(bitlen)
	if err != nil {
		return 0, err
	}
	return value, d.FinishValue()
}

func (d *valueDecoder) ReadValueInt(bitlen int) (int64, error) {
	if err := d.StartValue(Int64Type); err != nil {
		return 0, err
	}
	value, err := d.DecodeInt(bitlen)
	if err != nil {
		return 0, err
	}
	return value, d.FinishValue()
}

func (d *valueDecoder) ReadValueFloat(bitlen int) (float64, error) {
	if err := d.StartValue(Float64Type); err != nil {
		return 0, err
	}
	value, err := d.DecodeFloat(bitlen)
	if err != nil {
		return 0, err
	}
	return value, d.FinishValue()
}

func (d *valueDecoder) ReadValueTypeObject() (*Type, error) {
	if err := d.StartValue(TypeObjectType); err != nil {
		return nil, err
	}
	value, err := d.DecodeTypeObject()
	if err != nil {
		return nil, err
	}
	return value, d.FinishValue()
}

func (d *valueDecoder) ReadValueBytes(fixedLen int, x *[]byte) error {
	if err := d.StartValue(ttByteList); err != nil {
		return err
	}
	if err := d.DecodeBytes(fixedLen, x); err != nil {
		return err
	}
	return d.FinishValue()
}

func (d *valueDecoder) NextEntryValueBool() (done bool, _ bool, _ error) {
	if done, err := d.NextEntry(); done || err != nil {
		return done, false, err
	}
	value, err := d.ReadValueBool()
	return false, value, err
}

func (d *valueDecoder) NextEntryValueString() (done bool, _ string, _ error) {
	if done, err := d.NextEntry(); done || err != nil {
		return done, "", err
	}
	value, err := d.ReadValueString()
	return false, value, err
}

func (d *valueDecoder) NextEntryValueUint(bitlen int) (done bool, _ uint64, _ error) {
	if done, err := d.NextEntry(); done || err != nil {
		return done, 0, err
	}
	value, err := d.ReadValueUint(bitlen)
	return false, value, err
}

func (d *valueDecoder) NextEntryValueInt(bitlen int) (done bool, _ int64, _ error) {
	if done, err := d.NextEntry(); done || err != nil {
		return done, 0, err
	}
	value, err := d.ReadValueInt(bitlen)
	return false, value, err
}

func (d *valueDecoder) NextEntryValueFloat(bitlen int) (done bool, _ float64, _ error) {
	if done, err := d.NextEntry(); done || err != nil {
		return done, 0, err
	}
	value, err := d.ReadValueFloat(bitlen)
	return false, value, err
}

func (d *valueDecoder) NextEntryValueTypeObject() (done bool, _ *Type, _ error) {
	if done, err := d.NextEntry(); done || err != nil {
		return done, nil, err
	}
	value, err := d.ReadValueTypeObject()
	return false, value, err
}
