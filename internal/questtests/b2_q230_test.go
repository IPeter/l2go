package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q230 = "Q00230_TestOfTheSummoner"

// Test of the Summoner up to the duels: Galatea, Lara's list and its
// monster parts, the beginner arcana. The six summon duels need pets, the
// summon of a player, NPC HP, casts, deaths and death notices, so they are
// not played: the test sets the duel stage of the last summoner instead and
// checks the arcana and the Mark of Summoner from there.
func TestB2Q230Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q230)
	lara := &scripttest.Npc{ObjectID: 0x60000001, ID: 30063}
	galatea := &scripttest.Npc{ObjectID: 0x60000002, ID: 30634}
	almors := &scripttest.Npc{ObjectID: 0x60000003, ID: 30635}
	brynthea := &scripttest.Npc{ObjectID: 0x60000004, ID: 30640}

	knight := &scripttest.Player{ObjectID: 0x10000001, Name: "Knight", ClassID: 4, Level: 39}
	reply, ok := h.Talk(b2Q230, knight, galatea)
	b2Page(t, "knight", reply, ok, "30634-01.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 39, Race: 2, Level: 38}
	reply, ok = h.Talk(b2Q230, young, galatea)
	b2Page(t, "level 38", reply, ok, "30634-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", ClassID: 26, Race: 1, Level: 39}
	reply, ok = h.Talk(b2Q230, hero, galatea)
	b2Page(t, "offer", reply, ok, "30634-03.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q230, "30634-08.htm", hero, galatea)
	b2Page(t, "accept", reply, ok, "30634-08a.htm")
	b2Quest(t, "accept", hero, b2Q230, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{3352: 1, 7562: 122})
	b2Q223Var(t, hero, "secondClassChange39", "true")
	b2Sounds(t, "accept", h, hero, b2Accept)
	for _, name := range []string{"Almors", "Camoniell", "Belthus", "Basilla", "Celestiel", "Brynthea"} {
		if v, _ := hero.Quest(b2Q230).Get(name); v != "1" {
			t.Errorf("accept: %s = %q, want 1", name, v)
		}
	}
	reply, ok = h.Talk(b2Q230, hero, galatea)
	b2Page(t, "Galatea cond 1", reply, ok, "30634-09.htm")

	// Lara's list 1: red spore sacs and lizardman amulets.
	reply, ok = h.Talk(b2Q230, hero, lara)
	b2Page(t, "Lara", reply, ok, "30063-01.htm")
	b2Rand(t, h, rnd.Get(5, 0))
	h.Event(b2Q230, "30063-02.htm", hero, lara)
	b2Quest(t, "list 1", hero, b2Q230, scripting.StateStarted, 2)
	b2Items(t, "list 1", hero, map[int]int64{3352: 0, 3347: 1})
	reply, ok = h.Talk(b2Q230, hero, galatea)
	b2Page(t, "Galatea cond 2", reply, ok, "30634-10.htm")
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000010, ID: 20555}, 1, rnd.Get(100, 80))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000010, ID: 20555}, 31, rnd.Get(100, 79))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000011, ID: 20577}, 1, rnd.Get(100, 25))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000012, ID: 20580}, 10, rnd.Get(100, 49))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000013, ID: 20582}, 19, rnd.Get(100, 74))
	reply, ok = h.Talk(b2Q230, hero, lara)
	b2Page(t, "Lara 29 amulets", reply, ok, "30063-05.htm")
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000011, ID: 20578}, 1, rnd.Get(100, 24))
	b2Items(t, "list 1 parts", hero, map[int]int64{3338: 30, 3337: 30})
	reply, ok = h.Talk(b2Q230, hero, lara)
	b2Page(t, "Lara parts", reply, ok, "30063-06.htm")
	b2Quest(t, "arcana", hero, b2Q230, scripting.StateStarted, 3)
	b2Items(t, "arcana", hero, map[int]int64{3347: 0, 3338: 0, 3337: 0, 3353: 2})
	if _, ok := hero.Quest(b2Q230).Get("Lara"); ok {
		t.Error("Lara is still set")
	}
	reply, ok = h.Talk(b2Q230, hero, lara)
	b2Page(t, "Lara again", reply, ok, "30063-03.htm")
	reply, ok = h.Talk(b2Q230, hero, galatea)
	b2Page(t, "Galatea arcana", reply, ok, "30634-11.htm")

	// The summoners: Almors would take an arcana for a duel.
	reply, ok = h.Talk(b2Q230, hero, almors)
	b2Page(t, "Almors", reply, ok, "30635-01.htm")
	reply, ok = h.Event(b2Q230, "30635-02.htm", hero, almors)
	b2Page(t, "Almors arcana", reply, ok, "30635-03.htm")

	// After five won duels the last summoner's arcana completes the set.
	st := hero.Quest(b2Q230)
	for _, name := range []string{"Almors", "Camoniell", "Belthus", "Basilla", "Celestiel"} {
		st.Set(name, "7")
	}
	st.Set("Brynthea", "6")
	for _, id := range []int{3354, 3355, 3356, 3357, 3358, 3389} {
		hero.Give(id, 1)
	}
	reply, ok = h.Talk(b2Q230, hero, almors)
	b2Page(t, "Almors won", reply, ok, "30635-10.htm")
	reply, ok = h.Talk(b2Q230, hero, brynthea)
	b2Page(t, "Brynthea won", reply, ok, "30640-07.htm")
	b2Quest(t, "six arcana", hero, b2Q230, scripting.StateStarted, 4)
	b2Items(t, "six arcana", hero, map[int]int64{3389: 0, 3359: 1})

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q230, hero, galatea)
	b2Page(t, "mark", reply, ok, "30634-12.htm")
	b2Quest(t, "mark", hero, b2Q230, scripting.StateCompleted, 4)
	b2Items(t, "mark", hero, map[int]int64{3353: 0, 3354: 0, 3355: 0, 3356: 0, 3357: 0, 3358: 0, 3359: 0, 3336: 1, 7562: 122})
	b2ExpSp(t, "mark", hero, 148409, 30000)
	b2Q223Social(t, h, hero)
	b2Sounds(t, "mark", h, hero, b2Finish)
	reply, ok = h.Talk(b2Q230, hero, galatea)
	b2Completed(t, "completed", reply, ok)
}

// Lara's other lists name other monsters.
func TestB2Q230LaraList5(t *testing.T) {
	h := b2Load(t, b2Q230)
	lara := &scripttest.Npc{ObjectID: 0x60000001, ID: 30063}
	galatea := &scripttest.Npc{ObjectID: 0x60000002, ID: 30634}
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", ClassID: 11, Level: 39,
		Vars: map[string]string{"secondClassChange39": "true"}}
	h.Talk(b2Q230, hero, galatea)
	reply, ok := h.Event(b2Q230, "30634-08.htm", hero, galatea)
	b2Page(t, "accept", reply, ok, "30634-08.htm")
	b2Items(t, "accept", hero, map[int]int64{7562: 0})
	b2Rand(t, h, rnd.Get(5, 4))
	h.Event(b2Q230, "30063-02.htm", hero, lara)
	b2Items(t, "list 5", hero, map[int]int64{3351: 1})
	// List 1 monsters give nothing now (no draw either).
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000010, ID: 20555}, 1)
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000011, ID: 20089}, 30, rnd.Get(100, 59))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000012, ID: 20176}, 30, rnd.Get(100, 49))
	b2Items(t, "list 5 parts", hero, map[int]int64{3338: 0, 3344: 30, 3346: 30})
	reply, ok = h.Talk(b2Q230, hero, lara)
	b2Page(t, "Lara parts", reply, ok, "30063-06.htm")
	b2Quest(t, "arcana", hero, b2Q230, scripting.StateStarted, 3)
	b2Items(t, "arcana", hero, map[int]int64{3351: 0, 3344: 0, 3346: 0, 3353: 2})
}
