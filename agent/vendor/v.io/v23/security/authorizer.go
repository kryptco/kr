// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package security

import (
	"reflect"

	"v.io/v23/context"
)

// DefaultAuthorizer returns an Authorizer that implements a "reasonably secure"
// authorization policy that can be used whenever in doubt.
//
// It has the conservative policy that requires one end of the RPC to have a
// blessing that is extended from the blessing presented by the other end.
func DefaultAuthorizer() Authorizer {
	return defaultAuthorizer{}
}

type defaultAuthorizer struct{}

func (defaultAuthorizer) Authorize(ctx *context.T, call Call) error {
	var (
		localNames             = LocalBlessingNames(ctx, call)
		remoteNames, remoteErr = RemoteBlessingNames(ctx, call)
	)
	// Authorize if any element in localNames is a "delegate of" (i.e., has been
	// blessed by) any element in remoteNames, OR vice-versa.
	for _, l := range localNames {
		if BlessingPattern(l).MatchedBy(remoteNames...) {
			// One of remoteNames is an extension of l.
			return nil
		}
	}
	for _, r := range remoteNames {
		if BlessingPattern(r).MatchedBy(localNames...) {
			// One of localNames is an extension of r.
			return nil
		}
	}

	return NewErrAuthorizationFailed(ctx, remoteNames, remoteErr, localNames)
}

// AllowEveryone returns an Authorizer which implements a policy of always
// allowing access - irrespective of any parameters of the call or the
// blessings of the caller.
func AllowEveryone() Authorizer {
	return allowEveryone{}
}

type allowEveryone struct{}

func (allowEveryone) Authorize(*context.T, Call) error { return nil }

// PublicKeyAuthorizer only authorizes principals with a specific public key.
//
// Normally, authorizations in Vanadium should be based on blessing names and not
// public keys, since the former are resilient to key rotations and process
// replication. However, in rare circumstances it may be possible that blessing names
// cannot be used (for example, if the local end does not recognize the remote end's
// blessing root), and the PublicKey might be usable instead.
func PublicKeyAuthorizer(key PublicKey) Authorizer {
	return publicKeyAuthorizer{key}
}

type publicKeyAuthorizer struct{ key PublicKey }

func (a publicKeyAuthorizer) Authorize(ctx *context.T, call Call) error {
	remote := call.RemoteBlessings().PublicKey()
	if remote == nil {
		return NewErrPublicKeyNotAllowed(ctx, "", a.key.String())
	}
	if !reflect.DeepEqual(remote, a.key) {
		return NewErrPublicKeyNotAllowed(ctx, remote.String(), a.key.String())
	}
	return nil
}

// EndpointAuthorizer authorizes principals iff they present blessings that
// match those specified in call.RemoteEndpoint().
func EndpointAuthorizer() Authorizer {
	return endpointAuthorizer{}
}

type endpointAuthorizer struct{}

func (endpointAuthorizer) Authorize(ctx *context.T, call Call) error {
	patterns := call.RemoteEndpoint().BlessingNames()
	if len(patterns) == 0 {
		return nil
	}
	names, rejected := RemoteBlessingNames(ctx, call)
	for _, p := range patterns {
		if BlessingPattern(p).MatchedBy(names...) {
			return nil
		}
	}
	return NewErrEndpointAuthorizationFailed(ctx, call.RemoteEndpoint().String(), names, rejected)
}
