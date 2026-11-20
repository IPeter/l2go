package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const (
	fixQ235StarOfDestiny = 5011
	fixQ235PureSilver    = 6320
	fixQ235TrueGold      = 6321
	fixQ235SageStone     = 6322
	fixQ235BloodFire     = 6318
	fixQ235MimirElixir   = 6319
	fixQ235MixingStone   = 5905
	fixQ235EnchantA      = 729
)

// Mimir's Elixir: Pure Silver, Joan's True Gold for a Sage Stone, Blood
// Fire, the Mixing Urn, then the elixir turns the Star of Destiny into an
// A grade weapon enchant scroll (with Ladd's skill animation).
func TestFixQ235Walkthrough(t *testing.T) {
	q := b3aStart(t, "Q00235_MimirsElixir")
	b3cRequires(t, q.h, q.name)
	w := q.h.World()
	ladd := &scripttest.Npc{ObjectID: 0x60000001, ID: 30721, Name: "Ladd"}
	joan := &scripttest.Npc{ObjectID: 0x60000002, ID: 30718, Name: "Joan"}
	urn := &scripttest.Npc{ObjectID: 0x60000003, ID: 31149, Name: "Alchemist's Mixing Urn"}
	golem := &scripttest.Npc{ObjectID: 0x60000004, ID: 20965, Name: "Chimera Piece"}
	bloody := &scripttest.Npc{ObjectID: 0x60000005, ID: 21090, Name: "Bloody Guardian"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 74}
	q.talk(young, ladd, "30721-01b.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 75}
	q.talk(hero, ladd, "30721-01a.htm")
	hero.Give(fixQ235StarOfDestiny, 1)
	q.talk(hero, ladd, "30721-01.htm")
	q.event(hero, ladd, "30721-06.htm", "30721-06.htm")
	q.want("accept", hero, scripting.StateStarted, 1)

	q.talk(hero, ladd, "30721-07.htm")
	hero.Give(fixQ235PureSilver, 1)
	q.talk(hero, ladd, "30721-08.htm")
	q.want("silver", hero, scripting.StateStarted, 2)
	q.talk(hero, ladd, "30721-10.htm")

	q.talk(hero, joan, "30718-01.htm")
	q.event(hero, joan, "30718-03.htm", "30718-03.htm")
	q.want("joan", hero, scripting.StateStarted, 3)
	q.talk(hero, joan, "30718-04.htm")

	// A Sage Stone in 2 of 10; the Bloody Guardian needs cond 6 first.
	q.kill(hero, golem, rnd.Get(10, 2))
	b3aWantItems(t, "no stone", hero, map[int]int64{fixQ235SageStone: 0})
	q.kill(hero, bloody)
	q.kill(hero, golem, rnd.Get(10, 1))
	q.want("stone", hero, scripting.StateStarted, 4)
	b3aWantItems(t, "stone", hero, map[int]int64{fixQ235SageStone: 1})
	q.kill(hero, golem)

	q.talk(hero, joan, "30718-05.htm")
	q.want("gold", hero, scripting.StateStarted, 5)
	b3aWantItems(t, "gold", hero, map[int]int64{fixQ235SageStone: 0, fixQ235TrueGold: 1})
	q.talk(hero, joan, "30718-06.htm")

	q.talk(hero, ladd, "30721-11.htm")
	q.event(hero, ladd, "30721-12.htm", "30721-12.htm")
	q.want("mixing stone", hero, scripting.StateStarted, 6)
	b3aWantItems(t, "mixing stone", hero, map[int]int64{fixQ235MixingStone: 1})
	q.talk(hero, ladd, "30721-13.htm")

	// The urn checks the ingredients step by step.
	q.talk(hero, urn, "31149-01.htm")
	q.event(hero, urn, "31149-02.htm", "31149-02.htm")
	q.event(hero, urn, "31149-03.htm", "31149-03.htm")
	q.event(hero, urn, "31149-05.htm", "31149-05.htm")
	q.event(hero, urn, "31149-07.htm", "31149-havent.htm")
	q.event(hero, urn, "31149-success.htm", "31149-havent.htm")

	q.kill(hero, bloody, rnd.Get(10, 5))
	q.kill(hero, bloody, rnd.Get(10, 0))
	q.want("blood fire", hero, scripting.StateStarted, 7)
	b3aWantItems(t, "blood fire", hero, map[int]int64{fixQ235BloodFire: 1})
	q.talk(hero, ladd, "30721-13.htm")

	q.event(hero, urn, "31149-07.htm", "31149-07.htm")
	q.event(hero, urn, "31149-success.htm", "31149-success.htm")
	q.want("elixir", hero, scripting.StateStarted, 8)
	b3aWantItems(t, "elixir", hero, map[int]int64{fixQ235PureSilver: 0, fixQ235TrueGold: 0, fixQ235BloodFire: 0, fixQ235MimirElixir: 1, fixQ235MixingStone: 1})

	q.talk(hero, ladd, "30721-14.htm")
	w.ClearRecords()
	q.event(hero, ladd, "30721-16.htm", "30721-16.htm")
	q.want("done", hero, scripting.StateCompleted, 8)
	b3aWantItems(t, "reward", hero, map[int]int64{fixQ235EnchantA: 1, fixQ235StarOfDestiny: 0, fixQ235MimirElixir: 0, fixQ235MixingStone: 0})
	fixWantAction(t, "reward", w, "magic_skill_use", hero.ObjectID, hero.ObjectID, 4339)
	if got := fixActions(w, "magic_skill_use"); len(got) != 1 || got[0].Level != 1 || got[0].HitTime != time.Millisecond {
		t.Errorf("skill animation %+v", got)
	}
	fixWantAction(t, "reward", w, "social", hero.ObjectID, 0, 3)
	b3cSounds(t, "reward", w, hero, b3aSoundFinish)
	q.talk(hero, ladd, b3aCompleted)
}
