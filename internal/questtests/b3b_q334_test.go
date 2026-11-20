package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ334 = "Q00334_TheWishingPotion"

// The Wishing Potion: the secret book, the eight ingredients, the potion and
// two wishes (royalty, then Rupina, who vanishes after the talk).
func TestB3bQ334Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ334)
	w := h.World()
	matild := &scripttest.Npc{ObjectID: 0x60000001, ID: 30738}
	tree := &scripttest.Npc{ObjectID: 0x60000002, ID: 27139}
	basilisk := &scripttest.Npc{ObjectID: 0x60000003, ID: 20199}

	// Level 29 and below: refused (exit_quest does nothing to a CREATED
	// state, as in Mobius).
	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 29}
	reply, ok := h.Talk(b3bQ334, young, matild)
	b3bPage(t, "level 29", reply, ok, "30738-21.htm")
	b3bQuest(t, "level 29", young, b3bQ334, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 30}
	reply, ok = h.Talk(b3bQ334, hero, matild)
	b3bPage(t, "offer", reply, ok, "30738-01.htm")
	h.Event(b3bQ334, "30738-03.htm", hero, matild)
	b3bQuest(t, "accept", hero, b3bQ334, scripting.StateStarted, 1)
	b3bItems(t, "accept", hero, map[int]int64{3678: 1})
	reply, ok = h.Talk(b3bQ334, hero, matild)
	b3bPage(t, "cond 1", reply, ok, "30738-04.htm")

	// The Secret Keeper Tree always drops the book (rnd(100) < 100), then the
	// summon drop roll.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 99), rnd.Get(100, 50)))
	h.Kill(hero, tree)
	b3bQuest(t, "book", hero, b3bQ334, scripting.StateStarted, 2)
	b3bItems(t, "book", hero, map[int]int64{3679: 1})
	reply, ok = h.Talk(b3bQ334, hero, matild)
	b3bPage(t, "cond 2", reply, ok, "30738-05.htm")
	reply, ok = h.Event(b3bQ334, "30738-06.htm", hero, matild)
	b3bPage(t, "recipes", reply, ok, "30738-06.htm")
	b3bQuest(t, "recipes", hero, b3bQ334, scripting.StateStarted, 3)
	b3bItems(t, "recipes", hero, map[int]int64{3678: 0, 3679: 0, 3680: 1, 3681: 1})

	// Seven ingredients in hand; the basilisk's scale (rnd(100) < 15) is the eighth.
	for id := 3685; id <= 3691; id++ {
		hero.Give(id, 1)
	}
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 15), rnd.Get(100, 0), rnd.Get(100, 14), rnd.Get(100, 0)))
	h.Kill(hero, basilisk)
	b3bQuest(t, "no scale", hero, b3bQ334, scripting.StateStarted, 3)
	h.Kill(hero, basilisk)
	b3bQuest(t, "all ingredients", hero, b3bQ334, scripting.StateStarted, 4)
	reply, ok = h.Talk(b3bQ334, hero, matild)
	b3bPage(t, "cond 4", reply, ok, "30738-08.htm")

	reply, ok = h.Event(b3bQ334, "30738-10.htm", hero, matild)
	b3bPage(t, "potion", reply, ok, "30738-10.htm")
	b3bItems(t, "potion", hero, map[int]int64{3680: 0, 3681: 0, 3684: 0, 3691: 0, 3682: 1, 3467: 1})
	reply, ok = h.Talk(b3bQ334, hero, matild)
	b3bPage(t, "with potion", reply, ok, "30738-11.htm")
	reply, ok = h.Event(b3bQ334, "30738-14.htm", hero, matild)
	b3bPage(t, "wishes", reply, ok, "30738-15.htm")

	// Wish for royalty: 1..33 gives the certificate.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 32)))
	h.Event(b3bQ334, "30738-18.htm", hero, matild)
	b3bItems(t, "royalty", hero, map[int]int64{3467: 0, 3469: 1})
	reply, ok = h.Event(b3bQ334, "30738-18.htm", hero, matild)
	b3bPage(t, "no potion", reply, ok, "30738-14.htm")

	// With the orb and no potion, Matild starts the next round at cond 3.
	reply, ok = h.Talk(b3bQ334, hero, matild)
	b3bPage(t, "again", reply, ok, "30738-12.htm")
	b3bQuest(t, "again", hero, b3bQ334, scripting.StateStarted, 3)
	b3bItems(t, "again", hero, map[int]int64{3680: 1, 3681: 1})

	// Wish for love: Rupina appears (rnd(100) >= 50), gives a necklace and goes.
	hero.Give(3467, 1)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 50)))
	h.Event(b3bQ334, "30738-16.htm", hero, matild)
	if len(w.Spawned) != 1 || w.Spawned[0].ID != 30742 {
		t.Fatalf("spawned %+v, want Rupina", w.Spawned)
	}
	rupina := w.Spawned[0]
	// Mobius decays the player's target: the NPC clicked, here Rupina.
	hero.Target = rupina
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 4)))
	reply, ok = h.Talk(b3bQ334, hero, rupina)
	b3bPage(t, "Rupina", reply, ok, "30742-01.htm")
	b3bItems(t, "Rupina", hero, map[int]int64{931: 1})
	if !rupina.Deleted {
		t.Error("Rupina stays after the talk")
	}

	// The Wisdom Chest gives its spellbooks (rnd(100) < 20) and goes. Without
	// a target nothing is decayed (Mobius would fail on the null target).
	chest := &scripttest.Npc{ObjectID: 0x60000020, ID: 30743, Name: "Wisdom Chest"}
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 19)))
	reply, ok = h.Talk(b3bQ334, hero, chest)
	b3bPage(t, "chest without target", reply, ok, "30743-06.htm")
	if chest.Deleted {
		t.Error("the chest went without being the target")
	}
	hero.Target = chest
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 19)))
	reply, ok = h.Talk(b3bQ334, hero, chest)
	b3bPage(t, "chest", reply, ok, "30743-06.htm")
	b3bItems(t, "chest", hero, map[int]int64{1049: 2, 1050: 2})
	if !chest.Deleted {
		t.Error("the chest stays after the talk")
	}
}
