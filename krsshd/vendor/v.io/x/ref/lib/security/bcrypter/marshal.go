// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bcrypter

import (
	"v.io/x/lib/ibe"
)

// ToWire marshals the Ciphertext 'c' into the WireCiphertext 'wire'
func (c *Ciphertext) ToWire(wire *WireCiphertext) {
	*wire = c.wire
}

// FromWire unmarshals the provided WireCiphertext into the Ciphertext 'c'.
func (c *Ciphertext) FromWire(wire WireCiphertext) {
	c.wire = wire
}

// ToWire marshals the Params 'p' into the WireParams 'wire'.
func (p *Params) ToWire(wire *WireParams) error {
	ibeParamsBytes, err := ibe.MarshalParams(p.params)
	if err != nil {
		return err
	}
	wire.Blessing = p.blessing
	wire.Params = ibeParamsBytes
	return nil
}

// FromWire unmarshals the provided WireParams into the Params 'p'.
func (p *Params) FromWire(wire WireParams) error {
	ibeParams, err := ibe.UnmarshalParams(wire.Params)
	if err != nil {
		return err
	}
	p.params = ibeParams
	p.blessing = wire.Blessing
	return nil
}

// ToWire marshals the PrivateKey 'k' into the WirePrivateKey 'wire'.
func (k *PrivateKey) ToWire(wire *WirePrivateKey) error {
	if err := k.params.ToWire(&wire.Params); err != nil {
		return err
	}
	wire.Blessing = k.blessing
	wire.Keys = make([][]byte, len(k.keys))
	var err error
	for i, ibeKey := range k.keys {
		if wire.Keys[i], err = ibe.MarshalPrivateKey(ibeKey); err != nil {
			return err
		}
	}
	return nil
}

// FromWire unmarshals the provided WirePrivateKey into the PrivateKey 'k'.
func (k *PrivateKey) FromWire(wire WirePrivateKey) error {
	var params Params
	if err := params.FromWire(wire.Params); err != nil {
		return err
	}
	k.blessing = wire.Blessing
	k.params = params
	k.keys = make([]ibe.PrivateKey, len(wire.Keys))
	var err error
	for i, ibeKeyBytes := range wire.Keys {
		if k.keys[i], err = ibe.UnmarshalPrivateKey(k.params.params, ibeKeyBytes); err != nil {
			return err
		}
	}
	return nil
}
