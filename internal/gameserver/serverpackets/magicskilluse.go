package serverpackets

import (
	"time"

	"github.com/IPeter/l2go/internal/packets"
)

// OpMagicSkillUse is the opcode of MagicSkillUse.
const OpMagicSkillUse = 0x48

// MagicSkillUse starts the casting animation of a skill, shown to everyone
// who sees the caster. MagicSkillLaunched follows when the skill takes
// effect.
type MagicSkillUse struct {
	CasterID int32
	TargetID int32 // the caster itself for a skill on oneself
	// SkillID and SkillLevel are the skill to show: Mobius sends the
	// display ID and display level.
	SkillID    int32
	SkillLevel int32
	HitTime    time.Duration // the casting time
	ReuseDelay time.Duration
	X, Y, Z    int32 // the caster's position
	// Critical adds a zero short after the flag. Mobius has the field but
	// Interlude never sets it.
	Critical                  bool
	TargetX, TargetY, TargetZ int32
}

// NewMagicSkillUsePacket builds MagicSkillUse (Mobius
// MagicSkillUse.writeImpl). The times are sent in whole milliseconds.
func NewMagicSkillUsePacket(m MagicSkillUse) []byte {
	w := packets.NewWriter(55)
	w.WriteUint8(OpMagicSkillUse)
	w.WriteInt32(m.CasterID)
	w.WriteInt32(m.TargetID)
	w.WriteInt32(m.SkillID)
	w.WriteInt32(m.SkillLevel)
	w.WriteInt32(skillTimeMillis(m.HitTime))
	w.WriteInt32(skillTimeMillis(m.ReuseDelay))
	w.WriteInt32(m.X)
	w.WriteInt32(m.Y)
	w.WriteInt32(m.Z)
	if m.Critical {
		w.WriteInt32(1)
		w.WriteUint16(0)
	} else {
		w.WriteInt32(0)
	}
	w.WriteInt32(m.TargetX)
	w.WriteInt32(m.TargetY)
	w.WriteInt32(m.TargetZ)
	return w.Bytes()
}

// skillTimeMillis converts d to the int milliseconds Mobius keeps its skill
// times in, cut toward zero.
func skillTimeMillis(d time.Duration) int32 {
	return int32(d.Milliseconds()) //nolint:gosec // Java (int) cast keeps the low bits
}
