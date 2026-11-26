package serverpackets

import "testing"

// Mobius ExEnchantSkillList.writeImpl: c 0xFE, h 0x17; d count; per skill
// d id, d nextLevel, d sp, q exp.
func TestExEnchantSkillListPacket(t *testing.T) {
	m67Check(t, "empty", NewExEnchantSkillListPacket(nil), m67Hex(t, "fe 1700 00000000"))

	got := NewExEnchantSkillListPacket([]EnchantSkillEntry{
		{ID: 1, NextLevel: 101, SP: 306000, Exp: 3060000},
		{ID: 1, NextLevel: 141, SP: 306000, Exp: 3060000},
	})
	var g m67Golden
	g.c(0xfe).h(0x17).d(2)
	g.d(1).d(101).d(306000).q(3060000)
	g.d(1).d(141).d(306000).q(3060000)
	m67Check(t, "triple slash", got, g.b)
	m67Check(t, "hex", NewExEnchantSkillListPacket([]EnchantSkillEntry{{ID: 1, NextLevel: 101, SP: 306000, Exp: 3060000}}),
		m67Hex(t, "fe 1700 01000000 01000000 65000000 50ab0400 20b12e0000000000"))
}

// Mobius ExEnchantSkillInfo.writeImpl: c 0xFE, h 0x18; d id, d level, d
// spCost, q xpCost, d rate, d count; per requirement d type, d id, d count,
// d unk.
func TestExEnchantSkillInfoPacket(t *testing.T) {
	got := NewExEnchantSkillInfoPacket(ExEnchantSkillInfo{
		ID: 1, Level: 101, SPCost: 306000, ExpCost: 3060000, Rate: 82,
		Requirements: []EnchantSkillRequirement{{Type: 4, ItemID: 6622, Count: 1}},
	})
	var g m67Golden
	g.c(0xfe).h(0x18).d(1).d(101).d(306000).q(3060000).d(82).d(1)
	g.d(4).d(6622).d(1).d(0)
	m67Check(t, "with the book", got, g.b)

	var g2 m67Golden
	g2.c(0xfe).h(0x18).d(1).d(102).d(315000).q(3150000).d(80).d(0)
	m67Check(t, "without", NewExEnchantSkillInfoPacket(ExEnchantSkillInfo{ID: 1, Level: 102, SPCost: 315000, ExpCost: 3150000, Rate: 80}), g2.b)

	if ExOpEnchantSkillList != 0x17 || ExOpEnchantSkillInfo != 0x18 {
		t.Error("sub-opcodes differ from Mobius ServerPackets")
	}
}
