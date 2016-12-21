// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package naming defines types and utilities associated with naming.
//
//   Concept: https://vanadium.github.io/concepts/naming.html
//   Tutorial: (forthcoming)
//
// Object names are 'resolved' using a MountTable to obtain a MountedServer that
// RPC method invocations can be directed at. MountTables may be mounted on each
// other to typically create a hierarchy. The name resolution process can thus
// involve multiple MountTables. Although it is expected that a hierarchy will
// be the typical use, it is nonetheless possible to create a cyclic graph of
// MountTables which will lead to name resolution errors at runtime.
//
// Object names are strings with / used to separate the components of a name.
// Names may be started with / and the address of a MountTable or server, in
// which case they are considered 'rooted', otherwise they are 'relative' to the
// MountTable used to resolve them. Rooted names, unlike relative ones, have the
// same meaning regardless of the context in which they are accessed.
//
// The first component of a rooted name is the address of the MountTable to use
// for resolving the remaining components of the name. The address may be the
// string representation of an Endpoint, a <host>:<port>, or <ip>:<port>. In
// addition, <host> or <ip> may be used without a <port> being specified in
// which case a default port is used. The portion of the name following the
// address is a relative name.
//
// Thus:
//
// /host:port/a/b/c/d means starting at host:port resolve a/b/c/d and return the
// terminating server and the relative path from that server.
package naming
