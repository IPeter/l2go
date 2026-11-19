package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestDestroyItem is the opcode of RequestDestroyItem.
const OpRequestDestroyItem = 0x59

// RequestDestroyItem destroys Count of an inventory item. The count is not
// checked here: Mobius answers a count below 1 with a system message, and
// punishes a negative one.
type RequestDestroyItem struct {
	ObjectID int32
	Count    int32
}

// ParseRequestDestroyItem parses a RequestDestroyItem body (Mobius
// RequestDestroyItem.readImpl).
func ParseRequestDestroyItem(data []byte) (RequestDestroyItem, error) {
	r := packets.NewReader(data)
	d := RequestDestroyItem{
		ObjectID: r.ReadInt32(),
		Count:    r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return RequestDestroyItem{}, fmt.Errorf("RequestDestroyItem: %w", err)
	}
	return d, nil
}
