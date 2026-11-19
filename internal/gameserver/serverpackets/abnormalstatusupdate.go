package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpAbnormalStatusUpdate is the opcode of AbnormalStatusUpdate.
const OpAbnormalStatusUpdate = 0x7f

// AbnormalStatusEntry is one buff or debuff icon of the player.
type AbnormalStatusEntry struct {
	SkillID    int32 // the display ID of the skill
	SkillLevel int16 // the display level of the skill
	Seconds    int32 // the time left in seconds
}

// NewAbnormalStatusUpdatePacket builds AbnormalStatusUpdate (Mobius
// AbnormalStatusUpdate.writeImpl), the player's buff bar. An empty list is
// valid.
func NewAbnormalStatusUpdatePacket(effects []AbnormalStatusEntry) []byte {
	w := packets.NewWriter(3 + len(effects)*10)
	w.WriteUint8(OpAbnormalStatusUpdate)
	putCount16(w, len(effects))
	for _, e := range effects {
		w.WriteInt32(e.SkillID)
		putInt16(w, e.SkillLevel)
		w.WriteInt32(e.Seconds)
	}
	return w.Bytes()
}
