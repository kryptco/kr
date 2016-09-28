// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"path"
	"sort"
	"time"

	"v.io/v23/glob"
	"v.io/v23/verror"
)

// Glob returns the name and (optionally) the value of all the objects that
// match the given pattern and have been updated since 'updatedSince'. The
// 'root' argument is the name of the object where the pattern starts.
// Example:
//   a/b/c
//   a/b/d
//   b/e/f
// Glob("", "...", time.Time{}, true) will return "a/b/c", "a/b/d", "b/e/f" and
// their values.
// Glob("a/b", "*", time.Time{}, true) will return "c", "d" and their values.
func Glob(root string, pattern string, updatedSince time.Time, includeValues bool) *GlobIterator {
	g, err := glob.Parse(pattern)
	if err != nil {
		return &GlobIterator{err: err}
	}
	lock.RLock()
	defer lock.RUnlock()
	node := findNodeLocked(root, false)
	if node == nil {
		return &GlobIterator{err: verror.New(verror.ErrNoExist, nil, root)}
	}
	var out []KeyValue
	globStepLocked("", g, node, updatedSince, includeValues, &out)
	sort.Sort(keyValueSort(out))
	return &GlobIterator{results: out}
}

// globStepLocked applies a glob recursively.
func globStepLocked(prefix string, g *glob.Glob, n *node, updatedSince time.Time, includeValues bool, result *[]KeyValue) {
	if g.Len() == 0 {
		if updatedSince.IsZero() || (n.object != nil && !n.object.LastUpdate().Before(updatedSince)) {
			var v interface{}
			if includeValues && n.object != nil {
				v = n.object.Value()
			}
			*result = append(*result, KeyValue{prefix, v})
		}
	}
	if g.Empty() {
		return
	}
	matcher, left := g.Head(), g.Tail()
	for name, child := range n.children {
		if matcher.Match(name) {
			globStepLocked(path.Join(prefix, name), left, child, updatedSince, includeValues, result)
		}
	}
}

// KeyValue stores a Key and a Value.
type KeyValue struct {
	Key   string
	Value interface{}
}

// keyValueSort is used to sort a slice of KeyValue objects.
type keyValueSort []KeyValue

func (s keyValueSort) Len() int {
	return len(s)
}

func (s keyValueSort) Less(i, j int) bool {
	return s[i].Key < s[j].Key
}

func (s keyValueSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type GlobIterator struct {
	results []KeyValue
	next    KeyValue
	err     error
}

// Advance stages the next element so that the client can retrieve it with
// Value(). It returns true iff there is an element to retrieve. The client
// must call Advance() before calling Value(). Advance may block if an element
// is not immediately available.
func (i *GlobIterator) Advance() bool {
	if len(i.results) == 0 {
		return false
	}
	i.next = i.results[0]
	i.results = i.results[1:]
	return true
}

// Value returns the element that was staged by Advance. Value does not block.
func (i GlobIterator) Value() KeyValue {
	return i.next
}

// Err returns a non-nil error iff the stream encountered any errors.  Err does
// not block.
func (i GlobIterator) Err() error {
	return i.err
}
