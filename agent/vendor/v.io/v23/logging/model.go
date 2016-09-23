// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package loging defines an interface for logging modeled on Google's glog.
package logging

type InfoLog interface {
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
}

type ErrorLog interface {
	// Error logs to the ERROR and INFO logs.
	// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
	Error(args ...interface{})

	// ErrorDepth acts as Error but uses depth to determine which call frame to log.
	// A depth of 0 is equivalent to calling Error.
	ErrorDepth(depth int, args ...interface{})

	// Errorf logs to the ERROR and INFO logs.
	// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
	Errorf(format string, args ...interface{})
}

type FatalLog interface {
	// Fatal logs to the FATAL, ERROR and INFO logs,
	// including a stack trace of all running goroutines, then calls os.Exit(255).
	// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
	Fatal(args ...interface{})

	// FatalDepth acts as Fatal but uses depth to determine which call frame to log.
	// A depth of 0 is equivalent to calling Fatal.
	FatalDepth(depth int, args ...interface{})

	// Fatalf logs to the FATAL, ERROR and INFO logs,
	// including a stack trace of all running goroutines, then calls os.Exit(255).
	// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
	Fatalf(format string, args ...interface{})
}

type PanicLog interface {
	// Panic is equivalent to Error() followed by a call to panic().
	Panic(args ...interface{})

	// PanicDepth acts as Panic but uses depth to determine which call frame to log.
	// A depth of 0 is equivalent to calling Panic.
	PanicDepth(depth int, args ...interface{})

	// Panicf is equivalent to Errorf() followed by a call to panic().
	Panicf(format string, args ...interface{})
}

type Verbosity interface {
	// V returns true if the configured logging level is greater than or equal to its parameter
	V(level int) bool

	// VDepth acts as V but uses depth to determine which call frame to test against.
	VDepth(depth int, level int) bool

	// VI is like V, except that it returns an anonymous interface that with the
	// same method set as InfoLog that will either log (if level >= the configured level)
	// or discard its parameters. This allows for logger.VI(2).Info style usage. An
	// anonymous interface is used to allow for implementations that don't need to
	// depend on this package.
	VI(level int) interface {
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
	}

	// VIDepth acts as VI but uses depth to determine which call frame to test against.
	VIDepth(depth int, level int) interface {
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
	}
}

type Logger interface {
	InfoLog
	ErrorLog
	FatalLog
	PanicLog
	Verbosity
	// Flush flushes all pending log I/O.
	FlushLog()
}
