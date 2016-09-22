// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

// rtCacheT is a map from reflect.Type to *Type.  The only instance is rtCache,
// which is a global cache to speed up repeated lookups.
//
// All locking is performed in TypeFromReflect.
type rtCacheT struct {
	sync.RWMutex
	rtmap map[reflect.Type]*Type
}

var (
	rtCache = &rtCacheT{
		rtmap: map[reflect.Type]*Type{
			// Ensure TypeOf(WireError{}) returns the built-in VDL error type.
			reflect.TypeOf(WireError{}): ErrorType.Elem(),
		},
	}
	rtCacheEnabled = true
)

func (reg *rtCacheT) lookup(rt reflect.Type) *Type {
	if !rtCacheEnabled {
		return nil
	}
	return reg.rtmap[rt]
}

func (reg *rtCacheT) update(pending map[reflect.Type]TypeOrPending) {
	if !rtCacheEnabled {
		return
	}
	for rt, top := range pending {
		t, _ := getBuiltType(top)
		reg.rtmap[rt] = t
	}
}

// getBuiltType returns the built type from top.
func getBuiltType(top TypeOrPending) (*Type, error) {
	switch ttop := top.(type) {
	case *Type:
		return ttop, nil
	case PendingType:
		return ttop.Built()
	}
	panic(fmt.Errorf("vdl: unknown type in TypeOrPending: %T %v", top, top))
}

// TypeOf returns the type corresponding to v.  It's a helper for calling
// TypeFromReflect, and panics on any errors.
func TypeOf(v interface{}) *Type {
	t, err := TypeFromReflect(reflect.TypeOf(v))
	if err != nil {
		panic(fmt.Errorf("vdl: can't take TypeOf(%T): %v", v, err))
	}
	return t
}

// Normalize the rt type.  The VDL type system Optional is represented as a Go
// pointer.  Only structs may be optional in VDL, but we still allow the pointer
// forms of other types in Go.  E.g. VDL doesn't allow ?map, but we do allow Go
// **map, and consider that to be a VDL map; the pointers are flattened away.
//
// In addition all Go interfaces are represented with the single Any type,
// except for Go interfaces that describe Union types.
//
// By normalizing the rt type we simplify type creation, and also reduce
// redundancy in the rtCache.
func normalizeType(rt reflect.Type) reflect.Type {
	// Flatten rt to no pointers, and rtAtMostOnePtr to at most one pointer.
	hasPtr := false
	for rt.Kind() == reflect.Ptr {
		if ni := nativeInfoFromNative(rt); ni != nil {
			if hasPtr {
				return normalizeType(reflect.PtrTo(ni.WireType))
			}
			return normalizeType(ni.WireType)
		}
		hasPtr = true
		rt = rt.Elem()
	}
	if ni := nativeInfoFromNative(rt); ni != nil {
		if hasPtr {
			return normalizeType(reflect.PtrTo(ni.WireType))
		}
		return normalizeType(ni.WireType)
	}
	rtAtMostOnePtr := rt
	if hasPtr {
		rtAtMostOnePtr = reflect.PtrTo(rt)
	}
	// Handle errors that are implemented by arbitrary rv values.  E.g. the Go
	// standard errors.errorString implements the error interface, but is an
	// invalid vdl type since it doesn't have any exported fields.
	//
	// See corresponding special-case in reflect_writer.go
	if rt.Implements(rtError) || rtAtMostOnePtr.Implements(rtError) {
		return rtError
	}
	// Handle special cases.  Union may be either an interface or a struct, and
	// should be handled first.
	if ri, _, _ := deriveReflectInfo(rt); ri != nil {
		if len(ri.UnionFields) > 0 {
			return rt
		}
		// Use the type defined in the reflect info.  Typically this is the same as
		// the original rt, but in some cases we use __VDLReflect to override the
		// vdl type.  E.g. vom.RawBytes claims that it is AnyType.
		rt = ri.Type
	}
	switch {
	case rt.Kind() == reflect.Interface:
		// Collapse all interfaces to interface{}
		return rtInterface
	case rt.Kind() == reflect.Struct && rt.PkgPath() != "":
		// Named structs may be optional, so we keep the pointer.
		return rtAtMostOnePtr
	}
	return rt
}

// basicType returns the *Type corresponding to rt for basic types that cannot
// be named by the user, and have a well-known conversion.
func basicType(rt reflect.Type) *Type {
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	switch rt {
	case rtError:
		// TODO(bprosnitz) Remove this for new vdl error logic
		return ErrorType
	case rtInterface, rtValue:
		return AnyType
	case rtType:
		return TypeObjectType
	}
	return nil
}

// TypeFromReflect returns the type corresponding to rt.  Not all reflect types
// have a valid type; reflect.Chan, reflect.Func and reflect.UnsafePointer are
// unsupported, as are maps with pointer keys, as well as structs with only
// unexported fields.
func TypeFromReflect(rt reflect.Type) (*Type, error) {
	if rt == nil {
		// Special case to avoid panic for nil rt.
		return AnyType, nil
	}
	// Fastpath - grab the reader lock and check if rt is already in the cache.
	if t := basicType(rt); t != nil {
		return t, nil
	}
	rtCache.RLock()
	t := rtCache.lookup(rt)
	rtCache.RUnlock()
	if t != nil {
		return t, nil
	}
	// Slowpath - first register rt and all subtypes, to ensure they're known.
	// This avoids some of the tricky ordering issues between vdl.Register and
	// vdl.TypeFromReflect, when they are called in init functions.
	if err := registerRecursive(rt); err != nil {
		return nil, err
	}
	// Slowpath - grab the writer lock.  We hold the lock even while building the
	// type, since TypeBuilder requires that if two types are identical, they must
	// be represented by the same Type or PendingType.  Here's an example:
	//   type Str string
	//   type Foo struct { A, B Str }
	//
	// If we built type Foo without the lock, we might end up with this ordering:
	//    thread_1 build Foo
	//    thread_1 build Foo.A
	//
	//    thread_2 build Foo
	//    thread_2 build Foo.A
	//    thread_2 build Foo.B
	//    thread_2 update map
	//
	//    thread_1 build Foo.B // type Str found in map, different from Foo.A
	//
	// The problem is that thread_1 now has two different representations of Str
	// in its TypeBuilder - it built type Str itself for Foo.A, and it found Str
	// from the map for Foo.B.  The TypeBuilder notices this inconsistency, and
	// returns an error.
	//
	// Side note: you might think there's a simpler fix; if each thread updated
	// the map (under the lock) as it built each type, we wouldn't need to lock
	// the entire type-building operation.  But note that TypeBuilder only returns
	// PendingType as types are being built, and only returns the final Type when
	// Build is called at the very end, in order to support cyclic types.
	rtCache.Lock()
	defer rtCache.Unlock()
	// The strategy is to recursively populate the builder with the type and
	// subtypes, keeping track of new types in pending.  After all types have been
	// populated, we build the types and update rtCache with all pending types.
	builder := new(TypeBuilder)
	pending := make(map[reflect.Type]TypeOrPending)
	result, err := typeFromReflectLocked(rt, builder, pending)
	if err != nil {
		return nil, err
	}
	builder.Build()
	built, err := getBuiltType(result)
	if built == nil {
		// There must have been an error building one of the types.  Return the
		// first error we find, favoring errors from building the result itself.
		if err != nil {
			return nil, err
		}
		for _, top := range pending {
			_, err := getBuiltType(top)
			if err != nil {
				return nil, err
			}
		}
		panic(fmt.Errorf("vdl: inconsistent results from TypeBuilder.Build: %v", pending))
	}
	rtCache.update(pending)
	return built, nil
}

// typeFromReflectLocked returns the Type or PendingType corresponding to rt.
// It either returns the type directly from the cache or pending map, or makes
// the type based on rt.
//
// REQUIRES: rtCache is locked
func typeFromReflectLocked(rt reflect.Type, builder *TypeBuilder, pending map[reflect.Type]TypeOrPending) (TypeOrPending, error) {
	rtOrig := rt
	rt = normalizeType(rt)
	if t := basicType(rt); t != nil {
		pending[rtOrig] = t
		return t, nil
	}
	if t := rtCache.lookup(rt); t != nil {
		pending[rtOrig] = t
		return t, nil
	}
	if p, ok := pending[rt]; ok {
		// If the type is already in our pending map, we return it now.  This breaks
		// infinite loops from recursive types.
		return p, nil
	}
	return makeTypeFromReflectLocked(rtOrig, rt, builder, pending)
}

// validateType returns a non-nil error if rt is not a valid vdl type.
func validateType(rt reflect.Type) error {
	// Flatten pointers to simplify the checks.
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	// Now check error conditions.
	if rt == rtReflectValue {
		return errTypeFromReflectValue
	}
	switch rt.Kind() {
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return fmt.Errorf("reflect type %q not supported", rt)
	}
	return nil
}

// makeTypeFromReflectLocked makes the Type or PendingType corresponding to rt.
// Calls typeFromReflect to recursively generate subtypes.
//
// REQUIRES: rtCache is locked
// PRE-CONDITION:  rt doesn't exist in rtCache or pending.
// POST-CONDITION: rt exists in pending.
func makeTypeFromReflectLocked(rtOrig, rt reflect.Type, builder *TypeBuilder, pending map[reflect.Type]TypeOrPending) (TypeOrPending, error) {
	if err := validateType(rt); err != nil {
		return nil, err
	}
	if rt.Kind() == reflect.Ptr {
		// Pointers are turned into Optional.
		opt := builder.Optional()
		pending[rt] = opt
		pending[rtOrig] = opt
		elem, err := typeFromReflectLocked(rt.Elem(), builder, pending)
		if err != nil {
			return nil, err
		}
		opt.AssignElem(elem)
		return opt, nil
	}
	ri, _, err := deriveReflectInfo(rt)
	if err != nil {
		return nil, err
	}
	if ri.Name == "" {
		// Unnamed types are made directly.  There's no way to create a recursive
		// type based solely on unnamed types, so it's ok to to update pending
		// *after* making the unnamed type.
		unnamed, err := makeUnnamedFromReflectLocked(ri, builder, pending)
		if err != nil {
			return nil, err
		}
		pending[ri.Type] = unnamed
		pending[rtOrig] = unnamed
		return unnamed, nil
	}
	// Named types are trickier, since they may be recursive.  First create the
	// named type and add it to pending.  We must special-case union types; the
	// interface type and all field types are keyed in the pending map to point to
	// the created vdl type.
	named := builder.Named(ri.Name)
	pending[ri.Type] = named
	pending[rtOrig] = named
	for _, unionField := range ri.UnionFields {
		pending[unionField.RepType] = named
	}
	// Now make the unnamed underlying type.  Recursive types will find the
	// existing entry in pending, and avoid the infinite loop.
	unnamed, err := makeUnnamedFromReflectLocked(ri, builder, pending)
	if err != nil {
		return nil, err
	}
	// Finally assign the base type and we're done.
	named.AssignBase(unnamed)
	return named, nil
}

// makeUnnamedFromReflectLocked makes the underlying unnamed Type or PendingType
// corresponding to ri.
//
// REQUIRES: rtCache is locked
func makeUnnamedFromReflectLocked(ri *reflectInfo, builder *TypeBuilder, pending map[reflect.Type]TypeOrPending) (TypeOrPending, error) {
	// Handle enum types
	if len(ri.EnumLabels) > 0 {
		enum := builder.Enum()
		for _, label := range ri.EnumLabels {
			enum.AppendLabel(label)
		}
		return enum, nil
	}
	// Handle union types
	if len(ri.UnionFields) > 0 {
		union := builder.Union()
		for _, f := range ri.UnionFields {
			in, err := typeFromReflectLocked(f.Type, builder, pending)
			if err != nil {
				return nil, err
			}
			union.AppendField(f.Name, in)
		}
		return union, nil
	}
	// Handle composite types
	rt := ri.Type
	switch rt.Kind() {
	case reflect.Array:
		elem, err := typeFromReflectLocked(rt.Elem(), builder, pending)
		if err != nil {
			return nil, err
		}
		return builder.Array().AssignLen(rt.Len()).AssignElem(elem), nil
	case reflect.Slice:
		elem, err := typeFromReflectLocked(rt.Elem(), builder, pending)
		if err != nil {
			return nil, err
		}
		return builder.List().AssignElem(elem), nil
	case reflect.Map:
		if rt.Key().Kind() == reflect.Ptr {
			return nil, fmt.Errorf("invalid key %q in %q", rt.Key(), rt)
		}
		key, err := typeFromReflectLocked(rt.Key(), builder, pending)
		if err != nil {
			return nil, err
		}
		if rt.Elem() == rtUnnamedEmptyStruct {
			// The map actually represents a set
			return builder.Set().AssignKey(key), nil
		}
		elem, err := typeFromReflectLocked(rt.Elem(), builder, pending)
		if err != nil {
			return nil, err
		}
		return builder.Map().AssignKey(key).AssignElem(elem), nil
	case reflect.Struct:
		st := builder.Struct()
		for fx := 0; fx < rt.NumField(); fx++ {
			// TODO(toddw): How should we handle anonymous (aka embedded) fields?
			// Note that unexported embedded fields may themselves have exported
			// fields.  See https://github.com/golang/go/issues/12367
			rtField := rt.Field(fx)
			if rtField.PkgPath != "" {
				continue // field isn't exported
			}
			field, err := typeFromReflectLocked(rtField.Type, builder, pending)
			if err != nil {
				return nil, err
			}
			st.AppendField(rtField.Name, field)
		}
		if rt.NumField() > 0 && st.NumField() == 0 {
			return nil, fmt.Errorf("type %q only has unexported fields", rt)
		}
		return st, nil
	}
	// Handle scalar types
	if t := typeFromRTKind[rt.Kind()]; t != nil {
		return t, nil
	}
	panic(fmt.Errorf("vdl: makeUnnamedFromReflectLocked unhandled %v %v", rt.Kind(), rt))
}

var (
	errTypeFromReflectValue = errors.New("invalid vdl.TypeOf(reflect.Value{})")

	ttByteList = ListType(ByteType)

	rtInterface          = reflect.TypeOf((*interface{})(nil)).Elem()
	rtBool               = reflect.TypeOf(false)
	rtByte               = reflect.TypeOf(byte(0))
	rtByteList           = reflect.TypeOf([]byte(nil))
	rtUint16             = reflect.TypeOf(uint16(0))
	rtUint32             = reflect.TypeOf(uint32(0))
	rtUint64             = reflect.TypeOf(uint64(0))
	rtInt8               = reflect.TypeOf(int8(0))
	rtInt16              = reflect.TypeOf(int16(0))
	rtInt32              = reflect.TypeOf(int32(0))
	rtInt64              = reflect.TypeOf(int64(0))
	rtFloat32            = reflect.TypeOf(float32(0))
	rtFloat64            = reflect.TypeOf(float64(0))
	rtString             = reflect.TypeOf("")
	rtError              = reflect.TypeOf((*error)(nil)).Elem()
	rtWireError          = reflect.TypeOf(WireError{})
	rtType               = reflect.TypeOf(Type{})
	rtValue              = reflect.TypeOf(Value{})
	rtPtrToType          = reflect.TypeOf((*Type)(nil))
	rtReflectValue       = reflect.TypeOf(reflect.Value{})
	rtNamer              = reflect.TypeOf((*namer)(nil)).Elem()
	rtIndexer            = reflect.TypeOf((*indexer)(nil)).Elem()
	rtIsZeroer           = reflect.TypeOf((*IsZeroer)(nil)).Elem()
	rtVDLWriter          = reflect.TypeOf((*Writer)(nil)).Elem()
	rtUnnamedEmptyStruct = reflect.TypeOf(struct{}{})

	typeFromRTKind = [...]*Type{
		reflect.Bool:    BoolType,
		reflect.Uint8:   ByteType,
		reflect.Uint16:  Uint16Type,
		reflect.Uint32:  Uint32Type,
		reflect.Uint64:  Uint64Type,
		reflect.Uint:    uintType(8 * unsafe.Sizeof(uint(0))),
		reflect.Uintptr: uintType(8 * unsafe.Sizeof(uintptr(0))),
		reflect.Int8:    Int8Type,
		reflect.Int16:   Int16Type,
		reflect.Int32:   Int32Type,
		reflect.Int64:   Int64Type,
		reflect.Int:     intType(8 * unsafe.Sizeof(int(0))),
		reflect.Float32: Float32Type,
		reflect.Float64: Float64Type,
		reflect.String:  StringType,
	}
)

func uintType(bitlen uintptr) *Type {
	switch bitlen {
	case 32:
		return Uint32Type
	default:
		return Uint64Type
	}
}

func intType(bitlen uintptr) *Type {
	switch bitlen {
	case 32:
		return Int32Type
	default:
		return Int64Type
	}
}
