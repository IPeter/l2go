package clientpackets

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"unicode/utf16"

	"github.com/IPeter/l2go/internal/packets"
)

// m3Body builds a client packet body field by field, like the client:
// c byte, h short, d int, s string.
type m3Body struct{ b []byte }

func (m *m3Body) c(v byte) *m3Body { m.b = append(m.b, v); return m }

func (m *m3Body) h(v int16) *m3Body {
	m.b = binary.LittleEndian.AppendUint16(m.b, uint16(v))
	return m
}

func (m *m3Body) d(v int32) *m3Body {
	m.b = binary.LittleEndian.AppendUint32(m.b, uint32(v))
	return m
}

func (m *m3Body) s(v string) *m3Body {
	for _, u := range utf16.Encode([]rune(v)) {
		m.b = binary.LittleEndian.AppendUint16(m.b, u)
	}
	m.b = append(m.b, 0, 0)
	return m
}

// m3Parsed checks a successful parse.
func m3Parsed[T any](t *testing.T, name string, got T, err error, want T) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n got %+v\nwant %+v", name, got, want)
	}
}

// m3Failed checks a failed parse: ErrShortPacket and the zero value.
func m3Failed[T any](t *testing.T, name string, got T, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: parsed as %+v, want an error", name, got)
		return
	}
	if !errors.Is(err, packets.ErrShortPacket) {
		t.Errorf("%s: error %v, want ErrShortPacket", name, err)
	}
	var zero T
	if !reflect.DeepEqual(got, zero) {
		t.Errorf("%s: got %+v with the error, want the zero value", name, got)
	}
}

// m3Truncations checks that every proper prefix of full fails to parse.
func m3Truncations[T any](t *testing.T, name string, full []byte, parse func([]byte) (T, error)) {
	t.Helper()
	for n := range len(full) {
		got, err := parse(full[:n])
		m3Failed(t, fmt.Sprintf("%s, %d of %d bytes", name, n, len(full)), got, err)
	}
}

// m3Fuzz runs parse on fuzzed input. It must not panic, and a failure must
// be ErrShortPacket with the zero value. When size is not negative, the
// packet has a fixed size: parsing must succeed exactly when the input
// has at least size bytes, as Mobius ignores the bytes after the fields.
func m3Fuzz[T any](f *testing.F, parse func([]byte) (T, error), size int, seeds ...[]byte) {
	f.Add([]byte{})
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := parse(data)
		if err != nil {
			m3Failed(t, "fuzzed input", got, err)
		}
		if size >= 0 && (err == nil) != (len(data) >= size) {
			t.Errorf("%d bytes: error %v, fixed size %d", len(data), err, size)
		}
	})
}
