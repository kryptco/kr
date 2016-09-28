// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package v23

import "v.io/v23/context"

// TryInit is like Init, except that it returns an error instead of panicking.
func TryInit() (*context.T, Shutdown, error) {
	return internalInit()
}

// Init should be called once for each vanadium executable, providing
// the setup of the vanadium initial context.T and a Shutdown function
// that can be used to clean up the runtime.  We allow calling Init
// multiple times (useful in tests), but only as long as you call the
// Shutdown returned previously before calling Init the second time.
// Init panics if it encounters an error.
func Init() (*context.T, Shutdown) {
	ctx, shutdown, err := internalInit()
	if err != nil {
		panic(err)
	}
	return ctx, shutdown
}
