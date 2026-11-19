package serverpackets

import "testing"

// Mobius AcquireSkillList.writeImpl:
//
//	nothing at all when the list is empty, else:
//	c 0x8A
//	d skill type ordinal
//	d skills.size()
//	per skill d id, d nextLevel, d maxLevel, d spCost, d requirements

func TestAcquireSkillListPacket(t *testing.T) {
	// Class skills, as a trainer offers them: Mobius sends the level to
	// learn as the max level too, and 0 requirements.
	l := AcquireSkillList{
		Type: AcquireSkillClass,
		Skills: []AcquireSkillListEntry{
			{ID: 3, NextLevel: 2, MaxLevel: 2, SPCost: 470},
			{ID: 16, NextLevel: 1, MaxLevel: 1, SPCost: 50},
		},
	}
	m67Check(t, "class skills", NewAcquireSkillListPacket(l), m67Hex(t, `
		8a
		00000000
		02000000
		03000000 02000000 02000000 d6010000 00000000
		10000000 01000000 01000000 32000000 00000000`))
}

func TestAcquireSkillListPacketTypes(t *testing.T) {
	fishing := AcquireSkillList{
		Type:   AcquireSkillFishing,
		Skills: []AcquireSkillListEntry{{ID: 1312, NextLevel: 1, MaxLevel: 1, SPCost: 0, Requirements: 1}},
	}
	var g m67Golden
	g.c(0x8a).d(1).d(1)
	g.d(1312).d(1).d(1).d(0).d(1)
	m67Check(t, "fishing", NewAcquireSkillListPacket(fishing), g.b)

	pledge := AcquireSkillList{
		Type:   AcquireSkillPledge,
		Skills: []AcquireSkillListEntry{{ID: 370, NextLevel: 3, MaxLevel: 3, SPCost: 30000, Requirements: 5}},
	}
	g = m67Golden{}
	g.c(0x8a).d(2).d(1)
	g.d(370).d(3).d(3).d(30000).d(5)
	m67Check(t, "pledge", NewAcquireSkillListPacket(pledge), g.b)
}

// Mobius writes nothing for an empty list, and an empty packet is not sent.
func TestAcquireSkillListPacketEmpty(t *testing.T) {
	for _, l := range []AcquireSkillList{
		{},
		{Type: AcquireSkillPledge},
		{Type: AcquireSkillFishing, Skills: []AcquireSkillListEntry{}},
	} {
		if b := NewAcquireSkillListPacket(l); b != nil {
			t.Errorf("%+v: got % x, want nil", l, b)
		}
	}
}
