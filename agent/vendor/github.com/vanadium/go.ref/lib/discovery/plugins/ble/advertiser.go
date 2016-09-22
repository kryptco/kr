// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ble

import (
	"fmt"
	"sync"
	"time"

	"v.io/v23/context"
	"v.io/v23/discovery"

	idiscovery "v.io/x/ref/lib/discovery"
)

// TODO(jhahn): Remove the limit on the number of advertisements per interface.
//
// The current plan is
//    * if there is only one advertisement
//			- publish all information except attachments as is
//		* if there are more than one advertisements
//			- publish the latest directory addresses, # of advertisements
//			- if there are <= 8 advertisements,
//					- publish each advertisements up to 2K (4 characteristics)
//						* if it doesn't fit, publish only id, hash
//			- if there are <= 64 advertisements,
//				  - publish a list of a pair of id and hash (24 bytes / <id, hash>)
//
//		* all missed advertisements / information will be fetched from directory server.

type advertiser struct {
	ctx    *context.T
	driver Driver

	mu        sync.Mutex
	adRecords map[string]*adRecord // GUARDED_BY(mu)
}

type adRecord struct {
	uuid    idiscovery.Uuid
	adinfos map[discovery.AdId][]byte
	expiry  time.Time
}

const (
	uuidGcDelay = 10 * time.Minute
)

func (a *advertiser) addAd(adinfo *idiscovery.AdInfo) error {
	encoded, err := encodeAdInfo(adinfo)
	if err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.gcLocked()

	rec := a.adRecords[adinfo.Ad.InterfaceName]
	if rec == nil {
		rec = &adRecord{
			uuid:    newServiceUuid(adinfo.Ad.InterfaceName),
			adinfos: make(map[discovery.AdId][]byte),
		}
		a.adRecords[adinfo.Ad.InterfaceName] = rec
	} else {
		if len(rec.adinfos) >= maxNumPackedServices {
			return fmt.Errorf("too many advertisements per interface: %d > %d", len(rec.adinfos), maxNumPackedServices)
		}

		// Stop the current advertising and restart with a toggled uuid to avoid a new
		// advertisement from being deduped by cache.
		a.driver.RemoveService(rec.uuid.String())

		toggleServiceUuid(rec.uuid)
		rec.expiry = time.Time{}
	}
	rec.adinfos[adinfo.Ad.Id] = encoded

	cs := packToCharacteristics(rec.adinfos)
	err = a.driver.AddService(rec.uuid.String(), cs)
	if err != nil {
		rec.expiry = time.Now().Add(uuidGcDelay)
	}
	return err
}

func (a *advertiser) removeAd(adinfo *idiscovery.AdInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.gcLocked()

	rec := a.adRecords[adinfo.Ad.InterfaceName]
	if rec == nil {
		return
	}

	a.driver.RemoveService(rec.uuid.String())
	delete(rec.adinfos, adinfo.Ad.Id)

	if len(rec.adinfos) == 0 {
		rec.expiry = time.Now().Add(uuidGcDelay)
	} else {
		// Restart advertising with a toggled uuid to avoid the updated advertisement
		// from being deduped by cache.
		toggleServiceUuid(rec.uuid)

		cs := packToCharacteristics(rec.adinfos)
		if err := a.driver.AddService(rec.uuid.String(), cs); err != nil {
			a.ctx.Error(err)
			rec.expiry = time.Now().Add(uuidGcDelay)
		}
	}
}

func (a *advertiser) gcLocked() {
	// Instead of asynchronous gc, we purge old entries in every call to addAd or removeAd
	// for simplicity. We do not worry about purging all old entries since there will be
	// only a handful of ads in practice.
	now := time.Now()
	for interfaceName, rec := range a.adRecords {
		if rec.expiry.IsZero() || rec.expiry.After(now) {
			continue
		}
		delete(a.adRecords, interfaceName)
	}
}

func newAdvertiser(ctx *context.T, driver Driver) *advertiser {
	return &advertiser{
		ctx:       ctx,
		driver:    driver,
		adRecords: make(map[string]*adRecord),
	}
}
