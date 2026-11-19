package serverpackets

import (
	"testing"
	"time"
)

func TestSkillCoolTimePacket(t *testing.T) {
	// Mobius SkillCoolTime.writeImpl: c 0xC1, d count, then per skill
	// d skill ID, d level, d (int)(reuse > 0 ? reuse : remaining) / 1000,
	// d max(1, (int) remaining / 1000), times in milliseconds.
	m3Check(t, "empty SkillCoolTime", NewSkillCoolTimePacket(nil), unhex(t, "c1 00000000"))

	entries := []SkillCoolTimeEntry{
		{SkillID: 1177, SkillLevel: 5, Reuse: 30 * time.Second, Remaining: 12500 * time.Millisecond},
		{SkillID: 28, SkillLevel: 1, Reuse: 0, Remaining: 5999 * time.Millisecond},              // no reuse: the remaining time
		{SkillID: 16, SkillLevel: 2, Reuse: 2 * time.Second, Remaining: 400 * time.Millisecond}, // at least 1
	}
	var g m3Golden
	g.c(0xc1).d(3)
	g.d(1177).d(5).d(30).d(12)
	g.d(28).d(1).d(5).d(5)
	g.d(16).d(2).d(2).d(1)
	m3Check(t, "SkillCoolTime", NewSkillCoolTimePacket(entries), g.b)
}
