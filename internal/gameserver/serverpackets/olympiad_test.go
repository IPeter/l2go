package serverpackets

import (
	"testing"
	"unicode/utf16"
)

// olyStr appends a Mobius string: UTF-16LE with a 0 terminator.
func olyStr(g *m67Golden, s string) *m67Golden {
	for _, u := range utf16.Encode([]rune(s)) {
		g.h(int16(u))
	}
	return g.h(0)
}

// The Olympiad packets of Mobius (c 0xFE, h sub-opcode; ObservationEnter
// 0xDF and ObservationExit 0xE0 plain).
func TestOlympiadPackets(t *testing.T) {
	var g m67Golden
	g.c(0xfe).h(0x23).d(2)
	olyStr(&g, "Duelist").d(88)
	olyStr(&g, "Blades").d(5)
	olyStr(&g, "").d(0).d(2)
	olyStr(&g, "Sage").d(94)
	olyStr(&g, "").d(0)
	olyStr(&g, "").d(0).d(1)
	m67Check(t, "ExHeroList", NewExHeroListPacket([]HeroListEntry{
		{Name: "Duelist", ClassID: 88, ClanName: "Blades", ClanCrestID: 5, Count: 2},
		{Name: "Sage", ClassID: 94, Count: 1},
	}), g.b)
	m67Check(t, "ExHeroList empty", NewExHeroListPacket(nil), m67Hex(t, "fe 2300 00000000"))

	g = m67Golden{}
	g.c(0xfe).h(0x29).c(1).d(0x10000001)
	olyStr(&g, "Ab").d(88).d(1234).d(2000).d(500).d(800)
	m67Check(t, "ExOlympiadUserInfo", NewExOlympiadUserInfoPacket(OlympiadUserInfo{
		Side: 1, ObjectID: 0x10000001, Name: "Ab", ClassID: 88, HP: 1234, MaxHP: 2000, CP: 500, MaxCP: 800,
	}), g.b)

	g = m67Golden{}
	g.c(0xfe).h(0x2a).d(7).d(2).d(1204).h(2).d(1150).d(1086).h(1).d(-1)
	m67Check(t, "ExOlympiadSpelledInfo", NewExOlympiadSpelledInfoPacket(7, []SpelledEffect{
		{SkillID: 1204, Level: 2, Seconds: 1150}, {SkillID: 1086, Level: 1, Seconds: -1},
	}), g.b)

	m67Check(t, "ExOlympiadMode", NewExOlympiadModePacket(OlympiadModeSpectator), m67Hex(t, "fe 2b00 03"))
	m67Check(t, "ExOlympiadMatchEnd", NewExOlympiadMatchEndPacket(), m67Hex(t, "fe 2c00"))
	m67Check(t, "ObservationEnter", NewObservationEnterPacket(-20814, -21189, -3030),
		m67Hex(t, "df b2aeffff 3badffff 2af4ffff 00 c0 00"))
	m67Check(t, "ObservationExit", NewObservationExitPacket(1, -2, 3), m67Hex(t, "e0 01000000 feffffff 03000000"))
}
