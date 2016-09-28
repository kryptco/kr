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
	errReadMustReflect       = errors.New("vdl: read must be handled via reflection")
	errReadIntoNilValue      = errors.New("vdl: read into nil value")
	errReadReflectCantSet    = errors.New("vdl: read into unsettable reflect.Value")
	errReadAnyAlreadyStarted = errors.New("vdl: read into any after StartValue called")
	errReadAnyInterfaceOnly  = errors.New("vdl: read into any only supported for interfaces")
)

// Read uses dec to decode a value into v, calling VDLRead methods and fast
// compiled readers when available, and using reflection otherwise.  This is
// basically an all-purpose VDLRead implementation.
func Read(dec Decoder, v interface{}) error {
	if v == nil {
		return errReadIntoNilValue
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		// Fastpath check for non-reflect support.  Unfortunately we must use
		// reflection to detect the case where v is a nil pointer, which returns an
		// error in ReadReflect.
		//
		// TODO(toddw): If reflection is too slow, add the nil pointer check to all
		// VDLRead methods, as well as other readNonReflect cases below.
		if err := readNonReflect(dec, false, v); err != errReadMustReflect {
			return err
		}
	}
	return ReadReflect(dec, rv)
}

func readNonReflect(dec Decoder, calledStart bool, v interface{}) error {
	switch x := v.(type) {
	case Reader:
		// Reader handles code-generated VDLRead methods, and special-cases such as
		// vdl.Value and vom.RawBytes.
		if calledStart {
			dec.IgnoreNextStartValue()
		}
		return x.VDLRead(dec)
	case **Type:
		// Special-case type decoding, since we must assign the hash-consed pointer
		// for correctness, rather than filling in a newly-created Type.
		if calledStart {
			dec.IgnoreNextStartValue()
		}
		var err error
		*x, err = dec.ReadValueTypeObject()
		return err

		// TODO(toddw): Consider adding common-cases as performance optimizations.
	}
	return errReadMustReflect
}

// ReadReflect is like Read, but takes a reflect.Value argument.
func ReadReflect(dec Decoder, rv reflect.Value) error {
	if !rv.IsValid() {
		return errReadIntoNilValue
	}
	if !rv.CanSet() && rv.Kind() == reflect.Ptr && !rv.IsNil() {
		// Dereference the pointer a single time to make rv settable.
		rv = rv.Elem()
	}
	if !rv.CanSet() {
		return errReadReflectCantSet
	}
	tt, err := TypeFromReflect(rv.Type())
	if err != nil {
		return err
	}
	return readReflect(dec, false, rv, tt)
}

// readReflect uses dec to decode a value into rv, which has VDL type tt.  On
// success we guarantee that StartValue / FinishValue has been called on dec.
// If calledStart is true, StartValue has already been called.
func readReflect(dec Decoder, calledStart bool, rv reflect.Value, tt *Type) error {
	// Handle decoding into an any rv value first, since vom.RawBytes.VDLRead
	// doesn't support IgnoreNextStartValue, and requires that StartValue hasn't
	// been called yet.  Note that cases where the dec value is any but the rv
	// value isn't any will pass through.
	if tt == AnyType {
		return readIntoAny(dec, calledStart, rv)
	}
	// Handle optional types, which need StartValue to be called first to
	// determine whether the decoded value is nil.
	if tt.Kind() == Optional {
		if !calledStart {
			calledStart = true
			if err := dec.StartValue(tt); err != nil {
				return err
			}
		}
		// Handle nil decoded values next, to simplify the rest of the cases.  This
		// handles cases where the dec value is either any(nil) or optional(nil).
		if dec.IsNil() {
			return readFromNil(dec, rv, tt)
		}
	}
	// Now we know that rv isn't optional.  Flatten pointers and check for fast
	// non-reflect support.
	rv = rvFlattenPointers(rv)
	if err := readNonReflect(dec, calledStart, rv.Addr().Interface()); err != errReadMustReflect {
		return err
	}
	// Handle native types, which need the ToNative conversion.  Notice that rv is
	// never a pointer here, so we don't support native pointer types.  In theory
	// we could support native pointer types, but they're complicated and will
	// probably slow everything down.
	//
	// TODO(toddw): Investigate support for native pointer types.
	if ni := nativeInfoFromNative(rv.Type()); ni != nil {
		rvWire := reflect.New(ni.WireType).Elem()
		if err := readReflect(dec, calledStart, rvWire, tt); err != nil {
			return err
		}
		return ni.ToNative(rvWire, rv.Addr())
		// NOTE: readReflect guarantees that FinishValue has already been called.
	}
	tt = tt.NonOptional()
	// Handle scalar wire values.
	if ttReadIntoScalar(tt) {
		if calledStart {
			dec.IgnoreNextStartValue()
		}
		return readValueScalar(dec, rv, tt)
	}
	// Handle bytes wire values.
	if tt.IsBytes() {
		if calledStart {
			dec.IgnoreNextStartValue()
		}
		return readValueBytes(dec, rv, tt)
	}
	// Handle composite wire values.
	if !calledStart {
		if err := dec.StartValue(tt); err != nil {
			return err
		}
	}
	var err error
	switch tt.Kind() {
	case Array:
		err = readFixedLenList(dec, "array", rv.Len(), rv, tt)
	case List:
		err = readList(dec, rv, tt)
	case Set:
		err = readSet(dec, rv, tt)
	case Map:
		err = readMap(dec, rv, tt)
	case Struct:
		err = readStruct(dec, rv, tt)
	case Union:
		err = readUnion(dec, rv, tt)
	default:
		// Note that both Any and Optional were handled in readIntAny, and
		// TypeObject was handled via the readNonReflect special-case.
		return fmt.Errorf("vdl: Read unhandled type %v %v", rv.Type(), tt)
	}
	if err != nil {
		return err
	}
	return dec.FinishValue()
}

// settable exists to avoid a call to reflect.Call() to invoke Set()
// which results in an allocation
type settable interface {
	Set(string) error
}

// readIntoAny uses dec to decode a value into rv, which has VDL type any.
func readIntoAny(dec Decoder, calledStart bool, rv reflect.Value) error {
	if calledStart {
		// The existing code ensures that calledStart is always false here, since
		// readReflect(dec, true, ...) is only called in situations where it's
		// impossible to call readIntoAny.  E.g. it's called later in this function,
		// which never calls it with another any type.  If we did, we'd have a vdl
		// any(any), which isn't allowed.  This error tries to prevent future
		// changes that will break this requirement.
		//
		// The requirement is mandated by vom.RawBytes.VDLRead, which doesn't handle
		// IgnoreNextStartValue.
		return errReadAnyAlreadyStarted
	}
	// Flatten pointers and check for fast non-reflect support, which handles
	// vdl.Value and vom.RawBytes, and any other special-cases.
	rv = rvFlattenPointers(rv)
	if err := readNonReflect(dec, false, rv.Addr().Interface()); err != errReadMustReflect {
		return err
	}
	// The only case left is to handle interfaces.  We allow decoding into
	// all interfaces, including interface{}.
	if rv.Kind() != reflect.Interface {
		return errReadAnyInterfaceOnly
	}
	if err := dec.StartValue(AnyType); err != nil {
		return err
	}
	// Handle decoding any(nil) by setting the rv interface to nil.  Note that the
	// only case where dec.Type() is AnyType is when the value is any(nil).
	if dec.Type() == AnyType {
		if !rv.IsNil() {
			rv.Set(reflect.Zero(rv.Type()))
		}
		return dec.FinishValue()
	}
	// Lookup the reflect type based on the decoder type, and create a new value
	// to decode into.  If the dec value is optional, ensure that we lookup based
	// on an optional type.  Note that if the dec value is nil, dec.Type() is
	// already optional, so rtDecode will already be a pointer.
	ttDecode := dec.Type()
	if dec.IsOptional() && !dec.IsNil() {
		ttDecode = OptionalType(ttDecode)
	}
	rtDecode := typeToReflectNew(ttDecode)
	// Handle top-level "v.io/v23/vdl.WireError" types.  TypeToReflect will find
	// vdl.WireError based on regular wire type registration, and will find the Go
	// error interface based on regular native type registration, and these are
	// fine for nested error types.
	//
	// But this is the case where we're decoding into a top-level Go interface,
	// and we'll lose type information if the dec value is nil.  So instead we
	// return the registered verror.E type.  Examples:
	//
	//   ttDecode  ->  rtDecode
	//   -----------------------
	//   WireError     verror.E
	//   ?WireError    *verror.E
	//   []WireError   []vdl.WireError (1)
	//   []?WireError  []error
	//
	// TODO(toddw): The (1) case above is weird; we would like to return verror.E,
	// but that's hard because the native conversion we've registered doesn't
	// currently include the verror.E type:
	//
	//    ToNative(wire *vdl.WireError, native *error)
	//    FromNative(wire **vdl.WireError, native error)
	//
	// We could make this more consistent by registering a pair of conversion
	// functions instead:
	//
	//    ToNative(wire vdl.WireError, native *verror.E)
	//    FromNative(wire *vdl.WireError, native verror.E)
	//
	//    ToNative(wire *verror.E, native *error)
	//    FromNative(wire **verror.E, native error)
	if ttDecode.NonOptional().Name() == ErrorType.Elem().Name() {
		if ni, err := nativeInfoForError(); err == nil {
			if ttDecode.Kind() == Optional {
				rtDecode = reflect.PtrTo(ni.NativeType)
			} else {
				rtDecode = ni.NativeType
			}
		}
	}
	if rtDecode == nil {
		return fmt.Errorf("vdl: %v not registered, either call vdl.Register, or use vdl.Value or vom.RawBytes instead", dec.Type())
	}
	if !rtDecode.Implements(rv.Type()) {
		return fmt.Errorf("vdl: %v doesn't implement %v", rtDecode, rv.Type())
	}
	// Handle both nil and non-nil values by decoding into rvDecode, and setting
	// rv.  Both nil and non-nil values are handled in the readReflect call.
	rvDecode := reflect.New(rtDecode).Elem()
	if err := readReflect(dec, true, rvDecode, ttDecode); err != nil {
		return err
	}
	rv.Set(rvDecode)
	// NOTE: readReflect guarantees that FinishValue has already been called.
	return nil
}

// readFromNil uses dec to decode a nil value into rv, which has VDL type tt.
// The value in dec might be either any(nil) or optional(nil).
//
// REQUIRES: dec.IsNil() && tt != AnyType
func readFromNil(dec Decoder, rv reflect.Value, tt *Type) error {
	if tt.Kind() != Optional {
		return fmt.Errorf("vdl: can't decode nil into non-optional %v", tt)
	}
	// Flatten pointers until we have a single pointer left, or there were no
	// pointers to begin with.
	rt := rv.Type()
	for rt.Kind() == reflect.Ptr && rt.Elem().Kind() == reflect.Ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rt.Elem()))
		}
		rv, rt = rv.Elem(), rt.Elem()
	}
	// Handle tricky cases where rv is a native type.
	if handled, err := readFromNilNative(dec, rv, tt); handled {
		return err
	}
	// Now handle the simple case: rv has one pointer left, and should be set to a
	// nil pointer.
	if rt.Kind() != reflect.Ptr {
		return fmt.Errorf("vdl: can't decode nil into non-pointer %v optional %v", rt, tt)
	}
	if !rv.IsNil() {
		rv.Set(reflect.Zero(rt))
	}
	return dec.FinishValue()
}

// readFromNilNative handles tricky cases where rv is a native type.  Returns
// true if rv is a native type and was handled, otherwise returns false.
//
// REQUIRES: rv.Type() has at most one pointer.
func readFromNilNative(dec Decoder, rv reflect.Value, tt *Type) (bool, error) {
	var ni *nativeInfo
	if rt := rv.Type(); rt.Kind() != reflect.Ptr {
		// Handle the case where rv isn't a pointer; e.g. the Go error interface is
		// a non-pointer native type, and is handled here.
		ni = nativeInfoFromNative(rt)
	} else {
		// Handle the case where rv is a pointer, and the elem is a native type.
		// E.g. *error is handled here.  Note that we don't support native pointer
		// types; see comments at other calls to nativeInfoFromNative.
		ni = nativeInfoFromNative(rt.Elem())
		if ni != nil {
			if rv.IsNil() {
				rv.Set(reflect.New(rt.Elem()))
			}
			rv = rv.Elem()
		}
	}
	if ni != nil {
		// Handle the native type from either case above.  At this point, rv is the
		// native type and isn't a nil pointer.
		rvWire := reflect.New(ni.WireType).Elem()
		if err := readReflect(dec, true, rvWire, tt); err != nil {
			return true, err
		}
		return true, ni.ToNative(rvWire, rv.Addr())
		// NOTE: readReflect guarantees that FinishValue has already been called.
	}
	return false, nil
}

func ttReadIntoScalar(tt *Type) bool {
	switch tt.Kind() {
	case Bool, String, Enum, Byte, Uint16, Uint32, Uint64, Int8, Int16, Int32, Int64, Float32, Float64:
		return true
	}
	return false
}

func readValueScalar(dec Decoder, rv reflect.Value, tt *Type) error {
	switch kind := tt.Kind(); kind {
	case Bool:
		value, err := dec.ReadValueBool()
		if err != nil {
			return err
		}
		rv.SetBool(value)
		return nil
	case String:
		value, err := dec.ReadValueString()
		if err != nil {
			return err
		}
		rv.SetString(value)
		return nil
	case Enum:
		value, err := dec.ReadValueString()
		if err != nil {
			return err
		}
		return rv.Addr().Interface().(settable).Set(value)
	case Byte, Uint16, Uint32, Uint64:
		value, err := dec.ReadValueUint(kind.BitLen())
		if err != nil {
			return err
		}
		rv.SetUint(value)
		return nil
	case Int8, Int16, Int32, Int64:
		value, err := dec.ReadValueInt(kind.BitLen())
		if err != nil {
			return err
		}
		rv.SetInt(value)
		return nil
	case Float32, Float64:
		value, err := dec.ReadValueFloat(kind.BitLen())
		if err != nil {
			return err
		}
		rv.SetFloat(value)
		return nil
	}
	return fmt.Errorf("vdl: readValueScalar called on non-scalar %v, %v", tt, rv.Type())
}

func readNextEntryScalar(dec Decoder, rv reflect.Value, tt *Type) (bool, error) {
	switch kind := tt.Kind(); kind {
	case Bool:
		switch done, value, err := dec.NextEntryValueBool(); {
		case err != nil:
			return false, err
		case done:
			return true, nil
		default:
			rv.SetBool(value)
			return false, nil
		}
	case String:
		switch done, value, err := dec.NextEntryValueString(); {
		case err != nil:
			return false, err
		case done:
			return true, nil
		default:
			rv.SetString(value)
			return false, nil
		}
	case Enum:
		switch done, value, err := dec.NextEntryValueString(); {
		case err != nil:
			return false, err
		case done:
			return true, nil
		default:
			return false, rv.Addr().Interface().(settable).Set(value)
		}
	case Byte, Uint16, Uint32, Uint64:
		switch done, value, err := dec.NextEntryValueUint(kind.BitLen()); {
		case err != nil:
			return false, err
		case done:
			return true, nil
		default:
			rv.SetUint(value)
			return false, nil
		}
	case Int8, Int16, Int32, Int64:
		switch done, value, err := dec.NextEntryValueInt(kind.BitLen()); {
		case err != nil:
			return false, err
		case done:
			return true, nil
		default:
			rv.SetInt(value)
			return false, nil
		}
	case Float32, Float64:
		switch done, value, err := dec.NextEntryValueFloat(kind.BitLen()); {
		case err != nil:
			return false, err
		case done:
			return true, nil
		default:
			rv.SetFloat(value)
			return false, nil
		}
	}
	return false, fmt.Errorf("vdl: readNextEntryScalar called on non-scalar %v, %v", tt, rv.Type())
}

func readValueBytes(dec Decoder, rv reflect.Value, tt *Type) error {
	kind := tt.Kind()
	// Go doesn't allow type conversions from []MyByte to []byte, but the reflect
	// package does let us perform this conversion.  We slice arrays so that we
	// can fill them in directly.
	fixedLen, needAssign := -1, false
	var fillPtr *[]byte
	switch {
	case kind == Array:
		fixedLen = tt.Len()
		tmp := rv.Slice(0, fixedLen).Bytes()
		fillPtr = &tmp
	case tt.Name() != "" || tt.Elem() != ByteType:
		fillPtr = new([]byte)
		needAssign = true
	default: // rv has type []byte
		fillPtr = rv.Addr().Interface().(*[]byte)
	}
	if err := dec.ReadValueBytes(fixedLen, fillPtr); err != nil {
		return err
	}
	if needAssign {
		rv.SetBytes(*fillPtr)
	}
	return nil
}

func readFixedLenList(dec Decoder, name string, len int, rv reflect.Value, tt *Type) error {
	ttElem := tt.Elem()
	if ttReadIntoScalar(ttElem) {
		// Handle scalar element fastpath.
		for index := 0; index < len; index++ {
			rvIndex := rv.Index(index)
			switch done, err := readNextEntryScalar(dec, rvIndex, ttElem); {
			case err != nil:
				return err
			case done:
				return fmt.Errorf("vdl: short %s, got len %d < %d %v", name, index, len, rv.Type())
			}
		}
	} else {
		// Handle non-scalar elements.
		for index := 0; index < len; index++ {
			switch done, err := dec.NextEntry(); {
			case err != nil:
				return err
			case done:
				return fmt.Errorf("vdl: short %s, got len %d < %d %v", name, index, len, rv.Type())
			}
			if err := readReflect(dec, false, rv.Index(index), ttElem); err != nil {
				return err
			}
		}
	}
	switch done, err := dec.NextEntry(); {
	case err != nil:
		return err
	case !done:
		return fmt.Errorf("vdl: long %s, got len > %d %v", name, len, rv.Type())
	}
	return nil
}

func readList(dec Decoder, rv reflect.Value, tt *Type) error {
	rt := rv.Type()
	rtElem, ttElem := rt.Elem(), tt.Elem()
	if len := dec.LenHint(); len > 0 {
		// Handle fixed-length fastpath.
		rv.Set(reflect.MakeSlice(rt, len, len))
		return readFixedLenList(dec, "list", len, rv, tt)
	}
	// TODO(toddw): Make progressively larger slices, rather than creating each
	// element one at a time.
	rv.Set(reflect.Zero(rt))
	if ttReadIntoScalar(ttElem) {
		// Handle scalar element fastpath.
		for {
			rvElem := reflect.New(rtElem).Elem()
			switch done, err := readNextEntryScalar(dec, rvElem, ttElem); {
			case err != nil:
				return err
			case done:
				return nil
			}
			rv.Set(reflect.Append(rv, rvElem))
		}
	}
	// Handle non-scalar elements.
	for {
		switch done, err := dec.NextEntry(); {
		case err != nil:
			return err
		case done:
			return nil
		}
		rvElem := reflect.New(rtElem).Elem()
		if err := readReflect(dec, false, rvElem, ttElem); err != nil {
			return err
		}
		rv.Set(reflect.Append(rv, rvElem))
	}
}

var rvEmptyStruct = reflect.ValueOf(struct{}{})

func readSet(dec Decoder, rv reflect.Value, tt *Type) error {
	rt := rv.Type()
	rtKey, ttKey := rt.Key(), tt.Key()
	tmpSet, isNil := reflect.Zero(rt), true
	if ttReadIntoScalar(ttKey) {
		// Handle scalar key fastpath.
		for {
			rvKey := reflect.New(rtKey).Elem()
			switch done, err := readNextEntryScalar(dec, rvKey, ttKey); {
			case err != nil:
				return err
			case done:
				rv.Set(tmpSet)
				return nil
			}
			if isNil {
				tmpSet, isNil = reflect.MakeMap(rt), false
			}
			tmpSet.SetMapIndex(rvKey, rvEmptyStruct)
		}
	}
	// Handle non-scalar elements.
	for {
		switch done, err := dec.NextEntry(); {
		case err != nil:
			return err
		case done:
			rv.Set(tmpSet)
			return nil
		}
		rvKey := reflect.New(rtKey).Elem()
		if err := readReflect(dec, false, rvKey, ttKey); err != nil {
			return err
		}
		if isNil {
			tmpSet, isNil = reflect.MakeMap(rt), false
		}
		tmpSet.SetMapIndex(rvKey, rvEmptyStruct)
	}
}

func readMap(dec Decoder, rv reflect.Value, tt *Type) error {
	rt := rv.Type()
	rtKey, ttKey := rt.Key(), tt.Key()
	rtElem, ttElem := rt.Elem(), tt.Elem()
	tmpMap, isNil := reflect.Zero(rt), true
	if ttReadIntoScalar(ttKey) {
		// Handle scalar key fastpath.
		for {
			rvKey := reflect.New(rtKey).Elem()
			switch done, err := readNextEntryScalar(dec, rvKey, ttKey); {
			case err != nil:
				return err
			case done:
				rv.Set(tmpMap)
				return nil
			}
			rvElem := reflect.New(rtElem).Elem()
			if err := readReflect(dec, false, rvElem, ttElem); err != nil {
				return err
			}
			if isNil {
				tmpMap, isNil = reflect.MakeMap(rt), false
			}
			tmpMap.SetMapIndex(rvKey, rvElem)
		}
	}
	// Handle non-scalar keys.
	for {
		switch done, err := dec.NextEntry(); {
		case err != nil:
			return err
		case done:
			rv.Set(tmpMap)
			return nil
		}
		rvKey := reflect.New(rtKey).Elem()
		if err := readReflect(dec, false, rvKey, ttKey); err != nil {
			return err
		}
		rvElem := reflect.New(rtElem).Elem()
		if err := readReflect(dec, false, rvElem, ttElem); err != nil {
			return err
		}
		if isNil {
			tmpMap, isNil = reflect.MakeMap(rt), false
		}
		tmpMap.SetMapIndex(rvKey, rvElem)
	}
}

func readStruct(dec Decoder, rv reflect.Value, tt *Type) error {
	rt, decType := rv.Type(), dec.Type()
	// Reset to the zero struct, since fields may be missing.
	//
	// TODO(toddw): Avoid repeated zero-setting of nested structs.
	rvZero, err := rvZeroValue(rt, tt)
	if err != nil {
		return err
	}
	rv.Set(rvZero)
	for {
		index, err := dec.NextField()
		switch {
		case err != nil:
			return err
		case index == -1:
			return nil
		}
		var ttField Field
		if decType == tt {
			ttField = tt.Field(index)
		} else {
			ttField, index = tt.FieldByName(decType.Field(index).Name)
			if index == -1 {
				if err := dec.SkipValue(); err != nil {
					return err
				}
				continue
			}
		}
		rvField := rv.Field(rtFieldIndexByName(rt, ttField.Name))
		if ttReadIntoScalar(ttField.Type) {
			if err := readValueScalar(dec, rvField, ttField.Type); err != nil {
				return err
			}
		} else {
			if err := readReflect(dec, false, rvField, ttField.Type); err != nil {
				return err
			}
		}
	}
}

func readUnion(dec Decoder, rv reflect.Value, tt *Type) error {
	rt, decType := rv.Type(), dec.Type()
	index, err := dec.NextField()
	switch {
	case err != nil:
		return err
	case index == -1:
		return fmt.Errorf("missing field in union %v, from %v", rt, decType)
	}
	var ttField Field
	if decType == tt {
		ttField = tt.Field(index)
	} else {
		name := decType.Field(index).Name
		ttField, index = tt.FieldByName(name)
		if index == -1 {
			return fmt.Errorf("field %q not in union %v, from %v", name, rt, decType)

		}
	}
	// We have a union interface.  Create a new field based on its rep type, fill
	// in its value, and assign the field to the interface.
	ri, _, err := deriveReflectInfo(rt)
	if err != nil {
		return err
	}
	rvField := reflect.New(ri.UnionFields[index].RepType).Elem()
	if ttReadIntoScalar(ttField.Type) {
		if err := readValueScalar(dec, rvField.Field(0), ttField.Type); err != nil {
			return err
		}
	} else {
		if err := readReflect(dec, false, rvField.Field(0), ttField.Type); err != nil {
			return err
		}
	}
	rv.Set(rvField)
	switch index, err := dec.NextField(); {
	case err != nil:
		return err
	case index != -1:
		return fmt.Errorf("extra field %d in union %v, from %v", index, rt, decType)
	}
	return nil
}
