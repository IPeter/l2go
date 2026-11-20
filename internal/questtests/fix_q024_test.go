package questtests

import (
	"context"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// fixQ024Aggro runs the aggro-range handlers of npc for p.
func fixQ024Aggro(t *testing.T, h *scripttest.Harness, npc *scripttest.Npc, p *scripttest.Player, isPet bool) {
	t.Helper()
	h.World().Add(npc, p)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.Engine().AggroRangeEnter(ctx, npc.Ref(), p.Ref(), isPet); err != nil {
		t.Fatalf("aggro handler failed: %v", err)
	}
}

// Inhabitants of the Forest of the Dead: Dorian, the tombstone, the silver
// cross the vampire breaks, Lidia's maid, the hairpin, the totem and the
// Mysterious Wizard. The quest needs no pets: only the summon branch of the
// aggro handler did, and it does nothing until summons exist.
func TestFixQ024InhabitantsOfTheForestOfTheDead(t *testing.T) {
	q := b3aStart(t, "Q00024_InhabitantsOfTheForestOfTheDead")
	b3cRequires(t, q.h, q.name)
	w := q.h.World()
	dorian := &scripttest.Npc{ObjectID: 0x60000001, ID: 31389, Name: "Dorian"}
	wizard := &scripttest.Npc{ObjectID: 0x60000002, ID: 31522, Name: "Mysterious Wizard"}
	tombstone := &scripttest.Npc{ObjectID: 0x60000003, ID: 31531, Name: "Tombstone"}
	maid := &scripttest.Npc{ObjectID: 0x60000004, ID: 31532, Name: "Maid of Lidia"}
	vampire := &scripttest.Npc{ObjectID: 0x60000005, ID: 25332, Name: "Vampire"}
	mob := &scripttest.Npc{ObjectID: 0x60000006, ID: 21557}

	stranger := &scripttest.Player{ObjectID: 0x10000001, Name: "Stranger", Level: 70}
	q.talk(stranger, dorian, "31389-02.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 64}
	w.NewQuestState(young.Ref(), "Q00023_LidiasHeart").SetState(scripting.StateCompleted)
	q.talk(young, dorian, "31389-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 65}
	w.NewQuestState(hero.Ref(), "Q00023_LidiasHeart").SetState(scripting.StateCompleted)
	q.talk(hero, dorian, "31389-01.htm")
	q.event(hero, dorian, "31389-03.htm", "31389-03.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{7152: 1})
	q.talk(hero, dorian, "31389-04.htm")

	// The flower on the tombstone.
	w.ClearRecords()
	q.talk(hero, tombstone, "31531-01.htm")
	if got := w.SoundsOf(hero); len(got) != 1 || got[0] != "AmdSound.d_wind_loot_02" {
		t.Errorf("tombstone sounds %v", got)
	}
	q.event(hero, tombstone, "31531-02.htm", "31531-02.htm")
	q.want("flower", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "flower", hero, map[int]int64{7152: 0})
	q.talk(hero, tombstone, "31531-03.htm")

	q.talk(hero, dorian, "31389-05.htm")
	q.event(hero, dorian, "31389-08.htm", "31389-08.htm")
	q.talk(hero, dorian, "31389-09.htm")
	q.event(hero, dorian, "31389-13.htm", "31389-13.htm")
	q.want("silver cross", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "silver cross", hero, map[int]int64{7153: 1})
	q.talk(hero, dorian, "31389-14.htm")

	// The vampire sees the cross: it lets the player go and breaks the cross.
	w.ClearRecords()
	fixQ024Aggro(t, q.h, vampire, hero, false)
	q.want("broken cross", hero, scripting.StateStarted, 4)
	b3aWantItems(t, "broken cross", hero, map[int]int64{7153: 0, 7154: 1})
	fixWantAction(t, "vampire", w, "remove_attacker", vampire.ObjectID, hero.ObjectID, 0)
	if !fixSays(w, vampire.ObjectID, "That sign!") {
		t.Errorf("vampire actions %+v", w.Actions)
	}
	// A summon coming in does nothing until summons exist.
	w.ClearRecords()
	fixQ024Aggro(t, q.h, vampire, hero, true)
	if len(w.Actions) != 0 {
		t.Errorf("summon aggro: actions %+v", w.Actions)
	}

	q.talk(hero, dorian, "31389-15.htm")
	q.event(hero, dorian, "31389-18.htm", "31389-18.htm")
	q.event(hero, dorian, "31389-19.htm", "31389-19.htm")
	q.want("to the maid", hero, scripting.StateStarted, 5)
	b3aWantItems(t, "to the maid", hero, map[int]int64{7154: 0})
	q.talk(hero, dorian, "31389-20.htm")

	// The maid wants the hairpin.
	q.talk(hero, maid, "31532-01.htm")
	q.event(hero, maid, "31532-04.htm", "31532-04.htm")
	q.want("letter", hero, scripting.StateStarted, 6)
	b3aWantItems(t, "letter", hero, map[int]int64{7065: 1})
	q.talk(hero, maid, "31532-05.htm")
	q.event(hero, maid, "31532-06.htm", "31532-07.htm")
	q.want("no hairpin", hero, scripting.StateStarted, 7)
	q.talk(hero, maid, "31532-07a.htm")
	q.talk(hero, dorian, "31389-21.htm")
	q.want("hairpin", hero, scripting.StateStarted, 8)
	b3aWantItems(t, "hairpin", hero, map[int]int64{7148: 1})
	q.talk(hero, dorian, "31389-22.htm")
	q.event(hero, maid, "31532-06.htm", "31532-06.htm")
	b3aWantItems(t, "hairpin given", hero, map[int]int64{7148: 0, 7065: 0})
	q.talk(hero, maid, "31532-08.htm")
	q.event(hero, maid, "31532-10.htm", "31532-10.htm")
	q.talk(hero, maid, "31532-11.htm")
	q.event(hero, maid, "31532-14.htm", "31532-14.htm")
	q.talk(hero, maid, "31532-15.htm")
	q.event(hero, maid, "31532-19.htm", "31532-19.htm")
	q.want("totem hunt", hero, scripting.StateStarted, 9)
	q.talk(hero, maid, "31532-20.htm")

	q.kill(hero, mob, rnd.Get(100, 10))
	q.want("no totem", hero, scripting.StateStarted, 9)
	q.kill(hero, mob, rnd.Get(100, 9))
	q.want("totem", hero, scripting.StateStarted, 10)
	b3aWantItems(t, "totem", hero, map[int]int64{7151: 1})

	// The wizard.
	q.talk(hero, wizard, "31522-01.htm")
	q.event(hero, wizard, "31522-03.htm", "31522-03.htm")
	b3aWantItems(t, "totem given", hero, map[int]int64{7151: 0})
	q.talk(hero, wizard, "31522-04.htm")
	q.event(hero, wizard, "31522-08.htm", "31522-08.htm")
	q.want("wizard", hero, scripting.StateStarted, 11)
	q.talk(hero, wizard, "31522-09.htm")
	q.event(hero, wizard, "31522-17.htm", "31522-17.htm")
	q.talk(hero, wizard, "31522-18.htm")
	q.event(hero, wizard, "31522-21.htm", "31522-21.htm")
	q.want("done", hero, scripting.StateCompleted, 11)
	b3aWantItems(t, "done", hero, map[int]int64{7156: 1})
	q.talk(hero, wizard, "31522-22.htm")
	q.talk(hero, dorian, b3aCompleted)
}
