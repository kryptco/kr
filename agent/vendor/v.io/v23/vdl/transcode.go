// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

// Transcode transcodes from the decoder to the encoder.
func Transcode(e Encoder, d Decoder) error {
	if err := d.StartValue(AnyType); err != nil {
		return err
	}
	if d.IsNil() {
		if err := e.NilValue(d.Type()); err != nil {
			return err
		}
		return d.FinishValue()
	}
	if d.IsOptional() {
		e.SetNextStartValueIsOptional()
	}
	if err := e.StartValue(d.Type()); err != nil {
		return err
	}
	if err := transcodeNonNilValue(e, d); err != nil {
		return err
	}
	if err := d.FinishValue(); err != nil {
		return err
	}
	return e.FinishValue()
}

func transcodeNonNilValue(e Encoder, d Decoder) error {
	if d.Type().IsBytes() {
		var b []byte
		fixedLen := -1
		if d.Type().Kind() == Array {
			fixedLen = d.Type().Len()
			b = make([]byte, fixedLen)
		}
		if err := d.DecodeBytes(fixedLen, &b); err != nil {
			return err
		}
		return e.EncodeBytes(b)
	}
	switch d.Type().Kind() {
	case Bool:
		val, err := d.DecodeBool()
		if err != nil {
			return err
		}
		return e.EncodeBool(val)
	case Byte, Uint16, Uint32, Uint64:
		val, err := d.DecodeUint(d.Type().Kind().BitLen())
		if err != nil {
			return err
		}
		return e.EncodeUint(val)
	case Int8, Int16, Int32, Int64:
		val, err := d.DecodeInt(d.Type().Kind().BitLen())
		if err != nil {
			return err
		}
		return e.EncodeInt(val)
	case Float32, Float64:
		val, err := d.DecodeFloat(d.Type().Kind().BitLen())
		if err != nil {
			return err
		}
		return e.EncodeFloat(val)
	case String, Enum:
		val, err := d.DecodeString()
		if err != nil {
			return err
		}
		return e.EncodeString(val)
	case TypeObject:
		val, err := d.DecodeTypeObject()
		if err != nil {
			return err
		}
		return e.EncodeTypeObject(val)
	case List, Array, Set:
		return transcodeListOrArrayOrSet(e, d)
	case Map:
		return transcodeMap(e, d)
	case Struct, Union:
		return transcodeStructOrUnion(e, d)
	}
	panic("unhandled kind")
}

func transcodeListOrArrayOrSet(e Encoder, d Decoder) error {
	if err := e.SetLenHint(d.LenHint()); err != nil {
		return err
	}
	for {
		switch done, err := d.NextEntry(); {
		case err != nil:
			return err
		case done:
			return e.NextEntry(true)
		default:
			if err := e.NextEntry(false); err != nil {
				return err
			}
			if err := Transcode(e, d); err != nil {
				return err
			}
		}
	}
}

func transcodeMap(e Encoder, d Decoder) error {
	if err := e.SetLenHint(d.LenHint()); err != nil {
		return err
	}
	for {
		switch done, err := d.NextEntry(); {
		case err != nil:
			return err
		case done:
			return e.NextEntry(true)
		default:
			if err := e.NextEntry(false); err != nil {
				return err
			}
			if err := Transcode(e, d); err != nil {
				return err
			}
			if err := Transcode(e, d); err != nil {
				return err
			}
		}
	}
}

func transcodeStructOrUnion(e Encoder, d Decoder) error {
	for {
		switch index, err := d.NextField(); {
		case err != nil:
			return err
		case index == -1:
			return e.NextField(-1)
		default:
			if err := e.NextField(index); err != nil {
				return err
			}
			if err := Transcode(e, d); err != nil {
				return err
			}
		}
	}
}
