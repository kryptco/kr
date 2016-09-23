// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Print writes textual output of the histogram values.
func (v HistogramValue) Print(w io.Writer) {
	avg := float64(v.Sum) / float64(v.Count)
	fmt.Fprintf(w, "Count: %d  Min: %d  Max: %d  Avg: %.2f\n", v.Count, v.Min, v.Max, avg)
	fmt.Fprintf(w, "%s\n", strings.Repeat("-", 60))
	if v.Count <= 0 {
		return
	}

	maxBucketDigitLen := len(strconv.FormatInt(v.Buckets[len(v.Buckets)-1].LowBound, 10))
	if maxBucketDigitLen < 3 {
		// For "inf".
		maxBucketDigitLen = 3
	}
	maxCountDigitLen := len(strconv.FormatInt(v.Count, 10))
	percentMulti := 100 / float64(v.Count)

	accCount := int64(0)
	for i, b := range v.Buckets {
		fmt.Fprintf(w, "[%*d, ", maxBucketDigitLen, b.LowBound)
		if i+1 < len(v.Buckets) {
			fmt.Fprintf(w, "%*d)", maxBucketDigitLen, v.Buckets[i+1].LowBound)
		} else {
			fmt.Fprintf(w, "%*s)", maxBucketDigitLen, "inf")
		}

		accCount += b.Count
		fmt.Fprintf(w, "  %*d  %5.1f%%  %5.1f%%", maxCountDigitLen, b.Count, float64(b.Count)*percentMulti, float64(accCount)*percentMulti)

		const barScale = 0.1
		barLength := int(float64(b.Count)*percentMulti*barScale + 0.5)
		fmt.Fprintf(w, "  %s\n", strings.Repeat("#", barLength))
	}
}

// String returns the textual output of the histogram values as string.
func (v HistogramValue) String() string {
	var b bytes.Buffer
	v.Print(&b)
	return b.String()
}

// Print writes textual output of the TimeSeries values.
func (ts TimeSeries) Print(w io.Writer) {
	fmt.Fprintf(w, "Start time: %v\nResolution: %v\nValues: %v", ts.StartTime, ts.Resolution, ts.Values)
}

// String returns the textual output of the TimeSeries values as string.
func (ts TimeSeries) String() string {
	var b bytes.Buffer
	ts.Print(&b)
	return b.String()
}
