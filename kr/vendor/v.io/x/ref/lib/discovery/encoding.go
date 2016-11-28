// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"v.io/x/ref/lib/security/bcrypter"
)

// EncodingBuffer is used to encode and decode advertisements.
type EncodingBuffer struct {
	buf *bytes.Buffer
}

// Write appends a byte slice to the buffer.
func (e *EncodingBuffer) Write(p []byte) {
	e.buf.Write(p)
}

// Read reads the next len(p) bytes from the buffer. If the buffer has no
// enough data, io.EOF is returned.
func (e *EncodingBuffer) Read(p []byte) error {
	n, err := e.buf.Read(p)
	if err != nil {
		return err
	}
	if n < len(p) {
		return io.EOF
	}
	return nil
}

// WriteInt appends an integer to the buffer.
func (e *EncodingBuffer) WriteInt(x int) {
	var p [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(p[:], uint64(x))
	e.buf.Write(p[0:n])
}

// ReadInt reads an integer from the buffer.
func (e *EncodingBuffer) ReadInt() (int, error) {
	x, err := binary.ReadUvarint(e.buf)
	return int(x), err
}

// WriteBytes appends a byte slice to the buffer with its length.
func (e *EncodingBuffer) WriteBytes(p []byte) {
	e.WriteInt(len(p))
	e.buf.Write(p)
}

// ReadBytes reads a byte slice with its length from the buffer.
func (e *EncodingBuffer) ReadBytes() ([]byte, error) {
	n, err := e.ReadInt()
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, fmt.Errorf("invalid length %d", n)
	}
	p := e.buf.Next(n)
	if len(p) < n {
		return nil, io.EOF
	}
	return p, nil
}

// WriteString appends a string to the buffer.
func (e *EncodingBuffer) WriteString(s string) {
	e.WriteInt(len(s))
	e.buf.WriteString(s)
}

// ReadString reads a string from the buffer.
func (e *EncodingBuffer) ReadString() (string, error) {
	p, err := e.ReadBytes()
	if err != nil {
		return "", err
	}
	return string(p), nil
}

// Len returns the number of bytes of the unread portion of the buffer.
func (e *EncodingBuffer) Len() int {
	return e.buf.Len()
}

// Bytes returns a byte slice holding the unread portion of the buffer.
func (e *EncodingBuffer) Bytes() []byte {
	return e.buf.Bytes()
}

// Next returns a slice containing the next n bytes from the buffer. If there
// are fewer than n bytes in the buffer, it returns the entire buffer.
func (e *EncodingBuffer) Next(n int) []byte {
	return e.buf.Next(n)
}

// NewEncodingBuffer returns a new encoding buffer.
func NewEncodingBuffer(data []byte) *EncodingBuffer { return &EncodingBuffer{bytes.NewBuffer(data)} }

const (
	addrsUncompressed = 0
	addrsCompressed   = 1
)

// PackAddresses packs addresses into a byte slice.
func PackAddresses(addrs []string) []byte {
	if len(addrs) == 0 {
		return nil
	}
	// Format: Trailing byte lists format (addrsCompressed or addrsUncompressed)
	uncompressed := packAddressesUncompressed(addrs)
	// Try to compress, if it fails or is too large, return uncompressed.
	compressed := new(bytes.Buffer)
	if w, err := flate.NewWriter(compressed, flate.DefaultCompression); err == nil {
		if _, err := w.Write(uncompressed); err == nil && w.Close() == nil && compressed.Len() < len(uncompressed) {
			return append(compressed.Bytes(), addrsCompressed)
		}
	}
	return append(uncompressed, addrsUncompressed)
}

// UnpackAddresses unpacks addresses from a byte slice.
func UnpackAddresses(data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var uncompressed []byte
	switch footer := data[len(data)-1]; footer {
	case addrsUncompressed:
		uncompressed = data[:len(data)-1]
	case addrsCompressed:
		r := flate.NewReader(bytes.NewBuffer(data[:len(data)-1]))
		var err error
		if uncompressed, err = ioutil.ReadAll(r); err != nil {
			return nil, err
		}
		if err := r.Close(); err != nil {
			return nil, err
		}
	}
	return unpackAddressesUncompressed(uncompressed)
}

func packAddressesUncompressed(addrs []string) []byte {
	buf := NewEncodingBuffer(nil)
	for _, a := range addrs {
		buf.WriteString(a)
	}
	return buf.Bytes()
}

func unpackAddressesUncompressed(data []byte) ([]string, error) {
	buf := NewEncodingBuffer(data)
	var addrs []string
	for buf.Len() > 0 {
		addr, err := buf.ReadString()
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}

// PackEncryptionKeys packs encryption algorithm and keys into a byte slice.
func PackEncryptionKeys(algo EncryptionAlgorithm, keys []EncryptionKey) []byte {
	buf := NewEncodingBuffer(nil)
	buf.WriteInt(int(algo))
	for _, k := range keys {
		buf.WriteBytes(k)
	}
	return buf.Bytes()
}

// UnpackEncryptionKeys unpacks encryption algorithm and keys from a byte slice.
func UnpackEncryptionKeys(data []byte) (EncryptionAlgorithm, []EncryptionKey, error) {
	buf := NewEncodingBuffer(data)
	algo, err := buf.ReadInt()
	if err != nil {
		return NoEncryption, nil, err
	}
	var keys []EncryptionKey
	for buf.Len() > 0 {
		key, err := buf.ReadBytes()
		if err != nil {
			return NoEncryption, nil, err
		}
		keys = append(keys, EncryptionKey(key))
	}
	return EncryptionAlgorithm(algo), keys, nil
}

// EncodeCiphertext encodes the cipher text into a byte slice.
func EncodeWireCiphertext(wctext *bcrypter.WireCiphertext) []byte {
	buf := NewEncodingBuffer(nil)
	buf.WriteString(wctext.PatternId)
	for k, v := range wctext.Bytes {
		buf.WriteString(k)
		buf.WriteBytes(v)
	}
	return buf.Bytes()
}

// DecodeCiphertext decodes the cipher text from a byte slice.
func DecodeWireCiphertext(data []byte) (*bcrypter.WireCiphertext, error) {
	buf := NewEncodingBuffer(data)
	id, err := buf.ReadString()
	if err != nil {
		return nil, err
	}
	wctext := bcrypter.WireCiphertext{id, make(map[string][]byte)}
	for buf.Len() > 0 {
		k, err := buf.ReadString()
		if err != nil {
			return nil, err
		}
		v, err := buf.ReadBytes()
		if err != nil {
			return nil, err
		}
		wctext.Bytes[k] = v
	}
	return &wctext, nil
}

// EncodeTimestamp encodes the timestamp into a 8-byte byte slice.
func EncodeTimestamp(ts int64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(ts))
	return buf
}

// DecodeTimestamp decodes the timestamp from a 8-byte byte slice.
func DecodeTimestamp(data []byte) (int64, error) {
	if len(data) != 8 {
		return 0, errors.New("invalid length")
	}
	return int64(binary.LittleEndian.Uint64(data)), nil
}
