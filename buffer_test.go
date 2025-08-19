package recyclable

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// HOWTO implement a goro-safe BufferPool for Buffers
func Example() {

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

	/* HINTS:
	* Makes awesome ``http.Request.Body``s, especially since they get automatically ``.Close()``d when done with
	* Replaces ``bytes.Buffer`` and ``bytes.Reader`` for most uses
	* Isa Stringer and an error
	* As a Writer and a Reader can be used in pipes and elsewhere
	  * You can also pipe them to themselves, but that is a very bad idea unless you love watching OOMs
	*/

}

func TestBuffer(t *testing.T) {

	rPool := NewBufferPool()

	Convey("When a Buffer is fetched from a BufferPool, it is a Buffer", t, func() {
		rbx := rPool.Get()
		So(rbx, ShouldHaveSameTypeAs, &Buffer{})

		rb := rPool.Get()
		Convey("And setting it appears correct", func() {
			rb.Reset([]byte("Hello World"))
			So(rb.Bytes(), ShouldResemble, []byte("Hello World"))
			So(rb.Error(), ShouldEqual, "Hello World")
			So(rb.String(), ShouldEqual, "Hello World")

			Convey("And re-reading from it multiple times works too", func() {
				for range 10 {
					So(rb.Bytes(), ShouldResemble, []byte("Hello World"))
				}
			})

			Convey("Appending to it as an io.Writer works as well", func() {
				n, err := io.WriteString(rb, ", nice day?")
				So(n, ShouldBeGreaterThan, 0)
				So(err, ShouldBeNil)
				So(rb.Bytes(), ShouldResemble, []byte("Hello World, nice day?"))
			})
		})

		Convey("Resetting it using an io.Reader works as expected", func() {
			buff := bytes.NewBufferString("Hola Mundo")
			rb.ResetFromReader(buff)
			So(rb.Bytes(), ShouldResemble, []byte("Hola Mundo"))

			Convey("... and checking the string value is similarly correct", func() {
				So(rb.String(), ShouldEqual, "Hola Mundo")
			})
		})

		Convey("Resetting it, but limited, using an io.Reader works as expected", func() {
			buff := bytes.NewBufferString("Oh My Gosh")
			err := rb.ResetFromLimitedReader(buff, 20)
			So(err, ShouldBeNil)
			So(rb.Bytes(), ShouldResemble, []byte("Oh My Gosh"))

			Convey("... and when it's over the limit, that is handled as expected", func() {
				buff2 := bytes.NewBufferString("This is a long sentence")
				err := rb.ResetFromLimitedReader(buff2, 4)
				So(err, ShouldEqual, ErrTooLarge)
				So(rb.Bytes(), ShouldResemble, []byte("This"))
			})
		})

		Convey("Putting it back in the pool doesn't freak out", func() {
			So(func() { rb.Close() }, ShouldNotPanic)

			Convey("... doing it twice doesn't either (but don't ever do that, ever)...", func() {
				So(func() { rb.Close() }, ShouldNotPanic)
			})
		})
	})

	Convey("Using it as a WriterAt, with concurrent writes, works as expected.", t, func(c C) {
		buff := &Buffer{}

		// longSize is large to force the underlying buffer to need reallocation, so we
		// confirm it works as expected.
		longSize := 100

		wg := sync.WaitGroup{}

		alphabet := []byte("abcdefghijklmnopqrst")
		var longAlphabet = make([]byte, 0)
		for range longSize {
			longAlphabet = append(longAlphabet, alphabet...)
		}

		// Write lowercase letters a-t into the buffer, individually, concurrently.
		for i := 97; i < 117; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				b := make([]byte, 1)
				b[0] = byte(i)
				_, err := buff.WriteAt(b, int64(i-97))
				c.So(err, ShouldBeNil)
			}(i)
		}
		wg.Wait()
		c.So(buff.Bytes(), ShouldResemble, alphabet)

		// Write the lowercase letters a-t as a set, longSize number of times, sequentially,
		// to test reallocation.
		for i := range longSize {
			_, err := buff.WriteAt(alphabet, int64(i*20))
			c.So(err, ShouldBeNil)
		}
		c.So(buff.Bytes(), ShouldResemble, longAlphabet)
	})

}

func Test_BufferPointlessClose(t *testing.T) {
	Convey("When a Buffer is created outside of a Pool, and it is closed, it doesn't panic and returns the proper error", t, func() {
		b := &Buffer{}
		So(b.Close(), ShouldEqual, ErrPointlessClose)
	})
}

func Test_BufferInterfacesOhMy(t *testing.T) {
	Convey("When a Buffer it type checked against various interfaces, it passes", t, FailureContinues, func() {
		b := &Buffer{}
		So(b, ShouldImplement, (*io.Reader)(nil))
		So(b, ShouldImplement, (*io.ReadCloser)(nil))
		So(b, ShouldImplement, (*io.ReaderAt)(nil))
		So(b, ShouldImplement, (*io.ReadSeeker)(nil))
		So(b, ShouldImplement, (*io.ReadSeekCloser)(nil))
		So(b, ShouldImplement, (*io.Writer)(nil))
		So(b, ShouldImplement, (*io.WriterAt)(nil))
		So(b, ShouldImplement, (*io.WriteCloser)(nil))
		So(b, ShouldImplement, (*io.WriterTo)(nil))
		So(b, ShouldImplement, (*io.Seeker)(nil))
		So(b, ShouldImplement, (*io.ByteScanner)(nil))
		So(b, ShouldImplement, (*io.RuneScanner)(nil))
		So(b, ShouldImplement, (*error)(nil))
		So(b, ShouldImplement, (*fmt.Stringer)(nil))
	})
}

// Tests grabbing an new RB, using the NewFunc directly
func BenchmarkRBNewRaw(b *testing.B) {

	bp := NewBufferPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb := NewBuffer(bp, make([]byte, 0))
		//rb.Close()
		rb.Len()
	}
}

// Tests grabbing an new RB, using Pool.Get, and a pre-seeded BytePool to feed from in the NewFunc
func BenchmarkRBNewGet(b *testing.B) {

	bp := NewBufferPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb := bp.Get()
		rb.Close()
	}
}

// Tests grabbing an new RB, using Pool.Get, but make([]byte) in the NewFunc
func BenchmarkRBNewMake(b *testing.B) {

	bp := NewBufferPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb := bp.Get()
		//rb.Close()
		rb.Len()
	}
}

// Test grabbing an new RB, using Pool.Get, and Reseting using a premade fixed []byte
func BenchmarkRBNewGetResetFixed(b *testing.B) {
	var empty = make([]byte, 0)

	bp := NewBufferPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb := bp.Get()
		rb.Reset(empty)
		rb.Close()
	}
}

// Tests grabbing an new RB, using Pool.Get, and Reseting using make([]byte) every time
func BenchmarkRBNewGetResetMake(b *testing.B) {
	bp := NewBufferPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb := bp.Get()
		rb.Reset(make([]byte, 0))
		rb.Close()
	}
}

func BenchmarkBytes(b *testing.B) {
	r := NewBuffer(nil, make([]byte, 0))
	r.Reset([]byte("Hello World"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Bytes()
	}
}

func BenchmarkReset(b *testing.B) {
	r := NewBuffer(nil, make([]byte, 0))
	r.Reset([]byte("Hello World"))

	var ok = []byte("ok")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Reset(ok)
	}
}

func BenchmarkWrite(b *testing.B) {
	r := NewBuffer(nil, make([]byte, 0))
	r.Reset([]byte("Hello World"))

	var ok = []byte("ok")
	var hello = []byte("Hello World")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Write(ok)
		r.Reset(hello)
	}
}
