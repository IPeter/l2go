package serverpackets

import "testing"

func TestCharSelectedPacket(t *testing.T) {
	c := CharSelected{
		Name: "Frostwind", ObjectID: 0x10000001, Title: "Hero of Elmore",
		SessionID: -559038737, ClanID: 77, Female: true, Race: 1, ClassID: 25,
		X: -71338, Y: 258271, Z: -3104,
		CurHP: 89.5, CurMP: 30.25, SP: 1234, Exp: 9876543210, Level: 12,
		Karma: 3, PKKills: 2,
		INT: 21, STR: 22, CON: 23, MEN: 24, DEX: 25, WIT: 26,
		GameTime: 1500, // 1:00 on the second game day
	}

	// Mobius CharSelected.writeImpl: c 0x15, S name, d object ID, S title,
	// d session ID, d clan ID, d 0, d female, d race, d class ID, d 1
	// (active), d x, y, z, f current HP, f current MP, d SP, q exp, d level,
	// d karma, d PK kills, d INT, STR, CON, MEN, DEX, WIT, 30 × d 0, d 0,
	// d 0, d game time % 1440, d 0, d class ID, 12 × d 0.
	var g m3Golden
	g.c(0x15).s("Frostwind").d(0x10000001).s("Hero of Elmore")
	g.d(-559038737).d(77).d(0)
	g.d(1).d(1).d(25).d(1)
	g.d(-71338).d(258271).d(-3104)
	g.f(89.5).f(30.25)
	g.d(1234).q(9876543210).d(12).d(3).d(2)
	g.d(21).d(22).d(23).d(24).d(25).d(26)
	g.dn(30)
	g.d(0).d(0)
	g.d(60) // 1500 % 1440
	g.d(0).d(25)
	g.dn(12)
	m3Check(t, "CharSelected", NewCharSelectedPacket(c), g.b)
}

func TestCharSelectedPacketZero(t *testing.T) {
	var g m3Golden
	g.c(0x15).s("").d(0).s("").d(0).d(0).d(0)
	g.d(0).d(0).d(0).d(1)
	g.d(0).d(0).d(0).f(0).f(0)
	g.d(0).q(0).d(0).d(0).d(0)
	g.dn(6).dn(30).d(0).d(0).d(0).d(0).d(0).dn(12)
	m3Check(t, "zero CharSelected", NewCharSelectedPacket(CharSelected{}), g.b)
}
