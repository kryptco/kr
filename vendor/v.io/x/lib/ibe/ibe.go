// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ibe implements identity-based encryption.
//
// This package defines interfaces for identity-based-encryption (IBE) and
// provides specific implementations of those interfaces.  The idea was first
// proposed in 1984 by Adi Shamir in "Identity-Based Cryptosystems And
// Signature Schemes"
// (http://discovery.csc.ncsu.edu/Courses/csc774-S08/reading-assignments/shamir84.pdf)
// and a construction based on the Bilinear Diffie-Hellman problem was
// described in "Identity-Based Encryption from the Weil Paring" by Dan Boneh
// and Matthew Franklin (http://crypto.stanford.edu/~dabo/papers/bfibe.pdf).
//
// As described in the those papers, an IBE scheme consists of four operations:
//
// (1) Setup: Which generates global system parameters and a master key.
//
// (2) Extract: Uses the master key and global system parameters to generate
// the private key corresponding to an arbitrary identity string.
//
// (3) Encrypt: Uses the global system parameters to encrypt messages for a
// particular identity.
//
// (4) Decrypt: Uses the private key (and global system parameters) to decrypt
// messages. To be clear, the private key here is for the identity (sometimes
// refered to as the "identity key" in literature) and not the master secret.
//
// This package defines 3 interfaces: one for Extract, one for Encrypt and one
// for Decrypt and provides Setup function implementations for different IBE
// systems (at the time of this writing, only for the Boneh-Boyen scheme).
package ibe

// Master is the interface used to extract private keys for arbitrary identities.
type Master interface {
	Extract(id string) (PrivateKey, error)
	Params() Params
}

// Params represents the global system parameters that are used to encrypt
// messages for a particular identity.
type Params interface {
	// Encrypt encrypts m into C for the identity id.
	//
	// The slice C must be of size len(m) + CiphertextOverhead(), and the two
	// slices must not overlap.
	Encrypt(id string, m, C []byte) error

	// The additional space required to encrypt a message, that is,
	// if a message has length m, the size of the ciphertext is
	// m + CiphertextOverhead().
	CiphertextOverhead() int
}

// PrivateKey is the interface used to decrypt encrypted messages.
type PrivateKey interface {
	// Decrypt decrypts ciphertext C into m.
	//
	// The slice m must be of size len(C) - CiphertextOverhead(), and the two
	// sices must not overlap.
	Decrypt(C, m []byte) error

	// Params returns the global system parameters of the Master that
	// was used to extract this private key.
	Params() Params
}
