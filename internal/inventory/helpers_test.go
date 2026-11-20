package inventory

import (
	"fmt"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
)

// The types the game passes in implement the package's interfaces.
var (
	_ Templates = (*data.Data)(nil)
	_ IDSource  = (*storage.IDPool)(nil)
	_ Limits    = (*stats.Player)(nil)
	_ StatSet   = (*stats.Player)(nil)
)

// Template IDs of the test items.
const (
	idAdena     = 57
	idSword     = 1
	idTwoHand   = 2
	idBowD      = 3
	idArrowNG   = 4
	idArrowD    = 5
	idShield    = 6
	idRod       = 7
	idLure      = 8
	idEarring   = 9
	idRing      = 10
	idNecklace  = 11
	idFullArmor = 12
	idChest     = 13
	idLegs      = 14
	idHair      = 15
	idHair2     = 16
	idHairAll   = 17
	idFormal    = 18
	idHead      = 19
	idFeet      = 20
	idGloves    = 21
	idCloak     = 22
	idShirt     = 23
	idHeroSword = 24
	idPotion    = 25
	idQuestBox  = 26
	idShadow    = 27
	idHerb      = 28
	idDagger    = 29
	idEarring2  = 30
	idRing2     = 31
	idBowNG     = 32
)

func weapon(id int, part string, wt data.WeaponType, g data.CrystalType) *data.Item {
	return &data.Item{ID: id, Name: fmt.Sprint("weapon ", id), Type: data.ItemTypeWeapon, BodyPart: part,
		Grade: g, Weight: 1000, Weapon: &data.WeaponInfo{Type: wt}, Depositable: true, Tradable: true,
		Stats: []data.ItemStat{{Stat: mustStat("pAtk"), Op: data.OpSet, Value: 10}}}
}

func armor(id int, part string, at data.ArmorType) *data.Item {
	return &data.Item{ID: id, Name: fmt.Sprint("armor ", id), Type: data.ItemTypeArmor, BodyPart: part,
		Weight: 500, Armor: &data.ArmorInfo{Type: at}, Depositable: true, Tradable: true}
}

func etc(id int, part string, et data.EtcItemType, stackable bool, weight int) *data.Item {
	return &data.Item{ID: id, Name: fmt.Sprint("etc ", id), Type: data.ItemTypeEtc, BodyPart: part,
		Weight: weight, Stackable: stackable, Etc: &data.EtcInfo{Type: et}, Depositable: true, Tradable: true}
}

func mustStat(name string) data.Stat {
	var s data.Stat
	if err := s.UnmarshalText([]byte(name)); err != nil {
		panic(err)
	}
	return s
}

// tmpls is a Templates of the test items.
type tmpls map[int]*data.Item

func (m tmpls) ItemTemplate(id int) (*data.Item, bool) {
	t, ok := m[id]
	return t, ok
}

func testTemplates() tmpls {
	hero := weapon(idHeroSword, "rhand", data.WeaponSword, data.CrystalS)
	hero.Hero = true
	shadow := weapon(idShadow, "rhand", data.WeaponSword, data.CrystalD)
	shadow.Duration = 60
	herb := etc(idHerb, "", data.EtcNone, true, 0)
	herb.ExImmediateEffect = true
	arrowD := etc(idArrowD, "lhand", data.EtcArrow, true, 6)
	arrowD.Grade = data.CrystalD
	defEnchant := armor(idCloak, "back", data.ArmorNone)
	defEnchant.DefaultEnchant = 3
	list := []*data.Item{
		etc(idAdena, "", data.EtcNone, true, 0),
		weapon(idSword, "rhand", data.WeaponSword, data.CrystalNone),
		weapon(idTwoHand, "lrhand", data.WeaponSword, data.CrystalNone),
		weapon(idBowD, "lrhand", data.WeaponBow, data.CrystalD),
		etc(idArrowNG, "lhand", data.EtcArrow, true, 6),
		arrowD,
		armor(idShield, "lhand", data.ArmorShield),
		weapon(idRod, "lrhand", data.WeaponFishingrod, data.CrystalNone),
		etc(idLure, "lhand", data.EtcLure, true, 1),
		armor(idEarring, "rear;lear", data.ArmorNone),
		armor(idEarring2, "rear;lear", data.ArmorNone),
		armor(idRing, "rfinger;lfinger", data.ArmorNone),
		armor(idRing2, "rfinger;lfinger", data.ArmorNone),
		armor(idNecklace, "neck", data.ArmorNone),
		armor(idFullArmor, "fullarmor", data.ArmorHeavy),
		armor(idChest, "chest", data.ArmorHeavy),
		armor(idLegs, "legs", data.ArmorHeavy),
		armor(idHair, "hair", data.ArmorNone),
		armor(idHair2, "hair2", data.ArmorNone),
		armor(idHairAll, "hairall", data.ArmorNone),
		armor(idFormal, "alldress", data.ArmorNone),
		armor(idHead, "head", data.ArmorLight),
		armor(idFeet, "feet", data.ArmorLight),
		armor(idGloves, "gloves", data.ArmorLight),
		defEnchant,
		armor(idShirt, "underwear", data.ArmorNone),
		hero,
		etc(idPotion, "", data.EtcPotion, true, 10),
		etc(idQuestBox, "", data.EtcNone, false, 100),
		shadow,
		herb,
		weapon(idDagger, "rhand", data.WeaponDagger, data.CrystalNone),
		weapon(idBowNG, "lrhand", data.WeaponBow, data.CrystalNone),
	}
	m := make(tmpls, len(list))
	for _, t := range list {
		m[t.ID] = t
	}
	return m
}

// idSeq hands out object IDs from next up; left < 0 means no limit.
type idSeq struct {
	next int32
	left int
}

func (s *idSeq) TryNextN(n int) ([]int32, bool) {
	if n < 1 || (s.left >= 0 && n > s.left) {
		return nil, false
	}
	ids := make([]int32, n)
	for i := range ids {
		ids[i] = s.next
		s.next++
	}
	if s.left >= 0 {
		s.left -= n
	}
	return ids, true
}

// limits is a Limits with fixed values.
type limits struct{ slots, weight int }

func (l limits) InventoryLimit() int { return l.slots }
func (l limits) WeightLimit() int    { return l.weight }

// fixture is an inventory and a warehouse of one character.
type fixture struct {
	tmpls tmpls
	ids   *idSeq
	inv   *Inventory
	wh    *Container
	stats *fakeStats
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{tmpls: testTemplates(), ids: &idSeq{next: 1000, left: -1}, stats: &fakeStats{}}
	opts := Options{Templates: f.tmpls, IDs: f.ids, Tracker: NewTracker(), Config: DefaultConfig()}
	f.inv = NewInventory(limits{slots: 80, weight: 100000}, opts)
	f.inv.AddListener(&StatsListener{Stats: f.stats})
	f.wh = NewWarehouse(func() int { return 100 }, opts)
	return f
}

// give creates count items of the template in the inventory and returns
// the last one, dropping the updates and the save diff.
func (f *fixture) give(t *testing.T, itemID int32, count int64) *Item {
	t.Helper()
	it, err := f.inv.Create(itemID, count)
	if err != nil {
		t.Fatalf("Create(%d, %d): %v", itemID, count, err)
	}
	f.inv.TakeUpdates()
	f.inv.Tracker().Take()
	f.stats.calls = nil
	return it
}

// worn returns the item IDs in the paperdoll, by slot name.
func (f *fixture) worn() map[storage.PaperdollSlot]int32 {
	m := make(map[storage.PaperdollSlot]int32)
	for s := range storage.PaperdollSlot(storage.PaperdollTotalSlots) {
		if it := f.inv.Paperdoll(s); it != nil {
			m[s] = it.ItemID()
		}
	}
	return m
}

// fakeStats records the stats listener's calls.
type fakeStats struct{ calls []string }

func (s *fakeStats) Equip(slot stats.Slot, owner stats.Owner, it stats.Item) error {
	s.calls = append(s.calls, fmt.Sprintf("equip %d %d %s", slot, owner.ID, it.BodyPart))
	return nil
}

func (s *fakeStats) Unequip(slot stats.Slot) {
	s.calls = append(s.calls, fmt.Sprintf("unequip %d", slot))
}

func itemIDs(items []*Item) []int32 {
	out := make([]int32, len(items))
	for i, it := range items {
		out[i] = it.ItemID()
	}
	return out
}

func sortedIDs(items []*Item) []int32 {
	out := itemIDs(items)
	slices.Sort(out)
	return out
}

func objectIDs(items []storage.Item) []int32 {
	out := make([]int32, len(items))
	for i, it := range items {
		out[i] = it.ObjectID
	}
	return out
}
