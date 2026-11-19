package serverpackets

import (
	"encoding/binary"
	"math"
	"testing"
	"unicode/utf16"
)

// m3Golden builds an expected packet field by field, like the Mobius
// WriteBuffer, without the production packets.Writer or put helpers:
// c byte, h short, d int, q long, f double, s string.
type m3Golden struct{ b []byte }

func (g *m3Golden) c(v byte) *m3Golden { g.b = append(g.b, v); return g }

func (g *m3Golden) h(v int16) *m3Golden {
	g.b = binary.LittleEndian.AppendUint16(g.b, uint16(v))
	return g
}

func (g *m3Golden) d(v int32) *m3Golden {
	g.b = binary.LittleEndian.AppendUint32(g.b, uint32(v))
	return g
}

func (g *m3Golden) q(v int64) *m3Golden {
	g.b = binary.LittleEndian.AppendUint64(g.b, uint64(v))
	return g
}

func (g *m3Golden) f(v float64) *m3Golden {
	g.b = binary.LittleEndian.AppendUint64(g.b, math.Float64bits(v))
	return g
}

func (g *m3Golden) s(v string) *m3Golden {
	for _, u := range utf16.Encode([]rune(v)) {
		g.b = binary.LittleEndian.AppendUint16(g.b, u)
	}
	g.b = append(g.b, 0, 0)
	return g
}

// hn appends n zero shorts; dn appends n zero ints.
func (g *m3Golden) hn(n int) *m3Golden {
	for range n {
		g.h(0)
	}
	return g
}

func (g *m3Golden) dn(n int) *m3Golden {
	for range n {
		g.d(0)
	}
	return g
}

// m3Check compares a built packet with the expected bytes and, on a
// mismatch, reports the first differing offset as well.
func m3Check(t *testing.T, name string, got, want []byte) {
	t.Helper()
	if string(got) == string(want) {
		return
	}
	at := min(len(got), len(want))
	for i := range at {
		if got[i] != want[i] {
			at = i
			break
		}
	}
	t.Errorf("%s: first difference at offset %d (got %d bytes, want %d)\n got % x\nwant % x", name, at, len(got), len(want), got, want)
}

func TestRoundHalfUp32(t *testing.T) {
	tests := []struct {
		in   float64
		want int32
	}{
		{0, 0},
		{0.49, 0},
		{0.5, 1},
		{1.5, 2},
		{2.4999, 2},
		{380.5, 381},
		{-0.5, 0}, // Java Math.round rounds halves up, not away from zero
		{-1.5, -1},
		{-2.6, -3},
		{math.NaN(), 0},
		{math.Inf(1), math.MaxInt32},
		{math.Inf(-1), math.MinInt32},
		{3e9, math.MaxInt32},
		{-3e9, math.MinInt32},
	}
	for _, tt := range tests {
		if got := roundHalfUp32(tt.in); got != tt.want {
			t.Errorf("roundHalfUp32(%v) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestTruncToInt32(t *testing.T) {
	tests := []struct {
		in   float64
		want int32
	}{
		{0, 0},
		{2.9, 2},
		{249.99, 249},
		{-2.9, -2}, // toward zero
		{math.NaN(), 0},
		{1e10, math.MaxInt32},
		{-1e10, math.MinInt32},
		{math.Inf(1), math.MaxInt32},
	}
	for _, tt := range tests {
		if got := truncToInt32(tt.in); got != tt.want {
			t.Errorf("truncToInt32(%v) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
