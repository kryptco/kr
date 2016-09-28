// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package context implements a mechanism to carry data across API boundaries.
// The context.T struct carries deadlines and cancellation as well as other
// arbitrary values.
//
// Application code receives contexts in two main ways:
//
// 1) A context.T is returned from v23.Init().  This will generally be
// used to set up servers in main, or for stand-alone client programs.
//    func main() {
//      ctx, shutdown := v23.Init()
//      defer shutdown()
//
//      doSomething(ctx)
//    }
//
// 2) A context.T is passed to every server method implementation as the first
// parameter.
//    func (m *myServer) Method(ctx *context.T, call rpc.ServerCall) error {
//      doSomething(ctx)
//    }
//
// Once you have a context you can derive further contexts to change settings.
// for example to adjust a deadline you might do:
//    func main() {
//      ctx, shutdown := v23.Init()
//      defer shutdown()
//      // We'll use cacheCtx to lookup data in memcache
//      // if it takes more than a second to get data from
//      // memcache we should just skip the cache and perform
//      // the slow operation.
//      cacheCtx, cancel := WithTimeout(ctx, time.Second)
//      if err := FetchDataFromMemcache(cacheCtx, key); err != nil {
//        // Here we use the original ctx, not the derived cacheCtx
//        // so we aren't constrained by the 1 second timeout.
//        RecomputeData(ctx, key)
//      }
//    }
//
// Contexts form a tree where derived contexts are children of the
// contexts from which they were derived.  Children inherit all the
// properties of their parent except for the property being replaced
// (the deadline in the example above).
//
// Contexts are extensible.  The Value/WithValue functions allow you to attach
// new information to the context and extend its capabilities.
// In the same way we derive new contexts via the 'With' family of functions
// you can create methods to attach new data:
//
//    package auth
//
//    import "v.io/v23/context"
//
//    type Auth struct{...}
//
//    type key int
//    const authKey = key(0)
//
//    function WithAuth(parent *context.T, data *Auth) *context.T {
//        return context.WithValue(parent, authKey, data)
//    }
//
//    function GetAuth(ctx *context.T) *Auth {
//        data, _ := ctx.Value(authKey).(*Auth)
//        return data
//    }
//
// Note that a value of any type can be used as a key, but you should
// use an unexported value of an unexported type to ensure that no
// collisions can occur.
package context

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"v.io/v23/logging"
)

type internalKey int

const (
	rootKey = internalKey(iota)
	cancelKey
	deadlineKey
)

// ContextLogger is a logger that uses a passed in T to configure
// the logging behavior.
type ContextLogger interface {
	// InfoDepth logs to the INFO log. depth is used to determine which call frame to log.
	InfoDepth(ctx *T, depth int, args ...interface{})

	// InfoStack logs the current goroutine's stack if the all parameter
	// is false, or the stacks of all goroutines if it's true.
	InfoStack(ctx *T, all bool)

	// VDepth returns true if the configured logging level is greater than or equal to its parameter. depth
	// is used to determine which call frame to test against.
	VDepth(ctx *T, depth int, level int) bool

	// VIDepth is like VDepth, except that it returns nil if there level is greater than the
	// configured log level.
	VIDepth(ctx *T, depth int, level int) ContextLogger
}

// CancelFunc is used to cancel a context.  The first call will
// cause the paired context and all decendants to close their Done()
// channels.  Further calls do nothing.
type CancelFunc func()

// Cancelled is returned by contexts which have been cancelled.
var Canceled = errors.New("context canceled")

// DeadlineExceeded is returned by contexts that have exceeded their
// deadlines and therefore been canceled automatically.
var DeadlineExceeded = errors.New("context deadline exceeded")

// T carries deadlines, cancellation and data across API boundaries.
// It is safe to use a T from multiple goroutines simultaneously.  The
// zero-type of context is uninitialized and will panic if used
// directly by application code. It also implements v23/logging.Logger and
// hence can be used directly for logging (e.g. ctx.Infof(...)).
type T struct {
	parent     *T
	logger     logging.Logger
	ctxLogger  ContextLogger
	key, value interface{}
}

// RootContext creates a new root context with no data attached.
// A RootContext is cancelable (see WithCancel).
// Typically you should not call this function, instead you should derive
// contexts from other contexts, such as the context returned from v23.Init
// or the result of the Context() method on a ServerCall.  This function
// is sometimes useful in tests, where it is undesirable to initialize a
// runtime to test a function that reads from a T.
func RootContext() (*T, CancelFunc) {
	return WithCancel(&T{logger: logging.Discard, key: rootKey})
}

// WithLogger returns a child of the current context that embeds the supplied
// logger.
func WithLogger(parent *T, logger logging.Logger) *T {
	child := *parent
	child.logger = logger
	return &child
}

// WithContextLogger returns a child of the current context that embeds the supplied
// context logger
func WithContextLogger(parent *T, logger ContextLogger) *T {
	child := *parent
	child.ctxLogger = logger
	return &child
}

// LoggerImplementation returns the implementation of the logger associated
// with this context. It should almost never need to be used by application
// code.
func (t *T) LoggerImplementation() interface{} {
	return t.logger
}

// Initialized returns true if this context has been properly initialized
// by a runtime.
func (t *T) Initialized() bool {
	return t != nil && t.key != nil
}

// Value is used to carry data across API boundaries.  This should be
// used only for data that is relevant across multiple API boundaries
// and not just to pass extra parameters to functions and methods.
// Any type that supports equality can be used as a key, but an
// unexported type should be used to prevent collisions.
func (t *T) Value(key interface{}) interface{} {
	for t != nil {
		if key == t.key {
			return t.value
		}
		t = t.parent
	}
	return nil
}

// Deadline returns the time at which this context will be automatically
// canceled.
func (t *T) Deadline() (deadline time.Time, ok bool) {
	if deadline, ok := t.Value(deadlineKey).(*deadlineState); ok {
		return deadline.deadline, true
	}
	return
}

// After the channel returned by Done() is closed, Err() will return
// either Canceled or DeadlineExceeded.
func (t *T) Err() error {
	if cancel, ok := t.Value(cancelKey).(*cancelState); ok {
		cancel.mu.Lock()
		defer cancel.mu.Unlock()
		return cancel.err
	}
	return nil
}

// Done returns a channel which will be closed when this context.T
// is canceled or exceeds its deadline.  Successive calls will
// return the same value.  Implementations may return nil if they can
// never be canceled.
func (t *T) Done() <-chan struct{} {
	if cancel, ok := t.Value(cancelKey).(*cancelState); ok {
		return cancel.done
	}
	return nil
}

// cancelState helps pass cancellation down the context tree.
type cancelState struct {
	done chan struct{}

	mu       sync.Mutex
	err      error                 // GUARDED_BY(mu)
	children map[*cancelState]bool // GUARDED_BY(mu)
}

// A leakCheck is used to point from the cancel() func to cancelState.
// If leakedContextPCs > 0, leaked, uncancelled cancelState objects are reported.
type leakCheck struct {
	cs         *cancelState
	funcCalled bool // whether CancelFunc has been called; under cs.mu.
	stack      []uintptr
}

var leakedContextPCs int = 0 // stack frames to print for leaked allocation sites.
var initLeakCheckerOnce sync.Once
var leakCheckerFile *os.File

// initLeakChecker initializes leak checking if the file $VCONTEXT_LEAK_CHECK
// can be opened.
func initLeakChecker() {
	fileName := os.Getenv("VCONTEXT_LEAK_CHECK")
	if len(fileName) != 0 {
		var err error
		leakCheckerFile, err = os.OpenFile(fileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err == nil {
			leakedContextPCs = 20
		}
	}
}

// makeCancelFunc returns a function that cancels cancelState *cs with err, and
// it timer!=nil, stops *timer.  Requires that *cs has cancellation parent
// *cancelParent if cancelParent != nil.  It may use an indirection through a
// leakCheck if $VCONTEXT_LEAK_CHECK is set.
func makeCancelFunc(cs *cancelState, cancelParent *cancelState, timer *time.Timer, err error) (cancelFunc CancelFunc) {
	initLeakCheckerOnce.Do(initLeakChecker)
	if leakedContextPCs > 0 && err != DeadlineExceeded { // the timer is allowed to leak its callbacks.
		lc := &leakCheck{cs: cs, stack: make([]uintptr, leakedContextPCs)}
		lc.stack = lc.stack[:runtime.Callers(2, lc.stack)]
		runtime.SetFinalizer(lc, checkForLeaks)
		cancelFunc = func() { // captures cancelParent, timer, err, and lc (not cs).
			if cancelParent != nil {
				cancelParent.removeChild(lc.cs)
			}
			if timer != nil {
				timer.Stop()
			}
			lc.cs.cancel(err)
			runtime.SetFinalizer(lc, nil)
			lc.cs.mu.Lock()
			lc.funcCalled = true
			lc.cs.mu.Unlock()
		}
	} else {
		cancelFunc = func() { // captures cancelParent, timer, err, and cs.
			if cancelParent != nil {
				cancelParent.removeChild(cs)
			}
			if timer != nil {
				timer.Stop()
			}
			cs.cancel(err)
		}
	}
	return cancelFunc
}

// checkForLeaks is called when the garbage collector finds that a leakCheck
// object can be collected.
func checkForLeaks(lc *leakCheck) {
	cs := lc.cs
	cs.mu.Lock()
	funcCalled := lc.funcCalled
	cs.mu.Unlock()
	if !funcCalled {
		var stack string
		if lc.stack != nil {
			stack = ": stack:\n"
			for _, pc := range lc.stack {
				fnc := runtime.FuncForPC(pc)
				file, line := fnc.FileLine(pc)
				stack += fmt.Sprintf("   %s:%d: %s\n", file, line, fnc.Name())
			}
		}
		fmt.Fprintf(leakCheckerFile, "v.io/v23/context: CancelFunc garbage collected without call%s\n", stack)
	}
}

func (c *cancelState) addChild(child *cancelState) {
	c.mu.Lock()

	if c.err != nil {
		err := c.err
		c.mu.Unlock()
		child.cancel(err)
		return
	}

	if c.children == nil {
		c.children = make(map[*cancelState]bool)
	}
	c.children[child] = true
	c.mu.Unlock()
}

func (c *cancelState) removeChild(child *cancelState) {
	c.mu.Lock()
	delete(c.children, child)
	c.mu.Unlock()
}

func (c *cancelState) cancel(err error) {
	var children map[*cancelState]bool

	c.mu.Lock()
	if c.err == nil {
		c.err = err
		children = c.children
		c.children = nil
		close(c.done)
	}
	c.mu.Unlock()

	for child, _ := range children {
		child.cancel(err)
	}
}

// A deadlineState helps cancel contexts when a deadline expires.
type deadlineState struct {
	deadline time.Time
	timer    *time.Timer
}

// WithValue returns a child of the current context that will return
// the given val when Value(key) is called.
func WithValue(parent *T, key interface{}, val interface{}) *T {
	if !parent.Initialized() {
		panic("Trying to derive a context from an uninitialized context.")
	}
	if key == nil {
		panic("Attempting to store a context value with an untyped nil key.")
	}
	return &T{logger: parent.logger, ctxLogger: parent.ctxLogger, parent: parent, key: key, value: val}
}

func withCancelState(parent *T) (t *T, cs *cancelState, cancelParent *cancelState) {
	cs = &cancelState{done: make(chan struct{})}
	cancelParent, ok := parent.Value(cancelKey).(*cancelState)
	if ok {
		cancelParent.addChild(cs)
	}
	return WithValue(parent, cancelKey, cs), cs, cancelParent
}

// WithCancel returns a child of the current context along with
// a function that can be used to cancel it.  After cancel() is
// called the channels returned by the Done() methods of the new context
// (and all context further derived from it) will be closed.
func WithCancel(parent *T) (*T, CancelFunc) {
	t, cs, cancelParent := withCancelState(parent)
	return t, makeCancelFunc(cs, cancelParent, nil, Canceled)
}

func withDeadlineState(parent *T, deadline time.Time, timeout time.Duration) (*T, CancelFunc) {
	t, cs, cancelParent := withCancelState(parent)
	ds := &deadlineState{deadline, time.AfterFunc(timeout, makeCancelFunc(cs, cancelParent, nil, DeadlineExceeded))}
	return WithValue(t, deadlineKey, ds), makeCancelFunc(cs, cancelParent, ds.timer, Canceled)
}

// WithRootContext returns a context derived from parent, but that is
// detached from the deadlines and cancellation hierarchy so that this
// context will only ever be canceled when the returned CancelFunc is
// called, or the RootContext from which this context is ultimately
// derived is canceled.
func WithRootCancel(parent *T) (*T, CancelFunc) {
	var root *cancelState
	for ancestor := parent; ancestor != nil; ancestor = ancestor.parent {
		if cs, ok := ancestor.value.(*cancelState); ok {
			root = cs
		}
	}
	cs := &cancelState{done: make(chan struct{})}
	if root != nil {
		root.addChild(cs)
	}
	return WithValue(parent, cancelKey, cs), makeCancelFunc(cs, root, nil, Canceled)
}

// WithDeadline returns a child of the current context along with a
// function that can be used to cancel it at any time (as from
// WithCancel).  When the deadline is reached the context will be
// automatically cancelled.
// Contexts should be cancelled when they are no longer needed
// so that resources associated with their timers may be released.
func WithDeadline(parent *T, deadline time.Time) (*T, CancelFunc) {
	return withDeadlineState(parent, deadline, deadline.Sub(time.Now()))
}

// WithTimeout is similar to WithDeadline except a Duration is given
// that represents a relative point in time from now.
func WithTimeout(parent *T, timeout time.Duration) (*T, CancelFunc) {
	return withDeadlineState(parent, time.Now().Add(timeout), timeout)
}
