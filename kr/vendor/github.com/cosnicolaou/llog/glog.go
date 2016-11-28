// Low-level Go support for leveled logs, analogous to https://code.google.com/p/google-glog/, that avoids the use of global state and command line flags.
//
//
// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package llog implements support for logging analogous to the
// Google-internal C++ INFO/ERROR/V setup.
// It provides a low-level class (llog.T) that encapsulates all state and
// and methods of multi-leveled logging (i.e. Info, Warning, Error and Fatal)
// and formatting variants such as Infof. Each instance of T may be named
// and hence multiple sets of logs can be produced by a single process.
// It also provides V-style logging controlled by the -v and -vmodule=file=2
// style flags. Although command line flags are not directly implemented,
// implementations of the Go 1.2 flags.Value interface are provided to
// make it easy for users to parse their command line flags correctly.
// The NewLogger factory function accepts positional parameters that correspond
// google command line flags.
//
//
//	l := NewLogger("system")
//	l.Print(infoLog, Info("Prepare to repel boarders")
//
//	l.Printf(fatalLog, "Initialization failed: %s", err)
//
// See the documentation for the V function for an explanation of these examples:
//
//	if l.V(2) {
//		l.Print(infoLog,"Starting transaction...")
//	}
//
//
// Log output is buffered and written periodically using Flush. Programs
// should call Flush before exiting to guarantee all log output is written.
//
// By default, all log statements write to files in a temporary directory.
// This package provides several methods that modify this behavior.
// These methods must be called before any logging is done.
//
//	NewLogger(name)
//
//	SetLogDir(logDir)
//		log files will be written to this directory instead of the
//		default temporary directory.
//	SetLogToStderr(bool)
//		If true, logs are written to standard error instead of to files.
//	SetAlsoLogToStderr(bool)
//		If true, logs are written to standard error as well as to files.
//	SetStderrThreshold(level)
//		Log events at or above this severity are logged to standard
//		error as well as to files.
//	SetMaxStackBufSize(size)
//		Set the max size (bytes) of the byte buffer to use for stack
//		traces. The default max is 4096K; use powers of 2 since the
//		stack size will be grown exponentially until it exceeds the max.
//              A min of 128K is enforced and any attempts to reduce this will
//              be silently ignored.
//
//	Other controls provide aids to debugging.
//
//	SetLogBacktraceAt(location)
//		When set to a file and line number holding a logging statement,
//		such as
//			gopherflakes.go:234
//		a stack trace will be written to the Info log whenever execution
//		hits that statement. (Unlike with -vmodule, the ".go" must be
//		present.)
//	SetV(level)
//		Enable V-leveled logging at the specified level.
//	SetVModule(module)
//		The syntax of the argument is a comma-separated list of pattern=N,
//		where pattern is a literal file name (minus the ".go" suffix) or
//		"glob" pattern and N is a V level. For instance,
//			-gopher*=3
//		sets the V level to 3 in all Go files whose names begin "gopher".
// SetVFilepath(regexp)
//      The syntax of the argument is as per VModule, expect that regular
//      expressions on the entire file path path are used instead of glob
//      patterns on the file name component as SetVModule.
//
package llog

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// severity identifies the sort of log: info, warning etc. It also implements
// the flag.Value interface. The -stderrthreshold flag is of type severity and
// should be modified only through the flag.Value interface. The values match
// the corresponding constants in C++.
type Severity int32 // sync/atomic int32

const (
	InfoLog Severity = iota
	WarningLog
	ErrorLog
	FatalLog
	numSeverity = 4

	initialMaxStackBufSize = 128 * 1024
)

const severityChar = "IWEF"

var severityName = []string{
	InfoLog:    "INFO",
	WarningLog: "WARNING",
	ErrorLog:   "ERROR",
	FatalLog:   "FATAL",
}

// get returns the value of the severity.
func (s *Severity) get() Severity {
	return Severity(atomic.LoadInt32((*int32)(s)))
}

// set sets the value of the severity.
func (s *Severity) set(val Severity) {
	atomic.StoreInt32((*int32)(s), int32(val))
}

// String is part of the flag.Value interface.
func (s *Severity) String() string {
	return strconv.FormatInt(int64(*s), 10)
}

// Set is part of the flag.Value interface.
func (s *Severity) Set(value string) error {
	var threshold Severity
	// Is it a known name?
	if v, ok := severityByName(value); ok {
		threshold = v
	} else {
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		threshold = Severity(v)
	}
	*s = threshold
	return nil
}

func severityByName(s string) (Severity, bool) {
	s = strings.ToUpper(s)
	for i, name := range severityName {
		if name == s {
			return Severity(i), true
		}
	}
	return 0, false
}

// OutputStats tracks the number of output lines and bytes written.
type OutputStats struct {
	lines int64
	bytes int64
}

// Lines returns the number of lines written.
func (s *OutputStats) Lines() int64 {
	return atomic.LoadInt64(&s.lines)
}

// Bytes returns the number of bytes written.
func (s *OutputStats) Bytes() int64 {
	return atomic.LoadInt64(&s.bytes)
}

// Stats tracks the number of lines of output and number of bytes
// per severity level.
type Stats struct {
	Info, Warning, Error OutputStats
}

// Level is exported because it appears in the arguments to V and is
// the type of the v flag, which can be set programmatically.
// It's a distinct type because we want to discriminate it from logType.
// Variables of type level are only changed under logging.mu.
// The -v flag is read only with atomic ops, so the state of the logging
// module is consistent.

// Level is treated as a sync/atomic int32.

// Level specifies a level of verbosity for V logs. *Level implements
// flag.Value; the -v flag is of type Level and should be modified
// only through the flag.Value interface.
type Level int32

// get returns the value of the Level.
func (l *Level) get() Level {
	return Level(atomic.LoadInt32((*int32)(l)))
}

// set sets the value of the Level.
func (l *Level) set(val Level) {
	atomic.StoreInt32((*int32)(l), int32(val))
}

// String is part of the flag.Value interface.
func (l *Level) String() string {
	return strconv.FormatInt(int64(*l), 10)
}

// Get is part of the flag.Value interface.
func (l *Level) Get() interface{} {
	return *l
}

// Set is part of the flag.Value interface.
func (l *Level) Set(value string) error {
	v, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*l = Level(v)
	return nil
}

// moduleSpec represents the setting of the -vmodule flag.
type ModuleSpec struct {
	filter []modulePat
}

// modulePat contains a filter for the -vmodule flag.
// It holds a verbosity level and a file pattern to match.
type modulePat struct {
	pattern string
	literal bool // The pattern is a literal string
	level   Level
}

// match reports whether the file matches the pattern. It uses a string
// comparison if the pattern contains no metacharacters.
func (m *modulePat) match(file string) bool {
	if m.literal {
		return file == m.pattern
	}
	match, _ := filepath.Match(m.pattern, file)
	return match
}

// FilepathSpec represents the setting of the -vfilepath flag.
type FilepathSpec struct {
	filter []filepathPat
}

// filepathPat contains a filter for the -vfilepath flags.
type filepathPat struct {
	regexp  *regexp.Regexp
	pattern string
	level   Level
}

// match reports whether the file path matches the regexp.
func (f *filepathPat) match(path string) bool {
	return f.regexp.MatchString(path)
}

func parseFilter(pat string) (string, int, error) {
	if len(pat) == 0 {
		// Empty strings such as from a trailing comma can be ignored.
		return "", 0, nil
	}
	patLev := strings.Split(pat, "=")
	if len(patLev) != 2 || len(patLev[0]) == 0 || len(patLev[1]) == 0 {
		return "", 0, errVmoduleSyntax
	}
	pattern := patLev[0]
	v, err := strconv.Atoi(patLev[1])
	if err != nil {
		return "", 0, errors.New("syntax error: expect comma-separated list of filename=N")
	}
	if v < 0 {
		return "", 0, errors.New("negative value for vmodule level")
	}
	if v == 0 {
		// Ignore. It's harmless but no point in paying the overhead.
		return "", 0, nil
	}
	return pattern, v, nil
}

func (m *ModuleSpec) String() string {
	var b bytes.Buffer
	for i, f := range m.filter {
		if i > 0 {
			b.WriteRune(',')
		}
		fmt.Fprintf(&b, "%s=%d", f.pattern, f.level)
	}
	return b.String()
}

// Get is part of the (Go 1.2)  flag.Getter interface. It always returns nil for this flag type since the
// struct is not exported.
func (m *ModuleSpec) Get() interface{} {
	return nil
}

var errVmoduleSyntax = errors.New("syntax error: expect comma-separated list of filename=N")

// Syntax: recordio=2,file=1,gfs*=3
func (m *ModuleSpec) Set(value string) error {
	var filter []modulePat
	for _, pat := range strings.Split(value, ",") {
		pattern, v, err := parseFilter(pat)
		if err != nil {
			return err
		}
		if v == 0 {
			continue
		}
		// TODO: check syntax of filter?
		filter = append(filter, modulePat{pattern, isLiteralGlob(pattern), Level(v)})
	}
	m.filter = filter
	return nil
}

func (fp *FilepathSpec) String() string {
	var b bytes.Buffer
	for i, f := range fp.filter {
		if i > 0 {
			b.WriteRune(',')
		}
		fmt.Fprintf(&b, "%s=%d", f.pattern, f.level)
	}
	return b.String()
}

// Get is part of the (Go 1.2)  flag.Getter interface. It always returns nil for this flag type since the
// struct is not exported.
func (p *FilepathSpec) Get() interface{} {
	return nil
}

var errVpackageSyntax = errors.New("syntax error: expect comma-separated list of regexp=N")

// Syntax: foo/bar=2,foo/bar/.*=1,f*=3
func (p *FilepathSpec) Set(value string) error {
	var filter []filepathPat
	for _, pat := range strings.Split(value, ",") {
		pattern, v, err := parseFilter(pat)
		if err != nil {
			return err
		}
		if v == 0 {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("failed to compile %s as an regexp: %s", pattern, err)
		}
		// TODO: check syntax of filter?
		filter = append(filter, filepathPat{re, pattern, Level(v)})
	}
	p.filter = filter
	return nil
}

// isLiteralGlob reports whether the pattern is a literal string, that is,
// has no metacharacters that require filepath.Match to be called to match
// the pattern.
func isLiteralGlob(pattern string) bool {
	return !strings.ContainsAny(pattern, `*?[]\`)
}

// traceLocation represents the setting of the -log_backtrace_at flag.
type TraceLocation struct {
	file string
	line int
}

// isSet reports whether the trace location has been specified.
// logging.mu is held.
func (t *TraceLocation) isSet() bool {
	return t.line > 0
}

// match reports whether the specified file and line matches the trace location.
// The argument file name is the full path, not the basename specified in the flag.
func (t *TraceLocation) match(file string, line int) bool {
	if t.line != line {
		return false
	}
	if i := strings.LastIndex(file, "/"); i >= 0 {
		file = file[i+1:]
	}
	return t.file == file
}

func (t *TraceLocation) String() string {
	// Lock because the type is not atomic. TODO: clean this up.
	return fmt.Sprintf("%s:%d", t.file, t.line)
}

var errTraceSyntax = errors.New("syntax error: expect file.go:234")

// Syntax: gopherflakes.go:234
// Note that unlike vmodule the file extension is included here.
func (t *TraceLocation) Set(value string) error {
	if value == "" {
		// Unset.
		t.line = 0
		t.file = ""
	}
	fields := strings.Split(value, ":")
	if len(fields) != 2 {
		return errTraceSyntax
	}
	file, line := fields[0], fields[1]
	if !strings.Contains(file, ".") {
		return errTraceSyntax
	}
	v, err := strconv.Atoi(line)
	if err != nil {
		return errTraceSyntax
	}
	if v <= 0 {
		return errors.New("negative or zero value for level")
	}
	t.line = v
	t.file = file
	return nil
}

// flushSyncWriter is the interface satisfied by logging destinations.
type flushSyncWriter interface {
	Flush() error
	Sync() error
	io.Writer
}

// Log collects all the global state of the logging setup.
type Log struct {
	// the name of this logger (appears in the name of each log file)
	name string

	// logDirs lists the candidate directories for new log files.
	logDirs []string

	// Boolean flags. Not handled atomically because the flag.Value interface
	// does not let us avoid the =true, and that shorthand is necessary for
	// compatibility. TODO: does this matter enough to fix? Seems unlikely.
	toStderr     bool // The -logtostderr flag.
	alsoToStderr bool // The -alsologtostderr flag.

	// Level flag. Handled atomically.
	stderrThreshold Severity // The -stderrthreshold flag.

	// freeList is a list of byte buffers, maintained under freeListMu.
	freeList *buffer

	// freeListMu maintains the free list. It is separate from the main mutex
	// so buffers can be grabbed and printed to without holding the main lock,
	// for better parallelization.
	freeListMu sync.Mutex

	// mu protects the remaining elements of this structure and is
	// used to synchronize logging.
	mu sync.Mutex

	// file holds writer for each of the log types.
	file [numSeverity]flushSyncWriter

	// pcs is used in V to avoid an allocation when computing the caller's PC.
	pcs [1]uintptr

	// vmap is a cache of the V Level for each V() call site, identified by PC.
	// It is wiped whenever the vmodule flag changes state.
	vmap map[uintptr]Level

	// filterLength stores the length of the vmodule filter chain. If greater
	// than zero, it means vmodule is enabled. It may be read safely
	// using sync.LoadInt32, but is only modified under mu.
	filterLength int32

	// traceLocation is the state of the -log_backtrace_at flag.
	traceLocation TraceLocation
	// These flags are modified only under lock, although verbosity may be fetched
	// safely using atomic.LoadInt32.
	vmodule   ModuleSpec   // The state of the -vmodule flag.
	vfilepath FilepathSpec // The state of the -vfilepath flag.
	verbosity Level        // V logging level, the value of the -v flag/

	// track lines/bytes per severity level
	stats         *Stats
	severityStats [numSeverity]*OutputStats

	// number of stack frame to skip in order to reach the callpoint
	// to be logged. skip is calculated as per runtime.Caller.
	skip int

	// max size of buffer to use for stacks.
	maxStackBufSize int
}

// NewLogger creates a new logger.
// name is a non-empty string that appears in the names of log files
// to distinguish between separate instances of the logger writing to the
// same directory.
// skip is the number of stack frames to skip in order to reach the
// call point to be logged. 0 will log the caller of the logging methods,
// 1 their caller etc.
func NewLogger(name string, skip int) *Log {
	logging := &Log{stats: new(Stats)}
	logging.setVState(0, nil, nil, false)
	logging.skip = 2 + skip
	logging.maxStackBufSize = 4096 * 1024
	logging.name = name

	// Default stderrThreshold is ERROR.
	logging.stderrThreshold = ErrorLog
	logging.setVState(0, nil, nil, false)

	logging.severityStats[InfoLog] = &logging.stats.Info
	logging.severityStats[WarningLog] = &logging.stats.Warning
	logging.severityStats[ErrorLog] = &logging.stats.Error

	logging.logDirs = append(logging.logDirs, os.TempDir())
	go logging.flushDaemon()
	return logging
}

func (l *Log) String() string {
	return fmt.Sprintf("name=%s logdirs=%s logtostderr=%t alsologtostderr=%t max_stack_buf_size=%d v=%d stderrthreshold=%s vmodule=%s vfilepath=%s log_backtrace_at=%s",
		l.name, l.logDirs, l.toStderr, l.alsoToStderr, l.maxStackBufSize, l.verbosity, &l.stderrThreshold, &l.vmodule, &l.vfilepath, &l.traceLocation)
}

// logDir if non-empty, write log files to this directory.
func (l *Log) SetLogDir(logDir string) {
	if logDir != "" {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.logDirs = append([]string{logDir}, l.logDirs...)
	}
}

// SetLogToStderr sets the flag that, if true, logs to standard error instead of files
func (l *Log) SetLogToStderr(f bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.toStderr = f

}

// SetAlsoLogToStderr sets the flag that, if true, logs to standard error as well as files
func (l *Log) SetAlsoLogToStderr(f bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.alsoToStderr = f

}

// SetV sets the log level for V logs
func (l *Log) SetV(v Level) {
	l.verbosity.set(v)
}

// SetStderrThreshold sets the threshold for which logs at or above which go to stderr
func (l *Log) SetStderrThreshold(s Severity) {
	l.stderrThreshold.set(s)
}

// SetModuleSpec sets the comma-separated list of pattern=N settings for
// file-filtered logging
func (l *Log) SetVModule(spec ModuleSpec) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.setVState(l.verbosity, spec.filter, nil, true)
}

// SetModuleSpec sets the comma-separated list of pattern=N settings for
// file-filtered logging
func (l *Log) SetVFilepath(spec FilepathSpec) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.setVState(l.verbosity, nil, spec.filter, true)
}

// SetTaceLocation sets the location, file:N, which when encountered will cause logging to emit a stack trace
func (l *Log) SetTraceLocation(location TraceLocation) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.traceLocation = location
}

func (l *Log) SetMaxStackBufSize(max int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if max > initialMaxStackBufSize {
		l.maxStackBufSize = max
	}
}

// buffer holds a byte Buffer for reuse. The zero value is ready for use.
type buffer struct {
	bytes.Buffer
	tmp  [64]byte // temporary byte array for creating headers.
	next *buffer
}

// setVState sets a consistent state for V logging. A nil value for
// modules or filepaths will result in that filter not being changed.
// l.mu is held.
func (l *Log) setVState(verbosity Level, modules []modulePat, filepaths []filepathPat, setFilter bool) {
	// Turn verbosity off so V will not fire while we are in transition.
	l.verbosity.set(0)
	// Ditto for filter length.
	atomic.StoreInt32(&l.filterLength, 0)

	// Set the new filters and wipe the pc->Level map if the filter has changed.
	nfilters := 0
	if setFilter {
		if modules != nil {
			l.vmodule.filter = modules
		}
		if filepaths != nil {
			l.vfilepath.filter = filepaths
		}
		nfilters = len(l.vmodule.filter) + len(l.vfilepath.filter)
		l.vmap = make(map[uintptr]Level)
	}

	// Things are consistent now, so enable filtering and verbosity.
	// They are enabled in order opposite to that in V.
	atomic.StoreInt32(&l.filterLength, int32(nfilters))
	l.verbosity.set(verbosity)
}

// getBuffer returns a new, ready-to-use buffer.
func (l *Log) getBuffer() *buffer {
	l.freeListMu.Lock()
	b := l.freeList
	if b != nil {
		l.freeList = b.next
	}
	l.freeListMu.Unlock()
	if b == nil {
		b = new(buffer)
	} else {
		b.next = nil
		b.Reset()
	}
	return b
}

// putBuffer returns a buffer to the free list.
func (l *Log) putBuffer(b *buffer) {
	if b.Len() >= 256 {
		// Let big buffers die a natural death.
		return
	}
	l.freeListMu.Lock()
	b.next = l.freeList
	l.freeList = b
	l.freeListMu.Unlock()
}

var timeNow = time.Now // Stubbed out for testing.

/*
header formats a log header as defined by the C++ implementation.
It returns a buffer containing the formatted header.
The depth specifies how many stack frames above lives the source line to be identified in the log message.

Log lines have this form:
	Lmmdd hh:mm:ss.uuuuuu threadid file:line] msg...
where the fields are defined as follows:
	L                A single character, representing the log level (eg 'I' for INFO)
	mm               The month (zero padded; ie May is '05')
	dd               The day (zero padded)
	hh:mm:ss.uuuuuu  Time in hours, minutes and fractional seconds
	threadid         The space-padded thread ID as returned by GetTID()
	file             The file name
	line             The line number
	msg              The user-supplied message
*/
func (l *Log) header(s Severity, depth int) (*buffer, string, int) {
	// Lmmdd hh:mm:ss.uuuuuu threadid file:line]
	now := timeNow()
	_, file, line, ok := runtime.Caller(l.skip + depth)
	if !ok {
		file = "???"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		if slash >= 0 {
			file = file[slash+1:]
		}
	}
	if line < 0 {
		line = 0 // not a real line number, but acceptable to someDigits
	}
	if s > FatalLog {
		s = InfoLog // for safety.
	}
	buf := l.getBuffer()

	// Avoid Fprintf, for speed. The format is so simple that we can do it quickly by hand.
	// It's worth about 3X. Fprintf is hard.
	_, month, day := now.Date()
	hour, minute, second := now.Clock()
	buf.tmp[0] = severityChar[s]
	buf.twoDigits(1, int(month))
	buf.twoDigits(3, day)
	buf.tmp[5] = ' '
	buf.twoDigits(6, hour)
	buf.tmp[8] = ':'
	buf.twoDigits(9, minute)
	buf.tmp[11] = ':'
	buf.twoDigits(12, second)
	buf.tmp[14] = '.'
	buf.nDigits(6, 15, now.Nanosecond()/1000, '0')
	buf.tmp[21] = ' '
	buf.nDigits(7, 22, pid, ' ') // TODO: should be TID
	buf.tmp[29] = ' '
	buf.Write(buf.tmp[:30])
	buf.WriteString(file)
	buf.tmp[0] = ':'
	n := buf.someDigits(1, line)
	buf.tmp[n+1] = ']'
	buf.tmp[n+2] = ' '
	buf.Write(buf.tmp[:n+3])
	return buf, file, line
}

// Some custom tiny helper functions to print the log header efficiently.

const digits = "0123456789"

// twoDigits formats a zero-prefixed two-digit integer at buf.tmp[i].
func (buf *buffer) twoDigits(i, d int) {
	buf.tmp[i+1] = digits[d%10]
	d /= 10
	buf.tmp[i] = digits[d%10]
}

// nDigits formats an n-digit integer at buf.tmp[i],
// padding with pad on the left.
// It assumes d >= 0.
func (buf *buffer) nDigits(n, i, d int, pad byte) {
	j := n - 1
	for ; j >= 0 && d > 0; j-- {
		buf.tmp[i+j] = digits[d%10]
		d /= 10
	}
	for ; j >= 0; j-- {
		buf.tmp[i+j] = pad
	}
}

// someDigits formats a zero-prefixed variable-width integer at buf.tmp[i].
func (buf *buffer) someDigits(i, d int) int {
	// Print into the top, then copy down. We know there's space for at least
	// a 10-digit number.
	j := len(buf.tmp)
	for {
		j--
		buf.tmp[j] = digits[d%10]
		d /= 10
		if d == 0 {
			break
		}
	}
	return copy(buf.tmp[i:], buf.tmp[j:])
}

func (l *Log) Println(s Severity, args ...interface{}) {
	l.PrintlnDepth(s, 1, args...)
}

func (l *Log) Print(s Severity, args ...interface{}) {
	l.PrintDepth(s, 1, args...)
}

func (l *Log) Printf(s Severity, format string, args ...interface{}) {
	l.PrintfDepth(s, 1, format, args...)
}

func (l *Log) PrintlnDepth(s Severity, depth int, args ...interface{}) {
	buf, file, line := l.header(s, depth)
	fmt.Fprintln(buf, args...)
	l.output(s, buf, file, line)
}

func (l *Log) PrintDepth(s Severity, depth int, args ...interface{}) {
	buf, file, line := l.header(s, depth)
	fmt.Fprint(buf, args...)
	if buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	l.output(s, buf, file, line)
}

func (l *Log) PrintfDepth(s Severity, depth int, format string, args ...interface{}) {
	buf, file, line := l.header(s, depth)
	fmt.Fprintf(buf, format, args...)
	if buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	l.output(s, buf, file, line)
}

// output writes the data to the log files and releases the buffer.
func (l *Log) output(s Severity, buf *buffer, file string, line int) {
	l.mu.Lock()
	if l.traceLocation.isSet() {
		if l.traceLocation.match(file, line) {
			buf.Write(stacks(false, l.maxStackBufSize))
		}
	}
	data := buf.Bytes()
	if l.toStderr {
		os.Stderr.Write(data)
	} else {
		if l.alsoToStderr || s >= l.stderrThreshold.get() {
			os.Stderr.Write(data)
		}
		if l.file[s] == nil {
			if err := l.createFiles(s); err != nil {
				os.Stderr.Write(data) // Make sure the message appears somewhere.
				l.exit(err)
			}
		}
		switch s {
		case FatalLog:
			l.file[FatalLog].Write(data)
			fallthrough
		case ErrorLog:
			l.file[ErrorLog].Write(data)
			fallthrough
		case WarningLog:
			l.file[WarningLog].Write(data)
			fallthrough
		case InfoLog:
			l.file[InfoLog].Write(data)
		}
	}
	if s == FatalLog {
		// Make sure we see the trace for the current goroutine on standard error.
		if !l.toStderr {
			os.Stderr.Write(stacks(false, l.maxStackBufSize))
		}
		// Write the stack trace for all goroutines to the files.
		trace := stacks(true, l.maxStackBufSize)
		logExitFunc = func(error) {} // If we get a write error, we'll still exit below.
		for log := FatalLog; log >= InfoLog; log-- {
			if f := l.file[log]; f != nil { // Can be nil if -logtostderr is set.
				f.Write(trace)
			}
		}
		l.mu.Unlock()
		timeoutFlush(l, 10*time.Second)
		os.Exit(255) // C++ uses -1, which is silly because it's anded with 255 anyway.
	}
	l.putBuffer(buf)
	l.mu.Unlock()
	if stats := l.severityStats[s]; stats != nil {
		atomic.AddInt64(&stats.lines, 1)
		atomic.AddInt64(&stats.bytes, int64(len(data)))
	}
}

// timeoutFlush calls Flush and returns when it completes or after timeout
// elapses, whichever happens first.  This is needed because the hooks invoked
// by Flush may deadlock when glog.Fatal is called from a hook that holds
// a lock.
func timeoutFlush(l *Log, timeout time.Duration) {
	done := make(chan bool, 1)
	go func() {
		l.lockAndFlushAll()
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		fmt.Fprintln(os.Stderr, "glog: Flush took longer than", timeout)
	}
}

// stacks is a wrapper for runtime.Stack that attempts to recover the data for all goroutines.
func stacks(all bool, max int) []byte {
	// We don't know how big the traces are, so grow a few times if they don't fit. Start large, though.
	n := initialMaxStackBufSize
	var trace []byte
	for n <= max {
		trace = make([]byte, n)
		nbytes := runtime.Stack(trace, all)
		if nbytes < len(trace) {
			return trace[:nbytes]
		}
		n *= 2
	}
	return trace
}

// logExitFunc provides a simple mechanism to override the default behavior
// of exiting on error. Used in testing and to guarantee we reach a required exit
// for fatal logs. Instead, exit could be a function rather than a method but that
// would make its use clumsier.
var logExitFunc func(error)

// exit is called if there is trouble creating or writing log files.
// It flushes the logs and exits the program; there's no point in hanging around.
// l.mu is held.
func (l *Log) exit(err error) {
	fmt.Fprintf(os.Stderr, "log: exiting because of error: %s\n", err)
	// If logExitFunc is set, we do that instead of exiting.
	if logExitFunc != nil {
		logExitFunc(err)
		return
	}
	l.flushAll()
	os.Exit(2)
}

// syncBuffer joins a bufio.Writer to its underlying file, providing access to the
// file's Sync method and providing a wrapper for the Write method that provides log
// file rotation. There are conflicting methods, so the file cannot be embedded.
// l.mu is held for all its methods.
type syncBuffer struct {
	logger *Log
	*bufio.Writer
	file   *os.File
	sev    Severity
	nbytes uint64 // The number of bytes written to this file
}

func (sb *syncBuffer) Sync() error {
	return sb.file.Sync()
}

func (sb *syncBuffer) Write(p []byte) (n int, err error) {
	if sb.nbytes+uint64(len(p)) >= MaxSize {
		if err := sb.rotateFile(time.Now()); err != nil {
			sb.logger.exit(err)
		}
	}
	n, err = sb.Writer.Write(p)
	sb.nbytes += uint64(n)
	if err != nil {
		sb.logger.exit(err)
	}
	return
}

// rotateFile closes the syncBuffer's file and starts a new one.
func (sb *syncBuffer) rotateFile(now time.Time) error {
	if sb.file != nil {
		sb.Flush()
		sb.file.Close()
	}
	var err error
	sb.file, _, err = sb.logger.create(severityName[sb.sev], now)
	sb.nbytes = 0
	if err != nil {
		return err
	}

	sb.Writer = bufio.NewWriterSize(sb.file, bufferSize)

	// Write header.
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Log file created at: %s\n", now.Format("2006/01/02 15:04:05"))
	fmt.Fprintf(&buf, "Running on machine: %s\n", host)
	fmt.Fprintf(&buf, "Binary: Built with %s %s for %s/%s\n", runtime.Compiler, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&buf, "Log line format: [IWEF]mmdd hh:mm:ss.uuuuuu threadid file:line] msg\n")
	n, err := sb.file.Write(buf.Bytes())
	sb.nbytes += uint64(n)
	return err
}

func newSyncBuffer(l *Log, s Severity, now time.Time) (flushSyncWriter, error) {
	sb := &syncBuffer{
		logger: l,
		sev:    s,
	}
	return sb, sb.rotateFile(now)
}

// bufferSize sizes the buffer associated with each log file. It's large
// so that log records can accumulate without the logging thread blocking
// on disk I/O. The flushDaemon will block instead.
const bufferSize = 256 * 1024

// createFiles creates all the log files for severity from sev down to infoLog.
// l.mu is held.
func (l *Log) createFiles(sev Severity) error {
	now := time.Now()
	// Files are created in decreasing severity order, so as soon as we find one
	// has already been created, we can stop.
	for s := sev; s >= InfoLog && l.file[s] == nil; s-- {
		w, err := newFlushSyncWriter(l, s, now)
		if err != nil {
			return err
		}
		l.file[s] = w
	}
	return nil
}

const flushInterval = 30 * time.Second

// flushDaemon periodically flushes the log file buffers.
func (l *Log) flushDaemon() {
	for _ = range time.NewTicker(flushInterval).C {
		l.lockAndFlushAll()
	}
}

// lockAndFlushAll is like flushAll but locks l.mu first.
func (l *Log) lockAndFlushAll() {
	l.mu.Lock()
	l.flushAll()
	l.mu.Unlock()
}

func (l *Log) Flush() {
	l.lockAndFlushAll()
}

// flushAll flushes all the logs and attempts to "sync" their data to disk.
// l.mu is held.
func (l *Log) flushAll() {
	// Flush from fatal down, in case there's trouble flushing.
	for s := FatalLog; s >= InfoLog; s-- {
		file := l.file[s]
		if file != nil {
			file.Flush() // ignore error
			file.Sync()  // ignore error
		}
	}
}

// setV computes and remembers the V level for a given PC
// when vmodule is enabled.
// File pattern matching takes the basename of the file, stripped
// of its .go suffix, and uses 270.Match, which is a little more
// general than the *? matching used in C++.
// l.mu is held.
func (l *Log) setV(pc uintptr) Level {
	fn := runtime.FuncForPC(pc)
	file, _ := fn.FileLine(pc)
	// The file is something like /a/b/c/d.go. We want just the d.
	if strings.HasSuffix(file, ".go") {
		file = file[:len(file)-3]
	}
	module := file
	if slash := strings.LastIndex(file, "/"); slash >= 0 {
		module = file[slash+1:]
	}
	for _, filter := range l.vmodule.filter {
		if filter.match(module) {
			l.vmap[pc] = filter.level
			return filter.level
		}
	}
	for _, filter := range l.vfilepath.filter {
		if filter.match(file) {
			l.vmap[pc] = filter.level
			return filter.level
		}
	}
	l.vmap[pc] = 0
	return 0
}

// V reports whether verbosity at the call site is at least the requested level.
// The returned value is a boolean of type Verbose, which implements Info, Infoln
// and Infof. These methods will write to the Info log if called.
// Thus, one may write either
//	if glog.V(2) { glog.Info("log this") }
// or
//	glog.V(2).Info("log this")
// The second form is shorter but the first is cheaper if logging is off because it does
// not evaluate its arguments.
//
// Whether an individual call to V generates a log record depends on the setting of
// the -v and --vmodule flags; both are off by default. If the level in the call to
// V is at least the value of -v, or of -vmodule for the source file containing the
// call, the V call will log.
func (l *Log) VDepth(depth int, level Level) bool {
	// This function tries hard to be cheap unless there's work to do.
	// The fast path is two atomic loads and compares.

	// Here is a cheap but safe test to see if V logging is enabled globally.
	if l.verbosity.get() >= level {
		return true
	}

	// It's off globally but it vmodule may still be set.
	// Here is another cheap but safe test to see if vmodule is enabled.
	if atomic.LoadInt32(&l.filterLength) > 0 {
		// Now we need a proper lock to use the logging structure. The pcs field
		// is shared so we must lock before accessing it. This is fairly expensive,
		// but if V logging is enabled we're slow anyway.
		l.mu.Lock()
		defer l.mu.Unlock()
		// Note that runtime.Callers counts skip differently to
		// runtime.Caller - i.e. it is one greater than the skip
		// value for .Caller to reach the same stack frame. So, even though
		// we are one level closer to the caller here, we will still use the
		// same value as for runtime.Caller!
		if runtime.Callers(l.skip+depth, l.pcs[:]) == 0 {
			return false
		}
		v, ok := l.vmap[l.pcs[0]]
		if !ok {
			v = l.setV(l.pcs[0])
		}
		return v >= level
	}
	return false
}

func (l *Log) V(level Level) bool {
	// Here is a cheap but safe test to see if V logging is enabled globally.
	if l.verbosity.get() >= level {
		return true
	}

	// It's off globally but it vmodule may still be set.
	// Here is another cheap but safe test to see if vmodule is enabled.
	if atomic.LoadInt32(&l.filterLength) == 0 {
		return false
	}

	// Only call VDepth when there's work for it to do.
	return l.VDepth(1, level)
}

func (l *Log) Stats() Stats {
	return *l.stats
}
