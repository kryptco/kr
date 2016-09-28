// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vom

import "v.io/v23/vdl"

// hasChunkLen returns true iff the type t is encoded with a top-level
// chunk length.
func hasChunkLen(t *vdl.Type) bool {
	if t.IsBytes() {
		// TODO(bprosnitz) This should probably be chunked
		return false
	}
	switch t.Kind() {
	case vdl.Array, vdl.List, vdl.Set, vdl.Map, vdl.Struct, vdl.Any, vdl.Union, vdl.Optional:
		return true
	}
	return false
}

// isAllowedVersion returns true if the VOM version specified is supported.
func isAllowedVersion(version Version) bool {
	return version == Version81
}

var anyKindList []vdl.Kind = []vdl.Kind{vdl.Any}
var typeObjectKindList []vdl.Kind = []vdl.Kind{vdl.TypeObject}

// containsAny returns true if the provided type contains an any recursively within it.
func containsAny(t *vdl.Type) bool {
	return t.ContainsKind(vdl.WalkAll, anyKindList...)
}

// containsTypeObject returns true if the provided type contains an any recursively within it.
func containsTypeObject(t *vdl.Type) bool {
	return t.ContainsKind(vdl.WalkAll, typeObjectKindList...)
}
