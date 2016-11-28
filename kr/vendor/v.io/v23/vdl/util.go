// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

const (
	// IEEE 754 represents float64 using 52 bits to represent the mantissa, with
	// an extra implied leading bit.  That gives us 53 bits to store integers
	// without overflow - i.e. [0, (2^53)-1].  And since 2^53 is a small power of
	// two, it can also be stored without loss via mantissa=1 exponent=53.  Thus
	// we have our max and min values.  Ditto for float32, which uses 23 bits with
	// an extra implied leading bit.
	float64MaxInt = (1 << 53)
	float64MinInt = -(1 << 53)
	float32MaxInt = (1 << 24)
	float32MinInt = -(1 << 24)
)

var (
	bitlenReflect = [...]uintptr{
		reflect.Uint8:   8,
		reflect.Uint16:  16,
		reflect.Uint32:  32,
		reflect.Uint64:  64,
		reflect.Uint:    8 * unsafe.Sizeof(uint(0)),
		reflect.Uintptr: 8 * unsafe.Sizeof(uintptr(0)),
		reflect.Int8:    8,
		reflect.Int16:   16,
		reflect.Int32:   32,
		reflect.Int64:   64,
		reflect.Int:     8 * unsafe.Sizeof(int(0)),
		reflect.Float32: 32,
		reflect.Float64: 64,
	}

	bitlenVDL = [...]uintptr{
		Byte:    8,
		Uint16:  16,
		Uint32:  32,
		Uint64:  64,
		Int8:    8,
		Int16:   16,
		Int32:   32,
		Int64:   64,
		Float32: 32,
		Float64: 64,
	}
)

// bitlen{R,V} enforce static type safety on kind.
func bitlenR(kind reflect.Kind) uintptr { return bitlenReflect[kind] }
func bitlenV(kind Kind) uintptr         { return bitlenVDL[kind] }

// isRTBytes returns true iff rt is an array or slice of bytes.
func isRTBytes(rt reflect.Type) bool {
	return (rt.Kind() == reflect.Array || rt.Kind() == reflect.Slice) && rt.Elem().Kind() == reflect.Uint8
}

// rtBytes extracts []byte from rv.  Assumes isRTBytes(rv.Type()) == true.
func rtBytes(rv reflect.Value) []byte {
	// Fastpath if the underlying type is []byte
	if rv.Kind() == reflect.Slice && rv.Type().Elem() == rtByte {
		return rv.Bytes()
	}
	// Slowpath copying bytes one by one.
	ret := make([]byte, rv.Len())
	for ix := 0; ix < rv.Len(); ix++ {
		ret[ix] = rv.Index(ix).Convert(rtByte).Interface().(byte)
	}
	return ret
}

// IsZeroer is the interface that wraps the VDLIsZero method.
//
// VDLIsZero returns true iff the receiver that implements this method is the
// VDL zero value.
type IsZeroer interface {
	VDLIsZero() bool
}

type stringer interface {
	String() string
}
type namer interface {
	Name() string
}
type indexer interface {
	Index() int
}

// rvFlattenPointers repeatedly dereferences pointers, creating new values if
// the pointer is nil, and returns the final non-pointer reflect value.  As a
// special-case, *Type is returned as a pointer.
func rvFlattenPointers(rv reflect.Value) reflect.Value {
	for rv.Kind() == reflect.Ptr {
		if rv.Type() == rtPtrToType {
			// Special-case to stop at *Type, which is filled in via readNonReflect by
			// the reader, or by rvZeroValue.
			return rv
		}
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}
	return rv
}

// zeroDecoder is a decoder that only returns zero values.
type zeroDecoder struct{ tt *Type }

func (z zeroDecoder) StartValue(want *Type) error {
	if !Compatible(z.tt, want) {
		return fmt.Errorf("vdl: zero incompatible decode from %v into %v", z.tt, want)
	}
	return nil
}
func (z zeroDecoder) FinishValue() error       { return nil }
func (z zeroDecoder) SkipValue() error         { return nil }
func (z zeroDecoder) IgnoreNextStartValue()    {}
func (z zeroDecoder) NextEntry() (bool, error) { return true, nil }
func (z zeroDecoder) NextField() (int, error)  { return -1, nil }
func (z zeroDecoder) Type() *Type              { return z.tt }
func (z zeroDecoder) IsAny() bool              { return z.tt == AnyType }
func (z zeroDecoder) IsOptional() bool         { return z.tt.Kind() == Optional }
func (z zeroDecoder) IsNil() bool              { return z.IsAny() || z.IsOptional() }
func (z zeroDecoder) Index() int               { return 0 }
func (z zeroDecoder) LenHint() int             { return 0 }

func (z zeroDecoder) DecodeBool() (bool, error)        { return false, nil }
func (z zeroDecoder) DecodeString() (string, error)    { return "", nil }
func (z zeroDecoder) DecodeUint(int) (uint64, error)   { return 0, nil }
func (z zeroDecoder) DecodeInt(int) (int64, error)     { return 0, nil }
func (z zeroDecoder) DecodeFloat(int) (float64, error) { return 0, nil }
func (z zeroDecoder) DecodeTypeObject() (*Type, error) { return AnyType, nil }
func (z zeroDecoder) DecodeBytes(fixedLen int, x *[]byte) error {
	if fixedLen >= 0 {
		for i := 0; i < fixedLen; i++ {
			(*x)[i] = 0
		}
	} else {
		*x = nil
	}
	return nil
}

func (z zeroDecoder) ReadValueBool() (bool, error)               { return false, nil }
func (z zeroDecoder) ReadValueString() (string, error)           { return "", nil }
func (z zeroDecoder) ReadValueUint(bitlen int) (uint64, error)   { return 0, nil }
func (z zeroDecoder) ReadValueInt(bitlen int) (int64, error)     { return 0, nil }
func (z zeroDecoder) ReadValueFloat(bitlen int) (float64, error) { return 0, nil }
func (z zeroDecoder) ReadValueTypeObject() (*Type, error)        { return AnyType, nil }
func (z zeroDecoder) ReadValueBytes(fixedLen int, x *[]byte) error {
	return z.DecodeBytes(fixedLen, x)
}

func (z zeroDecoder) NextEntryValueBool() (bool, bool, error)               { return true, false, nil }
func (z zeroDecoder) NextEntryValueString() (bool, string, error)           { return true, "", nil }
func (z zeroDecoder) NextEntryValueUint(bitlen int) (bool, uint64, error)   { return true, 0, nil }
func (z zeroDecoder) NextEntryValueInt(bitlen int) (bool, int64, error)     { return true, 0, nil }
func (z zeroDecoder) NextEntryValueFloat(bitlen int) (bool, float64, error) { return true, 0, nil }
func (z zeroDecoder) NextEntryValueTypeObject() (bool, *Type, error)        { return true, nil, nil }

var (
	rvAnyType               = reflect.ValueOf(AnyType)
	kkZeroValueNotCanonical = []Kind{Any, TypeObject, Union}
	kkZeroValueNotUnique    = []Kind{Any, TypeObject, Union, List, Set, Map}
)

// rvZeroValue returns the zero value of rt, using the vdl zero rules.
//
// VDL and Go define zero values differently.  According to VDL:
//    Any:        nil
//    TypeObject: AnyType
//    Union:      zero value of the type at index 0
// The Go zero value isn't always right.  Here are the Go zero values:
//    Any:        interface{}(nil), *vom.RawBytes(nil) or *vdl.Value(nil)
//    TypeObject: (*Type)(nil)
//    Union:      UnionInterface(nil)
// Here are the Go values we actually want:
//    Any:        *vom.RawBytes or *vdl.Value representing any(nil)
//    TypeObject: AnyType
//    Union:      UnionStruct0
//
// Thus we must special-case values of these types, or any types that contain
// these types inline.  I.e. if an array, struct, or union contains one of these
// types, it will show up in the zero value, and needs special-casing.
//
// TODO(toddw): Cache the generated zero values, if it's too expensive to
// generate them each time.
func rvZeroValue(rt reflect.Type, tt *Type) (reflect.Value, error) {
	// Easy fastpath; if the type doesn't contain the hard types inline, the
	// regular Go zero value is sufficient.
	if !tt.ContainsKind(WalkInline, kkZeroValueNotCanonical...) {
		return reflect.Zero(rt), nil
	}
	// Create the result we'll return.
	result := reflect.New(rt).Elem()
	// Flatten pointers and check for fast non-reflect support.  We re-use the
	// readNonReflect logic with a decoder that only produces zero values.  This
	// handles vom.RawBytes/vdl.Value, as well as TypeObject.
	rv := rvFlattenPointers(result)
	rt = rv.Type()
	if err := readNonReflect(zeroDecoder{tt}, false, rv.Addr().Interface()); err != errReadMustReflect {
		return result, err
	}
	// The only representation left for Any types is nil interfaces
	if tt == AnyType && rt.Kind() == reflect.Interface {
		return result, nil
	}
	// Handle native types by returning the native value filled in with a zero
	// value of the wire type.
	if ni := nativeInfoFromNative(rt); ni != nil {
		rvWire := reflect.New(ni.WireType).Elem()
		ttWire, err := TypeFromReflect(ni.WireType)
		if err != nil {
			return reflect.Value{}, err
		}
		switch zero, err := rvZeroValue(ni.WireType, ttWire); {
		case err != nil:
			return reflect.Value{}, err
		default:
			rvWire.Set(zero)
		}
		if err := ni.ToNative(rvWire, rv.Addr()); err != nil {
			return reflect.Value{}, err
		}
		return result, nil
	}
	// Handle composite types with inline subtypes.
	switch {
	case tt.Kind() == Union && rt.Kind() == reflect.Interface:
		// Set the union interface with the zero value of the type at index 0.
		ri, _, err := deriveReflectInfo(rt)
		if err != nil {
			return reflect.Value{}, err
		}
		rvFieldStruct := reflect.New(ri.UnionFields[0].RepType).Elem()
		switch zero, err := rvZeroValue(rvFieldStruct.Field(0).Type(), tt.Field(0).Type); {
		case err != nil:
			return reflect.Value{}, err
		default:
			rvFieldStruct.Field(0).Set(zero)
			rv.Set(rvFieldStruct)
		}
	case tt.Kind() == Array && rt.Kind() == reflect.Array:
		for ix := 0; ix < rt.Len(); ix++ {
			switch zero, err := rvZeroValue(rt.Elem(), tt.Elem()); {
			case err != nil:
				return reflect.Value{}, err
			default:
				rv.Index(ix).Set(zero)
			}
		}
	case tt.Kind() == Struct && rt.Kind() == reflect.Struct:
		for ix := 0; ix < tt.NumField(); ix++ {
			field := tt.Field(ix)
			rvField := rv.Field(rtFieldIndexByName(rt, field.Name))
			switch zero, err := rvZeroValue(rvField.Type(), field.Type); {
			case err != nil:
				return reflect.Value{}, err
			default:
				rvField.Set(zero)
			}
		}
	default:
		return reflect.Value{}, fmt.Errorf("vdl: rvZeroValue unhandled rt: %v tt: %v", rt, tt)
	}
	return result, nil
}

// rvIsZeroValue returns true iff rv represents the VDL zero value.  Here are
// the types with multiple VDL zero value representations:
//   Any:            nil, or VDLIsZero on vdl.Value/vom.RawBytes
//   TypeObject:     nil, or AnyType
//   Union:          nil, or zero value of field 0
//   List, Set, Map: nil, or empty
func rvIsZeroValue(rv reflect.Value, tt *Type) (bool, error) {
	// Walk pointers and interfaces, and handle nil values.
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			// All nil pointers and nil interfaces are considered to be zero.  Note
			// that we may have a non-optional type that happens to be represented by
			// a pointer; technically nil might be considered an error, but it's
			// easier for the user (and for us) to treat it as zero.
			return true, nil
		}
		rv = rv.Elem()
	}
	// Optional types can only be zero via a nil pointer or interface.
	if tt.Kind() == Optional {
		return false, nil
	}
	rt := rv.Type()
	// Now we know that rv isn't a pointer or interface, and also isn't nil.  Call
	// VDLIsZero if it exists.  This handles the vdl.Value/vom.RawBytes cases, as
	// well generated code and user-implemented VDLIsZero methods.
	if rt.Implements(rtIsZeroer) {
		return rv.Interface().(IsZeroer).VDLIsZero(), nil
	}
	if reflect.PtrTo(rt).Implements(rtIsZeroer) {
		if rv.CanAddr() {
			return rv.Addr().Interface().(IsZeroer).VDLIsZero(), nil
		}
		// Handle the harder case where *T implements IsZeroer, but we can't take
		// the address of rv to turn it into *T.  Create a new *T value and fill it
		// in with rv, so that we can call VDLIsZero.  This is conceptually similar
		// to storing rv in a temporary variable, so that we can take the address.
		rvPtr := reflect.New(rt)
		rvPtr.Elem().Set(rv)
		return rvPtr.Interface().(IsZeroer).VDLIsZero(), nil
	}
	// Handle native types, by converting and checking the wire value for zero.
	if ni := nativeInfoFromNative(rt); ni != nil {
		rvWirePtr := reflect.New(ni.WireType)
		if err := ni.FromNative(rvWirePtr, rv); err != nil {
			return false, err
		}
		return rvIsZeroValue(rvWirePtr.Elem(), tt)
	}
	// The interface form of any was handled above in the nil checks, while the
	// non-interface forms were handled via VDLIsZero.
	if tt.Kind() == Optional || tt.Kind() == Any {
		return false, nil
	}
	// TODO(toddw): We could consider adding a "fastpath" here to check against
	// the go zero value, or the zero value created by rvZeroValue, and possibly
	// returning early.  This is tricky though; we can't use this fastpath if rt
	// contains any native types, but the only way to know whether rt contains any
	// native types is to look through the entire type, which might actually be
	// slower than the benefit of this "fastpath".  The cases where it'll help are
	// large arrays or structs.
	//
	// Handle all reflect cases.
	switch rv.Kind() {
	case reflect.Bool:
		return !rv.Bool(), nil
	case reflect.String:
		return rv.String() == "", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rv.Uint() == 0, nil
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0, nil
	case reflect.Complex64, reflect.Complex128:
		return rv.Complex() == 0, nil
	case reflect.UnsafePointer:
		return rv.Pointer() == 0, nil
	case reflect.Slice, reflect.Map:
		return rv.Len() == 0, nil
	case reflect.Array:
		for ix := 0; ix < rv.Len(); ix++ {
			if z, err := rvIsZeroValue(rv.Index(ix), tt.Elem()); err != nil || !z {
				return false, err
			}
		}
		return true, nil
	case reflect.Struct:
		switch tt.Kind() {
		case Struct:
			for ix := 0; ix < tt.NumField(); ix++ {
				ttField := tt.Field(ix)
				rvField := rv.Field(rtFieldIndexByName(rt, ttField.Name))
				if z, err := rvIsZeroValue(rvField, ttField.Type); err != nil || !z {
					return false, err
				}
			}
			return true, nil
		case Union:
			// We already handled the nil union interface case above in the regular
			// pointer/interface walking.  Here we check to make sure the union field
			// struct represents field 0, and is set to its zero value.
			//
			// TypeFromReflect already validated Index(); call without error checking.
			if index := rv.Interface().(indexer).Index(); index != 0 {
				return false, nil
			}
			return rvIsZeroValue(rv.Field(0), tt.Field(0).Type)
		}
	}
	return false, fmt.Errorf("vdl: rvIsZeroValue unhandled rt: %v tt: %v", rt, tt)
}

// rtFieldIndexByName returns the index of the struct field in rt with the given
// name.  Returns -1 if the field doesn't exist.
//
// This function is purely a performance optimization; the current
// implementation of reflect.Type.Field(index) causes an allocation, which is
// avoided in the common case by caching the result.
//
// REQUIRES: rt.Kind() == reflect.Struct
func rtFieldIndexByName(rt reflect.Type, name string) int {
	rtFieldCache.RLock()
	m, ok := rtFieldCache.Map[rt]
	rtFieldCache.RUnlock()
	// Fastpath cache hit.
	if ok {
		return m[name] - 1
	}
	// Slowpath cache miss, populate the cache.
	rtFieldCache.Lock()
	defer rtFieldCache.Unlock()
	// Handle benign race, where the cache was filled in while we upgraded from a
	// reader lock to an exclusive lock.
	if m, ok := rtFieldCache.Map[rt]; ok {
		return m[name] - 1
	}
	if numField := rt.NumField(); numField > 0 {
		m = make(map[string]int, numField)
		for i := 0; i < numField; i++ {
			m[rt.Field(i).Name] = i + 1
		}
	}
	rtFieldCache.Map[rt] = m
	return m[name] - 1
}

var rtFieldCache = &rtFieldCacheType{
	Map: make(map[reflect.Type]map[string]int),
}

type rtFieldCacheType struct {
	sync.RWMutex
	Map map[reflect.Type]map[string]int
}
