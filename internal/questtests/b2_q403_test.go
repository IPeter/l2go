package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q403 = "Q00403_PathOfTheRogue"

// Path to a Rogue: Spartoi bones for Neti with her weapons in hand, the
// stolen goods of Cats Eye Bandit, Bezique's recommendation.
func TestB2Q403Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q403)
	bezique := &scripttest.Npc{ObjectID: 0x60000001, ID: 30379}
	neti := &scripttest.Npc{ObjectID: 0x60000002, ID: 30425}
	soldier := &scripttest.Npc{ObjectID: 0x60000010, ID: 20035}
	leader := &scripttest.Npc{ObjectID: 0x60000011, ID: 20042}
	scout := &scripttest.Npc{ObjectID: 0x60000012, ID: 20054}
	bandit := &scripttest.Npc{ObjectID: 0x60000013, ID: 27038}

	// The refusals of 30379-05.htm.
	for i, tt := range []struct {
		name         string
		class, level int
		reward       bool
		page         string
	}{
		{"rogue already", 7, 20, false, "30379-02a.htm"},
		{"mage", 10, 20, false, "30379-02.htm"},
		{"too young", 0, 18, false, "30379-02.htm"},
		{"recommendation", 0, 19, true, "30379-04.htm"},
		{"fits", 0, 19, false, "30379-05.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000100 + int32(i), Name: tt.name, Level: tt.level, ClassID: tt.class}
		if tt.reward {
			p.Give(1190, 1)
		}
		reply, ok := h.Talk(b2Q403, p, bezique)
		b2Page(t, tt.name+" talk", reply, ok, "30379-01.htm")
		reply, ok = h.Event(b2Q403, "30379-05.htm", p, bezique)
		b2Page(t, tt.name, reply, ok, tt.page)
		b2Quest(t, tt.name, p, b2Q403, scripting.StateCreated, 0)
	}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 0}
	reply, ok := h.Talk(b2Q403, hero, bezique)
	b2Page(t, "offer", reply, ok, "30379-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q403, "30379-06.htm", hero, bezique)
	b2Page(t, "accept", reply, ok, "30379-06.htm")
	b2Quest(t, "accept", hero, b2Q403, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{1180: 1})
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q403, hero, bezique)
	b2Page(t, "Bezique cond 1", reply, ok, "30379-07.htm")
	reply, ok = h.Talk(b2Q403, hero, neti)
	b2Page(t, "Neti cond 1", reply, ok, "30425-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q403, "30425-05.htm", hero, neti)
	b2Page(t, "weapons", reply, ok, "30425-05.htm")
	b2Quest(t, "weapons", hero, b2Q403, scripting.StateStarted, 2)
	b2Items(t, "weapons", hero, map[int]int64{1181: 1, 1182: 1})
	b2Sounds(t, "weapons", h, hero, b2Middle)
	reply, ok = h.Talk(b2Q403, hero, bezique)
	b2Page(t, "Bezique cond 2", reply, ok, "30379-10.htm")
	reply, ok = h.Talk(b2Q403, hero, neti)
	b2Page(t, "Neti cond 2", reply, ok, "30425-06.htm")

	// Without Neti's bow or dagger in the right hand kills draw nothing.
	b2Rand(t, h)
	h.Kill(hero, soldier)
	hero.Equip(7, 1001)
	h.Kill(hero, soldier)
	b2Items(t, "bare hands", hero, map[int]int64{1183: 0})

	// Bones: Spartoi soldiers on rnd(10) < 2, leaders < 3, scouts < 8.
	hero.Equip(7, 1182)
	b2Rand(t, h, rnd.Get(10, 2), rnd.Get(10, 1), rnd.Get(10, 3), rnd.Get(10, 2), rnd.Get(10, 8), rnd.Get(10, 7))
	h.Kill(hero, soldier)
	h.Kill(hero, soldier)
	h.Kill(hero, leader)
	h.Kill(hero, leader)
	h.Kill(hero, scout)
	h.Kill(hero, scout)
	b2Items(t, "three bones", hero, map[int]int64{1183: 3})
	hero.Equip(7, 1181)
	h.World().ClearRecords()
	b2Rand(t, h, b2Q402Draws(7, 10, 0)...)
	b2Q402KillN(t, h, hero, scout, 7)
	b2Quest(t, "ten bones", hero, b2Q403, scripting.StateStarted, 3)
	b2Items(t, "ten bones", hero, map[int]int64{1183: 10})
	b2Sounds(t, "ten bones", h, hero, b2Q402Collect(7, b2Middle)...)
	b2Rand(t, h)
	h.Kill(hero, scout)
	reply, ok = h.Talk(b2Q403, hero, bezique)
	b2Page(t, "Bezique cond 3", reply, ok, "30379-10.htm")

	reply, ok = h.Talk(b2Q403, hero, neti)
	b2Page(t, "horseshoe", reply, ok, "30425-07.htm")
	b2Quest(t, "horseshoe", hero, b2Q403, scripting.StateStarted, 4)
	b2Items(t, "horseshoe", hero, map[int]int64{1183: 0, 1184: 1})
	reply, ok = h.Talk(b2Q403, hero, neti)
	b2Page(t, "Neti cond 4", reply, ok, "30425-08.htm")
	reply, ok = h.Talk(b2Q403, hero, bezique)
	b2Page(t, "wanted list", reply, ok, "30379-08.htm")
	b2Quest(t, "wanted list", hero, b2Q403, scripting.StateStarted, 5)
	b2Items(t, "wanted list", hero, map[int]int64{1184: 0, 1185: 1})
	reply, ok = h.Talk(b2Q403, hero, bezique)
	b2Page(t, "Bezique cond 5", reply, ok, "30379-11.htm")

	// Cats Eye Bandit drops one of the four stolen goods at random; a
	// repeat drops nothing.
	h.World().ClearRecords()
	b2Rand(t, h, rnd.Between(1186, 1189, 1186), rnd.Between(1186, 1189, 1186), rnd.Between(1186, 1189, 1188), rnd.Between(1186, 1189, 1187))
	b2Q402KillN(t, h, hero, bandit, 4)
	b2Items(t, "three goods", hero, map[int]int64{1186: 1, 1187: 1, 1188: 1, 1189: 0})
	b2Sounds(t, "three goods", h, hero, b2ItemGet, b2ItemGet, b2ItemGet)
	b2Quest(t, "three goods", hero, b2Q403, scripting.StateStarted, 5)
	h.World().ClearRecords()
	b2Rand(t, h, rnd.Between(1186, 1189, 1189))
	h.Kill(hero, bandit)
	b2Quest(t, "all goods", hero, b2Q403, scripting.StateStarted, 6)
	b2Items(t, "all goods", hero, map[int]int64{1189: 1})
	b2Sounds(t, "all goods", h, hero, b2Middle)
	b2Rand(t, h)
	h.Kill(hero, bandit)

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q403, hero, bezique)
	b2Page(t, "reward", reply, ok, "30379-09.htm")
	b2NoQuest(t, "reward", hero, b2Q403)
	b2Items(t, "reward", hero, map[int]int64{1190: 1, 1180: 0, 1181: 0, 1182: 0, 1185: 0, 1186: 0, 1187: 0, 1188: 0, 1189: 0})
	b2ExpSp(t, "reward", hero, 3200, 1500)
	b2Sounds(t, "reward", h, hero, b2Finish)
	b2Q402Social(t, "reward", h, hero)
	reply, ok = h.Talk(b2Q403, hero, bezique)
	b2Page(t, "again", reply, ok, "30379-01.htm")
	reply, ok = h.Event(b2Q403, "30379-05.htm", hero, bezique)
	b2Page(t, "again", reply, ok, "30379-04.htm")
}
