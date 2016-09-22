// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file defines an implementation of the IBE interfaces using the
// Boneh-Boyen scheme. The Fujisaki-Okamoto transformation is applied to
// obtain CCA2-security. The paper defining this algorithm (see comments for
// SetupBB1) uses multiplicative groups while the bn256 package used in the
// implementation here defines an additive group. The comments follow the
// notation in the paper while the code uses the bn256 library. For example,
// g^i corresponds to G1.ScalarBaseMult(i).

// The ciphertexts in the resulting CCA2-secure IBE scheme will consist of
// two parts: an IBE encryption of a symmetric key (called the 'kem' - key
// encapsulation mechanism) and a symmetric encryption of the payload
// (called the 'dem' - data encapsulation mechanism).

package ibe

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bn256"
	"golang.org/x/crypto/nacl/secretbox"
)

var errBadCiphertext = errors.New("invalid ciphertext")

const (
	marshaledG1Size = 2 * 32
	marshaledG2Size = 4 * 32
	marshaledGTSize = 12 * 32
	nonceSize       = 24
	encKeySize      = sha256.Size                    // size of encapsulated key = 32 bytes
	kemSize         = encKeySize + 2*marshaledG1Size // 160 bytes
)

// In the construction, we require several independent hash functions for
// hashing the different quantities (in the security reduction, these
// hash functions are modeled as random oracles). In the implementation,
// we use SHA-256 as the underlying hash function, but concatenate a prefix
// to differentiate the different hash functions. For example, we have
// H1(x) = SHA256(00 || x), H2(x) = SHA256(01 || x), and so on. If we model
// SHA256 as a random oracle, then H1, H2, ... are also independent random
// oracles.
var (
	idPrefix  = [1]byte{0x00}
	kemPrefix = [1]byte{0x01}
	demPrefix = [1]byte{0x02}
	ibePrefix = [1]byte{0x03}
)

// Setup creates an ibe.Master based on the BB1 scheme described in "Efficient
// Selective Identity-Based Encryption Without Random Oracles" by Dan Boneh and
// Xavier Boyen (http://crypto.stanford.edu/~dabo/papers/bbibe.pdf).
//
// Specifically, Section 4.3 of the paper is implemented.
//
// In addition, we apply the Fujisaki-Okamoto transformation to the BB-IBE
// scheme (http://link.springer.com/chapter/10.1007%2F3-540-48405-1_34)
// in order to obtain CCA2-security (in the random oracle model). The resulting
// scheme is a CCA2-secure hybrid encryption scheme where BB-IBE is used to
// encrypt a nonce used to derive a symmetric key, and NaCl/secretbox is used
// to encrypt the data under the symmetric key.
func SetupBB1() (Master, error) {
	var (
		m     = &bb1master{params: newbb1params(), g0Hat: new(bn256.G2)}
		pk    = m.params // shorthand
		g0Hat = m.g0Hat  // shorthand
	)

	// Set generators
	pk.g.ScalarBaseMult(big.NewInt(1))
	pk.gHat.ScalarBaseMult(big.NewInt(1))

	// Pick a random alpha and set g1 & g1Hat
	alpha, err := random()
	if err != nil {
		return nil, err
	}
	pk.g1.ScalarBaseMult(alpha)
	pk.g1Hat.ScalarBaseMult(alpha)

	// Pick a random delta and set h and hHat
	delta, err := random()
	if err != nil {
		return nil, err
	}
	pk.h.ScalarBaseMult(delta)
	pk.hHat.ScalarBaseMult(delta)

	// Pick a random beta and set g0Hat.
	beta, err := random()
	if err != nil {
		return nil, err
	}
	alphabeta := new(big.Int).Mul(alpha, beta)
	g0Hat.ScalarBaseMult(alphabeta.Mod(alphabeta, bn256.Order)) // g0Hat = gHat^*(alpha*beta)

	pk.v = bn256.Pair(pk.g, g0Hat)
	return m, nil
}

type bb1master struct {
	params *bb1params // Public params
	g0Hat  *bn256.G2  // Master key
}

func (m *bb1master) Extract(id string) (PrivateKey, error) {
	r, err := random()
	if err != nil {
		return nil, err
	}

	var (
		ret = &bb1PrivateKey{
			params: m.params,
			d0:     new(bn256.G2),
			d1:     new(bn256.G2),
		}
		// A bunch of shorthands
		d0    = ret.d0
		g1Hat = m.params.g1Hat
		g0Hat = m.g0Hat
		hHat  = m.params.hHat
		i     = val2bignum(idPrefix, []byte(id))
	)
	// ret.d0 = g0Hat * (g1Hat^i * hHat)^r
	d0.ScalarMult(g1Hat, i)
	d0.Add(d0, hHat)
	d0.ScalarMult(d0, r)
	ret.d0.Add(d0, g0Hat)
	ret.d1.ScalarBaseMult(r)
	return ret, nil
}

func (m *bb1master) Params() Params { return m.params }

type bb1params struct {
	g, g1, h          *bn256.G1
	gHat, g1Hat, hHat *bn256.G2
	v                 *bn256.GT
}

func newbb1params() *bb1params {
	return &bb1params{
		g:     new(bn256.G1),
		g1:    new(bn256.G1),
		h:     new(bn256.G1),
		gHat:  new(bn256.G2),
		g1Hat: new(bn256.G2),
		hHat:  new(bn256.G2),
		v:     new(bn256.GT),
	}
}

// Helper method that checks that the ciphertext slice for a given message has
// the correct size: len(C) = len(m) + CiphertextOverhead()
func checkSizes(m, C []byte, params Params) error {
	if msize, Csize := len(m), len(C); Csize != msize+params.CiphertextOverhead() {
		return fmt.Errorf("provided plaintext and ciphertext are of sizes (%d, %d), ciphertext size should be %d", msize, Csize, params.CiphertextOverhead())
	}
	return nil
}

// Helper method that constructs the first two components of the BB-IBE
// ciphertext. These components are re-computed during decryption to verify
// proper generation of the ciphertext. This is the Fujisaki-Okamoto transformation.
// The ciphertext C that is passed in must have size exactly
// encKeySize + marshaledG1Size.
func (e *bb1params) encapsulateKeyStart(sigma *[encKeySize]byte, s *big.Int, C []byte) error {
	if len(C) != encKeySize+marshaledG1Size {
		return fmt.Errorf("provided buffer has size %d, must be %d", len(C), encKeySize+marshaledG1Size)
	}

	var (
		vs    = new(bn256.GT)
		tmpG1 = new(bn256.G1)
		// Ciphertext C = (A, B, C1) - this method computes the first two components
		A = C[0:encKeySize]
		B = C[encKeySize : encKeySize+marshaledG1Size]
	)
	vs.ScalarMult(e.v, s)
	pad := hashval(ibePrefix, vs.Marshal())
	// A = sigma ⊕ H(v^s)
	for i := range sigma {
		A[i] = sigma[i] ^ pad[i]
	}
	// B = g^s
	if err := marshalG1(B, tmpG1.ScalarBaseMult(s)); err != nil {
		return err
	}

	return nil
}

// Compute the symmetric key used for data encapsulation (used to encrypt the payload
// during Encrypt and for verification during Decrypt)
func computeDemKey(sigma *[encKeySize]byte) *[encKeySize]byte {
	return hashval(demPrefix, sigma[:])
}

// Computes the randomness used to encrypt the symmetric key. The randomness is given
// by H(sigma || m).
func computeKemRandomness(sigma *[encKeySize]byte, m []byte) *big.Int {
	// Note: append(sigma[:], m) will allocate a new slice and copy the data
	// since sigma is a fixed-size array (not a slice with larger capacity). However,
	// if this changes in the future, this should be changed to ensure thread-safety.
	return val2bignum(kemPrefix, append(sigma[:], m...))
}

func (e *bb1params) Encrypt(id string, m, C []byte) error {
	if err := checkSizes(m, C, e); err != nil {
		return err
	}

	// Choose a random nonce for the Fujisaki-Okamoto transform
	var sigma [encKeySize]byte
	if _, err := rand.Read(sigma[:]); err != nil {
		return err
	}

	var (
		s      = computeKemRandomness(&sigma, m) // H_1(sigma, m)
		symKey = computeDemKey(&sigma)           // H_2(sigma)

		tmpG1 = new(bn256.G1)
		// Ciphertext C = (kem, dem)
		kem = C[0:kemSize]
		dem = C[kemSize:]
	)

	// kem = (A, B, C1). Invoke encasulateKeyStart to compute (A, B)
	if err := e.encapsulateKeyStart(&sigma, s, kem[0:encKeySize+marshaledG1Size]); err != nil {
		return err
	}
	C1 := kem[encKeySize+marshaledG1Size:]

	// C1 = (g1^H(id) h)^s
	tmpG1.ScalarMult(e.g1, val2bignum(idPrefix, []byte(id)))
	tmpG1.Add(tmpG1, e.h)
	tmpG1.ScalarMult(tmpG1, s)
	if err := marshalG1(C1, tmpG1); err != nil {
		return err
	}

	// Nonce for symmetric ecnryption can be all-zeroes string, because
	// we only require one-time semantic security of the underlying symmetric
	// scheme in the Fujisaki-Okamoto transformation.
	var nonce [nonceSize]byte
	if tmp := secretbox.Seal(dem[0:0], m, &nonce, symKey); &tmp[0] != &dem[0] {
		return fmt.Errorf("output of Seal has unexpected length: expected %d, received %d", len(dem), len(tmp))
	}

	return nil
}

func (e *bb1params) CiphertextOverhead() int {
	return kemSize + secretbox.Overhead
}

type bb1PrivateKey struct {
	params *bb1params // public parameters
	d0, d1 *bn256.G2
}

func (k *bb1PrivateKey) Decrypt(C, m []byte) error {
	if err := checkSizes(m, C, k.params); err != nil {
		return err
	}
	var (
		A  = C[0:encKeySize]
		B  = new(bn256.G1)
		C1 = new(bn256.G1)
		D  = C[kemSize:]
	)
	if _, ok := B.Unmarshal(C[encKeySize : encKeySize+marshaledG1Size]); !ok {
		return errBadCiphertext
	}
	if _, ok := C1.Unmarshal(C[encKeySize+marshaledG1Size : encKeySize+2*marshaledG1Size]); !ok {
		return errBadCiphertext
	}
	// sigma = A ⊕ H(e(B, d0)/e(C1,d1))
	var (
		numerator   = bn256.Pair(B, k.d0)
		denominator = bn256.Pair(C1, k.d1)
		hash        = hashval(ibePrefix, numerator.Add(numerator, denominator.Neg(denominator)).Marshal())
	)
	var sigma [encKeySize]byte
	for i := range sigma {
		sigma[i] = A[i] ^ hash[i]
	}

	symKey := computeDemKey(&sigma)

	var nonce [nonceSize]byte
	if tmp, success := secretbox.Open(m[0:0], D, &nonce, symKey); !success || (&tmp[0] != &m[0]) {
		return errBadCiphertext
	}

	// Check that consistent randomness was used to encrypt the symmetric key. It suffices
	// to check that the first two components of the KEM portion matches, since given the
	// message and the secret identity key, the first two components uniquely determine the third.

	// First, derive the randomness used for the KEM.
	s := computeKemRandomness(&sigma, m)

	// Check the first two components
	var kemChkBuf [encKeySize + marshaledG1Size]byte
	k.params.encapsulateKeyStart(&sigma, s, kemChkBuf[:])

	if !bytes.Equal(kemChkBuf[:], C[0:encKeySize+marshaledG1Size]) {
		return errBadCiphertext
	}
	return nil
}

func (k *bb1PrivateKey) Params() Params { return k.params }

// random returns a positive integer in the range [1, bn256.Order)
// (denoted by Zp in http://crypto.stanford.edu/~dabo/papers/bbibe.pdf).
//
// The paper refers to random numbers drawn from Zp*. From a theoretical
// perspective, the uniform distribution over Zp and Zp* start within a
// statistical distance of 1/p (where p=bn256.Order is a ~256bit prime).  Thus,
// drawing uniformly from Zp is no different from Zp*.
func random() (*big.Int, error) {
	for {
		k, err := rand.Int(rand.Reader, bn256.Order)
		if err != nil {
			return nil, err
		}
		if k.Sign() > 0 {
			return k, nil
		}
	}
}

// Hash a particular message with the given prefix. Specifically, this computes
// SHA256(prefix || data) where prefix is a fixed-length string.
func hashval(prefix [1]byte, data []byte) *[sha256.Size]byte {
	hasher := sha256.New()
	hasher.Write(prefix[:])
	hasher.Write(data)

	var ret [sha256.Size]byte
	copy(ret[:], hasher.Sum(nil))

	return &ret
}

// Hashes a value SHA256(prefix || data) where prefix is a fixed-length
// string.  The hashed value is then converted  to a value modulo the group order.
func val2bignum(prefix [1]byte, data []byte) *big.Int {
	k := new(big.Int).SetBytes(hashval(prefix, data)[:])
	return k.Mod(k, bn256.Order)
}

// marshalG1 writes the marshaled form of g into dst.
func marshalG1(dst []byte, g *bn256.G1) error {
	src := g.Marshal()
	if len(src) != len(dst) {
		return fmt.Errorf("bn256.G1.Marshal returned a %d byte slice, expected %d: the BB1 IBE implementation is likely broken", len(src), len(dst))
	}
	copy(dst, src)
	return nil
}
