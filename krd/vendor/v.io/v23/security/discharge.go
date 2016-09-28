// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package security

import (
	"reflect"
	"time"

	"v.io/v23/vom"
)

// Discharge represents a "proof" required for satisfying a ThirdPartyCaveat.
//
// A discharge may have caveats of its own (including ThirdPartyCaveats) that
// restrict the context in which the discharge is usable.
//
// Discharge objects are immutable and multiple goroutines may invoke methods
// on a Discharge simultaneously.
//
// See also: https://vanadium.github.io/glossary.html#discharge
type Discharge struct {
	wire WireDischarge
}

// ID returns the identifier for the third-party caveat that d is a discharge
// for.
func (d Discharge) ID() string {
	switch v := d.wire.(type) {
	case WireDischargePublicKey:
		return v.Value.ThirdPartyCaveatId
	default:
		return ""
	}
}

// ThirdPartyCaveats returns the set of third-party caveats on the scope of the
// discharge.
func (d Discharge) ThirdPartyCaveats() []ThirdPartyCaveat {
	var ret []ThirdPartyCaveat
	switch v := d.wire.(type) {
	case WireDischargePublicKey:
		for _, cav := range v.Value.Caveats {
			if tp := cav.ThirdPartyDetails(); tp != nil {
				ret = append(ret, tp)
			}
		}
	}
	return ret
}

// Expiry returns the time at which d will no longer be valid, or the zero
// value of time.Time if the discharge does not expire.
func (d Discharge) Expiry() time.Time {
	var min time.Time
	switch v := d.wire.(type) {
	case WireDischargePublicKey:
		for _, cav := range v.Value.Caveats {
			if t := expiryTime(cav); !t.IsZero() && (min.IsZero() || t.Before(min)) {
				min = t
			}
		}
	}
	return min
}

// Equivalent returns true if 'd' and 'discharge' can be used interchangeably,
// i.e. any authorizations that are enabled by 'd' will be enabled by
// 'discharge' and vice versa.
func (d Discharge) Equivalent(discharge Discharge) bool {
	return reflect.DeepEqual(d, discharge)
}

// VDLIsZero implements the vdl.IsZeroer interface, and returns true if d
// represents an empty discharge.
func (d Discharge) VDLIsZero() bool {
	return d.wire == nil || d.wire.VDLIsZero()
}

func WireDischargeToNative(wire WireDischarge, native *Discharge) error {
	native.wire = wire
	return nil
}

func WireDischargeFromNative(wire *WireDischarge, native Discharge) error {
	*wire = native.wire
	return nil
}

func expiryTime(cav Caveat) time.Time {
	switch cav.Id {
	case ExpiryCaveat.Id:
		var t time.Time
		if err := vom.Decode(cav.ParamVom, &t); err != nil {
			// TODO(jsimsa): Decide what (if any) logging mechanism to use.
			// vlog.Errorf("Failed to decode ParamVOM for cav(%v): %v", cav, err)
			return time.Time{}
		}
		return t
	}
	return time.Time{}
}
