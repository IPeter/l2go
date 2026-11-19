package packets

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestWriterLayout(t *testing.T) {
	var w Writer
	w.WriteUint8(0x01)
	w.WriteUint16(0x0302)
	w.WriteUint32(0x07060504)
	w.WriteUint64(0x0f0e0d0c0b0a0908)
	w.WriteInt32(-2)
	w.WriteFloat64(1)
	w.WriteBytes([]byte{0xaa, 0xbb})
	w.WriteString("Ab")

	want := []byte{
		0x01,
		0x02, 0x03,
		0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
		0xfe, 0xff, 0xff, 0xff,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x3f,
		0xaa, 0xbb,
		'A', 0x00, 'b', 0x00, 0x00, 0x00,
	}
	if !bytes.Equal(w.Bytes(), want) {
		t.Errorf("Bytes() =\n% x\nwant\n% x", w.Bytes(), want)
	}
	if w.Len() != len(want) {
		t.Errorf("Len() = %d, want %d", w.Len(), len(want))
	}
}

func TestNewWriterNegativeSize(t *testing.T) {
	w := NewWriter(-1)
	w.WriteUint8(1)
	if w.Len() != 1 {
		t.Errorf("Len() = %d, want 1", w.Len())
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		u8  uint8
		u16 uint16
		u32 uint32
		u64 uint64
		i32 int32
		f64 float64
	}{
		{},
		{1, 2, 3, 4, -5, 6.5},
		{math.MaxUint8, math.MaxUint16, math.MaxUint32, math.MaxUint64, math.MinInt32, -math.MaxFloat64},
		{0x80, 0x8000, 0x80000000, 1 << 63, math.MaxInt32, math.SmallestNonzeroFloat64},
		{f64: math.Inf(1)},
	}
	for _, tt := range tests {
		w := NewWriter(0)
		w.WriteUint8(tt.u8)
		w.WriteUint16(tt.u16)
		w.WriteUint32(tt.u32)
		w.WriteUint64(tt.u64)
		w.WriteInt32(tt.i32)
		w.WriteFloat64(tt.f64)

		r := NewReader(w.Bytes())
		if got := r.ReadUint8(); got != tt.u8 {
			t.Errorf("ReadUint8 = %d, want %d", got, tt.u8)
		}
		if got := r.ReadUint16(); got != tt.u16 {
			t.Errorf("ReadUint16 = %d, want %d", got, tt.u16)
		}
		if got := r.ReadUint32(); got != tt.u32 {
			t.Errorf("ReadUint32 = %d, want %d", got, tt.u32)
		}
		if got := r.ReadUint64(); got != tt.u64 {
			t.Errorf("ReadUint64 = %d, want %d", got, tt.u64)
		}
		if got := r.ReadInt32(); got != tt.i32 {
			t.Errorf("ReadInt32 = %d, want %d", got, tt.i32)
		}
		if got := r.ReadFloat64(); got != tt.f64 {
			t.Errorf("ReadFloat64 = %v, want %v", got, tt.f64)
		}
		if err := r.Err(); err != nil {
			t.Errorf("Err() = %v", err)
		}
		if r.Len() != 0 {
			t.Errorf("Len() = %d unread bytes, want 0", r.Len())
		}
	}
}

func TestFloat64NaN(t *testing.T) {
	w := NewWriter(8)
	w.WriteFloat64(math.NaN())
	if got := NewReader(w.Bytes()).ReadFloat64(); !math.IsNaN(got) {
		t.Errorf("ReadFloat64 = %v, want NaN", got)
	}
}

func TestReaderShortInput(t *testing.T) {
	reads := map[string]func(r *Reader){
		"ReadUint8":   func(r *Reader) { r.ReadUint8() },
		"ReadUint16":  func(r *Reader) { r.ReadUint16() },
		"ReadUint32":  func(r *Reader) { r.ReadUint32() },
		"ReadUint64":  func(r *Reader) { r.ReadUint64() },
		"ReadInt32":   func(r *Reader) { r.ReadInt32() },
		"ReadFloat64": func(r *Reader) { r.ReadFloat64() },
		"ReadBytes":   func(r *Reader) { r.ReadBytes(9) },
		"ReadString":  func(r *Reader) { r.ReadString() },
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			r := NewReader([]byte{0x41})
			read(r)
			if name == "ReadUint8" {
				// One byte is enough for a uint8; the next read fails.
				read(r)
			}
			if !errors.Is(r.Err(), ErrShortPacket) {
				t.Errorf("Err() = %v, want ErrShortPacket", r.Err())
			}
		})
	}
}

func TestReaderStickyError(t *testing.T) {
	r := NewReader([]byte{1, 2, 3})
	if got := r.ReadUint32(); got != 0 {
		t.Errorf("failed ReadUint32 = %d, want 0", got)
	}
	first := r.Err()
	if first == nil {
		t.Fatal("Err() = nil after a short read")
	}
	// Later reads return zero values even though bytes remain.
	if got := r.ReadUint8(); got != 0 {
		t.Errorf("ReadUint8 after error = %d, want 0", got)
	}
	if got := r.ReadBytes(1); got != nil {
		t.Errorf("ReadBytes after error = %v, want nil", got)
	}
	if got := r.ReadString(); got != "" {
		t.Errorf("ReadString after error = %q, want empty", got)
	}
	if !errors.Is(r.Err(), first) {
		t.Errorf("Err() changed to %v, want the first error", r.Err())
	}
}

func TestReadBytes(t *testing.T) {
	data := []byte{1, 2, 3}
	r := NewReader(data)
	got := r.ReadBytes(2)
	if !bytes.Equal(got, []byte{1, 2}) {
		t.Fatalf("ReadBytes = %v", got)
	}
	got[0] = 9
	if data[0] != 1 {
		t.Error("ReadBytes returned a slice aliasing the input")
	}
	if got := r.ReadBytes(0); got == nil || len(got) != 0 {
		t.Errorf("ReadBytes(0) = %#v, want empty non-nil", got)
	}
	if got := NewReader(nil).ReadBytes(0); got == nil {
		t.Error("ReadBytes(0) on a nil buffer = nil, want empty non-nil")
	}
	if got := r.ReadBytes(-1); got != nil || !errors.Is(r.Err(), ErrShortPacket) {
		t.Errorf("ReadBytes(-1) = %v, Err() = %v; want nil, ErrShortPacket", got, r.Err())
	}
}

// B5: ReadString used to return the raw UTF-16LE bytes as a Go string.
func TestReadStringDecodesUTF16(t *testing.T) {
	data := []byte{'N', 0, 'a', 0, 'm', 0, 'e', 0, 0, 0, 0x2a}
	r := NewReader(data)
	if got := r.ReadString(); got != "Name" {
		t.Errorf("ReadString = %q, want %q", got, "Name")
	}
	if got := r.ReadUint8(); got != 0x2a {
		t.Errorf("byte after string = %#x, want 0x2a", got)
	}
	if err := r.Err(); err != nil {
		t.Errorf("Err() = %v", err)
	}
}

func TestStringRoundTrip(t *testing.T) {
	for _, s := range []string{"", "a", "Name", "Árvíztűrő tükörfúrógép", "Лунный", "日本語", "😀 emoji (surrogate pair)"} {
		w := NewWriter(0)
		w.WriteString(s)
		w.WriteString("next")
		r := NewReader(w.Bytes())
		if got := r.ReadString(); got != s {
			t.Errorf("ReadString = %q, want %q", got, s)
		}
		if got := r.ReadString(); got != "next" {
			t.Errorf("second ReadString = %q, want %q", got, "next")
		}
		if err := r.Err(); err != nil {
			t.Errorf("Err() = %v", err)
		}
	}
}

func TestReadStringMalformed(t *testing.T) {
	tests := map[string][]byte{
		"empty":             {},
		"no terminator":     {'a', 0, 'b', 0},
		"odd trailing byte": {'a', 0, 0},
	}
	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			r := NewReader(data)
			if got := r.ReadString(); got != "" {
				t.Errorf("ReadString = %q, want empty", got)
			}
			if !errors.Is(r.Err(), ErrShortPacket) {
				t.Errorf("Err() = %v, want ErrShortPacket", r.Err())
			}
		})
	}

	// A lone surrogate decodes to U+FFFD instead of failing.
	r := NewReader([]byte{0x00, 0xd8, 0, 0})
	if got := r.ReadString(); got != "�" {
		t.Errorf("lone surrogate = %q, want U+FFFD", got)
	}
}

func FuzzReader(f *testing.F) {
	f.Add([]byte{}, []byte{})
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6, 7}, []byte{1, 2, 3, 4, 5, 6, 7})
	f.Add([]byte{'a', 0, 0, 0, 1}, []byte{7, 0})
	f.Fuzz(func(t *testing.T, data, ops []byte) {
		r := NewReader(data)
		for _, op := range ops {
			before := r.Len()
			switch op % 8 {
			case 0:
				r.ReadUint8()
			case 1:
				r.ReadUint16()
			case 2:
				r.ReadUint32()
			case 3:
				r.ReadUint64()
			case 4:
				r.ReadInt32()
			case 5:
				r.ReadFloat64()
			case 6:
				r.ReadBytes(int(op>>3) - 2)
			case 7:
				r.ReadString()
			}
			if r.Len() < 0 || r.Len() > before {
				t.Fatalf("Len() went from %d to %d", before, r.Len())
			}
		}
	})
}

func FuzzStringRoundTrip(f *testing.F) {
	f.Add("")
	f.Add("Name")
	f.Add("Árvíztűrő 😀")
	f.Fuzz(func(t *testing.T, s string) {
		if !utf8.ValidString(s) || strings.ContainsRune(s, 0) {
			t.Skip("only valid UTF-8 without NUL round-trips")
		}
		w := NewWriter(0)
		w.WriteString(s)
		r := NewReader(w.Bytes())
		if got := r.ReadString(); got != s || r.Err() != nil {
			t.Fatalf("round trip of %q = %q, %v", s, got, r.Err())
		}
	})
}
