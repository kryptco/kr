// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package access

import (
	"v.io/v23/context"
	"v.io/v23/security"
	"v.io/v23/vdl"
)

func init() {
	security.RegisterCaveatValidator(AccessTagCaveat, func(ctx *context.T, call security.Call, params []Tag) error {
		wantT := TypicalTagType()
		methodTags := call.MethodTags()
		for _, mt := range methodTags {
			if mt.Type() == wantT {
				for _, ct := range params {
					if mt.RawString() == vdl.ValueOf(ct).RawString() {
						return nil
					}
				}
			}
		}
		strs := make([]string, len(methodTags))
		for i, mt := range methodTags {
			strs[i] = mt.RawString()
		}
		return NewErrAccessTagCaveatValidation(ctx, strs, params)
	})
}

// NewAccessTagCaveat returns a Caveat that will validate iff the intersection
// of the tags on the method being invoked and those in 'tags' is non-empty.
func NewAccessTagCaveat(tags ...Tag) (security.Caveat, error) {
	return security.NewCaveat(AccessTagCaveat, tags)
}
