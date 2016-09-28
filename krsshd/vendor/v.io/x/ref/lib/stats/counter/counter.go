// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package counter implements counters that keeps track of their recent values
// over different periods of time.
// Example:
// c := counter.New()
// c.Incr(n)
// ...
// delta1h := c.Delta1h()
// delta10m := c.Delta10m()
// delta1m := c.Delta1m()
// and:
// rate1h := c.Rate1h()
// rate10m := c.Rate10m()
// rate1m := c.Rate1m()
package counter

import (
	"sync"
	"time"

	"v.io/x/ref/services/stats"
)

var (
	// Used for testing.
	TimeNow func() time.Time = time.Now
)

const (
	hour       = 0
	tenminutes = 1
	minute     = 2
)

// Counter is a counter that keeps track of its recent values over a given
// period of time, and with a given resolution. Use New() to instantiate.
type Counter struct {
	mu         sync.RWMutex
	ts         [3]*timeseries
	lastUpdate time.Time
}

// New returns a new Counter.
func New() *Counter {
	now := TimeNow()
	c := &Counter{}
	c.ts[hour] = newTimeSeries(now, time.Hour, time.Minute)
	c.ts[tenminutes] = newTimeSeries(now, 10*time.Minute, 10*time.Second)
	c.ts[minute] = newTimeSeries(now, time.Minute, time.Second)
	return c
}

func (c *Counter) advance() time.Time {
	now := TimeNow()
	for _, ts := range c.ts {
		ts.advanceTime(now)
	}
	return now
}

// Value returns the current value of the counter.
func (c *Counter) Value() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ts[minute].headValue()
}

// LastUpdate returns the last update time of the counter.
func (c *Counter) LastUpdate() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUpdate
}

// Set updates the current value of the counter.
func (c *Counter) Set(value int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastUpdate = c.advance()
	for _, ts := range c.ts {
		ts.set(value)
	}
}

// Incr increments the current value of the counter by 'delta'.
func (c *Counter) Incr(delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastUpdate = c.advance()
	for _, ts := range c.ts {
		ts.incr(delta)
	}
}

// Delta1h returns the delta for the last hour.
func (c *Counter) Delta1h() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.advance()
	return c.ts[hour].delta()
}

// Delta10m returns the delta for the last 10 minutes.
func (c *Counter) Delta10m() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.advance()
	return c.ts[tenminutes].delta()
}

// Delta1m returns the delta for the last minute.
func (c *Counter) Delta1m() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.advance()
	return c.ts[minute].delta()
}

// Rate1h returns the rate of change of the counter in the last hour.
func (c *Counter) Rate1h() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.advance()
	return c.ts[hour].rate()
}

// Rate10m returns the rate of change of the counter in the last 10 minutes.
func (c *Counter) Rate10m() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.advance()
	return c.ts[tenminutes].rate()
}

// Rate1m returns the rate of change of the counter in the last minute.
func (c *Counter) Rate1m() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.advance()
	return c.ts[minute].rate()
}

// TimeSeries1h returns the time series data in the last hour.
func (c *Counter) TimeSeries1h() stats.TimeSeries {
	return c.timeseries(c.ts[hour])
}

// TimeSeries10m returns the time series data in the last 10 minutes.
func (c *Counter) TimeSeries10m() stats.TimeSeries {
	return c.timeseries(c.ts[tenminutes])
}

// TimeSeries1m returns the time series data in the last minute.
func (c *Counter) TimeSeries1m() stats.TimeSeries {
	return c.timeseries(c.ts[minute])
}

func (c *Counter) timeseries(ts *timeseries) stats.TimeSeries {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return stats.TimeSeries{
		Values:     ts.values(),
		Resolution: ts.resolution,
		StartTime:  ts.tailTime(),
	}
}

// Reset resets the counter to an empty state.
func (c *Counter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := TimeNow()
	for _, ts := range c.ts {
		ts.reset(now)
	}
}
