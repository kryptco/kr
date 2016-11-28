// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package version defines a mechanism for versioning the RPC protocol.
package version

import (
	"v.io/v23/context"
)

// RPCVersion represents a version of the RPC protocol.
type RPCVersion uint32

const (
	// UnknownRPCVersion is used for Min/MaxRPCVersion in an Endpoint when
	// we don't know the relevant version numbers.  In this case the RPC
	// implementation will have to guess the correct values.
	UnknownRPCVersion RPCVersion = iota

	// DeprecatedRPCVersion is used to signal that a version number is no longer
	// relevant and that version information should be obtained elsewhere.
	DeprecatedRPCVersion

	rPCVersion2
	rPCVersion3
	rPCVersion4
	rPCVersion5
	rPCVersion6
	rPCVersion7
	rPCVersion8
	rPCVersion9

	// RPCVersion10 opens a special flow over which discharges for third-party
	// caveats on the server's blessings are sent.
	RPCVersion10

	// RPCVersion11 Optimized authentication.
	RPCVersion11

	// RPCVersion12 adds periodic healthchecks on the channel.
	RPCVersion12

	// RPCVersion13 adds error messages in responses from proxies.
	RPCVersion13

	// RPCVersion14 adds the setup message to the channel binding during
	// connection setup.
	RPCVersion14
)

// RPCVersionRange allows you to optionally specify a range of versions to
// use when calling FormatEndpoint
type RPCVersionRange struct {
	Min, Max RPCVersion
}

func CommonVersion(ctx *context.T, l, r RPCVersionRange) (RPCVersion, error) {
	minMax := l.Max
	if r.Max < minMax {
		minMax = r.Max
	}
	if l.Min > minMax || r.Min > minMax {
		return 0, NewErrNoCompatibleVersion(ctx,
			uint64(l.Min), uint64(l.Max), uint64(r.Min), uint64(r.Max))
	}
	return minMax, nil
}
