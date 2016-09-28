// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ibe

import (
	"bytes"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bn256"
)

var magicNumber = []byte{0x1b, 0xe0} // prefix that appears in the marshaled form: 2 bytes

type marshaledType byte

const (
	// size of field element (256 bits = 32 bytes)
	fieldElemSize = 32

	// types of encoded bytes, 1 byte
	typeBB1Params     marshaledType = 0
	typeBB1PrivateKey               = 1
	typeBB1MasterKey                = 2
	typeBB2Params                   = 3
	typeBB2PrivateKey               = 4
	typeBB2MasterKey                = 5

	// Sizes excluding the magic number and type header.
	headerSize                 = 3
	marshaledBB1ParamsSize     = 2*marshaledG1Size + 2*marshaledG2Size + marshaledGTSize
	marshaledBB1PrivateKeySize = 2 * marshaledG2Size
	marshaledBB1MasterKeySize  = marshaledG2Size
	marshaledBB2ParamsSize     = 2*marshaledG1Size + marshaledGTSize
	marshaledBB2PrivateKeySize = fieldElemSize + marshaledG2Size
	marshaledBB2MasterKeySize  = 2*fieldElemSize + marshaledG2Size
)

func writeFieldElement(elem *big.Int) []byte {
	elemBytes := elem.Bytes()
	ret := make([]byte, fieldElemSize)
	copy(ret[fieldElemSize-len(elemBytes):], elemBytes)
	return ret
}

func writeHeader(typ marshaledType) []byte {
	ret := make([]byte, headerSize)
	copy(ret, magicNumber)
	ret[len(magicNumber)] = byte(typ)
	return ret
}

// readHeader parses hdr and returns the message type and the remainder of the
// message, excluding the header.
func readHeader(hdr []byte) (marshaledType, []byte, error) {
	if len(hdr) < headerSize {
		return 0, nil, fmt.Errorf("header is too small")
	}
	if !bytes.Equal(hdr[0:len(magicNumber)], magicNumber) {
		return 0, nil, fmt.Errorf("invalid magic number")
	}
	return marshaledType(hdr[len(magicNumber)]), hdr[headerSize:], nil
}

// MarshalParams encodes p into a byte slice.
func MarshalParams(p Params) ([]byte, error) {
	switch p := p.(type) {
	case *bb1params:
		ret := make([]byte, 0, headerSize+marshaledBB1ParamsSize)
		// g and gHat are the generators, do not need to be marshaled.
		for _, field := range [][]byte{
			writeHeader(typeBB1Params),
			p.g1.Marshal(),
			p.h.Marshal(),
			p.g1Hat.Marshal(),
			p.hHat.Marshal(),
			p.v.Marshal(),
		} {
			ret = append(ret, field...)
		}
		return ret, nil
	case *bb2params:
		ret := make([]byte, 0, headerSize+marshaledBB2ParamsSize)
		for _, field := range [][]byte{
			writeHeader(typeBB2Params),
			p.X.Marshal(),
			p.Y.Marshal(),
			p.v.Marshal(),
		} {
			ret = append(ret, field...)
		}
		return ret, nil
	default:
		return nil, fmt.Errorf("MarshalParams for %T for implemented yet", p)
	}
}

// UnmarshalParams parses an encoded Params object.
func UnmarshalParams(data []byte) (Params, error) {
	var typ marshaledType
	var err error
	if typ, data, err = readHeader(data); err != nil {
		return nil, err
	}
	advance := func(n int) []byte {
		ret := data[0:n]
		data = data[n:]
		return ret
	}
	switch typ {
	case typeBB1Params:
		if len(data) != marshaledBB1ParamsSize {
			return nil, fmt.Errorf("invalid size")
		}
		p := newbb1params()
		one := big.NewInt(1)
		p.g.ScalarBaseMult(one)
		p.gHat.ScalarBaseMult(one)
		if _, ok := p.g1.Unmarshal(advance(marshaledG1Size)); !ok {
			return nil, fmt.Errorf("failed to unmarshal g1")
		}
		if _, ok := p.h.Unmarshal(advance(marshaledG1Size)); !ok {
			return nil, fmt.Errorf("failed to unmarshal h")
		}
		if _, ok := p.g1Hat.Unmarshal(advance(marshaledG2Size)); !ok {
			return nil, fmt.Errorf("failed to unmarshal g1Hat")
		}
		if _, ok := p.hHat.Unmarshal(advance(marshaledG2Size)); !ok {
			return nil, fmt.Errorf("failed to unmarshal hHat")
		}
		if _, ok := p.v.Unmarshal(advance(marshaledGTSize)); !ok {
			return nil, fmt.Errorf("failed to unmarshal v")
		}
		return p, nil
	case typeBB2Params:
		if len(data) != marshaledBB2ParamsSize {
			return nil, fmt.Errorf("invalid size")
		}
		p := newbb2params()
		if _, ok := p.X.Unmarshal(advance(marshaledG1Size)); !ok {
			return nil, fmt.Errorf("failed to unmarshal X")
		}
		if _, ok := p.Y.Unmarshal(advance(marshaledG1Size)); !ok {
			return nil, fmt.Errorf("failed to unmarshal Y")
		}
		if _, ok := p.v.Unmarshal(advance(marshaledGTSize)); !ok {
			return nil, fmt.Errorf("failed to unmarshal v")
		}
		return p, nil
	default:
		return nil, fmt.Errorf("unrecognized Params type (%d)", typ)
	}
}

// MarshalPrivateKey encodes the private component of k into a byte slice.
func MarshalPrivateKey(k PrivateKey) ([]byte, error) {
	switch k := k.(type) {
	case *bb1PrivateKey:
		ret := make([]byte, 0, headerSize+marshaledBB1PrivateKeySize)
		for _, field := range [][]byte{
			writeHeader(typeBB1PrivateKey),
			k.d0.Marshal(),
			k.d1.Marshal(),
		} {
			ret = append(ret, field...)
		}
		return ret, nil
	case *bb2PrivateKey:
		ret := make([]byte, 0, headerSize+marshaledBB2PrivateKeySize)
		for _, field := range [][]byte{
			writeHeader(typeBB2PrivateKey),
			writeFieldElement(k.r),
			k.K.Marshal(),
		} {
			ret = append(ret, field...)
		}
		return ret, nil
	default:
		return nil, fmt.Errorf("MarshalPrivateKey for %T not implemented yet", k)
	}
}

// UnmarshalPrivateKey parses an encoded PrivateKey object.
func UnmarshalPrivateKey(params Params, data []byte) (PrivateKey, error) {
	var typ marshaledType
	var err error
	if typ, data, err = readHeader(data); err != nil {
		return nil, err
	}
	advance := func(n int) []byte {
		ret := data[0:n]
		data = data[n:]
		return ret
	}
	switch typ {
	case typeBB1PrivateKey:
		if len(data) != marshaledBB1PrivateKeySize {
			return nil, fmt.Errorf("invalid size")
		}
		k := &bb1PrivateKey{
			d0: new(bn256.G2),
			d1: new(bn256.G2),
		}
		if _, ok := k.d0.Unmarshal(advance(marshaledG2Size)); !ok {
			return nil, fmt.Errorf("failed to unmarshal d0")
		}
		if _, ok := k.d1.Unmarshal(advance(marshaledG2Size)); !ok {
			return nil, fmt.Errorf("failed to unmarshal d1")
		}
		if p, ok := params.(*bb1params); !ok {
			return nil, fmt.Errorf("params type %T incompatible with %T", params, k)
		} else {
			k.params = new(bb1params)
			*(k.params) = *p
		}
		return k, nil
	case typeBB2PrivateKey:
		if len(data) != marshaledBB2PrivateKeySize {
			return nil, fmt.Errorf("invalid size")
		}
		k := &bb2PrivateKey{
			r: new(big.Int),
			K: new(bn256.G2),
		}
		k.r.SetBytes(advance(fieldElemSize))
		if _, ok := k.K.Unmarshal(advance(marshaledG2Size)); !ok {
			return nil, fmt.Errorf("failed to unmarshal K")
		}
		if p, ok := params.(*bb2params); !ok {
			return nil, fmt.Errorf("params type %T incompatible with %T", params, k)
		} else {
			k.params = new(bb2params)
			*(k.params) = *p
		}
		return k, nil
	default:
		return nil, fmt.Errorf("unrecognized private key type (%d)", typ)
	}
}

// MarshalMasterKey encodes the private component of m into a byte slice.
func MarshalMasterKey(m Master) ([]byte, error) {
	switch m := m.(type) {
	case *bb1master:
		ret := make([]byte, 0, headerSize+marshaledBB1MasterKeySize)
		for _, field := range [][]byte{
			writeHeader(typeBB1MasterKey),
			m.g0Hat.Marshal(),
		} {
			ret = append(ret, field...)
		}
		return ret, nil
	case *bb2master:
		ret := make([]byte, 0, headerSize+marshaledBB2MasterKeySize)
		for _, field := range [][]byte{
			writeHeader(typeBB2MasterKey),
			writeFieldElement(m.x),
			writeFieldElement(m.y),
			m.hHat.Marshal(),
		} {
			ret = append(ret, field...)
		}
		return ret, nil
	default:
		return nil, fmt.Errorf("MarshalMasterKey for %T not implemented yet", m)
	}
}

// UnmarshalMasterKey parses an encoded Master object.
func UnmarshalMasterKey(params Params, data []byte) (Master, error) {
	var typ marshaledType
	var err error
	if typ, data, err = readHeader(data); err != nil {
		return nil, err
	}
	advance := func(n int) []byte {
		ret := data[0:n]
		data = data[n:]
		return ret
	}
	switch typ {
	case typeBB1MasterKey:
		if len(data) != marshaledBB1MasterKeySize {
			return nil, fmt.Errorf("invalid size")
		}
		m := &bb1master{
			g0Hat: new(bn256.G2),
		}
		if _, ok := m.g0Hat.Unmarshal(advance(marshaledG2Size)); !ok {
			return nil, fmt.Errorf("failed to unmarshal g0Hat")
		}
		p, ok := params.(*bb1params)
		if !ok {
			return nil, fmt.Errorf("params type %T incompatible with %T", params, m)
		}
		m.params = new(bb1params)
		*(m.params) = *p
		return m, nil
	case typeBB2MasterKey:
		if len(data) != marshaledBB2MasterKeySize {
			return nil, fmt.Errorf("invalid size")
		}
		m := &bb2master{
			x:    new(big.Int),
			y:    new(big.Int),
			hHat: new(bn256.G2),
		}
		m.x.SetBytes(advance(fieldElemSize))
		m.y.SetBytes(advance(fieldElemSize))
		if _, ok := m.hHat.Unmarshal(advance(marshaledG2Size)); !ok {
			return nil, fmt.Errorf("failed to unmarshal hHat")
		}
		p, ok := params.(*bb2params)
		if !ok {
			return nil, fmt.Errorf("params type %T incompatible with %T", params, m)
		}
		m.params = new(bb2params)
		*(m.params) = *p
		return m, nil
	default:
		return nil, fmt.Errorf("unrecognized master key type (%d)", typ)
	}
}
