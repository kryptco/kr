// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package query_functions describes SyncQL's built-in functions.
//
// This package is called by the query_checker package to check that
// a function exists and is being passed the correct number of
// arguments and, if possible, the correct types of arguments.  If
// the function can be executed at check time (because
// the function has no args or because the args are all literals (or
// (recursively) other functions that can be executed at check time),
// the function is executed at check time and the result of the function
// is subsituted for the function in the AST.
//
// If the function cannot be executed at check time, it is executed by
// the query package once for each candicate row (in the case of function
// calls in the where clause) or selected row (in the case of function calls
// in the select clause).
//
// It is expected that functions will be grouped in some way into separate
// files (e.g., string functions are in str_funcs.go).
//
// Functions must be listed in the functions map in query_functions.go.
// Each entry is a function struct which contains the following fields:
//
// argTypes      []query_parser.OperandType
//               The arguments expected.  If the argument count is wrong,
//               the checker will produce an error.  The types are
//               informational only as the function itself is required
//               to attempt to coerce the args to the correct type or
//               return an error.
// hasVarArgs    bool
//               True if, in addition to any types listed in argTypes, the function
//               can take additional (optional) args.
// varArgsType   The type of the additional (optional) args.
// returnType    query_parser.OperandType
//               The return type of the function, for informational purposes
//               only.
// funcAddr      queryFunc
//               The address of the query function.
//               If the function cannot complete to success, it must return an error and the
//               argument responsible for the error.
// checkArgsAddr checkArgsFunc
//               The address of a function to check args at checker time.
//               This function should check any arguments that it can at checker time.
//               It can check literals.  Note: if all args are literals, the function itself
//               is called at checker time rather than this function.
//               DO NOT sepecify a checkArgsAddr if all that is to be checked is the number
//               and types of args. These checks are standard.
package query_functions
