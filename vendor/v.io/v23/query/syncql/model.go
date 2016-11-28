// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The ResultStream interface is used to iterate over query results.
package syncql

import (
	"v.io/v23/vom"
)

// ResultStream is an interface for iterating through results (i.e., rows)
// returned from a query. Each resulting row is an array of vdl.Values.
type ResultStream interface {
	// Advance stages an element so the client can retrieve it with Result.
	// Advance returns true iff there is a result to retrieve. The client must
	// call Advance before calling Result. The client must call Cancel if it
	// does not iterate through all elements (i.e. until Advance returns false).
	// Advance may block if an element is not immediately available.
	Advance() bool

	// Result returns the row (i.e., array of vdl.Values) that was staged by
	// Advance. Result may panic if Advance returned false or was not called at
	// all. Result does not block.
	Result() []*vom.RawBytes

	// Err returns a non-nil error iff the stream encountered any errors. Err does
	// not block.
	Err() error

	// Cancel notifies the ResultStream provider that it can stop producing
	// results. The client must call Cancel if it does not iterate through all
	// results (i.e. until Advance returns false). Cancel is idempotent and can be
	// called concurrently with a goroutine that is iterating via Advance/Result.
	// Cancel causes Advance to subsequently return false. Cancel does not block.
	Cancel()
}
