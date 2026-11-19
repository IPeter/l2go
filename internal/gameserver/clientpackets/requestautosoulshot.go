package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// ExOpRequestAutoSoulShot is the sub-opcode of the extended packet
// RequestAutoSoulShot.
const ExOpRequestAutoSoulShot = 0x05

// The values of RequestAutoSoulShot.Type.
const (
	AutoSoulShotOff = 0
	AutoSoulShotOn  = 1
)

// RequestAutoSoulShot turns the automatic use of a soulshot or spiritshot
// on or off, with a right click on the item's shortcut. Mobius answers with
// ExAutoSoulShot.
type RequestAutoSoulShot struct {
	ItemID int32 // the item template ID, not the object ID
	// Type is AutoSoulShotOn or AutoSoulShotOff, not checked here. Mobius
	// ignores other values.
	Type int32
}

// ParseRequestAutoSoulShot parses a RequestAutoSoulShot body, the data
// after the sub-opcode (Mobius RequestAutoSoulShot.readImpl): d item ID,
// d type.
func ParseRequestAutoSoulShot(data []byte) (RequestAutoSoulShot, error) {
	r := packets.NewReader(data)
	req := RequestAutoSoulShot{
		ItemID: r.ReadInt32(),
		Type:   r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return RequestAutoSoulShot{}, fmt.Errorf("RequestAutoSoulShot: %w", err)
	}
	return req, nil
}
