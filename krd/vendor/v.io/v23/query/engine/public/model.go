// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package public defines the QueryEngine interface which is returned
// from calling v.io/v23/query/engine.Create and PreparedStatement which is
// returned from the QueryEngine.PrepareStatement function.
package public

import (
	"v.io/v23/query/syncql"
	"v.io/v23/vom"
)

type QueryEngine interface {
	// Exec executes a syncQL query and returns the results. Headers (i.e., column
	// names) are returned separately from results (i.e., values).
	// q : the query (e.g., select v from Customers
	//               (e.g., delete from Customers where k = "101")
	Exec(q string) ([]string, syncql.ResultStream, error)

	// PrepareStatement parses query q and returns a PreparedStatement.  Queries passed to
	// PrepareStatement contain zero or more formal parameters (specified with a ?) for
	// operands in where clause expressions.
	// e.g., select k from Customer where Type(v) like ? and k like ?
	PrepareStatement(q string) (PreparedStatement, error)

	// Get an existing PreparedStatement from the int64 returned from calling
	// PreparedStatement.ToHandle.
	GetPreparedStatement(handle int64) (PreparedStatement, error)
}

type PreparedStatement interface {
	// Exec executes the already prepared statement with the supplied parameter values.
	// The number of paramValues supplied must match the number of formal parameters
	// specified in the query (else NotEnoughParamValuesSpecified or
	// TooManyParamValuesSpecified errors are returned).
	Exec(paramValues ...*vom.RawBytes) ([]string, syncql.ResultStream, error)

	// Handle returns an int64 handle that can be passed to QueryEngine.GetPreparedStatement
	// function.  This is useful for modules implementing query support as they need
	// not keep track of prepared statements; rather, they can send the handle to the client
	// library and have the client keep track of the statement.  When it makes it back to
	// the server, QueryEngine.GetPreparedStatement(handle) can be used to reconstruct
	// the PreparedStatement.
	Handle() int64

	// Call close to free up the space taken by the PreparedStatement when no longer
	// needed.  If close is not called, the space will be freed when the containing
	// QueryEngine is garbage collected.
	Close()
}
