package serverpackets

import "testing"

// The manor packets against Mobius's writeImpl, field by field.

var manorRewards = SeedRewards{Level: 10, Reward1: 1864, Reward2: 1878}

// rewards appends a seed's level and rewards: d level, c 1, d reward 1, c 1,
// d reward 2.
func (g *m3Golden) rewards() *m3Golden { return g.d(10).c(1).d(1864).c(1).d(1878) }

func TestExShowSeedInfoPacket(t *testing.T) {
	// c 0xFE, h 0x1C, c hide, d manor, d 0, d count; per seed d id, d amount,
	// d start, d price, the rewards.
	var g m3Golden
	g.c(0xfe).h(0x1c).c(1).d(1).d(0).d(2)
	g.d(5016).d(40).d(100).d(10).rewards()
	g.d(5650).d(0).d(5).d(30).rewards()
	m3Check(t, "ExShowSeedInfo", NewExShowSeedInfoPacket(ExShowSeedInfo{HideButtons: true, ManorID: 1, Seeds: []SeedInfo{
		{SeedID: 5016, Amount: 40, StartAmount: 100, Price: 10, SeedRewards: manorRewards},
		{SeedID: 5650, Amount: 0, StartAmount: 5, Price: 30, SeedRewards: manorRewards},
	}}), g.b)
	var empty m3Golden
	empty.c(0xfe).h(0x1c).c(0).d(3).d(0).d(0)
	m3Check(t, "ExShowSeedInfo without seeds", NewExShowSeedInfoPacket(ExShowSeedInfo{ManorID: 3}), empty.b)
}

func TestExShowCropInfoPacket(t *testing.T) {
	// ... per crop d id, d amount, d start, d price, c reward, the rewards.
	var g m3Golden
	g.c(0xfe).h(0x1d).c(0).d(2).d(0).d(1)
	g.d(5073).d(3).d(10).d(100).c(2).rewards()
	m3Check(t, "ExShowCropInfo", NewExShowCropInfoPacket(ExShowCropInfo{ManorID: 2, Crops: []CropInfo{
		{CropID: 5073, Amount: 3, StartAmount: 10, Price: 100, Reward: 2, SeedRewards: manorRewards},
	}}), g.b)
}

func TestExShowManorDefaultInfoPacket(t *testing.T) {
	// c 0xFE, h 0x1E, c hide, d count; per crop d crop, d level, d seed
	// price, d crop price, c 1, d reward 1, c 1, d reward 2.
	var g m3Golden
	g.c(0xfe).h(0x1e).c(1).d(1)
	g.d(5073).d(10).d(5).d(100).c(1).d(1864).c(1).d(1878)
	m3Check(t, "ExShowManorDefaultInfo", NewExShowManorDefaultInfoPacket(true, []ManorDefaultCrop{
		{CropID: 5073, Level: 10, SeedPrice: 5, CropPrice: 100, Reward1: 1864, Reward2: 1878},
	}), g.b)
}

func TestExShowSeedSettingPacket(t *testing.T) {
	// c 0xFE, h 0x1F, d manor, d count; per seed d id, the rewards, d limit,
	// d reference price, d min, d max, d start, d price, d next start, d next
	// price.
	var g m3Golden
	g.c(0xfe).h(0x1f).d(1).d(1)
	g.d(5016).rewards().d(8100).d(5).d(3).d(50).d(100).d(10).d(0).d(0)
	m3Check(t, "ExShowSeedSetting", NewExShowSeedSettingPacket(1, []SeedSetting{
		{SeedID: 5016, SeedRewards: manorRewards, Limit: 8100, ReferencePrice: 5, MinPrice: 3, MaxPrice: 50, Start: 100, Price: 10},
	}), g.b)
}

func TestExShowCropSettingPacket(t *testing.T) {
	// c 0xFE, h 0x20, d manor, d count; per crop d id, the rewards, d limit,
	// d 0, d min, d max, d start, d price, c reward, d next start, d next
	// price, c next reward.
	var g m3Golden
	g.c(0xfe).h(0x20).d(1).d(1)
	g.d(5073).rewards().d(9000).d(0).d(60).d(1000).d(0).d(0).c(0).d(10).d(100).c(1)
	m3Check(t, "ExShowCropSetting", NewExShowCropSettingPacket(1, []CropSetting{
		{CropID: 5073, SeedRewards: manorRewards, Limit: 9000, MinPrice: 60, MaxPrice: 1000, NextStart: 10, NextPrice: 100, NextReward: 1},
	}), g.b)
}

func TestExShowSellCropListPacket(t *testing.T) {
	// c 0xFE, h 0x21, d manor, d count; per crop d object, d crop, the
	// rewards, then d manor, d amount, d price, c reward (-1, 0, 0, 0 when
	// the manor does not buy it), d count.
	var g m3Golden
	g.c(0xfe).h(0x21).d(1).d(2)
	g.d(0x10000101).d(5073).rewards().d(1).d(7).d(100).c(1).d(12)
	g.d(0x10000102).d(5818).rewards().d(-1).d(0).d(0).c(0).d(3)
	m3Check(t, "ExShowSellCropList", NewExShowSellCropListPacket(1, []SellCrop{
		{ObjectID: 0x10000101, CropID: 5073, SeedRewards: manorRewards, Procured: true, ManorID: 1, Amount: 7, Price: 100, Reward: 1, Count: 12},
		{ObjectID: 0x10000102, CropID: 5818, SeedRewards: manorRewards, Count: 3},
	}), g.b)
}

func TestExShowProcureCropDetailPacket(t *testing.T) {
	// c 0xFE, h 0x22, d crop, d count; per castle d castle, d amount, d
	// price, c reward.
	var g m3Golden
	g.c(0xfe).h(0x22).d(5073).d(2).d(1).d(7).d(100).c(1).d(2).d(9).d(90).c(2)
	m3Check(t, "ExShowProcureCropDetail", NewExShowProcureCropDetailPacket(5073, []ProcureCastle{
		{CastleID: 1, Amount: 7, Price: 100, Reward: 1}, {CastleID: 2, Amount: 9, Price: 90, Reward: 2},
	}), g.b)
}

func TestBuyListSeedPacket(t *testing.T) {
	// c 0xE8, d money, d manor, h count; per seed h 4, d 0, d id, d amount,
	// h 4, h 0, d price.
	var g m3Golden
	g.c(0xe8).d(12345).d(1).h(1).h(4).d(0).d(5016).d(40).h(4).h(0).d(10)
	m3Check(t, "BuyListSeed", NewBuyListSeedPacket(12345, 1, []BuyListSeedEntry{{SeedID: 5016, Amount: 40, Price: 10}}), g.b)
	var empty m3Golden
	empty.c(0xe8).d(0).d(2).h(0)
	m3Check(t, "empty BuyListSeed", NewBuyListSeedPacket(0, 2, nil), empty.b)
}

func TestSellListProcurePacket(t *testing.T) {
	// c 0xE9, d money, d 0, h count; per item h type 1, d object, d display
	// ID, d count, h type 2, h 0, d 0.
	var g m3Golden
	g.c(0xe9).d(500).d(0).h(1).h(4).d(0x10000101).d(5073).d(7).h(5).h(0).d(0)
	m3Check(t, "SellListProcure", NewSellListProcurePacket(500, []ProcureItem{
		{Type1: 4, ObjectID: 0x10000101, ItemID: 5073, Count: 7, Type2: 5},
	}), g.b)
}
