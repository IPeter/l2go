package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpMultiSellChoose is the opcode of MultiSellChoose.
const OpMultiSellChoose = 0xa7

// MultiSellChoose makes Amount exchanges of one entry of the open multisell
// list. The values are not checked here: Mobius closes the list for an
// amount below 1 or above its limit, or for another list ID than the open
// one.
type MultiSellChoose struct {
	ListID  int32
	EntryID int32 // serverpackets.MultiSellEntry.EntryID
	Amount  int32
}

// ParseMultiSellChoose parses a MultiSellChoose body (Mobius
// MultiSellChoose.readImpl).
func ParseMultiSellChoose(data []byte) (MultiSellChoose, error) {
	r := packets.NewReader(data)
	m := MultiSellChoose{
		ListID:  r.ReadInt32(),
		EntryID: r.ReadInt32(),
		Amount:  r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return MultiSellChoose{}, fmt.Errorf("MultiSellChoose: %w", err)
	}
	return m, nil
}
