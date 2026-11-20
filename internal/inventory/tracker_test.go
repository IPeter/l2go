package inventory

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// TestSaveDiff checks decision I2's bookkeeping through the container
// operations.
func TestSaveDiff(t *testing.T) {
	t.Run("leaves and comes back: an upsert", func(t *testing.T) {
		f := newFixture(t)
		sword := f.give(t, idSword, 1)
		dropped, _ := f.inv.Drop(sword.ObjectID(), 1)
		if _, err := f.inv.Add(dropped); err != nil {
			t.Fatal(err)
		}
		ch := f.inv.Tracker().Take()
		if len(ch.Left) != 0 || len(ch.Destroyed) != 0 || !slices.Equal(objectIDs(ch.Upserts), []int32{sword.ObjectID()}) {
			t.Errorf("diff = %+v", ch)
		}
	})
	t.Run("changed, then destroyed: a delete", func(t *testing.T) {
		f := newFixture(t)
		p := f.give(t, idPotion, 10)
		f.inv.Destroy(p, 3)
		f.inv.Destroy(p, 7)
		ch := f.inv.Tracker().Take()
		if len(ch.Upserts) != 0 || !slices.Equal(ch.Destroyed, []int32{p.ObjectID()}) {
			t.Errorf("diff = %+v", ch)
		}
	})
	t.Run("drop, the other picks up", func(t *testing.T) {
		a, b := newFixture(t), newFixture(t)
		sword := a.give(t, idSword, 1)
		dropped, _ := a.inv.Drop(sword.ObjectID(), 1)
		b.inv.Add(dropped)
		if ch := a.inv.Tracker().Take(); !slices.Equal(ch.Left, []int32{sword.ObjectID()}) || len(ch.Upserts) != 0 {
			t.Errorf("dropper diff = %+v", ch)
		}
		if ch := b.inv.Tracker().Take(); !slices.Equal(objectIDs(ch.Upserts), []int32{sword.ObjectID()}) || ch.Upserts[0].Loc != storage.LocInventory {
			t.Errorf("picker diff = %+v", ch)
		}
	})
	t.Run("picked-up adena merges", func(t *testing.T) {
		a, b := newFixture(t), newFixture(t)
		aAdena := a.give(t, idAdena, 100)
		bAdena := b.give(t, idAdena, 5)
		dropped, _ := a.inv.Drop(aAdena.ObjectID(), 100)
		b.inv.Add(dropped)
		if ch := a.inv.Tracker().Take(); !slices.Equal(ch.Left, []int32{aAdena.ObjectID()}) {
			t.Errorf("dropper diff = %+v", ch)
		}
		ch := b.inv.Tracker().Take()
		if !slices.Equal(ch.Destroyed, []int32{aAdena.ObjectID()}) || !slices.Equal(objectIDs(ch.Upserts), []int32{bAdena.ObjectID()}) || ch.Upserts[0].Count != 105 {
			t.Errorf("picker diff = %+v", ch)
		}
	})
	t.Run("deposit and withdraw before a save", func(t *testing.T) {
		f := newFixture(t)
		sword := f.give(t, idSword, 1)
		f.inv.Transfer(sword.ObjectID(), 1, f.wh)
		f.wh.Transfer(sword.ObjectID(), 1, &f.inv.Container)
		ch := f.inv.Tracker().Take()
		if len(ch.Left) != 0 || len(ch.Upserts) != 1 || ch.Upserts[0].Loc != storage.LocInventory {
			t.Errorf("diff = %+v", ch)
		}
	})
	t.Run("take resets and clears dirty", func(t *testing.T) {
		f := newFixture(t)
		it, _ := f.inv.Create(idSword, 1)
		if !it.Dirty() || !f.inv.Tracker().Pending() {
			t.Fatal("a new item is not pending")
		}
		before := f.inv.Tracker().Changes()
		ch := f.inv.Tracker().Take()
		if len(before.Upserts) != 1 || len(ch.Upserts) != 1 || it.Dirty() || f.inv.Tracker().Pending() {
			t.Errorf("Take = %+v; dirty %v, pending %v", ch, it.Dirty(), f.inv.Tracker().Pending())
		}
		if ch := f.inv.Tracker().Take(); len(ch.Upserts)+len(ch.Destroyed)+len(ch.Left) != 0 {
			t.Errorf("second Take = %+v", ch)
		}
	})
	t.Run("upserts in first-change order with the latest state", func(t *testing.T) {
		f := newFixture(t)
		a, _ := f.inv.Create(idSword, 1)
		b, _ := f.inv.Create(idPotion, 2)
		a.SetEnchant(2)
		f.inv.Create(idPotion, 3)
		ch := f.inv.Tracker().Take()
		if !slices.Equal(objectIDs(ch.Upserts), []int32{a.ObjectID(), b.ObjectID()}) || ch.Upserts[0].Enchant != 2 || ch.Upserts[1].Count != 5 {
			t.Errorf("diff = %+v", ch)
		}
	})
}

func TestLoad(t *testing.T) {
	stored := []storage.Item{
		{ObjectID: 1, ItemID: idAdena, Count: 100, Loc: storage.LocInventory, Slot: storage.PaperdollNone, Mana: storage.NoMana},
		{ObjectID: 2, ItemID: idBowD, Count: 1, Loc: storage.LocPaperdoll, Slot: storage.PaperdollRHand, Mana: storage.NoMana},
		{ObjectID: 3, ItemID: idArrowNG, Count: 50, Loc: storage.LocInventory, Slot: storage.PaperdollNone, Mana: storage.NoMana},
		{ObjectID: 4, ItemID: idArrowD, Count: 50, Loc: storage.LocPaperdoll, Slot: storage.PaperdollLHand, Mana: storage.NoMana},
		{ObjectID: 5, ItemID: idEarring, Count: 1, Loc: storage.LocPaperdoll, Slot: storage.PaperdollLEar, Mana: storage.NoMana},
		{ObjectID: 6, ItemID: 999, Count: 1, Loc: storage.LocInventory, Slot: storage.PaperdollNone, Mana: storage.NoMana},
		{ObjectID: 7, ItemID: idSword, Count: 1, Loc: storage.LocWarehouse, Slot: storage.PaperdollNone, Mana: storage.NoMana},
		{ObjectID: 8, ItemID: idChest, Count: 1, Slot: storage.PaperdollChest, Mana: storage.NoMana}, // empty Loc: carried
		{ObjectID: 9, ItemID: idPotion, Count: 3, Loc: "CLANWH", Slot: storage.PaperdollNone, Mana: storage.NoMana},
		{ObjectID: 10, ItemID: idAdena, Count: 7, Loc: storage.LocWarehouse, Slot: storage.PaperdollNone, Mana: storage.NoMana},
	}

	t.Run("clean load", func(t *testing.T) {
		f := newFixture(t)
		skipped := f.inv.Load(stored, Wearer{})
		if !slices.Equal(objectIDs(skipped), []int32{6}) {
			t.Errorf("skipped = %v", objectIDs(skipped))
		}
		var ids []int32
		for _, it := range f.inv.Items() {
			ids = append(ids, it.ObjectID())
		}
		if !slices.Equal(ids, []int32{1, 2, 3, 4, 5, 8}) {
			t.Errorf("inventory = %v", ids)
		}
		want := map[storage.PaperdollSlot]int32{storage.PaperdollRHand: idBowD, storage.PaperdollLHand: idArrowD,
			storage.PaperdollLEar: idEarring, storage.PaperdollChest: idChest}
		if got := f.worn(); !mapsEqual(got, want) {
			t.Errorf("worn = %v, want %v", got, want)
		}
		if f.inv.Tracker().Pending() {
			t.Errorf("a clean load left changes: %+v", f.inv.Tracker().Changes())
		}
		if len(f.inv.TakeUpdates()) != 0 {
			t.Error("Load recorded updates")
		}
		if len(f.stats.calls) == 0 {
			t.Error("the stats listener heard nothing")
		}

		whSkipped := f.wh.Load(stored)
		if len(whSkipped) != 0 || f.wh.Len() != 2 || f.wh.Adena() != 7 || f.inv.Tracker().Pending() {
			t.Errorf("warehouse: skipped %v, %d items, adena %d", objectIDs(whSkipped), f.wh.Len(), f.wh.Adena())
		}
	})
	t.Run("merged stacks and moved items are saved", func(t *testing.T) {
		f := newFixture(t)
		items := []storage.Item{
			{ObjectID: 1, ItemID: idAdena, Count: 100, Loc: storage.LocInventory, Slot: storage.PaperdollNone, Mana: storage.NoMana},
			{ObjectID: 2, ItemID: idAdena, Count: 50, Loc: storage.LocInventory, Slot: storage.PaperdollNone, Mana: storage.NoMana},
			{ObjectID: 3, ItemID: idHeroSword, Count: 1, Loc: storage.LocPaperdoll, Slot: storage.PaperdollRHand, Mana: storage.NoMana},
			{ObjectID: 4, ItemID: idFormal, Count: 1, Loc: storage.LocPaperdoll, Slot: storage.PaperdollChest, Mana: storage.NoMana},
			{ObjectID: 5, ItemID: idLegs, Count: 1, Loc: storage.LocPaperdoll, Slot: storage.PaperdollLegs, Mana: storage.NoMana},
		}
		f.inv.Load(items, Wearer{})
		if f.inv.Adena() != 150 || f.inv.Len() != 4 {
			t.Errorf("adena %d, %d items", f.inv.Adena(), f.inv.Len())
		}
		hero := f.inv.ByObjectID(3)
		legs := f.inv.ByObjectID(5)
		if hero.Equipped() || legs.Equipped() || !f.inv.FormalWear() {
			t.Errorf("hero worn %v, legs worn %v under formal wear", hero.Equipped(), legs.Equipped())
		}
		ch := f.inv.Tracker().Take()
		if !slices.Equal(ch.Destroyed, []int32{2}) || !slices.Equal(sortedStorageIDs(ch.Upserts), []int32{1, 3, 5}) {
			t.Errorf("diff = %+v", ch)
		}
	})
	t.Run("hero keeps the hero item", func(t *testing.T) {
		f := newFixture(t)
		f.inv.Load([]storage.Item{{ObjectID: 3, ItemID: idHeroSword, Count: 1, Loc: storage.LocPaperdoll, Slot: storage.PaperdollRHand, Mana: storage.NoMana}}, Wearer{Hero: true})
		if !f.inv.ByObjectID(3).Equipped() || f.inv.Tracker().Pending() {
			t.Error("hero item not worn by a hero")
		}
	})
	t.Run("a lone earring stored right goes left", func(t *testing.T) {
		f := newFixture(t)
		f.inv.Load([]storage.Item{{ObjectID: 1, ItemID: idEarring, Count: 1, Loc: storage.LocPaperdoll, Slot: storage.PaperdollREar, Mana: storage.NoMana}}, Wearer{})
		if f.inv.ByObjectID(1).Slot() != storage.PaperdollLEar || !f.inv.Tracker().Pending() {
			t.Errorf("earring in %d", f.inv.ByObjectID(1).Slot())
		}
	})
	t.Run("two earrings stored left", func(t *testing.T) {
		f := newFixture(t)
		f.inv.Load([]storage.Item{
			{ObjectID: 1, ItemID: idEarring, Count: 1, Loc: storage.LocPaperdoll, Slot: storage.PaperdollLEar, Mana: storage.NoMana},
			{ObjectID: 2, ItemID: idEarring2, Count: 1, Loc: storage.LocPaperdoll, Slot: storage.PaperdollLEar, Mana: storage.NoMana},
		}, Wearer{})
		if f.inv.ByObjectID(2).Slot() != storage.PaperdollREar {
			t.Errorf("second earring in %d", f.inv.ByObjectID(2).Slot())
		}
		if ch := f.inv.Tracker().Take(); !slices.Equal(objectIDs(ch.Upserts), []int32{2}) {
			t.Errorf("diff = %+v", ch)
		}
	})
}

func mapsEqual(a, b map[storage.PaperdollSlot]int32) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func sortedStorageIDs(items []storage.Item) []int32 {
	ids := objectIDs(items)
	slices.Sort(ids)
	return ids
}
