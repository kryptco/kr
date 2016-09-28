// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build java android
//
// We only expose the functionality below for the above build tags, to
// discourage general usage.  The binaries that currently require this
// functionality are our langage proxies:
//   java/android: jni

package security

import (
	"v.io/v23/context"
)

// OverrideCaveatValidation overrides the validation mechanism for all caveats;
// e.g. for use in proxies that delegate caveat validation to another process.
// It may be called at most once in an address space; subsequent calls with
// panic.
//
// The given fn is used to validate sets of caveats in RemoteBlessingNames,
// where the fn is invoked with the context and call representing the current
// security state, along with the sets of caveats to validate.  It should return
// a slice of errors where len(sets) == len(errors), and errors[x] corresponds
// to the result of validating sets[x].
//
// WARNING: This is not meant as a general API, and may change without notice.
// It is restricted to specific build tags to discourage general usage.
func OverrideCaveatValidation(fn func(ctx *context.T, call Call, sets [][]Caveat) []error) {
	overrideCaveatValidation(fn)
}
