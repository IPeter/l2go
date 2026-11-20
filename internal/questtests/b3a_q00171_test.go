package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Acts of Evil: Alvah's investigation from Arodin's blade molds to Burai's
// Ol Mahum heads, with forced drops and the Turek Orc ambushes.
func TestB3aQ171ActsOfEvil(t *testing.T) {
	q := b3aStart(t, "Q00171_ActsOfEvil")
	w := q.h.World()
	alvah := &scripttest.Npc{ObjectID: 0x60000001, ID: 30381, Name: "Alvah"}
	arodin := &scripttest.Npc{ObjectID: 0x60000002, ID: 30207, Name: "Arodin"}
	tyra := &scripttest.Npc{ObjectID: 0x60000003, ID: 30420, Name: "Tyra"}
	rolento := &scripttest.Npc{ObjectID: 0x60000004, ID: 30437, Name: "Rolento"}
	neti := &scripttest.Npc{ObjectID: 0x60000005, ID: 30425, Name: "Neti"}
	burai := &scripttest.Npc{ObjectID: 0x60000006, ID: 30617, Name: "Burai"}
	turek := &scripttest.Npc{ObjectID: 0x60000010, ID: 20496, Name: "Turek Orc Warlord"}
	ranger := &scripttest.Npc{ObjectID: 0x60000011, ID: 20062, Name: "Ranger"}
	ranger2 := &scripttest.Npc{ObjectID: 0x60000012, ID: 20064, Name: "Ranger 2"}
	smuggler := &scripttest.Npc{ObjectID: 0x60000013, ID: 20438, Name: "Smuggler"}
	olMahum := &scripttest.Npc{ObjectID: 0x60000014, ID: 20066, Name: "Ol Mahum"}
	const (
		bladeMold   = 4239
		tyraBill    = 4240
		report1     = 4241
		report4     = 4244
		contract    = 4245
		directives  = 4246
		certificate = 4247
		cargoBox    = 4248
		olMahumHead = 4249
	)

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 26}
	q.talk(young, alvah, "30381-01a.htm")
	q.want("too young", young, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 27}
	q.kill(hero, turek) // no quest: no draw
	q.talk(hero, alvah, "30381-01.htm")
	q.event(hero, alvah, "30381-02.htm", "30381-02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, alvah, "30381-02a.htm")
	q.talk(hero, arodin, "30207-01.htm")
	q.event(hero, arodin, "30207-02.htm", "30207-02.htm")
	q.want("Arodin", hero, scripting.StateStarted, 2)
	q.talk(hero, arodin, "30207-01a.htm")
	q.talk(hero, tyra, "30420-01b.htm")

	// Blade molds: 53 % from the warlord; the sixth mold calls an ambush,
	// and from ten on each kill may call one (25 %).
	q.kill(hero, turek, rnd.Get(1000000, 529999))
	q.kill(hero, turek, rnd.Get(1000000, 530000))
	b3aWantItems(t, "first molds", hero, map[int]int64{bladeMold: 1})
	hero.Give(bladeMold, 4)
	q.kill(hero, turek, rnd.Get(1000000, 0))
	if len(w.Spawned) != 1 || w.Spawned[0].ID != 27190 {
		t.Fatalf("spawned %+v, want one 27190 ambush", w.Spawned)
	}
	hero.Give(bladeMold, 3)
	q.kill(hero, turek, rnd.Get(1000000, 0))
	q.kill(hero, turek, rnd.Get(1000000, 0), rnd.Get(100, 25))
	b3aWantItems(t, "eleven molds", hero, map[int]int64{bladeMold: 11})
	hero.Give(bladeMold, 9)
	q.kill(hero, turek, rnd.Get(1000000, 0), rnd.Get(100, 24))
	b3aWantItems(t, "twenty molds", hero, map[int]int64{bladeMold: 20})
	if len(w.Spawned) != 2 {
		t.Fatalf("spawned %d ambushes, want 2", len(w.Spawned))
	}

	q.talk(hero, tyra, "30420-01.htm")
	q.want("Tyra", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "Tyra", hero, map[int]int64{bladeMold: 0, tyraBill: 1})
	q.talk(hero, tyra, "30420-01a.htm")
	q.talk(hero, arodin, "30207-03.htm")
	q.want("bill", hero, scripting.StateStarted, 4)
	b3aWantItems(t, "bill", hero, map[int]int64{tyraBill: 0})
	q.talk(hero, arodin, "30207-03a.htm")
	q.talk(hero, tyra, "30420-02.htm")
	q.talk(hero, alvah, "30381-03.htm")
	q.event(hero, alvah, "30381-04.htm", "30381-04.htm")
	q.want("rangers", hero, scripting.StateStarted, 5)
	q.talk(hero, alvah, "30381-04a.htm")

	// Ranger reports: the first for sure, the others at 20 %.
	q.kill(hero, ranger)
	q.kill(hero, ranger2, rnd.Get(100, 19))
	q.kill(hero, ranger, rnd.Get(100, 20))
	q.kill(hero, ranger, rnd.Get(100, 0))
	q.kill(hero, ranger2, rnd.Get(100, 0))
	b3aWantItems(t, "reports", hero, map[int]int64{report1: 1, report1 + 1: 1, report1 + 2: 1, report4: 1})
	q.talk(hero, alvah, "30381-05.htm")
	q.want("reports", hero, scripting.StateStarted, 6)
	b3aWantItems(t, "reports given", hero, map[int]int64{report1: 0, report1 + 1: 0, report1 + 2: 0, report4: 0})
	q.talk(hero, alvah, "30381-05a.htm")

	q.kill(hero, smuggler, rnd.Get(100, 10))
	q.kill(hero, smuggler, rnd.Get(100, 9))
	q.kill(hero, smuggler, rnd.Get(100, 0))
	b3aWantItems(t, "smuggler", hero, map[int]int64{contract: 1, directives: 1})
	q.talk(hero, alvah, "30381-06.htm")
	q.event(hero, alvah, "30381-07.htm", "30381-07.htm")
	q.want("contract", hero, scripting.StateStarted, 7)
	b3aWantItems(t, "contract", hero, map[int]int64{contract: 0, directives: 1})
	q.talk(hero, alvah, "30381-07a.htm")

	q.talk(hero, neti, "30425-01.htm")
	q.want("Neti", hero, scripting.StateStarted, 8)
	q.talk(hero, neti, "30425-02.htm")
	q.talk(hero, rolento, "30437-01.htm")
	q.event(hero, rolento, "30437-03.htm", "30437-03.htm")
	q.want("Rolento", hero, scripting.StateStarted, 9)
	b3aWantItems(t, "Rolento", hero, map[int]int64{cargoBox: 1, certificate: 1})
	q.talk(hero, rolento, "30437-03a.htm")

	q.talk(hero, burai, "30617-01.htm")
	q.event(hero, burai, "30617-04.htm", "30617-04.htm")
	q.want("Burai", hero, scripting.StateStarted, 10)
	b3aWantItems(t, "Burai", hero, map[int]int64{cargoBox: 0, certificate: 0, directives: 0})
	q.talk(hero, burai, "30617-04a.htm")

	q.kill(hero, olMahum, rnd.Bool(false))
	q.kill(hero, olMahum, rnd.Bool(true))
	b3aWantItems(t, "one head", hero, map[int]int64{olMahumHead: 1})
	hero.Give(olMahumHead, 29)
	q.kill(hero, olMahum) // 30 heads: no draw
	q.talk(hero, burai, "30617-05.htm")
	q.want("heads", hero, scripting.StateStarted, 11)
	b3aWantItems(t, "heads", hero, map[int]int64{olMahumHead: 0, b3aAdena: 8000})

	q.talk(hero, alvah, "30381-08.htm")
	q.want("done", hero, scripting.StateCompleted, 11)
	b3aWantItems(t, "done", hero, map[int]int64{b3aAdena: 98000})
	q.talk(hero, alvah, b3aCompleted)
}
