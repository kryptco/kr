// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package verror

import "v.io/v23/vdl"

func init() {
	// TODO(bprosnitz) Remove this old-style registration.
	// We must register the error conversion functions between vdl.WireError and
	// the standard error interface with the vdl package.  This allows the vdl
	// package to have minimal dependencies.
	vdl.RegisterNativeError(WireToNative, WireFromNative)

	// New-style error registration.
	vdl.RegisterNative(ErrorToNative, ErrorFromNative)
}

// TODO(toddw): rename Error{To,From}Native to Wire{To,From}Native, after we've
// switched to the new vdl Encoder/Decoder, and the old functions are no longer
// used.

// ErrorToNative converts from the wire to native representation of errors.
func ErrorToNative(wire *vdl.WireError, native *error) error {
	if wire == nil {
		*native = nil
		return nil
	}
	e := new(E)
	*native = e
	return WireToNative(*wire, e)
}

// ErrorFromNative converts from the native to wire representation of errors.
func ErrorFromNative(wire **vdl.WireError, native error) error {
	if native == nil {
		*wire = nil
		return nil
	}
	if *wire == nil {
		*wire = new(vdl.WireError)
	}
	return WireFromNative(*wire, native)
}

// FromWire is a convenience for generated code to convert wire errors into
// native errors.
func FromWire(wire *vdl.WireError) error {
	var native error
	if err := ErrorToNative(wire, &native); err != nil {
		native = err
	}
	return native
}

// WireToNative converts from vdl.WireError to verror.E, which
// implements the standard go error interface.
//
// TODO(toddw): Remove this function after the switch to the new vdl
// Encoder/Decoder is complete.
func WireToNative(wire vdl.WireError, native *E) error {
	*native = E{
		ID:     ID(wire.Id),
		Action: retryToAction(wire.RetryCode),
		Msg:    wire.Msg,
	}
	for _, pWire := range wire.ParamList {
		var pNative interface{}
		if err := vdl.Convert(&pNative, pWire); err != nil {
			// It's questionable what to do if the conversion fails, rather than
			// ending up with a native Go value.
			//
			// At the moment, we plug the *vdl.Value into the native params.  The idea
			// is that this will still be more useful to the user, since they'll still
			// have the error Id and Action.
			//
			// TODO(toddw): Consider whether there is a better strategy.
			pNative = pWire
		}
		native.ParamList = append(native.ParamList, pNative)
	}
	return nil
}

// WireFromNative converts from the standard go error interface to
// verror.E, and then to vdl.WireError.
//
// TODO(toddw): Remove this function after the switch to the new vdl
// Encoder/Decoder is complete.
func WireFromNative(wire *vdl.WireError, native error) error {
	e := ExplicitConvert(ErrUnknown, "", "", "", native)
	*wire = vdl.WireError{
		Id:        string(ErrorID(e)),
		RetryCode: retryFromAction(Action(e)),
		Msg:       e.Error(),
	}
	for _, pNative := range params(e) {
		var pWire *vdl.Value
		if err := vdl.Convert(&pWire, pNative); err != nil {
			// It's questionable what to do here if the conversion fails, similarly to
			// the conversion failure above in WireToNative.
			//
			// TODO(toddw): Consider whether there is a better strategy.
			pWire = vdl.StringValue(nil, err.Error())
		}
		wire.ParamList = append(wire.ParamList, pWire)
	}
	return nil
}

func retryToAction(retry vdl.WireRetryCode) ActionCode {
	switch retry {
	case vdl.WireRetryCodeNoRetry:
		return NoRetry
	case vdl.WireRetryCodeRetryConnection:
		return RetryConnection
	case vdl.WireRetryCodeRetryRefetch:
		return RetryRefetch
	case vdl.WireRetryCodeRetryBackoff:
		return RetryBackoff
	}
	// Backoff to no retry by default.
	return NoRetry
}

func retryFromAction(action ActionCode) vdl.WireRetryCode {
	switch action.RetryAction() {
	case NoRetry:
		return vdl.WireRetryCodeNoRetry
	case RetryConnection:
		return vdl.WireRetryCodeRetryConnection
	case RetryRefetch:
		return vdl.WireRetryCodeRetryRefetch
	case RetryBackoff:
		return vdl.WireRetryCodeRetryBackoff
	}
	// Backoff to no retry by default.
	return vdl.WireRetryCodeNoRetry
}

// VDLRead implements the logic to read x from dec.
//
// Unlike regular VDLRead implementations, this handles the case where the
// decoder contains a nil value, to make code generation simpler.
func VDLRead(dec vdl.Decoder, x *error) error {
	if err := dec.StartValue(vdl.ErrorType.Elem()); err != nil {
		return err
	}
	if dec.IsNil() {
		*x = nil
		return dec.FinishValue()
	}
	dec.IgnoreNextStartValue()
	var wire vdl.WireError
	if err := wire.VDLRead(dec); err != nil {
		return err
	}
	nativePtr := new(E)
	if err := WireToNative(wire, nativePtr); err != nil {
		return err
	}
	*x = nativePtr
	return nil
}

// VDLWrite implements the logic to write x to enc.
//
// Unlike regular VDLWrite implementations, this handles the case where x
// contains a nil value, to make code generation simpler.
func VDLWrite(enc vdl.Encoder, x error) error {
	if x == nil {
		return enc.NilValue(vdl.ErrorType)
	}
	var wire vdl.WireError
	if err := WireFromNative(&wire, x); err != nil {
		return err
	}
	return wire.VDLWrite(enc)
}
