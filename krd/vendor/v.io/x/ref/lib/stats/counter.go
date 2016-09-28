// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"time"

	"v.io/x/ref/lib/stats/counter"
)

// NewCounter creates a new Counter StatsObject with the given name and
// returns a pointer to it.
func NewCounter(name string) *counter.Counter {
	lock.Lock()
	defer lock.Unlock()

	node := findNodeLocked(name, true)
	c := counter.New()
	cw := &counterWrapper{c}
	node.object = cw

	addCounterChild(node, name+"/delta1h", cw, time.Hour, cw.Delta1h)
	addCounterChild(node, name+"/delta10m", cw, 10*time.Minute, cw.Delta10m)
	addCounterChild(node, name+"/delta1m", cw, time.Minute, cw.Delta1m)
	addCounterChild(node, name+"/rate1h", cw, time.Hour, cw.Rate1h)
	addCounterChild(node, name+"/rate10m", cw, 10*time.Minute, cw.Rate10m)
	addCounterChild(node, name+"/rate1m", cw, time.Minute, cw.Rate1m)
	addCounterChild(node, name+"/timeseries1h", cw, time.Hour, cw.TimeSeries1h)
	addCounterChild(node, name+"/timeseries10m", cw, 10*time.Minute, cw.TimeSeries10m)
	addCounterChild(node, name+"/timeseries1m", cw, time.Minute, cw.TimeSeries1m)
	return c
}

type counterWrapper struct {
	c *counter.Counter
}

func (cw counterWrapper) LastUpdate() time.Time {
	return cw.c.LastUpdate()
}

func (cw counterWrapper) Value() interface{} {
	return cw.c.Value()
}

func (cw counterWrapper) Delta1h() interface{} {
	return cw.c.Delta1h()
}
func (cw counterWrapper) Delta10m() interface{} {
	return cw.c.Delta10m()
}
func (cw counterWrapper) Delta1m() interface{} {
	return cw.c.Delta1m()
}
func (cw counterWrapper) Rate1h() interface{} {
	return cw.c.Rate1h()
}
func (cw counterWrapper) Rate10m() interface{} {
	return cw.c.Rate10m()
}
func (cw counterWrapper) Rate1m() interface{} {
	return cw.c.Rate1m()
}
func (cw counterWrapper) TimeSeries1h() interface{} {
	return cw.c.TimeSeries1h()
}
func (cw counterWrapper) TimeSeries10m() interface{} {
	return cw.c.TimeSeries10m()
}
func (cw counterWrapper) TimeSeries1m() interface{} {
	return cw.c.TimeSeries1m()
}

type counterChild struct {
	c      *counterWrapper
	period time.Duration
	value  func() interface{}
}

func (cc counterChild) LastUpdate() time.Time {
	now := time.Now()
	if t := cc.c.LastUpdate().Add(cc.period); t.Before(now) {
		return t
	}
	return now
}

func (cc counterChild) Value() interface{} {
	return cc.value()
}

func addCounterChild(parent *node, name string, c *counterWrapper, period time.Duration, value func() interface{}) {
	child := findNodeLocked(name, true)
	child.object = &counterChild{c, period, value}
}
