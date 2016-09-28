// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query_functions

import (
	ds "v.io/v23/query/engine/datasource"
	"v.io/v23/query/engine/internal/query_parser"
	"v.io/v23/query/syncql"
	"v.io/v23/vdl"
)

// len returns the size of the argument passed in.
// For vdl's Array, List, Map and Set, it returns the number of entries.
// For TypString, it returns the number of bytes in the string.
// For TypNil, it returns 0.
// For all other types, it returns an error.
// e.g., Len("abc") returns 3
func lenFunc(db ds.Database, off int64, args []*query_parser.Operand) (*query_parser.Operand, error) {
	switch args[0].Type {
	case query_parser.TypNil:
		return makeIntOp(args[0].Off, 0), nil
	case query_parser.TypObject:
		switch args[0].Object.Kind() {
		case vdl.Array, vdl.List, vdl.Map, vdl.Set:
			// If array, list, set, map, call Value.Len()
			return makeIntOp(args[0].Off, int64(args[0].Object.Len())), nil
		}
	case query_parser.TypStr:
		// If string, call go's built-in len().
		return makeIntOp(args[0].Off, int64(len(args[0].Str))), nil
	}
	return nil, syncql.NewErrFunctionLenInvalidArg(db.GetContext(), args[0].Off)
}
