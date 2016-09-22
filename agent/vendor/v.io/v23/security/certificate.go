// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package security

import (
	"bytes"
	"crypto/sha256"
	"strings"
	"sync"
	"unicode"

	"v.io/v23/verror"
)

var (
	errCantMarshalKey             = verror.Register(pkgPath+".errCantMarshalKey", verror.NoRetry, "{1:}{2:}failed to marshal PublicKey into a Certificate{:_}")
	errInapproriateCertSignature  = verror.Register(pkgPath+".errInapproriateCertSignature", verror.NoRetry, "{1:}{2:}signature on certificate(for {3}) was not intended for certification (purpose={4}){:_}")
	errBadBlessingExtensionInCert = verror.Register(pkgPath+".errBadBlessingExtensionInCert", verror.NoRetry, "{1:}{2:}invalid blessing extension in certificate(for {3}){:_}")
	errBadCertSignature           = verror.Register(pkgPath+".errBadCertSignature", verror.NoRetry, "{1:}{2:}invalid Signature in certificate(for \"{3}\"), signing key{:_}")
	errBadBlessingEmptyExtension  = verror.Register(pkgPath+".errBadBlessingEmptyExtension", verror.NoRetry, "{1:}{2:}invalid blessing extension(empty string){:_}")
	errBadBlessingBadStart        = verror.Register(pkgPath+".errBadBlessingBadStart", verror.NoRetry, "{1:}{2:}invalid blessing extension(starts with {3}){:_}")
	errBadBlessingBadEnd          = verror.Register(pkgPath+".errBadBlessingBadEnd", verror.NoRetry, "{1:}{2:}invalid blessing extension(ends with {3}){:_}")
	errBadBlessingExtension       = verror.Register(pkgPath+".errBadBlessingExtension", verror.NoRetry, "{1:}{2:}invalid blessing extension({3}){:_}")
	errBadBlessingControlChar     = verror.Register(pkgPath+".errBadBlessingControlChar", verror.NoRetry, "{1:}{2:}invalid blessing extension({3} contains control character as a substring){:_}")
	errBadBlessingBadSubstring    = verror.Register(pkgPath+".errBadBlessingBadSubstring", verror.NoRetry, "{1:}{2:}invalid blessing extension({3} has {4} as a substring){:_}")

	// invalidBlessingSubStrings are strings that a blessing extension cannot have as a substring.
	invalidBlessingSubStrings = []string{string(AllPrincipals), ChainSeparator + ChainSeparator /* double chain separator not allowed */, ",", "@@", "(", ")", "<", ">", "/"}
	// invalidBlessingExtensions are strings that are disallowed as blessing extensions.
	invalidBlessingExtensions = []string{string(NoExtension)}

	// Cache of previously verified certificate chains (identified by the digest of the certificate chain).
	// Used to reduce computation overhead of certificate verification when the same set of certificates are
	// seen repeatedly.
	//
	// Caching scheme described in
	// https://docs.google.com/document/d/1jGbhwKw2SRFUIV_C55GdAwd_UzZtRoSEnnskt0GzNw4/edit?usp=sharing
	signatureCache = &sigCache{m: make(map[[sha256.Size]byte]bool)}
)

const signatureCacheMaxSize = 1 << 10 // 32 bytes * 1K = 32KB + map overhead in Go

func newUnsignedCertificate(extension string, key PublicKey, caveats ...Caveat) (*Certificate, error) {
	err := validateExtension(extension)
	if err != nil {
		return nil, err
	}
	cert := &Certificate{Extension: extension, Caveats: caveats}
	if cert.PublicKey, err = key.MarshalBinary(); err != nil {
		return nil, verror.New(errCantMarshalKey, nil, err)
	}
	return cert, nil
}

func (c *Certificate) contentDigest(hashfn Hash) []byte {
	var fields []byte
	w := func(data []byte) {
		fields = append(fields, hashfn.sum(data)...)
	}
	w(c.PublicKey)
	w([]byte(c.Extension))
	for _, cav := range c.Caveats {
		fields = append(fields, cav.digest(hashfn)...)
	}
	return hashfn.sum(fields)
}

// chainedDigests returns the digest and contentDigest of a certificate chain
// formed by chaining c to an another certificate chain (identified by its
// digest).
//
// If len(chain) == 0, the implication is that 'c' is the first certificate
// (a.k.a. "root") of the chain.
func (c *Certificate) chainedDigests(hashfn Hash, chain []byte) (digest, contentDigest []byte) {
	contentDigest = c.contentDigest(hashfn)
	digest = hashfn.sum(append(contentDigest, c.Signature.digest(hashfn)...))
	if len(chain) > 0 {
		// c is not the "root" of the chain
		// We hash (using 'hashfn') 'chain' and then append it to 'digest'
		// (or 'contentDigest'). Hashing 'chain' is important as it may be
		// of a different length from 'digest'. While the length of 'digest'
		// is the length of the output of 'hashfn', the length of 'chain'
		// would depend on the size of the public key in the last
		// certificate of the chain represented by it. Hashing 'chain'
		// using 'hasfn' guarantees that it will have the same length
		// as 'digest'.
		contentDigest = hashfn.sum(append(hashfn.sum(chain), contentDigest...))
		digest = hashfn.sum(append(hashfn.sum(chain), digest...))
	}
	return
}

func validateExtension(extension string) error {
	if len(extension) == 0 {
		return verror.New(errBadBlessingEmptyExtension, nil)
	}
	if strings.HasPrefix(extension, ChainSeparator) {
		return verror.New(errBadBlessingBadStart, nil, ChainSeparator)
	}
	if strings.HasSuffix(extension, ChainSeparator) {
		return verror.New(errBadBlessingBadEnd, nil, ChainSeparator)
	}
	for _, n := range invalidBlessingExtensions {
		if extension == n {
			return verror.New(errBadBlessingExtension, nil, extension)
		}
	}
	if strings.IndexFunc(extension, unicode.IsControl) != -1 {
		return verror.New(errBadBlessingControlChar, nil, extension)
	}
	for _, n := range invalidBlessingSubStrings {
		if strings.Contains(extension, n) {
			return verror.New(errBadBlessingBadSubstring, nil, extension, n)
		}
	}
	return nil
}

// Validation algorithm as specified in:
// https://docs.google.com/document/d/1jGbhwKw2SRFUIV_C55GdAwd_UzZtRoSEnnskt0GzNw4/edit?usp=sharing
func validateCertificateChain(chain []Certificate) (PublicKey, []byte, error) {
	pubkey, err := UnmarshalPublicKey(chain[len(chain)-1].PublicKey)
	if err != nil {
		return nil, nil, err
	}
	var (
		digest        = make([][]byte, len(chain))
		contentDigest = make([][]byte, len(chain))
	)
	digest[0], contentDigest[0] = chain[0].chainedDigests(chain[0].Signature.Hash, nil)
	for i := 1; i < len(chain); i++ {
		digest[i], contentDigest[i] = chain[i].chainedDigests(chain[i].Signature.Hash, digest[i-1])
	}
	chaindigest := digest[len(digest)-1]
	// Verify certificates in reverse order as per the algorithm linked to above.
	for i := len(chain) - 1; i >= 0; i-- {
		c := chain[i]
		// Check the in-memory cache
		if signatureCache.verify(digest[i]) {
			// chain[0:i] has been validated before
			// and chain[i:] has been validated in this for loop.
			signatureCache.cache(digest[i:])
			return pubkey, chaindigest, nil
		}
		// Some basic sanity checks on the certificate.
		if !bytes.Equal(c.Signature.Purpose, blessPurpose) {
			return nil, nil, verror.New(errInapproriateCertSignature, nil, c.Extension, c.Signature.Purpose)
		}
		if err := validateExtension(c.Extension); err != nil {
			return nil, nil, verror.New(errBadBlessingExtensionInCert, nil, c.Extension, err)
		}
		// Verify the signature.
		var signer PublicKey
		if i == 0 {
			signer, err = UnmarshalPublicKey(chain[0].PublicKey)
		} else {
			signer, err = UnmarshalPublicKey(chain[i-1].PublicKey)
		}
		if err != nil {
			return nil, nil, err
		}
		if !chain[i].Signature.Verify(signer, contentDigest[i]) {
			return nil, nil, verror.New(errBadCertSignature, nil, chain[i].Extension, signer)
		}
	}
	signatureCache.cache(digest)
	return pubkey, chaindigest, nil
}

func digestsForCertificateChain(chain []Certificate) (digest, contentDigest []byte) {
	for _, c := range chain {
		digest, contentDigest = c.chainedDigests(c.Signature.Hash, digest)
	}
	return
}

// chainCertificate binds cert to an existing certificate chain and returns the
// resulting chain (and the final digest).
func chainCertificate(signer Signer, chain []Certificate, cert Certificate) ([]Certificate, []byte, error) {
	parentDigest, _ := digestsForCertificateChain(chain)
	_, cdigest := cert.chainedDigests(signer.PublicKey().hash(), parentDigest)
	var err error
	if cert.Signature, err = signer.Sign(blessPurpose, cdigest); err != nil {
		return nil, nil, err
	}
	// digest has to be recomputed now that the signature has been set in the certificate.
	digest, _ := cert.chainedDigests(signer.PublicKey().hash(), parentDigest)
	cpy := make([]Certificate, len(chain)+1)
	copy(cpy, chain)
	cpy[len(cpy)-1] = cert
	return cpy, digest, nil
}

// Concurrent access friendly map of previously verified certificate chains
// (identified by their digest).
//
// TODO(ashankar,ataly): sigCache.verify may leak information based on the
// time it takes to execute because sigCache.m does not provide "constant time"
// lookups (in the sense of crypto/subtle.ConstantTimeEq for example).
// Technically, constant time lookups are required as per:
// https://docs.google.com/document/d/1jGbhwKw2SRFUIV_C55GdAwd_UzZtRoSEnnskt0GzNw4/edit?usp=sharing
type sigCache struct {
	disabled bool // Only here for microbenchmarks
	sync.RWMutex
	m map[[sha256.Size]byte]bool
}

func (s *sigCache) disable() {
	s.Lock()
	s.disabled = true
	s.m = make(map[[sha256.Size]byte]bool)
	s.Unlock()
}

func (s *sigCache) enable() {
	s.Lock()
	s.disabled = false
	s.Unlock()
}

func (s *sigCache) verify(digest []byte) bool {
	key := sha256.Sum256(digest)
	s.RLock()
	ret := s.m[key]
	s.RUnlock()
	return ret
}

func (s *sigCache) cache(digests [][]byte) {
	keys := make([][sha256.Size]byte, len(digests))
	for i, d := range digests {
		keys[i] = sha256.Sum256(d)
	}
	s.Lock()
	if s.disabled {
		s.Unlock()
		return
	}
	for _, k := range keys {
		s.m[k] = true
	}
	// Might have gone over our size limit for the cache, remove entries.
	// This may evict the entries that were just inserted, live with that.
	if len(s.m) > signatureCacheMaxSize {
		n := len(s.m) - signatureCacheMaxSize
		m := 0
		// Randomly evict and entry. Fortunately, map iteration is in random key order
		// (see "Iteration Order" in http://blog.golang.org/go-maps-in-action)
		for key, _ := range s.m {
			delete(s.m, key)
			m++
			if m >= n {
				break
			}
		}
	}
	s.Unlock()
}
