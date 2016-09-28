// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"v.io/v23/context"
	"v.io/v23/discovery"
)

// Factory is the interface for creating a new discovery.T instance.
type Factory interface {
	// New creates a new Discovery.T instance.
	New(*context.T) (discovery.T, error)

	// Shutdown closes all Discovery.T instances and shutdowns the factory.
	Shutdown()
}

// NewFactory returns a new discovery factory with the given plugins.
//
// For internal use only.
func NewFactory(ctx *context.T, plugins ...Plugin) (Factory, error) {
	d, err := newDiscovery(ctx, plugins)
	if err != nil {
		return nil, err
	}
	return newSessionedDiscoveryFactory(d)
}
