package serverpackets

import (
	"time"

	"github.com/IPeter/l2go/internal/packets"
)

// OpSkillCoolTime is the opcode of SkillCoolTime.
const OpSkillCoolTime = 0xc1

// SkillCoolTimeEntry is one skill that cannot be used yet.
type SkillCoolTimeEntry struct {
	SkillID    int32
	SkillLevel int32
	Reuse      time.Duration // the full reuse delay; 0 if unknown
	Remaining  time.Duration // the time left
}

// NewSkillCoolTimePacket builds SkillCoolTime (Mobius
// SkillCoolTime.writeImpl). An empty list is valid.
//
// Times are sent in whole seconds, the way Mobius computes them: the reuse
// (or the remaining time when the reuse is 0) is cut to a Java int of
// milliseconds before the division, and the remaining time is at least 1.
func NewSkillCoolTimePacket(entries []SkillCoolTimeEntry) []byte {
	w := packets.NewWriter(5 + len(entries)*16)
	w.WriteUint8(OpSkillCoolTime)
	putCount32(w, len(entries))
	for _, e := range entries {
		reuse := e.Reuse.Milliseconds()
		remaining := e.Remaining.Milliseconds()
		if reuse <= 0 {
			reuse = remaining
		}
		w.WriteInt32(e.SkillID)
		w.WriteInt32(e.SkillLevel)
		w.WriteInt32(int32(reuse) / 1000)           //nolint:gosec // Java (int) cast keeps the low bits
		w.WriteInt32(max(1, int32(remaining)/1000)) //nolint:gosec // as above
	}
	return w.Bytes()
}
