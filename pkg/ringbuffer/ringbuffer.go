// Copyright 2019 Andy Pan. All rights reserved.

// Copyright 2019 smallnest&Allenxuxu. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ringbuffer

import (
	"errors"
	"unsafe"

	"github.com/tangtaoit/limnet/pkg/bytebuffer"
)

const initSize = 1 << 12 // 4096 bytes for the first-time allocation on ring-buffer.

// ErrIsEmpty will be returned when trying to read a empty ring-buffer.
var ErrIsEmpty = errors.New("ring-buffer is empty")

// RingBuffer is a circular buffer that implement io.ReaderWriter interface.
type RingBuffer struct {
	buf     []byte
	size    int
	mask    int
	r       int // next position to read
	w       int // next position to write
	isEmpty bool
}

// New returns a new RingBuffer whose buffer has the given size.
func New(size int) *RingBuffer {
	if size == 0 {
		return &RingBuffer{isEmpty: true}
	}
	size = CeilToPowerOfTwo(size)
	return &RingBuffer{
		buf:     make([]byte, size),
		size:    size,
		mask:    size - 1,
		isEmpty: true,
	}
}

// LazyRead reads the bytes with given length but will not move the pointer of "read".
func (r *RingBuffer) LazyRead(len int) (head []byte, tail []byte) {
	if r.isEmpty {
		return
	}

	if len <= 0 {
		return
	}

	if r.w > r.r {
		n := r.w - r.r // Length
		if n > len {
			n = len
		}
		head = r.buf[r.r : r.r+n]
		return
	}

	n := r.size - r.r + r.w // Length
	if n > len {
		n = len
	}

	if r.r+n <= r.size {
		head = r.buf[r.r : r.r+n]
	} else {
		c1 := r.size - r.r
		head = r.buf[r.r:]
		c2 := n - c1
		tail = r.buf[:c2]
	}

	return
}

// LazyReadAll reads the all bytes in this ring-buffer but will not move the pointer of "read".
func (r *RingBuffer) LazyReadAll() (head []byte, tail []byte) {
	if r.isEmpty {
		return
	}

	if r.w > r.r {
		head = r.buf[r.r:r.w]
		return
	}

	head = r.buf[r.r:]
	if r.w != 0 {
		tail = r.buf[:r.w]
	}

	return
}

// Shift shifts the "read" pointer.
func (r *RingBuffer) Shift(n int) {
	if n <= 0 {
		return
	}

	if n < r.Length() {
		r.r = (r.r + n) & r.mask
		if r.r == r.w {
			r.isEmpty = true
		}
	} else {
		r.Reset()
	}
}

// Read reads up to len(p) bytes into p. It returns the number of bytes read (0 <= n <= len(p)) and any error
// encountered.
// Even if Read returns n < len(p), it may use all of p as scratch space during the call.
// If some data is available but not len(p) bytes, Read conventionally returns what is available instead of waiting
// for more.
// When Read encounters an error or end-of-file condition after successfully reading n > 0 bytes,
// it returns the number of bytes read. It may return the (non-nil) error from the same call or return the
// error (and n == 0) from a subsequent call.
// Callers should always process the n > 0 bytes returned before considering the error err.
// Doing so correctly handles I/O errors that happen after reading some bytes and also both of the allowed EOF
// behaviors.
func (r *RingBuffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if r.isEmpty {
		return 0, ErrIsEmpty
	}

	if r.w > r.r {
		n = r.w - r.r
		if n > len(p) {
			n = len(p)
		}
		copy(p, r.buf[r.r:r.r+n])
		r.r = r.r + n
		if r.r == r.w {
			r.isEmpty = true
		}
		return
	}

	n = r.size - r.r + r.w
	if n > len(p) {
		n = len(p)
	}

	if r.r+n <= r.size {
		copy(p, r.buf[r.r:r.r+n])
	} else {
		c1 := r.size - r.r
		copy(p, r.buf[r.r:])
		c2 := n - c1
		copy(p[c1:], r.buf[:c2])
	}
	r.r = (r.r + n) & r.mask
	if r.r == r.w {
		r.isEmpty = true
	}

	return n, err
}

// ReadByte reads and returns the next byte from the input or ErrIsEmpty.
func (r *RingBuffer) ReadByte() (b byte, err error) {
	if r.isEmpty {
		return 0, ErrIsEmpty
	}
	b = r.buf[r.r]
	r.r++
	if r.r == r.size {
		r.r = 0
	}
	if r.r == r.w {
		r.isEmpty = true
	}

	return b, err
}

// Write writes len(p) bytes from p to the underlying buf.
// It returns the number of bytes written from p (n == len(p) > 0) and any error encountered that caused the write to
// stop early.
// If the length of p is greater than the writable capacity of this ring-buffer, it will allocate more memory to
// this ring-buffer.
// Write must not modify the slice data, even temporarily.
func (r *RingBuffer) Write(p []byte) (n int, err error) {
	n = len(p)
	if n == 0 {
		return 0, nil
	}

	free := r.Free()
	if n > free {
		r.malloc(n - free)
	}

	if r.w >= r.r {
		c1 := r.size - r.w
		if c1 >= n {
			copy(r.buf[r.w:], p)
			r.w += n
		} else {
			copy(r.buf[r.w:], p[:c1])
			c2 := n - c1
			copy(r.buf, p[c1:])
			r.w = c2
		}
	} else {
		copy(r.buf[r.w:], p)
		r.w += n
	}

	if r.w == r.size {
		r.w = 0
	}

	r.isEmpty = false

	return n, err
}

// WriteByte writes one byte into buffer
func (r *RingBuffer) WriteByte(c byte) error {
	if r.Free() < 1 {
		r.malloc(1)
	}
	r.buf[r.w] = c
	r.w++

	if r.w == r.size {
		r.w = 0
	}
	r.isEmpty = false

	return nil
}

// Length returns the length of available read bytes.
func (r *RingBuffer) Length() int {
	if r.r == r.w {
		if r.isEmpty {
			return 0
		}
		return r.size
	}

	if r.w > r.r {
		return r.w - r.r
	}

	return r.size - r.r + r.w
}

// Len returns the length of the underlying buffer.
func (r *RingBuffer) Len() int {
	return len(r.buf)
}

// Cap returns the size of the underlying buffer.
func (r *RingBuffer) Cap() int {
	return r.size
}

// Free returns the length of available bytes to write.
func (r *RingBuffer) Free() int {
	if r.r == r.w {
		if r.isEmpty {
			return r.size
		}
		return 0
	}

	if r.w < r.r {
		return r.r - r.w
	}

	return r.size - r.w + r.r
}

// WriteString writes the contents of the string s to buffer, which accepts a slice of bytes.
func (r *RingBuffer) WriteString(s string) (n int, err error) {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	buf := *(*[]byte)(unsafe.Pointer(&h))
	return r.Write(buf)
}

// ByteBuffer returns all available read bytes. It does not move the read pointer and only copy the available data.
func (r *RingBuffer) ByteBuffer() *bytebuffer.ByteBuffer {
	if r.isEmpty {
		return nil
	} else if r.w == r.r {
		bb := bytebuffer.Get()
		_, _ = bb.Write(r.buf[r.r:])
		_, _ = bb.Write(r.buf[:r.w])
		return bb
	}

	bb := bytebuffer.Get()
	if r.w > r.r {
		_, _ = bb.Write(r.buf[r.r:r.w])
		return bb
	}

	_, _ = bb.Write(r.buf[r.r:])

	if r.w != 0 {
		_, _ = bb.Write(r.buf[:r.w])
	}

	return bb
}

// WithByteBuffer combines the available read bytes and the given bytes. It does not move the read pointer and
// only copy the available data.
func (r *RingBuffer) WithByteBuffer(b []byte) *bytebuffer.ByteBuffer {
	if r.isEmpty {
		return &bytebuffer.ByteBuffer{B: b}
	} else if r.w == r.r {
		bb := bytebuffer.Get()
		_, _ = bb.Write(r.buf[r.r:])
		_, _ = bb.Write(r.buf[:r.w])
		_, _ = bb.Write(b)
		return bb
	}

	bb := bytebuffer.Get()
	if r.w > r.r {
		_, _ = bb.Write(r.buf[r.r:r.w])
		_, _ = bb.Write(b)
		return bb
	}

	_, _ = bb.Write(r.buf[r.r:])

	if r.w != 0 {
		_, _ = bb.Write(r.buf[:r.w])
	}
	_, _ = bb.Write(b)

	return bb
}

// IsFull returns this ringbuffer is full.
func (r *RingBuffer) IsFull() bool {
	return r.r == r.w && !r.isEmpty
}

// IsEmpty returns this ringbuffer is empty.
func (r *RingBuffer) IsEmpty() bool {
	return r.isEmpty
}

// Reset the read pointer and writer pointer to zero.
func (r *RingBuffer) Reset() {
	r.r = 0
	r.w = 0
	r.isEmpty = true
}

func (r *RingBuffer) malloc(cap int) {
	var newCap int
	if r.size == 0 && initSize >= cap {
		newCap = initSize
	} else {
		newCap = CeilToPowerOfTwo(r.size + cap)
	}
	newBuf := make([]byte, newCap)
	oldLen := r.Length()
	_, _ = r.Read(newBuf)
	r.r = 0
	r.w = oldLen
	r.size = newCap
	r.mask = newCap - 1
	r.buf = newBuf
}

const (
	bitsize = 32 << (^uint(0) >> 63)
	//maxint        = int(1<<(bitsize-1) - 1)
	maxintHeadBit = 1 << (bitsize - 2)
)

// CeilToPowerOfTwo returns the least power of two integer value greater than
// or equal to n.
func CeilToPowerOfTwo(n int) int {
	if n&maxintHeadBit != 0 && n > maxintHeadBit {
		panic("argument is too large")
	}
	if n <= 2 {
		return 2
	}
	n--
	n = fillBits(n)
	n++
	return n
}
func fillBits(n int) int {
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n
}
