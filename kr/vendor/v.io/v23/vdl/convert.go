// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

import (
	"reflect"
)

// Convert converts from src to dst.
func Convert(dst, src interface{}) error {
	return convertPipe(dst, src)
}

// ConvertReflect converts reflect values from src to dst.
func ConvertReflect(dst, src reflect.Value) error {
	return convertPipeReflect(dst, src)
}

// ValueOf returns the value corresponding to v.  It's a helper for calling
// ValueFromReflect, and panics on any errors.
func ValueOf(v interface{}) *Value {
	vv, err := ValueFromReflect(reflect.ValueOf(v))
	if err != nil {
		panic(err)
	}
	return vv
}

// ValueFromReflect returns the value corresponding to rv.
func ValueFromReflect(rv reflect.Value) (*Value, error) {
	if !rv.IsValid() {
		// TODO(bprosnitz) Is this the behavior we want?
		return ZeroValue(AnyType), nil
	}
	var result *Value
	err := convertPipeReflect(reflect.ValueOf(&result), rv)
	return result, err
}
