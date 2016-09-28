// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package vdl implements the Vanadium Definition Language type and value
// system.
//
//   Concept: https://vanadium.github.io/concepts/rpc.html#vdl
//   Specification: https://vanadium.github.io/designdocs/vdl-spec.html
//
// VDL is an interface definition language designed to enable interoperation
// between clients and servers executing in heterogeneous environments.  E.g. it
// enables a frontend written in Javascript running on a phone to communicate
// with a backend written in Go running on a server.  VDL is compiled into an
// intermediate representation that is used to generate code in each target
// environment.
//
// The concepts in VDL are similar to the concepts used in general-purpose
// languages to specify interfaces and communication protocols.
package vdl
