// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

import (
	"fmt"
	"reflect"
	"unsafe"
)

// TODO(toddw): Add tests.

// Equaler is the interface that wraps the VDLEqual method.
//
// VDLEqual returns true iff the receiver that implements this method is equal
// to v.  The semantics of the equality must abide by VDL equality rules.  The
// caller of this method must ensure that the type of the receiver is the same
// as the type of v, and v is never nil.
type Equaler interface {
	VDLEqual(v interface{}) bool
}

var rtEqualer = reflect.TypeOf((*Equaler)(nil)).Elem()

// DeepEqual is like reflect.DeepEqual, with the following differences:
//   1. If a value is encountered that implements Equaler, we will use that for
//      the comparison.
//   2. If cyclic values are encountered, we require that the cyclic structure
//      of the two values is the same.
func DeepEqual(a, b interface{}) bool {
	return deepEqual(reflect.ValueOf(a), reflect.ValueOf(b), nil, nil)
}

// DeepEqualReflect is the same as DeepEqual, but takes reflect.Value arguments.
func DeepEqualReflect(a, b reflect.Value) bool {
	return deepEqual(a, b, nil, nil)
}

func findPathIndex(path []unsafe.Pointer, target unsafe.Pointer) int {
	for index, item := range path {
		if item == target {
			return index
		}
	}
	return -1
}

func deepEqual(a, b reflect.Value, pathA, pathB []unsafe.Pointer) bool {
	if !a.IsValid() || !b.IsValid() {
		return a.IsValid() == b.IsValid()
	}
	if a.Type() != b.Type() {
		return false
	}

	// Handle VDLEqual comparisons.
	if a.Kind() != reflect.Ptr || (!a.IsNil() && !b.IsNil()) {
		// It would be nice to use a.Interface() to get the actual value, and then
		// call the VDLEqual method directly.  But a.Interface() panics if a is an
		// unexported struct field.  We might actually encounter this case, if we
		// change our codegen to include an unexported "unknown bytes" field in
		// structs, in order to avoid read-modify-write slicing.
		//
		// TODO(toddw): Verify the logic below actually allows us to find and call
		// the VDLEqual method, if a is an unexported struct field.
		if a.Type().Implements(rtEqualer) && (a.Kind() != reflect.Ptr || !a.Type().Elem().Implements(rtEqualer)) {
			// Note: We check that the child type is not an Equaler because
			// the receiver of the VDLEqual method must be the same as the
			// argument type.
			return a.MethodByName("VDLEqual").Call([]reflect.Value{b})[0].Bool()
		}
	}

	// In order to handle cyclic values, we keep the path of possible "pointees"
	// as we traverse the value, where the "pointee" is the address that a pointer
	// could point to.  The pointer handling case below uses this information to
	// detect and handle cycles.
	//
	// We must convert the result of reflect.Value.UnsafeAddr() to unsafe.Pointer
	// in the same expression.  See https://golang.org/pkg/unsafe/#Pointer
	switch canA, canB := a.CanAddr(), b.CanAddr(); {
	case canA && canB:
		pathA = append(pathA, unsafe.Pointer(a.UnsafeAddr()))
		pathB = append(pathB, unsafe.Pointer(b.UnsafeAddr()))
	case canA:
		pathA = append(pathA, unsafe.Pointer(a.UnsafeAddr()))
		pathB = append(pathB, unsafe.Pointer(uintptr(0)))
	case canB:
		pathA = append(pathA, unsafe.Pointer(uintptr(0)))
		pathB = append(pathB, unsafe.Pointer(b.UnsafeAddr()))
	}

	switch a.Kind() {
	case reflect.Ptr:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() == b.IsNil()
		}
		// We must convert the result of reflect.Value.Pointer() to unsafe.Pointer
		// in the same expression.  See https://golang.org/pkg/unsafe/#Pointer
		pa, pb := unsafe.Pointer(a.Pointer()), unsafe.Pointer(b.Pointer())
		if pa == pb {
			// If the pointers are equal, the values are equal.
			return true
		}
		switch indexA, indexB := findPathIndex(pathA, pa), findPathIndex(pathB, pb); {
		case indexA != indexB:
			// The index is -1 if the pointer doesn't exist in the path, meaning this
			// isn't a cyclic value.  Otherwise the index tells us the which item the
			// cycle points back to.  Either way, if they are different, the values
			// are not equal.
			return false
		case indexA != -1:
			// If both values have cycles pointing back to the same relative item, we
			// need to stop, otherwise there is an infinite loop.  All previous items
			// in the path were equal, so we return true.
			return true
		}
		return deepEqual(a.Elem(), b.Elem(), pathA, pathB)
	case reflect.Array:
		if a.Len() != b.Len() {
			return false
		}
		for ix := 0; ix < a.Len(); ix++ {
			if !deepEqual(a.Index(ix), b.Index(ix), pathA, pathB) {
				return false
			}
		}
		return true
	case reflect.Slice:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() == b.IsNil()
		}
		if a.Len() != b.Len() {
			return false
		}
		for ix := 0; ix < a.Len(); ix++ {
			if !deepEqual(a.Index(ix), b.Index(ix), pathA, pathB) {
				return false
			}
		}
		return true
	case reflect.Map:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() == b.IsNil()
		}
		if a.Len() != b.Len() {
			return false
		}
		for _, key := range a.MapKeys() {
			if !deepEqual(a.MapIndex(key), b.MapIndex(key), pathA, pathB) {
				return false
			}
		}
		return true
	case reflect.Struct:
		for ix := 0; ix < a.NumField(); ix++ {
			if !deepEqual(a.Field(ix), b.Field(ix), pathA, pathB) {
				return false
			}
		}
		return true
	case reflect.Interface:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() == b.IsNil()
		}
		return deepEqual(a.Elem(), b.Elem(), pathA, pathB)

		// Ideally we would add a default clause here that would just return
		// a.Interface() == b.Interface(), but that panics if we're dealing with
		// unexported fields.  Instead we check each case manually.

	case reflect.Bool:
		return a.Bool() == b.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() == b.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return a.Uint() == b.Uint()
	case reflect.Float32, reflect.Float64:
		return a.Float() == b.Float()
	case reflect.Complex64, reflect.Complex128:
		return a.Complex() == b.Complex()
	case reflect.String:
		return a.String() == b.String()
	case reflect.UnsafePointer:
		return a.Pointer() == b.Pointer()
	case reflect.Func:
		// Same as regular Go comparisons; non-nil functions can't be compared.
		return a.IsNil() && b.IsNil()
	case reflect.Chan:
		// We must convert the result of reflect.Value.Pointer() to unsafe.Pointer
		// in the same expression.  See https://golang.org/pkg/unsafe/#Pointer
		return unsafe.Pointer(a.Pointer()) == unsafe.Pointer(b.Pointer())
	default:
		panic(fmt.Errorf("DeepEqual unhandled kind %v type %q", a.Kind(), a.Type()))
	}
}
