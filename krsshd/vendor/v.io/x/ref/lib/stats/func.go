// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"sync"
	"time"
)

// NewIntegerFunc creates a new StatsObject with the given name. The function
// argument must return an int64 value.
func NewIntegerFunc(name string, function func() int64) StatsObject {
	return newFunc(name, func() interface{} { return function() })
}

// NewFloatFunc creates a new StatsObject with the given name. The function
// argument must return a float64 value.
func NewFloatFunc(name string, function func() float64) StatsObject {
	return newFunc(name, func() interface{} { return function() })
}

// NewStringFunc creates a new StatsObject with the given name. The function
// argument must return a string value.
func NewStringFunc(name string, function func() string) StatsObject {
	return newFunc(name, func() interface{} { return function() })
}

func newFunc(name string, function func() interface{}) StatsObject {
	f := funcType{function: function}
	lock.Lock()
	defer lock.Unlock()
	node := findNodeLocked(name, true)
	node.object = &f
	return &f
}

// funcType implements the StatsObject interface by calling a user provided
// function.
type funcType struct {
	mu        sync.Mutex
	function  func() interface{}
	waiters   []chan interface{} // GUARDED_BY(mu)
	lastValue interface{}        // GUARDED_BY(mu)
}

// LastUpdate returns always returns the current time for this type of
// StatsObject because Value() is expected to get a current (fresh) value.
func (f *funcType) LastUpdate() time.Time {
	return time.Now()
}

// Value returns the value returned by the object's function. If the function
// takes more than 100 ms to return, the last value is used.
func (f *funcType) Value() interface{} {
	// There are two values that can be written to the channel, one from
	// fetch() and one from time.AfterFunc(). In some cases, they will both
	// be written but only one will be read. A buffer size of 1 would be
	// sufficient to avoid deadlocks, but 2 will guarantee that fetch()
	// never blocks on a channel.
	ch := make(chan interface{}, 2)
	f.mu.Lock()
	if f.waiters = append(f.waiters, ch); len(f.waiters) == 1 {
		go f.fetch()
	}
	f.mu.Unlock()

	defer time.AfterFunc(100*time.Millisecond, func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		ch <- f.lastValue
	}).Stop()

	return <-ch
}

func (f *funcType) fetch() {
	v := f.function()

	f.mu.Lock()
	waiters := f.waiters
	f.waiters = nil
	f.lastValue = v
	f.mu.Unlock()

	for _, c := range waiters {
		c <- v
	}
}
