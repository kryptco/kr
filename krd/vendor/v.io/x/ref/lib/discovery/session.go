// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"errors"
	"sync"

	"v.io/v23/context"
	"v.io/v23/discovery"
	"v.io/v23/security"
)

// sdiscovery is an implementation of discovery.T.
type sdiscovery struct {
	d       *idiscovery
	session sessionId
}

func (sd *sdiscovery) Advertise(ctx *context.T, ad *discovery.Advertisement, visibility []security.BlessingPattern) (<-chan struct{}, error) {
	return sd.d.advertise(ctx, sd.session, ad, visibility)
}

func (sd *sdiscovery) Scan(ctx *context.T, query string) (<-chan discovery.Update, error) {
	return sd.d.scan(ctx, sd.session, query)
}

// sdFactory is an implementation of Factory.
type sdFactory struct {
	d *idiscovery

	mu          sync.Mutex
	lastSession sessionId // GUARDED_BY(mu)
}

func (f *sdFactory) New(*context.T) (discovery.T, error) {
	session, err := f.newSession()
	if err != nil {
		return nil, err
	}
	return &sdiscovery{d: f.d, session: session}, nil
}

func (f *sdFactory) Shutdown() {
	f.d.shutdown()
}

func (f *sdFactory) newSession() (sessionId, error) {
	f.mu.Lock()
	session := f.lastSession + 1
	if session == 0 {
		f.mu.Unlock()
		return 0, errors.New("session overflow")
	}
	f.lastSession = session
	f.mu.Unlock()
	return session, nil
}

func newSessionedDiscoveryFactory(d *idiscovery) (Factory, error) {
	return &sdFactory{d: d}, nil
}
