// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package time

import (
	"fmt"
	"time"
)

const (
	nanosPerSecond = 1000 * 1000 * 1000
	secondsPerDay  = 24 * 60 * 60

	// Represent the unix epoch 1970-01-01 in terms of our epoch 0001-01-01, for
	// easy conversions.  Note that we use a proleptic Gregorian calendar; there
	// is a leap year every 4 years, except for years divisible by 100, but
	// including years divisible by 400.
	unixEpoch = (1969*365 + 1969/4 - 1969/100 + 1969/400) * secondsPerDay

	minInt64         = -(1 << 63)
	maxInt64         = ((1 << 63) - 1)
	minGoDurationSec = minInt64 / nanosPerSecond
	maxGoDurationSec = maxInt64 / nanosPerSecond
)

// TimeToNative is called by VDL for conversions from wire to native times.
func TimeToNative(wire Time, native *time.Time) error {
	*native = time.Unix(wire.Seconds-unixEpoch, int64(wire.Nanos)).UTC()
	return nil
}

// TimeFromNative is called by VDL for conversions from native to wire times.
func TimeFromNative(wire *Time, native time.Time) error {
	wire.Seconds = native.Unix() + unixEpoch
	wire.Nanos = int32(native.Nanosecond())
	*wire = wire.Normalize()
	return nil
}

// Normalize returns the normalized representation of x.  It makes a best-effort
// attempt to clean up invalid values, e.g. if Nanos is outside the valid range,
// or the sign of Nanos doesn't match the sign of Seconds.  The behavior is
// undefined for large invalid values, e.g. {int64max,int32max}.
func (x Time) Normalize() Time {
	return Time(Duration(x).Normalize())
}

// Now returns the current time.
func Now() Time {
	var t Time
	TimeFromNative(&t, time.Now())
	return t
}

// DurationToNative is called by VDL for conversions from wire to native
// durations.
func DurationToNative(wire Duration, native *time.Duration) error {
	*native = 0
	// Go represents duration as int64 nanoseconds, which has a much smaller range
	// than VDL duration, so we catch these cases and return an error.
	wire = wire.Normalize()
	if wire.Seconds < minGoDurationSec ||
		(wire.Seconds == minGoDurationSec && wire.Nanos < minInt64-minGoDurationSec*nanosPerSecond) ||
		wire.Seconds > maxGoDurationSec ||
		(wire.Seconds == maxGoDurationSec && wire.Nanos > maxInt64-maxGoDurationSec*nanosPerSecond) {
		return fmt.Errorf("vdl duration %+v out of range of go duration", wire)
	}
	*native = time.Duration(wire.Seconds*nanosPerSecond + int64(wire.Nanos))
	return nil
}

// DurationFromNative is called by VDL for conversions from native to wire
// durations.
func DurationFromNative(wire *Duration, native time.Duration) error {
	wire.Seconds = int64(native / nanosPerSecond)
	wire.Nanos = int32(native % nanosPerSecond)
	return nil
}

// Normalize returns the normalized representation of x.  It makes a best-effort
// attempt to clean up invalid values, e.g. if Nanos is outside the valid range,
// or the sign of Nanos doesn't match the sign of Seconds.  The behavior is
// undefined for large invalid values, e.g. {int64max,int32max}.
func (x Duration) Normalize() Duration {
	x.Seconds += int64(x.Nanos / nanosPerSecond)
	x.Nanos = x.Nanos % nanosPerSecond
	switch {
	case x.Seconds < 0 && x.Nanos > 0:
		x.Seconds += 1
		x.Nanos -= nanosPerSecond
	case x.Seconds > 0 && x.Nanos < 0:
		x.Seconds -= 1
		x.Nanos += nanosPerSecond
	}
	return x
}

// Deadline represents the deadline for an operation; it is the native
// representation for WireDeadline, and is automatically converted to/from
// WireDeadline during marshaling.
//
// Deadline represents the deadline as an absolute time, while WireDeadline
// represents the deadline as a relative duration from "now".
//
// To represent "no deadline", use the zero value for Deadline.
type Deadline struct {
	// Time represents the deadline as an absolute point in time.
	time.Time
}

// WireDeadlineToNative is called by VDL for conversions from wire to native
// deadlines.
func WireDeadlineToNative(wire WireDeadline, native *Deadline) error {
	if wire.FromNow == 0 {
		native.Time = time.Time{}
	} else {
		native.Time = time.Now().Add(wire.FromNow)
	}
	return nil
}

// WireDeadlineFromNative is called by VDL for conversions from native to wire
// deadlines.
func WireDeadlineFromNative(wire *WireDeadline, native Deadline) error {
	if native.IsZero() {
		wire.FromNow = 0
	} else {
		wire.FromNow = native.Sub(time.Now())
		// Ensure that we never set FromNow=0, since that is special-cased to mean
		// "no deadline".
		//
		// NOTE: A previous version of this code didn't include this special-case,
		// and instead set the WireDeadline.NoDeadline field (which has now been
		// removed) to indicate "no deadline".  This means that there is a tiny
		// chance that an old binary can send FromNow=0 to indicate "immediate
		// deadline", which a new binary will interpret as "no deadline".  As all
		// binaries are updated to this new code, this problem will go away.
		if wire.FromNow == 0 {
			wire.FromNow = 1
		}
	}
	return nil
}
