// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package security defines types and utilities associated with security.
//
//   Concept: https://vanadium.github.io/concepts/security.html
//   Tutorial: (forthcoming)
//
// The primitives and APIs defined in this package enable bi-directional,
// end-to-end authentication between communicating parties; authorization based
// on that authentication; and secrecy and integrity of all communication.
//
// Overview
//
// The Vanadium security model is centered around the concepts of principals
// and blessings.
//
// A principal in the Vanadium framework is a public and private key pair.
// Every RPC is executed on behalf of a principal.
//
// A blessing is a binding of a human-readable name to a principal, valid under
// some caveats, given by another principal. A principal can have multiple
// blessings bound to it. For instance, a television principal may have a
// blessing from the manufacturer (e.g., popularcorp:products:tv) as well as
// from the owner (e.g., alice:devices:hometv). Principals are authorized for
// operations based on the blessings bound to them.
//
// A principal can "bless" another principal by binding an extension of one of
// its own blessings to the other principal. This enables delegation of
// authority. For example, a principal with the blessing
// "johndoe" can delegate to his phone by blessing the phone as
// "johndoe:phone", which in-turn can delegate to the headset by blessing it as
// "johndoe:phone:headset".
//
// Caveats can be added to a blessing in order to restrict the contexts in which
// it can be used. Amongst other things, caveats can restrict the duration of use and
// the set of peers that can be communicated with using a blessing.
//
// Navigating the interfaces
//
// Godoc renders all interfaces in this package in alphabetical order.
// However, we recommend the following order in order to introduce yourself to
// the API:
//
//   * Principal
//   * Blessings
//   * BlessingStore
//   * BlessingRoots
//   * NewCaveat
//   * ThirdPartyCaveat
//   * NewPublicKeyCaveat
//
// Examples
//
// A principal can decide to name itself anything it wants:
//  // (in process A)
//  var p1 Principal
//  alice, _ := p1.BlessSelf("alice")
//
// This "alice" blessing can be presented to to another principal (typically a
// remote process), but that other principal will not recognize this
// "self-proclaimed" authority:
//  // (in process B)
//  var p2 Principal
//  ctx, call := GetContextAndCall() // current context and security state
//  names, rejected := RemoteBlessingNames(ctx, call)
//  fmt.Printf("%v %v", names, rejected) // Will print [] ["alice": "..."]
//
// However, p2 can decide to trust the roots of the "alice" blessing and then it
// will be able to recognize her delegates as well:
//  // (in process B)
//  AddToRoots(p2, call.RemoteBlessings())
//  names, rejected := RemoteBlessingNames(ctx, call)
//  fmt.Printf("%v %v", names, rejected) // Will print ["alice"] []
//
// Furthermore, p2 can seek a blessing from "alice":
//  // (in process A)
//  call := GetCall() // Call under which p2 is seeking a blessing from alice, call.LocalPrincipal = p1
//  key2 := call.RemoteBlessings().PublicKey()
//  onlyFor10Minutes := NewExpiryCaveat(time.Now().Add(10*time.Minute))
//  aliceFriend, _ := p1.Bless(key2, alice, "friend", onlyFor10Minutes)
//  SendBlessingToProcessB(aliceFriend)
//
// p2 can then add this blessing to its store such that this blessing will be
// presented to "alice" (and her delegates) anytime p2 communicates with it in
// the future:
//  // (in process B)
//  p2.BlessingStore().Set(aliceFriend, "alice")
//
// p2 can also choose to present multiple blessings to some servers:
//  // (in process B)
//  charlieFriend := ReceiveBlessingFromSomeWhere()
//  union, _ := UnionOfBlessings(aliceFriend, charlieFriend)
//  p2.BlessingStore().Set(union, "alice:mom")
//
// Thus, when communicating with a "server" that presents the blessing "alice:mom",
// p2 will declare that he is both "alice's friend" and "charlie's friend" and
// the server may authorize actions based on this fact.
//
// p2 may also choose that it wants to present these two blessings when acting
// as a "server", (i.e., when it does not know who the peer is):
//  // (in process B)
//  default, _ := UnionOfBlessings(aliceFriend, charlieFriend)
//  p2.BlessingStore().SetDefault(default)
package security

import (
	"time"

	"v.io/v23/context"
	"v.io/v23/naming"
	"v.io/v23/vdl"
)

// Principal represents an entity capable of making or receiving RPCs.
// Principals have a unique (public, private) key pair, have (zero or more)
// blessings bound to them and can bless other principals.
//
// Multiple goroutines may invoke methods on a Principal simultaneously.
//
// See also: https://vanadium.github.io/glossary.html#principal
type Principal interface {
	// Bless binds extensions of blessings held by this principal to
	// another principal (represented by its public key).
	//
	// For example, a principal with the blessings "google:alice"
	// and "v23:alice" can bind the blessings "google:alice:friend"
	// and "v23:alice:friend" to another principal using:
	//   Bless(<other principal>, <google:alice, v23:alice>, "friend", ...)
	//
	// To discourage unconstrained delegation of authority, the interface
	// requires at least one caveat to be provided. If unconstrained delegation
	// is desired, the UnconstrainedUse function can be used to produce
	// this argument.
	//
	// with.PublicKey must be the same as the principal's public key.
	Bless(key PublicKey, with Blessings, extension string, caveat Caveat, additionalCaveats ...Caveat) (Blessings, error)

	// BlessSelf creates a blessing with the provided name for this principal.
	BlessSelf(name string, caveats ...Caveat) (Blessings, error)

	// Sign uses the private key of the principal to sign message.
	Sign(message []byte) (Signature, error)

	// MintDischarge generates a discharge for 'tp'.
	//
	// It assumes that it is okay to generate a discharge, i.e., any
	// restrictions encoded within 'tp' are satisfied.
	//
	// The returned discharge will be usable only if the provided caveats
	// are met when using the discharge.
	MintDischarge(forThirdPartyCaveat, caveatOnDischarge Caveat, additionalCaveatsOnDischarge ...Caveat) (Discharge, error)

	// PublicKey returns the public key counterpart of the private key held
	// by the Principal.
	PublicKey() PublicKey

	// BlessingStore provides access to the BlessingStore containing blessings
	// that have been granted to this principal.
	BlessingStore() BlessingStore

	// Roots returns the set of recognized authorities (identified by their
	// public keys) on blessings that match specific patterns
	Roots() BlessingRoots
}

// BlessingStore is the interface for storing blessings bound to a
// principal and managing the subset of blessings to be presented to
// particular peers.
// BlessingStore implementations may also cache Discharges for third-party caveats
// on blessings, allowing unexpired Discharges to be reused.
type BlessingStore interface {
	// Set marks the set of blessings to be shared with peers.
	//
	// Set(b, pattern) marks the intention to reveal b to peers
	// who present blessings of their own matching pattern.
	//
	// If multiple calls to Set are made with the same pattern, the
	// last call prevails.
	//
	// Set(Blessings{}, pattern) can be used to remove the blessings
	// previously associated with the pattern (by a prior call to Set).
	//
	// It is an error to call Set with "blessings" whose public key does
	// not match the PublicKey of the principal for which this store hosts
	// blessings.
	//
	// Set returns the Blessings object which was previously associated
	// with the pattern.
	Set(blessings Blessings, forPeers BlessingPattern) (Blessings, error)

	// ForPeer returns the set of blessings that have been previously
	// added to the store with an intent of being shared with peers
	// that have at least one of the provided blessings.
	//
	// If no peerBlessings are provided then blessings marked for all peers
	// (i.e., added with the AllPrincipals pattern) is returned.
	//
	// Returns the zero value if there are no matching blessings in the store.
	ForPeer(peerBlessings ...string) Blessings

	// SetDefault sets up the Blessings made available on a subsequent call
	// to Default.
	//
	// It is an error to call SetDefault with a blessings whose public key
	// does not match the PublicKey of the principal for which this store
	// hosts blessings.
	SetDefault(blessings Blessings) error

	// Default returns the blessings to be shared with peers for which no
	// other information is available in order to select blessings from the
	// store.
	//
	// For example, Default can be used by servers to identify themselves
	// to clients before the client has identified itself.
	//
	// Default returns the blessings provided to the last call to
	// SetDefault and a channel which will be closed when the
	// Default changes (i.e., SetDefault is called again).
	//
	// Returns the zero value if there is no usable blessings.
	Default() (Blessings, <-chan struct{})

	// PublicKey returns the public key of the Principal for which
	// this store hosts blessings.
	PublicKey() PublicKey

	// PeerBlessings returns all the blessings that the BlessingStore
	// currently holds for various peers.
	PeerBlessings() map[BlessingPattern]Blessings

	// CacheDischarge inserts the discharge for the provided impetus and caveat into the cache.
	CacheDischarge(discharge Discharge, caveat Caveat, impetus DischargeImpetus)

	// ClearDischarges clears the input discharges from the BlessingStore's
	// discharge cache.
	ClearDischarges(discharges ...Discharge)

	// Discharge takes a caveat and DischargeImpetus and returns a cached discharge
	// and the time at which it was cached.  Zero values are returned if no
	// corresponding cached discharge can be found.  Note that in certain upgrade
	// situations a zero cacheTime may be returned if the real cache time cannot
	// be determined.
	Discharge(caveat Caveat, impetus DischargeImpetus) (discharge Discharge, cacheTime time.Time)

	// DebugString return a human-readable string description of the store.
	// This description is detailed and lists out the contents of the store.
	// Use fmt.Sprintf("%v", ...) for a more succinct description.
	DebugString() string
}

// BlessingRoots hosts the set of authoritative public keys for roots
// of blessings.
//
// See also: https://vanadium.github.io/glossary.html#blessing-root
type BlessingRoots interface {
	// Add marks 'root' (a DER-encoded public key) as an authoritative key
	// for blessings that match 'pattern'.
	//
	// Multiple keys can be added for the same pattern, in which
	// case all those keys are considered authoritative for
	// blessings that match the pattern.
	Add(root []byte, pattern BlessingPattern) error

	// Recognized returns nil iff the provided (DER-encoded) root public
	// key as an authority on a pattern that is matched by blessing.
	Recognized(root []byte, blessing string) error

	// Dump returns the set of recognized roots as a map from
	// blessing patterns to the set of authoritative keys for that
	// pattern.
	Dump() map[BlessingPattern][]PublicKey

	// DebugString returns a human-readable string description of the roots.
	// This description is detailed and lists out all the roots. Use
	// fmt.Sprintf("%v", ...) for a more succinct description.
	DebugString() string
}

// Signer is the interface for signing arbitrary length messages using private keys.
//
// Multiple goroutines may invoke methods on a Signer simultaneously.
type Signer interface {
	// Sign signs an arbitrary length message using the private key associated
	// with this Signer.
	//
	// The provided purpose is used to avoid "type attacks", wherein an honest
	// entity is cheated into interpreting a field in a message as one with a
	// type other than the intended one.
	Sign(purpose, message []byte) (Signature, error)

	// PublicKey returns the public key corresponding to the Signer's private key.
	PublicKey() PublicKey
}

// ThirdPartyCaveat is a restriction on the applicability of a blessing that is
// considered satisfied only when accompanied with a specific "discharge" from
// the third-party specified in the caveat. (The first two parties are the ones
// presenting a blessing and the one making authorization decisions based on
// the blessing presented).
//
// Multiple goroutines may invoke methods on a ThirdPartyCaveat simultaneously.
//
// See also: https://vanadium.github.io/glossary.html#third-party-caveat
type ThirdPartyCaveat interface {
	// ID returns a cryptographically unique identifier for the ThirdPartCaveat.
	ID() string

	// Location returns the Vanadium object name of the discharging third-party.
	Location() string

	// Requirements lists the information that the third-party requires
	// in order to issue a discharge.
	Requirements() ThirdPartyRequirements

	// Dischargeable validates all restrictions encoded within the third-party
	// caveat under the current call and returns nil iff they have been satisfied,
	// and thus ensures that it is okay to generate a discharge for this
	// ThirdPartyCaveat.
	//
	// It assumes that the ThirdPartCaveat was obtained from the remote end of
	// call.
	Dischargeable(ctx *context.T, call Call) error
}

// Call defines the state available for authorizing a principal.
type Call interface {
	// Timestamp returns the time at which the authorization state is to be checked.
	Timestamp() time.Time
	// Method returns the method being invoked.
	Method() string
	// MethodTags returns the tags attached to the method, typically through the
	// interface specification in VDL.
	MethodTags() []*vdl.Value
	// Suffix returns the object name suffix for the request.
	Suffix() string
	// LocalDischarges specify discharges for third-party caveats presented by
	// the local end of the call. It maps a third-party caveat identifier to the
	// corresponding discharge.
	LocalDischarges() map[string]Discharge
	// RemoteDischarges specify discharges for third-party caveats presented by
	// the remote end of the call. It maps a third-party caveat identifier to the
	// corresponding discharge.
	RemoteDischarges() map[string]Discharge
	// LocalPrincipal returns the principal used to authenticate to the remote end.
	LocalPrincipal() Principal
	// LocalBlessings returns the blessings (bound to the local end)
	// provided to the remote end for authentication.
	LocalBlessings() Blessings
	// RemoteBlessings returns the blessings (bound to the remote end)
	// provided to the local end during authentication.
	RemoteBlessings() Blessings
	// LocalEndpoint() returns the Endpoint of the principal at the local
	// end of communication.
	LocalEndpoint() naming.Endpoint
	// RemoteEndpoint() returns the Endpoint of the principal at the remote end
	// of communication.
	RemoteEndpoint() naming.Endpoint
}

// Authorizer is the interface for performing authorization checks.
type Authorizer interface {
	Authorize(ctx *context.T, call Call) error
}
