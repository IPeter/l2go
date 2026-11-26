package testclient

import (
	"errors"
	"reflect"
	"testing"
)

// The henna windows decode with the Mobius layouts, the entries up to the
// end of the packet (their counts are not the entry counts).
func TestHennaPackets(t *testing.T) {
	entry := func(w *wire, dye, item, count, fee int32) *wire {
		return w.i32(dye).i32(item).i32(count).i32(fee).i32(1)
	}
	stats := func(w *wire) *wire {
		return w.i32(21).u8(21).i32(40).u8(41).i32(43).u8(40).i32(25).u8(25).i32(30).u8(30).i32(11).u8(11)
	}
	info := HennaItemInfo{
		DyeID: 1, DyeItemID: 4445, Count: 10, Fee: 37000, Allowed: true, Adena: 40000,
		INT: HennaStat{21, 21}, STR: HennaStat{40, 41}, CON: HennaStat{43, 40},
		MEN: HennaStat{25, 25}, DEX: HennaStat{30, 30}, WIT: HennaStat{11, 11},
	}
	tests := []struct {
		name string
		raw  []byte
		want Packet
	}{
		{
			// HennaEquipList.writeImpl: d adena, d 3, d list size, then per
			// carried dye d dye, d item, d count, d fee, d 1.
			"HennaEquipList",
			entry(msg(0xe2).i32(50000).i32(3).i32(26), 1, 4445, 10, 37000).b,
			HennaEquipList{Adena: 50000, Slots: 3, Listed: 26, Entries: []HennaEntry{{1, 4445, 10, 37000, 1}}},
		},
		{"HennaEquipList, nothing carried", msg(0xe2).i32(5).i32(3).i32(26).b, HennaEquipList{Adena: 5, Slots: 3, Listed: 26}},
		{
			// HennaRemoveList.writeImpl: d adena, d empty, d |empty - 3|, then
			// per worn dye d dye, d item, d cancel count, d cancel fee, d 1.
			"HennaRemoveList",
			entry(entry(msg(0xe5).i32(9000).i32(0).i32(3), 1, 4445, 5, 7400), 2, 4446, 5, 7400).b,
			HennaRemoveList{Adena: 9000, EmptySlots: 0, Used: 3, Entries: []HennaEntry{{1, 4445, 5, 7400, 1}, {2, 4446, 5, 7400, 1}}},
		},
		{
			"HennaItemDrawInfo",
			stats(msg(0xe3).i32(1).i32(4445).i32(10).i32(37000).i32(1).i32(40000)).b,
			HennaItemDrawInfo{info},
		},
		{
			"HennaItemRemoveInfo",
			stats(msg(0xe6).i32(1).i32(4445).i32(10).i32(37000).i32(1).i32(40000)).b,
			HennaItemRemoveInfo{info},
		},
	}
	for _, tt := range tests {
		got, err := DecodeServerPacket(tt.raw)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", tt.name, got, tt.want)
		}
		if got.PacketName() != tt.want.PacketName() || serverPacketName(tt.raw) != tt.want.PacketName() {
			t.Errorf("%s: name %q", tt.name, got.PacketName())
		}
	}

	// A cut entry and a bad flag are malformed.
	for name, raw := range map[string][]byte{
		"cut entry":  msg(0xe2).i32(1).i32(3).i32(1).i32(1).i32(4445).b,
		"bad flag":   stats(msg(0xe3).i32(1).i32(4445).i32(10).i32(37000).i32(2).i32(40000)).b,
		"cut stats":  msg(0xe6).i32(1).i32(4445).i32(10).i32(37000).i32(1).i32(40000).i32(5).b,
		"cut header": msg(0xe5).i32(1).b,
	} {
		if _, err := DecodeServerPacket(raw); !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: error %v, want ErrMalformed", name, err)
		}
	}
}

func TestHennaRequests(t *testing.T) {
	for _, tt := range []struct {
		op   byte
		name string
	}{
		{0xba, "RequestHennaItemList"},
		{0xbb, "RequestHennaItemInfo"},
		{0xbc, "RequestHennaEquip"},
		{0xbd, "RequestHennaRemoveList"},
		{0xbe, "RequestHennaItemRemoveInfo"},
		{0xbf, "RequestHennaRemove"},
	} {
		raw := encodeInt(tt.op, 7)
		if want := msg(tt.op).i32(7).b; !reflect.DeepEqual(raw, want) {
			t.Errorf("%s = % x, want % x", tt.name, raw, want)
		}
		if got := clientPacketName(raw); got != tt.name {
			t.Errorf("clientPacketName(%#x) = %q, want %q", tt.op, got, tt.name)
		}
	}
}
