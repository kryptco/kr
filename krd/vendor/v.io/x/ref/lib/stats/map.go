// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"path"
	"sort"
	"sync"
	"time"
)

// NewMap creates a new Map StatsObject with the given name and
// returns a pointer to it.
func NewMap(name string) *Map {
	lock.Lock()
	defer lock.Unlock()
	node := findNodeLocked(name, true)
	m := Map{name: name, value: make(map[string]mapValue)}
	node.object = &m
	return &m
}

// Map implements the StatsObject interface. The map keys are strings and the
// values can be bool, int64, uint64, float64, string, or time.Time.
type Map struct {
	mu    sync.RWMutex // ACQUIRED_BEFORE(stats.lock)
	name  string
	value map[string]mapValue // GUARDED_BY(mu)
}

type mapValue struct {
	lastUpdate time.Time
	value      interface{}
}

// Set sets the values of the given keys. There must be exactly one value for
// each key.
func (m *Map) Set(kvpairs []KeyValue) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, kv := range kvpairs {
		var v interface{}
		switch value := kv.Value.(type) {
		case bool:
			v = bool(value)
		case int:
			v = int64(value)
		case int8:
			v = int64(value)
		case int16:
			v = int64(value)
		case int32:
			v = int64(value)
		case int64:
			v = int64(value)
		case uint:
			v = uint64(value)
		case uint8:
			v = uint64(value)
		case uint16:
			v = uint64(value)
		case uint32:
			v = uint64(value)
		case uint64:
			v = uint64(value)
		case float32:
			v = float64(value)
		case float64:
			v = float64(value)
		case string:
			v = string(value)
		case time.Time:
			v = value.String()
		default:
			panic("attempt to use an unsupported type as value")
		}
		m.value[kv.Key] = mapValue{now, v}
	}
	m.insertMissingNodes()
}

// Incr increments the value of the given key and returns the new value.
func (m *Map) Incr(key string, delta int64) interface{} {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.value[key]; !exists {
		m.value[key] = mapValue{now, int64(0)}
		oName := path.Join(m.name, key)
		lock.Lock()
		if n := findNodeLocked(oName, true); n.object == nil {
			n.object = &mapValueWrapper{m, key}
		}
		lock.Unlock()
	}
	var result interface{}
	switch value := m.value[key].value.(type) {
	case int64:
		result = value + delta
	case uint64:
		if delta >= 0 {
			result = value + uint64(delta)
		} else {
			result = value - uint64(-delta)
		}
	case float64:
		result = value + float64(delta)
	default:
		return nil
	}
	m.value[key] = mapValue{now, result}
	return result
}

// Delete deletes the given keys from the map object.
func (m *Map) Delete(keys []string) {
	// The lock order is important.
	m.mu.Lock()
	defer m.mu.Unlock()
	lock.Lock()
	defer lock.Unlock()
	n := findNodeLocked(m.name, false)
	for _, k := range keys {
		delete(m.value, k)
		if n != nil {
			delete(n.children, k)
		}
	}
}

// Keys returns a sorted list of all the keys in the map.
func (m *Map) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := []string{}
	for k, _ := range m.value {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// LastUpdate always returns a zero-value Time for a Map.
func (m *Map) LastUpdate() time.Time {
	return time.Time{}
}

// Value always returns nil for a Map.
func (m *Map) Value() interface{} {
	return nil
}

// insertMissingNodes inserts all the missing nodes.
func (m *Map) insertMissingNodes() {
	missing := []string{}
	lock.RLock()
	for key, _ := range m.value {
		oName := path.Join(m.name, key)
		if n := findNodeLocked(oName, false); n == nil {
			missing = append(missing, key)
		}
	}
	lock.RUnlock()
	if len(missing) == 0 {
		return
	}

	lock.Lock()
	for _, key := range missing {
		oName := path.Join(m.name, key)
		if n := findNodeLocked(oName, true); n.object == nil {
			n.object = &mapValueWrapper{m, key}
		}
	}
	lock.Unlock()
}

type mapValueWrapper struct {
	m   *Map
	key string
}

// LastUpdate returns the time at which the parent map object was last updated.
func (w *mapValueWrapper) LastUpdate() time.Time {
	w.m.mu.RLock()
	defer w.m.mu.RUnlock()
	if v, ok := w.m.value[w.key]; ok {
		return v.lastUpdate
	}
	return time.Time{}
}

// Value returns the current value for the map key.
func (w *mapValueWrapper) Value() interface{} {
	w.m.mu.RLock()
	defer w.m.mu.RUnlock()
	if v, ok := w.m.value[w.key]; ok {
		return v.value
	}
	return nil
}
