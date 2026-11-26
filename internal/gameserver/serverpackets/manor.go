package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// The manor packets (Plan 14 M11k, Mobius ExShowSeedInfo, ExShowCropInfo,
// ExShowManorDefaultInfo, ExShowSeedSetting, ExShowCropSetting,
// ExShowSellCropList, ExShowProcureCropDetail, BuyListSeed and
// SellListProcure). ExSendManorList is in exsendmanorlist.go.

// Sub-opcodes of the manor's Ex packets.
const (
	ExOpShowSeedInfo          = 0x1c
	ExOpShowCropInfo          = 0x1d
	ExOpShowManorDefaultInfo  = 0x1e
	ExOpShowSeedSetting       = 0x1f
	ExOpShowCropSetting       = 0x20
	ExOpShowSellCropList      = 0x21
	ExOpShowProcureCropDetail = 0x22
)

// Opcodes of the manor's buy and sell windows.
const (
	OpBuyListSeed     = 0xe8
	OpSellListProcure = 0xe9
)

// SeedRewards are a seed's level and the two rewards of its crop, as the
// manor packets show them after each seed or crop (a 1 byte before each
// reward).
type SeedRewards struct {
	Level            int32
	Reward1, Reward2 int32
}

func putSeedRewards(w *packets.Writer, r SeedRewards) {
	w.WriteInt32(r.Level)
	w.WriteUint8(1)
	w.WriteInt32(r.Reward1)
	w.WriteUint8(1)
	w.WriteInt32(r.Reward2)
}

// SeedInfo is a seed a castle sells in a period (ExShowSeedInfo).
type SeedInfo struct {
	SeedID, Amount, StartAmount, Price int32
	SeedRewards
}

// ExShowSeedInfo is the manor's seed list of a period. HideButtons hides
// the "Seed Purchase" button (the chamberlain's view). Mobius sends no seeds
// for the next period until it is approved: leave Seeds nil then.
type ExShowSeedInfo struct {
	HideButtons bool
	ManorID     int32
	Seeds       []SeedInfo
}

// NewExShowSeedInfoPacket builds ExShowSeedInfo (FE 1C).
func NewExShowSeedInfoPacket(p ExShowSeedInfo) []byte {
	w := newExWriter(ExOpShowSeedInfo, 13+len(p.Seeds)*30)
	putBool8(w, p.HideButtons)
	w.WriteInt32(p.ManorID)
	w.WriteInt32(0)
	putCount32(w, len(p.Seeds))
	for _, s := range p.Seeds {
		w.WriteInt32(s.SeedID)
		w.WriteInt32(s.Amount)
		w.WriteInt32(s.StartAmount)
		w.WriteInt32(s.Price)
		putSeedRewards(w, s.SeedRewards)
	}
	return w.Bytes()
}

// CropInfo is a crop a castle buys in a period (ExShowCropInfo).
type CropInfo struct {
	CropID, Amount, StartAmount, Price int32
	Reward                             int8
	SeedRewards
}

// ExShowCropInfo is the manor's crop list of a period. HideButtons hides the
// "Crop Sales" button. Crops is nil for the next period until it is
// approved.
type ExShowCropInfo struct {
	HideButtons bool
	ManorID     int32
	Crops       []CropInfo
}

// NewExShowCropInfoPacket builds ExShowCropInfo (FE 1D).
func NewExShowCropInfoPacket(p ExShowCropInfo) []byte {
	w := newExWriter(ExOpShowCropInfo, 13+len(p.Crops)*31)
	putBool8(w, p.HideButtons)
	w.WriteInt32(p.ManorID)
	w.WriteInt32(0)
	putCount32(w, len(p.Crops))
	for _, c := range p.Crops {
		w.WriteInt32(c.CropID)
		w.WriteInt32(c.Amount)
		w.WriteInt32(c.StartAmount)
		w.WriteInt32(c.Price)
		putInt8(w, c.Reward)
		putSeedRewards(w, c.SeedRewards)
	}
	return w.Bytes()
}

// ManorDefaultCrop is a crop of the basic seed information: its level, the
// reference prices of its seed and of itself, its rewards.
type ManorDefaultCrop struct {
	CropID, Level        int32
	SeedPrice, CropPrice int32
	Reward1, Reward2     int32
}

// NewExShowManorDefaultInfoPacket builds ExShowManorDefaultInfo (FE 1E):
// hideButtons hides "Seed Purchase" and "Crop Sales".
func NewExShowManorDefaultInfoPacket(hideButtons bool, crops []ManorDefaultCrop) []byte {
	w := newExWriter(ExOpShowManorDefaultInfo, 5+len(crops)*26)
	putBool8(w, hideButtons)
	putCount32(w, len(crops))
	for _, c := range crops {
		w.WriteInt32(c.CropID)
		w.WriteInt32(c.Level)
		w.WriteInt32(c.SeedPrice)
		w.WriteInt32(c.CropPrice)
		w.WriteUint8(1)
		w.WriteInt32(c.Reward1)
		w.WriteUint8(1)
		w.WriteInt32(c.Reward2)
	}
	return w.Bytes()
}

// SeedSetting is a seed of the lord's seed settings: the next period's
// limit, the reference, lowest and highest prices, and what the running and
// the next period sell (start amount and price; 0 when not listed).
type SeedSetting struct {
	SeedID int32
	SeedRewards
	Limit, ReferencePrice, MinPrice, MaxPrice int32
	Start, Price, NextStart, NextPrice        int32
}

// NewExShowSeedSettingPacket builds ExShowSeedSetting (FE 1F).
func NewExShowSeedSettingPacket(manorID int32, seeds []SeedSetting) []byte {
	w := newExWriter(ExOpShowSeedSetting, 8+len(seeds)*54)
	w.WriteInt32(manorID)
	putCount32(w, len(seeds))
	for _, s := range seeds {
		w.WriteInt32(s.SeedID)
		putSeedRewards(w, s.SeedRewards)
		w.WriteInt32(s.Limit)
		w.WriteInt32(s.ReferencePrice)
		w.WriteInt32(s.MinPrice)
		w.WriteInt32(s.MaxPrice)
		w.WriteInt32(s.Start)
		w.WriteInt32(s.Price)
		w.WriteInt32(s.NextStart)
		w.WriteInt32(s.NextPrice)
	}
	return w.Bytes()
}

// CropSetting is a crop of the lord's crop settings: the next period's limit,
// the lowest and highest prices, and what the running and the next period
// buy (start amount, price, reward; 0 when not listed).
type CropSetting struct {
	CropID int32
	SeedRewards
	Limit, MinPrice, MaxPrice int32
	Start, Price              int32
	Reward                    int8
	NextStart, NextPrice      int32
	NextReward                int8
}

// NewExShowCropSettingPacket builds ExShowCropSetting (FE 20).
func NewExShowCropSettingPacket(manorID int32, crops []CropSetting) []byte {
	w := newExWriter(ExOpShowCropSetting, 8+len(crops)*56)
	w.WriteInt32(manorID)
	putCount32(w, len(crops))
	for _, c := range crops {
		w.WriteInt32(c.CropID)
		putSeedRewards(w, c.SeedRewards)
		w.WriteInt32(c.Limit)
		w.WriteInt32(0)
		w.WriteInt32(c.MinPrice)
		w.WriteInt32(c.MaxPrice)
		w.WriteInt32(c.Start)
		w.WriteInt32(c.Price)
		putInt8(w, c.Reward)
		w.WriteInt32(c.NextStart)
		w.WriteInt32(c.NextPrice)
		putInt8(w, c.NextReward)
	}
	return w.Bytes()
}

// SellCrop is a crop of the player's inventory in the crop sale window: the
// manor buys it when Procured (its amount left, price and reward), else the
// manor is -1 and the rest 0.
type SellCrop struct {
	ObjectID, CropID int32
	SeedRewards
	Procured                bool
	ManorID, Amount, Price  int32
	Reward                  int8
	Count                   int32
}

// NewExShowSellCropListPacket builds ExShowSellCropList (FE 21).
func NewExShowSellCropListPacket(manorID int32, crops []SellCrop) []byte {
	w := newExWriter(ExOpShowSellCropList, 8+len(crops)*44)
	w.WriteInt32(manorID)
	putCount32(w, len(crops))
	for _, c := range crops {
		w.WriteInt32(c.ObjectID)
		w.WriteInt32(c.CropID)
		putSeedRewards(w, c.SeedRewards)
		if c.Procured {
			w.WriteInt32(c.ManorID)
			w.WriteInt32(c.Amount)
			w.WriteInt32(c.Price)
			putInt8(w, c.Reward)
		} else {
			w.WriteInt32(-1)
			w.WriteInt32(0)
			w.WriteInt32(0)
			w.WriteUint8(0)
		}
		w.WriteInt32(c.Count)
	}
	return w.Bytes()
}

// ProcureCastle is a castle that buys a crop, in ExShowProcureCropDetail.
type ProcureCastle struct {
	CastleID, Amount, Price int32
	Reward                  int8
}

// NewExShowProcureCropDetailPacket builds ExShowProcureCropDetail (FE 22):
// the castles that buy the crop.
func NewExShowProcureCropDetailPacket(cropID int32, castles []ProcureCastle) []byte {
	w := newExWriter(ExOpShowProcureCropDetail, 8+len(castles)*13)
	w.WriteInt32(cropID)
	putCount32(w, len(castles))
	for _, c := range castles {
		w.WriteInt32(c.CastleID)
		w.WriteInt32(c.Amount)
		w.WriteInt32(c.Price)
		putInt8(w, c.Reward)
	}
	return w.Bytes()
}

// BuyListSeedEntry is a seed of the manor's seed shop: how many are left and
// the price.
type BuyListSeedEntry struct {
	SeedID, Amount, Price int32
}

// NewBuyListSeedPacket builds BuyListSeed (0xE8), the seed shop of a manor:
// the player's adena, the manor, the seeds on sale (Mobius lists those with
// some left and a price).
func NewBuyListSeedPacket(money, manorID int32, seeds []BuyListSeedEntry) []byte {
	w := packets.NewWriter(11 + len(seeds)*20)
	w.WriteUint8(OpBuyListSeed)
	w.WriteInt32(money)
	w.WriteInt32(manorID)
	putCount16(w, len(seeds))
	for _, s := range seeds {
		w.WriteUint16(4) // type 1
		w.WriteInt32(0)  // object ID
		w.WriteInt32(s.SeedID)
		w.WriteInt32(s.Amount)
		w.WriteUint16(4) // type 2
		w.WriteUint16(0)
		w.WriteInt32(s.Price)
	}
	return w.Bytes()
}

// ProcureItem is an item of SellListProcure: a crop of the inventory with the
// amount the castle still buys.
type ProcureItem struct {
	Type1            int16
	ObjectID, ItemID int32
	Count            int32
	Type2            int16
}

// NewSellListProcurePacket builds SellListProcure (0xE9). Mobius never sends
// it (ExShowSellCropList is the crop sale window); it is here for the
// protocol's completeness.
func NewSellListProcurePacket(money int32, items []ProcureItem) []byte {
	w := packets.NewWriter(11 + len(items)*20)
	w.WriteUint8(OpSellListProcure)
	w.WriteInt32(money)
	w.WriteInt32(0) // lease
	putCount16(w, len(items))
	for _, it := range items {
		putInt16(w, it.Type1)
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.ItemID)
		w.WriteInt32(it.Count)
		putInt16(w, it.Type2)
		w.WriteUint16(0)
		w.WriteInt32(0) // price
	}
	return w.Bytes()
}
