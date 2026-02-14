// Package recyclable provides the recyclable.BufferPool, which is a never-ending font of
// recyclable.Buffer, a multiuse buffer that is very reusable, supports re-reading the contained
// buffer, and when Close()d, will return home to its BufferPool for reuse. Buffer also supports
// safe concurrent WriteAt and ReadAt.
//
// Since v2, Buffer is a cleaner fusion of a [bytes.Buffer] and [bytes.Reader], with large swaths of code
// reused from the Go standard library. The resulting Write performance gains are massive.
//
// Migrating from v1 should be quite seamless with two caveats:
//   - Buffer is now missing ReadRune() and UnreadRune() to simplify accounting, so is no longer an [io.RuneScanner]
//   - Read operations no longer do a `Seek(0,0)` after every operation, so the caller needs to do so themselves if they wish to re-Read the Buffer again.
package recyclable

/*
v1:
```
BenchmarkRBNewRaw-12              	1000000000	         0.2382 ns/op	       0 B/op	       0 allocs/op
BenchmarkRBNewGet-12              	100000000	        12.22 ns/op	       0 B/op	       0 allocs/op
BenchmarkRBNewMake-12             	 8608063	       157.4 ns/op	      64 B/op	       1 allocs/op
BenchmarkRBNewGetResetFixed-12    	96885499	        12.12 ns/op	       0 B/op	       0 allocs/op
BenchmarkRBNewGetResetMake-12     	100000000	        11.25 ns/op	       0 B/op	       0 allocs/op
BenchmarkBytes-12                 	11573713	        95.64 ns/op	     512 B/op	       1 allocs/op
BenchmarkReset-12                 	1000000000	         0.4705 ns/op	       0 B/op	       0 allocs/op
BenchmarkWrite-12                 	11343724	        97.44 ns/op	     512 B/op	       1 allocs/op
BenchmarkBBWrite-12               	273206918	         4.262 ns/op	       0 B/op	       0 allocs/op
BenchmarkWrite1000-12             	    2358	    445911 ns/op	 2588165 B/op	    2610 allocs/op
BenchmarkBBWrite1000-12           	  277684	      4194 ns/op	       0 B/op	       0 allocs/op
```

v2:
```
BenchmarkRBNewRaw-12              	1000000000	         0.2407 ns/op	       0 B/op	       0 allocs/op
BenchmarkRBNewGet-12              	100000000	        10.85 ns/op	       0 B/op	       0 allocs/op
BenchmarkRBNewMake-12             	 8120359	       141.9 ns/op	      48 B/op	       1 allocs/op
BenchmarkRBNewGetResetFixed-12    	109928661	        11.01 ns/op	       0 B/op	       0 allocs/op
BenchmarkRBNewGetResetMake-12     	100000000	        10.59 ns/op	       0 B/op	       0 allocs/op
BenchmarkBytes-12                 	1000000000	         0.3482 ns/op	       0 B/op	       0 allocs/op
BenchmarkEmpty-12                 	1000000000	         0.2344 ns/op	       0 B/op	       0 allocs/op
BenchmarkReset-12                 	1000000000	         0.4700 ns/op	       0 B/op	       0 allocs/op
BenchmarkWrite-12                 	285068593	         4.039 ns/op	       0 B/op	       0 allocs/op
BenchmarkBBWrite-12               	271641879	         4.260 ns/op	       0 B/op	       0 allocs/op
BenchmarkWrite1000-12             	  298587	      3942 ns/op	       0 B/op	       0 allocs/op
BenchmarkBBWrite1000-12           	  290127	      4180 ns/op	       0 B/op	       0 allocs/op
```
*/

import (
	"bytes"
	"io"
	"sync"
	"unicode/utf8"
)

const (
	// ErrTooLarge is returned when ResetFromLimitedReader is used and the supplied Reader writes too much
	ErrTooLarge = Error("read byte count too large")
	// ErrPointlessClose is returned when a Buffer is Close()d, but has no home to return to.
	// Safely ignorable if you understand what the previous sentence means.
	ErrPointlessClose = Error("closing a Buffer with no home")
	// ErrInvalidWhence is returned when the whence provided to Seek is not one of the io.Seek* constants.
	ErrInvalidWhence = Error("invalid whence")
	// ErrNegativePosition is returned when an index value would be below zero.
	ErrNegativePosition = Error("negative position")
	// ErrNegativeOffset is returned when an offset value would be below zero.
	ErrNegativeOffset = Error("negative offset")
	// ErrBeginning is returned when `UnreadByte` is called, but the index is already at the beginning.
	ErrBeginning = Error("at the beginning")

	// smallBufferSize is an initial allocation minimal capacity.
	smallBufferSize = 64
	//	maxInt is used for bounds checks
	maxInt = int(^uint(0) >> 1)
)

// Error is an error type
type Error string

// Error returns the stringified version of Error
func (e Error) Error() string {
	return string(e)
}

// Buffer is an io.Reader, io.ReadCloser, io.ReaderAt, io.Writer, io.WriterAt, io.WriteCloser, io.WriterTo,
// io.Seeker, io.ByteScanner, fmt.Stringer, error, and more (see [Test_BufferInterfacesOhMy] for the interfaces
// we assure we implement)!
//
// It's designed to work in coordination with a BufferPool for recycling, and it's `.Close()` method puts
// itself back in the Pool it came from.
//
// It is safe to use outside of a BufferPool, and a referenced zero value can be used organically as well
// (e.g. `b := &Buffer{}`).
type Buffer struct {
	home *BufferPool
	m    sync.Mutex

	buf   []byte
	index int64
}

// NewBuffer returns a Buffer with a proper home. Generally calling
// BufferPool.Get() is preferable to calling this directly.
// Calling NewBuffer with a `nil` home is perfectly safe.
func NewBuffer(home *BufferPool, bytes []byte) *Buffer {
	b := &Buffer{
		home: home,
		buf:  bytes,
	}
	return b
}

// NewBufferString returns a Buffer with a proper home. Generally calling
// BufferPool.Get() is preferable to calling this directly.
// Calling NewBufferString with a `nil` home is perfectly safe.
func NewBufferString(home *BufferPool, s string) *Buffer {
	return NewBuffer(home, []byte(s))
}

// Reset resets the Buffer to be reading from b.
func (r *Buffer) Reset(b []byte) {
	r.buf = b
	r.index = 0
}

// Empty resets the buffer to be empty, but it retains the underlying storage
// for use by future writes.
func (r *Buffer) Empty() {
	r.buf = r.buf[:0]
	r.index = 0
}

// Len returns the number of bytes used in the  buffer's underlying slice.
func (r *Buffer) Len() int { return len(r.buf) - int(r.index) }

// Cap returns the capacity of the buffer's underlying byte slice, that is, the
// total space allocated for the buffer's data.
func (r *Buffer) Cap() int { return cap(r.buf) }

// Available returns how many bytes are unused in the buffer.
func (r *Buffer) Available() int { return cap(r.buf) - len(r.buf) }

// AvailableBuffer returns an empty buffer with b.Available() capacity.
// This buffer is intended to be appended to and
// passed to an immediately succeeding [Buffer.Write] call.
// The buffer is only valid until the next write operation on b.
func (r *Buffer) AvailableBuffer() []byte { return r.buf[len(r.buf):] }

// Size returns the original length of the underlying byte slice.
// Size is the number of bytes available for reading.
func (r *Buffer) Size() int64 { return int64(len(r.buf)) }

// Next returns a slice containing the next n bytes from the buffer,
// advancing the buffer as if the bytes had been returned by [Buffer.Read].
// If there are fewer than n bytes in the buffer, Next returns the entire buffer.
// The slice is only valid until the next call to a read or write method.
func (r *Buffer) Next(n int) []byte {
	m := r.Len()
	if n > m {
		n = m
	}
	data := r.buf[r.index : r.index+int64(n)]
	r.index += int64(n)
	return data
}

// Read implements the [io.Reader] interface.
func (r *Buffer) Read(b []byte) (n int, err error) {
	if r.index >= int64(len(r.buf)) {
		return 0, io.EOF
	}
	n = copy(b, r.buf[r.index:])
	r.index += int64(n)
	return n, nil
}

// ReadString reads until the first occurrence of delim in the input,
// returning a string containing the data up to and including the delimiter.
// If ReadString encounters an error before finding a delimiter,
// it returns the data read before the error and the error itself (often [io.EOF]).
// ReadString returns err != nil if and only if the returned data does not end
// in delim.
func (r *Buffer) ReadString(delim byte) (line string, err error) {
	slice, err := r.readSlice(delim)
	return string(slice), err
}

// Seek implements the [io.Seeker] interface.
func (r *Buffer) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.index + offset
	case io.SeekEnd:
		abs = int64(len(r.buf)) + offset
	default:
		return 0, ErrInvalidWhence
	}
	if abs < 0 {
		return 0, ErrNegativeOffset
	}
	r.index = abs
	return abs, nil
}

// ReadAt implements the [io.ReaderAt] interface.
func (r *Buffer) ReadAt(b []byte, off int64) (n int, err error) {
	// cannot modify state - see io.ReaderAt
	if off < 0 {
		return 0, ErrNegativeOffset
	}
	if off >= int64(len(r.buf)) {
		return 0, io.EOF
	}
	n = copy(b, r.buf[off:])
	if n < len(b) {
		err = io.EOF
	}
	return
}

// ReadByte implements the [io.ByteReader] interface.
func (r *Buffer) ReadByte() (byte, error) {
	if r.index >= int64(len(r.buf)) {
		return 0, io.EOF
	}
	b := r.buf[r.index]
	r.index++
	return b, nil
}

// UnreadByte complements [Buffer.ReadByte] in implementing the [io.ByteScanner] interface.
// If UnreadByte is used outside of normal ByteReading, the result is undefined.
func (r *Buffer) UnreadByte() error {
	if r.index <= 0 {
		return ErrBeginning
	}
	r.index--
	return nil
}

// ReadBytes reads until the first occurrence of delim in the input,
// returning a slice containing the data up to and including the delimiter.
// If ReadBytes encounters an error before finding a delimiter,
// it returns the data read before the error and the error itself (often [io.EOF]).
// ReadBytes returns err != nil if and only if the returned data does not end in
// delim.
func (r *Buffer) ReadBytes(delim byte) (line []byte, err error) {
	slice, err := r.readSlice(delim)
	// return a copy of slice. The buffer's backing array may
	// be overwritten by later calls.
	line = append(line, slice...)
	return line, err
}

// readSlice is like ReadBytes but returns a reference to internal buffer data.
func (r *Buffer) readSlice(delim byte) (line []byte, err error) {
	i := bytes.IndexByte(r.buf[r.index:], delim)
	end := r.index + int64(i+1)
	if i < 0 {
		end = int64(len(r.buf))
		err = io.EOF
	}
	line = r.buf[r.index:end]
	r.index = end
	return line, err
}

// ReadFrom reads data from r until EOF and appends it to the buffer, growing
// the buffer as needed. The return value n is the number of bytes read. Any
// error except [io.EOF] encountered during the read is also returned. If the
// buffer becomes too large, ReadFrom will panic with [ErrTooLarge].
func (r *Buffer) ReadFrom(reader io.Reader) (n int64, err error) {
	for {
		i := r.grow(smallBufferSize)
		r.buf = r.buf[:i]
		m, e := reader.Read(r.buf[i:cap(r.buf)])
		if m < 0 {
			panic(ErrNegativePosition)
		}

		r.buf = r.buf[:i+m]
		n += int64(m)
		if e == io.EOF {
			return n, nil // e is EOF, so return nil explicitly
		}
		if e != nil {
			return n, e
		}
	}
}

// WriteTo implements the [io.WriterTo] interface.
func (r *Buffer) WriteTo(w io.Writer) (n int64, err error) {
	if r.index >= int64(len(r.buf)) {
		return 0, nil
	}
	b := r.buf[r.index:]
	m, err := w.Write(b)
	if m > len(b) {
		panic("invalid Write count")
	}
	r.index += int64(m)
	n = int64(m)
	if m != len(b) && err == nil {
		err = io.ErrShortWrite
	}
	return
}

// Close puts itself back into the [BufferPool] it came from. A harmless error is returned if Close is
// called but no BufferPool was specified at creation.
// This should absolutely **never** be called more than once per Buffer life.
// Implements [io.Closer] et. al.
func (r *Buffer) Close() error {
	if r.home == nil {
		return ErrPointlessClose
	}
	r.home.Put(r)
	return nil
}

// ResetFromReader performs a [Buffer.Reset] using the contents of the supplied Reader as the new content
func (r *Buffer) ResetFromReader(reader io.Reader) {
	b, _ := io.ReadAll(reader)
	r.Reset(b)
}

// ResetFromLimitedReader performs a [Buffer.Reset] using the contents of the supplied Reader as the new content,
// up to at most max bytes, returning ErrTooLarge if it's over. The error is not terminal, and the buffer
// may continue to be used, understanding the contents will be limited
func (r *Buffer) ResetFromLimitedReader(reader io.Reader, max int64) error {
	lr := io.LimitReader(reader, max+1)
	b, _ := io.ReadAll(lr)
	if int64(len(b)) > max {
		r.Reset(b[0:max])
		return ErrTooLarge
	}
	r.Reset(b)
	return nil
}

// Bytes returns a slice of length b.Len() holding the unread portion of the buffer.
// The slice is valid for use only until the next buffer modification (that is,
// only until the next call to a method like [Buffer.Read], [Buffer.Write], [Buffer.Reset], or [Buffer.Truncate]).
// The slice aliases the buffer content at least until the next buffer modification,
// so immediate changes to the slice will affect the result of future reads.
func (r *Buffer) Bytes() []byte { return r.buf[r.index:] }

// String returns the contents of the buffer as a string, and sets the seek pointer back to the beginning
func (r *Buffer) String() string {
	return string(r.Bytes())
}

// Error returns the contents of the buffer as a string.
// Implements [error].
func (r *Buffer) Error() string {
	return r.String()
}

// Grow grows the buffer's capacity, if necessary, to guarantee space for
// another n bytes. After Grow(n), at least n bytes can be written to the
// buffer without another allocation.
// If n is negative, Grow will panic.
// If the buffer can't grow it will panic with [ErrTooLarge].
func (r *Buffer) Grow(n int) {
	if n < 0 {
		panic("negative count")
	}
	m := r.grow(int64(n))
	r.buf = r.buf[:m]
}

// Write appends the contents of p to the buffer, growing the buffer as
// needed. The return value n is the length of p; err is always nil. If the
// buffer becomes too large, Write will panic with [ErrTooLarge].
func (r *Buffer) Write(p []byte) (n int, err error) {
	m, ok := r.tryGrowByReslice(int64(len(p)))
	if !ok {
		m = r.grow(int64(len(p)))
	}
	return copy(r.buf[m:], p), nil
}

// WriteString appends the contents of s to the buffer, growing the buffer as
// needed. The return value n is the length of s; err is always nil. If the
// buffer becomes too large, WriteString will panic with [ErrTooLarge].
func (r *Buffer) WriteString(s string) (n int, err error) {
	m, ok := r.tryGrowByReslice(int64(len(s)))
	if !ok {
		m = r.grow(int64(len(s)))
	}
	return copy(r.buf[m:], s), nil
}

// WriteByte appends the byte c to the buffer, growing the buffer as needed.
// The returned error is always nil, but is included to match [bufio.Writer]'s
// WriteByte. If the buffer becomes too large, WriteByte will panic with
// [ErrTooLarge].
func (r *Buffer) WriteByte(c byte) error {
	m, ok := r.tryGrowByReslice(1)
	if !ok {
		m = r.grow(1)
	}
	r.buf[m] = c
	return nil
}

// WriteRune appends the UTF-8 encoding of Unicode code point r to the
// buffer, returning its length and an error, which is always nil but is
// included to match [bufio.Writer]'s WriteRune. The buffer is grown as needed;
// if it becomes too large, WriteRune will panic with [ErrTooLarge].
func (r *Buffer) WriteRune(p rune) (n int, err error) {
	// Compare as uint32 to correctly handle negative runes.
	if int32(p) < utf8.RuneSelf {
		//#nosec G115 -- We ensured that the value of p is less than 128 previously, thus fits in a byte without overflow.
		r.WriteByte(byte(p))
		return 1, nil
	}
	m, ok := r.tryGrowByReslice(utf8.UTFMax)
	if !ok {
		m = r.grow(utf8.UTFMax)
	}
	r.buf = utf8.AppendRune(r.buf[:m], p)
	return len(r.buf) - m, nil
}

// WriteAt allows for asynchronous writes at various locations within a buffer.
// Mixing Write and WriteAt is unlikely to yield the results one expects.
// Implements [io.WriterAt].
func (r *Buffer) WriteAt(p []byte, pos int64) (n int, err error) {
	pLen := len(p)
	expLen := pos + int64(pLen)
	r.m.Lock()
	defer r.m.Unlock()

	if int64(len(r.buf)) < expLen {
		g := expLen - int64(len(r.buf))
		if g > 0 {
			r.grow(g)
		}
		r.buf = r.buf[:expLen]

	}

	copy(r.buf[pos:], p)
	return pLen, nil
}

// Truncate discards all but the first n unread bytes from the buffer
// but continues to use the same allocated storage.
// It panics if n is negative or greater than the length of the buffer.
func (r *Buffer) Truncate(n int) {
	if n == 0 {
		r.Empty()
		return
	}
	if n < 0 || n > r.Len() {
		panic("truncation out of range")
	}
	r.buf = r.buf[:r.index+int64(n)]
}

// tryGrowByReslice is an inlineable version of grow for the fast-case where the
// internal buffer only needs to be resliced.
// It returns the index where bytes should be written and whether it succeeded.
func (r *Buffer) tryGrowByReslice(n int64) (int, bool) {
	if l := len(r.buf); n <= int64(cap(r.buf)-l) {
		r.buf = r.buf[:int64(l)+n]
		return l, true
	}
	return 0, false
}

// grow grows the buffer to guarantee space for n more bytes.
// It returns the index where bytes should be written.
// If the buffer can't grow it will panic with ErrTooLarge.
func (r *Buffer) grow(n int64) int {
	m := r.Len()
	// If buffer is empty, reset to recover space.
	if m == 0 && r.index != 0 {
		r.Empty()
	}
	// Try to grow by means of a reslice.
	if i, ok := r.tryGrowByReslice(n); ok {
		return i
	}
	if r.buf == nil && n <= smallBufferSize {
		r.buf = make([]byte, n, smallBufferSize)
		return 0
	}
	c := int64(cap(r.buf))
	if n <= c/2-int64(m) {
		// We can slide things down instead of allocating a new
		// slice. We only need m+n <= c to slide, but
		// we instead let capacity get twice as large so we
		// don't spend all our time copying.
		copy(r.buf, r.buf[r.index:])
	} else if c > int64(maxInt)-c-n {
		panic(ErrTooLarge)
	} else {
		// Add b.off to account for b.buf[:b.off] being sliced off the front.
		r.buf = growSlice(r.buf[r.index:], r.index+n)
	}
	// Restore b.off and len(b.buf).
	r.index = 0
	r.buf = r.buf[:int64(m)+n]
	return m
}

// growSlice grows b by n, preserving the original content of b.
// If the allocation fails, it panics with ErrTooLarge.
func growSlice(b []byte, n int64) []byte {
	defer func() {
		if recover() != nil {
			panic(ErrTooLarge)
		}
	}()
	// TODO(http://golang.org/issue/51462): We should rely on the append-make
	// pattern so that the compiler can call runtime.growslice. For example:
	//	return append(b, make([]byte, n)...)
	// This avoids unnecessary zero-ing of the first len(b) bytes of the
	// allocated slice, but this pattern causes b to escape onto the heap.
	//
	// Instead use the append-make pattern with a nil slice to ensure that
	// we allocate buffers rounded up to the closest size class.
	c := int64(len(b)) + n // ensure enough space for n elements
	if c < 2*int64(cap(b)) {
		// The growth rate has historically always been 2x. In the future,
		// we could rely purely on append to determine the growth rate.
		c = 2 * int64(cap(b))
	}
	b2 := append([]byte(nil), make([]byte, c)...)
	i := copy(b2, b)
	return b2[:i]
}
