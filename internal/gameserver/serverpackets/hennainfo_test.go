package serverpackets

import "testing"

func TestHennaInfoPacket(t *testing.T) {
	// Mobius HennaInfo.writeImpl: c 0xE4, c INT, c STR, c CON, c MEN,
	// c DEX, c WIT, d 3 (slots), d count, then per dye d dye ID, d 1.
	m3Check(t, "empty HennaInfo", NewHennaInfoPacket(HennaInfo{}),
		unhex(t, "e4 00 00 00 00 00 00 03000000 00000000"))

	h := HennaInfo{INT: 1, STR: -3, CON: 2, MEN: 0, DEX: -1, WIT: 4, DyeIDs: []int32{21, 180}}
	var g m3Golden
	g.c(0xe4).c(1).c(0xfd).c(2).c(0).c(0xff).c(4)
	g.d(3).d(2).d(21).d(1).d(180).d(1)
	m3Check(t, "HennaInfo", NewHennaInfoPacket(h), g.b)
}
