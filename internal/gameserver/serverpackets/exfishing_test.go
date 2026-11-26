package serverpackets

import "testing"

// The fishing packets of Mobius (ExFishingStart, ExFishingEnd,
// ExFishingStartCombat, ExFishingHpRegen; c 0xFE, h sub-opcode).
func TestExFishingPackets(t *testing.T) {
	// A night lure's float at a fish of group 1.
	m67Check(t, "ExFishingStart",
		NewExFishingStartPacket(ExFishingStart{ObjectID: 0x10000001, FishType: 1, X: -80000, Y: 150000, Z: -3060, Night: true}),
		m67Hex(t, "fe 1300 01000010 01000000 80c7feff f0490200 0cf4ffff 01 00"))
	var g m67Golden
	g.c(0xfe).h(0x13).d(7).d(-1).d(1).d(2).d(3).c(0).c(0)
	m67Check(t, "ExFishingStart, wrong time of day", NewExFishingStartPacket(ExFishingStart{ObjectID: 7, FishType: -1, X: 1, Y: 2, Z: 3}), g.b)

	m67Check(t, "ExFishingEnd won",
		NewExFishingEndPacket(ExFishingEnd{ObjectID: 0x10000001, Win: true}),
		m67Hex(t, "fe 1400 01000010 01"))
	g = m67Golden{}
	g.c(0xfe).h(0x14).d(9).c(0)
	m67Check(t, "ExFishingEnd lost", NewExFishingEndPacket(ExFishingEnd{ObjectID: 9}), g.b)

	m67Check(t, "ExFishingStartCombat",
		NewExFishingStartCombatPacket(ExFishingStartCombat{ObjectID: 0x10000001, Time: 24, HP: 100, Mode: 1, LureType: 2, Deceptive: 1}),
		m67Hex(t, "fe 1500 01000010 18000000 64000000 01 02 01"))

	m67Check(t, "ExFishingHpRegen",
		NewExFishingHpRegenPacket(ExFishingHpRegen{ObjectID: 0x10000001, Time: 23, HP: 60, Mode: 0, GoodUse: 1, Anim: 1, Penalty: 2, BarColor: 1}),
		m67Hex(t, "fe 1600 01000010 17000000 3c000000 00 01 01 02000000 01"))
	g = m67Golden{}
	g.c(0xfe).h(0x16).d(5).d(0).d(208).c(1).c(2).c(2).d(0).c(0)
	m67Check(t, "ExFishingHpRegen failed reeling", NewExFishingHpRegenPacket(ExFishingHpRegen{ObjectID: 5, HP: 208, Mode: 1, GoodUse: 2, Anim: 2}), g.b)
}
