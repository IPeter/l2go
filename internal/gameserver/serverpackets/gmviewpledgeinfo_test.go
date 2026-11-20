package serverpackets

import "testing"

func TestGMViewPledgeInfoPacket(t *testing.T) {
	// Mobius GMViewPledgeInfo.writeImpl: c 0x90, S player name, d clan ID,
	// d 0, S clan name, S leader name, d crest, d level, d castle, d hideout,
	// d rank, d reputation, d 0, d 0, d ally ID, S ally name, d ally crest,
	// d at war, d member count, then per member S name, d level, d class,
	// d sex, d race, d object ID (0 offline), d has sponsor.
	c := ClanInfo{
		ID: 0x10000100, Name: "Dragons", LeaderName: "Leader", CrestID: 7, Level: 3,
		Rank: 2, Reputation: 150, AllyID: 0x10000100, AllyName: "Fire", AllyCrestID: 8, AtWar: true,
	}
	members := []PledgeMember{
		{Name: "Leader", Level: 40, ClassID: 2, Race: 0, ObjectID: 0x10000001, Online: true},
		{Name: "Bob", Level: 20, ClassID: 18, Female: true, Race: 1, ObjectID: 0x10000002, HasSponsor: true},
	}
	var g m3Golden
	g.c(0x90).s("Leader").d(0x10000100).d(0).s("Dragons").s("Leader")
	g.d(7).d(3).d(0).d(0).d(2).d(150).d(0).d(0).d(0x10000100).s("Fire").d(8).d(1)
	g.d(2)
	g.s("Leader").d(40).d(2).d(0).d(0).d(0x10000001).d(0)
	g.s("Bob").d(20).d(18).d(1).d(1).d(0).d(1)
	m3Check(t, "GMViewPledgeInfo", NewGMViewPledgeInfoPacket("Leader", c, members), g.b)
}
