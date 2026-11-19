package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpMyTargetSelected is the opcode of MyTargetSelected.
const OpMyTargetSelected = 0xa6

// MyTargetSelected tells a player its new target. Mobius follows it with a
// StatusUpdate carrying the target's MaxHP and CurHP for the HP bar.
type MyTargetSelected struct {
	ObjectID int32
	// Color is the player's level minus the target's level when the player
	// can attack the target (Mobius isAutoAttackable), else 0. The client
	// colours the target's name by it.
	Color int16
}

// NewMyTargetSelectedPacket builds MyTargetSelected (Mobius
// MyTargetSelected.writeImpl).
func NewMyTargetSelectedPacket(m MyTargetSelected) []byte {
	w := packets.NewWriter(11)
	w.WriteUint8(OpMyTargetSelected)
	w.WriteInt32(m.ObjectID)
	putInt16(w, m.Color)
	w.WriteInt32(0)
	return w.Bytes()
}
