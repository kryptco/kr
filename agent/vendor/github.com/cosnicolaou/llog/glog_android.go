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

// +build android

package llog

// #cgo LDFLAGS: -llog
//
// #include <stdlib.h>
// #include <android/log.h>
import "C"

import (
	"bytes"
	"runtime"
	"time"
	"unsafe"
)

const maxLogSize = 1023 // from an off-hand comment in android/log.h

type androidLogger struct {
	prio C.int
	tag  *C.char
}

func (l *androidLogger) Flush() error { return nil }
func (l *androidLogger) Sync() error  { return nil }
func (l *androidLogger) Write(p []byte) (int, error) {
	n := len(p)
	limit := bytes.IndexByte(p, '\n')
	for limit >= 0 {
		l.writeOneLine(p[:limit])
		p = p[limit+1:]
		limit = bytes.IndexByte(p, '\n')
	}
	l.writeOneLine(p)
	return n, nil
}
func (l *androidLogger) writeOneLine(p []byte) {
	cstr := C.CString(string(p))
	C.__android_log_write(l.prio, l.tag, cstr)
	C.free(unsafe.Pointer(cstr))
}

func newFlushSyncWriter(l *Log, s Severity, now time.Time) (flushSyncWriter, error) {
	var prio C.int
	switch {
	case s <= InfoLog:
		prio = C.ANDROID_LOG_INFO
	case s <= WarningLog:
		prio = C.ANDROID_LOG_WARN
	case s <= ErrorLog:
		prio = C.ANDROID_LOG_ERROR
	case s >= FatalLog:
		prio = C.ANDROID_LOG_FATAL
	default:
		prio = C.ANDROID_LOG_DEFAULT
	}
	ret := &androidLogger{
		prio: prio,
		tag:  C.CString(l.name),
	}
	runtime.SetFinalizer(ret, func(l *androidLogger) { C.free(unsafe.Pointer(l.tag)) })
	return ret, nil
}
