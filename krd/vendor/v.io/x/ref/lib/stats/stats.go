// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package stats implements a global repository of stats objects. Each object
// has a name and a value.
// Example:
//   bar1 := stats.NewInteger("foo/bar1")
//   bar2 := stats.NewFloat("foo/bar2")
//   bar3 := stats.NewCounter("foo/bar3")
//   bar1.Set(1)
//   bar2.Set(2)
//   bar3.Set(3)
// The values can be retrieved with:
//   v, err := stats.Value("foo/bar1")
package stats

import (
	"strings"
	"sync"
	"time"

	"v.io/v23/services/stats"
	"v.io/v23/verror"
)

// StatsObject is the interface for objects stored in the stats repository.
type StatsObject interface {
	// LastUpdate is used by WatchGlob to decide which updates to send.
	LastUpdate() time.Time
	// Value returns the current value of the object.
	Value() interface{}
}

type node struct {
	object   StatsObject
	children map[string]*node
}

var (
	lock       sync.RWMutex
	repository *node // GUARDED_BY(lock)
)

func init() {
	repository = newNode()
}

// GetStatsObject returns the object with that given name, or an error if the
// object doesn't exist.
func GetStatsObject(name string) (StatsObject, error) {
	lock.RLock()
	defer lock.RUnlock()
	node := findNodeLocked(name, false)
	if node == nil || node.object == nil {
		return nil, verror.New(verror.ErrNoExist, nil, name)
	}
	return node.object, nil
}

// Value returns the value of an object, or an error if the object doesn't
// exist.
func Value(name string) (interface{}, error) {
	obj, err := GetStatsObject(name)
	if err != nil {
		return 0, err
	}
	if obj == nil {
		return nil, verror.New(stats.ErrNoValue, nil, name)
	}
	return obj.Value(), nil
}

// Delete deletes a StatsObject and all its children, if any.
func Delete(name string) error {
	if name == "" {
		return verror.New(verror.ErrNoExist, nil, name)
	}
	elems := strings.Split(name, "/")
	last := len(elems) - 1
	dirname, basename := strings.Join(elems[:last], "/"), elems[last]
	lock.Lock()
	defer lock.Unlock()
	parent := findNodeLocked(dirname, false)
	if parent == nil {
		return verror.New(verror.ErrNoExist, nil, name)
	}
	delete(parent.children, basename)
	return nil
}

func newNode() *node {
	return &node{children: make(map[string]*node)}
}

// findNodeLocked finds a node, and optionally creates it if it doesn't already
// exist.
func findNodeLocked(name string, create bool) *node {
	elems := strings.Split(name, "/")
	node := repository
	for {
		if len(elems) == 0 {
			return node
		}
		if len(elems[0]) == 0 {
			elems = elems[1:]
			continue
		}
		if next, ok := node.children[elems[0]]; ok {
			node = next
			elems = elems[1:]
			continue
		}
		if create {
			node.children[elems[0]] = newNode()
			node = node.children[elems[0]]
			elems = elems[1:]
			continue
		}
		return nil
	}
}
