package serverpackets

import "testing"

func TestHennaEquipListPacket(t *testing.T) {
	// Mobius HennaEquipList.writeImpl: c 0xE2, d adena, d 3, d size of the
	// class's list, then per dye carried d symbol, d item, d wear count,
	// d wear fee, d 1.
	m3Check(t, "empty HennaEquipList", NewHennaEquipListPacket(HennaEquipList{Adena: 100, Listed: 26}),
		unhex(t, "e2 64000000 03000000 1a000000"))

	l := HennaEquipList{Adena: 50000, Listed: 26, Entries: []HennaEntry{
		{DyeID: 1, DyeItemID: 4445, Count: 10, Fee: 37000},
		{DyeID: 2, DyeItemID: 4446, Count: 10, Fee: 37000},
	}}
	var g m3Golden
	g.c(0xe2).d(50000).d(3).d(26)
	g.d(1).d(4445).d(10).d(37000).d(1)
	g.d(2).d(4446).d(10).d(37000).d(1)
	m3Check(t, "HennaEquipList", NewHennaEquipListPacket(l), g.b)
}

func TestHennaRemoveListPacket(t *testing.T) {
	// Mobius HennaRemoveList.writeImpl: c 0xE5, d adena, d empty slots,
	// d |empty slots - 3|, then per worn dye d symbol, d item, d cancel
	// count, d cancel fee, d 1.
	l := HennaRemoveList{Adena: 9000, EmptySlots: 1, Entries: []HennaEntry{
		{DyeID: 1, DyeItemID: 4445, Count: 5, Fee: 7400},
	}}
	var g m3Golden
	g.c(0xe5).d(9000).d(1).d(2) // a first class with one dye: 1 empty, "2"
	g.d(1).d(4445).d(5).d(7400).d(1)
	m3Check(t, "HennaRemoveList", NewHennaRemoveListPacket(l), g.b)

	m3Check(t, "HennaRemoveList of a base class", NewHennaRemoveListPacket(HennaRemoveList{Adena: 1}),
		unhex(t, "e5 01000000 00000000 03000000"))
}

func TestHennaItemInfoPackets(t *testing.T) {
	// Mobius HennaItemDrawInfo and HennaItemRemoveInfo.writeImpl: c 0xE3 or
	// 0xE6, d symbol, d item, d count, d fee, d allowed, d adena, then for
	// INT, STR, CON, MEN, DEX, WIT: d now, c after.
	i := HennaItemInfo{
		DyeID: 1, DyeItemID: 4445, Count: 10, Fee: 37000, Allowed: true, Adena: 40000,
		INT: HennaStat{21, 21}, STR: HennaStat{40, 41}, CON: HennaStat{43, 40},
		MEN: HennaStat{25, 25}, DEX: HennaStat{30, 30}, WIT: HennaStat{11, 11},
	}
	golden := func(op byte, allowed int32) []byte {
		var g m3Golden
		g.c(op).d(1).d(4445).d(10).d(37000).d(allowed).d(40000)
		g.d(21).c(21).d(40).c(41).d(43).c(40).d(25).c(25).d(30).c(30).d(11).c(11)
		return g.b
	}
	m3Check(t, "HennaItemDrawInfo", NewHennaItemDrawInfoPacket(i), golden(0xe3, 1))
	i.Allowed = false
	m3Check(t, "HennaItemRemoveInfo", NewHennaItemRemoveInfoPacket(i), golden(0xe6, 0))

	// writeByte keeps the low 8 bits: -2 is 0xFE, 300 is 0x2C.
	i.STR, i.CON = HennaStat{1, -2}, HennaStat{299, 300}
	got := NewHennaItemRemoveInfoPacket(i)
	if len(got) != 55 || got[34] != 0xfe || got[39] != 0x2c {
		t.Errorf("% x: want 55 bytes, STR after fe, CON after 2c", got)
	}
}

func TestHennaOpcodes(t *testing.T) {
	for _, tt := range []struct {
		name      string
		got, want int
	}{
		{"HennaEquipList", OpHennaEquipList, 0xe2},
		{"HennaItemInfo", OpHennaItemInfo, 0xe3},
		{"HennaInfo", OpHennaInfo, 0xe4},
		{"HennaRemoveList", OpHennaRemoveList, 0xe5},
		{"HennaItemRemoveInfo", OpHennaItemRemoveInfo, 0xe6},
	} {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}
