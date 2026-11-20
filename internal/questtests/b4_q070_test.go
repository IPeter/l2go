package questtests

import (
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Saga of the Phoenix Knight (Q070, scripts/lib/saga.lua) from the start
// to cond 20. The last step, the class change to Phoenix Knight, needs
// player:set_class_id, which the fake world's players do not have.
func TestB4Q070Walkthrough(t *testing.T) {
	h := b4Load(t, b4Q070)
	w := h.World()
	sedrick := &scripttest.Npc{ObjectID: 0x60000001, ID: b4Sedrick, Name: "Sedrick"}
	npc2 := &scripttest.Npc{ObjectID: 0x60000002, ID: b4Npc2}
	npc3 := &scripttest.Npc{ObjectID: 0x60000003, ID: b4Npc3}
	tablet1 := &scripttest.Npc{ObjectID: 0x60000006, ID: b4Tablet1}
	tablet2 := &scripttest.Npc{ObjectID: 0x60000007, ID: b4Tablet2}
	tablet3 := &scripttest.Npc{ObjectID: 0x60000008, ID: b4Tablet3}
	tablet4 := &scripttest.Npc{ObjectID: 0x60000009, ID: b4Tablet4}
	tablet5 := &scripttest.Npc{ObjectID: 0x6000000a, ID: b4Tablet5}
	tablet6 := &scripttest.Npc{ObjectID: 0x6000000b, ID: b4Tablet6}

	// Another class, even at level 76, gets nothing.
	knight := &scripttest.Player{ObjectID: 0x10000100, Name: "Knight", Level: 76, ClassID: 4}
	reply, ok := h.Talk(b4Q070, knight, sedrick)
	b4NoQuestMsg(t, "another class", reply, ok)

	// A level 75 Paladin hears the offer but is sent away. (The saga calls
	// exit_quest(true) on the CREATED state, which, as in Mobius, only ends
	// a started quest.)
	young := &scripttest.Player{ObjectID: 0x10000101, Name: "Young", Level: 75, ClassID: b4Paladin}
	reply, ok = h.Talk(b4Q070, young, sedrick)
	b4Page(t, "young offer", reply, ok, "0-01.htm")
	reply, ok = h.Event(b4Q070, "0-1", young, sedrick)
	b4Page(t, "young", reply, ok, "0-02.htm")
	b4Quest(t, "young", young, b4Q070, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 76, ClassID: b4Paladin}
	reply, ok = h.Talk(b4Q070, hero, sedrick)
	b4Page(t, "offer", reply, ok, "0-01.htm")
	reply, ok = h.Event(b4Q070, "0-1", hero, sedrick)
	b4Page(t, "0-1", reply, ok, "0-05.htm")
	reply, ok = h.Event(b4Q070, "0-012.htm", hero, sedrick)
	b4Page(t, "a page link", reply, ok, "0-012.htm")
	reply, ok = h.Event(b4Q070, "accept", hero, sedrick)
	b4Page(t, "accept", reply, ok, "0-03.htm")
	b4Quest(t, "accept", hero, b4Q070, scripting.StateStarted, 1)
	b4Items(t, "accept", hero, map[int]int64{b4Item11: 1})
	reply, ok = h.Talk(b4Q070, hero, sedrick)
	b4Page(t, "Sedrick cond 1", reply, ok, "0-04.htm")

	// Steps 1 to 4: npc[3], then npc[2] with the Ice Crystal and items[12].
	reply, ok = h.Talk(b4Q070, hero, npc3)
	b4Page(t, "npc3 cond 1", reply, ok, "2-01.htm")
	reply, ok = h.Event(b4Q070, "2-1", hero, npc3)
	b4Page(t, "2-1", reply, ok, "2-05.htm")
	b4Quest(t, "2-1", hero, b4Q070, scripting.StateStarted, 2)
	reply, ok = h.Talk(b4Q070, hero, npc3)
	b4Page(t, "npc3 cond 2", reply, ok, "2-02.htm")
	reply, ok = h.Talk(b4Q070, hero, npc2)
	b4Page(t, "npc2 cond 2", reply, ok, "1-01.htm")
	reply, ok = h.Event(b4Q070, "1-3", hero, npc2)
	b4Page(t, "1-3", reply, ok, "1-05.htm")
	b4Quest(t, "1-3", hero, b4Q070, scripting.StateStarted, 3)
	reply, ok = h.Talk(b4Q070, hero, npc2)
	b4NoQuestMsg(t, "no crystal", reply, ok)
	hero.Give(b4IceCrystal, 1)
	reply, ok = h.Talk(b4Q070, hero, npc2)
	b4Page(t, "crystal only", reply, ok, "1-02.htm")
	hero.Give(b4Item12, 1)
	reply, ok = h.Talk(b4Q070, hero, npc2)
	b4Page(t, "both", reply, ok, "1-03.htm")
	reply, ok = h.Event(b4Q070, "1-4", hero, npc2)
	b4Page(t, "1-4", reply, ok, "1-06.htm")
	b4Quest(t, "1-4", hero, b4Q070, scripting.StateStarted, 4)
	b4Items(t, "1-4", hero, map[int]int64{b4IceCrystal: 0, b4Item12: 0, b4Item2: 1})
	reply, ok = h.Talk(b4Q070, hero, npc2)
	b4Page(t, "npc2 cond 4", reply, ok, "1-04.htm")
	reply, ok = h.Talk(b4Q070, hero, npc3)
	b4Page(t, "npc3 cond 4", reply, ok, "2-03.htm")
	reply, ok = h.Event(b4Q070, "2-2", hero, npc3)
	b4Page(t, "2-2", reply, ok, "2-06.htm")
	b4Quest(t, "2-2", hero, b4Q070, scripting.StateStarted, 5)
	b4Items(t, "2-2", hero, map[int]int64{b4Item2: 0, b4Item5: 1})

	// Steps 5 and 6: the first Tablet of Vision and ten Guardian Angels.
	reply, ok = h.Talk(b4Q070, hero, npc3)
	b4Page(t, "npc3 cond 5", reply, ok, "2-04.htm")
	reply, ok = h.Talk(b4Q070, hero, tablet1)
	b4Page(t, "tablet 1", reply, ok, "5-01.htm")
	reply, ok = h.Event(b4Q070, "5-1", hero, tablet1)
	b4Page(t, "5-1", reply, ok, "5-02.htm")
	b4Quest(t, "5-1", hero, b4Q070, scripting.StateStarted, 6)
	b4Items(t, "5-1", hero, map[int]int64{b4Item5: 0})
	reply, ok = h.Talk(b4Q070, hero, tablet1)
	b4Page(t, "tablet 1 cond 6", reply, ok, "5-03.htm")
	reply, ok = h.Talk(b4Q070, hero, tablet2)
	b4Page(t, "tablet 2 cond 6", reply, ok, "6-01.htm")
	angel := &scripttest.Npc{ObjectID: 0x60000020, ID: b4Guardian}
	for range 10 {
		h.Kill(hero, angel)
	}
	b4Quest(t, "angels", hero, b4Q070, scripting.StateStarted, 7)
	b4Items(t, "angels", hero, map[int]int64{b4Item6: 1})

	// Steps 7 to 9: the second tablet, the monster of the third.
	reply, ok = h.Talk(b4Q070, hero, tablet2)
	b4Page(t, "tablet 2 cond 7", reply, ok, "6-02.htm")
	reply, ok = h.Event(b4Q070, "6-1", hero, tablet2)
	b4Page(t, "6-1", reply, ok, "6-03.htm")
	b4Quest(t, "6-1", hero, b4Q070, scripting.StateStarted, 8)
	b4Items(t, "6-1", hero, map[int]int64{b4Item6: 0})
	reply, ok = h.Talk(b4Q070, hero, tablet2)
	b4Page(t, "tablet 2 cond 8", reply, ok, "6-04.htm")
	reply, ok = h.Talk(b4Q070, hero, tablet3)
	b4Page(t, "tablet 3", reply, ok, "7-01.htm")
	reply, ok = h.Event(b4Q070, "7-1", hero, tablet3)
	b4Page(t, "7-1", reply, ok, "7-02.htm")
	mob1 := b4Spawned(h, b4Mob1)
	if len(mob1) != 1 {
		t.Fatalf("spawned %+v", w.Spawned)
	}
	w.ClearRecords()
	h.Advance(500 * time.Millisecond)
	b4WantSaid(t, "mob 1 appears", h, strings.ReplaceAll(b4Text1, "PLAYERNAME", "Hero"))
	w.ClearRecords()
	h.Kill(hero, mob1[0])
	b4WantSaid(t, "mob 1 dies", h, b4Text13)
	b4Quest(t, "mob 1", hero, b4Q070, scripting.StateStarted, 9)
	b4Items(t, "mob 1", hero, map[int]int64{b4Item7: 1})
	if !mob1[0].Deleted {
		t.Error("the dead monster was not deleted")
	}
	reply, ok = h.Talk(b4Q070, hero, tablet3)
	b4Page(t, "tablet 3 cond 9", reply, ok, "7-05.htm")
	reply, ok = h.Event(b4Q070, "7-2", hero, tablet3)
	b4Page(t, "7-2", reply, ok, "7-06.htm")
	b4Quest(t, "7-2", hero, b4Q070, scripting.StateStarted, 10)
	b4Items(t, "7-2", hero, map[int]int64{b4Item7: 0})

	// Steps 10 to 13: Sedrick (npc[4]) and the Divine Stone of Wisdom.
	reply, ok = h.Talk(b4Q070, hero, tablet3)
	b4Page(t, "tablet 3 cond 10", reply, ok, "7-07.htm")
	reply, ok = h.Talk(b4Q070, hero, sedrick)
	b4Page(t, "Sedrick cond 10", reply, ok, "3-01.htm")
	reply, ok = h.Event(b4Q070, "3-5", hero, sedrick)
	b4Page(t, "3-5", reply, ok, "3-07.htm")
	reply, ok = h.Event(b4Q070, "3-6", hero, sedrick)
	b4Page(t, "3-6", reply, ok, "3-02.htm")
	b4Quest(t, "3-6", hero, b4Q070, scripting.StateStarted, 11)
	reply, ok = h.Talk(b4Q070, hero, sedrick)
	b4Page(t, "no stone", reply, ok, "3-04.htm")
	reply, ok = h.Event(b4Q070, "3-7", hero, sedrick)
	b4Page(t, "3-7", reply, ok, "3-03.htm")
	b4Quest(t, "3-7", hero, b4Q070, scripting.StateStarted, 12)
	hero.Give(b4DivineStone, 1)
	reply, ok = h.Talk(b4Q070, hero, sedrick)
	b4Page(t, "stone", reply, ok, "3-05.htm")
	reply, ok = h.Event(b4Q070, "3-8", hero, sedrick)
	b4Page(t, "3-8", reply, ok, "3-08.htm")
	b4Quest(t, "3-8", hero, b4Q070, scripting.StateStarted, 13)
	b4Items(t, "3-8", hero, map[int]int64{b4DivineStone: 0, b4Item8: 1})

	// Steps 13 to 15: the fourth tablet, npc[12].
	reply, ok = h.Talk(b4Q070, hero, sedrick)
	b4Page(t, "Sedrick cond 13", reply, ok, "3-06.htm")
	reply, ok = h.Talk(b4Q070, hero, tablet4)
	b4Page(t, "tablet 4", reply, ok, "8-01.htm")
	reply, ok = h.Event(b4Q070, "8-1", hero, tablet4)
	b4Page(t, "8-1", reply, ok, "8-02.htm")
	b4Quest(t, "8-1", hero, b4Q070, scripting.StateStarted, 14)
	b4Items(t, "8-1", hero, map[int]int64{b4Item8: 0})
	reply, ok = h.Talk(b4Q070, hero, tablet4)
	b4Page(t, "tablet 4 cond 14", reply, ok, "8-03.htm")
	reply, ok = h.Talk(b4Q070, hero, npc3)
	b4Page(t, "npc12", reply, ok, "11-01.htm")
	reply, ok = h.Event(b4Q070, "11-9", hero, npc3)
	b4Page(t, "11-9", reply, ok, "11-03.htm")
	b4Quest(t, "11-9", hero, b4Q070, scripting.StateStarted, 15)
	reply, ok = h.Talk(b4Q070, hero, npc3)
	b4Page(t, "npc12 cond 15", reply, ok, "11-02.htm")
	reply, ok = h.Talk(b4Q070, hero, tablet5)
	b4Page(t, "tablet 5 cond 15", reply, ok, "9-01.htm")

	// Step 15: marks of Halisha, then Archon of Halisha.
	hero.Give(b4HalishaMark, 700)
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, &scripttest.Npc{ObjectID: 0x60000030, ID: b4Minion})
	archon := b4Spawned(h, b4Archon)
	if len(archon) != 1 {
		t.Fatalf("spawned %+v", w.Spawned)
	}
	h.Kill(hero, archon[0])
	b4Quest(t, "archon", hero, b4Q070, scripting.StateStarted, 16)
	b4Items(t, "archon", hero, map[int]int64{b4HalishaMark: 0, b4Item9: 1})

	// Steps 16 to 18: the fifth tablet, the sixth, and the fight of npc[5]
	// against mob[3].
	reply, ok = h.Talk(b4Q070, hero, tablet5)
	b4Page(t, "tablet 5 cond 16", reply, ok, "9-02.htm")
	reply, ok = h.Event(b4Q070, "9-1", hero, tablet5)
	b4Page(t, "9-1", reply, ok, "9-03.htm")
	b4Quest(t, "9-1", hero, b4Q070, scripting.StateStarted, 17)
	b4Items(t, "9-1", hero, map[int]int64{b4Item9: 0})
	reply, ok = h.Talk(b4Q070, hero, tablet5)
	b4Page(t, "tablet 5 cond 17", reply, ok, "9-04.htm")
	reply, ok = h.Talk(b4Q070, hero, tablet6)
	b4Page(t, "tablet 6", reply, ok, "10-01.htm")
	reply, ok = h.Event(b4Q070, "10-1", hero, tablet6)
	b4Page(t, "10-1", reply, ok, "10-02.htm")
	mob3, mob2 := b4Spawned(h, b4Mob3), b4Spawned(h, b4Npc5)
	if len(mob3) != 1 || len(mob2) != 1 {
		t.Fatalf("spawned %+v", w.Spawned)
	}
	if mob3[0].X != 46087 || mob2[0].X != 46066 {
		t.Errorf("spawn points %d and %d", mob3[0].X, mob2[0].X)
	}
	reply, ok = h.Event(b4Q070, "10-1", hero, tablet6)
	b4Page(t, "10-1 again", reply, ok, "10-03.htm")
	reply, ok = h.FirstTalk(hero, mob2[0])
	b4Page(t, "npc5 first", reply, ok, "4-03.htm")
	// Another player's click shows the NPC's own page.
	stranger := &scripttest.Player{ObjectID: 0x10000200, Name: "Stranger", Level: 40}
	if reply, ok = h.FirstTalk(stranger, mob2[0]); ok && reply.Kind != scripting.ReplyNone {
		t.Errorf("stranger: reply = %+v, %v; want the NPC's own page", reply, ok)
	}

	w.ClearRecords()
	h.Advance(500 * time.Millisecond)
	b4WantSaid(t, "the fight starts", h, b4Text15, b4Text8)
	b4Var(t, "the fight starts", hero, b4Q070, "Quest1", "0")

	for range 15 {
		h.Attack(hero, mob3[0], 100)
	}
	b4Var(t, "mob 3 beaten", hero, b4Q070, "Tab", "1")
	b4Var(t, "mob 3 beaten", hero, b4Q070, "Quest0", "1")
	if !mob3[0].Deleted {
		t.Error("mob[3] stays after 15 hits")
	}
	reply, ok = h.FirstTalk(hero, mob2[0])
	b4Page(t, "npc5 after the fight", reply, ok, "4-06.htm")
	reply, ok = h.Event(b4Q070, "4-1", hero, mob2[0])
	b4Page(t, "4-1", reply, ok, "4-010.htm")
	w.ClearRecords()
	h.Event(b4Q070, "4-3", hero, mob2[0])
	b4WantSaid(t, "4-3", h, b4Text14)
	b4Quest(t, "4-3", hero, b4Q070, scripting.StateStarted, 18)
	b4Items(t, "4-3", hero, map[int]int64{b4Item10: 1})
	if !mob2[0].Deleted {
		t.Error("npc[5] stays after 4-3")
	}
	// The game cancels the timers of deleted NPCs; nothing more happens.
	h.Engine().NpcGone(mob2[0].ObjectID)
	h.Engine().NpcGone(mob3[0].ObjectID)
	w.ClearRecords()
	h.Advance(2 * time.Minute)
	b4WantSaid(t, "after the fight", h)
	b4NoTimerErrors(t, h)
	reply, ok = h.FirstTalk(hero, mob2[0])
	b4Page(t, "npc5 cond 18", reply, ok, "4-08.htm")

	// Steps 18 to 20.
	reply, ok = h.Talk(b4Q070, hero, tablet6)
	b4Page(t, "tablet 6 cond 18", reply, ok, "10-05.htm")
	reply, ok = h.Event(b4Q070, "10-2", hero, tablet6)
	b4Page(t, "10-2", reply, ok, "10-06.htm")
	b4Quest(t, "10-2", hero, b4Q070, scripting.StateStarted, 19)
	b4Items(t, "10-2", hero, map[int]int64{b4Item10: 0})
	reply, ok = h.Talk(b4Q070, hero, tablet6)
	b4Page(t, "tablet 6 cond 19", reply, ok, "10-07.htm")
	reply, ok = h.Talk(b4Q070, hero, sedrick)
	b4Page(t, "Sedrick cond 19", reply, ok, "0-06.htm")

	// Below level 76 Sedrick waits (cond 20); Mobius's level check.
	hero.Level = 75
	reply, ok = h.Event(b4Q070, "0-2", hero, sedrick)
	b4Page(t, "0-2 at 75", reply, ok, "0-08.htm")
	b4Quest(t, "0-2 at 75", hero, b4Q070, scripting.StateStarted, 20)
	b4Items(t, "0-2 at 75", hero, map[int]int64{b4Item11: 0})
	reply, ok = h.Talk(b4Q070, hero, sedrick)
	b4Page(t, "cond 20 at 75", reply, ok, "0-010.htm")
}
