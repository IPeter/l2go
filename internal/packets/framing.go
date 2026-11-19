package packets

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// HeaderSize is the size of the length prefix in front of every packet. The
// prefix is a little-endian uint16 that counts itself plus the body.
const HeaderSize = 2

// MaxBodySize is the largest body the length prefix can describe.
const MaxBodySize = math.MaxUint16 - HeaderSize

// ErrPacketSize is returned for a length prefix that describes an empty
// body or a body larger than allowed.
var ErrPacketSize = errors.New("packets: invalid packet size")

// ReadPacket reads one length-prefixed packet from r and returns its body
// without the prefix. A body must hold at least one byte (the opcode) and at
// most maxSize bytes.
//
// A clean end of stream before the prefix is reported as an error wrapping
// io.EOF; a stream that ends inside a packet gives io.ErrUnexpectedEOF.
func ReadPacket(r io.Reader, maxSize int) ([]byte, error) {
	var header [HeaderSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("packets: read header: %w", err)
	}
	size := int(binary.LittleEndian.Uint16(header[:])) - HeaderSize
	if size <= 0 || size > min(maxSize, MaxBodySize) {
		return nil, fmt.Errorf("%w: body of %d bytes (max %d)", ErrPacketSize, size, maxSize)
	}
	body := make([]byte, size)
	if _, err := io.ReadFull(r, body); err != nil {
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return nil, fmt.Errorf("packets: read body: %w", err)
	}
	return body, nil
}

// WritePacket writes body to w with its length prefix, in a single Write.
func WritePacket(w io.Writer, body []byte) error {
	if len(body) == 0 || len(body) > MaxBodySize {
		return fmt.Errorf("%w: body of %d bytes", ErrPacketSize, len(body))
	}
	buf := make([]byte, HeaderSize, HeaderSize+len(body))
	binary.LittleEndian.PutUint16(buf, uint16(HeaderSize+len(body))) //nolint:gosec // bounded by MaxBodySize above
	buf = append(buf, body...)
	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("packets: write: %w", err)
	}
	return nil
}
