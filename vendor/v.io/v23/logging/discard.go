// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

type discard struct{}

// Discard is an implementation of Logger that discards all input, the equivalent
// of /dev/null. Fatal*, Panic* and all management  methods are also no-ops.
var Discard Logger = &discard{}

func (*discard) Info(args ...interface{})                 {}
func (*discard) InfoDepth(depth int, args ...interface{}) {}
func (*discard) Infof(format string, args ...interface{}) {}
func (*discard) InfoStack(all bool)                       {}

func (*discard) Error(args ...interface{})                 {}
func (*discard) ErrorDepth(depth int, args ...interface{}) {}
func (*discard) Errorf(format string, args ...interface{}) {}

func (*discard) Fatal(args ...interface{})                 {}
func (*discard) FatalDepth(depth int, args ...interface{}) {}
func (*discard) Fatalf(format string, args ...interface{}) {}

func (*discard) Panic(args ...interface{})                 {}
func (*discard) PanicDepth(depth int, args ...interface{}) {}
func (*discard) Panicf(format string, args ...interface{}) {}

func (*discard) V(level int) bool                 { return false }
func (*discard) VDepth(depth int, level int) bool { return false }

func (d *discard) VI(level int) interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	InfoDepth(depth int, args ...interface{})
	InfoStack(all bool)
} {
	return d
}

func (d *discard) VIDepth(depth int, level int) interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	InfoDepth(depth int, args ...interface{})
	InfoStack(all bool)
} {
	return d
}

func (*discard) FlushLog() {}
