// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"sort"

	"v.io/v23/naming"
)

// hashAd hashes the advertisement.
func hashAd(adinfo *AdInfo) {
	w := func(w io.Writer, data []byte) {
		sum := sha256.Sum256(data)
		w.Write(sum[:])
	}

	hasher := sha256.New()

	w(hasher, adinfo.Ad.Id[:])
	w(hasher, []byte(adinfo.Ad.InterfaceName))

	field := sha256.New()
	for _, addr := range adinfo.Ad.Addresses {
		w(field, []byte(addr))
	}
	hasher.Write(field.Sum(nil))

	field.Reset()
	if n := len(adinfo.Ad.Attributes); n > 0 {
		keys := make([]string, 0, n)
		for k, _ := range adinfo.Ad.Attributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			w(field, []byte(k))
			w(field, []byte(adinfo.Ad.Attributes[k]))
		}
	}
	hasher.Write(field.Sum(nil))

	field.Reset()
	if n := len(adinfo.Ad.Attachments); n > 0 {
		keys := make([]string, 0, n)
		for k, _ := range adinfo.Ad.Attachments {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			w(field, []byte(k))
			w(field, []byte(adinfo.Ad.Attachments[k]))
		}
	}
	hasher.Write(field.Sum(nil))

	field.Reset()
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, adinfo.EncryptionAlgorithm)
	w(field, buf.Bytes())
	for _, key := range adinfo.EncryptionKeys {
		w(field, []byte(key))
	}
	hasher.Write(field.Sum(nil))

	// We use the first 8 bytes to reduce the advertise packet size.
	copy(adinfo.Hash[:], hasher.Sum(nil))
}

func sortedNames(eps []naming.Endpoint) []string {
	names := make([]string, len(eps))
	for i, ep := range eps {
		names[i] = ep.Name()
	}
	sort.Strings(names)
	return names
}

func sortedStringsEqual(a, b []string) bool {
	// We want to make a nil and an empty slices equal to avoid unnecessary inequality by that.
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
