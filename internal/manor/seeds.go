package manor

import (
	"cmp"
	"slices"

	"github.com/IPeter/l2go/internal/data"
)

// Seed is a seed of a castle's manor (Mobius Seed) with the reference prices
// of its seed and crop items and its limits at the drop rate.
type Seed struct {
	data.Seed
	// SeedPrice and CropPrice are the reference prices of the seed and crop
	// items (1 when the item is unknown, as Mobius sets them).
	SeedPrice, CropPrice int64
	// SeedLimit and CropLimit are LimitSeeds and LimitCrops times the drop
	// rate (getSeedLimit, getCropLimit).
	SeedLimit, CropLimit int64
}

// Reward returns the reward item of a reward type: Reward1 for 1, Reward2
// otherwise (Seed.getReward).
func (s *Seed) Reward(kind int) int {
	if kind == 1 {
		return s.Reward1
	}
	return s.Reward2
}

// SeedMaxPrice and the others are the price bounds the lords must keep
// (getSeedMaxPrice: ten times the reference price, getSeedMinPrice: 60 % of
// it, cut to an int).
func (s *Seed) SeedMaxPrice() int64 { return s.SeedPrice * 10 }
func (s *Seed) SeedMinPrice() int64 { return int64(float64(s.SeedPrice) * 0.6) }
func (s *Seed) CropMaxPrice() int64 { return s.CropPrice * 10 }
func (s *Seed) CropMinPrice() int64 { return int64(float64(s.CropPrice) * 0.6) }

// PriceOf returns the reference price of an item, false when it is unknown.
type PriceOf func(itemID int) (int64, bool)

// Table is the seeds of every manor (Mobius CastleManorManager._seeds, a
// HashMap by seed ID). Read-only after NewTable.
type Table struct {
	seeds  []*Seed // in Mobius's map order
	bySeed map[int]*Seed
	crops  []int // the crop IDs in Mobius's getCropIds order
}

// NewTable builds the table of the seeds of Seeds.xml with the item prices
// and the drop rate.
func NewTable(list []data.Seed, price PriceOf, dropRate int) *Table {
	t := &Table{bySeed: make(map[int]*Seed, len(list))}
	keys := make([]int, 0, len(list))
	for _, d := range list {
		if _, dup := t.bySeed[d.SeedID]; dup {
			continue // the loader refuses it
		}
		s := &Seed{Seed: d, SeedPrice: 1, CropPrice: 1,
			SeedLimit: int64(d.LimitSeeds) * int64(dropRate), CropLimit: int64(d.LimitCrops) * int64(dropRate)}
		if p, ok := price(d.CropID); ok {
			s.CropPrice = p
		}
		if p, ok := price(d.SeedID); ok {
			s.SeedPrice = p
		}
		t.bySeed[d.SeedID] = s
		t.seeds = append(t.seeds, s)
		keys = append(keys, d.SeedID)
	}
	order := javaHashOrder(keys)
	sorted := make([]*Seed, len(order))
	for i, idx := range order {
		sorted[i] = t.seeds[idx]
	}
	t.seeds = sorted
	// getCropIds: a HashSet filled from the map's values.
	var crops []int
	seen := make(map[int]bool)
	for _, s := range t.seeds {
		if !seen[s.CropID] {
			seen[s.CropID] = true
			crops = append(crops, s.CropID)
		}
	}
	for _, idx := range javaHashOrder(crops) {
		t.crops = append(t.crops, crops[idx])
	}
	return t
}

// Len returns the number of seeds.
func (t *Table) Len() int { return len(t.seeds) }

// Seeds returns every seed in Mobius's map order. Don't change the slice.
func (t *Table) Seeds() []*Seed { return t.seeds }

// Seed returns the seed of a seed item, or nil (getSeed).
func (t *Table) Seed(seedID int) *Seed { return t.bySeed[seedID] }

// SeedByCrop returns the first seed of the crop in map order, or nil
// (getSeedByCrop(cropId)). The seeds of one crop share its mature crop,
// rewards and level.
func (t *Table) SeedByCrop(cropID int) *Seed {
	for _, s := range t.seeds {
		if s.CropID == cropID {
			return s
		}
	}
	return nil
}

// SeedByCropIn returns the seed of the crop in a castle's manor, or nil
// (getSeedByCrop(cropId, castleId)).
func (t *Table) SeedByCropIn(cropID, castleID int) *Seed {
	for _, s := range t.seeds {
		if s.CropID == cropID && s.CastleID == castleID {
			return s
		}
	}
	return nil
}

// SeedsFor returns the seeds of a castle's manor in map order
// (getSeedsForCastle; Mobius returns a HashSet of Seed, whose order is the
// JVM's).
func (t *Table) SeedsFor(castleID int) []*Seed {
	var out []*Seed
	for _, s := range t.seeds {
		if s.CastleID == castleID {
			out = append(out, s)
		}
	}
	return out
}

// Crops returns the first seed of each crop, in map order (getCrops).
func (t *Table) Crops() []*Seed {
	var out []*Seed
	seen := make(map[int]bool)
	for _, s := range t.seeds {
		if !seen[s.CropID] {
			seen[s.CropID] = true
			out = append(out, s)
		}
	}
	return out
}

// CropIDs returns the crop IDs in Mobius's getCropIds order. Don't change
// the slice.
func (t *Table) CropIDs() []int { return t.crops }

// IsCrop reports whether the item is a crop.
func (t *Table) IsCrop(itemID int) bool { return slices.Contains(t.crops, itemID) }

// javaHashOrder returns the indexes of keys in the iteration order of a Java
// HashMap<Integer, V> (or HashSet<Integer>) filled with them in order: by
// bucket of the final table (16, doubled while the size passes three
// quarters of it), then in insertion order within a bucket.
func javaHashOrder(keys []int) []int {
	capacity := 16
	for len(keys) > capacity*3/4 {
		capacity *= 2
	}
	mask := uint32(capacity - 1) //nolint:gosec // a small power of two
	bucket := func(k int) uint32 {
		h := uint32(int32(k)) //nolint:gosec // Integer.hashCode is the int's bits
		return (h ^ h>>16) & mask
	}
	order := make([]int, len(keys))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(bucket(keys[a]), bucket(keys[b])) })
	return order
}

// JavaHashOrder is javaHashOrder for the game's maps of Mobius's packets
// (ExShowSellCropList's crops of the inventory).
func JavaHashOrder(keys []int) []int { return javaHashOrder(keys) }
