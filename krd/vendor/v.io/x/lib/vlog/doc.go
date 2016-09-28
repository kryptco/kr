// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package vlog implements a general-purpose logging system.  It is modeled on
// glog; the differences from glog are:
//
// - interfaces are used to allow for multiple implementations and instances.
// In particular, application and runtime logging can be separated.
// We also expect to stream log messages to external log collectors rather
// to local storage.
//
// - the Warn family of methods are not provided; their main use
// is to avoid the flush that's implicit in the Error routines
// rather than any semantic difference between warnings and errors.
//
// - Info logging and Event logging is separated with the former expected
// to be somewhat spammy and the latter to be used sparingly.
//
// - Event logging includes methods for unconditionally (i.e. regardless
// of any command line options) logging the current goroutine's stack
// or the stacks of all goroutines.
//
// - The use of interfaces and encapsulated state means that a single
// function (V) can no longer be used for 'if guarded' and 'chained' logging.
// That is:
//   if vlog.V(1) { ... } and vlog.V(1).Infof( ... )
// becomes
//   if logger.V(1) { ... }  and logger.VI(1).Infof( ... )
//
// vlog also creates a global instance of the Logger (vlog.Log) and
// provides command line flags (see flags.go). Parsing of these flags is
// performed by calling one of ConfigureLibraryLoggerFromFlags or
// ConfigureLoggerFromFlags .
//
// The supported flags are:
//
//	-logtostderr=false
//		Logs are written to standard error instead of to files.
//	-alsologtostderr=false
//		Logs are written to standard error as well as to files.
//	-stderrthreshold=ERROR
//		Log events at or above this severity are logged to standard
//		error as well as to files.
//	-log_dir=""
//		Log files will be written to this directory instead of the
//		default temporary directory.
//
// Other flags provide aids to debugging:
//
//	-log_backtrace_at=""
//		When set to a file and line number holding a logging statement,
//		such as
//			-log_backtrace_at=gopherflakes.go:234
//		a stack trace will be written to the Info log whenever execution
//		hits that statement. (Unlike with -vmodule, the ".go" must be
//		present.)
//	-v=0
//		Enable V-leveled logging at the specified level.
//	-vmodule=""
//		The syntax of the argument is a comma-separated list of pattern=N,
//		where pattern is a literal file name (minus the ".go" suffix) or
//		"glob" pattern and N is a V level. For instance,
//			-vmodule=gopher*=3
//		sets the V level to 3 in all Go files whose names begin "gopher".
//	-max_stack_buf_size=<size in bytes>
//		Set the max size (bytes) of the byte buffer to use for stack
//		traces. The default max is 4M; use powers of 2 since the
//		stack size will be grown exponentially until it exceeds the max.
//		A min of 128K is enforced and any attempts to reduce this will
//		be silently ignored.
//
package vlog
