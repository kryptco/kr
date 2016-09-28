// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package uniqueid defines functions that are likely to generate
// globally unique identifiers. We want to be able to generate many
// Ids quickly, so we make a time/space tradeoff.  We reuse the same
// random data many times with a counter appended.  Note: these Ids are
// NOT useful as a security mechanism as they will be predictable.
package uniqueid

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
)

var random = RandomGenerator{}

func (id Id) String() string {
	return fmt.Sprintf("0x%x", [16]byte(id))
}

// Valid returns true if the given Id is valid.
func Valid(id Id) bool {
	return id != Id{}
}

func FromHexString(s string) (Id, error) {
	var id Id
	var slice []byte
	if strings.HasPrefix(s, "0x") {
		s = s[2:]
	}
	if _, err := fmt.Sscanf(s, "%x", &slice); err != nil {
		return id, err
	}
	if len(slice) != len(id) {
		// Ideally we would generate a verror error here, but Go
		// complains about the import cycle:  verror, vtrace, and
		// uniqueid.  In most languages the linker would just pull in
		// all three implementations, but Go conflates implementations
		// and their interfaces, so cannot be sure that this isn't an
		// interface definition cycle, and thus gives up.
		return id, fmt.Errorf("Cannot convert %s to Id, size mismatch.", s)
	}
	copy(id[:], slice)
	return id, nil
}

// A RandomGenerator can generate random Ids.
// The zero value of RandomGenerator is ready to use.
type RandomGenerator struct {
	mu    sync.Mutex
	id    Id
	count uint16
}

// NewId produces a new probably unique identifier.
func (g *RandomGenerator) NewID() (Id, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.count > 0x7fff || g.count == uint16(0) {
		g.count = 0

		// Either the generator is uninitialized or the counter
		// has wrapped.  We need a new random prefix.
		if _, err := rand.Read(g.id[:14]); err != nil {
			return Id{}, err
		}
	}
	binary.BigEndian.PutUint16(g.id[14:], g.count)
	g.count++
	g.id[14] |= 0x80 // Use this bit as a reserved bit (set to 1) to support future format changes.
	return g.id, nil
}

// Random produces a new probably unique identifier using the RandomGenerator.
func Random() (Id, error) {
	return random.NewID()
}
