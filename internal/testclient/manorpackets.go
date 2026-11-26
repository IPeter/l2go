package testclient

import (
	"context"

	"github.com/IPeter/l2go/internal/packets"
)

// The manor packets (Plan 14 M11k). Every reader follows the writeImpl of the
// Mobius class of the same name, field by field; the encoders follow the
// readImpl of the client packets.

// Game server opcodes and Ex sub-opcodes of the manor (Mobius ServerPackets).
const (
	OpBuyListSeed     byte = 0xe8
	OpSellListProcure byte = 0xe9

	ExOpShowSeedInfo          uint16 = 0x1c
	ExOpShowCropInfo          uint16 = 0x1d
	ExOpShowManorDefaultInfo  uint16 = 0x1e
	ExOpShowSeedSetting       uint16 = 0x1f
	ExOpShowCropSetting       uint16 = 0x20
	ExOpShowSellCropList      uint16 = 0x21
	ExOpShowProcureCropDetail uint16 = 0x22
)

// Client opcodes of the manor (Mobius ClientPackets and ExClientPackets).
const (
	OpRequestBuySeed           byte   = 0xc4
	ExOpRequestProcureCropList uint16 = 0x09
	ExOpRequestSetSeed         uint16 = 0x0a
	ExOpRequestSetCrop         uint16 = 0x0b
)

// The decoder and name entries are added here rather than in the shared
// tables, so the manor packets stay in one file.
func init() {
	serverPackets[OpBuyListSeed] = serverPacket{"BuyListSeed", readBuyListSeed}
	serverPackets[OpSellListProcure] = serverPacket{"SellListProcure", readSellListProcure}
	exServerPackets[ExOpShowSeedInfo] = serverPacket{"ExShowSeedInfo", readExShowSeedInfo}
	exServerPackets[ExOpShowCropInfo] = serverPacket{"ExShowCropInfo", readExShowCropInfo}
	exServerPackets[ExOpShowManorDefaultInfo] = serverPacket{"ExShowManorDefaultInfo", readExShowManorDefaultInfo}
	exServerPackets[ExOpShowSeedSetting] = serverPacket{"ExShowSeedSetting", readExShowSeedSetting}
	exServerPackets[ExOpShowCropSetting] = serverPacket{"ExShowCropSetting", readExShowCropSetting}
	exServerPackets[ExOpShowSellCropList] = serverPacket{"ExShowSellCropList", readExShowSellCropList}
	exServerPackets[ExOpShowProcureCropDetail] = serverPacket{"ExShowProcureCropDetail", readExShowProcureCropDetail}
	clientPacketNames[OpRequestBuySeed] = "RequestBuySeed"
	exClientPacketNames[ExOpRequestProcureCropList] = "RequestProcureCropList"
	exClientPacketNames[ExOpRequestSetSeed] = "RequestSetSeed"
	exClientPacketNames[ExOpRequestSetCrop] = "RequestSetCrop"
}

// ManorRewards are a seed's level and its crop's two rewards.
type ManorRewards struct {
	Level            int32
	Reward1, Reward2 int32
}

// seedRewards reads d level, c 1, d reward 1, c 1, d reward 2.
func (f *fields) seedRewards() ManorRewards {
	var r ManorRewards
	r.Level = f.i32()
	f.manorOne("reward 1 type")
	r.Reward1 = f.i32()
	f.manorOne("reward 2 type")
	r.Reward2 = f.i32()
	return r
}

// manorOne reads a byte that Mobius always writes as 1.
func (f *fields) manorOne(field string) {
	if v := f.u8(); v != 1 && f.ok() {
		f.failf("%s is %d, want 1", field, v)
	}
}

// ManorSeed is a seed of ExShowSeedInfo.
type ManorSeed struct {
	SeedID, Amount, StartAmount, Price int32
	ManorRewards
}

// ExShowSeedInfo is the manor's seed list of a period (FE 1C).
type ExShowSeedInfo struct {
	HideButtons bool
	ManorID     int32
	Seeds       []ManorSeed
}

// PacketName implements Packet.
func (ExShowSeedInfo) PacketName() string { return "ExShowSeedInfo" }

func readExShowSeedInfo(f *fields) Packet {
	var p ExShowSeedInfo
	p.HideButtons = f.flag8("hide buttons")
	p.ManorID = f.i32()
	f.manorZero("unknown")
	n := f.count32("seed count")
	for i := 0; i < n && f.ok(); i++ {
		s := ManorSeed{SeedID: f.i32(), Amount: f.i32(), StartAmount: f.i32(), Price: f.i32()}
		s.ManorRewards = f.seedRewards()
		p.Seeds = append(p.Seeds, s)
	}
	return p
}

// manorZero reads an int that Mobius always writes as 0.
func (f *fields) manorZero(field string) {
	if v := f.i32(); v != 0 && f.ok() {
		f.failf("%s is %d, want 0", field, v)
	}
}

// ManorCrop is a crop of ExShowCropInfo.
type ManorCrop struct {
	CropID, Amount, StartAmount, Price int32
	Reward                             int8
	ManorRewards
}

// ExShowCropInfo is the manor's crop list of a period (FE 1D).
type ExShowCropInfo struct {
	HideButtons bool
	ManorID     int32
	Crops       []ManorCrop
}

// PacketName implements Packet.
func (ExShowCropInfo) PacketName() string { return "ExShowCropInfo" }

func readExShowCropInfo(f *fields) Packet {
	var p ExShowCropInfo
	p.HideButtons = f.flag8("hide buttons")
	p.ManorID = f.i32()
	f.manorZero("unknown")
	n := f.count32("crop count")
	for i := 0; i < n && f.ok(); i++ {
		c := ManorCrop{CropID: f.i32(), Amount: f.i32(), StartAmount: f.i32(), Price: f.i32(), Reward: f.i8()}
		c.ManorRewards = f.seedRewards()
		p.Crops = append(p.Crops, c)
	}
	return p
}

// ManorDefaultCrop is a crop of ExShowManorDefaultInfo.
type ManorDefaultCrop struct {
	CropID, Level, SeedPrice, CropPrice int32
	Reward1, Reward2                    int32
}

// ExShowManorDefaultInfo is the basic seed information (FE 1E).
type ExShowManorDefaultInfo struct {
	HideButtons bool
	Crops       []ManorDefaultCrop
}

// PacketName implements Packet.
func (ExShowManorDefaultInfo) PacketName() string { return "ExShowManorDefaultInfo" }

func readExShowManorDefaultInfo(f *fields) Packet {
	var p ExShowManorDefaultInfo
	p.HideButtons = f.flag8("hide buttons")
	n := f.count32("crop count")
	for i := 0; i < n && f.ok(); i++ {
		c := ManorDefaultCrop{CropID: f.i32(), Level: f.i32(), SeedPrice: f.i32(), CropPrice: f.i32()}
		f.manorOne("reward 1 type")
		c.Reward1 = f.i32()
		f.manorOne("reward 2 type")
		c.Reward2 = f.i32()
		p.Crops = append(p.Crops, c)
	}
	return p
}

// SeedSetting is a seed of ExShowSeedSetting.
type SeedSetting struct {
	SeedID int32
	ManorRewards
	Limit, ReferencePrice, MinPrice, MaxPrice int32
	Start, Price, NextStart, NextPrice        int32
}

// ExShowSeedSetting is the lord's seed settings (FE 1F).
type ExShowSeedSetting struct {
	ManorID int32
	Seeds   []SeedSetting
}

// PacketName implements Packet.
func (ExShowSeedSetting) PacketName() string { return "ExShowSeedSetting" }

func readExShowSeedSetting(f *fields) Packet {
	var p ExShowSeedSetting
	p.ManorID = f.i32()
	n := f.count32("seed count")
	for i := 0; i < n && f.ok(); i++ {
		s := SeedSetting{SeedID: f.i32()}
		s.ManorRewards = f.seedRewards()
		s.Limit, s.ReferencePrice, s.MinPrice, s.MaxPrice = f.i32(), f.i32(), f.i32(), f.i32()
		s.Start, s.Price, s.NextStart, s.NextPrice = f.i32(), f.i32(), f.i32(), f.i32()
		p.Seeds = append(p.Seeds, s)
	}
	return p
}

// CropSetting is a crop of ExShowCropSetting.
type CropSetting struct {
	CropID int32
	ManorRewards
	Limit, MinPrice, MaxPrice int32
	Start, Price              int32
	Reward                    int8
	NextStart, NextPrice      int32
	NextReward                int8
}

// ExShowCropSetting is the lord's crop settings (FE 20).
type ExShowCropSetting struct {
	ManorID int32
	Crops   []CropSetting
}

// PacketName implements Packet.
func (ExShowCropSetting) PacketName() string { return "ExShowCropSetting" }

func readExShowCropSetting(f *fields) Packet {
	var p ExShowCropSetting
	p.ManorID = f.i32()
	n := f.count32("crop count")
	for i := 0; i < n && f.ok(); i++ {
		c := CropSetting{CropID: f.i32()}
		c.ManorRewards = f.seedRewards()
		c.Limit = f.i32()
		f.manorZero("unknown")
		c.MinPrice, c.MaxPrice = f.i32(), f.i32()
		c.Start, c.Price, c.Reward = f.i32(), f.i32(), f.i8()
		c.NextStart, c.NextPrice, c.NextReward = f.i32(), f.i32(), f.i8()
		p.Crops = append(p.Crops, c)
	}
	return p
}

// SellCrop is a crop of the inventory in ExShowSellCropList; ManorID is -1
// when the manor does not buy it.
type SellCrop struct {
	ObjectID, CropID int32
	ManorRewards
	ManorID, Amount, Price int32
	Reward                 int8
	Count                  int32
}

// ExShowSellCropList is the crop sale window (FE 21).
type ExShowSellCropList struct {
	ManorID int32
	Crops   []SellCrop
}

// PacketName implements Packet.
func (ExShowSellCropList) PacketName() string { return "ExShowSellCropList" }

func readExShowSellCropList(f *fields) Packet {
	var p ExShowSellCropList
	p.ManorID = f.i32()
	n := f.count32("crop count")
	for i := 0; i < n && f.ok(); i++ {
		c := SellCrop{ObjectID: f.i32(), CropID: f.i32()}
		c.ManorRewards = f.seedRewards()
		c.ManorID, c.Amount, c.Price, c.Reward = f.i32(), f.i32(), f.i32(), f.i8()
		c.Count = f.i32()
		p.Crops = append(p.Crops, c)
	}
	return p
}

// ProcureCastle is a castle of ExShowProcureCropDetail.
type ProcureCastle struct {
	CastleID, Amount, Price int32
	Reward                  int8
}

// ExShowProcureCropDetail lists the castles that buy a crop (FE 22).
type ExShowProcureCropDetail struct {
	CropID  int32
	Castles []ProcureCastle
}

// PacketName implements Packet.
func (ExShowProcureCropDetail) PacketName() string { return "ExShowProcureCropDetail" }

func readExShowProcureCropDetail(f *fields) Packet {
	var p ExShowProcureCropDetail
	p.CropID = f.i32()
	n := f.count32("castle count")
	for i := 0; i < n && f.ok(); i++ {
		p.Castles = append(p.Castles, ProcureCastle{CastleID: f.i32(), Amount: f.i32(), Price: f.i32(), Reward: f.i8()})
	}
	return p
}

// SeedOnSale is a seed of BuyListSeed.
type SeedOnSale struct {
	SeedID, Amount, Price int32
}

// BuyListSeed is a manor's seed shop (0xE8).
type BuyListSeed struct {
	Money, ManorID int32
	Seeds          []SeedOnSale
}

// PacketName implements Packet.
func (BuyListSeed) PacketName() string { return "BuyListSeed" }

func readBuyListSeed(f *fields) Packet {
	var p BuyListSeed
	p.Money, p.ManorID = f.i32(), f.i32()
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		if t := f.u16(); t != 4 && f.ok() {
			f.failf("type 1 is %d, want 4", t)
		}
		f.manorZero("object ID")
		s := SeedOnSale{SeedID: f.i32(), Amount: f.i32()}
		if t := f.u16(); t != 4 && f.ok() {
			f.failf("type 2 is %d, want 4", t)
		}
		f.u16()
		s.Price = f.i32()
		p.Seeds = append(p.Seeds, s)
	}
	return p
}

// ProcureItem is an item of SellListProcure.
type ProcureItem struct {
	Type1            int16
	ObjectID, ItemID int32
	Count            int32
	Type2            int16
}

// SellListProcure is the procure sale list (0xE9), which Mobius never sends.
type SellListProcure struct {
	Money int32
	Items []ProcureItem
}

// PacketName implements Packet.
func (SellListProcure) PacketName() string { return "SellListProcure" }

func readSellListProcure(f *fields) Packet {
	var p SellListProcure
	p.Money = f.i32()
	f.manorZero("lease")
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		it := ProcureItem{Type1: f.i16(), ObjectID: f.i32(), ItemID: f.i32(), Count: f.i32(), Type2: f.i16()}
		f.u16()
		f.manorZero("price")
		p.Items = append(p.Items, it)
	}
	return p
}

// ManorSeedOrder is a seed to buy with RequestBuySeed.
type ManorSeedOrder struct {
	SeedID, Count int32
}

// CropSale is a crop to sell with RequestProcureCropList.
type CropSale struct {
	ObjectID, ItemID, ManorID, Count int32
}

// SeedPlan is a seed the lord's castle sells next period (RequestSetSeed).
type SeedPlan struct {
	SeedID, Sales, Price int32
}

// CropPlan is a crop the lord's castle buys next period (RequestSetCrop).
type CropPlan struct {
	CropID, Sales, Price int32
	Reward               int8
}

// encodeRequestBuySeed builds RequestBuySeed: d manor, d count, per seed d
// ID, d count.
func encodeRequestBuySeed(manorID int32, seeds []ManorSeedOrder) []byte {
	w := packets.NewWriter(9 + 8*len(seeds))
	w.WriteUint8(OpRequestBuySeed)
	w.WriteInt32(manorID)
	w.WriteInt32(int32(len(seeds))) //nolint:gosec // a test client sends short lists
	for _, s := range seeds {
		w.WriteInt32(s.SeedID)
		w.WriteInt32(s.Count)
	}
	return w.Bytes()
}

// encodeRequestProcureCropList builds RequestProcureCropList: d count, per
// crop d object, d item, d manor, d count.
func encodeRequestProcureCropList(crops []CropSale) []byte {
	w := packets.NewWriter(7 + 16*len(crops))
	w.WriteUint8(OpExClient)
	w.WriteUint16(ExOpRequestProcureCropList)
	w.WriteInt32(int32(len(crops))) //nolint:gosec // a test client sends short lists
	for _, c := range crops {
		w.WriteInt32(c.ObjectID)
		w.WriteInt32(c.ItemID)
		w.WriteInt32(c.ManorID)
		w.WriteInt32(c.Count)
	}
	return w.Bytes()
}

// encodeRequestSetSeed builds RequestSetSeed: d manor, d count, per seed d
// ID, d sales, d price.
func encodeRequestSetSeed(manorID int32, seeds []SeedPlan) []byte {
	w := packets.NewWriter(11 + 12*len(seeds))
	w.WriteUint8(OpExClient)
	w.WriteUint16(ExOpRequestSetSeed)
	w.WriteInt32(manorID)
	w.WriteInt32(int32(len(seeds))) //nolint:gosec // a test client sends short lists
	for _, s := range seeds {
		w.WriteInt32(s.SeedID)
		w.WriteInt32(s.Sales)
		w.WriteInt32(s.Price)
	}
	return w.Bytes()
}

// encodeRequestSetCrop builds RequestSetCrop: d manor, d count, per crop d
// ID, d sales, d price, c reward.
func encodeRequestSetCrop(manorID int32, crops []CropPlan) []byte {
	w := packets.NewWriter(11 + 13*len(crops))
	w.WriteUint8(OpExClient)
	w.WriteUint16(ExOpRequestSetCrop)
	w.WriteInt32(manorID)
	w.WriteInt32(int32(len(crops))) //nolint:gosec // a test client sends short lists
	for _, c := range crops {
		w.WriteInt32(c.CropID)
		w.WriteInt32(c.Sales)
		w.WriteInt32(c.Price)
		w.WriteUint8(uint8(c.Reward)) //nolint:gosec // the Java byte's bits
	}
	return w.Bytes()
}

// RequestBuySeed buys seeds at the manor manager the character talked to
// last; nothing is awaited (Mobius answers with InventoryUpdate, system
// messages or ActionFailed).
func (c *GameConn) RequestBuySeed(ctx context.Context, manorID int32, seeds []ManorSeedOrder) error {
	return c.send(ctx, encodeRequestBuySeed(manorID, seeds))
}

// RequestProcureCropList sells crops to the manors at the manor manager the
// character talked to last; nothing is awaited.
func (c *GameConn) RequestProcureCropList(ctx context.Context, crops []CropSale) error {
	return c.send(ctx, encodeRequestProcureCropList(crops))
}

// RequestSetSeed sets the seeds the lord's castle sells next period; Mobius
// answers nothing on success.
func (c *GameConn) RequestSetSeed(ctx context.Context, manorID int32, seeds []SeedPlan) error {
	return c.send(ctx, encodeRequestSetSeed(manorID, seeds))
}

// RequestSetCrop sets the crops the lord's castle buys next period.
func (c *GameConn) RequestSetCrop(ctx context.Context, manorID int32, crops []CropPlan) error {
	return c.send(ctx, encodeRequestSetCrop(manorID, crops))
}
