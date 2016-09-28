// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdl

// The "fast" Decoder ReadValue* and NextEntryValue* methods aren't actually
// fast, they just call the appropriate methods in sequence.

func (d *pipeDecoder) ReadValueBool() (bool, error) {
	if err := d.StartValue(BoolType); err != nil {
		return false, err
	}
	value, err := d.DecodeBool()
	if err != nil {
		return false, err
	}
	return value, d.FinishValue()
}

func (d *pipeDecoder) ReadValueString() (string, error) {
	if err := d.StartValue(StringType); err != nil {
		return "", err
	}
	value, err := d.DecodeString()
	if err != nil {
		return "", err
	}
	return value, d.FinishValue()
}

func (d *pipeDecoder) ReadValueUint(bitlen int) (uint64, error) {
	if err := d.StartValue(Uint64Type); err != nil {
		return 0, err
	}
	value, err := d.DecodeUint(bitlen)
	if err != nil {
		return 0, err
	}
	return value, d.FinishValue()
}

func (d *pipeDecoder) ReadValueInt(bitlen int) (int64, error) {
	if err := d.StartValue(Int64Type); err != nil {
		return 0, err
	}
	value, err := d.DecodeInt(bitlen)
	if err != nil {
		return 0, err
	}
	return value, d.FinishValue()
}

func (d *pipeDecoder) ReadValueFloat(bitlen int) (float64, error) {
	if err := d.StartValue(Float64Type); err != nil {
		return 0, err
	}
	value, err := d.DecodeFloat(bitlen)
	if err != nil {
		return 0, err
	}
	return value, d.FinishValue()
}

func (d *pipeDecoder) ReadValueTypeObject() (*Type, error) {
	if err := d.StartValue(TypeObjectType); err != nil {
		return nil, err
	}
	value, err := d.DecodeTypeObject()
	if err != nil {
		return nil, err
	}
	return value, d.FinishValue()
}

func (d *pipeDecoder) ReadValueBytes(fixedLen int, x *[]byte) error {
	if err := d.StartValue(ttByteList); err != nil {
		return err
	}
	if err := d.DecodeBytes(fixedLen, x); err != nil {
		return err
	}
	return d.FinishValue()
}

func (d *pipeDecoder) NextEntryValueBool() (done bool, _ bool, _ error) {
	if done, err := d.NextEntry(); done || err != nil {
		return done, false, err
	}
	value, err := d.ReadValueBool()
	return false, value, err
}

func (d *pipeDecoder) NextEntryValueString() (done bool, _ string, _ error) {
	if done, err := d.NextEntry(); done || err != nil {
		return done, "", err
	}
	value, err := d.ReadValueString()
	return false, value, err
}

func (d *pipeDecoder) NextEntryValueUint(bitlen int) (done bool, _ uint64, _ error) {
	if done, err := d.NextEntry(); done || err != nil {
		return done, 0, err
	}
	value, err := d.ReadValueUint(bitlen)
	return false, value, err
}

func (d *pipeDecoder) NextEntryValueInt(bitlen int) (done bool, _ int64, _ error) {
	if done, err := d.NextEntry(); done || err != nil {
		return done, 0, err
	}
	value, err := d.ReadValueInt(bitlen)
	return false, value, err
}

func (d *pipeDecoder) NextEntryValueFloat(bitlen int) (done bool, _ float64, _ error) {
	if done, err := d.NextEntry(); done || err != nil {
		return done, 0, err
	}
	value, err := d.ReadValueFloat(bitlen)
	return false, value, err
}

func (d *pipeDecoder) NextEntryValueTypeObject() (done bool, _ *Type, _ error) {
	if done, err := d.NextEntry(); done || err != nil {
		return done, nil, err
	}
	value, err := d.ReadValueTypeObject()
	return false, value, err
}

// The "fast" Encoder WriteValue*, NextEntryValue* and NextFieldValue* methods
// aren't actually fast, they just call the appropriate methods in sequence.

func (e *pipeEncoder) WriteValueBool(tt *Type, value bool) error {
	if err := e.StartValue(tt); err != nil {
		return err
	}
	if err := e.EncodeBool(value); err != nil {
		return err
	}
	return e.FinishValue()
}

func (e *pipeEncoder) WriteValueString(tt *Type, value string) error {
	if err := e.StartValue(tt); err != nil {
		return err
	}
	if err := e.EncodeString(value); err != nil {
		return err
	}
	return e.FinishValue()
}

func (e *pipeEncoder) WriteValueUint(tt *Type, value uint64) error {
	if err := e.StartValue(tt); err != nil {
		return err
	}
	if err := e.EncodeUint(value); err != nil {
		return err
	}
	return e.FinishValue()
}

func (e *pipeEncoder) WriteValueInt(tt *Type, value int64) error {
	if err := e.StartValue(tt); err != nil {
		return err
	}
	if err := e.EncodeInt(value); err != nil {
		return err
	}
	return e.FinishValue()
}

func (e *pipeEncoder) WriteValueFloat(tt *Type, value float64) error {
	if err := e.StartValue(tt); err != nil {
		return err
	}
	if err := e.EncodeFloat(value); err != nil {
		return err
	}
	return e.FinishValue()
}

func (e *pipeEncoder) WriteValueTypeObject(value *Type) error {
	if err := e.StartValue(TypeObjectType); err != nil {
		return err
	}
	if err := e.EncodeTypeObject(value); err != nil {
		return err
	}
	return e.FinishValue()
}

func (e *pipeEncoder) WriteValueBytes(tt *Type, value []byte) error {
	if err := e.StartValue(tt); err != nil {
		return err
	}
	if err := e.EncodeBytes(value); err != nil {
		return err
	}
	return e.FinishValue()
}

func (e *pipeEncoder) NextEntryValueBool(tt *Type, value bool) error {
	if err := e.NextEntry(false); err != nil {
		return err
	}
	return e.WriteValueBool(tt, value)
}

func (e *pipeEncoder) NextEntryValueString(tt *Type, value string) error {
	if err := e.NextEntry(false); err != nil {
		return err
	}
	return e.WriteValueString(tt, value)
}

func (e *pipeEncoder) NextEntryValueUint(tt *Type, value uint64) error {
	if err := e.NextEntry(false); err != nil {
		return err
	}
	return e.WriteValueUint(tt, value)
}

func (e *pipeEncoder) NextEntryValueInt(tt *Type, value int64) error {
	if err := e.NextEntry(false); err != nil {
		return err
	}
	return e.WriteValueInt(tt, value)
}

func (e *pipeEncoder) NextEntryValueFloat(tt *Type, value float64) error {
	if err := e.NextEntry(false); err != nil {
		return err
	}
	return e.WriteValueFloat(tt, value)
}

func (e *pipeEncoder) NextEntryValueTypeObject(value *Type) error {
	if err := e.NextEntry(false); err != nil {
		return err
	}
	return e.WriteValueTypeObject(value)
}

func (e *pipeEncoder) NextEntryValueBytes(tt *Type, value []byte) error {
	if err := e.NextEntry(false); err != nil {
		return err
	}
	return e.WriteValueBytes(tt, value)
}

func (e *pipeEncoder) NextFieldValueBool(index int, tt *Type, value bool) error {
	if err := e.NextField(index); err != nil {
		return err
	}
	return e.WriteValueBool(tt, value)
}

func (e *pipeEncoder) NextFieldValueString(index int, tt *Type, value string) error {
	if err := e.NextField(index); err != nil {
		return err
	}
	return e.WriteValueString(tt, value)
}

func (e *pipeEncoder) NextFieldValueUint(index int, tt *Type, value uint64) error {
	if err := e.NextField(index); err != nil {
		return err
	}
	return e.WriteValueUint(tt, value)
}

func (e *pipeEncoder) NextFieldValueInt(index int, tt *Type, value int64) error {
	if err := e.NextField(index); err != nil {
		return err
	}
	return e.WriteValueInt(tt, value)
}

func (e *pipeEncoder) NextFieldValueFloat(index int, tt *Type, value float64) error {
	if err := e.NextField(index); err != nil {
		return err
	}
	return e.WriteValueFloat(tt, value)
}

func (e *pipeEncoder) NextFieldValueTypeObject(index int, value *Type) error {
	if err := e.NextField(index); err != nil {
		return err
	}
	return e.WriteValueTypeObject(value)
}

func (e *pipeEncoder) NextFieldValueBytes(index int, tt *Type, value []byte) error {
	if err := e.NextField(index); err != nil {
		return err
	}
	return e.WriteValueBytes(tt, value)
}
