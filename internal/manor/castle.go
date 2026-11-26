package manor

import (
	"slices"

	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/rnd"
)

// Production is a seed a castle sells in a period (Mobius SeedProduction):
// Amount left of StartAmount, at Price each.
type Production struct {
	SeedID      int
	Amount      int64
	Price       int64
	StartAmount int64
}

// Procure is a crop a castle buys in a period (Mobius CropProcure): Amount
// left of StartAmount, at Price each, paid with reward Reward (1 or 2).
type Procure struct {
	CropID      int
	Amount      int64
	Price       int64
	StartAmount int64
	Reward      int
}

// Castle is a castle's manor: the seeds it sells and the crops it buys in the
// running period and the next one (Mobius _production, _productionNext,
// _procure and _procureNext).
//
// Mobius quirk, kept: after a maintenance that keeps the next period's lists,
// the running and the next lists hold the same SeedProduction and
// CropProcure objects until the lord sets new ones, so a purchase lowers the
// next period's amount too. ProductionShared and ProcureShared mark that
// state; it is not stored, as Mobius loses it at a restart.
type Castle struct {
	Production, ProductionNext []Production
	Procure, ProcureNext       []Procure

	ProductionShared, ProcureShared bool
}

// Products returns the running (next false) or next period's seeds.
func (c *Castle) Products(next bool) []Production {
	if next {
		return c.ProductionNext
	}
	return c.Production
}

// Procures returns the running (next false) or next period's crops.
func (c *Castle) Procures(next bool) []Procure {
	if next {
		return c.ProcureNext
	}
	return c.Procure
}

// Product returns the seed of a period, or nil (getSeedProduct).
func (c *Castle) Product(seedID int, next bool) *Production {
	list := c.Products(next)
	for i := range list {
		if list[i].SeedID == seedID {
			return &list[i]
		}
	}
	return nil
}

// CropProcure returns the crop of a period, or nil (getCropProcure).
func (c *Castle) CropProcure(cropID int, next bool) *Procure {
	list := c.Procures(next)
	for i := range list {
		if list[i].CropID == cropID {
			return &list[i]
		}
	}
	return nil
}

// DecreaseSeed is SeedProduction.decreaseAmount on the running period's
// seed: false, and nothing changes, when fewer than n are left.
func (c *Castle) DecreaseSeed(seedID int, n int64) bool {
	i := slices.IndexFunc(c.Production, func(p Production) bool { return p.SeedID == seedID })
	if i < 0 || c.Production[i].Amount-n < 0 {
		return false
	}
	c.Production[i].Amount -= n
	if c.ProductionShared && i < len(c.ProductionNext) {
		c.ProductionNext[i].Amount = c.Production[i].Amount
	}
	return true
}

// DecreaseCrop is CropProcure.decreaseAmount on the running period's crop.
func (c *Castle) DecreaseCrop(cropID int, n int64) bool {
	i := slices.IndexFunc(c.Procure, func(p Procure) bool { return p.CropID == cropID })
	if i < 0 || c.Procure[i].Amount-n < 0 {
		return false
	}
	c.Procure[i].Amount -= n
	if c.ProcureShared && i < len(c.ProcureNext) {
		c.ProcureNext[i].Amount = c.Procure[i].Amount
	}
	return true
}

// SetNextProduction is setNextSeedProduction: the next period's seeds.
func (c *Castle) SetNextProduction(list []Production) {
	c.ProductionNext, c.ProductionShared = slices.Clone(list), false
}

// SetNextProcure is setNextCropProcure: the next period's crops.
func (c *Castle) SetNextProcure(list []Procure) {
	c.ProcureNext, c.ProcureShared = slices.Clone(list), false
}

// Reset is resetManorData: every list emptied (the castle lost its owner).
func (c *Castle) Reset() { *c = Castle{} }

// Clone returns a copy that shares no list with c.
func (c *Castle) Clone() Castle {
	out := *c
	out.Production, out.ProductionNext = slices.Clone(c.Production), slices.Clone(c.ProductionNext)
	out.Procure, out.ProcureNext = slices.Clone(c.Procure), slices.Clone(c.ProcureNext)
	return out
}

// Cost is getManorCost: what a period's lists cost the castle, the seeds at
// their reference price (1 for an unknown seed) and the crops at the lord's
// price, by start amount.
func (t *Table) Cost(c *Castle, next bool) int64 {
	var total int64
	for _, p := range c.Products(next) {
		if s := t.Seed(p.SeedID); s != nil {
			total += s.SeedPrice * p.StartAmount
		} else {
			total++
		}
	}
	for _, p := range c.Procures(next) {
		total += p.Price * p.StartAmount
	}
	return total
}

// Harvest is a mature crop the castle's clan warehouse gets at the
// maintenance.
type Harvest struct {
	MatureID int
	Count    int64
}

// Maintenance is the APPROVED → MAINTENANCE change for a castle with an
// owner (changeMode): for each crop of the running period that was bought,
// 90 % of what players sold (at least 1 with a 90 % chance,
// Rnd.get(99) < 90, when that rounds to 0) goes to the warehouse as mature
// crops, and the money not spent goes back to the treasury; then the next
// period becomes the running one, and the next one is emptied when the
// treasury cannot pay the new running period, else starts again at its start
// amounts. It returns the harvest and the new treasury.
func (t *Table) Maintenance(c *Castle, treasury int64, r rnd.Source) ([]Harvest, int64) {
	var crops []Harvest
	for _, p := range c.Procure {
		if p.StartAmount <= 0 {
			continue
		}
		if p.StartAmount != p.Amount {
			n := int64(float64(p.StartAmount-p.Amount) * 0.9)
			if n < 1 && r.Get(99) < 90 {
				n = 1
			}
			if s := t.SeedByCrop(p.CropID); n > 0 && s != nil {
				crops = append(crops, Harvest{MatureID: s.MatureID, Count: n})
			}
		}
		if p.Amount > 0 {
			treasury, _ = residence.AddNoTax(true, treasury, p.Amount*p.Price)
		}
	}
	c.Production, c.Procure = c.ProductionNext, c.ProcureNext
	if treasury < t.Cost(c, false) {
		c.ProductionNext, c.ProcureNext = nil, nil
		c.ProductionShared, c.ProcureShared = false, false
		return crops, treasury
	}
	c.Production, c.Procure = slices.Clone(c.Production), slices.Clone(c.Procure)
	for i := range c.Production {
		c.Production[i].Amount = c.Production[i].StartAmount
	}
	for i := range c.Procure {
		c.Procure[i].Amount = c.Procure[i].StartAmount
	}
	c.ProductionNext, c.ProcureNext = slices.Clone(c.Production), slices.Clone(c.Procure)
	c.ProductionShared, c.ProcureShared = true, true
	return crops, treasury
}

// Approve is the MODIFIABLE → APPROVED change for a castle with an owner
// (changeMode): the next period is paid from the treasury now. When the clan
// warehouse is over its capacity and the treasury cannot pay, the next
// period's lists are emptied instead (cleared true; the lord is told).
//
// Mobius quirk, kept: the slots the new mature crops need are counted with
// getAllItemsByItemId(...) == null, which is never true, so the capacity is
// checked for 0 slots; and a treasury too small for the cost with a
// warehouse within capacity pays nothing (addToTreasuryNoTax refuses) and
// keeps the lists.
func (t *Table) Approve(c *Castle, treasury int64, warehouseFits func(slots int) bool) (int64, bool) {
	cost := t.Cost(c, true)
	if !warehouseFits(0) && treasury < cost {
		c.ProductionNext, c.ProcureNext = nil, nil
		c.ProductionShared, c.ProcureShared = false, false
		return treasury, true
	}
	if v, ok := residence.AddNoTax(true, treasury, -cost); ok {
		treasury = v
	}
	return treasury, false
}

// FilterProduction keeps the seeds a lord may set (RequestSetSeed): known
// seeds with a start amount within the seed limit and a price within the
// bounds.
func (t *Table) FilterProduction(list []Production) []Production {
	out := make([]Production, 0, len(list))
	for _, p := range list {
		s := t.Seed(p.SeedID)
		if s != nil && p.StartAmount <= s.SeedLimit && p.Price >= s.SeedMinPrice() && p.Price <= s.SeedMaxPrice() {
			out = append(out, p)
		}
	}
	return out
}

// FilterProcure keeps the crops a lord of the castle may set
// (RequestSetCrop): crops of the castle's manor with a start amount within
// the crop limit and a price within the bounds.
func (t *Table) FilterProcure(list []Procure, castleID int) []Procure {
	out := make([]Procure, 0, len(list))
	for _, p := range list {
		s := t.SeedByCropIn(p.CropID, castleID)
		if s != nil && p.StartAmount <= s.CropLimit && p.Price >= s.CropMinPrice() && p.Price <= s.CropMaxPrice() {
			out = append(out, p)
		}
	}
	return out
}

// SeedIncome is the seed income of the running period the chamberlain's
// vault page shows: what the seeds sold so far brought.
func (c *Castle) SeedIncome() int64 {
	var total int64
	for _, p := range c.Production {
		if d := p.StartAmount - p.Amount; d != 0 {
			total += d * p.Price
		}
	}
	return total
}
