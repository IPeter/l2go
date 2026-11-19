package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpUseItem is the opcode of UseItem.
const OpUseItem = 0x14

// UseItem is a double click on an inventory item: it puts equipment on or
// takes it off, and uses other items.
type UseItem struct {
	ObjectID int32
	// CtrlPressed is set when the int the client sends is not 0. Mobius
	// compares with 0 here, not with 1 as in RequestActionUse, and passes the
	// flag to the item handler.
	CtrlPressed bool
}

// ParseUseItem parses a UseItem body (Mobius UseItem.readImpl).
func ParseUseItem(data []byte) (UseItem, error) {
	r := packets.NewReader(data)
	u := UseItem{
		ObjectID:    r.ReadInt32(),
		CtrlPressed: r.ReadInt32() != 0,
	}
	if err := r.Err(); err != nil {
		return UseItem{}, fmt.Errorf("UseItem: %w", err)
	}
	return u, nil
}
