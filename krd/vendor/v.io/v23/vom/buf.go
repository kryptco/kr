// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vom

import "io"

const minBufFree = 1024 // buffers always have at least 1K free after growth

// encbuf manages the write buffer for encoders.  The approach is similar to
// bytes.Buffer, but the implementation is simplified to only deal with many
// writes followed by a read of the whole buffer.
type encbuf struct {
	// It's faster to hold end than to use the len and cap properties of buf,
	// since end is cheaper to update than buf.
	buf []byte
	end int // [0, end) is data that's already written
}

func newEncbuf() *encbuf {
	return &encbuf{
		buf: make([]byte, minBufFree),
	}
}

// Bytes returns a slice of the bytes written so far.
func (b *encbuf) Bytes() []byte { return b.buf[:b.end] }

// Len returns the number of bytes written so far.
func (b *encbuf) Len() int { return b.end }

// Reset the length to 0 to start a new round of writes.
func (b *encbuf) Reset() { b.end = 0 }

// reserve at least min free bytes in the buffer.
func (b *encbuf) reserve(min int) {
	if len(b.buf)-b.end < min {
		newlen := len(b.buf) * 2
		if newlen-b.end < min {
			newlen = b.end + min + minBufFree
		}
		newbuf := make([]byte, newlen)
		copy(newbuf, b.buf[:b.end])
		b.buf = newbuf
	}
}

// Grow the buffer by n bytes, and returns those bytes.
//
// Different from bytes.Buffer.Grow, which doesn't return the bytes.
func (b *encbuf) Grow(n int) []byte {
	b.reserve(n)
	oldend := b.end
	b.end += n
	return b.buf[oldend:b.end]
}

// WriteOneByte writes byte x into the buffer.
func (b *encbuf) WriteOneByte(x byte) {
	b.reserve(1)
	b.buf[b.end] = x
	b.end++
}

// Write writes byte slice x to the buffer.
func (b *encbuf) Write(x []byte) {
	b.reserve(len(x))
	b.end += copy(b.buf[b.end:], x)
}

// WriteString writes string x to the buffer.
func (b *encbuf) WriteString(x string) {
	b.reserve(len(x))
	b.end += copy(b.buf[b.end:], x)
}

// decbuf manages the read buffer for decoders.  The approach is similar to
// bufio.Reader, but the API is better suited for fast decoding.
type decbuf struct {
	// It's faster to hold end than to use the len and cap properties of buf,
	// since end is cheaper to update than buf.
	buf      []byte
	beg, end int // [beg, end) is data read from reader but unread by the user

	// lim holds the number of bytes left in the limit, or if it is any negative
	// number, there is no limit.  By allowing any negative number to convey "no
	// limit", we avoid an extra conditional branch in the Read and Peek methods,
	// making things faster.  The downside is we need to worry about wraparound,
	// but the limit gets reset often enough that this doesn't matter.
	lim int

	reader  io.Reader
	version Version
}

// newDecbuf returns a new decbuf that fills its internal buffer by reading r.
func newDecbuf(r io.Reader) *decbuf {
	return &decbuf{
		buf:    make([]byte, minBufFree),
		lim:    -1,
		reader: r,
	}
}

// newDecbufFromBytes returns a new decbuf that reads directly from b.
func newDecbufFromBytes(b []byte) *decbuf {
	return &decbuf{
		buf:    b,
		end:    len(b),
		lim:    -1,
		reader: alwaysEOFReader{},
	}
}

type alwaysEOFReader struct{}

func (alwaysEOFReader) Read([]byte) (int, error) { return 0, io.EOF }

// Reset resets the buffer so it has no data.
func (b *decbuf) Reset() {
	b.beg = 0
	b.end = 0
	b.lim = -1
}

// SetLimit sets a limit to the bytes that are returned by decbuf; after a limit
// is set, subsequent reads cannot read past the limit, even if more bytes are
// available.  Attempts to read past the limit return io.EOF.  Call RemoveLimit
// to remove the limit.
//
// REQUIRES: limit >=0,
func (b *decbuf) SetLimit(limit int) {
	b.lim = limit
}

// RemoveLimit removes the limit, and returns the number of leftover bytes.
// Returns a negative number if no limit was set.
func (b *decbuf) RemoveLimit() int {
	leftover := b.lim
	b.lim = -1
	return leftover
}

// IsAvailable returns true iff at least n bytes are available to read, peek or
// skip.  Call Fill to replenish the available bytes.
//
// The code is factored into IsAvailable followed by {Read,Peek,Skip}Available,
// since each of these methods is very short and doesn't call any other
// functions, allowing them to be inlined at the call site.  This gives us a
// speedup in the common case where bytes are already available in the buffer.
func (b *decbuf) IsAvailable(n int) bool {
	return b.end-b.beg >= n && (b.lim >= n || b.lim < 0)
}

// Fill the buffer with at least min bytes of data.  Returns an error if fewer
// than min bytes could be filled.  Doesn't advance the read position.
func (b *decbuf) Fill(min int) error {
	if b.lim >= 0 && b.lim < min {
		return io.EOF
	}
	switch avail := b.end - b.beg; {
	case avail >= min:
		// Fastpath - enough bytes are available.
		return nil
	case len(b.buf) < min:
		// The buffer isn't big enough.  Make a new buffer that's big enough and
		// copy existing data to the front.
		newlen := len(b.buf) * 2
		if newlen < min+minBufFree {
			newlen = min + minBufFree
		}
		newbuf := make([]byte, newlen)
		b.end = copy(newbuf, b.buf[b.beg:b.end])
		b.beg = 0
		b.buf = newbuf
	default:
		// The buffer is big enough.  Move existing data to the front.
		b.moveDataToFront()
	}
	// INVARIANT: len(b.buf)-b.beg >= min
	//
	// Fill [b.end:] until min bytes are available.  We must loop since Read may
	// return success with fewer bytes than requested.
	for b.end-b.beg < min {
		switch nread, err := b.reader.Read(b.buf[b.end:]); {
		case nread > 0:
			b.end += nread
		case err != nil:
			return err
		}
	}
	return nil
}

// moveDataToFront moves existing data in buf to the front, so that b.beg is 0.
func (b *decbuf) moveDataToFront() {
	b.end = copy(b.buf, b.buf[b.beg:b.end])
	b.beg = 0
}

// ReadAvailable returns a buffer with the next n bytes, and increments the read
// position past those bytes.  The returned slice points directly at our
// internal buffer, and is only valid until the next decbuf call.
//
// REQUIRES: b.IsAvailable(n) && n >= 0
func (b *decbuf) ReadAvailable(n int) []byte {
	b.lim -= n
	buf := b.buf[b.beg : b.beg+n]
	b.beg += n
	return buf
}

// PeekAvailable is like ReadAvailable, but doesn't increment the read position.
func (b *decbuf) PeekAvailable(n int) []byte {
	return b.buf[b.beg : b.beg+n]
}

// ReadAvailableByte returns the next byte, and increments the read position.
//
// REQUIRES: b.IsAvailable(1)
func (b *decbuf) ReadAvailableByte() byte {
	b.lim--
	ret := b.buf[b.beg]
	b.beg++
	return ret
}

// PeekAvailableByte is like ReadAvailableByte, but doesn't increment the read
// position.
func (b *decbuf) PeekAvailableByte() byte {
	return b.buf[b.beg]
}

// ReadByte returns the next byte, and increments the read position.
func (b *decbuf) ReadByte() (byte, error) {
	if !b.IsAvailable(1) {
		if err := b.Fill(1); err != nil {
			return 0, err
		}
	}
	return b.ReadAvailableByte(), nil
}

// SkipAvailable increments the read position past the next n bytes.
//
// REQUIRES: b.IsAvailable(n) && n >= 0
func (b *decbuf) SkipAvailable(n int) {
	b.lim -= n
	b.beg += n
}

// Skip increments the read position past the next n bytes.  Returns an error if
// fewer than n bytes are available.
//
// REQUIRES: n >= 0
func (b *decbuf) Skip(n int) error {
	if b.lim >= 0 && b.lim < n {
		return io.EOF
	}
	b.lim -= n
	// If enough bytes are available, just update indices.
	avail := b.end - b.beg
	if avail >= n {
		b.beg += n
		return nil
	}
	n -= avail
	// Keep reading into buf until we've read enough bytes.
	for {
		switch nread, err := b.reader.Read(b.buf); {
		case nread > 0:
			if nread >= n {
				b.beg = n
				b.end = nread
				return nil
			}
			n -= nread
		case err != nil:
			return err
		}
	}
}

// ReadIntoBuf reads the next len(p) bytes into p, and increments the read position
// past those bytes.  Returns an error if fewer than len(p) bytes are available.
func (b *decbuf) ReadIntoBuf(p []byte) error {
	if b.lim > -1 {
		if b.lim < len(p) {
			return io.EOF
		}
		b.lim -= len(p)
	}
	// Copy bytes from the buffer.
	ncopy := copy(p, b.buf[b.beg:b.end])
	b.beg += ncopy
	p = p[ncopy:]
	// Keep reading into p until we've read enough bytes.
	for len(p) > 0 {
		switch nread, err := b.reader.Read(p); {
		case nread > 0:
			p = p[nread:]
		case err != nil:
			return err
		}
	}
	return nil
}
