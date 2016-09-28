// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"time"

	"v.io/v23/context"
)

// Update is the interface for a discovery update.
type Update interface {
	// IsLost returns true when this update corresponds to an advertisement
	// that led to a previous update vanishing.
	IsLost() bool

	// Id returns the universal unique identifier of the advertisement.
	Id() AdId

	// InterfaceName returns the interface name that the service implements.
	InterfaceName() string

	// Addresses returns the addresses (vanadium object names) that the service
	// is served on.
	Addresses() []string

	// Attribute returns the named attribute. An empty string is returned if
	// not found.
	Attribute(name string) string

	// Attachment returns the channel on which the named attachment can be read.
	// Nil data is returned if not found.
	//
	// This may do RPC calls if the attachment is not fetched yet and fetching
	// will fail if the context is canceled or exceeds its deadline.
	//
	// Attachments may not be available when this update is for lost advertisement.
	Attachment(ctx *context.T, name string) <-chan DataOrError

	// Advertisement returns a copy of the advertisement that this update
	// corresponds to.
	//
	// The returned advertisement may not include all attachments.
	Advertisement() Advertisement

	// Timestamp returns the time when advertising began for the corresponding
	// Advertisement.
	Timestamp() time.Time
}

// DataOrError contains either an attachment data or an error
// encountered fetching the attachment.
type DataOrError struct {
	Data  []byte
	Error error
}
