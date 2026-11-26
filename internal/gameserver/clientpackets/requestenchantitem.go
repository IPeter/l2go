package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestEnchantItem is the opcode of RequestEnchantItem.
const OpRequestEnchantItem = 0x58

// RequestEnchantItem names the item to enchant with the scroll the player
// used before (ChooseInventoryItem). Interlude sends no support item.
type RequestEnchantItem struct {
	ObjectID int32
}

// ParseRequestEnchantItem parses a RequestEnchantItem body (Mobius
// RequestEnchantItem.readImpl).
func ParseRequestEnchantItem(data []byte) (RequestEnchantItem, error) {
	r := packets.NewReader(data)
	e := RequestEnchantItem{ObjectID: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestEnchantItem{}, fmt.Errorf("RequestEnchantItem: %w", err)
	}
	return e, nil
}
