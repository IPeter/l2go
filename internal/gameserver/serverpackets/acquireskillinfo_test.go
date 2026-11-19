package serverpackets

import "testing"

// Mobius AcquireSkillInfo.writeImpl:
//
//	c 0x8B
//	d id, d level, d spCost, d skill type ordinal
//	d reqs.size()
//	per required item: d 99, d itemId, q count, d 50
//
// The count is a long in Mobius CT_0 Interlude.

func TestAcquireSkillInfoPacket(t *testing.T) {
	// Attack Aura 1 for 920 SP and a spellbook, the sample in the Mobius
	// source.
	a := AcquireSkillInfo{
		SkillID: 77, Level: 1, SPCost: 920, Type: AcquireSkillClass,
		Items: []AcquireSkillItem{{ItemID: 1095, Count: 1}},
	}
	m67Check(t, "spellbook", NewAcquireSkillInfoPacket(a), m67Hex(t, `
		8b
		4d000000 01000000 98030000
		00000000
		01000000
		63000000 47040000 0100000000000000 32000000`))
}

func TestAcquireSkillInfoPacketItems(t *testing.T) {
	// A clan skill with two items, one count beyond an int.
	a := AcquireSkillInfo{
		SkillID: 370, Level: 3, SPCost: 30000, Type: AcquireSkillPledge,
		Items: []AcquireSkillItem{
			{ItemID: 8176, Count: 2},
			{ItemID: 57, Count: 5_000_000_000},
		},
	}
	var g m67Golden
	g.c(0x8b).d(370).d(3).d(30000).d(2)
	g.d(2)
	g.d(99).d(8176).q(2).d(50)
	g.d(99).d(57).q(5_000_000_000).d(50)
	m67Check(t, "clan skill", NewAcquireSkillInfoPacket(a), g.b)

	// No items: only the count of 0.
	g = m67Golden{}
	g.c(0x8b).d(1312).d(1).d(0).d(1).d(0)
	m67Check(t, "no items",
		NewAcquireSkillInfoPacket(AcquireSkillInfo{SkillID: 1312, Level: 1, Type: AcquireSkillFishing}), g.b)
}
