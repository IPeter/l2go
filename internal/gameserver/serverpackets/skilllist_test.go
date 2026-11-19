package serverpackets

import "testing"

func TestSkillListPacket(t *testing.T) {
	// Mobius SkillList.writeImpl: c 0x58, d count, then per skill
	// d passive, d level, d ID, c disabled.
	m3Check(t, "empty SkillList", NewSkillListPacket(nil), unhex(t, "58 00000000"))

	skills := []SkillListEntry{
		{ID: 194, Level: 1, Passive: true},
		{ID: 1177, Level: 5, Disabled: true},
	}
	var g m3Golden
	g.c(0x58).d(2)
	g.d(1).d(1).d(194).c(0)
	g.d(0).d(5).d(1177).c(1)
	m3Check(t, "SkillList", NewSkillListPacket(skills), g.b)
}
