package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpShortBuffStatusUpdate is the opcode of ShortBuffStatusUpdate.
const OpShortBuffStatusUpdate = 0xf4

// ShortBuffStatusUpdate sets the short buff icon, the separate slot Mobius
// uses for the effect of a healing potion. The zero value clears it (Mobius
// RESET_SHORT_BUFF).
type ShortBuffStatusUpdate struct {
	// SkillID and SkillLevel are the skill's own ID and level, not the
	// display ones.
	SkillID    int32
	SkillLevel int32
	Seconds    int32 // the time left in seconds
}

// NewShortBuffStatusUpdatePacket builds ShortBuffStatusUpdate (Mobius
// ShortBuffStatusUpdate.writeImpl).
func NewShortBuffStatusUpdatePacket(s ShortBuffStatusUpdate) []byte {
	w := packets.NewWriter(13)
	w.WriteUint8(OpShortBuffStatusUpdate)
	w.WriteInt32(s.SkillID)
	w.WriteInt32(s.SkillLevel)
	w.WriteInt32(s.Seconds)
	return w.Bytes()
}
