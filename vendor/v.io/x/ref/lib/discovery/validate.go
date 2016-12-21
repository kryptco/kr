// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"v.io/v23/discovery"
)

const (
	// TODO(jhahn): Temporarily increase the max size.
	// Decrease to 512 when the address compression is ready.
	maxAdvertisementSize = 1024
	maxAttachmentSize    = 4096
	maxNumAttributes     = 32
	maxNumAttachments    = 32
)

// validateKey returns an error if the key is not suitable for advertising.
func validateKey(key string) error {
	if len(key) == 0 {
		return errors.New("empty key")
	}
	if strings.HasPrefix(key, "_") {
		return fmt.Errorf("key starts with '_': %q", key)
	}
	for _, c := range key {
		if c < 0x20 || c > 0x7e {
			return fmt.Errorf("key is not printable US-ASCII: %q", key)
		}
		if c == '=' {
			return fmt.Errorf("key includes '=': %q", key)
		}
	}
	return nil
}

// validateAttributes returns an error if the attributes are not suitable for advertising.
func validateAttributes(attributes discovery.Attributes) error {
	if len(attributes) > maxNumAttributes {
		return fmt.Errorf("too many: %d > %d", len(attributes), maxNumAttributes)
	}
	for k, v := range attributes {
		if err := validateKey(k); err != nil {
			return err
		}
		if !utf8.ValidString(v) {
			return fmt.Errorf("value is not valid UTF-8 string: %q", v)
		}
	}
	return nil
}

// validateAttachments returns an error if the attachments are not suitable for advertising.
func validateAttachments(attachments discovery.Attachments) error {
	if len(attachments) > maxNumAttachments {
		return fmt.Errorf("too many: %d > %d", len(attachments), maxNumAttachments)
	}
	for k, v := range attachments {
		if err := validateKey(k); err != nil {
			return err
		}
		if len(v) > maxAttachmentSize {
			return fmt.Errorf("too large value for %q: %d > %d", k, len(v), maxAttachmentSize)
		}
	}
	return nil
}

// sizeOfAd returns the size of ad excluding id and attachments.
func sizeOfAd(ad *discovery.Advertisement) int {
	size := len(ad.InterfaceName) + len(PackAddresses(ad.Addresses))
	for k, v := range ad.Attributes {
		size += len(k) + len(v)
	}
	return size
}

// validateAd returns an error if ad is not suitable for advertising.
func validateAd(ad *discovery.Advertisement) error {
	if !ad.Id.IsValid() {
		return errors.New("id not valid")
	}
	if len(ad.InterfaceName) == 0 {
		return errors.New("interface name not provided")
	}
	if len(ad.Addresses) == 0 {
		return errors.New("address not provided")
	}
	if err := validateAttributes(ad.Attributes); err != nil {
		return fmt.Errorf("attributes not valid: %v", err)
	}
	if err := validateAttachments(ad.Attachments); err != nil {
		return fmt.Errorf("attachments not valid: %v", err)
	}
	if sizeOfAd(ad) > maxAdvertisementSize {
		return fmt.Errorf("advertisement too large: %d > %d", sizeOfAd(ad), maxAdvertisementSize)
	}
	return nil
}
