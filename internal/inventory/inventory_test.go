package inventory

import (
	"errors"
	"maps"
	"slices"
	"strconv"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
)

// TestEquipRules checks Mobius Inventory.equipItem. Items are named by
// their index in have; worn maps slots to indexes.
func TestEquipRules(t *testing.T) {
	type slot = storage.PaperdollSlot
	const (
		under, lear, rear, neck, lfinger, rfinger = storage.PaperdollUnder, storage.PaperdollLEar, storage.PaperdollREar,
			storage.PaperdollNeck, storage.PaperdollLFinger, storage.PaperdollRFinger
		head, rhand, lhand, gloves, chest, legs, feet, cloak, hair, hair2 = storage.PaperdollHead, storage.PaperdollRHand,
			storage.PaperdollLHand, storage.PaperdollGloves, storage.PaperdollChest, storage.PaperdollLegs,
			storage.PaperdollFeet, storage.PaperdollCloak, storage.PaperdollHair, storage.PaperdollHair2
	)
	tests := []struct {
		name    string
		have    []int32
		before  []int
		w       Wearer
		equip   int
		worn    map[slot]int
		changed []int // indexes, sorted
	}{
		{"one-handed weapon", []int32{idSword}, nil, Wearer{}, 0, map[slot]int{rhand: 0}, []int{0}},
		{"two-handed takes off the shield", []int32{idShield, idTwoHand}, []int{0}, Wearer{}, 1, map[slot]int{rhand: 1}, []int{0, 1}},
		{"two-handed replaces sword and shield", []int32{idSword, idShield, idTwoHand}, []int{0, 1}, Wearer{}, 2, map[slot]int{rhand: 2}, []int{0, 1, 2}},
		{"shield takes off a two-hander", []int32{idTwoHand, idShield}, []int{0}, Wearer{}, 1, map[slot]int{lhand: 1}, []int{0, 1}},
		{"one-hander keeps the shield", []int32{idShield, idSword}, []int{0}, Wearer{}, 1, map[slot]int{rhand: 1, lhand: 0}, []int{1}},
		{"bow puts on arrows of its grade", []int32{idArrowNG, idArrowD, idBowD}, nil, Wearer{}, 2, map[slot]int{rhand: 2, lhand: 1}, []int{1, 2}},
		{"bow without arrows of its grade", []int32{idArrowD, idBowNG}, nil, Wearer{}, 1, map[slot]int{rhand: 1}, []int{1}},
		{"no-grade bow takes no-grade arrows", []int32{idArrowD, idArrowNG, idBowNG}, nil, Wearer{}, 2, map[slot]int{rhand: 2, lhand: 1}, []int{1, 2}},
		{"arrows go with the bow", []int32{idBowD, idArrowNG}, []int{0}, Wearer{}, 1, map[slot]int{rhand: 0, lhand: 1}, []int{1}},
		{"shield takes off bow and arrows", []int32{idBowD, idArrowD, idShield}, []int{0}, Wearer{}, 2, map[slot]int{lhand: 2}, []int{0, 1, 2}},
		{"sword replaces the bow, arrows come off", []int32{idBowD, idArrowD, idSword}, []int{0}, Wearer{}, 2, map[slot]int{rhand: 2}, []int{0, 1, 2}},
		{"lure goes with the rod", []int32{idRod, idLure}, []int{0}, Wearer{}, 1, map[slot]int{rhand: 0, lhand: 1}, []int{1}},
		{"arrows take off the rod", []int32{idRod, idArrowNG}, []int{0}, Wearer{}, 1, map[slot]int{lhand: 1}, []int{0, 1}},
		{"lure takes off the bow", []int32{idBowNG, idLure}, []int{0}, Wearer{}, 1, map[slot]int{lhand: 1}, []int{0, 1}},
		{"first earring goes left", []int32{idEarring}, nil, Wearer{}, 0, map[slot]int{lear: 0}, []int{0}},
		{"second earring goes right", []int32{idEarring, idEarring2}, []int{0}, Wearer{}, 1, map[slot]int{lear: 0, rear: 1}, []int{1}},
		{"third earring replaces the left", []int32{idEarring, idEarring2, idEarring}, []int{0, 1}, Wearer{}, 2, map[slot]int{lear: 2, rear: 1}, []int{0, 2}},
		{"first ring goes left", []int32{idRing}, nil, Wearer{}, 0, map[slot]int{lfinger: 0}, []int{0}},
		{"second ring goes right", []int32{idRing, idRing2}, []int{0}, Wearer{}, 1, map[slot]int{lfinger: 0, rfinger: 1}, []int{1}},
		{"third ring replaces the left", []int32{idRing, idRing2, idRing}, []int{0, 1}, Wearer{}, 2, map[slot]int{lfinger: 2, rfinger: 1}, []int{0, 2}},
		{"necklace", []int32{idNecklace}, nil, Wearer{}, 0, map[slot]int{neck: 0}, []int{0}},
		{"full armor takes off the legs", []int32{idChest, idLegs, idFullArmor}, []int{0, 1}, Wearer{}, 2, map[slot]int{chest: 2}, []int{0, 1, 2}},
		{"legs take off full armor", []int32{idFullArmor, idLegs}, []int{0}, Wearer{}, 1, map[slot]int{legs: 1}, []int{0, 1}},
		{"legs go with a chest", []int32{idChest, idLegs}, []int{0}, Wearer{}, 1, map[slot]int{chest: 0, legs: 1}, []int{1}},
		{"chest keeps the legs", []int32{idLegs, idChest}, []int{0}, Wearer{}, 1, map[slot]int{chest: 1, legs: 0}, []int{1}},
		{"hairall takes off hair and hair2", []int32{idHair, idHair2, idHairAll}, []int{0, 1}, Wearer{}, 2, map[slot]int{hair: 2}, []int{0, 1, 2}},
		{"hair replaces hairall", []int32{idHairAll, idHair}, []int{0}, Wearer{}, 1, map[slot]int{hair: 1}, []int{0, 1}},
		{"hair2 takes off hairall", []int32{idHairAll, idHair2}, []int{0}, Wearer{}, 1, map[slot]int{hair2: 1}, []int{0, 1}},
		{"hair2 keeps hair", []int32{idHair, idHair2}, []int{0}, Wearer{}, 1, map[slot]int{hair: 0, hair2: 1}, []int{1}},
		{"hair keeps hair2", []int32{idHair2, idHair}, []int{0}, Wearer{}, 1, map[slot]int{hair: 1, hair2: 0}, []int{1}},
		{"hair replaces hair", []int32{idHair, idHair2, idHair}, []int{0, 1}, Wearer{}, 2, map[slot]int{hair: 2, hair2: 1}, []int{0, 2}},
		{
			"formal wear takes off weapons and armor",
			[]int32{idSword, idShield, idHead, idFeet, idGloves, idChest, idLegs, idNecklace, idFormal},
			[]int{0, 1, 2, 3, 4, 5, 6, 7}, Wearer{}, 8,
			map[slot]int{chest: 8, neck: 7}, []int{0, 1, 2, 3, 4, 5, 6, 8},
		},
		{"formal wear blocks legs", []int32{idFormal, idLegs}, []int{0}, Wearer{}, 1, map[slot]int{chest: 0}, nil},
		{"formal wear blocks head", []int32{idFormal, idHead}, []int{0}, Wearer{}, 1, map[slot]int{chest: 0}, nil},
		{"formal wear blocks feet", []int32{idFormal, idFeet}, []int{0}, Wearer{}, 1, map[slot]int{chest: 0}, nil},
		{"formal wear blocks gloves", []int32{idFormal, idGloves}, []int{0}, Wearer{}, 1, map[slot]int{chest: 0}, nil},
		{"a weapon takes off formal wear", []int32{idFormal, idSword}, []int{0}, Wearer{}, 1, map[slot]int{rhand: 1}, []int{0, 1}},
		{"a shield takes off formal wear", []int32{idFormal, idShield}, []int{0}, Wearer{}, 1, map[slot]int{lhand: 1}, []int{0, 1}},
		{"formal wear keeps jewels", []int32{idFormal, idRing}, []int{0}, Wearer{}, 1, map[slot]int{chest: 0, lfinger: 1}, []int{1}},
		{"a chest replaces formal wear", []int32{idFormal, idChest}, []int{0}, Wearer{}, 1, map[slot]int{chest: 1}, []int{0, 1}},
		{"cloak", []int32{idCloak}, nil, Wearer{}, 0, map[slot]int{cloak: 0}, []int{0}},
		{"underwear", []int32{idShirt}, nil, Wearer{}, 0, map[slot]int{under: 0}, []int{0}},
		{"hero item needs a hero", []int32{idHeroSword}, nil, Wearer{}, 0, map[slot]int{}, nil},
		{"hero item, hero", []int32{idHeroSword}, nil, Wearer{Hero: true}, 0, map[slot]int{rhand: 0}, []int{0}},
		{"hero item, GM", []int32{idHeroSword}, nil, Wearer{GM: true}, 0, map[slot]int{rhand: 0}, []int{0}},
		{"store mode", []int32{idSword}, nil, Wearer{StoreMode: true}, 0, map[slot]int{}, nil},
		{"no body part", []int32{idPotion}, nil, Wearer{}, 0, map[slot]int{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			items := make([]*Item, len(tt.have))
			for i, id := range tt.have {
				count := int64(1)
				if f.tmpls[int(id)].Stackable {
					count = 100
				}
				items[i] = f.give(t, id, count)
			}
			for _, i := range tt.before {
				f.inv.Equip(items[i], Wearer{})
				if !items[i].Equipped() {
					t.Fatalf("setup: item %d not equipped", i)
				}
			}
			got := f.inv.Equip(items[tt.equip], tt.w)

			index := func(it *Item) int { return slices.Index(items, it) }
			worn := make(map[slot]int)
			for s := range slot(storage.PaperdollTotalSlots) {
				if it := f.inv.Paperdoll(s); it != nil {
					worn[s] = index(it)
				}
			}
			if !maps.Equal(worn, tt.worn) {
				t.Errorf("worn = %v, want %v", worn, tt.worn)
			}
			var changed []int
			for _, it := range got {
				changed = append(changed, index(it))
			}
			slices.Sort(changed)
			if !slices.Equal(changed, tt.changed) {
				t.Errorf("changed = %v, want %v", changed, tt.changed)
			}
			// Every item agrees with the paperdoll.
			for i, it := range items {
				s, isWorn := slot(-1), false
				for ws, wi := range worn {
					if wi == i {
						s, isWorn = ws, true
					}
				}
				if it.Equipped() != isWorn || it.Slot() != s {
					t.Errorf("item %d: equipped %v in %d, want %v in %d", i, it.Equipped(), it.Slot(), isWorn, s)
				}
				wantLoc := storage.LocInventory
				if isWorn {
					wantLoc = storage.LocPaperdoll
				}
				if it.Loc() != wantLoc {
					t.Errorf("item %d: location %s, want %s", i, it.Loc(), wantLoc)
				}
			}
		})
	}
}

func TestUnequip(t *testing.T) {
	t.Run("item by body part", func(t *testing.T) {
		f := newFixture(t)
		e1, e2 := f.give(t, idEarring, 1), f.give(t, idEarring2, 1)
		f.inv.Equip(e1, Wearer{})
		if got := f.inv.Equip(e1, Wearer{}); got != nil || f.inv.Paperdoll(storage.PaperdollREar) != nil {
			t.Fatalf("Equip of a worn earring = %v, worn %v", itemIDs(got), f.worn())
		}
		f.inv.Equip(e2, Wearer{})
		got := f.inv.UnequipItem(e2)
		if !slices.Equal(got, []*Item{e2}) || e2.Equipped() || !e1.Equipped() {
			t.Errorf("UnequipItem(right earring) = %v; e1 worn %v, e2 worn %v", itemIDs(got), e1.Equipped(), e2.Equipped())
		}
		if f.inv.UnequipItem(e2) != nil {
			t.Error("UnequipItem of an item not worn changed something")
		}
	})
	t.Run("full armor from the chest", func(t *testing.T) {
		f := newFixture(t)
		fa := f.give(t, idFullArmor, 1)
		f.inv.Equip(fa, Wearer{})
		if BodyPartOf(fa) != data.BodyPartFullArmor {
			t.Errorf("BodyPartOf(full armor) = %v", BodyPartOf(fa))
		}
		if got := f.inv.UnequipItem(fa); len(got) != 1 || fa.Equipped() {
			t.Errorf("UnequipItem(full armor) = %v, worn %v", itemIDs(got), fa.Equipped())
		}
	})
	t.Run("hairall body part empties hair", func(t *testing.T) {
		f := newFixture(t)
		h, h2 := f.give(t, idHair, 1), f.give(t, idHair2, 1)
		f.inv.Equip(h, Wearer{})
		f.inv.Equip(h2, Wearer{})
		got := f.inv.UnequipBodyPart(data.BodyPartHairall)
		if !slices.Equal(got, []*Item{h}) || !h2.Equipped() {
			t.Errorf("UnequipBodyPart(hairall) = %v, hair2 worn %v", itemIDs(got), h2.Equipped())
		}
	})
	t.Run("pairs have no slot", func(t *testing.T) {
		f := newFixture(t)
		r := f.give(t, idRing, 1)
		f.inv.Equip(r, Wearer{})
		if got := f.inv.UnequipBodyPart(data.BodyPartLRFinger); got != nil || !r.Equipped() {
			t.Errorf("UnequipBodyPart(lr finger) = %v", itemIDs(got))
		}
		if got := f.inv.UnequipBodyPart(data.BodyPartNone); got != nil {
			t.Errorf("UnequipBodyPart(none) = %v", itemIDs(got))
		}
	})
	t.Run("bow takes the arrows along", func(t *testing.T) {
		f := newFixture(t)
		bow, arrows := f.give(t, idBowD, 1), f.give(t, idArrowD, 50)
		f.inv.Equip(bow, Wearer{})
		got := f.inv.UnequipSlot(storage.PaperdollRHand)
		if !slices.Equal(sortedIDs(got), []int32{idBowD, idArrowD}) || arrows.Equipped() {
			t.Errorf("UnequipSlot(rhand) = %v, arrows worn %v", itemIDs(got), arrows.Equipped())
		}
	})
	t.Run("rod takes the lure along", func(t *testing.T) {
		f := newFixture(t)
		rod, lure := f.give(t, idRod, 1), f.give(t, idLure, 5)
		f.inv.Equip(rod, Wearer{})
		f.inv.Equip(lure, Wearer{})
		f.inv.UnequipItem(rod)
		if lure.Equipped() {
			t.Error("lure still worn")
		}
	})
	t.Run("arrows alone keep the bow", func(t *testing.T) {
		f := newFixture(t)
		bow, arrows := f.give(t, idBowD, 1), f.give(t, idArrowD, 50)
		f.inv.Equip(bow, Wearer{})
		f.inv.UnequipSlot(storage.PaperdollLHand)
		if !bow.Equipped() || arrows.Equipped() {
			t.Errorf("bow worn %v, arrows worn %v", bow.Equipped(), arrows.Equipped())
		}
	})
}

func TestPaperdollIndex(t *testing.T) {
	tests := []struct {
		part data.BodyPart
		want storage.PaperdollSlot
	}{
		{data.BodyPartLRHand, storage.PaperdollRHand},
		{data.BodyPartFullArmor, storage.PaperdollChest},
		{data.BodyPartAlldress, storage.PaperdollChest},
		{data.BodyPartHairall, storage.PaperdollHair},
		{data.BodyPartREar, storage.PaperdollREar},
		{data.BodyPartLFinger, storage.PaperdollLFinger},
		{data.BodyPartBack, storage.PaperdollCloak},
		{data.BodyPartLREar, storage.PaperdollNone},
		{data.BodyPartNone, storage.PaperdollNone},
		{data.BodyPartWolf, storage.PaperdollNone},
	}
	for _, tt := range tests {
		if got := PaperdollIndex(tt.part); got != tt.want {
			t.Errorf("PaperdollIndex(%v) = %d, want %d", tt.part, got, tt.want)
		}
	}
}

// TestListeners checks the order of the listener calls and the stats
// listener: the old item first, the bow's arrows inside the bow's equip.
func TestListeners(t *testing.T) {
	f := newFixture(t)
	sword, shield := f.give(t, idSword, 1), f.give(t, idShield, 1)
	bow, arrows := f.give(t, idBowD, 1), f.give(t, idArrowD, 10)
	f.inv.Equip(sword, Wearer{})
	f.inv.Equip(shield, Wearer{})
	f.stats.calls = nil
	f.inv.Equip(bow, Wearer{})
	// Shield off, sword off, then the arrows on: the bow listener runs
	// before the stats listener hears of the bow.
	want := []string{
		"unequip 8",
		"unequip 7",
		"equip 8 " + itoa(arrows.ObjectID()) + " lhand",
		"equip 7 " + itoa(bow.ObjectID()) + " lrhand",
	}
	if !slices.Equal(f.stats.calls, want) {
		t.Errorf("stat calls = %q, want %q", f.stats.calls, want)
	}
	if mask := f.inv.WearedMask(); mask != bow.Template().Mask() {
		t.Errorf("WearedMask = %#x, want %#x (the bow; arrows have none)", mask, bow.Template().Mask())
	}

	var errs int
	l := &StatsListener{Stats: errStats{}, OnError: func(*Item, error) { errs++ }}
	f.inv.AddListener(l)
	f.inv.Equip(sword, Wearer{})
	f.inv.RemoveListener(l)
	if errs != 1 {
		t.Errorf("OnError called %d times, want 1", errs)
	}
}

func itoa(n int32) string { return strconv.Itoa(int(n)) }

// errStats refuses every item.
type errStats struct{}

func (errStats) Equip(stats.Slot, stats.Owner, stats.Item) error {
	return errors.New("bad item")
}

func (errStats) Unequip(stats.Slot) {}

func TestStatsItem(t *testing.T) {
	tm := testTemplates()[idSword]
	tm.Stats = append(tm.Stats, data.ItemStat{Stat: mustStat("pAtk"), Op: data.OpEnchant, Value: 0})
	got := StatsItem(tm, 4)
	if got.Type != "Weapon" || got.BodyPart != "rhand" || got.WeaponType != "SWORD" || got.Enchant != 4 || len(got.Stats) != 1 || got.Stats[0].Stat != "pAtk" {
		t.Errorf("StatsItem = %+v", got)
	}
}
