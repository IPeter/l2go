package serverpackets

import (
	"math"

	"github.com/IPeter/l2go/internal/packets"
)

// The put helpers write the Java values that packets.Writer has no method
// for, with the same bytes as the Mobius WriteBuffer.

// putBool8 writes b as one byte, 1 or 0 (WriteBuffer.writeByte(boolean)).
func putBool8(w *packets.Writer, b bool) {
	if b {
		w.WriteUint8(1)
		return
	}
	w.WriteUint8(0)
}

// putBool16 writes b as a short, 1 or 0 (WriteBuffer.writeShort(boolean)).
func putBool16(w *packets.Writer, b bool) {
	if b {
		w.WriteUint16(1)
		return
	}
	w.WriteUint16(0)
}

// putBool32 writes b as an int, 1 or 0 (WriteBuffer.writeInt(boolean)).
func putBool32(w *packets.Writer, b bool) {
	if b {
		w.WriteInt32(1)
		return
	}
	w.WriteInt32(0)
}

// putInt8 writes a signed byte.
func putInt8(w *packets.Writer, v int8) { w.WriteUint8(uint8(v)) } //nolint:gosec // same bits

// putInt16 writes a signed little-endian short.
func putInt16(w *packets.Writer, v int16) { w.WriteUint16(uint16(v)) } //nolint:gosec // same bits

// putInt64 writes a signed little-endian long.
func putInt64(w *packets.Writer, v int64) { w.WriteUint64(uint64(v)) } //nolint:gosec // same bits

// putCount16 writes a list length as a short, like writeShort(list.size()).
// The lists in these packets are far shorter than 32,768 entries.
func putCount16(w *packets.Writer, n int) { w.WriteUint16(uint16(n)) } //nolint:gosec // bounded by game rules

// putCount32 writes a list length as an int, like writeInt(list.size()).
func putCount32(w *packets.Writer, n int) { w.WriteInt32(int32(n)) } //nolint:gosec // bounded by game rules

// truncToInt32 converts x like the Java cast (int) x: toward zero, NaN as
// 0, and values outside the int range clamped to it.
func truncToInt32(x float64) int32 {
	switch {
	case math.IsNaN(x):
		return 0
	case x >= math.MaxInt32:
		return math.MaxInt32
	case x <= math.MinInt32:
		return math.MinInt32
	}
	return int32(x)
}

// roundHalfUp32 converts x like the Java (int) Math.round(x): to the
// nearest integer with halves rounded up (toward +Inf).
func roundHalfUp32(x float64) int32 {
	if math.IsNaN(x) {
		return 0
	}
	return truncToInt32(math.Floor(x + 0.5))
}
