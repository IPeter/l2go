package serverpackets

import "testing"

// Mobius PartySpelled.writeImpl:
//
//	c 0xEE
//	d isServitor ? 2 : isPet ? 1 : 0
//	d objectId
//	d effects.size()
//	per effect d display ID, h display level, d seconds left
//
// The count is an int here, a short in AbnormalStatusUpdate.

func TestPartySpelledPacket(t *testing.T) {
	p := PartySpelled{
		Type:     PartySpelledPlayer,
		ObjectID: 0x10000002,
		Effects: []AbnormalStatusEntry{
			{SkillID: 1204, SkillLevel: 2, Seconds: 1150},
			{SkillID: 1040, SkillLevel: 3, Seconds: 600},
		},
	}
	m67Check(t, "party member", NewPartySpelledPacket(p), m67Hex(t, `
		ee
		00000000
		02000010
		02000000
		b4040000 0200 7e040000
		10040000 0300 58020000`))
}

func TestPartySpelledPacketSummons(t *testing.T) {
	pet := PartySpelled{
		Type: PartySpelledPet, ObjectID: 0x10000100,
		Effects: []AbnormalStatusEntry{{SkillID: 4342, SkillLevel: 1, Seconds: -1}},
	}
	var g m67Golden
	g.c(0xee).d(1).d(0x10000100).d(1)
	g.d(4342).h(1).d(-1)
	m67Check(t, "pet", NewPartySpelledPacket(pet), g.b)

	// A servitor without effects.
	m67Check(t, "servitor",
		NewPartySpelledPacket(PartySpelled{Type: PartySpelledServitor, ObjectID: 0x60000200}),
		m67Hex(t, "ee 02000000 00020060 00000000"))
}
