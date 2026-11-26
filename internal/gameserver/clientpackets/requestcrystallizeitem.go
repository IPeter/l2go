package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestCrystallizeItem is the opcode of RequestCrystallizeItem.
const OpRequestCrystallizeItem = 0x72

// RequestCrystallizeItem crystallizes Count of an inventory item. The count
// is not checked here: Mobius punishes a count below 1.
type RequestCrystallizeItem struct {
	ObjectID int32
	Count    int32
}

// ParseRequestCrystallizeItem parses a RequestCrystallizeItem body (Mobius
// RequestCrystallizeItem.readImpl).
func ParseRequestCrystallizeItem(data []byte) (RequestCrystallizeItem, error) {
	r := packets.NewReader(data)
	c := RequestCrystallizeItem{
		ObjectID: r.ReadInt32(),
		Count:    r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return RequestCrystallizeItem{}, fmt.Errorf("RequestCrystallizeItem: %w", err)
	}
	return c, nil
}
