// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package access

import "v.io/v23/vdl"

// TypicalTagType returns the type of the predefined tags in this access
// package.
//
// Typical use of this is to setup an Authorizer that uses these predefined
// tags:
//   authorizer, err := PermissionsAuthorizerFromFile(name, TypicalTagType())
//
// For the common case of setting up an Authorizer for a Permissions object with
// these predefined tags, a convenience function is provided:
//   authorizer := TypicalTagTypePermissionsAuthorizer(myperms)
func TypicalTagType() *vdl.Type {
	return vdl.TypeOf(Tag(""))
}

// AllTypicalTags returns all access.Tag values defined in this package.
func AllTypicalTags() []Tag {
	return []Tag{Admin, Read, Write, Debug, Resolve}
}

// TagStrings converts access.Tag values into []string for use with methods on
// access.Permissions.
func TagStrings(tags ...Tag) []string {
	sts := make([]string, 0, len(tags))
	for _, t := range tags {
		sts = append(sts, string(t))
	}
	return sts
}
