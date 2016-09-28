// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"time"

	"v.io/x/ref/lib/stats/histogram"
)

// NewHistogram creates a new Histogram StatsObject with the given name and
// returns a pointer to it.
func NewHistogram(name string, opts histogram.Options) *histogram.Histogram {
	lock.Lock()
	defer lock.Unlock()

	node := findNodeLocked(name, true)
	h := histogram.New(opts)
	hw := &histogramWrapper{h}
	node.object = hw

	addHistogramChild(node, name+"/delta1h", hw, time.Hour, hw.Delta1h)
	addHistogramChild(node, name+"/delta10m", hw, 10*time.Minute, hw.Delta10m)
	addHistogramChild(node, name+"/delta1m", hw, time.Minute, hw.Delta1m)
	return h
}

type histogramWrapper struct {
	h *histogram.Histogram
}

func (hw histogramWrapper) LastUpdate() time.Time {
	return hw.h.LastUpdate()
}

func (hw histogramWrapper) Value() interface{} {
	return hw.h.Value()
}

func (hw histogramWrapper) Delta1h() interface{} {
	return hw.h.Delta1h()
}

func (hw histogramWrapper) Delta10m() interface{} {
	return hw.h.Delta10m()
}

func (hw histogramWrapper) Delta1m() interface{} {
	return hw.h.Delta1m()
}

type histogramChild struct {
	h      *histogramWrapper
	period time.Duration
	value  func() interface{}
}

func (hc histogramChild) LastUpdate() time.Time {
	now := time.Now()
	if t := hc.h.LastUpdate().Add(hc.period); t.Before(now) {
		return t
	}
	return now
}

func (hc histogramChild) Value() interface{} {
	return hc.value()
}

func addHistogramChild(parent *node, name string, h *histogramWrapper, period time.Duration, value func() interface{}) {
	child := findNodeLocked(name, true)
	child.object = &histogramChild{h, period, value}
}
