// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package access

import (
	"encoding/json"
	"io"
	"sort"
	"strings"
	"v.io/v23/context"
	"v.io/v23/security"
	"v.io/v23/verror"
)

// Includes returns true iff the AccessList grants access to a principal that
// presents blessings (i.e., if at least one of the blessings matches the
// AccessList).
func (acl AccessList) Includes(blessings ...string) bool {
	blessings = acl.pruneBlacklisted(blessings)
	for _, pattern := range acl.In {
		if pattern.MatchedBy(blessings...) {
			return true
		}
	}
	return false
}

func (acl AccessList) pruneBlacklisted(blessings []string) []string {
	if len(acl.NotIn) == 0 {
		return blessings
	}
	var filtered []string
	for _, b := range blessings {
		blacklisted := false
		for _, bp := range acl.NotIn {
			if security.BlessingPattern(bp).MatchedBy(b) {
				blacklisted = true
				break
			}
		}
		if !blacklisted {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

// Authorize implements security.Authorizer where the request is authorized
// only if the remote blessings are included in the AccessList.
func (acl AccessList) Authorize(ctx *context.T, call security.Call) error {
	blessingsForCall, invalid := security.RemoteBlessingNames(ctx, call)
	if acl.Includes(blessingsForCall...) {
		return nil
	}
	return NewErrAccessListMatch(ctx, blessingsForCall, invalid)
}

// Enforceable checks if the AccessList is enforceable by the provided
// principal.
//
// It returns nil if all blessing patterns in the 'In' list are valid and
// matched by a blessing name that is recognized by one of the provided
// principal's roots.
//
// An error with identifier ErrOpenAccessList.ID is returned if the 'In' list
// contains the pattern "..." along with other patterns in the 'In' or 'NotIn'
// lists. Otherwise an error with identifier ErrUnenforceablePatterns.ID is
// returned.
func (acl AccessList) Enforceable(ctx *context.T, p security.Principal) error {
	if acl.isOpen() {
		return nil
	}

	var (
		rootPatterns []security.BlessingPattern
		rejected     []security.BlessingPattern
	)
	for pattern, _ := range p.Roots().Dump() {
		rootPatterns = append(rootPatterns, pattern)
	}

	for _, p := range acl.In {
		if p == security.AllPrincipals {
			return NewErrInvalidOpenAccessList(ctx)
		}
		if !p.IsValid() {
			rejected = append(rejected, p)
			continue
		}
		if !isRecognized(p, rootPatterns) {
			rejected = append(rejected, p)
		}
	}
	if len(rejected) == 0 {
		return nil
	}
	return NewErrUnenforceablePatterns(ctx, rejected)
}

func (acl AccessList) isOpen() bool {
	if len(acl.NotIn) == 0 && (len(acl.In) == 1 && acl.In[0] == security.AllPrincipals) {
		return true
	}
	return false
}

// WritePermissions writes the JSON-encoded representation of a Permissions to w.
func WritePermissions(w io.Writer, m Permissions) error {
	return json.NewEncoder(w).Encode(m.Normalize())
}

// ReadPermissions reads the JSON-encoded representation of a Permissions from r.
func ReadPermissions(r io.Reader) (m Permissions, err error) {
	err = json.NewDecoder(r).Decode(&m)
	return
}

// Add updates m to so that blessings matching pattern will be included in the
// access lists for the provided tags (by adding to the "In" lists).
// It returns m.
func (m Permissions) Add(pattern security.BlessingPattern, tags ...string) Permissions {
	for _, tag := range tags {
		list := m[tag]
		list.In = append(list.In, pattern)
		list.In = removeDuplicatePatterns(list.In)
		sort.Sort(byPattern(list.In))
		m[tag] = list
	}
	return m
}

// Blacklist updates m so that the provided blessing will be excluded from
// the access lists for the provided tags (by adding to the "NotIn" lists).
// It returns m.
func (m Permissions) Blacklist(blessing string, tags ...string) Permissions {
	for _, tag := range tags {
		list := m[tag]
		list.NotIn = append(list.NotIn, blessing)
		list.NotIn = removeDuplicateStrings(list.NotIn)
		sort.Strings(list.NotIn)
		m[tag] = list
	}
	return m
}

// Clear removes all references to blessingOrPattern from all the provided
// tags in the AccessList, or all tags if len(tags) = 0. It returns m.
func (m Permissions) Clear(blessingOrPattern string, tags ...string) Permissions {
	if len(tags) == 0 {
		tags = make([]string, 0, len(m))
		for t, _ := range m {
			tags = append(tags, t)
		}
	}
	for _, t := range tags {
		oldList := m[t]
		var newList AccessList
		for _, p := range oldList.In {
			if string(p) != blessingOrPattern {
				newList.In = append(newList.In, p)
			}
		}
		for _, b := range oldList.NotIn {
			if b != blessingOrPattern {
				newList.NotIn = append(newList.NotIn, b)
			}
		}
		m[t] = newList
	}
	return m
}

// Copy returns a new Permissions that is a copy of m.
func (m Permissions) Copy() Permissions {
	ret := make(Permissions)
	for tag, list := range m {
		var newlist AccessList
		if len(list.In) > 0 {
			newlist.In = make([]security.BlessingPattern, len(list.In))
		}
		if len(list.NotIn) > 0 {
			newlist.NotIn = make([]string, len(list.NotIn))
		}
		for idx, item := range list.In {
			newlist.In[idx] = item
		}
		for idx, item := range list.NotIn {
			newlist.NotIn[idx] = item
		}
		ret[tag] = newlist
	}
	return ret
}

// Normalize re-organizes 'm' so that two equivalent Permissions are
// comparable via reflection. It returns 'm'.
func (m Permissions) Normalize() Permissions {
	for tag, list := range m {
		list.In = removeDuplicatePatterns(list.In)
		list.NotIn = removeDuplicateStrings(list.NotIn)
		sort.Sort(byPattern(list.In))
		sort.Strings(list.NotIn)
		if len(list.In) == 0 && list.In != nil {
			list.In = nil
		}
		if len(list.NotIn) == 0 && list.NotIn != nil {
			list.NotIn = nil
		}
		m[tag] = list
	}
	return m
}

// UnenforceablePatterns checks if the error has the identifier
// ErrUnenforceablePatterns.ID, and if so returns the set of
// unenforceable patterns encapsulated in it.  It returns nil otherwise.
func IsUnenforceablePatterns(err error) []security.BlessingPattern {
	if verror.ErrorID(err) != ErrUnenforceablePatterns.ID {
		return nil
	}
	verr, ok := err.(verror.E)
	if !ok {
		return nil
	}
	params := verr.ParamList
	if len(params) != 3 {
		return nil
	}
	patterns, ok := params[2].([]security.BlessingPattern)
	if !ok {
		return nil
	}
	return patterns
}

func removeDuplicatePatterns(l []security.BlessingPattern) (ret []security.BlessingPattern) {
	m := make(map[security.BlessingPattern]bool)
	for _, s := range l {
		if _, ok := m[s]; ok {
			continue
		}
		ret = append(ret, s)
		m[s] = true
	}
	return ret
}

func removeDuplicateStrings(l []string) (ret []string) {
	m := make(map[string]bool)
	for _, s := range l {
		if _, ok := m[s]; ok {
			continue
		}
		ret = append(ret, s)
		m[s] = true
	}
	return ret
}

// This method assumes that p != security.AllPrincipals
func isRecognized(p security.BlessingPattern, rootPatterns []security.BlessingPattern) bool {
	if p == security.NoExtension {
		return true
	}

	const nonExtSuffix = security.ChainSeparator + string(security.NoExtension)

	s := string(p)
	nonExtPattern := strings.HasSuffix(s, nonExtSuffix)
	if nonExtPattern {
		s = strings.TrimSuffix(s, nonExtSuffix)
	}

	for _, root := range rootPatterns {
		if root == security.NoExtension {
			continue
		}
		nonExtRoot := strings.HasSuffix(string(root), nonExtSuffix)
		if !nonExtPattern && nonExtRoot {
			// The root, by virtue of being non-extendable, will only be matched
			// by a single blessing name, whereas, the pattern, by virtue of being
			// extendable, will be matched by infinitely many blessing names.
			// Therefore, not all blessing names that match the pattern are recognized
			// by the root.
			continue
		}
		if root.MatchedBy(s) {
			return true
		}
	}
	return false
}

type byPattern []security.BlessingPattern

func (a byPattern) Len() int           { return len(a) }
func (a byPattern) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPattern) Less(i, j int) bool { return a[i] < a[j] }
