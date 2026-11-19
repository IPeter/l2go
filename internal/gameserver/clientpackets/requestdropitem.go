package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestDropItem is the opcode of RequestDropItem.
const OpRequestDropItem = 0x12

// RequestDropItem drops Count of an inventory item on the ground at the
// point the player chose. The values are not checked here: Mobius refuses a
// count of 0 or more than the player has, and a point too far away.
type RequestDropItem struct {
	ObjectID int32
	Count    int32
	X, Y, Z  int32
}

// ParseRequestDropItem parses a RequestDropItem body (Mobius
// RequestDropItem.readImpl).
func ParseRequestDropItem(data []byte) (RequestDropItem, error) {
	r := packets.NewReader(data)
	d := RequestDropItem{
		ObjectID: r.ReadInt32(),
		Count:    r.ReadInt32(),
		X:        r.ReadInt32(),
		Y:        r.ReadInt32(),
		Z:        r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return RequestDropItem{}, fmt.Errorf("RequestDropItem: %w", err)
	}
	return d, nil
}
