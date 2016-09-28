// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math

func Complex64ToNative(wire Complex64, native *complex64) error {
	*native = complex(wire.Real, wire.Imag)
	return nil
}
func Complex64FromNative(wire *Complex64, native complex64) error {
	wire.Real = real(native)
	wire.Imag = imag(native)
	return nil
}

func Complex128ToNative(wire Complex128, native *complex128) error {
	*native = complex(wire.Real, wire.Imag)
	return nil
}
func Complex128FromNative(wire *Complex128, native complex128) error {
	wire.Real = real(native)
	wire.Imag = imag(native)
	return nil
}
