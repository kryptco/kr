// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package discovery defines types and interfaces for discovering services.
package discovery

import (
	"v.io/v23/context"
	"v.io/v23/security"
)

// T is the interface for discovery operations; it is the client side library
// for the discovery service.
type T interface {
	// Advertise broadcasts the advertisement to be discovered by "Scan" operations.
	//
	// visibility is used to limit the principals that can see the advertisement. An
	// empty set means that there are no restrictions on visibility (i.e, equivalent
	// to []security.BlessingPattern{security.AllPrincipals}).
	//
	// If the advertisement id is not specified, a random unique a random unique identifier
	// will be assigned. The advertisement should not be changed while it is being advertised.
	//
	// It is an error to have simultaneously active advertisements for two identical
	// instances (Advertisement.Id).
	//
	// Advertising will continue until the context is canceled or exceeds its deadline
	// and the returned channel will be closed when it stops.
	Advertise(ctx *context.T, ad *Advertisement, visibility []security.BlessingPattern) (<-chan struct{}, error)

	// Scan scans advertisements that match the query and returns the channel of updates.
	//
	// Scan excludes the advertisements that are advertised from the same discovery
	// instance.
	//
	// The query is a WHERE expression of a syncQL query against advertisements, where
	// key is Advertisement.Id and value is Advertisement.
	//
	// Examples
	//
	//    v.InterfaceName = "v.io/i"
	//    v.InterfaceName = "v.io/i" AND v.Attributes["a"] = "v"
	//    v.Attributes["a"] = "v1" OR v.Attributes["a"] = "v2"
	//
	// SyncQL tutorial at:
	//    https://vanadium.github.io/tutorials/syncbase/syncql-tutorial.html
	//
	// Scanning will continue until the context is canceled or exceeds its deadline.
	Scan(ctx *context.T, query string) (<-chan Update, error)
}
