// Package recyclable provides the recyclable.BufferPool, which is a never-ending font of
// recyclable.Buffer, a multiuse buffer that very reusable, supports re-reading the contained
// buffer, and when Close()d, will return home to its BufferPool for reuse.
package recyclable

import (
	"bytes"
	"errors"
	"io"
	"sync"
)

var (
	// ErrTooLarge is returned when ResetFromLimitedReader is used and the supplied Reader writes too much
	ErrTooLarge = errors.New("read byte count too large")
	// ErrPointlessClose is returned when a Buffer is Close()d, but has no home to return to.
	// Safely ignorable if you understand what the previous sentence means.
	ErrPointlessClose = errors.New("closing a Buffer with no home")
)

// Buffer is an io.Reader, io.ReadCloser, io.ReaderAt,
// io.Writer, io.WriterAt, io.WriteCloser, io.WriterTo,
// io.Seeker, io.ByteScanner, io.RuneScanner, fmt.Stringer,
// error, and probably more!
// It's designed to work in coordination with a BufferPool for
// recycling, and it's `.Close()` method puts itself back in
// the Pool it came from
type Buffer struct {
	bytes.Reader

	home *BufferPool
	m    sync.Mutex
}

// NewBuffer returns a Buffer with a proper home. Generally calling
// BufferPool.Get() is preferable to calling this directly.
func NewBuffer(home *BufferPool, bytes []byte) *Buffer {
	b := &Buffer{
		home: home,
	}
	b.Reset(bytes)
	return b
}

// Close puts itself back in the Pool it came from. This should absolutely **never** be
// called more than once per Buffer life.
// Implements `io.Closer` (also `io.ReadCloser` and `io.WriteCloser`)
func (r *Buffer) Close() error {
	if r.home == nil {
		return ErrPointlessClose
	}
	r.home.Put(r)
	return nil
}

// ResetFromReader performs a Reset() using the contents of the supplied Reader as the new content
func (r *Buffer) ResetFromReader(reader io.Reader) {
	b, _ := io.ReadAll(reader)
	r.Reset(b)
}

// ResetFromLimitedReader performs a Reset() using the contents of the supplied Reader as the new content,
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

// Bytes returns the contents of the buffer, and sets the seek pointer back to the beginning
func (r *Buffer) Bytes() []byte {
	b, _ := io.ReadAll(&r.Reader)
	r.Seek(0, 0) // reset the seeker
	return b
}

// String returns the contents of the buffer as a string, and sets the seek pointer back to the beginning
func (r *Buffer) String() string {
	b, _ := io.ReadAll(&r.Reader)
	r.Seek(0, 0) // reset the seeker
	return string(b)
}

// Error returns the contents of the buffer as a string.
// Implements “error“.
func (r *Buffer) Error() string {
	return r.String()
}

// Writer adds the bytes the written to the buffer.
// Implements “io.Writer“.
func (r *Buffer) Write(p []byte) (n int, err error) {
	b, err := io.ReadAll(&r.Reader)
	if err != nil {
		return 0, err
	}
	b = append(b, p...)

	r.Reset(b)
	return len(p), nil
}

// WriteAt allows for asynchronous writes at various locations within a buffer.
// Mixing Write and WriteAt is unlikely to yield the results one expects.
// Implements "io.WriterAt".
func (r *Buffer) WriteAt(p []byte, pos int64) (n int, err error) {
	pLen := len(p)
	expLen := pos + int64(pLen)
	r.m.Lock()
	defer r.m.Unlock()

	b, err := io.ReadAll(&r.Reader)
	if err != nil {
		return 0, err
	}
	if int64(len(b)) < expLen {
		if int64(cap(b)) < expLen {
			newBuf := make([]byte, expLen)
			copy(newBuf, b)
			b = newBuf
		}
		b = b[:expLen]
	}
	copy(b[pos:], p)
	r.Reset(b)
	return pLen, nil
}
