// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"sync"
	"time"
)

// NewFloat creates a new Float StatsObject with the given name and
// returns a pointer to it.
func NewFloat(name string) *Float {
	lock.Lock()
	defer lock.Unlock()

	node := findNodeLocked(name, true)
	f := Float{value: 0}
	node.object = &f
	return &f
}

// Float implements the StatsObject interface.
type Float struct {
	mu         sync.RWMutex
	lastUpdate time.Time
	value      float64
}

// Set sets the value of the object.
func (f *Float) Set(value float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastUpdate = time.Now()
	f.value = value
}

// Incr increments the value of the object.
func (f *Float) Incr(delta float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.value += delta
	f.lastUpdate = time.Now()
}

// LastUpdate returns the time at which the object was last updated.
func (f *Float) LastUpdate() time.Time {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.lastUpdate
}

// Value returns the current value of the object.
func (f *Float) Value() interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.value
}
