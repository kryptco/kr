// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package glob defines a globbing syntax and implements matching routines.
//
// Globs match a slash separated series of glob expressions.
//
//   // Patterns:
//   term ['/' term]*
//   term:
//   '*'         matches any sequence of non-Separator characters
//   '?'         matches any single non-Separator character
//   '[' [ '^' ] { character-range } ']'
//   // Character classes (must be non-empty):
//   c           matches character c (c != '*', '?', '\\', '[', '/')
//   '\\' c      matches character c
//   // Character-ranges:
//   c           matches character c (c != '\\', '-', ']')
//   '\\' c      matches character c
//   lo '-' hi   matches character c for lo <= c <= hi
package glob

import (
	"path"
	"strings"
)

// Glob represents a slash separated path glob pattern.
type Glob struct {
	elems      []*Element
	recursive  bool
	restricted bool
}

// Parse returns a new Glob.
func Parse(pattern string) (*Glob, error) {
	if len(pattern) > 0 && pattern[0] == '/' {
		return nil, path.ErrBadPattern
	}

	g := &Glob{}
	if pattern == "" {
		return g, nil
	}

	elems := strings.Split(pattern, "/")
	if last := len(elems) - 1; last >= 0 {
		if elems[last] == "..." {
			elems = elems[:last]
			g.recursive = true
		} else if elems[last] == "***" {
			elems = elems[:last]
			g.recursive = true
			g.restricted = true
		}
	}
	g.elems = make([]*Element, len(elems))
	for i, elem := range elems {
		g.elems[i] = &Element{pattern: elem}
		if err := g.elems[i].validate(); err != nil {
			return nil, err
		}
	}

	return g, nil
}

// Len returns the number of path elements represented by the glob expression.
func (g *Glob) Len() int {
	return len(g.elems)
}

// Empty returns true if the pattern cannot match anything.
func (g *Glob) Empty() bool {
	return !g.recursive && len(g.elems) == 0
}

// Recursive returns true if the pattern is recursive.
func (g *Glob) Recursive() bool {
	return g.recursive
}

// Restricted returns true if recursion is restricted (up to the caller to
// know what that means).
func (g *Glob) Restricted() bool {
	return g.restricted
}

// Tail returns the suffix of g starting at the second element.
func (g *Glob) Tail() *Glob {
	if len(g.elems) <= 1 {
		return &Glob{elems: nil, recursive: g.recursive, restricted: g.restricted}
	}
	return &Glob{elems: g.elems[1:], recursive: g.recursive, restricted: g.restricted}
}

// Head returns an Element for the first element of the glob pattern.
func (g *Glob) Head() *Element {
	if len(g.elems) == 0 {
		if g.recursive {
			return &Element{alwaysMatch: true}
		}
		return &Element{neverMatch: true}
	}
	return g.elems[0]
}

// SplitFixedElements returns the part of the glob pattern that contains only
// fixed elements, and the glob that follows it.
func (g *Glob) SplitFixedElements() ([]string, *Glob) {
	var prefix []string
	tail := g
	for _, elem := range g.elems {
		if pfx, fixed := elem.FixedPrefix(); fixed {
			prefix = append(prefix, pfx)
			tail = tail.Tail()
		} else {
			break
		}
	}
	return prefix, tail
}

// String returns the string representation of the glob pattern.
func (g *Glob) String() string {
	elems := make([]string, len(g.elems))
	for i, e := range g.elems {
		elems[i] = e.pattern
	}
	if g.recursive {
		if g.restricted {
			elems = append(elems, "***")
		} else {
			elems = append(elems, "...")
		}
	}
	return path.Join(elems...)
}

// Element represents a single element of a glob pattern.
type Element struct {
	pattern     string
	alwaysMatch bool
	neverMatch  bool
}

// Match returns true iff this pattern element matches the given segment.
func (m *Element) Match(segment string) bool {
	if m.neverMatch {
		return false
	}
	if m.alwaysMatch {
		return true
	}
	matches, err := path.Match(m.pattern, segment)
	return err == nil && matches
}

// FixedPrefix returns the unescaped fixed part of the pattern, and whether the
// prefix is the whole pattern. The fixed part does not contain any wildcards.
func (m *Element) FixedPrefix() (string, bool) {
	if m.neverMatch {
		return "", true
	}
	if m.alwaysMatch {
		return "", false
	}
	unescaped := ""
	escape := false
	for _, c := range m.pattern {
		if escape {
			escape = false
			unescaped += string(c)
		} else if strings.ContainsRune("*?[", c) {
			return unescaped, false
		} else if c == '\\' {
			escape = true
		} else {
			unescaped += string(c)
		}
	}
	return unescaped, true
}

func (m *Element) validate() error {
	if len(m.pattern) == 0 {
		return path.ErrBadPattern
	}
	escape := false
	inrange := false
	for _, c := range m.pattern {
		if escape {
			escape = false
			continue
		}
		switch c {
		case '\\':
			escape = true
		case '[':
			inrange = true
		case ']':
			inrange = false
		}
	}
	// If we are in the middle of an escape or character range, the expression is incomplete.
	if escape || inrange {
		return path.ErrBadPattern
	}
	return nil
}
