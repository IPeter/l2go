package stats

import (
	"cmp"
	"fmt"
	"maps"
	"math"
	"slices"
)

// AdenaID is the item ID of adena, which has its own level-gap rule.
const AdenaID = 57

// DropRand is the random source of CalculateDrops. Mobius rolls drops with
// Rnd.nextDouble (Float64) and amounts with Rnd.get(min, max) (Between);
// rnd.Source has both.
type DropRand interface {
	Float64() float64
	Between(lo, hi int) int
}

// DropConfig holds the Mobius drop settings (Rates.ini).
// DefaultDropConfig returns the Mobius defaults. The multipliers are Java
// floats there, so they are used at float32 precision.
type DropConfig struct {
	// DeathChance and DeathAmount are DeathDropChanceMultiplier and
	// DeathDropAmountMultiplier (1): the rates of ordinary drops.
	DeathChance, DeathAmount float64
	// HerbChance and HerbAmount are HerbDropChanceMultiplier and
	// HerbDropAmountMultiplier (1): the rates of herbs (DropItem.Herb).
	HerbChance, HerbAmount float64
	// RaidChance and RaidAmount are RaidDropChanceMultiplier and
	// RaidDropAmountMultiplier (1): the rates of raid drops.
	RaidChance, RaidAmount float64
	// ChanceByItem and AmountByItem are DropChanceMultiplierByItemId
	// ({57: 1}) and DropAmountMultiplierByItemId (57, 6656-6662 and 8191:
	// 1). An item listed here uses its own rate instead of the ones above.
	ChanceByItem, AmountByItem map[int]float64
	// MaxOccurrencesNormal and MaxOccurrencesRaid are
	// DropMaxOccurrencesNormal (2) and DropMaxOccurrencesRaidboss (7): how
	// many drops under 100 % one list may give. 0 or less turns drops off.
	MaxOccurrencesNormal, MaxOccurrencesRaid int
	// The level gap: a killer AdenaMinLevelDifference (8) to
	// AdenaMaxLevelDifference (15) levels above the victim sees the adena
	// chance fall from 100 to AdenaMinLevelGapChance (10) percent; items
	// alike with ItemMinLevelDifference (5), ItemMaxLevelDifference (10)
	// and ItemMinLevelGapChance (10).
	AdenaMinLevelDifference, AdenaMaxLevelDifference int
	AdenaMinLevelGapChance                           float64
	ItemMinLevelDifference, ItemMaxLevelDifference   int
	ItemMinLevelGapChance                            float64
}

// DefaultDropConfig returns the Mobius defaults.
func DefaultDropConfig() DropConfig {
	return DropConfig{
		DeathChance: 1, DeathAmount: 1,
		HerbChance: 1, HerbAmount: 1,
		RaidChance: 1, RaidAmount: 1,
		ChanceByItem: map[int]float64{AdenaID: 1},
		AmountByItem: map[int]float64{
			AdenaID: 1, 6656: 1, 6657: 1, 6658: 1, 6659: 1, 6660: 1, 6661: 1, 6662: 1, 8191: 1,
		},
		MaxOccurrencesNormal:    2,
		MaxOccurrencesRaid:      7,
		AdenaMinLevelDifference: 8,
		AdenaMaxLevelDifference: 15,
		AdenaMinLevelGapChance:  10,
		ItemMinLevelDifference:  5,
		ItemMaxLevelDifference:  10,
		ItemMinLevelGapChance:   10,
	}
}

func (c *DropConfig) validate() error {
	rates := []struct {
		name string
		v    float64
	}{
		{"DeathChance", c.DeathChance}, {"DeathAmount", c.DeathAmount},
		{"HerbChance", c.HerbChance}, {"HerbAmount", c.HerbAmount},
		{"RaidChance", c.RaidChance}, {"RaidAmount", c.RaidAmount},
		{"AdenaMinLevelGapChance", c.AdenaMinLevelGapChance},
		{"ItemMinLevelGapChance", c.ItemMinLevelGapChance},
	}
	for _, m := range []map[int]float64{c.ChanceByItem, c.AmountByItem} {
		for id, v := range m {
			rates = append(rates, struct {
				name string
				v    float64
			}{fmt.Sprintf("rate of item %d", id), v})
		}
	}
	for _, r := range rates {
		if !(r.v >= 0) || math.IsInf(r.v, 0) {
			return fmt.Errorf("stats: %w: drop config %s is %v, want 0 or more", ErrInvalid, r.name, r.v)
		}
	}
	// MathUtil.scaleToRange divides by the width of the level range.
	if c.AdenaMaxLevelDifference <= c.AdenaMinLevelDifference || c.ItemMaxLevelDifference <= c.ItemMinLevelDifference {
		return fmt.Errorf("stats: %w: drop config level differences adena %d-%d, items %d-%d: want the maximum above the minimum",
			ErrInvalid, c.AdenaMinLevelDifference, c.AdenaMaxLevelDifference, c.ItemMinLevelDifference, c.ItemMaxLevelDifference)
	}
	return nil
}

// DropRules are the drop settings of a server. They do not change after
// NewDropRules.
type DropRules struct {
	cfg DropConfig
}

// NewDropRules checks the config and returns the drop rules. The rules keep
// copies of the config's maps.
func NewDropRules(cfg DropConfig) (*DropRules, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg.ChanceByItem = maps.Clone(cfg.ChanceByItem)
	cfg.AmountByItem = maps.Clone(cfg.AmountByItem)
	return &DropRules{cfg: cfg}, nil
}

// Config returns a copy of the config the rules were made with.
func (r *DropRules) Config() DropConfig {
	cfg := r.cfg
	cfg.ChanceByItem = maps.Clone(cfg.ChanceByItem)
	cfg.AmountByItem = maps.Clone(cfg.AmountByItem)
	return cfg
}

// DropItem is one possible drop (Mobius DropHolder): Chance percent of
// Min to Max of the item.
type DropItem struct {
	ItemID   int
	Min, Max int
	Chance   float64
	// Herb is set for items with ex_immediate_effect (herbs), which roll
	// with the herb rates.
	Herb bool
}

// DropGroup is a drop group: at x1 it gives at most one of its items, each
// with Chance percent of the item's share.
type DropGroup struct {
	Chance float64
	Items  []DropItem
}

// DropList is an NPC's death drop list. The game builds it from the NPC
// data, leaving out items the item data does not know (as NpcData does when
// it loads), and calls Sort once.
type DropList struct {
	// Groups keep the data order; each group's items go low to high.
	Groups []DropGroup
	// Items are the ungrouped drops, high to low.
	Items []DropItem
}

// Sort puts the list in the order NpcData gives it: each group's items by
// chance, low to high; the ungrouped items by chance, high to low. The
// sorts are stable, as Java's List.sort.
func (l *DropList) Sort() {
	for i := range l.Groups {
		slices.SortStableFunc(l.Groups[i].Items, func(a, b DropItem) int { return cmp.Compare(a.Chance, b.Chance) })
	}
	slices.SortStableFunc(l.Items, func(a, b DropItem) int { return cmp.Compare(b.Chance, a.Chance) })
}

// DropKill is the kill the drops are rolled for.
type DropKill struct {
	// VictimLevel and KillerLevel decide the level gap.
	VictimLevel, KillerLevel int
	// Raid is set when the victim is a raid (its multipliers and occurrence
	// limit).
	Raid bool
	// Player is the stat set of the player behind the killer (Mobius
	// asPlayer: the player, or a summon's owner), nil when no player
	// killed. Its bonusDropRate, bonusDropAmount and bonusDropAdena stats
	// scale the rates.
	Player *Player
}

// ItemDrop is one rolled drop: an item and its count (Mobius ItemHolder).
type ItemDrop struct {
	ItemID int
	Count  int
}

// dropRoll is the state of one list's rolls: Mobius's calculatedDrops,
// randomDrops, cachedItem and dropOccurrenceCounter. Drops are pointers, as
// Mobius removes ItemHolders by identity.
type dropRoll struct {
	counter int
	drops   []*ItemDrop // calculatedDrops; nil until the first drop
	random  []*ItemDrop // randomDrops; nil until the first drop
	cached  *ItemDrop
}

// swapOut is the max-occurrence rule: once the list gave its allowed drops
// under 100 %, the next item under 100 % first takes the oldest of them out
// (Mobius's "remove highest chance item", which is the first that dropped,
// not the most likely). The item is kept aside and comes back at the end
// unless a later drop replaces it. The counter becomes 1 again.
func (d *dropRoll) swapOut(remove bool) {
	if remove && len(d.random) > 0 {
		d.cached = d.random[0]
		d.random = d.random[1:]
		if i := slices.Index(d.drops, d.cached); i >= 0 {
			d.drops = slices.Delete(d.drops, i, i+1)
		}
	}
	d.counter = 1
}

// add records a drop; random marks one under 100 % that counts against the
// limit, and keep one that may be swapped out.
func (d *dropRoll) add(drop *ItemDrop, random, keep bool) {
	if d.drops == nil {
		d.drops, d.random = []*ItemDrop{}, []*ItemDrop{}
	}
	if random {
		d.counter--
		if keep {
			d.random = append(d.random, drop)
		}
	}
	d.drops = append(d.drops, drop)
}

// finish puts the item aside back when nothing replaced it.
func (d *dropRoll) finish() []*ItemDrop {
	if d.counter > 0 && d.cached != nil && d.drops != nil {
		d.drops = append(d.drops, d.cached)
	}
	return d.drops
}

// CalculateDrops is NpcTemplate.calculateDrops for the death drop list of
// a monster that is not a champion and not seeded: the groups, then the
// ungrouped items. Spoil (M7), champions, premium and the manor are left
// out. Each list allows MaxOccurrencesNormal (raids MaxOccurrencesRaid)
// drops under 100 %, with Mobius's swap-out rule (dropRoll.swapOut).
//
// Groups, in order, each item in its order:
//   - The item's rate: ChanceByItem (adena at most 100), else the herb,
//     raid or death rate; × the player's bonusDropRate multiplier.
//   - At rate 1 the item's chance adds to the group's running total, else
//     it replaces it. Its chance is total × group chance / 100 × rate.
//   - The swap-out check, then the level-gap roll: Float64() × 100 above the
//     adena or item level-gap chance skips the item. It draws even when the
//     chance is 100.
//   - The drop roll: Float64() × 100 below the chance drops (int)
//     (Between(min, max) × amount rate). At rate 1 the group stops at its
//     first drop.
//
// Ungrouped items alike, with their own limit, without the running total:
// the drop roll compares with chance × rate, and the limit counts items
// whose chance alone (× their ChanceByItem rate) is under 100.
//
// The draws per item: the level-gap Float64, then (unless skipped) the drop
// Float64, then for a drop Between(min, max) when min < max. The result
// keeps Mobius's order (an item put aside and given back comes last); nil
// when nothing dropped.
func (r *DropRules) CalculateDrops(rnd DropRand, list *DropList, k DropKill) []ItemDrop {
	var out []ItemDrop
	for _, d := range r.groupDrops(rnd, list.Groups, &k) {
		out = append(out, *d)
	}
	for _, d := range r.ungroupedDrops(rnd, list.Items, &k) {
		out = append(out, *d)
	}
	return out
}

// maxOccurrences is the occurrence limit of a list.
func (r *DropRules) maxOccurrences(k *DropKill) int {
	if k.Raid {
		return r.cfg.MaxOccurrencesRaid
	}
	return r.cfg.MaxOccurrencesNormal
}

// levelGapChances are the level-gap chances of adena and other items
// (MathUtil.scaleToRange of the level difference).
func (r *DropRules) levelGapChances(k *DropKill) (adena, item float64) {
	c := &r.cfg
	diff := float64(k.VictimLevel - k.KillerLevel)
	adena = scaleToRange(diff, float64(-c.AdenaMaxLevelDifference), float64(-c.AdenaMinLevelDifference), c.AdenaMinLevelGapChance, 100)
	item = scaleToRange(diff, float64(-c.ItemMaxLevelDifference), float64(-c.ItemMinLevelDifference), c.ItemMinLevelGapChance, 100)
	return adena, item
}

// scaleToRange is MathUtil.scaleToRange(double...): v clamped to the source
// range, mapped linearly onto the target range.
func scaleToRange(v, srcMin, srcMax, dstMin, dstMax float64) float64 {
	v = math.Max(srcMin, math.Min(srcMax, v))
	return mul(v-srcMin, dstMax-dstMin)/(srcMax-srcMin) + dstMin
}

// levelGapSkips draws the level-gap roll of an item.
func levelGapSkips(rnd DropRand, it *DropItem, adena, item float64) bool {
	gap := item
	if it.ItemID == AdenaID {
		gap = adena
	}
	return mul(rnd.Float64(), 100) > gap
}

// chanceRate is the chance multiplier of an item.
func (r *DropRules) chanceRate(it *DropItem, k *DropKill) float64 {
	c := &r.cfg
	rate := 1.0
	if v, ok := c.ChanceByItem[it.ItemID]; ok {
		rate = mul(rate, f32(v))
		if it.ItemID == AdenaID && rate > 100 {
			rate = 100
		}
	} else if it.Herb {
		rate = mul(rate, f32(c.HerbChance))
	} else if k.Raid {
		rate = mul(rate, f32(c.RaidChance))
	} else {
		rate = mul(rate, f32(c.DeathChance))
	}
	if k.Player != nil {
		rate = mul(rate, 1+k.Player.Calc(BonusDropRate, 0)/100)
	}
	return rate
}

// amount rolls the count of a dropped item: (int) (Between(min, max) ×
// amount rate).
func (r *DropRules) amount(rnd DropRand, it *DropItem, k *DropKill) *ItemDrop {
	c := &r.cfg
	rate := 1.0
	if v, ok := c.AmountByItem[it.ItemID]; ok {
		rate = mul(rate, f32(v))
	} else if it.Herb {
		rate = mul(rate, f32(c.HerbAmount))
	} else if k.Raid {
		rate = mul(rate, f32(c.RaidAmount))
	} else {
		rate = mul(rate, f32(c.DeathAmount))
	}
	if k.Player != nil {
		rate = mul(rate, 1+k.Player.Calc(BonusDropAmount, 0)/100)
		if it.ItemID == AdenaID {
			rate = mul(rate, 1+k.Player.Calc(BonusDropAdena, 0)/100)
		}
	}
	n := rnd.Between(it.Min, it.Max)
	return &ItemDrop{ItemID: it.ItemID, Count: int(javaInt(mul(float64(n), rate)))}
}

// groupDrops is NpcTemplate.calculateGroupDrops.
func (r *DropRules) groupDrops(rnd DropRand, groups []DropGroup, k *DropKill) []*ItemDrop {
	d := dropRoll{counter: r.maxOccurrences(k)}
	if d.counter <= 0 || len(groups) == 0 {
		return nil
	}
	adenaGap, itemGap := r.levelGapChances(k)
	for gi := range groups {
		g := &groups[gi]
		total := 0.0
		for ii := range g.Items {
			it := &g.Items[ii]
			rate := r.chanceRate(it, k)
			if rate == 1 {
				total += it.Chance
			} else {
				total = it.Chance
			}
			chance := mul(mul(total, g.Chance/100), rate)
			if d.counter == 0 && chance < 100 && d.random != nil && d.drops != nil {
				d.swapOut(rate == 1)
			}
			if levelGapSkips(rnd, it, adenaGap, itemGap) {
				continue
			}
			if !(mul(rnd.Float64(), 100) < chance) {
				continue
			}
			drop := r.amount(rnd, it, k)
			random := chance < 100
			if v, ok := r.cfg.ChanceByItem[it.ItemID]; ok {
				random = mul(chance, f32(v)) < 100
			}
			d.add(drop, random, rate == 1)
			if rate == 1 {
				break
			}
		}
	}
	return d.finish()
}

// ungroupedDrops is NpcTemplate.calculateUngroupedDrops for DROP items.
func (r *DropRules) ungroupedDrops(rnd DropRand, items []DropItem, k *DropKill) []*ItemDrop {
	d := dropRoll{counter: r.maxOccurrences(k)}
	if d.counter <= 0 || len(items) == 0 {
		return nil
	}
	adenaGap, itemGap := r.levelGapChances(k)
	for ii := range items {
		it := &items[ii]
		if d.counter == 0 && it.Chance < 100 && d.random != nil && d.drops != nil {
			d.swapOut(true)
		}
		if levelGapSkips(rnd, it, adenaGap, itemGap) {
			continue
		}
		rate := r.chanceRate(it, k)
		if !(mul(rnd.Float64(), 100) < mul(it.Chance, rate)) {
			continue
		}
		drop := r.amount(rnd, it, k)
		random := it.Chance < 100
		if v, ok := r.cfg.ChanceByItem[it.ItemID]; ok {
			random = mul(it.Chance, f32(v)) < 100
		}
		d.add(drop, random, true)
	}
	return d.finish()
}
