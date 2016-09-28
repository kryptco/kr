// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package security

import (
	"fmt"
	"time"

	"v.io/v23/context"
	"v.io/v23/verror"
)

// Prefix for error codes.
const pkgPath = "v.io/v23/security"

var (
	errMissingDischarge = verror.Register(pkgPath+".errMissingDischarge", verror.NoRetry, "{1:}{2:}missing discharge for third party caveat(id={3}){:_}")
	errInvalidDischarge = verror.Register(pkgPath+".errInvalidDischarge", verror.NoRetry, "{1:}{2:}invalid discharge({3}) for caveat({4}){:_}")
	errFailedDischarge  = verror.Register(pkgPath+".errFailedDischarge", verror.NoRetry, "{1:}{2:}a caveat({3}) on the discharge failed to validate{:_}")
)

func init() {
	RegisterCaveatValidator(ConstCaveat, func(ctx *context.T, _ Call, isValid bool) error {
		if isValid {
			return nil
		}
		return NewErrConstCaveatValidation(ctx)
	})

	RegisterCaveatValidator(ExpiryCaveat, func(ctx *context.T, call Call, expiry time.Time) error {
		now := call.Timestamp()
		if now.After(expiry) {
			return NewErrExpiryCaveatValidation(ctx, now, expiry)
		}
		return nil
	})

	RegisterCaveatValidator(MethodCaveat, func(ctx *context.T, call Call, methods []string) error {
		for _, m := range methods {
			if call.Method() == m {
				return nil
			}
		}
		return NewErrMethodCaveatValidation(ctx, call.Method(), methods)
	})

	RegisterCaveatValidator(PeerBlessingsCaveat, func(ctx *context.T, call Call, patterns []BlessingPattern) error {
		lnames := LocalBlessingNames(ctx, call)
		for _, p := range patterns {
			if p.MatchedBy(lnames...) {
				return nil

			}
		}
		return NewErrPeerBlessingsCaveatValidation(ctx, lnames, patterns)
	})

	RegisterCaveatValidator(PublicKeyThirdPartyCaveat, func(ctx *context.T, call Call, params publicKeyThirdPartyCaveatParam) error {
		discharge, ok := call.RemoteDischarges()[params.ID()]
		if !ok {
			return verror.New(errMissingDischarge, ctx, params.ID())
		}
		// Must be of the valid type.
		var d *PublicKeyDischarge
		switch v := discharge.wire.(type) {
		case WireDischargePublicKey:
			d = &v.Value
		default:
			return verror.New(errInvalidDischarge, ctx, fmt.Sprintf("%T", v), fmt.Sprintf("%T", params))
		}
		// Must be signed by the principal designated by c.DischargerKey
		key, err := params.discharger(ctx)
		if err != nil {
			return err
		}
		if err := d.verify(ctx, key); err != nil {
			return err
		}
		// And all caveats on the discharge must be met.
		for _, cav := range d.Caveats {
			if err := cav.Validate(ctx, call); err != nil {
				return verror.New(errFailedDischarge, ctx, cav, err)
			}
		}
		return nil
	})
}
