// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vom

import (
	"math"
	"reflect"
	"unsafe"

	"v.io/v23/vdl"
	"v.io/v23/verror"
)

// Binary encoding and decoding routines.

const pkgPath = "v.io/v23/vom"

var (
	errInvalid                = verror.Register(pkgPath+".errInvalid", verror.NoRetry, "{1:}{2:} vom: invalid encoding{:_}")
	errInvalidLenOrControl    = verror.Register(pkgPath+".errInvalidLenOrControl", verror.NoRetry, "{1:}{2:} vom: invalid len or control byte {3}{:_}")
	errMsgLen                 = verror.Register(pkgPath+".errMsgLen", verror.NoRetry, "{1:}{2:} vom: message larger than {3} bytes{:_}")
	errUintOverflow           = verror.Register(pkgPath+".errUintOverflow", verror.NoRetry, "{1:}{2:} vom: scalar larger than 8 bytes{:_}")
	errBadControlCode         = verror.Register(pkgPath+".errBadControlCode", verror.NoRetry, "{1:}{2:} invalid control code{:_}")
	errBadVersionByte         = verror.Register(pkgPath+".errBadVersionByte", verror.NoRetry, "{1:}{2:} bad version byte {3}")
	errEndedBeforeVersionByte = verror.Register(pkgPath+".errEndedBeforeVersionByte", verror.NoRetry, "{1:}{2:} ended before version byte received {:_}")
)

const (
	uint64Size          = 8
	maxEncodedUintBytes = uint64Size + 1 // +1 for length byte
	maxBinaryMsgLen     = 1 << 30        // 1GiB limit to each message
)

// lenUint retuns the number of bytes used to represent the provided
// uint value
func lenUint(v uint64) int {
	switch {
	case v <= 0x7f:
		return 1
	case v <= 0xff:
		return 2
	case v <= 0xffff:
		return 3
	case v <= 0xffffff:
		return 4
	case v <= 0xffffffff:
		return 5
	case v <= 0xffffffffff:
		return 6
	case v <= 0xffffffffffff:
		return 7
	case v <= 0xffffffffffffff:
		return 8
	default:
		return 9
	}
}

func binaryEncodeControl(buf *encbuf, v byte) {
	if v < 0x80 || v > 0xef {
		panic(verror.New(errBadControlCode, nil, v))
	}
	buf.WriteOneByte(v)
}

// binaryDecodeControlOnly only decodes and advances the read position if the
// next byte is a control byte.  Returns an error if the control byte doesn't
// match want.
func binaryDecodeControlOnly(buf *decbuf, want byte) (bool, error) {
	if !buf.IsAvailable(1) {
		if err := buf.Fill(1); err != nil {
			return false, err
		}
	}
	ctrl := buf.PeekAvailableByte()
	if ctrl < 0x80 || ctrl > 0xef {
		return false, nil // not a control byte
	}
	if ctrl != want {
		return false, verror.New(errBadControlCode, nil, ctrl)
	}
	buf.SkipAvailable(1)
	return true, nil
}

// binaryPeekControl returns the next byte as a control byte, or 0 if it is not
// a control byte.  Doesn't advance the read position.
func binaryPeekControl(buf *decbuf) (byte, error) {
	if !buf.IsAvailable(1) {
		if err := buf.Fill(1); err != nil {
			return 0, err
		}
	}
	ctrl := buf.PeekAvailableByte()
	if ctrl < 0x80 || ctrl > 0xef {
		return 0, nil
	}
	return ctrl, nil
}

// Bools are encoded as a byte where 0 = false and anything else is true.
func binaryEncodeBool(buf *encbuf, v bool) {
	if v {
		buf.WriteOneByte(1)
	} else {
		buf.WriteOneByte(0)
	}
}

func binaryDecodeBool(buf *decbuf) (bool, error) {
	if !buf.IsAvailable(1) {
		if err := buf.Fill(1); err != nil {
			return false, err
		}
	}
	value := buf.ReadAvailableByte()
	if value > 1 {
		return false, verror.New(errInvalid, nil) // TODO: better error
	}
	return value != 0, nil
}

// Unsigned integers are the basis for all other primitive values.  This is a
// two-state encoding.  If the number is less than 128 (0 through 0x7f), its
// value is written directly.  Otherwise the value is written in big-endian byte
// order preceded by the negated byte length.
func binaryEncodeUint(buf *encbuf, v uint64) {
	switch {
	case v <= 0x7f:
		buf.Grow(1)[0] = byte(v)
	case v <= 0xff:
		buf := buf.Grow(2)
		buf[0] = 0xff
		buf[1] = byte(v)
	case v <= 0xffff:
		buf := buf.Grow(3)
		buf[0] = 0xfe
		buf[1] = byte(v >> 8)
		buf[2] = byte(v)
	case v <= 0xffffff:
		buf := buf.Grow(4)
		buf[0] = 0xfd
		buf[1] = byte(v >> 16)
		buf[2] = byte(v >> 8)
		buf[3] = byte(v)
	case v <= 0xffffffff:
		buf := buf.Grow(5)
		buf[0] = 0xfc
		buf[1] = byte(v >> 24)
		buf[2] = byte(v >> 16)
		buf[3] = byte(v >> 8)
		buf[4] = byte(v)
	case v <= 0xffffffffff:
		buf := buf.Grow(6)
		buf[0] = 0xfb
		buf[1] = byte(v >> 32)
		buf[2] = byte(v >> 24)
		buf[3] = byte(v >> 16)
		buf[4] = byte(v >> 8)
		buf[5] = byte(v)
	case v <= 0xffffffffffff:
		buf := buf.Grow(7)
		buf[0] = 0xfa
		buf[1] = byte(v >> 40)
		buf[2] = byte(v >> 32)
		buf[3] = byte(v >> 24)
		buf[4] = byte(v >> 16)
		buf[5] = byte(v >> 8)
		buf[6] = byte(v)
	case v <= 0xffffffffffffff:
		buf := buf.Grow(8)
		buf[0] = 0xf9
		buf[1] = byte(v >> 48)
		buf[2] = byte(v >> 40)
		buf[3] = byte(v >> 32)
		buf[4] = byte(v >> 24)
		buf[5] = byte(v >> 16)
		buf[6] = byte(v >> 8)
		buf[7] = byte(v)
	default:
		buf := buf.Grow(9)
		buf[0] = 0xf8
		buf[1] = byte(v >> 56)
		buf[2] = byte(v >> 48)
		buf[3] = byte(v >> 40)
		buf[4] = byte(v >> 32)
		buf[5] = byte(v >> 24)
		buf[6] = byte(v >> 16)
		buf[7] = byte(v >> 8)
		buf[8] = byte(v)
	}
}

func binaryDecodeUint(buf *decbuf) (uint64, error) {
	if !buf.IsAvailable(1) {
		if err := buf.Fill(1); err != nil {
			return 0, err
		}
	}
	firstByte := buf.ReadAvailableByte()
	// Handle single-byte encoding.
	if firstByte <= 0x7f {
		return uint64(firstByte), nil
	}
	// Handle multi-byte encoding.
	byteLen := int(-int8(firstByte))
	if byteLen < 1 || byteLen > uint64Size {
		return 0, verror.New(errInvalidLenOrControl, nil, firstByte)
	}
	if !buf.IsAvailable(byteLen) {
		if err := buf.Fill(byteLen); err != nil {
			return 0, err
		}
	}
	bytes := buf.ReadAvailable(byteLen)
	var uvalue uint64
	for _, b := range bytes {
		uvalue = uvalue<<8 | uint64(b)
	}
	return uvalue, nil
}

func binaryPeekUint(buf *decbuf) (uint64, int, error) {
	if !buf.IsAvailable(1) {
		if err := buf.Fill(1); err != nil {
			return 0, 0, err
		}
	}
	firstByte := buf.PeekAvailableByte()
	// Handle single-byte encoding.
	if firstByte <= 0x7f {
		return uint64(firstByte), 1, nil
	}
	// Handle multi-byte encoding.
	byteLen := int(-int8(firstByte))
	if byteLen < 1 || byteLen > uint64Size {
		return 0, 0, verror.New(errInvalidLenOrControl, nil, firstByte)
	}
	byteLen++ // account for initial len byte
	if !buf.IsAvailable(byteLen) {
		if err := buf.Fill(byteLen); err != nil {
			return 0, 0, err
		}
	}
	bytes := buf.PeekAvailable(byteLen)
	var uvalue uint64
	for _, b := range bytes[1:] {
		uvalue = uvalue<<8 | uint64(b)
	}
	return uvalue, byteLen, nil
}

func binarySkipUint(buf *decbuf) error {
	if !buf.IsAvailable(1) {
		if err := buf.Fill(1); err != nil {
			return err
		}
	}
	firstByte := buf.PeekAvailableByte()
	// Handle single-byte encoding.
	if firstByte <= 0x7f {
		buf.SkipAvailable(1)
		return nil
	}
	// Handle multi-byte encoding.
	byteLen := int(-int8(firstByte))
	if byteLen < 1 || byteLen > uint64Size {
		return verror.New(errInvalidLenOrControl, nil, firstByte)
	}
	byteLen++ // account for initial len byte
	if !buf.IsAvailable(byteLen) {
		if err := buf.Fill(byteLen); err != nil {
			return err
		}
	}
	buf.SkipAvailable(byteLen)
	return nil
}

func binaryPeekUintByteLen(buf *decbuf) (int, error) {
	if !buf.IsAvailable(1) {
		if err := buf.Fill(1); err != nil {
			return 0, err
		}
	}
	firstByte := buf.PeekAvailableByte()
	// Handle single-byte encoding.
	if firstByte <= 0x7f {
		return 1, nil
	}
	// Handle multi-byte encoding.
	byteLen := int(-int8(firstByte))
	if byteLen < 1 || byteLen > uint64Size {
		return 0, verror.New(errInvalidLenOrControl, nil, firstByte)
	}
	return 1 + byteLen, nil
}

func binaryDecodeLen(buf *decbuf) (int, error) {
	ulen, err := binaryDecodeUint(buf)
	switch {
	case err != nil:
		return 0, err
	case ulen > maxBinaryMsgLen:
		return 0, verror.New(errMsgLen, nil, maxBinaryMsgLen)
	}
	return int(ulen), nil
}

func binaryDecodeLenOrArrayLen(buf *decbuf, t *vdl.Type) (int, error) {
	len, err := binaryDecodeLen(buf)
	if err != nil {
		return 0, err
	}
	if t.Kind() == vdl.Array {
		if len != 0 {
			return 0, verror.New(errInvalid, nil) // TODO(toddw): better error
		}
		return t.Len(), nil
	}
	return len, nil
}

// Signed integers are encoded as unsigned integers, where the low bit says
// whether to complement the other bits to recover the int.
func binaryEncodeInt(buf *encbuf, v int64) {
	var uvalue uint64
	if v < 0 {
		uvalue = uint64(^v<<1) | 1
	} else {
		uvalue = uint64(v << 1)
	}
	binaryEncodeUint(buf, uvalue)
}

func binaryDecodeInt(buf *decbuf) (int64, error) {
	if !buf.IsAvailable(1) {
		if err := buf.Fill(1); err != nil {
			return 0, err
		}
	}
	firstByte := buf.ReadAvailableByte()
	// Handle single-byte encoding.
	if firstByte <= 0x7f {
		if firstByte&1 == 1 {
			return ^int64(firstByte >> 1), nil
		}
		return int64(firstByte >> 1), nil
	}
	// Handle multi-byte encoding.
	byteLen := int(-int8(firstByte))
	if byteLen < 1 || byteLen > uint64Size {
		return 0, verror.New(errInvalidLenOrControl, nil, firstByte)
	}
	if !buf.IsAvailable(byteLen) {
		if err := buf.Fill(byteLen); err != nil {
			return 0, err
		}
	}
	bytes := buf.ReadAvailable(byteLen)
	var uvalue uint64
	for _, b := range bytes {
		uvalue = uvalue<<8 | uint64(b)
	}
	if uvalue&1 == 1 {
		return ^int64(uvalue >> 1), nil
	}
	return int64(uvalue >> 1), nil
}

func binaryPeekInt(buf *decbuf) (int64, int, error) {
	if !buf.IsAvailable(1) {
		if err := buf.Fill(1); err != nil {
			return 0, 0, err
		}
	}
	firstByte := buf.PeekAvailableByte()
	// Handle single-byte encoding.
	if firstByte <= 0x7f {
		if firstByte&1 == 1 {
			return ^int64(firstByte >> 1), 1, nil
		}
		return int64(firstByte >> 1), 1, nil
	}
	// Handle multi-byte encoding.
	byteLen := int(-int8(firstByte))
	if byteLen < 1 || byteLen > uint64Size {
		return 0, 0, verror.New(errInvalidLenOrControl, nil, firstByte)
	}
	byteLen++ // account for initial len byte
	if !buf.IsAvailable(byteLen) {
		if err := buf.Fill(byteLen); err != nil {
			return 0, 0, err
		}
	}
	bytes := buf.PeekAvailable(byteLen)
	var uvalue uint64
	for _, b := range bytes[1:] {
		uvalue = uvalue<<8 | uint64(b)
	}
	if uvalue&1 == 1 {
		return ^int64(uvalue >> 1), byteLen, nil
	}
	return int64(uvalue >> 1), byteLen, nil
}

// Floating point numbers are encoded as byte-reversed ieee754.
func binaryEncodeFloat(buf *encbuf, v float64) {
	ieee := math.Float64bits(v)
	// Manually-unrolled byte-reversing.
	uvalue := (ieee&0xff)<<56 |
		(ieee&0xff00)<<40 |
		(ieee&0xff0000)<<24 |
		(ieee&0xff000000)<<8 |
		(ieee&0xff00000000)>>8 |
		(ieee&0xff0000000000)>>24 |
		(ieee&0xff000000000000)>>40 |
		(ieee&0xff00000000000000)>>56
	binaryEncodeUint(buf, uvalue)
}

func binaryDecodeFloat(buf *decbuf) (float64, error) {
	uvalue, err := binaryDecodeUint(buf)
	if err != nil {
		return 0, err
	}
	// Manually-unrolled byte-reversing.
	ieee := (uvalue&0xff)<<56 |
		(uvalue&0xff00)<<40 |
		(uvalue&0xff0000)<<24 |
		(uvalue&0xff000000)<<8 |
		(uvalue&0xff00000000)>>8 |
		(uvalue&0xff0000000000)>>24 |
		(uvalue&0xff000000000000)>>40 |
		(uvalue&0xff00000000000000)>>56
	return math.Float64frombits(ieee), nil
}

// Strings are encoded as the byte count followed by uninterpreted bytes.
func binaryEncodeString(buf *encbuf, s string) {
	binaryEncodeUint(buf, uint64(len(s)))
	buf.WriteString(s)
}

func binaryDecodeString(buf *decbuf) (string, error) {
	len, err := binaryDecodeLen(buf)
	if len == 0 || err != nil {
		return "", err
	}
	data := make([]byte, len)
	if err := buf.ReadIntoBuf(data); err != nil {
		return "", err
	}
	// Go makes an extra copy if we simply perform the conversion string(data), so
	// we use unsafe to transfer the contents from data into s without a copy.
	s := ""
	p := (*reflect.StringHeader)(unsafe.Pointer(&s))
	p.Data = uintptr(unsafe.Pointer(&data[0]))
	p.Len = len
	return s, nil
}

func binarySkipString(buf *decbuf) error {
	len, err := binaryDecodeLen(buf)
	if err != nil {
		return err
	}
	return buf.Skip(len)
}

// binaryEncodeUintEnd writes into the trailing part of buf and returns the start
// index of the encoded data.
//
// REQUIRES: buf is big enough to hold the encoded value.
func binaryEncodeUintEnd(buf []byte, v uint64) int {
	end := len(buf) - 1
	switch {
	case v <= 0x7f:
		buf[end] = byte(v)
		return end
	case v <= 0xff:
		buf[end-1] = 0xff
		buf[end] = byte(v)
		return end - 1
	case v <= 0xffff:
		buf[end-2] = 0xfe
		buf[end-1] = byte(v >> 8)
		buf[end] = byte(v)
		return end - 2
	case v <= 0xffffff:
		buf[end-3] = 0xfd
		buf[end-2] = byte(v >> 16)
		buf[end-1] = byte(v >> 8)
		buf[end] = byte(v)
		return end - 3
	case v <= 0xffffffff:
		buf[end-4] = 0xfc
		buf[end-3] = byte(v >> 24)
		buf[end-2] = byte(v >> 16)
		buf[end-1] = byte(v >> 8)
		buf[end] = byte(v)
		return end - 4
	case v <= 0xffffffffff:
		buf[end-5] = 0xfb
		buf[end-4] = byte(v >> 32)
		buf[end-3] = byte(v >> 24)
		buf[end-2] = byte(v >> 16)
		buf[end-1] = byte(v >> 8)
		buf[end] = byte(v)
		return end - 5
	case v <= 0xffffffffffff:
		buf[end-6] = 0xfa
		buf[end-5] = byte(v >> 40)
		buf[end-4] = byte(v >> 32)
		buf[end-3] = byte(v >> 24)
		buf[end-2] = byte(v >> 16)
		buf[end-1] = byte(v >> 8)
		buf[end] = byte(v)
		return end - 6
	case v <= 0xffffffffffffff:
		buf[end-7] = 0xf9
		buf[end-6] = byte(v >> 48)
		buf[end-5] = byte(v >> 40)
		buf[end-4] = byte(v >> 32)
		buf[end-3] = byte(v >> 24)
		buf[end-2] = byte(v >> 16)
		buf[end-1] = byte(v >> 8)
		buf[end] = byte(v)
		return end - 7
	default:
		buf[end-8] = 0xf8
		buf[end-7] = byte(v >> 56)
		buf[end-6] = byte(v >> 48)
		buf[end-5] = byte(v >> 40)
		buf[end-4] = byte(v >> 32)
		buf[end-3] = byte(v >> 24)
		buf[end-2] = byte(v >> 16)
		buf[end-1] = byte(v >> 8)
		buf[end] = byte(v)
		return end - 8
	}
}

// binaryEncodeIntEnd writes into the trailing part of buf and returns the start
// index of the encoded data.
//
// REQUIRES: buf is big enough to hold the encoded value.
func binaryEncodeIntEnd(buf []byte, v int64) int {
	var uvalue uint64
	if v < 0 {
		uvalue = uint64(^v<<1) | 1
	} else {
		uvalue = uint64(v << 1)
	}
	return binaryEncodeUintEnd(buf, uvalue)
}
