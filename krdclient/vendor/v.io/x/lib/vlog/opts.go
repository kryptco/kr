// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vlog

type LoggingOpts interface {
	LoggingOpt()
}

type AutoFlush bool
type AlsoLogToStderr bool
type LogDir string
type LogToStderr bool
type OverridePriorConfiguration bool
type MaxStackBufSize int

// If true, logs are written to standard error as well as to files.
func (AlsoLogToStderr) LoggingOpt() {}

// Enable V-leveled logging at the specified level.
func (Level) LoggingOpt() {}

// log files will be written to this directory instead of the
// default temporary directory.
func (LogDir) LoggingOpt() {}

// If true, logs are written to standard error instead of to files.
func (LogToStderr) LoggingOpt() {}

// Set the max size (bytes) of the byte buffer to use for stack
// traces. The default max is 4M; use powers of 2 since the
// stack size will be grown exponentially until it exceeds the max.
// A min of 128K is enforced and any attempts to reduce this will
// be silently ignored.
func (MaxStackBufSize) LoggingOpt() {}

// The syntax of the argument is a comma-separated list of pattern=N,
// where pattern is a literal file name (minus the ".go" suffix) or
// "glob" pattern and N is a V level. For instance, gopher*=3
// sets the V level to 3 in all Go files whose names begin "gopher".
func (ModuleSpec) LoggingOpt() {}

// The syntax of the argument is a comma-separated list of regexp=N,
// where pattern is a regular expression matched against the full path name
// of files and N is a V level. For instance, myco.com/web/.*=3
// sets the V level to 3 in all Go files whose path names match myco.com/web/.*".
func (FilepathSpec) LoggingOpt() {}

// Log events at or above this severity are logged to standard
// error as well as to files.
func (StderrThreshold) LoggingOpt() {}

// When set to a file and line number holding a logging statement, such as
//	gopherflakes.go:234
// a stack trace will be written to the Info log whenever execution
// hits that statement. (Unlike with -vmodule, the ".go" must be
// present.)
func (TraceLocation) LoggingOpt() {}

// If true, enables automatic flushing of log output on every call
func (AutoFlush) LoggingOpt() {}

// If true, allows this call to ConfigureLogger to override a prior configuration.
func (OverridePriorConfiguration) LoggingOpt() {}
