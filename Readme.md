
# recyclable
`import "github.com/cognusion/go-recyclable"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)
* [Examples](#pkg-examples)

## <a name="pkg-overview">Overview</a>
Package recyclable provides the recyclable.BufferPool, which is a never-ending font of
recyclable.Buffer, a multiuse buffer that very reusable, supports re-reading the contained
buffer, and when Close()d, will return home to its BufferPool for reuse.


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
for i := 0; i < 10; i++ {
  if string(rb.Bytes()) != "Hello World" {
    panic("OMG! Can't reread?!!!")
  }
}

// Or get the string value, if you prefer (and know it's safe)
for i := 0; i < 10; i++ {
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

/* HINTS:
* Makes awesome ``http.Request.Body``s, especially since they get automatically ``.Close()``d when done with
* Replaces ``bytes.Buffer`` and ``bytes.Reader`` for most uses
* Isa Stringer and an error
* As a Writer and a Reader can be used in pipes and elsewhere
  * You can also pipe them to themselves, but that is a very bad idea unless you love watching OOMs
*/
```



## <a name="pkg-index">Index</a>
* [Variables](#pkg-variables)
* [type Buffer](#Buffer)
  * [func NewBuffer(home *BufferPool, bytes []byte) *Buffer](#NewBuffer)
  * [func (r *Buffer) Bytes() []byte](#Buffer.Bytes)
  * [func (r *Buffer) Close() error](#Buffer.Close)
  * [func (r *Buffer) Error() string](#Buffer.Error)
  * [func (r *Buffer) ResetFromLimitedReader(reader io.Reader, max int64) error](#Buffer.ResetFromLimitedReader)
  * [func (r *Buffer) ResetFromReader(reader io.Reader)](#Buffer.ResetFromReader)
  * [func (r *Buffer) String() string](#Buffer.String)
  * [func (r *Buffer) Write(p []byte) (n int, err error)](#Buffer.Write)
* [type BufferPool](#BufferPool)
  * [func NewBufferPool() *BufferPool](#NewBufferPool)
  * [func (p *BufferPool) Get() *Buffer](#BufferPool.Get)
  * [func (p *BufferPool) Put(b *Buffer)](#BufferPool.Put)

#### <a name="pkg-examples">Examples</a>
* [Package](#example-)

#### <a name="pkg-files">Package files</a>
[buffer.go](https://github.com/cognusion/go-recyclable/tree/master/buffer.go) [bufferpool.go](https://github.com/cognusion/go-recyclable/tree/master/bufferpool.go)



## <a name="pkg-variables">Variables</a>
``` go
var ErrTooLarge = errors.New("read byte count too large")
```
ErrTooLarge is returned when ResetFromLimitedReader is used and the supplied Reader writes too much




## <a name="Buffer">type</a> [Buffer](https://github.com/cognusion/go-recyclable/tree/master/buffer.go?s=782:837#L21)
``` go
type Buffer struct {
    bytes.Reader
    // contains filtered or unexported fields
}

```
Buffer is an io.Reader, io.ReadCloser, io.ReaderAt, io.Writer,
io.WriteCloser, io.WriterTo, io.Seeker, io.ByteScanner,
io.RuneScanner, and more!
It's designed to work in coordination with a BufferPool for
recycling, and it's `.Close()` method puts itself back in
the Pool it came from







### <a name="NewBuffer">func</a> [NewBuffer](https://github.com/cognusion/go-recyclable/tree/master/buffer.go?s=967:1021#L29)
``` go
func NewBuffer(home *BufferPool, bytes []byte) *Buffer
```
NewBuffer returns a Buffer with a proper home. Generally calling
BufferPool.Get() is preferable to calling this directly.





### <a name="Buffer.Bytes">func</a> (\*Buffer) [Bytes](https://github.com/cognusion/go-recyclable/tree/master/buffer.go?s=2131:2162#L64)
``` go
func (r *Buffer) Bytes() []byte
```
Bytes returns the contents of the buffer, and sets the seek pointer back to the beginning




### <a name="Buffer.Close">func</a> (\*Buffer) [Close](https://github.com/cognusion/go-recyclable/tree/master/buffer.go?s=1261:1291#L38)
``` go
func (r *Buffer) Close() error
```
Close puts itself back in the Pool it came from. This should absolutely **never** be
called more than once per Buffer life.
Implements `io.Closer` (also `io.ReadCloser` and `io.WriteCloser`)




### <a name="Buffer.Error">func</a> (\*Buffer) [Error](https://github.com/cognusion/go-recyclable/tree/master/buffer.go?s=2550:2581#L78)
``` go
func (r *Buffer) Error() string
```
Error returns the contents of the buffer as a string. Implements “error“




### <a name="Buffer.ResetFromLimitedReader">func</a> (\*Buffer) [ResetFromLimitedReader](https://github.com/cognusion/go-recyclable/tree/master/buffer.go?s=1803:1877#L52)
``` go
func (r *Buffer) ResetFromLimitedReader(reader io.Reader, max int64) error
```
ResetFromLimitedReader performs a Reset() using the contents of the supplied Reader as the new content,
up to at most max bytes, returning ErrTooLarge if it's over. The error is not terminal, and the buffer
may continue to be used, understanding the contents will be limited




### <a name="Buffer.ResetFromReader">func</a> (\*Buffer) [ResetFromReader](https://github.com/cognusion/go-recyclable/tree/master/buffer.go?s=1423:1473#L44)
``` go
func (r *Buffer) ResetFromReader(reader io.Reader)
```
ResetFromReader performs a Reset() using the contents of the supplied Reader as the new content




### <a name="Buffer.String">func</a> (\*Buffer) [String](https://github.com/cognusion/go-recyclable/tree/master/buffer.go?s=2349:2381#L71)
``` go
func (r *Buffer) String() string
```
String returns the contents of the buffer as a string, and sets the seek pointer back to the beginning




### <a name="Buffer.Write">func</a> (\*Buffer) [Write](https://github.com/cognusion/go-recyclable/tree/master/buffer.go?s=2685:2736#L83)
``` go
func (r *Buffer) Write(p []byte) (n int, err error)
```
Writer adds the bytes the written to the buffer. Implements “io.Writer“




## <a name="BufferPool">type</a> [BufferPool](https://github.com/cognusion/go-recyclable/tree/master/bufferpool.go?s=84:126#L6)
``` go
type BufferPool struct {
    // contains filtered or unexported fields
}

```
BufferPool is a self-managing pool of Buffers







### <a name="NewBufferPool">func</a> [NewBufferPool](https://github.com/cognusion/go-recyclable/tree/master/bufferpool.go?s=179:211#L11)
``` go
func NewBufferPool() *BufferPool
```
NewBufferPool returns an initialized BufferPool





### <a name="BufferPool.Get">func</a> (\*BufferPool) [Get](https://github.com/cognusion/go-recyclable/tree/master/bufferpool.go?s=524:558#L24)
``` go
func (p *BufferPool) Get() *Buffer
```
Get will return an existing Buffer or a new one if the pool is empty.
REMEMBER to Reset the Buffer and don't just start using it, as it
may very well have old data in it!




### <a name="BufferPool.Put">func</a> (\*BufferPool) [Put](https://github.com/cognusion/go-recyclable/tree/master/bufferpool.go?s=708:743#L30)
``` go
func (p *BufferPool) Put(b *Buffer)
```
Put returns a Buffer to the pool. Generally calling Buffer.Close() is
preferable to calling this directly.








- - -
Generated by [godoc2md](http://godoc.org/github.com/cognusion/godoc2md)
