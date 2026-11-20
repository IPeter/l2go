package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ333 = "Q00333_HuntOfTheBlackLion"

// Hunt of the Black Lion: Sophya's letters, drops and cargo boxes, Redfoot's
// box opening, Rupio's statue, Undrias, Morgan, lion claws, and the end.
func TestB3bQ333Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ333)
	npc := func(oid int32, id int) *scripttest.Npc { return &scripttest.Npc{ObjectID: oid, ID: id} }
	sophya, redfoot, rupio := npc(0x60000001, 30735), npc(0x60000002, 30736), npc(0x60000003, 30471)
	undrias, morgan := npc(0x60000004, 30130), npc(0x60000005, 30737)
	zombie := npc(0x60000010, 20160)

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 24}
	reply, ok := h.Talk(b3bQ333, young, sophya)
	b3bPage(t, "level 24", reply, ok, "30735-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 25}
	reply, ok = h.Talk(b3bQ333, hero, sophya)
	b3bPage(t, "no mark", reply, ok, "30735-02.htm")
	hero.Give(1369, 1)
	reply, ok = h.Talk(b3bQ333, hero, sophya)
	b3bPage(t, "offer", reply, ok, "30735-03.htm")
	h.Event(b3bQ333, "30735-04.htm", hero, sophya)
	b3bQuest(t, "accept", hero, b3bQ333, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ333, hero, sophya)
	b3bPage(t, "no letter", reply, ok, "30735-14.htm")
	h.Event(b3bQ333, "30735-10.htm", hero, sophya)
	b3bItems(t, "letter", hero, map[int]int64{3671: 1})

	// Ash below 500000, a cargo box below 90000 (of a million).
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 499999), rnd.Get(1000000, 89999), rnd.Get(1000000, 500000), rnd.Get(1000000, 90000)))
	h.Kill(hero, zombie)
	h.Kill(hero, zombie)
	b3bItems(t, "drops", hero, map[int]int64{3848: 1, 3440: 1})

	hero.Give(3848, 19)
	reply, ok = h.Talk(b3bQ333, hero, sophya)
	b3bPage(t, "sell ash", reply, ok, "30735-23.htm")
	b3bItems(t, "sell ash", hero, map[int]int64{3848: 0, 57: 700, 3675: 1})

	// Redfoot opens the box for 650 adena: i0 = 76, i1 = 10, then a statue piece.
	reply, ok = h.Talk(b3bQ333, hero, redfoot)
	b3bPage(t, "Redfoot", reply, ok, "30736-02.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 76), rnd.Get(100, 10), rnd.Bool(true)))
	reply, ok = h.Event(b3bQ333, "30736-03.htm", hero, redfoot)
	b3bPage(t, "open box", reply, ok, "30736-04n.htm")
	b3bItems(t, "open box", hero, map[int]int64{57: 50, 3440: 0, 3457: 1})
	reply, ok = h.Event(b3bQ333, "30736-03.htm", hero, redfoot)
	b3bPage(t, "no box", reply, ok, "30736-05.htm")

	// Rupio builds the statue, Undrias buys it.
	hero.Give(3458, 1)
	hero.Give(3459, 1)
	hero.Give(3460, 1)
	reply, ok = h.Talk(b3bQ333, hero, rupio)
	b3bPage(t, "Rupio", reply, ok, "30471-02.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Bool(true)))
	reply, ok = h.Event(b3bQ333, "30471-03.htm", hero, rupio)
	b3bPage(t, "statue", reply, ok, "30471-04.htm")
	reply, ok = h.Talk(b3bQ333, hero, undrias)
	b3bPage(t, "Undrias", reply, ok, "30130-03.htm")
	h.Event(b3bQ333, "30130-04.htm", hero, undrias)
	b3bItems(t, "statue sold", hero, map[int]int64{3461: 0, 57: 30050})

	// Morgan pays for a cargo box with a guild coin.
	hero.Give(3441, 1)
	reply, ok = h.Event(b3bQ333, "30737-06.htm", hero, morgan)
	b3bPage(t, "Morgan", reply, ok, "30737-03.htm")
	b3bItems(t, "Morgan", hero, map[int]int64{3441: 0, 57: 30150, 3677: 1})

	// Ten lion claws: a lion eye and soulshots for a fighter.
	hero.Give(3675, 9)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 30)))
	reply, ok = h.Event(b3bQ333, "30735-16.htm", hero, sophya)
	b3bPage(t, "claws", reply, ok, "30735-17a.htm")
	b3bItems(t, "claws", hero, map[int]int64{3675: 0, 3676: 1, 1463: 100})

	reply, ok = h.Event(b3bQ333, "30735-26.htm", hero, sophya)
	b3bPage(t, "quit", reply, ok, "30735-26.htm")
	b3bItems(t, "quit", hero, map[int]int64{1369: 0, 3676: 0, 3677: 0, 3671: 0, 57: 42550})
	b3bNoQuest(t, "quit", hero, b3bQ333)
}
