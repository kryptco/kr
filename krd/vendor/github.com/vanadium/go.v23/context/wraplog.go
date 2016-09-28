// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package context

import "fmt"

func (t *T) Info(args ...interface{}) {
	t.logger.InfoDepth(1, args...)
	if t.ctxLogger != nil {
		t.ctxLogger.InfoDepth(t, 1, args...)
	}
}
func (t *T) InfoDepth(depth int, args ...interface{}) {
	t.logger.InfoDepth(depth+1, args...)
	if t.ctxLogger != nil {
		t.ctxLogger.InfoDepth(t, depth+1, args...)
	}
}

func (t *T) Infof(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	t.logger.InfoDepth(1, line)
	if t.ctxLogger != nil {
		t.ctxLogger.InfoDepth(t, 1, line)
	}
}

func (t *T) InfoStack(all bool) {
	t.logger.InfoStack(all)
	if t.ctxLogger != nil {
		t.ctxLogger.InfoStack(t, all)
	}
}

func (t *T) Error(args ...interface{}) {
	t.logger.ErrorDepth(1, args...)
	if t.ctxLogger != nil {
		t.ctxLogger.InfoDepth(t, 1, args...)
	}
}
func (t *T) ErrorDepth(depth int, args ...interface{}) {
	t.logger.ErrorDepth(depth+1, args...)
	if t.ctxLogger != nil {
		t.ctxLogger.InfoDepth(t, depth+1, args...)
	}
}
func (t *T) Errorf(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	t.logger.ErrorDepth(1, line)
	if t.ctxLogger != nil {
		t.ctxLogger.InfoDepth(t, 1, line)
	}
}

func (t *T) Fatal(args ...interface{}) {
	t.logger.FatalDepth(1, args...)
}
func (t *T) FatalDepth(depth int, args ...interface{}) {
	t.logger.FatalDepth(depth+1, args...)
}
func (t *T) Fatalf(format string, args ...interface{}) {
	t.logger.FatalDepth(1, fmt.Sprintf(format, args...))
}

func (t *T) Panic(args ...interface{}) {
	t.logger.PanicDepth(1, args...)
}

func (t *T) PanicDepth(depth int, args ...interface{}) {
	t.logger.PanicDepth(depth+1, args...)
}

func (t *T) Panicf(format string, args ...interface{}) {
	t.logger.PanicDepth(1, fmt.Sprintf(format, args...))
}

func (t *T) V(level int) bool {
	if t.logger.VDepth(1, level) {
		return true
	}
	return t.ctxLogger != nil && t.ctxLogger.VDepth(t, 1, level)
}

func (t *T) VDepth(depth int, level int) bool {
	if t.logger.VDepth(depth+1, level) {
		return true
	}
	return t.ctxLogger != nil && t.ctxLogger.VDepth(t, depth+1, level)
}

type viLogger struct {
	ctx    *T
	logger interface {
		Info(args ...interface{})
		Infof(format string, args ...interface{})
		InfoDepth(depth int, args ...interface{})
		InfoStack(all bool)
	}
	ctxLogger ContextLogger
}

func (v *viLogger) Info(args ...interface{}) {
	v.logger.InfoDepth(1, args...)
	if v.ctxLogger != nil {
		v.ctxLogger.InfoDepth(v.ctx, 1, args...)
	}
}

func (v *viLogger) Infof(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	v.logger.InfoDepth(1, line)
	if v.ctxLogger != nil {
		v.ctxLogger.InfoDepth(v.ctx, 1, line)
	}
}

func (v *viLogger) InfoDepth(depth int, args ...interface{}) {
	v.logger.InfoDepth(depth+1, args...)
	if v.ctxLogger != nil {
		v.ctxLogger.InfoDepth(v.ctx, depth+1, args...)
	}
}

func (v *viLogger) InfoStack(all bool) {
	v.logger.InfoStack(all)
	if v.ctxLogger != nil {
		v.ctxLogger.InfoStack(v.ctx, all)
	}
}

func (t *T) VI(level int) interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	InfoDepth(depth int, args ...interface{})
	InfoStack(all bool)
} {
	out := &viLogger{
		ctx:    t,
		logger: t.logger.VIDepth(1, level),
	}
	if t.ctxLogger != nil {
		out.ctxLogger = t.ctxLogger.VIDepth(t, 1, level)
	}
	return out
}
func (t *T) VIDepth(depth int, level int) interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	InfoDepth(depth int, args ...interface{})
	InfoStack(all bool)
} {
	out := &viLogger{
		ctx:    t,
		logger: t.logger.VIDepth(depth+1, level),
	}
	if t.ctxLogger != nil {
		out.ctxLogger = t.ctxLogger.VIDepth(t, depth+11, level)
	}
	return out
}

func (t *T) FlushLog() { t.logger.FlushLog() }
