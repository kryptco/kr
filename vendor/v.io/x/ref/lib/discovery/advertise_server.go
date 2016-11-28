// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"v.io/v23"
	"v.io/v23/context"
	"v.io/v23/discovery"
	"v.io/v23/naming"
	"v.io/v23/rpc"
	"v.io/v23/security"
)

// AdvertiseServer advertises the server with the given advertisement. It uses
// the server's endpoints and the given suffix as the advertisement addresses,
// and the addresses will be updated automatically when the underlying network
// are changed.
//
// The advertisement should not be changed while it is being advertised.
//
// Advertising will continue until the context is canceled or exceeds its deadline
// and the returned channel will be closed when it stops.
//
// If a discovery instance is not provided, this will create a new one.
func AdvertiseServer(ctx *context.T, d discovery.T, server rpc.Server, suffix string, ad *discovery.Advertisement, visibility []security.BlessingPattern) (<-chan struct{}, error) {
	if d == nil {
		var err error
		d, err = v23.NewDiscovery(ctx)
		if err != nil {
			return nil, err
		}
	}

	status := server.Status()
	curAddrs := sortedNames(status.Endpoints)
	stop, err := advertiseServer(ctx, d, ad, curAddrs, suffix, visibility)
	if err != nil {
		return nil, err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		for {
			select {
			case <-status.Dirty:
				status = server.Status()
				if status.State != rpc.ServerActive {
					stop()
					return
				}
				newAddrs := sortedNames(status.Endpoints)
				if sortedStringsEqual(curAddrs, newAddrs) {
					continue
				}

				stop() // Stop the previous advertisement.
				stop, err = advertiseServer(ctx, d, ad, newAddrs, suffix, visibility)
				if err != nil {
					ctx.Error(err)
					return
				}
				curAddrs = newAddrs
			case <-ctx.Done():
				stop()
				return
			}
		}
	}()
	return done, nil
}

func advertiseServer(ctx *context.T, d discovery.T, ad *discovery.Advertisement, addrs []string, suffix string, visibility []security.BlessingPattern) (func(), error) {
	ad.Addresses = make([]string, len(addrs))
	for i, addr := range addrs {
		ad.Addresses[i] = naming.JoinAddressName(addr, suffix)
	}
	ctx, cancel := context.WithCancel(ctx)
	done, err := d.Advertise(ctx, ad, visibility)
	if err != nil {
		cancel()
		return nil, err
	}
	stop := func() {
		cancel()
		<-done
	}
	return stop, nil
}
