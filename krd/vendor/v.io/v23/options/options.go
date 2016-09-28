// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package options defines common options recognized by vanadium implementations.
//
// Below are the common options required of all vanadium implementations.  Let's
// say we have functions MyFuncA and MyFuncB in package demo:
//
//   package demo
//   func MyFuncA(a, b, c int, opts ...MyFuncAOpt)
//   func MyFuncB(opts ...MyFuncBOpt)
//
//   type MyFuncAOpt interface {
//     DemoMyFuncAOpt()
//   }
//   type MyFuncBOpt interface {
//     DemoMyFuncBOpt()
//   }
//
// The MyFuncAOpt interface is used solely to constrain the types of options
// that MyFuncA accepts, and ditto for MyFuncBOpt and MyFuncB.  In order to
// enable an option to be accepted by a particular function, you simply add a
// no-op function definition with the appropriate name.  An example:
//
//   type Foo int
//   func (Foo) DemoMyFuncAOpt() {}
//   func (Foo) DemoMyFuncBOpt() {}
//
//   type Bar string
//   func (Bar) DemoMyFuncBOpt() {}
//
// Foo is accepted by both demo.MyFuncA and demo.MyFuncB, while Bar is only
// accepted by demo.MyFuncB.  The methods defined for each option essentially
// act as annotations telling us which functions will accept them.
//
// Go stipulates that methods may only be attached to named types, and the type
// may not be an interface.  E.g.
//
//   // BAD: can't attach methods to named interfaces.
//   type Bad interface{}
//   func (Bad) DemoMyFuncAOpt() {}
//
//   // GOOD: wrap the interface in a named struct.
//   type Good struct { val interface{} }
//
//   func (Good) DemoMyFuncAOpt() {}
//
// These options can then be passed to the function as so:
//   MyFuncA(a, b, c, Foo(1), Good{object})
package options

import (
	"time"

	"v.io/v23/naming"
	"v.io/v23/security"
)

// ServerPeers is the set of peers to whom a process (a "server") accepting
// network connections must reveal its blessings.
//
// If this option is not provided, then the server's blessings will be revealed
// to all processes (clients) from whom the server accepts network connections.
//
// NOTE: This is an experimental option and may go away at any time.
type ServerPeers []security.BlessingPattern

func (ServerPeers) RPCServerOpt() {}

// ServerAuthorizer encapsulates the authorization policy used by a client to
// authorize the end server of an RPC.
//
// This policy is applied before the client sends information about itself
// (public key, blessings, the RPC request) to the server. Thus, if a server
// does not satisfy this policy then the client will abort the request.
//
// Authorization of other servers communicated with in the process of
// contacting the end server are controlled by other options, like
// NameResolutionAuthorizer.
//
// Runtime implementations are expected to use security.EndpointAuthorizer
// if no explicit ServerAuthorizer has been provided for the call.
type ServerAuthorizer struct{ security.Authorizer }

func (ServerAuthorizer) RPCCallOpt() {}

// NameResolutionAuthorizer encapsulates the authorization policy used by a
// client to authorize mounttable servers before sending them a name resolution
// request. By specifying this policy, clients avoid revealing the names they
// are interested in resolving to unauthorized mounttables.
//
// If no such option is provided, then runtime implementations are expected to
// default to security.EndpointAuthorizer.
type NameResolutionAuthorizer struct{ security.Authorizer }

func (NameResolutionAuthorizer) RPCCallOpt() {}
func (NameResolutionAuthorizer) NSOpt()      {}

// Preresolved specifies that the RPC call should not further Resolve the name.
// If a MountEntry is provided, use it.  Otherwise use the name passed in the
// RPC call.  If the name is relative, it will be made global using
// the roots in the namespace.
type Preresolved struct {
	Resolution *naming.MountEntry
}

func (Preresolved) RPCCallOpt() {}
func (Preresolved) NSOpt()      {}

// Create a server that will be used to serve a MountTable. This server
// cannot be used for any other purpose.
type ServesMountTable bool

func (ServesMountTable) RPCServerOpt() {}

// LameDuckTimeout specifies the time to wait for all server operations to complete after Stop is called.
type LameDuckTimeout time.Duration

func (LameDuckTimeout) RPCServerOpt() {}

// Create a server that will be used to serve a leaf service.
type IsLeaf bool

func (IsLeaf) RPCServerOpt() {}

// When NoRetry is specified, the client will not retry calls that fail but would
// normally be retried.
type NoRetry struct{}

func (NoRetry) NSOpt()      {}
func (NoRetry) RPCCallOpt() {}

// ChannelTimeout is the amount of time before we notice that a channel is
// not responsive and close it.  Note that ChannelTimeout(0) is the same as
// not setting a timeout.
type ChannelTimeout time.Duration

func (ChannelTimeout) RPCCallOpt()   {}
func (ChannelTimeout) RPCServerOpt() {}

// ConnectionTimeout is the amount of time we will try establishing a connection
// to the remote end during an RPC. Zero means only use cached connections and
// do not attempt to retry if no connection exists in the cache.
type ConnectionTimeout time.Duration

func (ConnectionTimeout) RPCCallOpt() {}
