package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Devil's Legacy: pirate zombies drop treasure chests; Collob opens them
// for 600 adena (and sometimes curses the player with skill 4082 through
// skills.apply_effects), Randolf buys the rest at 1600 adena each.
func TestFixQ365Walkthrough(t *testing.T) {
	q := b3aStart(t, "Q00365_DevilsLegacy")
	w := q.h.World()
	randolf := &scripttest.Npc{ObjectID: 0x60000001, ID: 30095, Name: "Randolf"}
	collob := &scripttest.Npc{ObjectID: 0x60000002, ID: 30092, Name: "Collob"}
	zombie := &scripttest.Npc{ObjectID: 0x60000003, ID: 20836, Name: "Pirate Zombie"}
	captain := &scripttest.Npc{ObjectID: 0x60000004, ID: 21629, Name: "Pirate Zombie Captain"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 38}
	q.talk(young, randolf, "30095-00.htm")
	// Without the quest a kill draws nothing.
	q.kill(young, zombie)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 39}
	q.talk(hero, randolf, "30095-01.htm")
	q.event(hero, randolf, "30095-02.htm", "30095-02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, randolf, "30095-03.htm")
	q.talk(hero, collob, "30092-01.htm")
	q.event(hero, collob, "30092-05.htm", "30092-02.htm")

	// Pirate Zombie: 36%; the captains: 52%.
	w.ClearRecords()
	q.kill(hero, zombie, rnd.Float64(0.35))
	q.kill(hero, zombie, rnd.Float64(0.37))
	q.kill(hero, captain, rnd.Float64(0.5))
	b3aWantItems(t, "chests", hero, map[int]int64{5873: 2})
	b3cSounds(t, "chests", w, hero, b3aSoundItemGet, b3aSoundItemGet)

	// Collob wants 600 adena per chest.
	q.event(hero, collob, "30092-05.htm", "30092-03.htm")
	hero.Give(b3aAdena, 1800)

	// 80%: a plain reward.
	q.h.SetRand(rnd.NewScripted(t, rnd.Get(100, 79), rnd.Get(100, 50)))
	q.event(hero, collob, "30092-05.htm", "30092-05.htm")
	b3aWantItems(t, "plain chest", hero, map[int]int64{5873: 1, b3aAdena: 1200, 1884: 1})

	// 20%: a better reward and the curse.
	w.ClearRecords()
	q.h.SetRand(rnd.NewScripted(t, rnd.Get(100, 80), rnd.Get(1000, 5)))
	q.event(hero, collob, "30092-05.htm", "30092-06.htm")
	b3aWantItems(t, "cursed chest", hero, map[int]int64{5873: 0, b3aAdena: 600, 951: 1})
	fixWantAction(t, "curse", w, "apply_effects", collob.ObjectID, hero.ObjectID, 4082)
	if !slices.Contains(hero.Effects, 4082) {
		t.Errorf("curse: effects %v, want 4082", hero.Effects)
	}

	// A cursed player is not cursed again.
	hero.Give(5873, 1)
	w.ClearRecords()
	q.h.SetRand(rnd.NewScripted(t, rnd.Get(100, 99), rnd.Get(1000, 999)))
	q.event(hero, collob, "30092-05.htm", "30092-06.htm")
	b3aWantItems(t, "second curse", hero, map[int]int64{5873: 0, b3aAdena: 0, 1881: 1})
	if got := fixActions(w, "apply_effects"); len(got) != 0 {
		t.Errorf("second curse: effects applied again: %+v", got)
	}
	q.event(hero, collob, "30092-05.htm", "30092-02.htm")

	// Randolf pays 1600 adena per chest.
	hero.Give(5873, 2)
	q.talk(hero, randolf, "30095-05.htm")
	b3aWantItems(t, "sold", hero, map[int]int64{5873: 0, b3aAdena: 3200})
	q.want("sold", hero, scripting.StateStarted, 1)

	// Quitting deletes the quest and its chests.
	hero.Give(5873, 1)
	q.event(hero, randolf, "30095-06.htm", "30095-06.htm")
	b3cNoQuest(t, "quit", hero, q.name)
	b3aWantItems(t, "quit", hero, map[int]int64{5873: 0})
}
