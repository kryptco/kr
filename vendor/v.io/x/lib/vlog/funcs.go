// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vlog

import (
	"github.com/cosnicolaou/llog"
)

// Info logs to the INFO log.
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func Info(args ...interface{}) {
	Log.log.Print(llog.InfoLog, args...)
	Log.maybeFlush()
}

// Infof logs to the INFO log.
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func Infof(format string, args ...interface{}) {
	fn := Log.log.Printf // needed to avoid go vet warning
	fn(llog.InfoLog, format, args...)
	Log.maybeFlush()
}

// InfoDepth acts as Info but uses depth to determine which call frame to log.
// A depth of 0 is equivalent to calling Info.
func InfoDepth(depth int, args ...interface{}) {
	Log.log.PrintDepth(llog.InfoLog, depth, args...)
	Log.maybeFlush()
}

// InfoStack logs the current goroutine's stack if the all parameter
// is false, or the stacks of all goroutines if it's true.
func InfoStack(all bool) {
	infoStack(Log, all)
}

// V returns true if the configured logging level is greater than or equal to its parameter
func V(level Level) bool {
	return Log.log.VDepth(0, llog.Level(level))
}

// VI is like V, except that it returns an instance of the Info
// interface that will either log (if level >= the configured level)
// or discard its parameters. This allows for logger.VI(2).Info
// style usage.
func VI(level Level) interface {
	// Info logs to the INFO log.
	// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
	Info(args ...interface{})

	// Infoln logs to the INFO log.
	// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
	Infof(format string, args ...interface{})

	// InfoDepth acts as Info but uses depth to determine which call frame to log.
	// A depth of 0 is equivalent to calling Info.
	InfoDepth(depth int, args ...interface{})

	// InfoStack logs the current goroutine's stack if the all parameter
	// is false, or the stacks of all goroutines if it's true.
	InfoStack(all bool)
} {
	if Log.log.VDepth(0, llog.Level(level)) {
		return Log
	}
	return &discardInfo{}
}

// Flush flushes all pending log I/O.
func FlushLog() {
	Log.FlushLog()
}

// Error logs to the ERROR and INFO logs.
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func Error(args ...interface{}) {
	Log.log.Print(llog.ErrorLog, args...)
	Log.maybeFlush()
}

// ErrorDepth acts as Error but uses depth to determine which call frame to log.
// A depth of 0 is equivalent to calling Error.
func ErrorDepth(depth int, args ...interface{}) {
	Log.log.PrintDepth(llog.ErrorLog, depth, args...)
	Log.maybeFlush()
}

// Errorf logs to the ERROR and INFO logs.
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func Errorf(format string, args ...interface{}) {
	fn := Log.log.Printf // needed to avoid go vet warning
	fn(llog.ErrorLog, format, args...)
	Log.maybeFlush()
}

// Fatal logs to the FATAL, ERROR and INFO logs,
// including a stack trace of all running goroutines, then calls os.Exit(255).
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func Fatal(args ...interface{}) {
	Log.log.Print(llog.FatalLog, args...)
}

// FatalDepth acts as Fatal but uses depth to determine which call frame to log.
// A depth of 0 is equivalent to calling Fatal.
func FatalDepth(depth int, args ...interface{}) {
	Log.log.PrintDepth(llog.FatalLog, depth, args...)
}

// Fatalf logs to the FATAL, ERROR and INFO logs,
// including a stack trace of all running goroutines, then calls os.Exit(255).
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func Fatalf(format string, args ...interface{}) {
	fn := Log.log.Printf // needed to avoid go vet warning
	fn(llog.FatalLog, format, args...)
}

// ConfigureLogging configures all future logging. Some options
// may not be usable if ConfigureLogging is called from an init function,
// in which case an error will be returned. The Configured error is
// returned if ConfigureLogger has already been called unless the
// OverridePriorConfiguration options is included.
func Configure(opts ...LoggingOpts) error {
	return Log.Configure(opts...)
}

// Stats returns stats on how many lines/bytes haven been written to
// this set of logs.
func Stats() (Info, Error struct{ Lines, Bytes int64 }) {
	return Log.Stats()
}

// Panic is equivalent to Error() followed by a call to panic().
func Panic(args ...interface{}) {
	Log.Panic(args...)
}

// PanicDepth acts as Panic but uses depth to determine which call frame to log.
// A depth of 0 is equivalent to calling Panic.
func PanicDepth(depth int, args ...interface{}) {
	Log.PanicDepth(depth, args...)
}

// Panicf is equivalent to Errorf() followed by a call to panic().
func Panicf(format string, args ...interface{}) {
	Log.Panicf(format, args...)
}
