// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package engine defines a Create function which returns an instance of datasource.QueryEngine
package engine

import (
	ds "v.io/v23/query/engine/datasource"
	"v.io/v23/query/engine/internal"
	"v.io/v23/query/engine/public"
)

// Create returns an instance of datasource.QueryEngine
func Create(db ds.Database) public.QueryEngine {
	return internal.Create(db)
}
