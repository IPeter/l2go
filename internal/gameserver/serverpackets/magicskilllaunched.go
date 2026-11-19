package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpMagicSkillLaunched is the opcode of MagicSkillLaunched.
const OpMagicSkillLaunched = 0x76

// MagicSkillLaunched shows a skill taking effect on its targets, after the
// MagicSkillUse that started it.
type MagicSkillLaunched struct {
	CasterID   int32
	SkillID    int32 // the display ID of the skill
	SkillLevel int32 // the display level of the skill
	// TargetIDs are the objects the skill affects. They are sent as given,
	// an empty list too. The Mobius constructors without targets send the
	// caster as the only one: give []int32{CasterID} for that.
	TargetIDs []int32
}

// NewMagicSkillLaunchedPacket builds MagicSkillLaunched (Mobius
// MagicSkillLaunched.writeImpl).
func NewMagicSkillLaunchedPacket(m MagicSkillLaunched) []byte {
	w := packets.NewWriter(17 + 4*len(m.TargetIDs))
	w.WriteUint8(OpMagicSkillLaunched)
	w.WriteInt32(m.CasterID)
	w.WriteInt32(m.SkillID)
	w.WriteInt32(m.SkillLevel)
	putCount32(w, len(m.TargetIDs))
	for _, id := range m.TargetIDs {
		w.WriteInt32(id)
	}
	return w.Bytes()
}
