package recyclable

// These tests were adapted from the "bytes" package for Reader or Buffer.

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"sync"
	"testing"
)

const N = 10000 // make this bigger for a larger (and slower) test

var (
	testString string // test data for write tests
	testBytes  []byte // test data; same as testString but as a slice.
)

// negativeReader ...
type negativeReader struct{}

func (r *negativeReader) Read([]byte) (int, error) { return -1, nil }

// panicReader ...
type panicReader struct{ panic bool }

func (r panicReader) Read(p []byte) (int, error) {
	if r.panic {
		panic("oops")
	}
	return 0, io.EOF
}

func init() {
	testBytes = make([]byte, N)
	for i := range N {
		testBytes[i] = 'a' + byte(i%26)
	}
	testString = string(testBytes)
}

func equal(a, b []byte) bool {
	return string(a) == string(b)
}

// Verify that contents of buf match the string s.
func check(t *testing.T, testname string, buf *Buffer, s string) {
	bytes := buf.Bytes()
	str := buf.String()
	if buf.Len() != len(bytes) {
		t.Errorf("%s: buf.Len() == %d, len(buf.Bytes()) == %d", testname, buf.Len(), len(bytes))
	}

	if buf.Len() != len(str) {
		t.Errorf("%s: buf.Len() == %d, len(buf.String()) == %d", testname, buf.Len(), len(str))
	}

	if buf.Len() != len(s) {
		t.Errorf("%s: buf.Len() == %d, len(s) == %d", testname, buf.Len(), len(s))
	}

	if string(bytes) != s {
		t.Errorf("%s: string(buf.Bytes()) == %q, s == %q", testname, string(bytes), s)
	}
}

// Fill buf through n writes of string fus.
// The initial contents of buf corresponds to the string s;
// the result is the final contents of buf returned as a string.
func fillString(t *testing.T, testname string, buf *Buffer, s string, n int, fus string) string {
	check(t, testname+" (fill 1)", buf, s)
	for ; n > 0; n-- {
		m, err := buf.WriteString(fus)
		if m != len(fus) {
			t.Errorf(testname+" (fill 2): m == %d, expected %d", m, len(fus))
		}
		if err != nil {
			t.Errorf(testname+" (fill 3): err should always be nil, found err == %s", err)
		}
		s += fus
		check(t, testname+" (fill 4)", buf, s)
	}
	return s
}

// Fill buf through n writes of byte slice fub.
// The initial contents of buf corresponds to the string s;
// the result is the final contents of buf returned as a string.
func fillBytes(t *testing.T, testname string, buf *Buffer, s string, n int, fub []byte) string {
	check(t, testname+" (fill 1)", buf, s)
	for ; n > 0; n-- {
		m, err := buf.Write(fub)
		if m != len(fub) {
			t.Errorf(testname+" (fill 2): m == %d, expected %d", m, len(fub))
		}
		if err != nil {
			t.Errorf(testname+" (fill 3): err should always be nil, found err == %s", err)
		}
		s += string(fub)
		check(t, testname+" (fill 4)", buf, s)
	}
	return s
}

// Empty buf through repeated reads into fub.
// The initial contents of buf corresponds to the string s.
func empty(t *testing.T, testname string, buf *Buffer, s string, fub []byte) {
	check(t, testname+" (empty 1)", buf, s)

	for {
		n, err := buf.Read(fub)
		if n == 0 {
			break
		}
		if err != nil {
			t.Errorf(testname+" (empty 2): err should always be nil, found err == %s", err)
		}
		s = s[n:]
		check(t, testname+" (empty 3)", buf, s)
	}

	check(t, testname+" (empty 4)", buf, "")
}

func TestReader(t *testing.T) {
	r := NewBuffer(nil, []byte("0123456789"))
	tests := []struct {
		off     int64
		seek    int
		n       int
		want    string
		wantpos int64
		readerr error
		seekerr string
	}{
		{seek: io.SeekStart, off: 0, n: 20, want: "0123456789"},
		{seek: io.SeekStart, off: 1, n: 1, want: "1"},
		{seek: io.SeekCurrent, off: 1, wantpos: 3, n: 2, want: "34"},
		{seek: io.SeekStart, off: -1, seekerr: ErrNegativeOffset.Error()},
		{seek: io.SeekStart, off: 1 << 33, wantpos: 1 << 33, readerr: io.EOF},
		{seek: io.SeekCurrent, off: 1, wantpos: 1<<33 + 1, readerr: io.EOF},
		{seek: io.SeekStart, n: 5, want: "01234"},
		{seek: io.SeekCurrent, n: 5, want: "56789"},
		{seek: io.SeekEnd, off: -1, n: 1, wantpos: 9, want: "9"},
	}

	for i, tt := range tests {
		pos, err := r.Seek(tt.off, tt.seek)
		if err == nil && tt.seekerr != "" {
			t.Errorf("%d. want seek error %q", i, tt.seekerr)
			continue
		}
		if err != nil && err != ErrNegativeOffset {
			t.Errorf("%d. seek error = %q; want %q", i, err.Error(), tt.seekerr)
			continue
		}
		if tt.wantpos != 0 && tt.wantpos != pos {
			t.Errorf("%d. pos = %d, want %d", i, pos, tt.wantpos)
		}
		buf := make([]byte, tt.n)
		n, err := r.Read(buf)
		if err != tt.readerr {
			t.Errorf("%d. read = %v; want %v", i, err, tt.readerr)
			continue
		}
		got := string(buf[:n])
		if got != tt.want {
			t.Errorf("%d. got %q; want %q", i, got, tt.want)
		}
	}
}

func TestReadAfterBigSeek(t *testing.T) {
	r := NewBuffer(nil, []byte("0123456789"))
	if _, err := r.Seek(1<<31+5, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if n, err := r.Read(make([]byte, 10)); n != 0 || err != io.EOF {
		t.Errorf("Read = %d, %v; want 0, EOF", n, err)
	}
}

func TestReaderAt(t *testing.T) {
	r := NewBuffer(nil, []byte("0123456789"))
	tests := []struct {
		off     int64
		n       int
		want    string
		wanterr any
	}{
		{0, 10, "0123456789", nil},
		{1, 10, "123456789", io.EOF},
		{1, 9, "123456789", nil},
		{11, 10, "", io.EOF},
		{0, 0, "", nil},
		{-1, 0, "", ErrNegativeOffset.Error()},
	}
	for i, tt := range tests {
		b := make([]byte, tt.n)
		rn, err := r.ReadAt(b, tt.off)
		got := string(b[:rn])
		if got != tt.want {
			t.Errorf("%d. got %q; want %q", i, got, tt.want)
		}
		if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wanterr) {
			t.Errorf("%d. got error = %v; want %v", i, err, tt.wanterr)
		}
	}
}

func TestReaderAtConcurrent(t *testing.T) {
	// Test for the race detector, to verify ReadAt doesn't mutate
	// any state.
	r := NewBuffer(nil, []byte("0123456789"))
	var wg sync.WaitGroup
	for i := range 5 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var buf [1]byte
			r.ReadAt(buf[:], int64(i))
		}(i)
	}
	wg.Wait()
}

func TestEmptyReaderConcurrent(t *testing.T) {
	// Test for the race detector, to verify a Read that doesn't yield any bytes
	// is okay to use from multiple goroutines. This was our historic behavior.
	// See golang.org/issue/7856
	r := NewBuffer(nil, []byte{})
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			var buf [1]byte
			r.Read(buf[:])
		}()
		go func() {
			defer wg.Done()
			r.Read(nil)
		}()
	}
	wg.Wait()
}

func TestReaderWriteTo(t *testing.T) {
	for i := 0; i < 30; i += 3 {
		var l int
		if i > 0 {
			l = len(testString) / i
		}
		s := testString[:l]
		r := NewBuffer(nil, testBytes[:l])
		var b Buffer
		n, err := r.WriteTo(&b)
		if expect := int64(len(s)); n != expect {
			t.Errorf("got %v; want %v", n, expect)
		}
		if err != nil {
			t.Errorf("for length %d: got error = %v; want nil", l, err)
		}
		if b.String() != s {
			t.Errorf("got string %q; want %q", b.String(), s)
		}
		if r.Len() != 0 {
			t.Errorf("reader contains %v bytes; want 0", r.Len())
		}
	}
}

func TestReaderLen(t *testing.T) {
	const data = "hello world"
	r := NewBuffer(nil, []byte(data))
	if got, want := r.Len(), 11; got != want {
		t.Errorf("r.Len(): got %d, want %d", got, want)
	}
	if n, err := r.Read(make([]byte, 10)); err != nil || n != 10 {
		t.Errorf("Read failed: read %d %v", n, err)
	}
	if got, want := r.Len(), 1; got != want {
		t.Errorf("r.Len(): got %d, want %d", got, want)
	}
	if n, err := r.Read(make([]byte, 1)); err != nil || n != 1 {
		t.Errorf("Read failed: read %d %v; want 1, nil", n, err)
	}
	if got, want := r.Len(), 0; got != want {
		t.Errorf("r.Len(): got %d, want %d", got, want)
	}
}

// verify that copying from an empty reader always has the same results,
// regardless of the presence of a WriteTo method.
func TestReaderCopyNothing(t *testing.T) {
	type nErr struct {
		n   int64
		err error
	}
	type justReader struct {
		io.Reader
	}
	type justWriter struct {
		io.Writer
	}
	discard := justWriter{io.Discard} // hide ReadFrom

	var with, withOut nErr
	with.n, with.err = io.Copy(discard, NewBuffer(nil, nil))
	withOut.n, withOut.err = io.Copy(discard, justReader{NewBuffer(nil, nil)})
	if with != withOut {
		t.Errorf("behavior differs: with = %#v; without: %#v", with, withOut)
	}
}

// tests that Len is affected by reads, but Size is not.
func TestReaderLenSize(t *testing.T) {
	r := NewBuffer(nil, []byte("abc"))
	io.CopyN(io.Discard, r, 1)
	if r.Len() != 2 {
		t.Errorf("Len = %d; want 2", r.Len())
	}
	if r.Size() != 3 {
		t.Errorf("Size = %d; want 3", r.Size())
	}
}

func TestReaderZero(t *testing.T) {
	if l := (&Buffer{}).Len(); l != 0 {
		t.Errorf("Len: got %d, want 0", l)
	}

	if n, err := (&Buffer{}).Read(nil); n != 0 || err != io.EOF {
		t.Errorf("Read: got %d, %v; want 0, io.EOF", n, err)
	}

	if n, err := (&Buffer{}).ReadAt(nil, 11); n != 0 || err != io.EOF {
		t.Errorf("ReadAt: got %d, %v; want 0, io.EOF", n, err)
	}

	if b, err := (&Buffer{}).ReadByte(); b != 0 || err != io.EOF {
		t.Errorf("ReadByte: got %d, %v; want 0, io.EOF", b, err)
	}

	if offset, err := (&Buffer{}).Seek(11, io.SeekStart); offset != 11 || err != nil {
		t.Errorf("Seek: got %d, %v; want 11, nil", offset, err)
	}

	if s := (&Buffer{}).Size(); s != 0 {
		t.Errorf("Size: got %d, want 0", s)
	}

	if (&Buffer{}).UnreadByte() == nil {
		t.Errorf("UnreadByte: got nil, want error")
	}

	if n, err := (&Buffer{}).WriteTo(io.Discard); n != 0 || err != nil {
		t.Errorf("WriteTo: got %d, %v; want 0, nil", n, err)
	}
}

//********************************************

func TestNewBuffer(t *testing.T) {
	buf := NewBuffer(nil, testBytes)
	check(t, "NewBuffer", buf, testString)
}

// Calling NewBuffer and immediately shallow copying the Buffer struct
// should not result in any allocations.
// This can be used to reset the underlying []byte of an existing Buffer.
func TestNewBufferShallow(t *testing.T) {
	var buf Buffer

	n := testing.AllocsPerRun(1000, func() {
		buf = *NewBuffer(nil, testBytes)
	})
	if n > 0 {
		t.Errorf("allocations occurred while shallow copying")
	}
	check(t, "NewBuffer", &buf, testString)
}

func TestNewBufferString(t *testing.T) {
	buf := NewBufferString(nil, testString)
	check(t, "NewBufferString", buf, testString)
}

func TestBasicOperations(t *testing.T) {
	var buf Buffer

	for range 5 {
		check(t, "TestBasicOperations (1)", &buf, "")

		buf.Empty()
		check(t, "TestBasicOperations (2)", &buf, "")

		buf.Truncate(0)
		check(t, "TestBasicOperations (3)", &buf, "")

		n, err := buf.Write(testBytes[0:1])
		if want := 1; err != nil || n != want {
			t.Errorf("Write: got (%d, %v), want (%d, %v)", n, err, want, nil)
		}
		check(t, "TestBasicOperations (4)", &buf, "a")

		buf.WriteByte(testString[1])
		check(t, "TestBasicOperations (5)", &buf, "ab")

		n, err = buf.Write(testBytes[2:26])
		if want := 24; err != nil || n != want {
			t.Errorf("Write: got (%d, %v), want (%d, %v)", n, err, want, nil)
		}
		check(t, "TestBasicOperations (6)", &buf, testString[0:26])

		buf.Truncate(26)
		check(t, "TestBasicOperations (7)", &buf, testString[0:26])

		buf.Truncate(20)
		check(t, "TestBasicOperations (8)", &buf, testString[0:20])

		empty(t, "TestBasicOperations (9)", &buf, testString[0:20], make([]byte, 5))
		empty(t, "TestBasicOperations (10)", &buf, "", make([]byte, 100))

		buf.WriteByte(testString[1])
		c, err := buf.ReadByte()
		if want := testString[1]; err != nil || c != want {
			t.Errorf("ReadByte: got (%q, %v), want (%q, %v)", c, err, want, nil)
		}
		c, err = buf.ReadByte()
		if err != io.EOF {
			t.Errorf("ReadByte: got (%q, %v), want (%q, %v)", c, err, byte(0), io.EOF)
		}
	}
}

func TestLargeStringWrites(t *testing.T) {
	var buf Buffer
	limit := 30
	if testing.Short() {
		limit = 9
	}
	for i := 3; i < limit; i += 3 {
		s := fillString(t, "TestLargeWrites (1)", &buf, "", 5, testString)
		empty(t, "TestLargeStringWrites (2)", &buf, s, make([]byte, len(testString)/i))
	}
	check(t, "TestLargeStringWrites (3)", &buf, "")
}

func TestLargeByteWrites(t *testing.T) {
	var buf Buffer
	limit := 30
	if testing.Short() {
		limit = 9
	}
	for i := 3; i < limit; i += 3 {
		s := fillBytes(t, "TestLargeWrites (1)", &buf, "", 5, testBytes)
		empty(t, "TestLargeByteWrites (2)", &buf, s, make([]byte, len(testString)/i))
	}
	check(t, "TestLargeByteWrites (3)", &buf, "")
}

func TestLargeStringReads(t *testing.T) {
	var buf Buffer
	for i := 3; i < 30; i += 3 {
		s := fillString(t, "TestLargeReads (1)", &buf, "", 5, testString[:len(testString)/i])
		empty(t, "TestLargeReads (2)", &buf, s, make([]byte, len(testString)))
	}
	check(t, "TestLargeStringReads (3)", &buf, "")
}

func TestLargeByteReads(t *testing.T) {
	var buf Buffer
	for i := 3; i < 30; i += 3 {
		s := fillBytes(t, "TestLargeReads (1)", &buf, "", 5, testBytes[:len(testBytes)/i])
		empty(t, "TestLargeReads (2)", &buf, s, make([]byte, len(testString)))
	}
	check(t, "TestLargeByteReads (3)", &buf, "")
}

func TestMixedReadsAndWrites(t *testing.T) {
	var buf Buffer
	s := ""
	for i := range 50 {
		wlen := rand.Intn(len(testString))
		if i%2 == 0 {
			s = fillString(t, "TestMixedReadsAndWrites (1)", &buf, s, 1, testString[0:wlen])
		} else {
			s = fillBytes(t, "TestMixedReadsAndWrites (1)", &buf, s, 1, testBytes[0:wlen])
		}

		rlen := rand.Intn(len(testString))
		fub := make([]byte, rlen)
		n, _ := buf.Read(fub)
		s = s[n:]
	}
	empty(t, "TestMixedReadsAndWrites (2)", &buf, s, make([]byte, buf.Len()))
}

func TestCapWithPreallocatedSlice(t *testing.T) {
	buf := NewBuffer(nil, make([]byte, 10))
	n := buf.Cap()
	if n != 10 {
		t.Errorf("expected 10, got %d", n)
	}
}

func TestCapWithSliceAndWrittenData(t *testing.T) {
	buf := NewBuffer(nil, make([]byte, 0, 10))
	buf.Write([]byte("test"))
	n := buf.Cap()
	if n != 10 {
		t.Errorf("expected 10, got %d", n)
	}
}

func TestReadFrom(t *testing.T) {
	var buf Buffer
	for i := 3; i < 30; i += 3 {
		s := fillBytes(t, "TestReadFrom (1)", &buf, "", 5, testBytes[:len(testBytes)/i])
		var b Buffer
		b.ReadFrom(&buf)
		empty(t, "TestReadFrom (2)", &b, s, make([]byte, len(testString)))
	}
}

// Make sure that an empty Buffer remains empty when
// it is "grown" before a Read that panics
func TestReadFromPanicReader(t *testing.T) {

	// First verify non-panic behaviour
	var buf Buffer
	i, err := buf.ReadFrom(panicReader{})
	if err != nil {
		t.Fatal(err)
	}
	if i != 0 {
		t.Fatalf("unexpected return from bytes.ReadFrom (1): got: %d, want %d", i, 0)
	}
	check(t, "TestReadFromPanicReader (1)", &buf, "")

	// Confirm that when Reader panics, the empty buffer remains empty
	var buf2 Buffer
	defer func() {
		recover()
		check(t, "TestReadFromPanicReader (2)", &buf2, "")
	}()
	buf2.ReadFrom(panicReader{panic: true})
}

func TestReadFromNegativeReader(t *testing.T) {
	var b Buffer
	defer func() {
		switch err := recover().(type) {
		case nil:
			t.Fatal("bytes.Buffer.ReadFrom didn't panic")
		case error:
			// this is the error string of errNegativeRead
			wantError := ErrNegativePosition.Error()
			if err.Error() != wantError {
				t.Fatalf("recovered panic: got %v, want %v", err.Error(), wantError)
			}
		default:
			t.Fatalf("unexpected panic value: %#v", err)
		}
	}()

	b.ReadFrom(new(negativeReader))
}

func TestWriteTo(t *testing.T) {
	var buf Buffer
	for i := 3; i < 30; i += 3 {
		s := fillBytes(t, "TestWriteTo (1)", &buf, "", 5, testBytes[:len(testBytes)/i])
		var b Buffer
		buf.WriteTo(&b)
		empty(t, "TestWriteTo (2)", &b, s, make([]byte, len(testString)))
	}
}

func TestWriteAppend(t *testing.T) {
	var got Buffer
	var want []byte
	for i := range 1000 {
		b := got.AvailableBuffer()
		b = strconv.AppendInt(b, int64(i), 10)
		want = strconv.AppendInt(want, int64(i), 10)
		got.Write(b)
	}
	if !equal(got.Bytes(), want) {
		t.Fatalf("Bytes() = %q, want %q", &got, want)
	}

	// With a sufficiently sized buffer, there should be no allocations.
	n := testing.AllocsPerRun(100, func() {
		got.Empty()
		for i := range 1000 {
			b := got.AvailableBuffer()
			b = strconv.AppendInt(b, int64(i), 10)
			got.Write(b)
		}
	})
	if n > 0 {
		t.Errorf("allocations occurred while appending")
	}
}

func TestNext(t *testing.T) {
	b := []byte{0, 1, 2, 3, 4}
	tmp := make([]byte, 5)
	for i := 0; i <= 5; i++ {
		for j := i; j <= 5; j++ {
			for k := 0; k <= 6; k++ {
				// 0 <= i <= j <= 5; 0 <= k <= 6
				// Check that if we start with a buffer
				// of length j at offset i and ask for
				// Next(k), we get the right bytes.
				buf := NewBuffer(nil, b[0:j])
				n, _ := buf.Read(tmp[0:i])
				if n != i {
					t.Fatalf("Read %d returned %d", i, n)
				}
				bb := buf.Next(k)
				want := k
				if want > j-i {
					want = j - i
				}
				if len(bb) != want {
					t.Fatalf("in %d,%d: len(Next(%d)) == %d", i, j, k, len(bb))
				}
				for l, v := range bb {
					if v != byte(l+i) {
						t.Fatalf("in %d,%d: Next(%d)[%d] = %d, want %d", i, j, k, l, v, l+i)
					}
				}
			}
		}
	}
}

var readBytesTests = []struct {
	buffer   string
	delim    byte
	expected []string
	err      error
}{
	{"", 0, []string{""}, io.EOF},
	{"a\x00", 0, []string{"a\x00"}, nil},
	{"abbbaaaba", 'b', []string{"ab", "b", "b", "aaab"}, nil},
	{"hello\x01world", 1, []string{"hello\x01"}, nil},
	{"foo\nbar", 0, []string{"foo\nbar"}, io.EOF},
	{"alpha\nbeta\ngamma\n", '\n', []string{"alpha\n", "beta\n", "gamma\n"}, nil},
	{"alpha\nbeta\ngamma", '\n', []string{"alpha\n", "beta\n", "gamma"}, io.EOF},
}

func TestReadBytes(t *testing.T) {
	for _, test := range readBytesTests {
		buf := NewBufferString(nil, test.buffer)
		var err error
		for _, expected := range test.expected {
			var bytes []byte
			bytes, err = buf.ReadBytes(test.delim)
			if string(bytes) != expected {
				t.Errorf("expected %q, got %q", expected, bytes)
			}
			if err != nil {
				break
			}
		}
		if err != test.err {
			t.Errorf("expected error %v, got %v", test.err, err)
		}
	}
}

func TestReadString(t *testing.T) {
	for _, test := range readBytesTests {
		buf := NewBufferString(nil, test.buffer)
		var err error
		for _, expected := range test.expected {
			var s string
			s, err = buf.ReadString(test.delim)
			if s != expected {
				t.Errorf("expected %q, got %q", expected, s)
			}
			if err != nil {
				break
			}
		}
		if err != test.err {
			t.Errorf("expected error %v, got %v", test.err, err)
		}
	}
}

func TestGrow(t *testing.T) {
	x := []byte{'x'}
	y := []byte{'y'}
	tmp := make([]byte, 72)
	for _, growLen := range []int{0, 100, 1000, 10000, 100000} {
		for _, startLen := range []int{0, 100, 1000, 10000, 100000} {
			xBytes := bytes.Repeat(x, startLen)

			buf := NewBuffer(nil, xBytes)
			// If we read, this affects buf.off, which is good to test.
			readBytes, _ := buf.Read(tmp)
			yBytes := bytes.Repeat(y, growLen)
			allocs := testing.AllocsPerRun(100, func() {
				buf.Grow(growLen)
				buf.Write(yBytes)
			})
			// Check no allocation occurs in write, as long as we're single-threaded.
			if allocs != 0 {
				t.Errorf("allocation occurred during write")
			}
			// Check that buffer has correct data.
			if !equal(buf.Bytes()[0:startLen-readBytes], xBytes[readBytes:]) {
				t.Errorf("bad initial data at %d %d", startLen, growLen)
			}
			if !equal(buf.Bytes()[startLen-readBytes:startLen-readBytes+growLen], yBytes) {
				t.Errorf("bad written data at %d %d", startLen, growLen)
			}
		}
	}
}

func TestGrowOverflow(t *testing.T) {
	defer func() {
		if err := recover(); err != ErrTooLarge {
			t.Errorf("after too-large Grow, recover() = %v; want %v", err, ErrTooLarge)
		}
	}()

	buf := NewBuffer(nil, make([]byte, 1))
	const maxInt = int(^uint(0) >> 1)
	buf.Grow(maxInt)
}

func TestUnreadByte(t *testing.T) {
	b := new(Buffer)

	// check at EOF
	if err := b.UnreadByte(); err == nil {
		t.Fatal("UnreadByte at EOF: got no error")
	}
	if _, err := b.ReadByte(); err == nil {
		t.Fatal("ReadByte at EOF: got no error")
	}
	if err := b.UnreadByte(); err == nil {
		t.Fatal("UnreadByte after ReadByte at EOF: got no error")
	}

	// check not at EOF
	b.WriteString("abcdefghijklmnopqrstuvwxyz")

	// after unsuccessful read
	if n, err := b.Read(nil); n != 0 || err != nil {
		t.Fatalf("Read(nil) = %d,%v; want 0,nil", n, err)
	}
	if err := b.UnreadByte(); err == nil {
		t.Fatal("UnreadByte after Read(nil): got no error")
	}

	// after successful read
	if _, err := b.ReadBytes('m'); err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if err := b.UnreadByte(); err != nil {
		t.Fatalf("UnreadByte: %v", err)
	}
	c, err := b.ReadByte()
	if err != nil {
		t.Fatalf("ReadByte: %v", err)
	}
	if c != 'm' {
		t.Errorf("ReadByte = %q; want %q", c, 'm')
	}
}

// Tests that we occasionally compact. Issue 5154.
func TestBufferGrowth(t *testing.T) {
	var b Buffer
	buf := make([]byte, 1024)
	b.Write(buf[0:1])
	var cap0 int
	for i := 0; i < 5<<10; i++ {
		b.Write(buf)
		b.Read(buf)
		if i == 0 {
			cap0 = b.Cap()
		}
	}
	cap1 := b.Cap()
	// (*Buffer).grow allows for 2x capacity slop before sliding,
	// so set our error threshold at 3x.
	if cap1 > cap0*3 {
		t.Errorf("buffer cap = %d; too big (grew from %d)", cap1, cap0)
	}
}
