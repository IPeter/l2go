// Package checksum implements the XOR checksum of login packets.
//
// The checksum is the XOR of all little-endian 32-bit words before the
// checksum word. L2Go places it in the second-to-last word of the padded
// packet (offset len-8), followed by one zero word. A peer that expects the
// checksum in the last word also accepts this layout, because the XOR of all
// words including the checksum is zero.
package checksum

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Size is the length of the checksum in bytes.
const Size = 4

// ErrLength is returned for data that cannot hold a checksum: shorter than
// 8 bytes or not a multiple of 4.
var ErrLength = errors.New("checksum: invalid data length")

func offset(b []byte) (int, bool) {
	if len(b) < 2*Size || len(b)%Size != 0 {
		return 0, false
	}
	return len(b) - 2*Size, true
}

func compute(b []byte, end int) uint32 {
	var sum uint32
	for i := 0; i < end; i += Size {
		sum ^= binary.LittleEndian.Uint32(b[i:])
	}
	return sum
}

// Verify reports whether b carries a valid checksum at offset len(b)-8.
func Verify(b []byte) bool {
	off, ok := offset(b)
	if !ok {
		return false
	}
	return compute(b, off) == binary.LittleEndian.Uint32(b[off:])
}

// Put computes the checksum of b and stores it at offset len(b)-8.
func Put(b []byte) error {
	off, ok := offset(b)
	if !ok {
		return fmt.Errorf("%w: %d bytes", ErrLength, len(b))
	}
	binary.LittleEndian.PutUint32(b[off:], compute(b, off))
	return nil
}
