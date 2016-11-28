// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"v.io/v23/context"
)

// Plugin is the basic interface for discovery plugins.
//
// All implementation should be goroutine-safe.
type Plugin interface {
	// Advertise advertises the advertisement.
	//
	// The advertisement will not be changed while it is being advertised.
	//
	// If the advertisement is too large, the plugin may drop any information
	// except Id, InterfaceName, Hash, Timestamp, and DirAddrs.
	//
	// Advertising should continue until the context is canceled or exceeds
	// its deadline. done should be called once when advertising is done or
	// canceled.
	Advertise(ctx *context.T, adinfo *AdInfo, done func()) error

	// Scan scans advertisements that match the interface name and returns scanned
	// advertisements via the callback.
	//
	// An empty interface name means any advertisements.
	//
	// The callback takes ownership of the provided AdInfo, and the plugin
	// should not use the advertisement after invoking the callback.
	//
	// Scanning should continue until the context is canceled or exceeds its
	// deadline. done should be called once when scanning is done or canceled.
	Scan(ctx *context.T, interfaceName string, callback func(*AdInfo), done func()) error

	// Close closes the plugin.
	//
	// This will be called after all active tasks have been cancelled.
	Close()
}
