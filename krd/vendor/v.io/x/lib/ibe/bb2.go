// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file defines an implementation of the IBE interfaces using the
// Boneh-Boyen scheme (BB2). The Fujisaki-Okamoto transformation is applied to
// obtain CCA2-security. The paper defining this algorithm (see comments for
// SetupBB2) uses multiplicative groups while the bn256 package used in the
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
	"fmt"
	"math/big"

	"golang.org/x/crypto/bn256"
	"golang.org/x/crypto/nacl/secretbox"
)

// Setup creates an ibe.Master based on the BB2 scheme described in "Efficient
// Selective Identity-Based Encryption Without Random Oracles" by Dan Boneh and
// Xavier Boyen (http://crypto.stanford.edu/~dabo/papers/bbibe.pdf).
//
// Specifically, a variant of Section 5.1 (from the section labeled Hash-BDHI
// construction in Section 5.1) of the paper is implemented.
//
// In addition, we apply the Fujisaki-Okamoto transformation to the BB-IBE
// scheme (http://link.springer.com/chapter/10.1007%2F3-540-48405-1_34)
// in order to obtain CCA2-security (in the random oracle model). The resulting
// scheme is a CCA2-secure hybrid encryption scheme where BB-IBE is used to
// encrypt a nonce used to derive a symmetric key, and NaCl/secretbox is used
// to encrypt the data under the symmetric key.
//
// The BB2 scheme uses approximately 30% less CPU for the Decrypt operation
// compared to the BB1 scheme, and public keys are more compact (3 group
// elements vs. 5 group elements). The main disadvantage is that it relies on
// a more complicated assumption on bilinear groups (the hash bilinear Diffie-
// Hellman inversion assumption). This assumption holds in the generic group
// model, and to date, there are no known attacks on the assumption other than
// the generic ones.
func SetupBB2() (Master, error) {
	var (
		m = &bb2master{
			params: newbb2params(),
			hHat:   new(bn256.G2),
		}
		g    = new(bn256.G1).ScalarBaseMult(big.NewInt(1)) // generator for G1
		pk   = m.params                                    // shorthand
		hHat = m.hHat                                      // shorthand
	)

	// Pick a random exponent r and set hHat = gHat^r
	r, err := random()
	if err != nil {
		return nil, err
	}
	hHat.ScalarBaseMult(r)

	// Pick exponent x for master secret key and compute g^x
	m.x, err = random()
	if err != nil {
		return nil, err
	}
	pk.X.ScalarBaseMult(m.x)

	// Pick exponent y for master secret key and compute g^y
	m.y, err = random()
	if err != nil {
		return nil, err
	}
	pk.Y.ScalarBaseMult(m.y)

	pk.v = bn256.Pair(g, hHat)
	return m, nil
}

type bb2master struct {
	params *bb2params // Public params
	x      *big.Int   // Master key component
	y      *big.Int   // Master key component
	hHat   *bn256.G2  // Master key component
}

func (m *bb2master) Extract(id string) (PrivateKey, error) {
	var (
		ret = &bb2PrivateKey{
			params: m.params,
			r:      nil,
			K:      new(bn256.G2),
		}
		// some shorthands
		hHat   = m.hHat
		z      = new(big.Int)
		sum    = new(big.Int)
		invSum = new(big.Int)
		zero   = big.NewInt(0)
	)

	// z = x + H(id) (mod p)
	z.Add(m.x, val2bignum(idPrefix, []byte(id)))
	z.Mod(z, bn256.Order)

	for {
		r, err := random()
		if err != nil {
			return nil, err
		}
		// sum = x + r y + H(id) = z + r y (mod p)
		sum.Mul(r, m.y)
		sum.Add(z, sum)
		sum.Mod(sum, bn256.Order)
		if sum.Cmp(zero) != 0 {
			ret.r = r
			break
		}
	}

	invSum.ModInverse(sum, bn256.Order)
	ret.K.ScalarMult(hHat, invSum)
	return ret, nil
}

func (m *bb2master) Params() Params { return m.params }

type bb2params struct {
	X, Y *bn256.G1
	v    *bn256.GT
}

func newbb2params() *bb2params {
	return &bb2params{
		X: new(bn256.G1),
		Y: new(bn256.G1),
		v: new(bn256.GT),
	}
}

func (e *bb2params) encapsulateKeyStart(sigma *[encKeySize]byte, s *big.Int, C []byte) error {
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
	// B = Y^s
	if err := marshalG1(B, tmpG1.ScalarMult(e.Y, s)); err != nil {
		return err
	}

	return nil
}

func (e *bb2params) Encrypt(id string, m, C []byte) error {
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

	// C1 = (X g^H(id))^s
	tmpG1.ScalarBaseMult(val2bignum(idPrefix, []byte(id)))
	tmpG1.Add(tmpG1, e.X)
	tmpG1.ScalarMult(tmpG1, s)
	if err := marshalG1(C1, tmpG1); err != nil {
		return err
	}

	// Nonce for symmetric encryption can be all-zeroes string, because
	// we only require one-time semantic security of the underlying symmetric
	// scheme in the Fujisaki-Okamoto transformation.
	var nonce [nonceSize]byte
	if tmp := secretbox.Seal(dem[0:0], m, &nonce, symKey); &tmp[0] != &dem[0] {
		return fmt.Errorf("output of Seal has unexpected length: expected %d, received %d", len(dem), len(tmp))
	}

	return nil
}

func (e *bb2params) CiphertextOverhead() int {
	return kemSize + secretbox.Overhead
}

type bb2PrivateKey struct {
	params *bb2params // public params
	r      *big.Int
	K      *bn256.G2
}

func (k *bb2PrivateKey) Decrypt(C, m []byte) error {
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

	// sigma = A ⊕ H(e(B^r C, K)
	var tmpG1 = new(bn256.G1)
	tmpG1.ScalarMult(B, k.r)
	tmpG1.Add(tmpG1, C1)
	var hash = hashval(ibePrefix, bn256.Pair(tmpG1, k.K).Marshal())

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

func (k *bb2PrivateKey) Params() Params { return k.params }
