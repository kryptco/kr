// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"sync"
	"time"
)

// NewString creates a new String StatsObject with the given name and
// returns a pointer to it.
func NewString(name string) *String {
	lock.Lock()
	defer lock.Unlock()

	node := findNodeLocked(name, true)
	s := String{value: ""}
	node.object = &s
	return &s
}

// String implements the StatsObject interface.
type String struct {
	mu         sync.RWMutex
	lastUpdate time.Time
	value      string
}

// Set sets the value of the object.
func (s *String) Set(value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastUpdate = time.Now()
	s.value = value
}

// LastUpdate returns the time at which the object was last updated.
func (s *String) LastUpdate() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdate
}

// Value returns the current value of the object.
func (s *String) Value() interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}
