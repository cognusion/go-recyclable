

# recyclable
`import "github.com/cognusion/go-recyclable/v2"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)
* [Examples](#pkg-examples)

## <a name="pkg-overview">Overview</a>
Package recyclable provides the recyclable.BufferPool, which is a never-ending font of
recyclable.Buffer, a multiuse buffer that is very reusable, supports re-reading the contained
buffer, and when Close()d, will return home to its BufferPool for reuse. Buffer also supports
safe concurrent WriteAt and ReadAt.

Since v2, Buffer is a cleaner fusion of a [bytes.Buffer] and [bytes.Reader], with large swaths of code
reused from the Go standard library. The resulting Write performance gains are massive.

Migrating from v1 should be quite seamless with two caveats:


	- Buffer is now missing ReadRune() and UnreadRune() to simplify accounting, so is no longer an [io.RuneScanner]
	- Read operations no longer do a `Seek(0,0)` after every operation, so the caller needs to do so themselves if they wish to re-Read the Buffer again.


##### Example :
HOWTO implement a goro-safe BufferPool for Buffers

``` go
// BufferPool allows us to have a never-ending font of Buffers.
// If the Pool is empty, a new one is created. If there is one someone put
// back, then it is returned. Saves on allocs like crazy. <3
rPool := NewBufferPool()

// Let's grab a Buffer
    rb := rPool.Get()

    // And immediately reset the value, as we can't trust it to be empty
rb.Reset([]byte("Hello World"))

// Unlike most buffers, we can re-read it:
for range 10 {
    if string(rb.Bytes()) != "Hello World" {
        panic("OMG! Can't reread?!!!")
    }
}

// Or get the string value, if you prefer (and know it's safe)
    for range 10 {
        if rb.String() != "Hello World" {
            panic("OMG! Can't reread?!!!")
        }
    }

    // Appending to it as an io.Writer works as well
    io.WriteString(rb, ", nice day?")
    if string(rb.Bytes()) != "Hello World, nice day?" {
        panic("OMG! Append failed?!")
    }

    // Lastly, when you're all done, just close it.
    rb.Close() // and it will go back into the Pool.
    // Please don't use it anymore. Get a fresh one.

    rb = rPool.Get() // See, not hard?
    defer rb.Close() // Just remember to close it, unless you're passing it elsewhere
    rb.Empty()       // Ready to go

    /* HINTS:
    * Makes awesome ``http.Request.Body``s, especially since they get automatically ``.Close()``d when done with
    * Replaces ``bytes.Buffer`` and ``bytes.Reader`` for most uses
    * Isa Stringer and an error
    * As a Writer and a Reader can be used in pipes and elsewhere
      * You can also pipe them to themselves, but that is a very bad idea unless you love watching OOMs
    */
```



## <a name="pkg-index">Index</a>
* [Constants](#pkg-constants)
* [type Buffer](#Buffer)
  * [func NewBuffer(home *BufferPool, bytes []byte) *Buffer](#NewBuffer)
  * [func NewBufferString(home *BufferPool, s string) *Buffer](#NewBufferString)
  * [func (r *Buffer) Available() int](#Buffer.Available)
  * [func (r *Buffer) AvailableBuffer() []byte](#Buffer.AvailableBuffer)
  * [func (r *Buffer) Bytes() []byte](#Buffer.Bytes)
  * [func (r *Buffer) Cap() int](#Buffer.Cap)
  * [func (r *Buffer) Close() error](#Buffer.Close)
  * [func (r *Buffer) Empty()](#Buffer.Empty)
  * [func (r *Buffer) Error() string](#Buffer.Error)
  * [func (r *Buffer) Grow(n int)](#Buffer.Grow)
  * [func (r *Buffer) Len() int](#Buffer.Len)
  * [func (r *Buffer) Next(n int) []byte](#Buffer.Next)
  * [func (r *Buffer) Read(b []byte) (n int, err error)](#Buffer.Read)
  * [func (r *Buffer) ReadAt(b []byte, off int64) (n int, err error)](#Buffer.ReadAt)
  * [func (r *Buffer) ReadByte() (byte, error)](#Buffer.ReadByte)
  * [func (r *Buffer) ReadBytes(delim byte) (line []byte, err error)](#Buffer.ReadBytes)
  * [func (r *Buffer) ReadFrom(reader io.Reader) (n int64, err error)](#Buffer.ReadFrom)
  * [func (r *Buffer) ReadString(delim byte) (line string, err error)](#Buffer.ReadString)
  * [func (r *Buffer) Reset(b []byte)](#Buffer.Reset)
  * [func (r *Buffer) ResetFromLimitedReader(reader io.Reader, max int64) error](#Buffer.ResetFromLimitedReader)
  * [func (r *Buffer) ResetFromReader(reader io.Reader)](#Buffer.ResetFromReader)
  * [func (r *Buffer) Seek(offset int64, whence int) (int64, error)](#Buffer.Seek)
  * [func (r *Buffer) Size() int64](#Buffer.Size)
  * [func (r *Buffer) String() string](#Buffer.String)
  * [func (r *Buffer) Truncate(n int)](#Buffer.Truncate)
  * [func (r *Buffer) UnreadByte() error](#Buffer.UnreadByte)
  * [func (r *Buffer) Write(p []byte) (n int, err error)](#Buffer.Write)
  * [func (r *Buffer) WriteAt(p []byte, pos int64) (n int, err error)](#Buffer.WriteAt)
  * [func (r *Buffer) WriteByte(c byte) error](#Buffer.WriteByte)
  * [func (r *Buffer) WriteRune(p rune) (n int, err error)](#Buffer.WriteRune)
  * [func (r *Buffer) WriteString(s string) (n int, err error)](#Buffer.WriteString)
  * [func (r *Buffer) WriteTo(w io.Writer) (n int64, err error)](#Buffer.WriteTo)
* [type BufferPool](#BufferPool)
  * [func NewBufferPool() *BufferPool](#NewBufferPool)
  * [func (p *BufferPool) Get() *Buffer](#BufferPool.Get)
  * [func (p *BufferPool) Put(b *Buffer)](#BufferPool.Put)
* [type Error](#Error)
  * [func (e Error) Error() string](#Error.Error)

#### <a name="pkg-examples">Examples</a>
* [Package](#example-)

#### <a name="pkg-files">Package files</a>
[buffer.go](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go) [bufferpool.go](https://github.com/cognusion/go-recyclable/tree/master/v2/bufferpool.go)


## <a name="pkg-constants">Constants</a>
``` go
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
)
```




## <a name="Buffer">type</a> [Buffer](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=4981:5066#L92)
``` go
type Buffer struct {
    // contains filtered or unexported fields
}

```
Buffer is an io.Reader, io.ReadCloser, io.ReaderAt, io.Writer, io.WriterAt, io.WriteCloser, io.WriterTo,
io.Seeker, io.ByteScanner, fmt.Stringer, error, and more (see [Test_BufferInterfacesOhMy] for the interfaces
we assure we implement)!

It's designed to work in coordination with a BufferPool for recycling, and it's `.Close()` method puts
itself back in the Pool it came from.

It is safe to use outside of a BufferPool, and a referenced zero value can be used organically as well
(e.g. `b := &Buffer{}`).







### <a name="NewBuffer">func</a> [NewBuffer](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=5254:5308#L103)
``` go
func NewBuffer(home *BufferPool, bytes []byte) *Buffer
```
NewBuffer returns a Buffer with a proper home. Generally calling
BufferPool.Get() is preferable to calling this directly.
Calling NewBuffer with a `nil` home is perfectly safe.


### <a name="NewBufferString">func</a> [NewBufferString](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=5569:5625#L114)
``` go
func NewBufferString(home *BufferPool, s string) *Buffer
```
NewBufferString returns a Buffer with a proper home. Generally calling
BufferPool.Get() is preferable to calling this directly.
Calling NewBufferString with a `nil` home is perfectly safe.





### <a name="Buffer.Available">func</a> (\*Buffer) [Available](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=6326:6358#L139)
``` go
func (r *Buffer) Available() int
```
Available returns how many bytes are unused in the buffer.




### <a name="Buffer.AvailableBuffer">func</a> (\*Buffer) [AvailableBuffer](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=6641:6682#L145)
``` go
func (r *Buffer) AvailableBuffer() []byte
```
AvailableBuffer returns an empty buffer with b.Available() capacity.
This buffer is intended to be appended to and
passed to an immediately succeeding [Buffer.Write] call.
The buffer is only valid until the next write operation on b.




### <a name="Buffer.Bytes">func</a> (\*Buffer) [Bytes](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=12877:12908#L348)
``` go
func (r *Buffer) Bytes() []byte
```
Bytes returns a slice of length b.Len() holding the unread portion of the buffer.
The slice is valid for use only until the next buffer modification (that is,
only until the next call to a method like [Buffer.Read], [Buffer.Write], [Buffer.Reset], or [Buffer.Truncate]).
The slice aliases the buffer content at least until the next buffer modification,
so immediate changes to the slice will affect the result of future reads.




### <a name="Buffer.Cap">func</a> (\*Buffer) [Cap](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=6214:6240#L136)
``` go
func (r *Buffer) Cap() int
```
Cap returns the capacity of the buffer's underlying byte slice, that is, the
total space allocated for the buffer's data.




### <a name="Buffer.Close">func</a> (\*Buffer) [Close](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=11594:11624#L315)
``` go
func (r *Buffer) Close() error
```
Close puts itself back into the [BufferPool] it came from. A harmless error is returned if Close is
called but no BufferPool was specified at creation.
This should absolutely **never** be called more than once per Buffer life.
Implements [io.Closer] et. al.




### <a name="Buffer.Empty">func</a> (\*Buffer) [Empty](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=5884:5908#L126)
``` go
func (r *Buffer) Empty()
```
Empty resets the buffer to be empty, but it retains the underlying storage
for use by future writes.




### <a name="Buffer.Error">func</a> (\*Buffer) [Error](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=13187:13218#L357)
``` go
func (r *Buffer) Error() string
```
Error returns the contents of the buffer as a string.
Implements [error].




### <a name="Buffer.Grow">func</a> (\*Buffer) [Grow](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=13529:13557#L366)
``` go
func (r *Buffer) Grow(n int)
```
Grow grows the buffer's capacity, if necessary, to guarantee space for
another n bytes. After Grow(n), at least n bytes can be written to the
buffer without another allocation.
If n is negative, Grow will panic.
If the buffer can't grow it will panic with [ErrTooLarge].




### <a name="Buffer.Len">func</a> (\*Buffer) [Len](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=6021:6047#L132)
``` go
func (r *Buffer) Len() int
```
Len returns the number of bytes used in the  buffer's underlying slice.




### <a name="Buffer.Next">func</a> (\*Buffer) [Next](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=7195:7230#L155)
``` go
func (r *Buffer) Next(n int) []byte
```
Next returns a slice containing the next n bytes from the buffer,
advancing the buffer as if the bytes had been returned by [Buffer.Read].
If there are fewer than n bytes in the buffer, Next returns the entire buffer.
The slice is only valid until the next call to a read or write method.




### <a name="Buffer.Read">func</a> (\*Buffer) [Read](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=7396:7446#L166)
``` go
func (r *Buffer) Read(b []byte) (n int, err error)
```
Read implements the [io.Reader] interface.




### <a name="Buffer.ReadAt">func</a> (\*Buffer) [ReadAt](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=8546:8609#L207)
``` go
func (r *Buffer) ReadAt(b []byte, off int64) (n int, err error)
```
ReadAt implements the [io.ReaderAt] interface.




### <a name="Buffer.ReadByte">func</a> (\*Buffer) [ReadByte](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=8880:8921#L223)
``` go
func (r *Buffer) ReadByte() (byte, error)
```
ReadByte implements the [io.ByteReader] interface.




### <a name="Buffer.ReadBytes">func</a> (\*Buffer) [ReadBytes](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=9696:9759#L248)
``` go
func (r *Buffer) ReadBytes(delim byte) (line []byte, err error)
```
ReadBytes reads until the first occurrence of delim in the input,
returning a slice containing the data up to and including the delimiter.
If ReadBytes encounters an error before finding a delimiter,
it returns the data read before the error and the error itself (often [io.EOF]).
ReadBytes returns err != nil if and only if the returned data does not end in
delim.




### <a name="Buffer.ReadFrom">func</a> (\*Buffer) [ReadFrom](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=10584:10648#L273)
``` go
func (r *Buffer) ReadFrom(reader io.Reader) (n int64, err error)
```
ReadFrom reads data from r until EOF and appends it to the buffer, growing
the buffer as needed. The return value n is the number of bytes read. Any
error except [io.EOF] encountered during the read is also returned. If the
buffer becomes too large, ReadFrom will panic with [ErrTooLarge].




### <a name="Buffer.ReadString">func</a> (\*Buffer) [ReadString](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=7963:8027#L181)
``` go
func (r *Buffer) ReadString(delim byte) (line string, err error)
```
ReadString reads until the first occurrence of delim in the input,
returning a string containing the data up to and including the delimiter.
If ReadString encounters an error before finding a delimiter,
it returns the data read before the error and the error itself (often [io.EOF]).
ReadString returns err != nil if and only if the returned data does not end
in delim.




### <a name="Buffer.Reset">func</a> (\*Buffer) [Reset](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=5715:5747#L119)
``` go
func (r *Buffer) Reset(b []byte)
```
Reset resets the Buffer to be reading from b.




### <a name="Buffer.ResetFromLimitedReader">func</a> (\*Buffer) [ResetFromLimitedReader](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=12200:12274#L332)
``` go
func (r *Buffer) ResetFromLimitedReader(reader io.Reader, max int64) error
```
ResetFromLimitedReader performs a [Buffer.Reset] using the contents of the supplied Reader as the new content,
up to at most max bytes, returning ErrTooLarge if it's over. The error is not terminal, and the buffer
may continue to be used, understanding the contents will be limited




### <a name="Buffer.ResetFromReader">func</a> (\*Buffer) [ResetFromReader](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=11813:11863#L324)
``` go
func (r *Buffer) ResetFromReader(reader io.Reader)
```
ResetFromReader performs a [Buffer.Reset] using the contents of the supplied Reader as the new content




### <a name="Buffer.Seek">func</a> (\*Buffer) [Seek](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=8140:8202#L187)
``` go
func (r *Buffer) Seek(offset int64, whence int) (int64, error)
```
Seek implements the [io.Seeker] interface.




### <a name="Buffer.Size">func</a> (\*Buffer) [Size](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=6834:6863#L149)
``` go
func (r *Buffer) Size() int64
```
Size returns the original length of the underlying byte slice.
Size is the number of bytes available for reading.




### <a name="Buffer.String">func</a> (\*Buffer) [String](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=13043:13075#L351)
``` go
func (r *Buffer) String() string
```
String returns the contents of the buffer as a string, and sets the seek pointer back to the beginning




### <a name="Buffer.Truncate">func</a> (\*Buffer) [Truncate](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=16144:16176#L452)
``` go
func (r *Buffer) Truncate(n int)
```
Truncate discards all but the first n unread bytes from the buffer
but continues to use the same allocated storage.
It panics if n is negative or greater than the length of the buffer.




### <a name="Buffer.UnreadByte">func</a> (\*Buffer) [UnreadByte](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=9204:9239#L234)
``` go
func (r *Buffer) UnreadByte() error
```
UnreadByte complements [Buffer.ReadByte] in implementing the [io.ByteScanner] interface.
If UnreadByte is used outside of normal ByteReading, the result is undefined.




### <a name="Buffer.Write">func</a> (\*Buffer) [Write](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=13860:13911#L377)
``` go
func (r *Buffer) Write(p []byte) (n int, err error)
```
Write appends the contents of p to the buffer, growing the buffer as
needed. The return value n is the length of p; err is always nil. If the
buffer becomes too large, Write will panic with [ErrTooLarge].




### <a name="Buffer.WriteAt">func</a> (\*Buffer) [WriteAt](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=15635:15699#L430)
``` go
func (r *Buffer) WriteAt(p []byte, pos int64) (n int, err error)
```
WriteAt allows for asynchronous writes at various locations within a buffer.
Mixing Write and WriteAt is unlikely to yield the results one expects.
Implements [io.WriterAt].




### <a name="Buffer.WriteByte">func</a> (\*Buffer) [WriteByte](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=14687:14727#L400)
``` go
func (r *Buffer) WriteByte(c byte) error
```
WriteByte appends the byte c to the buffer, growing the buffer as needed.
The returned error is always nil, but is included to match [bufio.Writer]'s
WriteByte. If the buffer becomes too large, WriteByte will panic with
[ErrTooLarge].




### <a name="Buffer.WriteRune">func</a> (\*Buffer) [WriteRune](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=15113:15166#L413)
``` go
func (r *Buffer) WriteRune(p rune) (n int, err error)
```
WriteRune appends the UTF-8 encoding of Unicode code point r to the
buffer, returning its length and an error, which is always nil but is
included to match [bufio.Writer]'s WriteRune. The buffer is grown as needed;
if it becomes too large, WriteRune will panic with [ErrTooLarge].




### <a name="Buffer.WriteString">func</a> (\*Buffer) [WriteString](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=14260:14317#L388)
``` go
func (r *Buffer) WriteString(s string) (n int, err error)
```
WriteString appends the contents of s to the buffer, growing the buffer as
needed. The return value n is the length of s; err is always nil. If the
buffer becomes too large, WriteString will panic with [ErrTooLarge].




### <a name="Buffer.WriteTo">func</a> (\*Buffer) [WriteTo](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=11008:11066#L294)
``` go
func (r *Buffer) WriteTo(w io.Writer) (n int64, err error)
```
WriteTo implements the [io.WriterTo] interface.




## <a name="BufferPool">type</a> [BufferPool](https://github.com/cognusion/go-recyclable/tree/master/v2/bufferpool.go?s=84:126#L6)
``` go
type BufferPool struct {
    // contains filtered or unexported fields
}

```
BufferPool is a self-managing pool of Buffers







### <a name="NewBufferPool">func</a> [NewBufferPool](https://github.com/cognusion/go-recyclable/tree/master/v2/bufferpool.go?s=179:211#L11)
``` go
func NewBufferPool() *BufferPool
```
NewBufferPool returns an initialized BufferPool





### <a name="BufferPool.Get">func</a> (\*BufferPool) [Get](https://github.com/cognusion/go-recyclable/tree/master/v2/bufferpool.go?s=525:559#L24)
``` go
func (p *BufferPool) Get() *Buffer
```
Get will return an existing Buffer or a new one if the pool is empty.
REMEMBER to Reset or Empty the Buffer and don't just start using it, as it
may very well have old data in it!




### <a name="BufferPool.Put">func</a> (\*BufferPool) [Put](https://github.com/cognusion/go-recyclable/tree/master/v2/bufferpool.go?s=709:744#L30)
``` go
func (p *BufferPool) Put(b *Buffer)
```
Put returns a Buffer to the pool. Generally calling Buffer.Close() is
preferable to calling this directly.




## <a name="Error">type</a> [Error](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=4324:4341#L76)
``` go
type Error string
```
Error is an error type










### <a name="Error.Error">func</a> (Error) [Error](https://github.com/cognusion/go-recyclable/tree/master/v2/buffer.go?s=4393:4422#L79)
``` go
func (e Error) Error() string
```
Error returns the stringified version of Error








- - -
Generated by [godoc2md](http://github.com/cognusion/godoc2md)
