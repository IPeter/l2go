package packets

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"testing/iotest"
)

func TestWriteThenReadPacket(t *testing.T) {
	var buf bytes.Buffer
	bodies := [][]byte{{0x00}, {0x01, 0x02, 0x03}, bytes.Repeat([]byte{0xab}, 1000)}
	for _, b := range bodies {
		if err := WritePacket(&buf, b); err != nil {
			t.Fatalf("WritePacket: %v", err)
		}
	}
	for _, want := range bodies {
		got, err := ReadPacket(&buf, MaxBodySize)
		if err != nil {
			t.Fatalf("ReadPacket: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("ReadPacket = % x, want % x", got, want)
		}
	}
	if _, err := ReadPacket(&buf, MaxBodySize); !errors.Is(err, io.EOF) {
		t.Errorf("ReadPacket at end = %v, want io.EOF", err)
	}
}

func TestWritePacketHeader(t *testing.T) {
	var buf bytes.Buffer
	if err := WritePacket(&buf, []byte{0xaa, 0xbb, 0xcc}); err != nil {
		t.Fatal(err)
	}
	// The length counts the 2-byte header itself, little-endian.
	want := []byte{0x05, 0x00, 0xaa, 0xbb, 0xcc}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("wire = % x, want % x", buf.Bytes(), want)
	}
}

// countingWriter counts Write calls.
type countingWriter struct {
	bytes.Buffer
	calls int
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.calls++
	return w.Buffer.Write(p)
}

func TestWritePacketSingleWrite(t *testing.T) {
	var w countingWriter
	if err := WritePacket(&w, []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	if w.calls != 1 {
		t.Errorf("WritePacket made %d Write calls, want 1", w.calls)
	}
}

func TestWritePacketSizeLimits(t *testing.T) {
	for _, n := range []int{0, MaxBodySize + 1} {
		if err := WritePacket(io.Discard, make([]byte, n)); !errors.Is(err, ErrPacketSize) {
			t.Errorf("WritePacket(%d bytes) = %v, want ErrPacketSize", n, err)
		}
	}
	var buf bytes.Buffer
	if err := WritePacket(&buf, make([]byte, MaxBodySize)); err != nil {
		t.Fatalf("WritePacket(MaxBodySize) = %v", err)
	}
	if got := buf.Bytes()[:2]; !bytes.Equal(got, []byte{0xff, 0xff}) {
		t.Errorf("header = % x, want ff ff", got)
	}
}

type errWriter struct{ err error }

func (w errWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWritePacketError(t *testing.T) {
	boom := errors.New("boom")
	if err := WritePacket(errWriter{boom}, []byte{1}); !errors.Is(err, boom) {
		t.Errorf("WritePacket = %v, want %v", err, boom)
	}
}

// B1: header and body used to be read with one Read call each, so a short
// read from TCP broke the stream.
func TestReadPacketShortReads(t *testing.T) {
	wire := []byte{0x06, 0x00, 0x01, 0x02, 0x03, 0x04, 0x03, 0x00, 0x08}
	r := iotest.OneByteReader(bytes.NewReader(wire))
	got, err := ReadPacket(r, MaxBodySize)
	if err != nil {
		t.Fatalf("ReadPacket: %v", err)
	}
	if want := []byte{0x01, 0x02, 0x03, 0x04}; !bytes.Equal(got, want) {
		t.Errorf("first body = % x, want % x", got, want)
	}
	got, err = ReadPacket(r, MaxBodySize)
	if err != nil {
		t.Fatalf("ReadPacket: %v", err)
	}
	if want := []byte{0x08}; !bytes.Equal(got, want) {
		t.Errorf("second body = % x, want % x", got, want)
	}
}

func TestReadPacketHalfReader(t *testing.T) {
	var buf bytes.Buffer
	body := bytes.Repeat([]byte{1, 2, 3}, 100)
	if err := WritePacket(&buf, body); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPacket(iotest.HalfReader(&buf), MaxBodySize)
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("ReadPacket = %d bytes, %v; want %d bytes", len(got), err, len(body))
	}
}

// B2: a size below 3 used to give a negative slice length or an empty body
// that crashed the opcode read.
func TestReadPacketInvalidSize(t *testing.T) {
	tests := map[string][]byte{
		"size 0":   {0x00, 0x00},
		"size 1":   {0x01, 0x00},
		"size 2":   {0x02, 0x00, 0xff},
		"over max": {0x0d, 0x00, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
	}
	for name, wire := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ReadPacket(bytes.NewReader(wire), 10)
			if !errors.Is(err, ErrPacketSize) {
				t.Errorf("ReadPacket = %v, want ErrPacketSize", err)
			}
		})
	}
}

func TestReadPacketMaxSize(t *testing.T) {
	wire := []byte{0x0c, 0x00, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if _, err := ReadPacket(bytes.NewReader(wire), 10); err != nil {
		t.Errorf("body of exactly maxSize: %v", err)
	}
}

func TestReadPacketTruncated(t *testing.T) {
	tests := map[string]struct {
		wire []byte
		want error
	}{
		"empty stream": {nil, io.EOF},
		"half header":  {[]byte{0x05}, io.ErrUnexpectedEOF},
		"no body":      {[]byte{0x05, 0x00}, io.ErrUnexpectedEOF},
		"partial body": {[]byte{0x05, 0x00, 1, 2}, io.ErrUnexpectedEOF},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ReadPacket(bytes.NewReader(tt.wire), MaxBodySize)
			if !errors.Is(err, tt.want) {
				t.Errorf("ReadPacket = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestReadPacketReaderError(t *testing.T) {
	boom := errors.New("boom")
	r := io.MultiReader(bytes.NewReader([]byte{0x05, 0x00, 1}), iotest.ErrReader(boom))
	if _, err := ReadPacket(r, MaxBodySize); !errors.Is(err, boom) {
		t.Errorf("ReadPacket = %v, want %v", err, boom)
	}
}

func FuzzReadPacket(f *testing.F) {
	f.Add([]byte{0x03, 0x00, 0x01})
	f.Add([]byte{0x02, 0x00})
	f.Add([]byte{0xff, 0xff, 0x00})
	f.Add([]byte{0x05, 0x00, 1, 2, 3, 0x03, 0x00, 9})
	f.Fuzz(func(t *testing.T, wire []byte) {
		r := bytes.NewReader(wire)
		for {
			consumedBefore := len(wire) - r.Len()
			body, err := ReadPacket(r, 512)
			if err != nil {
				return
			}
			if len(body) == 0 || len(body) > 512 {
				t.Fatalf("body of %d bytes accepted", len(body))
			}
			// Writing the body back must give the bytes just read.
			var out bytes.Buffer
			if err := WritePacket(&out, body); err != nil {
				t.Fatalf("WritePacket: %v", err)
			}
			consumed := wire[consumedBefore : len(wire)-r.Len()]
			if !bytes.Equal(out.Bytes(), consumed) {
				t.Fatalf("re-encoded % x, read % x", out.Bytes(), consumed)
			}
		}
	})
}
