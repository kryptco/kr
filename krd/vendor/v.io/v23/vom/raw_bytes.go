// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vom

import (
	"bytes"
	"fmt"
	"strconv"

	"v.io/v23/vdl"
)

type RawBytes struct {
	Version    Version
	Type       *vdl.Type
	RefTypes   []*vdl.Type
	AnyLengths []int
	Data       []byte
}

func (RawBytes) __VDLReflect(struct {
	Type interface{} // ensure vdl.TypeOf(RawBytes{}) returns vdl.AnyType
}) {
}

func RawBytesOf(value interface{}) *RawBytes {
	rb, err := RawBytesFromValue(value)
	if err != nil {
		panic(err)
	}
	return rb
}

func RawBytesFromValue(value interface{}) (*RawBytes, error) {
	// TODO(bprosnitz) This implementation is temporary - we should make it faster
	data, err := Encode(value)
	if err != nil {
		return nil, err
	}
	rb := new(RawBytes)
	if err := Decode(data, rb); err != nil {
		return nil, err
	}
	return rb, nil
}

// String outputs a string representation of RawBytes of the form
// RawBytes{Version81, int8, RefTypes{bool, string}, AnyLengths{4}, fa0e9dcc}
func (rb *RawBytes) String() string {
	var buf bytes.Buffer
	buf.WriteString("RawBytes{")
	buf.WriteString(rb.Version.String())
	buf.WriteString(", ")
	buf.WriteString(rb.Type.String())
	buf.WriteString(", ")
	if len(rb.RefTypes) > 0 {
		buf.WriteString("RefTypes{")
		for i, t := range rb.RefTypes {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(t.String())
		}
		buf.WriteString("}, ")
	}
	if len(rb.AnyLengths) > 0 {
		buf.WriteString("AnyLengths{")
		for i, l := range rb.AnyLengths {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(strconv.Itoa(l))
		}
		buf.WriteString("}, ")
	}
	buf.WriteString(fmt.Sprintf("%x}", rb.Data))
	return buf.String()
}

func (rb *RawBytes) ToValue(value interface{}) error {
	// TODO(bprosnitz) This implementation is temporary - we should make it faster
	dat, err := Encode(rb)
	if err != nil {
		return err
	}
	return Decode(dat, value)
}

func (rb *RawBytes) IsNil() bool {
	if rb == nil {
		return true
	}
	if rb.Type != vdl.AnyType && rb.Type.Kind() != vdl.Optional {
		return false
	}
	return len(rb.Data) == 1 && rb.Data[0] == WireCtrlNil
}

func (rb *RawBytes) Decoder() vdl.Decoder {
	decoder := NewDecoder(bytes.NewReader(rb.Data))
	dec := &decoder.dec
	dec.buf.version = rb.Version
	dec.refTypes.tids = make([]TypeId, len(rb.RefTypes))
	for i, refType := range rb.RefTypes {
		tid := TypeId(i) + WireIdFirstUserType
		dec.typeDec.idToType[tid] = refType
		dec.refTypes.tids[i] = tid
	}
	dec.refAnyLens.lens = make([]int, len(rb.AnyLengths))
	for i, anyLen := range rb.AnyLengths {
		dec.refAnyLens.lens[i] = anyLen
	}
	tt, lenHint, flag, err := dec.setupType(rb.Type, nil)
	if err != nil {
		panic(err) // TODO(toddw): Change this to not panic.
	}
	dec.stack = append(dec.stack, decStackEntry{
		Type:    tt,
		Index:   -1,
		LenHint: lenHint,
		Flag:    flag,
	})
	dec.flag = dec.flag.Set(decFlagIgnoreNextStartValue)
	return dec
}

func (rb *RawBytes) VDLIsZero() bool {
	return rb == nil || (rb.Type == vdl.AnyType && rb.IsNil())
}

// TODO(toddw) This is slow - we should fix this.
func (rb *RawBytes) VDLEqual(x interface{}) bool {
	orb, ok := x.(*RawBytes)
	if !ok {
		return false
	}
	return vdl.EqualValue(vdl.ValueOf(rb), vdl.ValueOf(orb))
}

func (rb *RawBytes) VDLRead(dec vdl.Decoder) error {
	// Fastpath: the bytes are already available in the decoder81.  Note that this
	// also handles the case where dec is RawBytes.Decoder().
	if d, ok := dec.(*decoder81); ok {
		return d.readRawBytes(rb)
	}
	// Slowpath: the bytes are not available, we must encode new bytes.
	var buf bytes.Buffer
	enc := newEncoderForRawBytes(&buf)
	if err := vdl.Transcode(enc, dec); err != nil {
		return err
	}
	// Fill in rb with the results of the transcoding, captured in enc.
	rb.Version = enc.version
	rb.Type = enc.msgType
	if enc.tids == nil || len(enc.tids.tids) == 0 {
		rb.RefTypes = nil
	} else {
		tids, idToType := enc.tids.tids, enc.typeEnc.makeIdToTypeUnlocked()
		rb.RefTypes = make([]*vdl.Type, len(tids))
		for i, tid := range tids {
			tt := bootstrapIdToType[tid]
			if tt == nil {
				if tt = idToType[tid]; tt == nil {
					return fmt.Errorf("vom: internal error, type id %d in %v doesn't exist in %v", i, tids, idToType)
				}
			}
			rb.RefTypes[i] = tt
		}
	}
	if enc.anyLens == nil || len(enc.anyLens.lens) == 0 {
		rb.AnyLengths = nil
	} else {
		rb.AnyLengths = enc.anyLens.lens
	}
	rb.Data = buf.Bytes()
	return nil
}

func (rb *RawBytes) VDLWrite(enc vdl.Encoder) error {
	// Fastpath: we're trying to encode into an encoder81.  We can only write
	// directly if the versions are the same and if the encoder hasn't written any
	// other values yet, since otherwise rb.Data needs to be re-written to account
	// for the differences.
	//
	// TODO(toddw): Code a variant that performs the re-writing.
	if e, ok := enc.(*encoder81); ok && e.version == rb.Version && len(e.stack) == 0 {
		return e.writeRawBytes(rb)
	}
	// Slowpath: decodes bytes from rb and fill in enc.
	return vdl.Transcode(enc, rb.Decoder())
}
