// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

import (
	"fmt"
	"reflect"
	"sync"
)

// Register registers a type, identified by a value for that type.  The type
// should be a type that will be sent over the wire.  Subtypes are recursively
// registered.  This creates a type name <-> reflect.Type bijective mapping.
//
// Type registration is only required for VDL conversion into interface{}
// values, so that values of the correct type may be generated.  Conversion into
// interface{} values for types that are not registered will fill in *vdl.Value
// into the interface{} value.
//
// Panics if wire is not a valid wire type, or if the name<->type mapping is not
// bijective.
//
// Register is not intended to be called by end users; calls are auto-generated
// for all types defined in *.vdl files.
func Register(wire interface{}) {
	if wire == nil {
		return
	}
	if err := registerRecursive(reflect.TypeOf(wire)); err != nil {
		panic(err)
	}
}

func registerRecursive(rt reflect.Type) error {
	// 1) Normalize and derive reflect information.
	rt = normalizeType(rt)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	if rt == rtWireError || rt == rtError {
		// Multiple types map to ?WireError. verror.E should be treated as the
		// canonical reflect type so ignore the others.
		// TODO(bprosnitz) Remove this special case
		return nil
	}
	ri, added, err := deriveReflectInfo(rt)
	if err != nil {
		return err
	}
	if !added {
		// Break cyles for recursive types.
		//
		// TODO(toddw): There is a glaring bug in this logic.  Our first step is to
		// call normalizeType, which itself calls deriveReflectInfo.  Thus our
		// subsequent call to deriveReflectInfo will always return added=false, and
		// we will never run the logic below.  In addition, deriveReflectInfo is
		// also called outside of registerRecursive, with the same effect.
		//
		// The general philosophy for the fix:
		//   1) Types may be registered explicitly via Register.
		//   2) Types may be registered implicitly via all calls in vdl that take an
		//      interface{} argument.
		//   3) RegisterNative registers both wire and native types.
		//   4) Subtypes are always registered recursively.
		//   5) Once a type is registered, decode/convert into an interface{} works
		//      as expected, returning the concrete Go value.
		return nil
	}
	// 2) Recurse on subtypes contained in composite types.
	if len(ri.UnionFields) > 0 {
		// Special-case to recurse on union fields.
		for _, field := range ri.UnionFields {
			if err := registerRecursive(field.Type); err != nil {
				return err
			}
		}
		return nil
	}
	switch wt := ri.Type; wt.Kind() {
	case reflect.Array, reflect.Slice, reflect.Ptr:
		return registerRecursive(wt.Elem())
	case reflect.Map:
		if err := registerRecursive(wt.Key()); err != nil {
			return err
		}
		return registerRecursive(wt.Elem())
	case reflect.Struct:
		for ix := 0; ix < wt.NumField(); ix++ {
			if err := registerRecursive(wt.Field(ix).Type); err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

// riRegistry holds the reflectInfo registry.  Unlike rtRegistry (used for the
// rtCache), this information cannot be regenerated at will.  We expect a
// limited number of types to be used within a single address space.
type riRegistry struct {
	sync.RWMutex
	fromName map[string]*reflectInfo
	fromType map[reflect.Type]*reflectInfo
}

var riReg = &riRegistry{
	fromName: make(map[string]*reflectInfo),
	fromType: make(map[reflect.Type]*reflectInfo),
}

// reflectInfoFromName returns the reflectInfo for the given vdl type name, or
// nil if Register has not been called for a type with the given name.
func reflectInfoFromName(name string) *reflectInfo {
	riReg.RLock()
	ri := riReg.fromName[name]
	riReg.RUnlock()
	return ri
}

// WrapInUnionInterface returns a value of the union interface type that holds
// the union value rv.  Returns an invalid reflect.Value if rv isn't a union
// value.  Returns rv unchanged if its type is already the interface type.
func WrapInUnionInterface(rv reflect.Value) reflect.Value {
	ri, _, err := deriveReflectInfo(rv.Type())
	switch {
	case err != nil || len(ri.UnionFields) == 0:
		return reflect.Value{} // rv isn't a union
	case ri.Type == rv.Type():
		return rv // rv's type is already the interface type
	}
	// Here ri.Type is the union interface type, while rv is a concrete struct
	// type.  Wrap rv in a value of the interface type.
	rvIface := reflect.New(ri.Type).Elem()
	rvIface.Set(rv)
	return rvIface
}

// reflectInfo holds the reflection information for a type.  All fields are
// populated via reflection over the Type.
//
// The type may include a special __VDLReflect function to describe metadata.
// This is only required for enum and union vdl types, which don't have a
// canonical Go representation.  All other fields are optional.
//
//   type Foo struct{}
//   func (Foo) __VDLReflect(struct{
//     // Type represents the base type.  This is used by union to describe the
//     // union interface type, as opposed to the concrete struct field types.
//     Type Foo
//
//     // Name holds the vdl type name, including the package path, in a tag.
//     Name string "vdl/pkg.Foo"
//
//     // Only one of Enum or Union should be set; they're both shown here for
//     // explanatory purposes.
//
//     // Enum describes the labels for an enum type.
//     Enum struct { A, B string }
//
//     // Union describes the union field names, along with the concrete struct
//     // field types, which contain the actual field types.
//     Union struct {
//       A FieldA
//       B FieldB
//     }
//   }
type reflectInfo struct {
	// Type is the basis for all other information in this struct.
	Type reflect.Type

	// Name is the vdl type name including the vdl package path,
	// e.g. "v.io/v23/vdl.Foo".
	Name string

	// EnumLabels holds the labels of an enum; it is non-empty iff the Type
	// represents a vdl enum.
	EnumLabels []string

	// UnionFields holds the fields of a union; it is non-empty iff the Type
	// represents a vdl union.
	UnionFields []reflectField
}

// reflectField describes the reflection info for a Union field.
type reflectField struct {
	// Given a vdl type Foo union{A bool;B string}, we generate:
	//   type Foo interface{...}
	//   type FooA struct{ Value bool }
	//   type FooB struct{ Value string }
	Name    string       // Field name, e.g. "A", "B"
	Type    reflect.Type // Field type, e.g. bool, string
	RepType reflect.Type // Concrete type representing the field, e.g. FooA, FooB
}

// deriveReflectInfo returns the reflectInfo corresponding to rt.
// REQUIRES: rt has been normalized, and pointers have been flattened.
func deriveReflectInfo(rt reflect.Type) (*reflectInfo, bool, error) {
	riReg.RLock()
	if ri, ok := riReg.fromType[rt]; ok {
		riReg.RUnlock()
		return ri, false, nil
	}
	riReg.RUnlock()

	// Set reasonable defaults for types that don't have the __VDLReflect method.
	ri := new(reflectInfo)
	ri.Type = rt
	if rt.PkgPath() != "" {
		ri.Name = rt.PkgPath() + "." + rt.Name()
	}
	// If rt is an non-interface type, methods include the receiver as the first
	// in-arg, otherwise they don't.
	offsetIn := 1
	if rt.Kind() == reflect.Interface {
		offsetIn = 0
	}
	// If rt has a __VDLReflect method, use it to extract metadata.
	if method, ok := rt.MethodByName("__VDLReflect"); ok {
		mtype := method.Type
		if mtype.NumOut() != 0 || mtype.NumIn() != 1+offsetIn || mtype.In(offsetIn).Kind() != reflect.Struct {
			return nil, false, fmt.Errorf("type %q invalid __VDLReflect (want __VDLReflect(struct{...}))", rt)
		}
		// rtReflect corresponds to the argument to __VDLReflect.
		rtReflect := mtype.In(offsetIn)
		if field, ok := rtReflect.FieldByName("Type"); ok {
			ri.Type = field.Type
			if wt := ri.Type; wt.PkgPath() != "" {
				ri.Name = wt.PkgPath() + "." + wt.Name()
			} else {
				ri.Name = ""
			}
		}
		if field, ok := rtReflect.FieldByName("Name"); ok {
			ri.Name = field.Tag.Get("vdl")
			if ri.Name == "" {
				return nil, false, fmt.Errorf("empty vdl tag on __VDLReflect Name field")
			}
		}
		if field, ok := rtReflect.FieldByName("Enum"); ok {
			if err := describeEnum(field.Type, rt, ri); err != nil {
				return nil, false, err
			}
		}
		if field, ok := rtReflect.FieldByName("Union"); ok {
			if err := describeUnion(field.Type, rt, ri); err != nil {
				return nil, false, err
			}
		}
		if len(ri.EnumLabels) > 0 && len(ri.UnionFields) > 0 {
			return nil, false, fmt.Errorf("type %q is both an enum and a union", rt)
		}
	}

	riReg.Lock()
	defer riReg.Unlock()

	if ri, ok := riReg.fromType[rt]; ok {
		return ri, false, nil
	}
	if ri.Name != "" {
		if riDup := riReg.fromName[ri.Name]; riDup != nil && ri.Type != riDup.Type {
			return nil, false, fmt.Errorf("vdl: Register(%v) duplicate name %q: %#v and %#v", rt, ri.Name, ri, riDup)
		}
		riReg.fromName[ri.Name] = ri
	}
	riReg.fromType[rt] = ri
	return ri, true, nil
}

// describeEnum fills in ri; we expect enumReflect has this format:
//   struct {A, B, C Foo}
//
// Here's the full type for vdl type Foo enum{A;B}
//   type Foo int
//   const (
//     FooA Foo = iota
//     FooB
//   )
//   func (Foo) __VDLReflect(struct{
//     Type Foo
//     Enum struct { A, B Foo }
//   }) {}
//   func (Foo) String() string {}
//   func (*Foo) Set(string) error {}
func describeEnum(enumReflect, rt reflect.Type, ri *reflectInfo) error {
	if rt != ri.Type || rt.Kind() == reflect.Interface {
		return fmt.Errorf("enum type %q invalid (mismatched type %q)", rt, ri.Type)
	}
	if enumReflect.Kind() != reflect.Struct || enumReflect.NumField() == 0 {
		return fmt.Errorf("enum type %q invalid (no labels)", rt)
	}
	for ix := 0; ix < enumReflect.NumField(); ix++ {
		ri.EnumLabels = append(ri.EnumLabels, enumReflect.Field(ix).Name)
	}
	if s, ok := rt.MethodByName("String"); !ok ||
		s.Type.NumIn() != 1 ||
		s.Type.NumOut() != 1 || s.Type.Out(0) != rtString {
		return fmt.Errorf("enum type %q must have method String() string", rt)
	}
	_, nonptr := rt.MethodByName("Set")
	if a, ok := reflect.PtrTo(rt).MethodByName("Set"); !ok || nonptr ||
		a.Type.NumIn() != 2 || a.Type.In(1) != rtString ||
		a.Type.NumOut() != 1 || a.Type.Out(0) != rtError {
		return fmt.Errorf("enum type %q must have pointer method Set(string) error", rt)
	}
	return nil
}

// describeUnion fills in ri; we expect unionReflect has this format:
//   struct {
//     A FooA
//     B FooB
//   }
//
// Here's the full type for vdl type Foo union{A bool; B string}
//   type (
//     // Foo is the union interface type, that can hold any field.
//     Foo interface {
//       Index() int
//       Name() string
//       __VDLReflect(__FooReflect)
//     }
//     // FooA and FooB are the concrete field types.
//     FooA struct { Value bool }
//     FooB struct { Value string }
//     // __FooReflect lets us re-construct the union type via reflection.
//     __FooReflect struct {
//       Type  Foo // Tells us the union interface type.
//       Union struct {
//         A FooA  // Tells us field 0 has name A and concrete type FooA.
//         B FooB  // Tells us field 1 has name B and concrete type FooB.
//       }
//     }
//   )
func describeUnion(unionReflect, rt reflect.Type, ri *reflectInfo) error {
	if ri.Type.Kind() != reflect.Interface {
		return fmt.Errorf("union type %q has non-interface type %q", rt, ri.Type)
	}
	if unionReflect.Kind() != reflect.Struct || unionReflect.NumField() == 0 {
		return fmt.Errorf("union type %q invalid (no fields)", rt)
	}
	for ix := 0; ix < unionReflect.NumField(); ix++ {
		f := unionReflect.Field(ix)
		if f.PkgPath != "" {
			return fmt.Errorf("union type %q field %q.%q must be exported", rt, f.PkgPath, f.Name)
		}
		// f.Type corresponds to FooA and FooB in __FooReflect above.
		if f.Type.Kind() != reflect.Struct || f.Type.NumField() != 1 || f.Type.Field(0).Name != "Value" {
			return fmt.Errorf("union type %q field %q has bad concrete field type %q", rt, f.Name, f.Type)
		}
		ri.UnionFields = append(ri.UnionFields, reflectField{
			Name:    f.Name,
			Type:    f.Type.Field(0).Type,
			RepType: f.Type,
		})
	}
	// Check for Name and Index methods on interface and concrete field structs.
	if !ri.Type.Implements(rtNamer) {
		return fmt.Errorf("union interface type %q must have method Name() string", ri.Type)
	}
	if !ri.Type.Implements(rtIndexer) {
		return fmt.Errorf("union interface type %q must have method Index() int", ri.Type)
	}
	for _, f := range ri.UnionFields {
		if !f.RepType.Implements(rtNamer) {
			return fmt.Errorf("union field %q type %q must have method Name() string", f.Name, f.RepType)
		}
		if !f.RepType.Implements(rtIndexer) {
			return fmt.Errorf("union field %q type %q must have method Index() int", f.Name, f.RepType)
		}
	}
	return nil
}

// TypeToReflect returns the reflect.Type corresponding to t.  We look up
// named types in our registry, and build the unnamed types that we can via the
// Go reflect package.  Returns nil for types that can't be manufactured.
func TypeToReflect(t *Type) reflect.Type {
	if t.Name() != "" {
		// Named types cannot be manufactured via Go reflect, so we lookup in our
		// registry instead.
		if ri := reflectInfoFromName(t.Name()); ri != nil {
			if ni := nativeInfoFromWire(ri.Type); ni != nil {
				return ni.NativeType
			}
			return ri.Type
		}
		return nil
	}
	// We can make some unnamed types via Go reflect.  Return nil otherwise.
	switch t.Kind() {
	case Any, Enum, Union:
		// We can't make unnamed versions of any of these types.
		return nil
	case Optional:
		if elem := TypeToReflect(t.Elem()); elem != nil {
			return reflect.PtrTo(elem)
		}
		return nil
	case Array:
		if elem := TypeToReflect(t.Elem()); elem != nil {
			return reflect.ArrayOf(t.Len(), elem)
		}
		return nil
	case List:
		if elem := TypeToReflect(t.Elem()); elem != nil {
			return reflect.SliceOf(elem)
		}
		return nil
	case Set:
		if key := TypeToReflect(t.Key()); key != nil {
			return reflect.MapOf(key, rtUnnamedEmptyStruct)
		}
		return nil
	case Map:
		if key, elem := TypeToReflect(t.Key()), TypeToReflect(t.Elem()); key != nil && elem != nil {
			return reflect.MapOf(key, elem)
		}
		return nil
	case Struct:
		if t.NumField() == 0 {
			return rtUnnamedEmptyStruct
		}
		return nil
	default:
		return rtFromKind[t.Kind()]
	}
}

// typeToReflectNew returns the reflect.Type corresponding to t.  We look up
// named types in our registry, and build the unnamed types that we can via the
// Go reflect package.  Returns nil for types that can't be manufactured.
//
// TODO(toddw): Replace TypeToReflect with this function, after the old
// conversion logic has been removed.  Using this function with the old
// conversion logic breaks the tests, which aren't worth it to fix.
func typeToReflectNew(t *Type) reflect.Type {
	if t.Name() != "" {
		// Named types cannot be manufactured via Go reflect, so we lookup in our
		// registry instead.
		if ri := reflectInfoFromName(t.Name()); ri != nil {
			if ni := nativeInfoFromWire(ri.Type); ni != nil {
				return ni.NativeType
			}
			return ri.Type
		}
		return nil
	}
	// We can make some unnamed types via Go reflect.  Return nil otherwise.
	switch t.Kind() {
	case Enum, Union:
		// We can't make unnamed versions of these types.
		return nil
	case Any:
		return rtInterface
	case Optional:
		// Handle native types that were registered with a pointer wire type,
		// e.g. wire=*WireError, native=error.
		if elem := t.Elem(); elem.Name() != "" {
			if ri := reflectInfoFromName(elem.Name()); ri != nil {
				if ni := nativeInfoFromWire(reflect.PtrTo(ri.Type)); ni != nil {
					return ni.NativeType
				}
			}
		}
		if elem := typeToReflectNew(t.Elem()); elem != nil {
			return reflect.PtrTo(elem)
		}
		return nil
	case Array:
		if elem := typeToReflectNew(t.Elem()); elem != nil {
			return reflect.ArrayOf(t.Len(), elem)
		}
		return nil
	case List:
		if elem := typeToReflectNew(t.Elem()); elem != nil {
			return reflect.SliceOf(elem)
		}
		return nil
	case Set:
		if key := typeToReflectNew(t.Key()); key != nil {
			return reflect.MapOf(key, rtUnnamedEmptyStruct)
		}
		return nil
	case Map:
		if key, elem := typeToReflectNew(t.Key()), typeToReflectNew(t.Elem()); key != nil && elem != nil {
			return reflect.MapOf(key, elem)
		}
		return nil
	case Struct:
		if t.NumField() == 0 {
			return rtUnnamedEmptyStruct
		}
		return nil
	default:
		return rtFromKind[t.Kind()]
	}
}

var rtFromKind = [...]reflect.Type{
	Bool:       rtBool,
	Byte:       rtByte,
	Uint16:     rtUint16,
	Uint32:     rtUint32,
	Uint64:     rtUint64,
	Int8:       rtInt8,
	Int16:      rtInt16,
	Int32:      rtInt32,
	Int64:      rtInt64,
	Float32:    rtFloat32,
	Float64:    rtFloat64,
	String:     rtString,
	TypeObject: rtPtrToType,
}
