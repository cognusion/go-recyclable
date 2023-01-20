// Package recyclable provides the recyclable.BufferPool, which is a never-ending font of
// recyclable.Buffer, a multiuse buffer that very reusable, supports re-reading the contained
// buffer, and when Close()d, will return home to its BufferPool for reuse.
package recyclable

import (
	"bytes"
	"errors"
	"io"
)

// ErrTooLarge is returned when ResetFromLimitedReader is used and the supplied Reader writes too much
var ErrTooLarge = errors.New("read byte count too large")

// Buffer is an io.Reader, io.ReadCloser, io.ReaderAt, io.Writer,
// io.WriteCloser, io.WriterTo, io.Seeker, io.ByteScanner,
// io.RuneScanner, and more!
// It's designed to work in coordination with a BufferPool for
// recycling, and it's `.Close()` method puts itself back in
// the Pool it came from
type Buffer struct {
	bytes.Reader

	home *BufferPool
}

// NewBuffer returns a Buffer with a proper home. Generally calling
// BufferPool.Get() is preferable to calling this directly.
func NewBuffer(home *BufferPool, bytes []byte) *Buffer {
	return &Buffer{
		home: home,
	}
}

// Close puts itself back in the Pool it came from. This should absolutely **never** be
// called more than once per Buffer life.
// Implements `io.Closer` (also `io.ReadCloser` and `io.WriteCloser`)
func (r *Buffer) Close() error {
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

// Error returns the contents of the buffer as a string. Implements “error“
func (r *Buffer) Error() string {
	return r.String()
}

// Writer adds the bytes the written to the buffer. Implements “io.Writer“
func (r *Buffer) Write(p []byte) (n int, err error) {
	b, err := io.ReadAll(&r.Reader)
	if err != nil {
		return 0, err
	}
	b = append(b, p...)

	r.Reset(b)
	return len(p), nil
}
