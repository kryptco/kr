// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

import (
	"fmt"
	"strings"
)

// Kind represents the kind of type that a Type represents.
type Kind int

const (
	// Variant kinds
	Any      Kind = iota // any type
	Optional             // value might not exist
	// Scalar kinds
	Bool       // boolean
	Byte       // 8 bit unsigned integer
	Uint16     // 16 bit unsigned integer
	Uint32     // 32 bit unsigned integer
	Uint64     // 64 bit unsigned integer
	Int8       // 8 bit signed integer
	Int16      // 16 bit signed integer
	Int32      // 32 bit signed integer
	Int64      // 64 bit signed integer
	Float32    // 32 bit IEEE 754 floating point
	Float64    // 64 bit IEEE 754 floating point
	String     // unicode string (encoded as UTF-8 in memory)
	Enum       // one of a set of labels
	TypeObject // type represented as a value
	// Composite kinds
	Array  // fixed-length ordered sequence of elements
	List   // variable-length ordered sequence of elements
	Set    // unordered collection of distinct keys
	Map    // unordered association between distinct keys and values
	Struct // conjunction of an ordered sequence of (name,type) fields
	Union  // disjunction of an ordered sequence of (name,type) fields

	// Internal kinds; they never appear in a *Type returned to the user.
	internalNamed // placeholder for named types while they're being built.
)

func (k Kind) String() string {
	switch k {
	case Any:
		return "any"
	case Optional:
		return "optional"
	case Bool:
		return "bool"
	case Byte:
		return "byte"
	case Uint16:
		return "uint16"
	case Uint32:
		return "uint32"
	case Uint64:
		return "uint64"
	case Int8:
		return "int8"
	case Int16:
		return "int16"
	case Int32:
		return "int32"
	case Int64:
		return "int64"
	case Float32:
		return "float32"
	case Float64:
		return "float64"
	case String:
		return "string"
	case Enum:
		return "enum"
	case TypeObject:
		return "typeobject"
	case Array:
		return "array"
	case List:
		return "list"
	case Set:
		return "set"
	case Map:
		return "map"
	case Struct:
		return "struct"
	case Union:
		return "union"
	}
	panic(fmt.Errorf("vdl: unhandled kind: %d", k))
}

// IsNumber returns true iff the kind is a number.
func (k Kind) IsNumber() bool {
	switch k {
	case Byte, Uint16, Uint32, Uint64, Int8, Int16, Int32, Int64, Float32, Float64:
		return true
	}
	return false
}

// BitLen returns the number of bits in the representation of the kind;
// e.g. Int32 returns 32.  Returns -1 for non-number kinds.
func (k Kind) BitLen() int {
	switch k {
	case Byte, Int8:
		return 8
	case Uint16, Int16:
		return 16
	case Uint32, Int32, Float32:
		return 32
	case Uint64, Int64, Float64:
		return 64
	}
	return -1
}

type kindBitMask uint32

func (k *kindBitMask) Set(kind Kind) {
	*k |= (1 << uint(kind))
}

func (k kindBitMask) IsSet(kind Kind) bool {
	return (k & (1 << uint(kind))) != 0
}

// SplitIdent splits the given identifier into its package path and local name.
//   a/b.Foo   -> (a/b, Foo)
//   a.b/c.Foo -> (a.b/c, Foo)
//   Foo       -> ("",  Foo)
//   a/b       -> ("",  a/b)
func SplitIdent(ident string) (pkgpath, name string) {
	dot := strings.LastIndex(ident, ".")
	if dot == -1 {
		return "", ident
	}
	return ident[:dot], ident[dot+1:]
}

// Type is the representation of a vanadium type.  Types are hash-consed; each
// unique type is represented by exactly one *Type instance, so to test for type
// equality you just compare the *Type instances.
//
// Not all methods apply to all kinds of types.  Restrictions are noted in the
// documentation for each method.  Calling a method inappropriate to the kind of
// type causes a run-time panic.
//
// Cyclic types are supported; e.g. you can represent a tree via:
//   type Node struct {
//     Val      string
//     Children []Node
//   }
type Type struct {
	kind         Kind           // used by all kinds
	name         string         // used by all kinds
	labels       []string       // used by Enum
	len          int            // used by Array
	elem         *Type          // used by Optional, Array, List, Map
	key          *Type          // used by Set, Map
	fields       []Field        // used by Struct, Union
	fieldIndices map[string]int // used by Struct, Union
	unique       string         // used by all kinds, filled in by typeCons
	containsKind kindBitMask    // does this type recursively contain a given kind
}

// Field describes a single field in a Struct or Union.
type Field struct {
	Name string
	Type *Type
}

// Kind returns the kind of type t.
func (t *Type) Kind() Kind { return t.kind }

// Name returns the name of type t.  Empty names are allowed.
func (t *Type) Name() string { return t.name }

// String returns a human-readable description of type t.  Do not rely on the
// output format; it may change without notice.  See Unique for a format that is
// guaranteed never to change.
func (t *Type) String() string {
	return t.Unique()
}

// Unique returns a unique representation of type t.  Two types A and B are
// guaranteed to return the same unique string iff A is equal to B.  The format
// is guaranteed to never change.
//
// A typical use case is to hash the unique representation to produce
// globally-unique type ids.
//
// TODO(toddw): Make sure we're comfortable with the format we produce; if it
// needs to change, it needs to happen soon.
func (t *Type) Unique() string {
	if t.unique != "" {
		return t.unique
	}
	// The only time that t.unique isn't set is while we're in the process of
	// building the type, and we're printing the type for errors.  The type might
	// have unnamed cycles, so we need to use short cycle names.
	return uniqueTypeStr(t, make(map[*Type]bool), true)
}

// CanBeNil returns true iff values of t can be nil.
//
// Any and Optional values can be nil.
func (t *Type) CanBeNil() bool {
	return t.kind == Any || t.kind == Optional
}

// CanBeNamed returns true iff t can be made into a named type.
//
// Any and TypeObject cannot be named.
func (t *Type) CanBeNamed() bool {
	return t.kind != Any && t.kind != TypeObject
}

// CanBeKey returns true iff t can be used as a set or map key.
//
// Any, List, Map, Optional, Set and TypeObject cannot be keys, nor can
// composite types that contain these types.
func (t *Type) CanBeKey() bool {
	return !t.ContainsKind(WalkAll, Any, List, Map, Optional, Set, TypeObject)
}

// CanBeOptional returns true iff t can be made into an optional type.
//
// Only named structs can be optional.
func (t *Type) CanBeOptional() bool {
	// Our philosophy is that we should retain the full type information in our
	// generated code, and generating annotations to distinguish optional from
	// non-optional types is awkward for unnamed types.
	//
	// Allowing optionality for named types other than structs is also awkward.
	// E.g. if we allowed optional named maps, it's unclear how we'd generate it
	// in Go.  We might just generate a map, which is already a reference type and
	// may be nil, but then we can't distinguish optional map types from
	// non-optional map types.
	return t.name != "" && t.kind == Struct
}

// IsBytes returns true iff the kind of type is []byte or [N]byte.
func (t *Type) IsBytes() bool {
	return (t.kind == List || t.kind == Array) && t.elem.kind == Byte
}

var enumLabelAllowed = []Kind{Enum}

// EnumLabel returns the Enum label at the given index.  It panics if the index
// is out of range.
func (t *Type) EnumLabel(index int) string {
	t.checkKind("EnumLabel", enumLabelAllowed...)
	return t.labels[index]
}

var enumIndexAllowed = []Kind{Enum}

// EnumIndex returns the Enum index for the given label.  Returns -1 if the
// label doesn't exist.
func (t *Type) EnumIndex(label string) int {
	t.checkKind("EnumIndex", enumIndexAllowed...)
	// We typically have a small number of labels, so linear search is fine.
	for index, l := range t.labels {
		if l == label {
			return index
		}
	}
	return -1
}

var numEnumLabelAllowed = []Kind{Enum}

// NumEnumLabel returns the number of labels in an Enum.
func (t *Type) NumEnumLabel() int {
	t.checkKind("NumEnumLabel", numEnumLabelAllowed...)
	return len(t.labels)
}

var lenAllowed = []Kind{Array}

// Len returns the length of an Array.
func (t *Type) Len() int {
	t.checkKind("Len", lenAllowed...)
	return t.len
}

var elemAllowed = []Kind{Optional, Array, List, Map}

// Elem returns the element type of an Optional, Array, List or Map.
func (t *Type) Elem() *Type {
	t.checkKind("Elem", elemAllowed...)
	return t.elem
}

// NonOptional returns t.Elem() if t is Optional, otherwise returns t.
func (t *Type) NonOptional() *Type {
	if t.kind == Optional {
		return t.elem
	}
	return t
}

var keyAllowed = []Kind{Set, Map}

// Key returns the key type of a Set or Map.
func (t *Type) Key() *Type {
	t.checkKind("Key", keyAllowed...)
	return t.key
}

var fieldAllowed = []Kind{Struct, Union}

// Field returns a description of the Struct or Union field at the given index.
func (t *Type) Field(index int) Field {
	t.checkKind("Field", fieldAllowed...)
	return t.fields[index]
}

var fieldByNameAllowed = []Kind{Struct, Union}

// FieldByName returns a description of the Struct or Union field with the given
// name, and its index.  Returns -1 if the name doesn't exist.
func (t *Type) FieldByName(name string) (Field, int) {
	t.checkKind("FieldByName", fieldByNameAllowed...)
	if index, ok := t.fieldIndices[name]; ok {
		return t.fields[index], index
	}
	return Field{}, -1
}

// FieldIndexByName returns the index of the Struct or Union field with
// the given name.  Returns -1 if the name doesn't exist.
func (t *Type) FieldIndexByName(name string) int {
	t.checkKind("FieldIndexByName", fieldByNameAllowed...)
	if index, ok := t.fieldIndices[name]; ok {
		return index
	}
	return -1
}

var numFieldAllowed = []Kind{Struct, Union}

// NumField returns the number of fields in a Struct or Union.
func (t *Type) NumField() int {
	t.checkKind("NumField", numFieldAllowed...)
	return len(t.fields)
}

// AssignableFrom returns true iff values of t may be assigned from f:
//   o Allowed if t and the type of f are identical.
//   o Allowed if t is Any.
//   o Allowed if t is Optional, and f is Any(nil).
//
// The first rule establishes strict static typing.  The second rule relaxes
// things for Any, which is dynamically typed.  The third rule relaxes things
// further, to allow implicit conversions from Any(nil) to all Optional types.
func (t *Type) AssignableFrom(f *Value) bool {
	return t == f.t || t.kind == Any || (t.kind == Optional && f.t.kind == Any && f.IsNil())
}

// VDLIsZero returns true if t is nil or AnyType.
func (t *Type) VDLIsZero() bool {
	return t == nil || t == AnyType
}

// VDLWrite uses enc to encode type t.
//
// Unlike regular VDLWrite implementations, this handles the case where t
// contains a nil value, to make code generation simpler.
func (t *Type) VDLWrite(enc Encoder) error {
	if t == nil {
		t = AnyType
	}
	return enc.WriteValueTypeObject(t)
}

// ptype implements the TypeOrPending interface.
func (t *Type) ptype() *Type { return t }

func (t *Type) errKind(method string, allowed ...Kind) error {
	return fmt.Errorf("vdl: %s mismatched kind; got: %v, want: %v", method, t, allowed)
}

func (t *Type) errBytes(method string) error {
	return fmt.Errorf("vdl: %s mismatched type; got: %v, want: bytes", method, t)
}

func (t *Type) checkKind(method string, allowed ...Kind) {
	if t != nil {
		for _, k := range allowed {
			if k == t.kind {
				return
			}
		}
	}
	panic(t.errKind(method, allowed...))
}

func (t *Type) checkIsBytes(method string) {
	if !t.IsBytes() {
		panic(t.errBytes(method))
	}
}

// ContainsKind returns true iff t or subtypes of t match any of the kinds.
func (t *Type) ContainsKind(mode WalkMode, kinds ...Kind) bool {
	var containsKind bool
	for _, kind := range kinds {
		if t.containsKind.IsSet(kind) {
			containsKind = true
			break
		}
	}
	if mode == WalkAll || containsKind == false {
		return containsKind
	}
	return !t.Walk(mode, func(visit *Type) bool {
		for _, kind := range kinds {
			if kind == visit.kind {
				return false
			}
		}
		return true
	})
}

// ContainsType returns true iff t or subtypes of t match any of the types.
func (t *Type) ContainsType(mode WalkMode, types ...*Type) bool {
	return !t.Walk(mode, func(visit *Type) bool {
		for _, ty := range types {
			if ty == visit {
				return false
			}
		}
		return true
	})
}

// Walk performs a DFS walk through the type graph starting from t, calling fn
// for each visited type.  If fn returns false on a visited type, the walk is
// terminated early, and false is returned by Walk.  The mode controls which
// types in the type graph we will visit.
func (t *Type) Walk(mode WalkMode, fn func(*Type) bool) bool {
	return typeWalk(mode, t, fn, make(map[*Type]bool))
}

// WalkMode is the mode to perform a Walk through the type graph.
type WalkMode int

const (
	// WalkAll indicates we should walk through all types in the type graph.
	WalkAll WalkMode = iota
	// WalkInline indicates we should only visit subtypes of array, struct and
	// union.  Values of array, struct and union always include values of their
	// subtypes, thus the subtypes are considered to be inline.  Values of
	// optional, list, set and map might not include values of their subtypes, and
	// are not considered to be inline.
	WalkInline
)

func (t *Type) subTypesInline() bool {
	switch t.kind {
	case Array, Struct, Union:
		return true
	}
	return false
}

func typeWalk(mode WalkMode, t *Type, fn func(*Type) bool, seen map[*Type]bool) bool {
	if seen[t] {
		return true
	}
	seen[t] = true
	if !fn(t) {
		return false
	}
	if mode == WalkInline && !t.subTypesInline() {
		return true
	}
	switch t.kind {
	case Optional, Array, List:
		return typeWalk(mode, t.elem, fn, seen)
	case Set:
		return typeWalk(mode, t.key, fn, seen)
	case Map:
		return typeWalk(mode, t.key, fn, seen) && typeWalk(mode, t.elem, fn, seen)
	case Struct, Union:
		for _, field := range t.fields {
			if !typeWalk(mode, field.Type, fn, seen) {
				return false
			}
		}
	}
	return true
}

// IsPartOfCycle returns true iff t is part of a cycle.  Note that t is not
// considered to be part of a cycle if it merely contains another type that is
// part of a cycle; the type graph must cycle back through t to return true.
func (t *Type) IsPartOfCycle() bool {
	return partOfCycle(t, make(map[*Type]bool))
}

func partOfCycle(t *Type, inCycle map[*Type]bool) bool {
	if c, ok := inCycle[t]; ok {
		return c
	}
	inCycle[t] = true
	switch t.kind {
	case Optional, Array, List:
		if partOfCycle(t.elem, inCycle) {
			return true
		}
	case Set:
		if partOfCycle(t.key, inCycle) {
			return true
		}
	case Map:
		if partOfCycle(t.key, inCycle) {
			return true
		}
		if partOfCycle(t.elem, inCycle) {
			return true
		}
	case Struct, Union:
		for _, x := range t.fields {
			if partOfCycle(x.Type, inCycle) {
				return true
			}
		}
	}
	inCycle[t] = false
	return false
}
