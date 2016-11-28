// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package security

import (
	"fmt"
	"regexp"
	"strings"

	"v.io/v23/naming"
)

// Syntax is /{endpoint}/__({pattern})/{name}
var namePatternRegexp = regexp.MustCompile(`^__\(([^)]*)\)($|/)(.*)`)

// MatchedBy returns true iff one of the presented blessings matches
// p as per the rules described in documentation for the BlessingPattern type.
func (p BlessingPattern) MatchedBy(blessings ...string) bool {
	if len(p) == 0 || !p.IsValid() {
		return false
	}
	if p == AllPrincipals {
		return true
	}
	pstr, glob := trimNoExtension(string(p))
	if pstr == "" {
		return false
	}
	for _, b := range blessings {
		if b == pstr {
			return true
		}
		if glob && strings.HasPrefix(b, pstr) && strings.HasPrefix(b[len(pstr):], ChainSeparator) {
			return true
		}
	}
	return false
}

// trimNoExtension removes the trailing NoExtension component from pattern.
// Returns true if nothing was trimmed.
func trimNoExtension(pattern string) (string, bool) {
	if suffix := string(NoExtension); pattern == suffix {
		return "", false
	}
	if suffix := ChainSeparator + string(NoExtension); strings.HasSuffix(pattern, suffix) {
		return pattern[0 : len(pattern)-len(suffix)], false
	}
	return pattern, true
}

// splitBlessing splits in into the first component upto the ChainSeparator and
// the rest.
func splitBlessing(in string) (prefix, rest string) {
	idx := strings.Index(in, ChainSeparator)
	if idx == -1 {
		return in, ""
	}
	return in[0:idx], in[idx+1:]
}

// IsValid returns true iff the BlessingPattern is well formed, as per the
// rules described in documentation for the BlessingPattern type.
func (p BlessingPattern) IsValid() bool {
	if len(p) == 0 {
		return false
	}
	if p == AllPrincipals {
		return true
	}
	pstr, _ := trimNoExtension(string(p))
	if strings.HasSuffix(pstr, ChainSeparator) {
		return false
	}
	for len(pstr) > 0 {
		prefix, rest := splitBlessing(pstr)
		if validateExtension(prefix) != nil {
			return false
		}
		pstr = rest
	}
	return true
}

// MakeNonExtendable returns a pattern that is matched exactly
// by the blessing specified by the given pattern string.
//
// For example:
//   onlyAlice := BlessingPattern("google:alice").MakeNonExtendable()
//   onlyAlice.MatchedBy("google:alice")  // Returns true
//   onlyAlice.MatchedBy("google")  // Returns false
//   onlyAlice.MatchedBy("google:alice:bob")  // Returns false
func (p BlessingPattern) MakeNonExtendable() BlessingPattern {
	if len(p) == 0 || p == BlessingPattern(NoExtension) {
		return BlessingPattern(NoExtension)
	}
	if strings.HasSuffix(string(p), ChainSeparator+string(NoExtension)) {
		return p
	}
	return BlessingPattern(string(p) + ChainSeparator + string(NoExtension))
}

// PrefixPatterns returns a set of BlessingPatterns that are matched by
// blessings that either directly match the provided pattern or can be
// extended to match the provided pattern.
//
// For example:
// BlessingPattern("google:alice:friend").PrefixPatterns() returns
//   ["google:$", "google:alice:$", "google:alice:friend"]
// BlessingPattern("google:alice:friend:$").PrefixPatterns() returns
//   ["google:$", "google:alice:$", "google:alice:friend:$"]
//
// The returned set of BlessingPatterns are ordered by the number of
// ":"-separated components in the pattern.
func (p BlessingPattern) PrefixPatterns() []BlessingPattern {
	if p == NoExtension {
		return []BlessingPattern{p}
	}
	parts := strings.Split(string(p), ChainSeparator)
	if parts[len(parts)-1] == string(NoExtension) {
		parts = parts[:len(parts)-2]
	} else {
		parts = parts[:len(parts)-1]
	}
	var ret []BlessingPattern
	for i := 0; i < len(parts); i++ {
		ret = append(ret, BlessingPattern(strings.Join(parts[:i+1], ChainSeparator)).MakeNonExtendable())
	}
	return append(ret, p)
}

// SplitPatternName takes an object name and parses out the server blessing pattern.
// It returns the pattern specified, and the name with the pattern removed.
func SplitPatternName(origName string) (BlessingPattern, string) {
	rooted := naming.Rooted(origName)
	ep, name := naming.SplitAddressName(origName)
	match := namePatternRegexp.FindStringSubmatch(name)
	if len(match) == 0 {
		return BlessingPattern(""), origName
	}

	pattern := BlessingPattern(match[1])
	name = naming.Clean(match[3])
	if rooted {
		name = naming.JoinAddressName(ep, name)
	}
	return pattern, name
}

// JoinPatternName embeds the specified pattern into a name.
func JoinPatternName(pattern BlessingPattern, name string) string {
	if len(pattern) == 0 {
		return name
	}
	ep, rel := naming.SplitAddressName(name)
	return naming.JoinAddressName(ep, fmt.Sprintf("__(%s)/%s", pattern, rel))
}
