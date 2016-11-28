// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	errWriteMustReflect = errors.New("vdl: write must be handled via reflection")
)

// Write uses enc to encode value v, calling VDLWrite methods and fast compiled
// writers when available, and using reflection otherwise.  This is basically an
// all-purpose VDLWrite implementation.
func Write(enc Encoder, v interface{}) error {
	if v == nil {
		return enc.NilValue(AnyType)
	}
	rv := reflect.ValueOf(v)
	// Fastpath check for non-reflect support.  Unfortunately we must use
	// reflection to detect the case where v is a pointer, which is handled by the
	// more complicated optional-checking logic in writeReflect.
	//
	// TODO(toddw): *vom.RawBytes is 50% faster if we could special-case it here,
	// without breaking support for optional types.
	if rv.Kind() != reflect.Ptr {
		if err := writeNonReflect(enc, v); err != errWriteMustReflect {
			return err
		}
	}
	tt, err := TypeFromReflect(rv.Type())
	if err != nil {
		return err
	}
	return writeReflect(enc, rv, tt)
}

func writeNonReflect(enc Encoder, v interface{}) error {
	switch x := v.(type) {
	case Writer:
		// Writer handles code-generated VDLWrite methods, and special-cases such as
		// vdl.Value and vom.RawBytes.
		return x.VDLWrite(enc)

		// Cases after this point are purely performance optimizations.
		// TODO(toddw): Handle other common cases.
	case []byte:
		return enc.WriteValueBytes(ttByteList, x)
	}
	return errWriteMustReflect
}

// WriteReflect is like Write, but takes a reflect.Value argument.
func WriteReflect(enc Encoder, rv reflect.Value) error {
	if !rv.IsValid() {
		return enc.NilValue(AnyType)
	}
	tt, err := TypeFromReflect(rv.Type())
	if err != nil {
		return err
	}
	return writeReflect(enc, rv, tt)
}

func writeReflect(enc Encoder, rv reflect.Value, tt *Type) error {
	// Fastpath check for non-reflect support.  Optional types are tricky, since
	// they may be nil, and need SetNextStartValueIsOptional() to be set, so they
	// can't use this fastpath.  This handles the non-nil *vom.RawBytes and
	// *vdl.Value cases, and avoids an expensive copy of all their fields.
	if tt.Kind() != Optional && (rv.Kind() != reflect.Ptr || !rv.IsNil()) {
		if err := writeNonReflect(enc, rv.Interface()); err != errWriteMustReflect {
			return err
		}
	}
	// Walk pointers and interfaces in rv, and handle nil values.
	for {
		isPtr, isIface := rv.Kind() == reflect.Ptr, rv.Kind() == reflect.Interface
		if !isPtr && !isIface {
			break
		}
		if rv.IsNil() {
			switch {
			case tt.Kind() == TypeObject:
				// Treat nil *Type as AnyType.
				return AnyType.VDLWrite(enc)
			case tt.Kind() == Union && isIface:
				// Treat nil Union interface as the zero value of the type at index 0.
				return ZeroValue(tt).VDLWrite(enc)
			case tt.Kind() == Optional:
				enc.SetNextStartValueIsOptional()
				return enc.NilValue(tt)
			case tt == AnyType:
				return enc.NilValue(tt)
			}
			return fmt.Errorf("vdl: can't encode nil from non-any non-optional %v", tt)
		}
		rv = rv.Elem()
		// Recompute tt as we pass interface boundaries.  There's no need to
		// recompute as we traverse pointers, since tt won't change.
		if isIface {
			var err error
			if tt, err = TypeFromReflect(rv.Type()); err != nil {
				return err
			}
		}
	}
	if tt.Kind() == Optional {
		enc.SetNextStartValueIsOptional()
	}
	// Check for faster non-reflect support, which also handles vdl.Value and
	// vom.RawBytes, and any other special-cases.
	if err := writeNonReflect(enc, rv.Interface()); err != errWriteMustReflect {
		return err
	}
	if reflect.PtrTo(rv.Type()).Implements(rtVDLWriter) {
		if rv.CanAddr() {
			return writeNonReflect(enc, rv.Addr().Interface())
		} else {
			// This handles the case where rv implements VDLWrite with a pointer
			// receiver, but we can't address rv to get a pointer.  E.g.
			//    type Foo string
			//    func (x *Foo) VDLWrite(enc vdl.Encoder) error {...}
			//    rv := Foo{}
			//
			// TODO(toddw): Do we need to handle this case?
			rvPtr := reflect.New(rv.Type())
			rvPtr.Elem().Set(rv)
			return writeNonReflect(enc, rvPtr.Interface())
		}
	}
	// Handle marshaling from native type to wire type.
	if ni := nativeInfoFromNative(rv.Type()); ni != nil {
		rvWirePtr := reflect.New(ni.WireType)
		if err := ni.FromNative(rvWirePtr, rv); err != nil {
			return err
		}
		return writeReflect(enc, rvWirePtr.Elem(), tt)
	}
	// Handle errors that are implemented by arbitrary rv values.  E.g. the Go
	// standard errors.errorString implements the error interface, but is an
	// invalid vdl type since it doesn't have any exported fields.
	//
	// See corresponding special-case in reflect_type.go
	if tt == ErrorType {
		if rv.Type().Implements(rtError) {
			return writeNonNilError(enc, rv)
		}
		if rv.CanAddr() && rv.Addr().Type().Implements(rtError) {
			return writeNonNilError(enc, rv.Addr())
		}
	}
	tt = tt.NonOptional()
	// Handle fastpath values.
	if ttWriteHasFastpath(tt) {
		return writeValueFastpath(enc, rv, tt)
	}
	// Handle composite wire values.
	if err := enc.StartValue(tt); err != nil {
		return err
	}
	var err error
	switch tt.Kind() {
	case Array, List:
		err = writeArrayOrList(enc, rv, tt)
	case Set, Map:
		err = writeSetOrMap(enc, rv, tt)
	case Struct:
		err = writeStruct(enc, rv, tt)
	case Union:
		err = writeUnion(enc, rv, tt)
	default:
		// Special representations like vdl.Type, vdl.Value and vom.RawBytes
		// implement VDLWrite, and were handled by writeNonReflect.  Nil optional
		// and any were handled by the pointer-flattening loop.
		return fmt.Errorf("vdl: Write unhandled type %v %v", rv.Type(), tt)
	}
	if err != nil {
		return err
	}
	return enc.FinishValue()
}

// writeNonNilError writes rvNative, which must be a non-nil implementation of
// the Go error interface, out to enc.
func writeNonNilError(enc Encoder, rvNative reflect.Value) error {
	ni := nativeInfoFromNative(rtError)
	if ni == nil {
		return errNoRegisterNativeError
	}
	rvWirePtr := reflect.New(ni.WireType)
	if err := ni.FromNative(rvWirePtr, rvNative); err != nil {
		return err
	}
	return writeReflect(enc, rvWirePtr.Elem(), ErrorType)
}

func extractBytes(rv reflect.Value, tt *Type) []byte {
	// Go doesn't allow type conversions from []MyByte to []byte, but the reflect
	// package does let us perform this conversion.
	if tt.Kind() == List {
		return rv.Bytes()
	}
	switch {
	case rv.CanAddr():
		return rv.Slice(0, tt.Len()).Bytes()
	case tt.Elem() == ByteType:
		// Unaddressable arrays can't be sliced, so we must copy the bytes.
		// TODO(toddw): Find a better way to do this.
		bytes := make([]byte, tt.Len())
		reflect.Copy(reflect.ValueOf(bytes), rv)
		return bytes
	default:
		// Unaddressable arrays can't be sliced, so we must copy the bytes.
		// TODO(toddw): Find a better way to do this.
		rt, len := rv.Type(), tt.Len()
		rvSlice := reflect.MakeSlice(reflect.SliceOf(rt.Elem()), len, len)
		reflect.Copy(rvSlice, rv)
		return rvSlice.Bytes()
	}
}

func ttWriteHasFastpath(tt *Type) bool {
	switch tt.Kind() {
	case Bool, String, Enum, Byte, Uint16, Uint32, Uint64, Int8, Int16, Int32, Int64, Float32, Float64:
		return true
	}
	return tt.IsBytes()
}

func writeValueFastpath(enc Encoder, rv reflect.Value, tt *Type) error {
	switch tt.Kind() {
	case Bool:
		return enc.WriteValueBool(tt, rv.Bool())
	case String:
		return enc.WriteValueString(tt, rv.String())
	case Enum:
		// TypeFromReflect already validated String(); call without error checking.
		return enc.WriteValueString(tt, rv.Interface().(stringer).String())
	case Byte, Uint16, Uint32, Uint64:
		return enc.WriteValueUint(tt, rv.Uint())
	case Int8, Int16, Int32, Int64:
		return enc.WriteValueInt(tt, rv.Int())
	case Float32, Float64:
		return enc.WriteValueFloat(tt, rv.Float())
	}
	if !tt.IsBytes() {
		return fmt.Errorf("vdl: writeValueFastpath called on non-fastpath type %v, %v", tt, rv.Type())
	}
	return enc.WriteValueBytes(tt, extractBytes(rv, tt))
}

func writeNextEntryFastpath(enc Encoder, rv reflect.Value, tt *Type) error {
	switch tt.Kind() {
	case Bool:
		return enc.NextEntryValueBool(tt, rv.Bool())
	case String:
		return enc.NextEntryValueString(tt, rv.String())
	case Enum:
		// TypeFromReflect already validated String(); call without error checking.
		return enc.NextEntryValueString(tt, rv.Interface().(stringer).String())
	case Byte, Uint16, Uint32, Uint64:
		return enc.NextEntryValueUint(tt, rv.Uint())
	case Int8, Int16, Int32, Int64:
		return enc.NextEntryValueInt(tt, rv.Int())
	case Float32, Float64:
		return enc.NextEntryValueFloat(tt, rv.Float())
	}
	if !tt.IsBytes() {
		return fmt.Errorf("vdl: writeNextEntryFastpath called on non-fastpath type %v, %v", tt, rv.Type())
	}
	return enc.NextEntryValueBytes(tt, extractBytes(rv, tt))
}

func writeNextFieldFastpath(enc Encoder, rv reflect.Value, tt *Type, index int) error {
	switch tt.Kind() {
	case Bool:
		return enc.NextFieldValueBool(index, tt, rv.Bool())
	case String:
		return enc.NextFieldValueString(index, tt, rv.String())
	case Enum:
		// TypeFromReflect already validated String(); call without error checking.
		return enc.NextFieldValueString(index, tt, rv.Interface().(stringer).String())
	case Byte, Uint16, Uint32, Uint64:
		return enc.NextFieldValueUint(index, tt, rv.Uint())
	case Int8, Int16, Int32, Int64:
		return enc.NextFieldValueInt(index, tt, rv.Int())
	case Float32, Float64:
		return enc.NextFieldValueFloat(index, tt, rv.Float())
	}
	if !tt.IsBytes() {
		return fmt.Errorf("vdl: writeNextFieldFastpath called on non-fastpath type %v, %v", tt, rv.Type())
	}
	return enc.NextFieldValueBytes(index, tt, extractBytes(rv, tt))
}

func writeArrayOrList(enc Encoder, rv reflect.Value, tt *Type) error {
	if tt.Kind() == List {
		if err := enc.SetLenHint(rv.Len()); err != nil {
			return err
		}
	}
	ttElem := tt.Elem()
	for ix := 0; ix < rv.Len(); ix++ {
		rvElem := rv.Index(ix)
		if ttWriteHasFastpath(ttElem) {
			if err := writeNextEntryFastpath(enc, rvElem, ttElem); err != nil {
				return err
			}
		} else {
			if err := enc.NextEntry(false); err != nil {
				return err
			}
			if err := writeReflect(enc, rvElem, ttElem); err != nil {
				return err
			}
		}
	}
	return enc.NextEntry(true)
}

func writeSetOrMap(enc Encoder, rv reflect.Value, tt *Type) error {
	if err := enc.SetLenHint(rv.Len()); err != nil {
		return err
	}
	kind, ttKey := tt.Kind(), tt.Key()
	for _, rvKey := range rv.MapKeys() {
		if ttWriteHasFastpath(ttKey) {
			if err := writeNextEntryFastpath(enc, rvKey, ttKey); err != nil {
				return err
			}
		} else {
			if err := enc.NextEntry(false); err != nil {
				return err
			}
			if err := writeReflect(enc, rvKey, ttKey); err != nil {
				return err
			}
		}
		if kind == Map {
			if err := writeReflect(enc, rv.MapIndex(rvKey), tt.Elem()); err != nil {
				return err
			}
		}
	}
	return enc.NextEntry(true)
}

func writeStruct(enc Encoder, rv reflect.Value, tt *Type) error {
	rt := rv.Type()
	// Loop through tt fields rather than rt fields, since the VDL type tt might
	// have ignored some of the fields in rt, e.g. unexported fields.
	for index := 0; index < tt.NumField(); index++ {
		field := tt.Field(index)
		rvField := rv.Field(rtFieldIndexByName(rt, field.Name))
		switch isZero, err := rvIsZeroValue(rvField, field.Type); {
		case err != nil:
			return err
		case isZero:
			continue // skip zero-valued fields
		}
		if ttWriteHasFastpath(field.Type) {
			if err := writeNextFieldFastpath(enc, rvField, field.Type, index); err != nil {
				return err
			}
		} else {
			if err := enc.NextField(index); err != nil {
				return err
			}
			if err := writeReflect(enc, rvField, field.Type); err != nil {
				return err
			}
		}
	}
	return enc.NextField(-1)
}

func writeUnion(enc Encoder, rv reflect.Value, tt *Type) error {
	// TypeFromReflect already validated Index().
	iface := rv.Interface()
	index := iface.(indexer).Index()
	ttField := tt.Field(index).Type
	// Since this is a non-nil union, we're guaranteed rv is the concrete field
	// struct, so we can just grab the "Value" field.
	rvField := rv.Field(0)
	if ttWriteHasFastpath(ttField) {
		if err := writeNextFieldFastpath(enc, rvField, ttField, index); err != nil {
			return err
		}
	} else {
		if err := enc.NextField(index); err != nil {
			return err
		}
		if err := writeReflect(enc, rvField, ttField); err != nil {
			return err
		}
	}
	return enc.NextField(-1)
}
