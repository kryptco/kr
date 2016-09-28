// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import "sync"

// A Barrier is a simple incremental barrier that runs a done closure after
// all closures returned by Add() have been invoked. The done closure will
// run only once and run in a separate goroutine.
type Barrier struct {
	mu sync.Mutex
	n  int // GUARDED_BY(mu)

	done func()
}

// Add increments the barrier. Each closure returned by Add() should eventually
// be run once, otherwise 'done' will never be run. It returns no-op function
// if the done closure has been already called.
func (b *Barrier) Add() func() {
	b.mu.Lock()
	if b.done == nil {
		b.mu.Unlock()
		return func() {}
	}
	b.n++
	b.mu.Unlock()
	return b.run
}

func (b *Barrier) run() {
	b.mu.Lock()
	b.n--
	if b.n > 0 {
		b.mu.Unlock()
		return
	}
	done := b.done
	b.done = nil
	b.mu.Unlock()
	if done != nil {
		go done()
	}
}

func NewBarrier(done func()) *Barrier {
	return &Barrier{done: done}
}
