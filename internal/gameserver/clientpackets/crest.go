package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestSetPledgeCrest is the opcode of RequestSetPledgeCrest. The
// large crest upload is the extended packet ExOpRequestExSetPledgeCrestLarge
// and the alliance crest upload OpRequestSetAllyCrest.
const OpRequestSetPledgeCrest = 0x53

// The largest crest images the uploads read, in bytes (the Mobius readImpl
// checks): the clan crest (16×12 pixels in the Mobius message), the large
// clan crest and the alliance crest (8×12).
const (
	MaxPledgeCrestSize      = 256
	MaxPledgeCrestLargeSize = 2176
	MaxAllyCrestSize        = 192
)

// CrestUpload is the body of RequestSetPledgeCrest,
// RequestExSetPledgeCrestLarge and RequestSetAllyCrest: a crest image.
type CrestUpload struct {
	// Length is the image size the client sent. 0 deletes the crest.
	Length int32
	// Data is the image. It is nil when Length is above the packet's
	// maximum: Mobius then does not read it, and answers with a message
	// about the image size (no message for the large crest).
	Data []byte
}

// parseCrestUpload reads d length and, when the length is at most max,
// that many bytes. A negative length, or a length the data does not hold,
// fails as in Mobius, whose readBytes throws.
func parseCrestUpload(packet string, data []byte, maxSize int32) (CrestUpload, error) {
	r := packets.NewReader(data)
	c := CrestUpload{Length: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return CrestUpload{}, fmt.Errorf("%s: %w", packet, err)
	}
	if c.Length > maxSize {
		return c, nil
	}
	c.Data = r.ReadBytes(int(c.Length))
	if err := r.Err(); err != nil {
		return CrestUpload{}, fmt.Errorf("%s: %w", packet, err)
	}
	return c, nil
}

// ParseRequestSetPledgeCrest parses a RequestSetPledgeCrest body (Mobius
// RequestSetPledgeCrest.readImpl): d length, then at most
// MaxPledgeCrestSize bytes.
func ParseRequestSetPledgeCrest(data []byte) (CrestUpload, error) {
	return parseCrestUpload("RequestSetPledgeCrest", data, MaxPledgeCrestSize)
}

// ParseRequestExSetPledgeCrestLarge parses a RequestExSetPledgeCrestLarge
// body, the data after the sub-opcode (Mobius
// RequestExSetPledgeCrestLarge.readImpl): d length, then at most
// MaxPledgeCrestLargeSize bytes.
func ParseRequestExSetPledgeCrestLarge(data []byte) (CrestUpload, error) {
	return parseCrestUpload("RequestExSetPledgeCrestLarge", data, MaxPledgeCrestLargeSize)
}

// ParseRequestSetAllyCrest parses a RequestSetAllyCrest body (Mobius
// RequestSetAllyCrest.readImpl): d length, then at most MaxAllyCrestSize
// bytes.
func ParseRequestSetAllyCrest(data []byte) (CrestUpload, error) {
	return parseCrestUpload("RequestSetAllyCrest", data, MaxAllyCrestSize)
}
