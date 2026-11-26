package testclient

import (
	"bytes"
	"reflect"
	"testing"
)

// rewards writes a seed's level and its crop's rewards: d level, c 1, d
// reward 1, c 1, d reward 2.
func (w *wire) manorRewards(level, r1, r2 int32) *wire {
	return w.i32(level).u8(1).i32(r1).u8(1).i32(r2)
}

func manorEx(sub uint16) *wire { return msg(0xfe).u16(sub) }

// The manor packets decode from the Mobius writeImpl layouts.
func TestDecodeManorPackets(t *testing.T) {
	rw := ManorRewards{Level: 10, Reward1: 1864, Reward2: 1878}
	for _, tt := range []struct {
		name string
		raw  []byte
		want Packet
	}{
		{"ExShowSeedInfo", manorEx(0x1c).u8(1).i32(1).i32(0).i32(1).i32(5016).i32(40).i32(100).i32(10).manorRewards(10, 1864, 1878).b,
			ExShowSeedInfo{HideButtons: true, ManorID: 1, Seeds: []ManorSeed{{5016, 40, 100, 10, rw}}}},
		{"ExShowCropInfo", manorEx(0x1d).u8(0).i32(2).i32(0).i32(1).i32(5073).i32(3).i32(10).i32(100).u8(2).manorRewards(10, 1864, 1878).b,
			ExShowCropInfo{ManorID: 2, Crops: []ManorCrop{{5073, 3, 10, 100, 2, rw}}}},
		{"ExShowManorDefaultInfo", manorEx(0x1e).u8(1).i32(1).i32(5073).i32(10).i32(5).i32(100).u8(1).i32(1864).u8(1).i32(1878).b,
			ExShowManorDefaultInfo{HideButtons: true, Crops: []ManorDefaultCrop{{5073, 10, 5, 100, 1864, 1878}}}},
		{"ExShowSeedSetting", manorEx(0x1f).i32(1).i32(1).i32(5016).manorRewards(10, 1864, 1878).
			i32(8100).i32(5).i32(3).i32(50).i32(100).i32(10).i32(0).i32(0).b,
			ExShowSeedSetting{ManorID: 1, Seeds: []SeedSetting{{SeedID: 5016, ManorRewards: rw, Limit: 8100, ReferencePrice: 5, MinPrice: 3, MaxPrice: 50, Start: 100, Price: 10}}}},
		{"ExShowCropSetting", manorEx(0x20).i32(1).i32(1).i32(5073).manorRewards(10, 1864, 1878).
			i32(9000).i32(0).i32(60).i32(1000).i32(0).i32(0).u8(0).i32(10).i32(100).u8(1).b,
			ExShowCropSetting{ManorID: 1, Crops: []CropSetting{{CropID: 5073, ManorRewards: rw, Limit: 9000, MinPrice: 60, MaxPrice: 1000, NextStart: 10, NextPrice: 100, NextReward: 1}}}},
		{"ExShowSellCropList", manorEx(0x21).i32(1).i32(1).i32(0x10000101).i32(5073).manorRewards(10, 1864, 1878).
			i32(-1).i32(0).i32(0).u8(0).i32(3).b,
			ExShowSellCropList{ManorID: 1, Crops: []SellCrop{{ObjectID: 0x10000101, CropID: 5073, ManorRewards: rw, ManorID: -1, Count: 3}}}},
		{"ExShowProcureCropDetail", manorEx(0x22).i32(5073).i32(1).i32(1).i32(7).i32(100).u8(1).b,
			ExShowProcureCropDetail{CropID: 5073, Castles: []ProcureCastle{{1, 7, 100, 1}}}},
		{"BuyListSeed", msg(0xe8).i32(12345).i32(1).u16(1).u16(4).i32(0).i32(5016).i32(40).u16(4).u16(0).i32(10).b,
			BuyListSeed{Money: 12345, ManorID: 1, Seeds: []SeedOnSale{{5016, 40, 10}}}},
		{"SellListProcure", msg(0xe9).i32(500).i32(0).u16(1).u16(4).i32(0x10000101).i32(5073).i32(7).u16(5).u16(0).i32(0).b,
			SellListProcure{Money: 500, Items: []ProcureItem{{4, 0x10000101, 5073, 7, 5}}}},
	} {
		got, err := DecodeServerPacket(tt.raw)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", tt.name, got, tt.want)
		}
		if got.PacketName() != tt.name {
			t.Errorf("%s: name %q", tt.name, got.PacketName())
		}
	}
	// A reward type other than 1 is malformed.
	if _, err := DecodeServerPacket(manorEx(0x1e).u8(0).i32(1).i32(5073).i32(10).i32(5).i32(100).u8(2).i32(1864).u8(1).i32(1878).b); err == nil {
		t.Error("a reward type of 2 decoded")
	}
}

// The manor requests follow the Mobius readImpl layouts.
func TestEncodeManorRequests(t *testing.T) {
	for _, tt := range []struct {
		name      string
		got, want []byte
	}{
		{"RequestBuySeed", encodeRequestBuySeed(1, []ManorSeedOrder{{5016, 10}}), msg(0xc4).i32(1).i32(1).i32(5016).i32(10).b},
		{"RequestProcureCropList", encodeRequestProcureCropList([]CropSale{{0x10000101, 5073, 1, 7}}),
			msg(0xd0).u16(0x09).i32(1).i32(0x10000101).i32(5073).i32(1).i32(7).b},
		{"RequestSetSeed", encodeRequestSetSeed(1, []SeedPlan{{5016, 100, 10}}), msg(0xd0).u16(0x0a).i32(1).i32(1).i32(5016).i32(100).i32(10).b},
		{"RequestSetCrop", encodeRequestSetCrop(1, []CropPlan{{5073, 10, 100, 2}}), msg(0xd0).u16(0x0b).i32(1).i32(1).i32(5073).i32(10).i32(100).u8(2).b},
	} {
		if !bytes.Equal(tt.got, tt.want) {
			t.Errorf("%s:\n got % x\nwant % x", tt.name, tt.got, tt.want)
		}
		if name := clientPacketName(tt.got); name != tt.name {
			t.Errorf("%s: name %q", tt.name, name)
		}
	}
}
