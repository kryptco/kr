// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ble

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"time"

	"v.io/v23/context"
	"v.io/v23/discovery"
	"v.io/v23/naming"
	idiscovery "v.io/x/ref/lib/discovery"
	"v.io/x/ref/lib/stats"
)

const (
	// TTL for scanned advertisement. If we do not see the advertisement again
	// during that period, we send a "Lost" notification.
	defaultTTL = 90 * time.Second
)

var (
	statMu  sync.Mutex
	statIdx int
	seenIdx int
)

type plugin struct {
	advertiser *advertiser
	scanner    *scanner
	adStopper  *idiscovery.Trigger
	statPrefix string
}

func (p *plugin) Advertise(ctx *context.T, adinfo *idiscovery.AdInfo, done func()) (err error) {
	if err := p.advertiser.addAd(adinfo); err != nil {
		done()
		return err
	}
	stop := func() {
		p.advertiser.removeAd(adinfo)
		done()
	}
	p.adStopper.Add(stop, ctx.Done())
	return nil
}

func (p *plugin) Scan(ctx *context.T, interfaceName string, callback func(*idiscovery.AdInfo), done func()) error {
	go func() {
		defer done()

		listener := p.scanner.addListener(interfaceName)
		defer p.scanner.removeListener(interfaceName, listener)

		seen := make(map[discovery.AdId]*idiscovery.AdInfo)

		// TODO(ashankar,jhahn): To prevent plugins from stepping over
		// each other (e.g., a Lost even from one undoing a Found event
		// from another), the discovery implementation that uses
		// plugins should be made aware of the plugin that sent the event.
		// In that case, perhaps these stats should also be exported there,
		// rather than in each plugin implementation?
		statMu.Lock()
		stat := naming.Join(p.statPrefix, "seen", fmt.Sprint(seenIdx))
		seenIdx++
		statMu.Unlock()
		var seenMu sync.Mutex // Safety between this goroutine and stats
		stats.NewStringFunc(stat, func() string {
			seenMu.Lock()
			defer seenMu.Unlock()
			buf := new(bytes.Buffer)
			fmt.Fprintf(buf, "InterfaceName: %q\n", interfaceName)
			for k, v := range seen {
				fmt.Fprintf(buf, "%s: %v\n\n", k, *v)
			}
			return buf.String()
		})
		defer stats.Delete(stat)
		for {
			select {
			case adinfo := <-listener:
				if adinfo.Lost {
					seenMu.Lock()
					delete(seen, adinfo.Ad.Id)
					seenMu.Unlock()
				} else if prev := seen[adinfo.Ad.Id]; prev != nil && (prev.Hash == adinfo.Hash || prev.TimestampNs >= adinfo.TimestampNs) {
					continue
				} else {
					seenMu.Lock()
					seen[adinfo.Ad.Id] = adinfo
					seenMu.Unlock()
				}
				copied := *adinfo
				callback(&copied)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (p *plugin) Close() {
	p.scanner.shutdown()
}

// New returns a new BLE plugin instance with default ttl (90s).
//
// TODO(jhahn): Rename to New() once we remove old codes.
func New(ctx *context.T, host string) (idiscovery.Plugin, error) {
	return newWithTTL(ctx, host, defaultTTL)
}

func newWithTTL(ctx *context.T, host string, ttl time.Duration) (idiscovery.Plugin, error) {
	driver, err := driverFactory(ctx, host)
	if err != nil {
		return nil, err
	}
	statMu.Lock()
	statPrefix := naming.Join("discovery", "ble", fmt.Sprint(statIdx))
	statIdx++
	stats.NewStringFunc(naming.Join(statPrefix, "driver"), func() string {
		return driver.DebugString()
	})
	statMu.Unlock()
	p := &plugin{
		advertiser: newAdvertiser(ctx, driver),
		scanner:    newScanner(ctx, driver, ttl),
		adStopper:  idiscovery.NewTrigger(),
		statPrefix: statPrefix,
	}
	runtime.SetFinalizer(p, func(p *plugin) { stats.Delete(statPrefix) })
	return p, nil
}
