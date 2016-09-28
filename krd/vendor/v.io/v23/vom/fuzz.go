// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build gofuzz

package vom

// To use go-fuzz:
//
// $ jiri go get github.com/dvyukov/go-fuzz/go-fuzz{,-build}
// $ cd $JIRI_ROOT/release/go/src/v.io/v23/vom
// $ jiri go test -tags fuzzdump
// $ jiri run go-fuzz-build -o fuzz-vom.zip v.io/v23/vom
// $ jiri run go-fuzz -bin fuzz-vom.zip -workdir fuzz-workdir
//
// Inputs resulting in crashes will be in workdir/crashers.
//
// go-fuzz will explore the space of possible input faster if
// you put bigger inputs into fuzz-workdir/corpus to help it. To do so,
// run "jiri go test -tags fuzzdump" once. This will copy all inputs
// used by the tests into fuzz-workdir/corpus (see fuzzdump_test.go).

import "bytes"

func Fuzz(data []byte) int {
	var v interface{}
	d := NewDecoder(bytes.NewReader(data))
	if err := d.Decode(&v); err != nil {
		return 0 // failed decode; fuzz is indifferent
	}
	return 1 // successful decode; give fuzz priority
}
