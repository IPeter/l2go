// Package augment holds the rules of augmentation (Plan 14, M11d): which
// items, life stones and gemstones may be used (Mobius
// AbstractRefinePacket.isValid), the gemstone counts, the removal fees,
// the random augmentation of a life stone (Mobius
// AugmentationData.generateRandomAugmentation with RetailLikeAugmentation)
// and the options an augmentation carries.
//
// The package is pure: no world, packets or I/O. internal/game adapts it to
// the refine packets, the Augment bypass, the worn items and the saver.
package augment

import (
	"slices"

	"github.com/IPeter/l2go/internal/data"
)

// Augmentation is the augmentation of an item, one int as Mobius keeps it
// (Augmentation.getAugmentationId, item_attributes.augAttributes): option 1
// in the low 16 bits, option 2 in the high 16. 0 means none.
type Augmentation int32

// Make returns the augmentation of two options as RequestRefine builds it:
// (stat34 << 16) + stat12.
func Make(stat12, stat34 int) Augmentation {
	return Augmentation(int32(stat34)<<16 + int32(stat12)) //nolint:gosec // option IDs are below 2^16
}

// Stat12 returns the low option, 0x0000FFFF & id.
func (a Augmentation) Stat12() int { return int(a & 0xffff) }

// Stat34 returns the high option, id >> 16 (an arithmetic shift, as Java's).
func (a Augmentation) Stat34() int { return int(a >> 16) }

// Options returns the two option IDs in the order Mobius applies them:
// the low one, then the high one (AugmentationStatBonus).
func (a Augmentation) Options() [2]int { return [2]int{a.Stat12(), a.Stat34()} }

// Config holds the augmentation settings of Player.ini, with the Mobius
// defaults from DefaultConfig.
type Config struct {
	// GradeChances are RetailLikeAugmentation{NoGrade,MidGrade,HighGrade,
	// TopGrade}Chance: per life stone grade, the percent chances of the
	// yellow, blue, purple and red categories (55, 35, 7 and 3 of
	// retailchances.xml).
	GradeChances [4][4]int
	// Blacklist are the items that cannot be augmented
	// (AugmentationBlackList).
	Blacklist []int
	// AllowPvP lets PvP items be augmented and their augmentation removed
	// (AltAllowAugmentPvPItems).
	AllowPvP bool
	// AllowTrade lets augmented items be traded, dropped and sold
	// (AltAllowAugmentTrade).
	AllowTrade bool
	// AllowDestroy lets augmented items be destroyed and crystallized
	// (AltAllowAugmentDestroy).
	AllowDestroy bool
}

// DefaultConfig returns the Mobius defaults: 55/35/7/3 for every grade, the
// grand boss jewels and 8191 on the blacklist, PvP items and trading off,
// destroying on.
func DefaultConfig() Config {
	split := [4]int{55, 35, 7, 3}
	return Config{
		GradeChances: [4][4]int{split, split, split, split},
		Blacklist:    []int{6656, 6657, 6658, 6659, 6660, 6661, 6662, 8191},
		AllowDestroy: true,
	}
}

// Blacklisted reports whether an item template is on the blacklist.
func (c Config) Blacklisted(itemID int) bool { return slices.Contains(c.Blacklist, itemID) }

// gradeChance returns the category chances of a life stone grade, those of
// the no-grade stones for an unknown grade (the switch's default).
func (c Config) gradeChance(grade int) [4]int {
	if grade < data.LifeStoneGradeNone || grade > data.LifeStoneGradeTop {
		grade = data.LifeStoneGradeNone
	}
	return c.GradeChances[grade]
}

// Rand is the part of the game's random source augmentation draws from
// (Mobius Rnd.get(int)); rnd.Source implements it.
type Rand interface {
	Get(n int) int
}

// Categories are the <category probability> values of the second option
// that the grade roll picks between, in the order of the roll: yellow,
// blue, purple, red.
var Categories = [4]int{55, 35, 7, 3}

// Tables are the checked augmentation data with the lookups the rules need.
// They must not be changed and are then safe for concurrent use.
type Tables struct {
	d     *data.Augment
	gems  map[data.CrystalType]data.AugmentGemstone
	fees  map[data.CrystalType][]data.AugmentCancelFee
	npcs  map[int]bool
	pools map[poolKey]*pool
}

// poolKey selects the chances of a weapon kind and a life stone.
type poolKey struct {
	weapon string
	stone  int
}

// pool are the chances of one weapon kind and life stone, in file order:
// stat12 the variation 1 entries, stat34 the others with their category.
type pool struct {
	stat12 []data.AugmentChance
	stat34 []categorized
}

type categorized struct {
	category int
	data.AugmentChance
}

// NewTables indexes the augmentation data.
func NewTables(d *data.Augment) *Tables {
	t := &Tables{
		d:     d,
		gems:  make(map[data.CrystalType]data.AugmentGemstone),
		fees:  make(map[data.CrystalType][]data.AugmentCancelFee),
		npcs:  make(map[int]bool),
		pools: make(map[poolKey]*pool),
	}
	for _, g := range d.Refine.Gemstones {
		t.gems[g.Grade] = g
	}
	for _, f := range d.Refine.CancelFees {
		t.fees[f.Grade] = append(t.fees[f.Grade], f)
	}
	for _, id := range d.Refine.Npcs {
		t.npcs[id] = true
	}
	for _, g := range d.Chances {
		k := poolKey{g.Weapon, g.Stone}
		p := t.pools[k]
		if p == nil {
			p = &pool{}
			t.pools[k] = p
		}
		for _, c := range g.Augments {
			if g.Variation == 1 {
				p.stat12 = append(p.stat12, c)
			} else {
				p.stat34 = append(p.stat34, categorized{g.Category, c})
			}
		}
	}
	return t
}

// Data returns the data the tables index.
func (t *Tables) Data() *data.Augment { return t.d }

// LifeStone returns the life stone of an item template.
func (t *Tables) LifeStone(itemID int) (data.LifeStone, bool) { return t.d.LifeStone(itemID) }

// GemstoneID returns the gemstone an item of the grade needs, 0 for none
// (getGemStoneId).
func (t *Tables) GemstoneID(grade data.CrystalType) int { return t.gems[grade].ItemID }

// GemstoneCount returns how many gemstones an item of the grade needs, 0
// for none (getGemStoneCount). The life stone grade does not count, as in
// Mobius.
func (t *Tables) GemstoneCount(grade data.CrystalType, _ int) int { return t.gems[grade].Count }

// CancelFee returns the adena removing an augmentation costs for an item
// of the grade with crystalCount crystals; false for a grade without fees
// (RequestRefineCancel).
func (t *Tables) CancelFee(grade data.CrystalType, crystalCount int) (int64, bool) {
	for _, f := range t.fees[grade] {
		if f.Below == 0 || crystalCount < f.Below {
			return f.Fee, true
		}
	}
	return 0, false
}

// Blacksmith reports whether an NPC template offers augmentation (decision
// X10).
func (t *Tables) Blacksmith(npcID int) bool { return t.npcs[npcID] }

// Options returns the options of an augmentation in the order Mobius
// applies them, and the option IDs the data does not have (Mobius warns
// "Couldn't find option" and skips them).
func (t *Tables) Options(a Augmentation) (opts []*data.AugmentOption, missing []int) {
	for _, id := range a.Options() {
		if o, ok := t.d.Option(id); ok {
			opts = append(opts, o)
		} else {
			missing = append(missing, id)
		}
	}
	return opts, missing
}

// Generate is AugmentationData.generateRandomAugmentation with
// RetailLikeAugmentation: the life stone's chances for the weapon kind
// (mage for magic weapons, warrior for everything else, accessories too)
// give the low option; a roll of the life stone grade's GradeChances picks
// the category of the high option, whose chances give it. Each pick draws
// r = Rnd.get(10000) and walks the list with s = 10000 as a Java float:
// while s > r, s -= chance × 100 and the entry is taken, so the last entry
// taken wins. A list without entries gives 0.
func (t *Tables) Generate(r Rand, cfg Config, stoneID, stoneGrade int, magic bool) Augmentation {
	weapon := data.AugmentWeaponWarrior
	if magic {
		weapon = data.AugmentWeaponMage
	}
	p := t.pools[poolKey{weapon, stoneID}]
	if p == nil {
		p = &pool{}
	}
	stat12 := pickChance(r, p.stat12)

	gc := cfg.gradeChance(stoneGrade)
	c := r.Get(100)
	switch {
	case c < gc[0]:
		c = Categories[0]
	case c < gc[0]+gc[1]:
		c = Categories[1]
	case c < gc[0]+gc[1]+gc[2]:
		c = Categories[2]
	default:
		c = Categories[3]
	}
	var final []data.AugmentChance
	for _, e := range p.stat34 {
		if e.category == c {
			final = append(final, e.AugmentChance)
		}
	}
	stat34 := pickChance(r, final)
	return Make(stat12, stat34)
}

// pickChance is the selection loop of generateRandomAugmentation, in Java
// float arithmetic.
func pickChance(r Rand, list []data.AugmentChance) int {
	roll := float32(r.Get(10000))
	s := float32(10000)
	id := 0
	for _, e := range list {
		if s > roll {
			// The conversion rounds the product to a float, as Java does,
			// and keeps the compiler from fusing it into the subtraction.
			s -= float32(e.Chance * 100)
			id = e.ID
		}
	}
	return id
}
