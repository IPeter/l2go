package inventory

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// The augmentation is one int per item (Plan 14, M11d): set and removed
// like the enchant level, recorded for InventoryUpdate and the save, and
// loaded and stored with the item.
func TestItemAugmentation(t *testing.T) {
	f := newFixture(t)
	it := f.give(t, idSword, 1)
	if it.Augmented() || it.Augmentation() != 0 {
		t.Fatalf("a new item is augmented: %d", it.Augmentation())
	}
	const aug int32 = 14578<<16 | 91
	it.SetAugmentation(aug)
	if !it.Augmented() || it.Augmentation() != aug || it.Storage().Augmentation != aug {
		t.Errorf("augmentation %d, stored %d; want %d", it.Augmentation(), it.Storage().Augmentation, aug)
	}
	if ups := f.inv.TakeUpdates(); len(ups) != 1 || ups[0].Item != it || ups[0].Change != Modified {
		t.Errorf("updates %+v, want the sword modified", ups)
	}
	if ch := f.inv.Tracker().Take(); len(ch.Upserts) != 1 || ch.Upserts[0].Augmentation != aug {
		t.Errorf("save diff %+v, want the sword with its augmentation", ch)
	}
	it.SetAugmentation(aug) // no change, nothing recorded
	if ups := f.inv.TakeUpdates(); len(ups) != 0 {
		t.Errorf("updates %+v after setting the same augmentation", ups)
	}
	// Augmented items stay out of the clan warehouse (AltAllowAugmentTrade).
	if it.Depositable(false) || !it.Depositable(true) {
		t.Error("augmented: want depositable in the private warehouse only")
	}
	it.SetAugmentation(0)
	if it.Augmented() || !it.Depositable(false) {
		t.Error("removed augmentation: still augmented")
	}

	// Loading keeps it.
	g := newFixture(t)
	g.inv.Load([]storage.Item{{ObjectID: 5, ItemID: idSword, Count: 1, Slot: storage.PaperdollRHand, Mana: storage.NoMana, Augmentation: aug}}, Wearer{})
	if w := g.inv.Paperdoll(storage.PaperdollRHand); w == nil || w.Augmentation() != aug {
		t.Errorf("loaded sword %+v", w)
	}
}
