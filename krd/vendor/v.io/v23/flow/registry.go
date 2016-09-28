// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flow

import (
	"fmt"
	"sync"
	"time"

	"v.io/v23/context"
	"v.io/v23/naming"
)

// Protocol is the interface that protocols for use with vanadium RPCs must implement.
type Protocol interface {
	// Dial is the function used to create Conn objects given a
	// protocol-specific string representation of an address.
	// The returned Conn must also frame the connection.
	Dial(ctx *context.T, protocol, address string, timeout time.Duration) (Conn, error)
	// Resolve is the function used for protocol-specific address normalization.
	// e.g. the TCP resolve performs DNS resolution.
	// Resolve returns the protocol and resolved addresses.
	Resolve(ctx *context.T, protocol, address string) (string, []string, error)
	// Listen is the function used to create Listener objects given a
	// protocol-specific string representation of the address a server will listen on.
	// The Conns returned from Listener must frame connections.
	Listen(ctx *context.T, protocol, address string) (Listener, error)
}

// RegisterProtocol makes available a Protocol to RegisteredProtocol.
// If the protocol represents other actual protocols, you need to specify all the
// actual protocols. E.g, "wsh" represents "tcp4", "tcp6", "ws4", and "ws6".
//
// Implementations of the Manager interface are expected to use this registry
// in order to expand the reach of the types of network protocols they can
// handle.
//
// Successive calls to RegisterProtocol replace the contents of a previous
// call to it and returns trues if a previous value was replaced, false otherwise.
func RegisterProtocol(protocol string, obj Protocol, p ...string) bool {
	// This is for handling the common case where protocol is a "singleton", to
	// make it easier to specify.
	if len(p) == 0 {
		p = []string{protocol}
	}
	registryLock.Lock()
	defer registryLock.Unlock()
	_, present := registry[protocol]
	registry[protocol] = registryEntry{obj, p}
	return present
}

// RegisterUnknownProtocol registers a Protocol for endpoints with
// no specified protocol.
//
// The desired protocol provided in the first argument will be passed to the
// Protocol methods as the actual protocol to use when dialing, resolving, or listening.
//
// The protocol itself must have already been registered before RegisterUnknownProtocol
// is called, otherwise we'll panic.
func RegisterUnknownProtocol(protocol string, obj Protocol) bool {
	var p []string
	registryLock.RLock()
	r, present := registry[protocol]
	if !present {
		panic(fmt.Sprintf("%s not registered", protocol))
	}
	p = r.p
	registryLock.RUnlock()
	return RegisterProtocol(naming.UnknownProtocol, wrappedProtocol{protocol, obj}, p...)
}

type wrappedProtocol struct {
	protocol string
	obj      Protocol
}

func (p wrappedProtocol) Dial(ctx *context.T, _, address string, timeout time.Duration) (Conn, error) {
	return p.obj.Dial(ctx, p.protocol, address, timeout)
}

func (p wrappedProtocol) Resolve(ctx *context.T, _, address string) (string, []string, error) {
	return p.obj.Resolve(ctx, p.protocol, address)
}

func (p wrappedProtocol) Listen(ctx *context.T, _, address string) (Listener, error) {
	return p.obj.Listen(ctx, p.protocol, address)
}

// RegisteredProtocol returns the Protocol object registered with a
// previous call to RegisterProtocol.
func RegisteredProtocol(protocol string) (Protocol, []string) {
	registryLock.RLock()
	e := registry[protocol]
	registryLock.RUnlock()
	return e.obj, e.p
}

// RegisteredProtocols returns the list of protocols that have been previously
// registered using RegisterProtocol. The underlying implementation will
// support additional protocols such as those supported by the native RPC stack.
func RegisteredProtocols() []string {
	registryLock.RLock()
	defer registryLock.RUnlock()
	p := make([]string, 0, len(registry))
	for k, _ := range registry {
		p = append(p, k)
	}
	return p
}

type registryEntry struct {
	obj Protocol
	p   []string
}

var (
	registryLock sync.RWMutex
	registry     = make(map[string]registryEntry)
)
