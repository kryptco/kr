// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"sort"
	"sync"

	"v.io/v23/context"
	"v.io/v23/discovery"
)

type scanChanElem struct {
	src uint // index into idiscovery.plugins
	val *AdInfo
}

func (d *idiscovery) scan(ctx *context.T, session sessionId, query string) (<-chan discovery.Update, error) {
	// TODO(jhahn): Consider to use multiple target services so that the plugins
	// can filter advertisements more efficiently if possible.
	matcher, err := NewMatcher(ctx, query)
	if err != nil {
		return nil, err
	}

	ctx, cancel, err := d.addTask(ctx)
	if err != nil {
		return nil, err
	}

	// TODO(jhahn): Revisit the buffer size.
	scanCh := make(chan scanChanElem, 10)
	updateCh := make(chan discovery.Update, 10)

	barrier := NewBarrier(func() {
		close(scanCh)
		close(updateCh)
		d.removeTask(ctx)
	})
	for idx, plugin := range d.plugins {
		p := uint(idx) // https://golang.org/doc/faq#closures_and_goroutines
		callback := func(ad *AdInfo) {
			select {
			case scanCh <- scanChanElem{p, ad}:
			case <-ctx.Done():
			}
		}
		if err := plugin.Scan(ctx, matcher.TargetInterfaceName(), callback, barrier.Add()); err != nil {
			cancel()
			return nil, err
		}
	}
	go d.doScan(ctx, session, matcher, scanCh, updateCh, barrier.Add())
	return updateCh, nil
}

type adref struct {
	adinfo *AdInfo
	refs   uint32 // Bitmap of plugin indices that saw the ad
}

func (a *adref) set(plugin uint) {
	mask := uint32(1) << plugin
	a.refs = a.refs | mask
}

func (a *adref) unset(plugin uint) bool {
	mask := uint32(1) << plugin
	a.refs = a.refs & (^mask)
	return a.refs == 0
}

func (d *idiscovery) doScan(ctx *context.T, session sessionId, matcher Matcher, scanCh chan scanChanElem, updateCh chan<- discovery.Update, done func()) {
	// Some plugins may not return a full advertisement information when it is lost.
	// So we keep the advertisements that we've seen so that we can provide the
	// full advertisement information when it is lost. Note that plugins will not
	// include attachments unless they're tiny enough.
	seen := make(map[discovery.AdId]*adref)
	send := func(u discovery.Update) bool {
		select {
		case updateCh <- u:
			return true
		case <-ctx.Done():
			return false
		}
	}

	var wg sync.WaitGroup
	defer func() {
		wg.Wait()
		done()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case e := <-scanCh:
			plugin, adinfo := e.src, e.val
			id := adinfo.Ad.Id
			prev := seen[adinfo.Ad.Id]
			if adinfo.Lost {
				// A 'Lost' advertisement may not have complete
				// information.  Send the lost notification on
				// updateCh only if a found event was
				// previously sent, and all plugins that found
				// it have lost it.
				if prev == nil || !prev.unset(plugin) {
					continue
				}
				delete(seen, id)
				prev.adinfo.Lost = true
				if !send(NewUpdate(prev.adinfo)) {
					return
				}
				continue
			}
			if d.getAdSession(id) == session {
				// Ignore advertisements made within the same session.
				continue
			}
			if prev != nil && prev.adinfo.Hash == adinfo.Hash {
				prev.set(plugin)
				if prev.adinfo.Status == AdReady {
					continue
				}
			}
			if adinfo.Status == AdReady {
				// Clear the unnecessary directory addresses.
				adinfo.DirAddrs = nil
			} else if len(adinfo.DirAddrs) == 0 {
				ctx.Errorf("no directory address available for partial advertisement %v - ignored", id)
				continue
			} else if adinfo.Status == AdNotReady {
				// Fetch not-ready-to-serve advertisements from the directory server.
				wg.Add(1)
				go fetchAd(ctx, adinfo.DirAddrs, id, plugin, scanCh, wg.Done)
				continue
			}

			// Sort the directory addresses to make it easy to compare.
			sort.Strings(adinfo.DirAddrs)

			if err := decrypt(ctx, adinfo); err != nil {
				// Couldn't decrypt it. Ignore it.
				if err != errNoPermission {
					ctx.Error(err)
				}
				continue
			}

			if matched, err := matcher.Match(&adinfo.Ad); err != nil {
				ctx.Error(err)
				continue
			} else if !matched {
				continue
			}

			if prev == nil {
				// Never seen before
				ref := &adref{adinfo: adinfo}
				ref.set(plugin)
				seen[id] = ref
				if !send(NewUpdate(adinfo)) {
					return
				}
				continue
			}
			if prev.adinfo.TimestampNs > adinfo.TimestampNs {
				// Ignore old ad.
				continue
			}
			// TODO(jhahn): Compare proximity as well
			if prev.adinfo.Hash != adinfo.Hash || (prev.adinfo.Status != AdReady && !sortedStringsEqual(prev.adinfo.DirAddrs, adinfo.DirAddrs)) {
				// Changed contents of a previously seen ad. Treat it like a newly seen ad.
				ref := &adref{adinfo: adinfo}
				ref.set(plugin)
				seen[id] = ref
				prev.adinfo.Lost = true
				if !send(NewUpdate(prev.adinfo)) || !send(NewUpdate(adinfo)) {
					return
				}
			}
		}
	}
}

func fetchAd(ctx *context.T, dirAddrs []string, id discovery.AdId, plugin uint, scanCh chan<- scanChanElem, done func()) {
	defer done()

	dir := newDirClient(dirAddrs)
	adinfo, err := dir.Lookup(ctx, id)
	if err != nil {
		select {
		case <-ctx.Done():
		default:
			ctx.Error(err)
		}
		return
	}
	select {
	case scanCh <- scanChanElem{plugin, adinfo}:
	case <-ctx.Done():
	}
}
