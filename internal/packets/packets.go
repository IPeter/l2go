// Package packets builds and parses Lineage II packet bodies and handles
// the length-prefixed framing used on every connection.
//
// All multi-byte values are little-endian. Strings are UTF-16LE and end
// with a zero code unit.
package packets

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"unicode/utf16"
)

// ErrShortPacket is reported by Reader when a read goes past the end of the
// data, or a string has no terminator.
var ErrShortPacket = errors.New("packets: short packet")

// Writer builds a packet body. The zero value is ready to use.
type Writer struct {
	buf []byte
}

// NewWriter returns a Writer with room for size bytes.
func NewWriter(size int) *Writer {
	return &Writer{buf: make([]byte, 0, max(size, 0))}
}

// WriteUint8 appends v.
func (w *Writer) WriteUint8(v uint8) { w.buf = append(w.buf, v) }

// WriteUint16 appends v in little-endian order.
func (w *Writer) WriteUint16(v uint16) { w.buf = binary.LittleEndian.AppendUint16(w.buf, v) }

// WriteUint32 appends v in little-endian order.
func (w *Writer) WriteUint32(v uint32) { w.buf = binary.LittleEndian.AppendUint32(w.buf, v) }

// WriteUint64 appends v in little-endian order.
func (w *Writer) WriteUint64(v uint64) { w.buf = binary.LittleEndian.AppendUint64(w.buf, v) }

// WriteInt32 appends v in little-endian two's complement.
func (w *Writer) WriteInt32(v int32) { w.WriteUint32(uint32(v)) } //nolint:gosec // same bits

// WriteFloat64 appends v as a little-endian IEEE 754 double.
func (w *Writer) WriteFloat64(v float64) { w.WriteUint64(math.Float64bits(v)) }

// WriteString appends s as UTF-16LE followed by a zero code unit.
func (w *Writer) WriteString(s string) {
	for _, u := range utf16.Encode([]rune(s)) {
		w.WriteUint16(u)
	}
	w.WriteUint16(0)
}

// WriteBytes appends b unchanged.
func (w *Writer) WriteBytes(b []byte) { w.buf = append(w.buf, b...) }

// Len returns the number of bytes written so far.
func (w *Writer) Len() int { return len(w.buf) }

// Bytes returns the packet body. It aliases the Writer's buffer.
func (w *Writer) Bytes() []byte { return w.buf }

// Reader parses a packet body.
//
// Errors are sticky: after the first failed read every later read returns a
// zero value, and Err reports the first error. Reads never panic.
type Reader struct {
	buf []byte
	off int
	err error
}

// NewReader returns a Reader over b. It does not copy b.
func NewReader(b []byte) *Reader {
	return &Reader{buf: b}
}

// Err returns the first error met while reading, or nil.
func (r *Reader) Err() error { return r.err }

// Len returns the number of unread bytes.
func (r *Reader) Len() int { return len(r.buf) - r.off }

// next returns the next n bytes. On failure it sets the sticky error and
// returns false.
func (r *Reader) next(n int) ([]byte, bool) {
	if r.err != nil {
		return nil, false
	}
	if n < 0 || n > r.Len() {
		r.err = fmt.Errorf("%w: need %d bytes at offset %d, have %d", ErrShortPacket, n, r.off, r.Len())
		return nil, false
	}
	b := r.buf[r.off : r.off+n]
	r.off += n
	return b, true
}

// ReadUint8 reads one byte.
func (r *Reader) ReadUint8() uint8 {
	b, ok := r.next(1)
	if !ok {
		return 0
	}
	return b[0]
}

// ReadUint16 reads a little-endian uint16.
func (r *Reader) ReadUint16() uint16 {
	b, ok := r.next(2)
	if !ok {
		return 0
	}
	return binary.LittleEndian.Uint16(b)
}

// ReadUint32 reads a little-endian uint32.
func (r *Reader) ReadUint32() uint32 {
	b, ok := r.next(4)
	if !ok {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}

// ReadUint64 reads a little-endian uint64.
func (r *Reader) ReadUint64() uint64 {
	b, ok := r.next(8)
	if !ok {
		return 0
	}
	return binary.LittleEndian.Uint64(b)
}

// ReadInt32 reads a little-endian int32.
func (r *Reader) ReadInt32() int32 { return int32(r.ReadUint32()) } //nolint:gosec // same bits

// ReadFloat64 reads a little-endian IEEE 754 double.
func (r *Reader) ReadFloat64() float64 { return math.Float64frombits(r.ReadUint64()) }

// ReadBytes reads n bytes and returns a copy of them, or nil on error.
func (r *Reader) ReadBytes(n int) []byte {
	b, ok := r.next(n)
	if !ok {
		return nil
	}
	out := make([]byte, n)
	copy(out, b)
	return out
}

// ReadString reads a zero-terminated UTF-16LE string. Invalid surrogates
// decode to U+FFFD.
func (r *Reader) ReadString() string {
	if r.err != nil {
		return ""
	}
	var units []uint16
	for i := r.off; i+1 < len(r.buf); i += 2 {
		u := binary.LittleEndian.Uint16(r.buf[i:])
		if u == 0 {
			r.off = i + 2
			return string(utf16.Decode(units))
		}
		units = append(units, u)
	}
	r.err = fmt.Errorf("%w: unterminated string at offset %d", ErrShortPacket, r.off)
	return ""
}
