// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vlog

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/cosnicolaou/llog"
)

const (
	initialMaxStackBufSize = 128 * 1024
)

// Level specifies a level of verbosity for V logs.
// It can be set via the Level optional parameter to Configure.
// It implements the flag.Value interface to support command line option parsing.
type Level llog.Level

// Set is part of the flag.Value interface.
func (l *Level) Set(v string) error {
	return (*llog.Level)(l).Set(v)
}

// Get is part of the flag.Value interface.
func (l *Level) Get(v string) interface{} {
	return *l
}

// String is part of the flag.Value interface.
func (l *Level) String() string {
	return (*llog.Level)(l).String()
}

// StderrThreshold identifies the sort of log: info, warning etc.
// The values match the corresponding constants in C++ - e.g WARNING etc.
// It can be set via the StderrThreshold optional parameter to Configure.
// It implements the flag.Value interface to support command line option parsing.
type StderrThreshold llog.Severity

// Set is part of the flag.Value interface.
func (s *StderrThreshold) Set(v string) error {
	return (*llog.Severity)(s).Set(v)
}

// Get is part of the flag.Value interface.
func (s *StderrThreshold) Get(v string) interface{} {
	return *s
}

// String is part of the flag.Value interface.
func (s *StderrThreshold) String() string {
	return (*llog.Severity)(s).String()
}

// ModuleSpec allows for the setting of specific log levels for specific
// modules. The syntax is recordio=2,file=1,gfs*=3
// It can be set via the ModuleSpec optional parameter to Configure.
// It implements the flag.Value interface to support command line option parsing.
type ModuleSpec struct {
	llog.ModuleSpec
}

// FilepathSpec allows for the setting of specific log levels for specific
// files matched by a regular expression. The syntax is <re>=3,<re1>=2.
// It can be set via the FilepathSpec optional parameter to Configure.
// It implements the flag.Value interface to support command line option parsing.
type FilepathSpec struct {
	llog.FilepathSpec
}

// TraceLocation specifies the location, file:N, which when encountered will
// cause logging to emit a stack trace.
// It can be set via the TraceLocation optional parameter to Configure.
// It implements the flag.Value interface to support command line option parsing.
type TraceLocation struct {
	llog.TraceLocation
}

type Logger struct {
	log             *llog.Log
	mu              sync.Mutex // guards updates to the vars below.
	autoFlush       bool
	maxStackBufSize int
	logDir          string
	configured      bool
}

func (l *Logger) maybeFlush() {
	if l.autoFlush {
		l.log.Flush()
	}
}

var (
	Log           *Logger
	ErrConfigured = errors.New("logger has already been configured")
)

const stackSkip = 1

func init() {
	Log = &Logger{log: llog.NewLogger("vlog", stackSkip)}
}

// NewLogger creates a new instance of the logging interface.
func NewLogger(name string) *Logger {
	// Create an instance of the runtime with just logging enabled.
	return &Logger{log: llog.NewLogger(name, stackSkip)}
}

// Configure configures all future logging. Some options
// may not be usable if ConfigureLogging is called from an init function,
// in which case an error will be returned. The Configured error is returned
// if ConfigureLogger has already been called unless the
// OverridePriorConfiguration options is included.
func (l *Logger) Configure(opts ...LoggingOpts) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	override := false
	for _, o := range opts {
		switch v := o.(type) {
		case OverridePriorConfiguration:
			override = bool(v)
		}
	}
	if l.configured && !override {
		return ErrConfigured
	}
	for _, o := range opts {
		switch v := o.(type) {
		case AlsoLogToStderr:
			l.log.SetAlsoLogToStderr(bool(v))
		case Level:
			l.log.SetV(llog.Level(v))
		case LogDir:
			l.logDir = string(v)
			l.log.SetLogDir(l.logDir)
		case LogToStderr:
			l.log.SetLogToStderr(bool(v))
		case MaxStackBufSize:
			sz := int(v)
			if sz > initialMaxStackBufSize {
				l.maxStackBufSize = sz
				l.log.SetMaxStackBufSize(sz)
			}
		case ModuleSpec:
			l.log.SetVModule(v.ModuleSpec)
		case FilepathSpec:
			l.log.SetVFilepath(v.FilepathSpec)
		case TraceLocation:
			l.log.SetTraceLocation(v.TraceLocation)
		case StderrThreshold:
			l.log.SetStderrThreshold(llog.Severity(v))
		case AutoFlush:
			l.autoFlush = bool(v)
		}
	}
	l.configured = true
	return nil
}

// LogDir returns the directory where the log files are written.
func (l *Logger) LogDir() string {
	if len(l.logDir) != 0 {
		return l.logDir
	}
	return os.TempDir()
}

// Stats returns stats on how many lines/bytes haven been written to
// this set of logs.
func (l *Logger) Stats() (Info, Error struct{ Lines, Bytes int64 }) {
	stats := l.log.Stats()
	return struct{ Lines, Bytes int64 }{Lines: stats.Info.Lines(), Bytes: stats.Info.Bytes()},
		struct{ Lines, Bytes int64 }{Lines: stats.Error.Lines(), Bytes: stats.Error.Bytes()}
}

// Info logs to the INFO log.
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func (l *Logger) Info(args ...interface{}) {
	l.log.Print(llog.InfoLog, args...)
	l.maybeFlush()
}

// Infof logs to the INFO log.
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log.PrintfDepth(llog.InfoLog, 0, format, args...)
	l.maybeFlush()
}

// InfoDepth acts as Info but uses depth to determine which call frame to log.
// A depth of 0 is equivalent to calling Info.
func (l *Logger) InfoDepth(depth int, args ...interface{}) {
	l.log.PrintDepth(llog.InfoLog, depth, args...)
	l.maybeFlush()
}

func infoStack(l *Logger, all bool) {
	n := initialMaxStackBufSize
	var trace []byte
	for n <= l.maxStackBufSize {
		trace = make([]byte, n)
		nbytes := runtime.Stack(trace, all)
		if nbytes < len(trace) {
			l.log.PrintfDepth(llog.InfoLog, 0, "%s", trace[:nbytes])
			return
		}
		n *= 2
	}
	l.log.PrintfDepth(llog.InfoLog, 0, "%s", trace)
	l.maybeFlush()
}

// InfoStack logs the current goroutine's stack if the all parameter
// is false, or the stacks of all goroutines if it's true.
func (l *Logger) InfoStack(all bool) {
	infoStack(l, all)
}

func (l *Logger) V(v int) bool {
	return l.log.VDepth(0, llog.Level(v))
}

func (l *Logger) VDepth(depth int, v int) bool {
	return l.log.VDepth(depth, llog.Level(v))
}

type discardInfo struct{}

func (*discardInfo) Info(...interface{})           {}
func (*discardInfo) Infof(string, ...interface{})  {}
func (*discardInfo) InfoDepth(int, ...interface{}) {}
func (*discardInfo) InfoStack(bool)                {}

func (l *Logger) VI(v int) interface {
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
	if l.log.VDepth(0, llog.Level(v)) {
		return l
	}
	return &discardInfo{}
}

func (l *Logger) VIDepth(depth int, v int) interface {
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
	if l.log.VDepth(depth, llog.Level(v)) {
		return l
	}
	return &discardInfo{}
}

// Flush flushes all pending log I/O.
func (l *Logger) FlushLog() {
	l.log.Flush()
}

// Error logs to the ERROR and INFO logs.
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func (l *Logger) Error(args ...interface{}) {
	l.log.Print(llog.ErrorLog, args...)
	l.maybeFlush()
}

// ErrorDepth acts as Error but uses depth to determine which call frame to log.
// A depth of 0 is equivalent to calling Error.
func (l *Logger) ErrorDepth(depth int, args ...interface{}) {
	l.log.PrintDepth(llog.ErrorLog, depth, args...)
	l.maybeFlush()
}

// Errorf logs to the ERROR and INFO logs.
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log.PrintfDepth(llog.ErrorLog, 0, format, args...)
	l.maybeFlush()
}

// Fatal logs to the FATAL, ERROR and INFO logs,
// including a stack trace of all running goroutines, then calls os.Exit(255).
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func (l *Logger) Fatal(args ...interface{}) {
	l.log.Print(llog.FatalLog, args...)
}

// FatalDepth acts as Fatal but uses depth to determine which call frame to log.
// A depth of 0 is equivalent to calling Fatal.
func (l *Logger) FatalDepth(depth int, args ...interface{}) {
	l.log.PrintDepth(llog.FatalLog, depth, args...)
}

// Fatalf logs to the FATAL, ERROR and INFO logs,
// including a stack trace of all running goroutines, then calls os.Exit(255).
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log.PrintfDepth(llog.FatalLog, 0, format, args...)
}

// Panic is equivalent to Error() followed by a call to panic().
func (l *Logger) Panic(args ...interface{}) {
	l.Error(args...)
	panic(fmt.Sprint(args...))
}

// PanicDepth acts as Panic but uses depth to determine which call frame to log.
// A depth of 0 is equivalent to calling Panic.
func (l *Logger) PanicDepth(depth int, args ...interface{}) {
	l.ErrorDepth(depth, args...)
	panic(fmt.Sprint(args...))
}

// Panicf is equivalent to Errorf() followed by a call to panic().
func (l *Logger) Panicf(format string, args ...interface{}) {
	l.Errorf(format, args...)
	panic(fmt.Sprintf(format, args...))
}
