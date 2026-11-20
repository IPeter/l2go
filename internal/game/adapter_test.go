package game

import (
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
)

// statsOps maps the data's stat functions to the stat engine's.
var statsOps = map[data.StatOp]stats.Op{
	data.OpAdd:   stats.OpAdd,
	data.OpDiv:   stats.OpDiv,
	data.OpMul:   stats.OpMul,
	data.OpSet:   stats.OpSet,
	data.OpShare: stats.OpShare,
	data.OpSub:   stats.OpSub,
}

// TestStatsItemAgreesWithData checks every item of the data: the stat
// engine must apply exactly the stat functions the data lists (enchant
// functions aside, which it computes itself), with the same operation, at
// the same order (0, as Mobius DocumentItem gives every item stat) and
// value.
func TestStatsItemAgreesWithData(t *testing.T) {
	d := testGameData(t)
	for i := range d.Items {
		it := &d.Items[i]
		mods, err := stats.ItemModifiers(stats.ItemOwner(1), statsItem(it, 0))
		if err != nil {
			t.Errorf("item %d (%s): %v", it.ID, it.Name, err)
			continue
		}
		var want []stats.Modifier
		for _, st := range it.Stats {
			if st.Op == data.OpEnchant || st.Op == data.OpEnchantHP {
				continue
			}
			s, ok := stats.ParseStat(st.Stat.String())
			if !ok {
				t.Errorf("item %d (%s): the stat engine does not know %s", it.ID, it.Name, st.Stat)
				continue
			}
			want = append(want, stats.Modifier{Stat: s, Op: statsOps[st.Op], Order: st.Order, Value: st.Value})
		}
		if len(mods) != len(want) {
			t.Errorf("item %d (%s): the stat engine applies %d functions, the data lists %d", it.ID, it.Name, len(mods), len(want))
			continue
		}
		for j, m := range mods {
			w := want[j]
			if m.Stat != w.Stat || m.Op != w.Op || m.Order != w.Order || m.Value != w.Value {
				t.Errorf("item %d (%s) function %d: engine %s %s order %d %v, data %s %s order %d %v",
					it.ID, it.Name, j, m.Stat, m.Op, m.Order, m.Value, w.Stat, w.Op, w.Order, w.Value)
			}
		}
	}
}

func TestStatsClassTemplates(t *testing.T) {
	d := testGameData(t)
	classes, err := statsClassTemplates(d)
	if err != nil {
		t.Fatalf("statsClassTemplates: %v", err)
	}
	if len(classes) != len(d.Classes) {
		t.Errorf("%d stat templates for %d classes", len(classes), len(d.Classes))
	}
	hf := classes[classHumanFighter]
	if hf == nil {
		t.Fatal("no Human Fighter")
	}
	tmpl, _ := d.ClassTemplate(classHumanFighter)
	switch {
	case hf.Base != (stats.BaseStats{STR: 40, CON: 43, DEX: 30, INT: 21, WIT: 11, MEN: 25}):
		t.Errorf("Human Fighter base stats %+v", hf.Base)
	case hf.RunSpeed != 115 || hf.WalkSpeed != 80:
		t.Errorf("Human Fighter speeds %v/%v, want 115/80", hf.RunSpeed, hf.WalkSpeed)
	case hf.PDef != (stats.ArmorDefs{Chest: 31, Legs: 18, Head: 12, Feet: 7, Gloves: 8, Underwear: 3, Cloak: 1}):
		t.Errorf("Human Fighter P.Def slots %+v", hf.PDef)
	case len(hf.Levels) != tmpl.MaxLevel():
		t.Errorf("Human Fighter has %d levels, the data %d", len(hf.Levels), tmpl.MaxLevel())
	case hf.Levels[0].HP != 80 || hf.Levels[0].MP != 30 || hf.Levels[0].CP != 32:
		t.Errorf("Human Fighter level 1 vitals %+v", hf.Levels[0])
	}
}

func TestItemInfo(t *testing.T) {
	d := testGameData(t)
	sword, _ := d.ItemTemplate(itemSquireSword)
	inv := inventory.NewInventory(testLimits{}, inventory.Options{Templates: d})
	inv.Load([]storage.Item{{ObjectID: 0x10000010, ItemID: itemSquireSword, Count: 1, Enchant: 3, Slot: storage.PaperdollRHand, Mana: storage.NoMana}}, inventory.Wearer{})
	got := ItemInfo(inv.ByObjectID(0x10000010))
	if got.Type1 != 0 || got.Type2 != 0 || !got.Equipped || got.Enchant != 3 || got.Mana != -1 ||
		got.ObjectID != 0x10000010 || got.ItemID != int32(sword.DisplayID) || got.BodyPart != int32(sword.BodyPartMask) {
		t.Errorf("ItemInfo of an equipped Squire's Sword +3 = %+v", got)
	}

	// A shadow item shows its mana.
	for i := range d.Items {
		it := &d.Items[i]
		if it.Duration <= 0 {
			continue
		}
		info := ItemInfo(inventory.NewItem(1, it, 1))
		if info.Mana != int32(it.Duration) || info.Equipped {
			t.Errorf("ItemInfo of shadow item %d = %+v, want mana %d", it.ID, info, it.Duration)
		}
		break
	}
}

func TestInventory(t *testing.T) {
	e := newEnv(t, nil)
	c := e.createCharacter("hero", "Hero", classHumanFighter, storage.AccessLevelPlayer)
	// An item the data does not know is kept for saving but not shown.
	c.Items = append(c.Items, storage.Item{ObjectID: 0x7fff0000, ItemID: 999999, Count: 1, Slot: storage.PaperdollNone})
	if err := e.store.Update(e.ctx, &c); err != nil {
		t.Fatalf("Update: %v", err)
	}
	p := e.enter(newSession("hero"), 0)
	e.onLoop(func() {
		inv := p.Inventory()
		if n := len(inv.Items()); n != len(c.Items)-1 {
			t.Errorf("inventory has %d items, want %d", n, len(c.Items)-1)
		}
		if n := len(inv.ItemInfos()); n != len(c.Items)-1 {
			t.Errorf("ItemList shows %d items, want %d", n, len(c.Items)-1)
		}
		if w := inv.Paperdoll(storage.PaperdollRHand); w == nil || w.ItemID() != itemSquireSword {
			t.Errorf("right hand holds %v, want the Squire's Sword", w)
		}
		if inv.Paperdoll(storage.PaperdollNone) != nil || inv.Paperdoll(storage.PaperdollHead) != nil {
			t.Error("empty slots hold items")
		}
		var weight int32
		for _, it := range inv.Items() {
			weight += int32(it.Template().Weight) * int32(it.Count())
		}
		if got := inv.Load(); got != weight || got <= 0 {
			t.Errorf("weight %d, want %d", got, weight)
		}
		if snap := p.Snapshot(); len(snap.Items) != len(c.Items) {
			t.Errorf("the snapshot has %d items, want %d (unknown ones kept)", len(snap.Items), len(c.Items))
		}
	})
}
