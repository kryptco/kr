// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package histogram implements a basic histogram to keep track of data
// distribution.
package histogram

import (
	"time"

	"v.io/v23/verror"
	"v.io/x/ref/lib/stats/counter"
	"v.io/x/ref/services/stats"
)

const pkgPath = "v.io/x/ref/lib/stats/histogram"

var (
	errNoBucketForValue = verror.Register(pkgPath+".errNoBucketForValue", verror.NoRetry, "{1:}{2:} no bucket for value{:_}")
)

// A Histogram accumulates values in the form of a histogram. The type of the
// values is int64, which is suitable for keeping track of things like RPC
// latency in milliseconds. New histogram objects should be obtained via the
// New() function.
type Histogram struct {
	opts    Options
	buckets []bucketInternal
	count   *counter.Counter
	sum     *counter.Counter
	tracker *counter.Tracker
}

// Options contains the parameters that define the histogram's buckets.
type Options struct {
	// NumBuckets is the number of buckets.
	NumBuckets int
	// GrowthFactor is the growth factor of the buckets. A value of 0.1
	// indicates that bucket N+1 will be 10% larger than bucket N.
	GrowthFactor float64
	// SmallestBucketSize is the size of the first bucket. Bucket sizes are
	// rounded down to the nearest integer.
	SmallestBucketSize float64
	// MinValue is the lower bound of the first bucket.
	MinValue int64
}

// bucketInternal is the internal representation of a bucket, which includes a
// rate counter.
type bucketInternal struct {
	lowBound int64
	count    *counter.Counter
}

// New returns a pointer to a new Histogram object that was created with the
// provided options.
func New(opts Options) *Histogram {
	if opts.NumBuckets == 0 {
		opts.NumBuckets = 32
	}
	if opts.SmallestBucketSize == 0.0 {
		opts.SmallestBucketSize = 1.0
	}
	h := Histogram{
		opts:    opts,
		buckets: make([]bucketInternal, opts.NumBuckets),
		count:   counter.New(),
		sum:     counter.New(),
		tracker: counter.NewTracker(),
	}
	low := opts.MinValue
	delta := opts.SmallestBucketSize
	for i := 0; i < opts.NumBuckets; i++ {
		h.buckets[i].lowBound = low
		h.buckets[i].count = counter.New()
		low = low + int64(delta)
		delta = delta * (1.0 + opts.GrowthFactor)
	}
	return &h
}

// Opts returns a copy of the options used to create the Histogram.
func (h *Histogram) Opts() Options {
	return h.opts
}

// Add adds a value to the histogram.
func (h *Histogram) Add(value int64) error {
	bucket, err := h.findBucket(value)
	if err != nil {
		return err
	}
	h.buckets[bucket].count.Incr(1)
	h.count.Incr(1)
	h.sum.Incr(value)
	h.tracker.Push(value)
	return nil
}

// LastUpdate returns the time at which the object was last updated.
func (h *Histogram) LastUpdate() time.Time {
	return h.count.LastUpdate()
}

// Value returns the accumulated state of the histogram since it was created.
func (h *Histogram) Value() stats.HistogramValue {
	b := make([]stats.HistogramBucket, len(h.buckets))
	for i, v := range h.buckets {
		b[i] = stats.HistogramBucket{
			LowBound: v.lowBound,
			Count:    v.count.Value(),
		}
	}

	v := stats.HistogramValue{
		Count:   h.count.Value(),
		Sum:     h.sum.Value(),
		Min:     h.tracker.Min(),
		Max:     h.tracker.Max(),
		Buckets: b,
	}
	return v
}

// Delta1h returns the change in the last hour.
func (h *Histogram) Delta1h() stats.HistogramValue {
	b := make([]stats.HistogramBucket, len(h.buckets))
	for i, v := range h.buckets {
		b[i] = stats.HistogramBucket{
			LowBound: v.lowBound,
			Count:    v.count.Delta1h(),
		}
	}

	v := stats.HistogramValue{
		Count:   h.count.Delta1h(),
		Sum:     h.sum.Delta1h(),
		Min:     h.tracker.Min1h(),
		Max:     h.tracker.Max1h(),
		Buckets: b,
	}
	return v
}

// Delta10m returns the change in the last 10 minutes.
func (h *Histogram) Delta10m() stats.HistogramValue {
	b := make([]stats.HistogramBucket, len(h.buckets))
	for i, v := range h.buckets {
		b[i] = stats.HistogramBucket{
			LowBound: v.lowBound,
			Count:    v.count.Delta10m(),
		}
	}

	v := stats.HistogramValue{
		Count:   h.count.Delta10m(),
		Sum:     h.sum.Delta10m(),
		Min:     h.tracker.Min10m(),
		Max:     h.tracker.Max10m(),
		Buckets: b,
	}
	return v
}

// Delta1m returns the change in the last 10 minutes.
func (h *Histogram) Delta1m() stats.HistogramValue {
	b := make([]stats.HistogramBucket, len(h.buckets))
	for i, v := range h.buckets {
		b[i] = stats.HistogramBucket{
			LowBound: v.lowBound,
			Count:    v.count.Delta1m(),
		}
	}

	v := stats.HistogramValue{
		Count:   h.count.Delta1m(),
		Sum:     h.sum.Delta1m(),
		Min:     h.tracker.Min1m(),
		Max:     h.tracker.Max1m(),
		Buckets: b,
	}
	return v
}

// findBucket does a binary search to find in which bucket the value goes.
func (h *Histogram) findBucket(value int64) (int, error) {
	lastBucket := len(h.buckets) - 1
	min, max := 0, lastBucket
	for max >= min {
		b := (min + max) / 2
		if value >= h.buckets[b].lowBound && (b == lastBucket || value < h.buckets[b+1].lowBound) {
			return b, nil
		}
		if value < h.buckets[b].lowBound {
			max = b - 1
			continue
		}
		min = b + 1
	}
	return 0, verror.New(errNoBucketForValue, nil, value)
}
