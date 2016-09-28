// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"sync"
	"time"
)

// NewInteger creates a new Integer StatsObject with the given name and
// returns a pointer to it.
func NewInteger(name string) *Integer {
	lock.Lock()
	defer lock.Unlock()

	node := findNodeLocked(name, true)
	i := Integer{value: 0}
	node.object = &i
	return &i
}

// Integer implements the StatsObject interface.
type Integer struct {
	mu         sync.RWMutex
	lastUpdate time.Time
	value      int64
}

// Set sets the value of the object.
func (i *Integer) Set(value int64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.lastUpdate = time.Now()
	i.value = value
}

// Incr increments the value of the object.
func (i *Integer) Incr(delta int64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.value += delta
	i.lastUpdate = time.Now()
}

// LastUpdate returns the time at which the object was last updated.
func (i *Integer) LastUpdate() time.Time {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.lastUpdate
}

// Value returns the current value of the object.
func (i *Integer) Value() interface{} {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.value
}
