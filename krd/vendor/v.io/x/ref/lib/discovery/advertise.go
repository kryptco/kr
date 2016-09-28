// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"fmt"
	"sync"
	"time"

	"v.io/v23/context"
	"v.io/v23/discovery"
	"v.io/v23/naming"
	"v.io/v23/security"
	"v.io/x/lib/vlog"
	"v.io/x/ref/lib/stats"
)

const (
	msPerSec = int64(time.Second / time.Millisecond)
	nsPerMs  = int64(time.Millisecond / time.Nanosecond)
)

func (d *idiscovery) advertise(ctx *context.T, session sessionId, ad *discovery.Advertisement, visibility []security.BlessingPattern) (<-chan struct{}, error) {
	if !ad.Id.IsValid() {
		var err error
		if ad.Id, err = discovery.NewAdId(); err != nil {
			return nil, err
		}
	}
	if err := validateAd(ad); err != nil {
		return nil, NewErrBadAdvertisement(ctx, err)
	}

	adinfo := &AdInfo{Ad: *ad}
	if err := encrypt(ctx, adinfo, visibility); err != nil {
		return nil, err
	}
	hashAd(adinfo)
	adinfo.TimestampNs = d.newAdTimestampNs()

	ctx, cancel, err := d.addTask(ctx)
	if err != nil {
		return nil, err
	}

	id := adinfo.Ad.Id
	if !d.addAd(id, session) {
		cancel()
		d.removeTask(ctx)
		return nil, NewErrAlreadyBeingAdvertised(ctx, id)
	}

	subtask := &adSubtask{parent: ctx}
	d.adMu.Lock()
	d.adSubtasks[id] = subtask
	d.adMu.Unlock()

	done := make(chan struct{})
	stop := func() {
		d.stopAdvertising(id)
		d.dirServer.unpublish(id)
		d.removeAd(id)
		d.removeTask(ctx)
		close(done)
	}

	// Lock the subtask to prevent any update from directory server endpoint changes while
	// the advertising is being started to not lose any endpoint change during starting.
	subtask.mu.Lock()
	d.dirServer.publish(adinfo)
	subtask.stop, err = d.startAdvertising(ctx, adinfo)
	subtask.mu.Unlock()
	if err != nil {
		cancel()
		stop()
		return nil, err
	}
	d.adStopTrigger.Add(stop, ctx.Done())
	return done, nil
}

func (d *idiscovery) newAdTimestampNs() int64 {
	now := time.Now()
	timestampNs := now.UnixNano()
	d.adMu.Lock()
	if d.adTimestampNs >= timestampNs {
		timestampNs = d.adTimestampNs + 1
	}
	d.adTimestampNs = timestampNs
	d.adMu.Unlock()
	return timestampNs
}

func (d *idiscovery) addAd(id discovery.AdId, session sessionId) bool {
	d.adMu.Lock()
	if _, exist := d.adSessions[id]; exist {
		d.adMu.Unlock()
		return false
	}
	d.adSessions[id] = session
	d.adMu.Unlock()
	return true
}

func (d *idiscovery) removeAd(id discovery.AdId) {
	d.adMu.Lock()
	delete(d.adSessions, id)
	d.adMu.Unlock()
}

func (d *idiscovery) getAdSession(id discovery.AdId) sessionId {
	d.adMu.Lock()
	session := d.adSessions[id]
	d.adMu.Unlock()
	return session
}

func (d *idiscovery) startAdvertising(ctx *context.T, adinfo *AdInfo) (func(), error) {
	statName := naming.Join(d.statsPrefix, "ad", adinfo.Ad.Id.String())
	stats.NewStringFunc(statName, func() string { return fmt.Sprint(*adinfo) })
	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	var lastErr error
	isAdvertising := false
	for _, plugin := range d.plugins {
		wg.Add(1)
		if err := plugin.Advertise(ctx, adinfo, wg.Done); err != nil {
			// Only log errors advertising just as long as it succeeds in at least one plugin.
			// See https://github.com/vanadium/issues/issues/1404 for discussion on issues
			// with more than 16 BLE advertisements.
			vlog.Error("discovery: Unable to startAdvertisement for plugin: ", err)
			lastErr = err
		} else {
			isAdvertising = true
		}
	}
	if !isAdvertising {
		cancel()
		return nil, lastErr
	}

	stop := func() {
		stats.Delete(statName)
		cancel()
		wg.Wait()
	}
	return stop, nil
}

func (d *idiscovery) stopAdvertising(id discovery.AdId) {
	d.adMu.Lock()
	subtask := d.adSubtasks[id]
	delete(d.adSubtasks, id)
	d.adMu.Unlock()
	if subtask == nil {
		return
	}

	subtask.mu.Lock()
	if subtask.stop != nil {
		subtask.stop()
		subtask.stop = nil
	}
	subtask.mu.Unlock()
}

func (d *idiscovery) updateAdvertising(adinfo *AdInfo) {
	d.adMu.Lock()
	subtask := d.adSubtasks[adinfo.Ad.Id]
	if subtask == nil {
		d.adMu.Unlock()
		return
	}
	d.adMu.Unlock()

	subtask.mu.Lock()
	defer subtask.mu.Unlock()

	if subtask.stop == nil {
		return
	}
	subtask.stop()

	ctx := subtask.parent

	var err error
	subtask.stop, err = d.startAdvertising(ctx, adinfo)
	if err != nil {
		ctx.Error(err)
		d.cancelTask(ctx)
	}
}
