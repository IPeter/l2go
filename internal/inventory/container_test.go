package inventory

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

func TestCreate(t *testing.T) {
	t.Run("new stack", func(t *testing.T) {
		f := newFixture(t)
		it, err := f.inv.Create(idAdena, 500)
		if err != nil || it.Count() != 500 || it.Loc() != storage.LocInventory || it.LastChange() != Added || it.ObjectID() != 1000 {
			t.Fatalf("Create = %+v, %v", it, err)
		}
		if u := f.inv.TakeUpdates(); len(u) != 1 || u[0] != (Update{it, Added}) {
			t.Errorf("updates = %+v", u)
		}
	})
	t.Run("stackable merges", func(t *testing.T) {
		f := newFixture(t)
		first := f.give(t, idAdena, 500)
		it, err := f.inv.Create(idAdena, 300)
		if err != nil || it != first || it.Count() != 800 || f.inv.Len() != 1 || it.LastChange() != Modified {
			t.Fatalf("Create = %+v, %v; len %d", it, err, f.inv.Len())
		}
		if f.ids.next != 1001 {
			t.Errorf("a merge took an object ID")
		}
	})
	t.Run("one item per count", func(t *testing.T) {
		f := newFixture(t)
		last, err := f.inv.Create(idSword, 3)
		if err != nil || f.inv.Len() != 3 || last.ObjectID() != 1002 || last.Count() != 1 {
			t.Fatalf("Create = %+v, %v; len %d", last, err, f.inv.Len())
		}
		if u := f.inv.TakeUpdates(); len(u) != 3 {
			t.Errorf("%d updates, want 3", len(u))
		}
	})
	t.Run("one item without MultipleItemDrop", func(t *testing.T) {
		f := newFixture(t)
		f.inv.cfg.MultipleItemDrop = false
		if _, err := f.inv.Create(idSword, 3); err != nil || f.inv.Len() != 1 {
			t.Fatalf("Create: %v; len %d", err, f.inv.Len())
		}
	})
	t.Run("default enchant and mana", func(t *testing.T) {
		f := newFixture(t)
		cloak, _ := f.inv.Create(idCloak, 1)
		shadow, _ := f.inv.Create(idShadow, 1)
		sword, _ := f.inv.Create(idSword, 1)
		if cloak.Enchant() != 3 || shadow.Mana() != 60 || !shadow.Shadow() || sword.Mana() != storage.NoMana || sword.Shadow() {
			t.Errorf("cloak enchant %d, shadow mana %d, sword mana %d", cloak.Enchant(), shadow.Mana(), sword.Mana())
		}
	})
	t.Run("all object IDs or none", func(t *testing.T) {
		f := newFixture(t)
		f.ids.left = 2
		if _, err := f.inv.Create(idSword, 3); !errors.Is(err, ErrNoObjectID) || f.inv.Len() != 0 {
			t.Fatalf("Create: %v; len %d", err, f.inv.Len())
		}
		if f.inv.Tracker().Pending() || len(f.inv.TakeUpdates()) != 0 {
			t.Error("a failed Create left changes")
		}
	})
	t.Run("errors", func(t *testing.T) {
		f := newFixture(t)
		if _, err := f.inv.Create(999, 1); !errors.Is(err, ErrUnknownItem) {
			t.Errorf("unknown template: %v", err)
		}
		if _, err := f.inv.Create(idAdena, 0); !errors.Is(err, ErrInvalidCount) {
			t.Errorf("count 0: %v", err)
		}
		f.inv.ids = nil
		if _, err := f.inv.Create(idSword, 1); !errors.Is(err, ErrNoObjectID) {
			t.Errorf("no ID source: %v", err)
		}
	})
	t.Run("adena caps at MaxAdena", func(t *testing.T) {
		f := newFixture(t)
		f.give(t, idAdena, DefaultMaxAdena-10)
		it, _ := f.inv.Create(idAdena, 100)
		if it.Count() != DefaultMaxAdena {
			t.Errorf("adena = %d, want %d", it.Count(), DefaultMaxAdena)
		}
		p, _ := f.inv.Create(idPotion, math.MaxInt32)
		p, _ = f.inv.Create(idPotion, 5)
		if p.Count() != math.MaxInt32 {
			t.Errorf("potions = %d, want the int range", p.Count())
		}
	})
}

func TestAdd(t *testing.T) {
	f := newFixture(t)
	stack := f.give(t, idAdena, 100)
	ground := NewItem(5000, f.tmpls[idAdena], 50)
	if ground.Loc() != LocVoid || ground.Count() != 50 {
		t.Fatalf("NewItem = %+v", ground)
	}
	got, err := f.inv.Add(ground)
	if err != nil || got != stack || stack.Count() != 150 || ground.LastChange() != Removed || ground.Container() != nil {
		t.Fatalf("Add(adena) = %+v, %v", got, err)
	}
	ch := f.inv.Tracker().Take()
	if !slices.Equal(ch.Destroyed, []int32{5000}) || !slices.Equal(objectIDs(ch.Upserts), []int32{stack.ObjectID()}) {
		t.Errorf("diff = %+v", ch)
	}

	sword := NewItem(5001, f.tmpls[idSword], 7)
	if sword.Count() != 1 {
		t.Errorf("a non-stackable NewItem has count %d", sword.Count())
	}
	got, err = f.inv.Add(sword)
	if err != nil || got != sword || sword.Loc() != storage.LocInventory || sword.LastChange() != Added || sword.Container() != &f.inv.Container {
		t.Fatalf("Add(sword) = %+v, %v", got, err)
	}
	if _, err := f.wh.Add(sword); !errors.Is(err, ErrInContainer) {
		t.Errorf("Add of an item in a container: %v", err)
	}
}

func TestTransfer(t *testing.T) {
	t.Run("item that does not stack moves whole", func(t *testing.T) {
		f := newFixture(t)
		sword := f.give(t, idSword, 1)
		got, err := f.inv.Transfer(sword.ObjectID(), 5, f.wh)
		if err != nil || got != sword || sword.Loc() != storage.LocWarehouse || f.inv.Len() != 0 || f.wh.Len() != 1 {
			t.Fatalf("Transfer = %+v, %v", got, err)
		}
		if u := f.inv.TakeUpdates(); len(u) != 1 || u[0].Change != Removed {
			t.Errorf("inventory updates = %+v", u)
		}
		// Inventory and warehouse share the tracker: one upsert.
		ch := f.inv.Tracker().Take()
		if len(ch.Left) != 0 || len(ch.Destroyed) != 0 || len(ch.Upserts) != 1 || ch.Upserts[0].Loc != storage.LocWarehouse {
			t.Errorf("diff = %+v", ch)
		}
	})
	t.Run("part of a stack splits with a new ID", func(t *testing.T) {
		f := newFixture(t)
		adena := f.give(t, idAdena, 1000)
		got, err := f.inv.Transfer(adena.ObjectID(), 300, f.wh)
		if err != nil || got == adena || got.Count() != 300 || adena.Count() != 700 || got.ObjectID() != 1001 || got.Loc() != storage.LocWarehouse {
			t.Fatalf("Transfer = %+v, %v; left %d", got, err, adena.Count())
		}
	})
	t.Run("stack onto a stack", func(t *testing.T) {
		f := newFixture(t)
		adena := f.give(t, idAdena, 1000)
		whStack, _ := f.wh.Create(idAdena, 5)
		f.inv.Tracker().Take()
		got, err := f.inv.Transfer(adena.ObjectID(), 5000, f.wh)
		if err != nil || got != whStack || whStack.Count() != 1005 || f.inv.Len() != 0 || adena.LastChange() != Removed {
			t.Fatalf("Transfer = %+v, %v", got, err)
		}
		ch := f.inv.Tracker().Take()
		if !slices.Equal(ch.Destroyed, []int32{adena.ObjectID()}) || !slices.Equal(objectIDs(ch.Upserts), []int32{whStack.ObjectID()}) {
			t.Errorf("diff = %+v", ch)
		}
	})
	t.Run("whole stack gets a new ID", func(t *testing.T) {
		f := newFixture(t)
		adena := f.give(t, idAdena, 1000)
		got, err := f.inv.Transfer(adena.ObjectID(), 1000, f.wh)
		if err != nil || got.ObjectID() == adena.ObjectID() || got.Count() != 1000 || f.inv.Len() != 0 {
			t.Fatalf("Transfer = %+v, %v", got, err)
		}
	})
	t.Run("to another character", func(t *testing.T) {
		f, g := newFixture(t), newFixture(t)
		sword := f.give(t, idSword, 1)
		if _, err := f.inv.Transfer(sword.ObjectID(), 1, &g.inv.Container); err != nil {
			t.Fatal(err)
		}
		if ch := f.inv.Tracker().Take(); !slices.Equal(ch.Left, []int32{sword.ObjectID()}) || len(ch.Upserts) != 0 {
			t.Errorf("giver diff = %+v", ch)
		}
		if ch := g.inv.Tracker().Take(); !slices.Equal(objectIDs(ch.Upserts), []int32{sword.ObjectID()}) {
			t.Errorf("receiver diff = %+v", ch)
		}
	})
	t.Run("no object ID changes nothing", func(t *testing.T) {
		f := newFixture(t)
		adena := f.give(t, idAdena, 1000)
		f.ids.left = 0
		if _, err := f.inv.Transfer(adena.ObjectID(), 10, f.wh); !errors.Is(err, ErrNoObjectID) || adena.Count() != 1000 || f.wh.Len() != 0 {
			t.Errorf("Transfer: %v; count %d", err, adena.Count())
		}
	})
	t.Run("errors", func(t *testing.T) {
		f := newFixture(t)
		adena := f.give(t, idAdena, 10)
		if _, err := f.inv.Transfer(42, 1, f.wh); !errors.Is(err, ErrNotFound) {
			t.Errorf("unknown object: %v", err)
		}
		if _, err := f.inv.Transfer(adena.ObjectID(), 0, f.wh); !errors.Is(err, ErrInvalidCount) {
			t.Errorf("count 0: %v", err)
		}
		if _, err := f.inv.Transfer(adena.ObjectID(), 1, nil); !errors.Is(err, ErrNotFound) {
			t.Errorf("no target: %v", err)
		}
	})
}

func TestDestroy(t *testing.T) {
	t.Run("part of a stack", func(t *testing.T) {
		f := newFixture(t)
		p := f.give(t, idPotion, 10)
		got, err := f.inv.DestroyByObjectID(p.ObjectID(), 4)
		if err != nil || got != p || p.Count() != 6 || p.LastChange() != Modified {
			t.Fatalf("Destroy = %+v, %v", got, err)
		}
	})
	t.Run("all of it", func(t *testing.T) {
		f := newFixture(t)
		p := f.give(t, idPotion, 10)
		if _, err := f.inv.Destroy(p, 10); err != nil || f.inv.Len() != 0 || p.LastChange() != Removed {
			t.Fatalf("Destroy: %v", err)
		}
		ch := f.inv.Tracker().Take()
		if !slices.Equal(ch.Destroyed, []int32{p.ObjectID()}) || len(ch.Upserts) != 0 {
			t.Errorf("diff = %+v", ch)
		}
		if u := f.inv.TakeUpdates(); len(u) != 1 || u[0].Change != Removed {
			t.Errorf("updates = %+v", u)
		}
	})
	t.Run("more than it has", func(t *testing.T) {
		f := newFixture(t)
		p := f.give(t, idPotion, 10)
		if _, err := f.inv.Destroy(p, 11); !errors.Is(err, ErrInvalidCount) || p.Count() != 10 {
			t.Errorf("Destroy: %v", err)
		}
		if _, err := f.inv.DestroyByObjectID(42, 1); !errors.Is(err, ErrNotFound) {
			t.Errorf("unknown object: %v", err)
		}
	})
	t.Run("worn item comes off", func(t *testing.T) {
		f := newFixture(t)
		sword := f.give(t, idSword, 1)
		f.inv.Equip(sword, Wearer{})
		f.stats.calls = nil
		if _, err := f.inv.Destroy(sword, 1); err != nil {
			t.Fatal(err)
		}
		if f.inv.Paperdoll(storage.PaperdollRHand) != nil || !slices.Equal(f.stats.calls, []string{"unequip 7"}) {
			t.Errorf("paperdoll %v, stat calls %q", f.worn(), f.stats.calls)
		}
	})
	t.Run("by item ID, unworn first", func(t *testing.T) {
		f := newFixture(t)
		worn := f.give(t, idSword, 1)
		f.inv.Equip(worn, Wearer{})
		spare := f.give(t, idSword, 1)
		got, err := f.inv.DestroyByItemID(idSword, 1)
		if err != nil || got != spare || !worn.Equipped() || f.inv.Len() != 1 {
			t.Fatalf("DestroyByItemID = %+v, %v", got, err)
		}
		// Only the worn one is left: Mobius takes it.
		if got, err := f.inv.DestroyByItemID(idSword, 1); err != nil || got != worn || f.inv.Len() != 0 {
			t.Fatalf("DestroyByItemID(worn) = %+v, %v", got, err)
		}
	})
	t.Run("by item ID, several that do not stack", func(t *testing.T) {
		f := newFixture(t)
		f.give(t, idQuestBox, 3)
		worn := f.give(t, idSword, 1)
		f.inv.Equip(worn, Wearer{})
		if _, err := f.inv.DestroyByItemID(idQuestBox, 2); err != nil || f.inv.Count(idQuestBox, -1, true) != 1 {
			t.Fatalf("DestroyByItemID: %v; left %d", err, f.inv.Count(idQuestBox, -1, true))
		}
		if _, err := f.inv.DestroyByItemID(idQuestBox, 2); !errors.Is(err, ErrInvalidCount) || f.inv.Count(idQuestBox, -1, true) != 1 {
			t.Errorf("too many: %v", err)
		}
		f.give(t, idSword, 1)
		// The worn one is not counted: one unworn sword is too few for 2.
		if _, err := f.inv.DestroyByItemID(idSword, 2); !errors.Is(err, ErrInvalidCount) {
			t.Errorf("worn swords counted: %v", err)
		}
		if _, err := f.inv.DestroyByItemID(999, 1); !errors.Is(err, ErrNotFound) {
			t.Errorf("unknown: %v", err)
		}
	})
	t.Run("warehouse by item ID takes the first", func(t *testing.T) {
		f := newFixture(t)
		last, _ := f.wh.Create(idSword, 2)
		got, err := f.wh.DestroyByItemID(idSword, 1)
		if err != nil || got == last || f.wh.Len() != 1 || f.wh.Items()[0] != last {
			t.Errorf("DestroyByItemID = %+v, %v", got, err)
		}
	})
}

func TestDrop(t *testing.T) {
	t.Run("part of a stack", func(t *testing.T) {
		f := newFixture(t)
		adena := f.give(t, idAdena, 1000)
		got, err := f.inv.Drop(adena.ObjectID(), 100)
		if err != nil || got == adena || got.Count() != 100 || got.Loc() != LocVoid || got.Container() != nil || adena.Count() != 900 {
			t.Fatalf("Drop = %+v, %v", got, err)
		}
		ch := f.inv.Tracker().Take()
		if !slices.Equal(objectIDs(ch.Upserts), []int32{adena.ObjectID()}) || len(ch.Left) != 0 {
			t.Errorf("diff = %+v", ch)
		}
	})
	t.Run("whole worn item", func(t *testing.T) {
		f := newFixture(t)
		sword := f.give(t, idSword, 1)
		f.inv.Equip(sword, Wearer{})
		got, err := f.inv.Drop(sword.ObjectID(), 1)
		if err != nil || got != sword || sword.Loc() != LocVoid || sword.Equipped() || f.inv.Paperdoll(storage.PaperdollRHand) != nil {
			t.Fatalf("Drop = %+v, %v", got, err)
		}
		ch := f.inv.Tracker().Take()
		if !slices.Equal(ch.Left, []int32{sword.ObjectID()}) || len(ch.Upserts) != 0 {
			t.Errorf("diff = %+v", ch)
		}
		if u := f.inv.TakeUpdates(); len(u) != 1 || u[0].Change != Removed {
			t.Errorf("updates = %+v", u)
		}
	})
	t.Run("more than the stack drops it whole", func(t *testing.T) {
		f := newFixture(t)
		adena := f.give(t, idAdena, 10)
		if got, err := f.inv.Drop(adena.ObjectID(), 50); err != nil || got != adena {
			t.Errorf("Drop = %+v, %v", got, err)
		}
	})
	t.Run("errors", func(t *testing.T) {
		f := newFixture(t)
		adena := f.give(t, idAdena, 10)
		if _, err := f.inv.Drop(42, 1); !errors.Is(err, ErrNotFound) {
			t.Errorf("unknown: %v", err)
		}
		if _, err := f.inv.Drop(adena.ObjectID(), -1); !errors.Is(err, ErrInvalidCount) {
			t.Errorf("negative: %v", err)
		}
		f.ids.left = 0
		if _, err := f.inv.Drop(adena.ObjectID(), 5); !errors.Is(err, ErrNoObjectID) || adena.Count() != 10 {
			t.Errorf("no ID: %v", err)
		}
	})
}

func TestCountAndAdena(t *testing.T) {
	f := newFixture(t)
	if f.inv.Adena() != 0 || f.inv.AdenaItem() != nil {
		t.Error("adena in an empty inventory")
	}
	f.give(t, idAdena, 77)
	s1 := f.give(t, idSword, 1)
	s2 := f.give(t, idSword, 1)
	s2.SetEnchant(5)
	f.inv.Equip(s1, Wearer{})
	tests := []struct {
		item     int32
		enchant  int32
		equipped bool
		want     int64
	}{
		{idAdena, -1, true, 77},
		{idSword, -1, true, 2},
		{idSword, -1, false, 1},
		{idSword, 5, true, 1},
		{idSword, 0, false, 0},
		{idPotion, -1, true, 0},
	}
	for _, tt := range tests {
		if got := f.inv.Count(tt.item, tt.enchant, tt.equipped); got != tt.want {
			t.Errorf("Count(%d, %d, %v) = %d, want %d", tt.item, tt.enchant, tt.equipped, got, tt.want)
		}
	}
	if f.inv.Adena() != 77 || len(f.inv.AllByItemID(idSword)) != 2 {
		t.Errorf("Adena %d, swords %d", f.inv.Adena(), len(f.inv.AllByItemID(idSword)))
	}
}

func TestLimits(t *testing.T) {
	f := newFixture(t)
	f.inv.owner = limits{slots: 3, weight: 2000}
	f.give(t, idSword, 1)   // 1000
	f.give(t, idPotion, 50) // 500
	f.give(t, idAdena, 1000)
	if f.inv.Weight() != 1500 {
		t.Errorf("Weight = %d, want 1500", f.inv.Weight())
	}
	capacity := []struct {
		limit    int
		slots    int64
		autoLoot bool
		want     bool
	}{
		{3, 0, false, true}, {2, 0, false, true}, {2, 0, true, false}, {3, 0, true, true},
		{3, 1, false, false}, {4, 1, false, true},
	}
	for _, tt := range capacity {
		f.inv.owner = limits{slots: tt.limit, weight: 2000}
		f.inv.cfg.AutoLootSlotLimit = tt.autoLoot
		if got := f.inv.ValidateCapacity(tt.slots); got != tt.want {
			t.Errorf("3 items, limit %d: ValidateCapacity(%d), auto loot %v = %v, want %v", tt.limit, tt.slots, tt.autoLoot, got, tt.want)
		}
	}
	f.inv.owner = limits{slots: 3, weight: 2000}
	f.inv.cfg.AutoLootSlotLimit = false
	if !f.inv.ValidateWeight(500) || f.inv.ValidateWeight(501) {
		t.Error("ValidateWeight around the limit")
	}
	// Full inventory: stacking onto a held stack fits, a new one does not.
	if !f.inv.ValidateCapacityByItemID(idPotion) || f.inv.ValidateCapacityByItemID(idQuestBox) {
		t.Error("ValidateCapacityByItemID")
	}
	if !f.inv.ValidateCapacityFor(NewItem(1, f.tmpls[idAdena], 5)) || f.inv.ValidateCapacityFor(NewItem(2, f.tmpls[idArrowD], 5)) {
		t.Error("ValidateCapacityFor stackables")
	}
	if !f.inv.ValidateCapacityFor(NewItem(3, f.tmpls[idHerb], 1)) {
		t.Error("ValidateCapacityFor: an item that takes effect at once needs no slot")
	}

	wh := NewWarehouse(func() int { return 2 }, Options{Templates: f.tmpls, IDs: f.ids})
	wh.Create(idSword, 1)
	if !wh.ValidateCapacity(1) || wh.ValidateCapacity(2) {
		t.Error("warehouse ValidateCapacity")
	}

	need := f.inv.SlotsNeeded([]Want{{idSword, 2}, {idAdena, 100}, {idArrowD, 5}, {999, 3}})
	if need != 2+0+1+3 {
		t.Errorf("SlotsNeeded = %d, want 6", need)
	}
}

func TestWeightCap(t *testing.T) {
	f := newFixture(t)
	f.tmpls[idPotion].Weight = 1_000_000
	defer func() { f.tmpls[idPotion].Weight = 10 }()
	f.give(t, idPotion, 1_000_000)
	if f.inv.Weight() != math.MaxInt32 {
		t.Errorf("Weight = %d, want the int range", f.inv.Weight())
	}
}

func TestUpdates(t *testing.T) {
	f := newFixture(t)
	a := f.give(t, idAdena, 10)
	b, _ := f.inv.Create(idSword, 1) // added
	f.inv.Equip(b, Wearer{})         // added then modified: stays added
	f.inv.Create(idAdena, 5)         // a modified
	f.inv.Destroy(a, 15)             // a removed
	u := f.inv.TakeUpdates()
	want := []Update{{b, Added}, {a, Removed}}
	if !slices.Equal(u, want) {
		t.Errorf("updates = %+v, want %+v", u, want)
	}
	if len(f.inv.TakeUpdates()) != 0 {
		t.Error("TakeUpdates did not reset")
	}
}

func TestItem(t *testing.T) {
	f := newFixture(t)
	shadow := f.give(t, idShadow, 1)
	arrows := f.give(t, idArrowD, 5)
	formal := f.give(t, idFormal, 1)
	if !shadow.Equipable() || arrows.Equipable() || !formal.Equipable() || f.give(t, idPotion, 1).Equipable() {
		t.Error("Equipable")
	}
	if !shadow.Depositable(true) || shadow.Depositable(false) || !arrows.Depositable(false) {
		t.Error("Depositable")
	}
	f.inv.Equip(formal, Wearer{})
	if formal.Depositable(true) {
		t.Error("a worn item is depositable")
	}
	shadow.SetMana(59)
	shadow.SetEnchant(-3)
	if shadow.Mana() != 59 || shadow.Enchant() != 0 || !shadow.Dirty() {
		t.Errorf("mana %d, enchant %d, dirty %v", shadow.Mana(), shadow.Enchant(), shadow.Dirty())
	}
	si := formal.Storage()
	want := storage.Item{ObjectID: formal.ObjectID(), ItemID: idFormal, Count: 1, Slot: storage.PaperdollChest, Loc: storage.LocPaperdoll, Mana: storage.NoMana}
	if si != want {
		t.Errorf("Storage = %+v, want %+v", si, want)
	}
}
