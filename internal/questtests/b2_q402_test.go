package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q402 = "Q00402_PathOfTheHumanKnight"

// b2Q402Draws is n equal rnd(bound) draws of v.
func b2Q402Draws(n, bound, v int) []rnd.Draw {
	return slices.Repeat([]rnd.Draw{rnd.Get(bound, v)}, n)
}

// b2Q402KillN kills npc n times.
func b2Q402KillN(t *testing.T, h *scripttest.Harness, p *scripttest.Player, npc *scripttest.Npc, n int) {
	t.Helper()
	for range n {
		h.Kill(p, npc)
	}
}

// b2Q402Collect is the sounds of n drops: itemget for each but the last,
// then last.
func b2Q402Collect(n int, last string) []string {
	return append(slices.Repeat([]string{b2ItemGet}, n-1), last)
}

// b2Q402Social checks that p, and only p, did social action 3 since the
// last ClearRecords.
func b2Q402Social(t *testing.T, what string, h *scripttest.Harness, p *scripttest.Player) {
	t.Helper()
	want := []scripttest.Action{{Kind: "social", Object: p.ObjectID, Value: 3}}
	if got := h.World().Actions; !slices.Equal(got, want) {
		t.Errorf("%s: actions = %+v, want %+v", what, got, want)
	}
}

// Path to a Human Knight: six coins from six lords, the Sword of Ritual
// from Sir Klaus Vasper.
func TestB2Q402Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q402)
	klaus := &scripttest.Npc{ObjectID: 0x60000001, ID: 30417}
	bathis := &scripttest.Npc{ObjectID: 0x60000002, ID: 30332}
	raymond := &scripttest.Npc{ObjectID: 0x60000003, ID: 30289}
	bezique := &scripttest.Npc{ObjectID: 0x60000004, ID: 30379}
	levian := &scripttest.Npc{ObjectID: 0x60000005, ID: 30037}
	gilbert := &scripttest.Npc{ObjectID: 0x60000006, ID: 30039}
	biotin := &scripttest.Npc{ObjectID: 0x60000007, ID: 30031}
	aaron := &scripttest.Npc{ObjectID: 0x60000008, ID: 30653}
	bugbear := &scripttest.Npc{ObjectID: 0x60000010, ID: 20775}
	priest := &scripttest.Npc{ObjectID: 0x60000011, ID: 27024}
	spider := &scripttest.Npc{ObjectID: 0x60000012, ID: 20043}
	lizardman := &scripttest.Npc{ObjectID: 0x60000013, ID: 20027}
	giantSpider := &scripttest.Npc{ObjectID: 0x60000014, ID: 20103}
	horror := &scripttest.Npc{ObjectID: 0x60000015, ID: 20404}

	// The refusals of 30417-05.htm.
	for i, tt := range []struct {
		name         string
		class, level int
		sword        bool
		page         string
	}{
		{"knight already", 4, 20, false, "30417-02a.htm"},
		{"mage", 10, 20, false, "30417-03.htm"},
		{"too young", 0, 18, false, "30417-02.htm"},
		{"sword", 0, 19, true, "30417-04.htm"},
		{"fits", 0, 19, false, "30417-05.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000100 + int32(i), Name: tt.name, Level: tt.level, ClassID: tt.class}
		if tt.sword {
			p.Give(1161, 1)
		}
		reply, ok := h.Talk(b2Q402, p, klaus)
		b2Page(t, tt.name+" talk", reply, ok, "30417-01.htm")
		reply, ok = h.Event(b2Q402, "30417-05.htm", p, klaus)
		b2Page(t, tt.name, reply, ok, tt.page)
		b2Quest(t, tt.name, p, b2Q402, scripting.StateCreated, 0)
	}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 0}
	reply, ok := h.Talk(b2Q402, hero, klaus)
	b2Page(t, "offer", reply, ok, "30417-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q402, "30417-08.htm", hero, klaus)
	b2Page(t, "accept", reply, ok, "30417-08.htm")
	b2Quest(t, "accept", hero, b2Q402, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{1271: 1})
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q402, hero, klaus)
	b2Page(t, "no coins", reply, ok, "30417-09.htm")
	reply, ok = h.Talk(b2Q402, hero, aaron)
	b2Page(t, "Aaron", reply, ok, "30653-01.htm")

	// Bathis: ten Bugbear Necklaces, no random.
	reply, ok = h.Talk(b2Q402, hero, bathis)
	b2Page(t, "Bathis", reply, ok, "30332-01.htm")
	h.World().ClearRecords()
	h.Event(b2Q402, "30332-02.htm", hero, bathis)
	b2Items(t, "Bathis mark", hero, map[int]int64{1168: 1})
	b2Sounds(t, "Bathis mark", h, hero, b2Middle)
	reply, ok = h.Talk(b2Q402, hero, bathis)
	b2Page(t, "Bathis no necklaces", reply, ok, "30332-03.htm")
	h.World().ClearRecords()
	b2Rand(t, h)
	b2Q402KillN(t, h, hero, bugbear, 11)
	b2Items(t, "necklaces", hero, map[int]int64{1169: 10})
	b2Sounds(t, "necklaces", h, hero, b2Q402Collect(10, b2Middle)...)
	reply, ok = h.Talk(b2Q402, hero, bathis)
	b2Page(t, "Bathis coin", reply, ok, "30332-04.htm")
	b2Items(t, "Bathis coin", hero, map[int]int64{1162: 1, 1168: 0, 1169: 0})
	reply, ok = h.Talk(b2Q402, hero, bathis)
	b2Page(t, "Bathis done", reply, ok, "30332-05.htm")

	// Raymond: twelve Einhasad Crucifixes from Undead Priests, rnd(10) < 5.
	reply, ok = h.Talk(b2Q402, hero, raymond)
	b2Page(t, "Raymond", reply, ok, "30289-01.htm")
	h.Event(b2Q402, "30289-03.htm", hero, raymond)
	b2Items(t, "Raymond mark", hero, map[int]int64{1170: 1})
	reply, ok = h.Talk(b2Q402, hero, raymond)
	b2Page(t, "Raymond no crucifixes", reply, ok, "30289-04.htm")
	b2Rand(t, h, rnd.Get(10, 5))
	h.Kill(hero, priest)
	b2Items(t, "no crucifix", hero, map[int]int64{1171: 0})
	h.World().ClearRecords()
	b2Rand(t, h, b2Q402Draws(12, 10, 4)...)
	b2Q402KillN(t, h, hero, priest, 12)
	b2Items(t, "crucifixes", hero, map[int]int64{1171: 12})
	// The sound turns to middle from the tenth crucifix, as in Mobius.
	b2Sounds(t, "crucifixes", h, hero, append(slices.Repeat([]string{b2ItemGet}, 9), b2Middle, b2Middle, b2Middle)...)
	b2Rand(t, h)
	h.Kill(hero, priest)
	reply, ok = h.Talk(b2Q402, hero, raymond)
	b2Page(t, "Raymond coin", reply, ok, "30289-05.htm")
	b2Items(t, "Raymond coin", hero, map[int]int64{1163: 1, 1170: 0, 1171: 0})
	reply, ok = h.Talk(b2Q402, hero, raymond)
	b2Page(t, "Raymond done", reply, ok, "30289-06.htm")

	// Bezique: twenty Spider Legs, no random.
	reply, ok = h.Talk(b2Q402, hero, bezique)
	b2Page(t, "Bezique", reply, ok, "30379-01.htm")
	h.Event(b2Q402, "30379-02.htm", hero, bezique)
	b2Items(t, "Bezique mark", hero, map[int]int64{1172: 1})
	reply, ok = h.Talk(b2Q402, hero, bezique)
	b2Page(t, "Bezique no legs", reply, ok, "30379-03.htm")
	h.World().ClearRecords()
	b2Q402KillN(t, h, hero, spider, 21)
	b2Items(t, "legs", hero, map[int]int64{1173: 20})
	b2Sounds(t, "legs", h, hero, b2Q402Collect(20, b2Middle)...)
	reply, ok = h.Talk(b2Q402, hero, bezique)
	b2Page(t, "Bezique coin", reply, ok, "30379-04.htm")
	b2Items(t, "Bezique coin", hero, map[int]int64{1164: 1, 1172: 0, 1173: 0})
	reply, ok = h.Talk(b2Q402, hero, bezique)
	b2Page(t, "Bezique done", reply, ok, "30379-05.htm")
	reply, ok = h.Talk(b2Q402, hero, klaus)
	b2Page(t, "three coins", reply, ok, "30417-10.htm")

	// Levian: twenty Lizardman Totems, rnd(10) < 5.
	reply, ok = h.Talk(b2Q402, hero, levian)
	b2Page(t, "Levian", reply, ok, "30037-01.htm")
	h.Event(b2Q402, "30037-02.htm", hero, levian)
	b2Items(t, "Levian mark", hero, map[int]int64{1174: 1})
	reply, ok = h.Talk(b2Q402, hero, levian)
	b2Page(t, "Levian no totems", reply, ok, "30037-03.htm")
	h.World().ClearRecords()
	b2Rand(t, h, append([]rnd.Draw{rnd.Get(10, 5)}, b2Q402Draws(20, 10, 0)...)...)
	b2Q402KillN(t, h, hero, lizardman, 21)
	b2Items(t, "totems", hero, map[int]int64{1175: 20})
	b2Sounds(t, "totems", h, hero, b2Q402Collect(20, b2Middle)...)
	reply, ok = h.Talk(b2Q402, hero, levian)
	b2Page(t, "Levian coin", reply, ok, "30037-04.htm")
	b2Items(t, "Levian coin", hero, map[int]int64{1165: 1, 1174: 0, 1175: 0})
	reply, ok = h.Talk(b2Q402, hero, levian)
	b2Page(t, "Levian done", reply, ok, "30037-05.htm")
	reply, ok = h.Talk(b2Q402, hero, klaus)
	b2Page(t, "four coins", reply, ok, "30417-11.htm")

	// Gilbert: twenty Giant Spider Husks, rnd(10) < 4.
	reply, ok = h.Talk(b2Q402, hero, gilbert)
	b2Page(t, "Gilbert", reply, ok, "30039-01.htm")
	h.Event(b2Q402, "30039-02.htm", hero, gilbert)
	b2Items(t, "Gilbert mark", hero, map[int]int64{1176: 1})
	reply, ok = h.Talk(b2Q402, hero, gilbert)
	b2Page(t, "Gilbert no husks", reply, ok, "30039-03.htm")
	h.World().ClearRecords()
	b2Rand(t, h, append([]rnd.Draw{rnd.Get(10, 4)}, b2Q402Draws(20, 10, 3)...)...)
	b2Q402KillN(t, h, hero, giantSpider, 21)
	b2Items(t, "husks", hero, map[int]int64{1177: 20})
	b2Sounds(t, "husks", h, hero, b2Q402Collect(20, b2Middle)...)
	reply, ok = h.Talk(b2Q402, hero, gilbert)
	b2Page(t, "Gilbert coin", reply, ok, "30039-04.htm")
	b2Items(t, "Gilbert coin", hero, map[int]int64{1166: 1, 1176: 0, 1177: 0})
	reply, ok = h.Talk(b2Q402, hero, gilbert)
	b2Page(t, "Gilbert done", reply, ok, "30039-05.htm")

	// Biotin: ten Horrible Skulls, rnd(10) < 4.
	reply, ok = h.Talk(b2Q402, hero, biotin)
	b2Page(t, "Biotin", reply, ok, "30031-01.htm")
	h.Event(b2Q402, "30031-02.htm", hero, biotin)
	b2Items(t, "Biotin mark", hero, map[int]int64{1178: 1})
	reply, ok = h.Talk(b2Q402, hero, biotin)
	b2Page(t, "Biotin no skulls", reply, ok, "30031-03.htm")
	h.World().ClearRecords()
	b2Rand(t, h, append([]rnd.Draw{rnd.Get(10, 4)}, b2Q402Draws(10, 10, 0)...)...)
	b2Q402KillN(t, h, hero, horror, 11)
	b2Items(t, "skulls", hero, map[int]int64{1179: 10})
	b2Sounds(t, "skulls", h, hero, b2Q402Collect(10, b2Middle)...)
	reply, ok = h.Talk(b2Q402, hero, biotin)
	b2Page(t, "Biotin coin", reply, ok, "30031-04.htm")
	b2Items(t, "Biotin coin", hero, map[int]int64{1167: 1, 1178: 0, 1179: 0})

	// Six coins: the sword at once.
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q402, hero, klaus)
	b2Page(t, "six coins", reply, ok, "30417-12.htm")
	b2NoQuest(t, "six coins", hero, b2Q402)
	b2Items(t, "six coins", hero, map[int]int64{1161: 1, 1271: 0, 1162: 0, 1163: 0, 1164: 0, 1165: 0, 1166: 0, 1167: 0})
	b2ExpSp(t, "six coins", hero, 3200, 7260)
	b2Sounds(t, "six coins", h, hero, b2Finish)
	b2Q402Social(t, "six coins", h, hero)

	// The quest is repeatable, but the sword stops a new start.
	reply, ok = h.Talk(b2Q402, hero, klaus)
	b2Page(t, "again", reply, ok, "30417-01.htm")
	reply, ok = h.Event(b2Q402, "30417-05.htm", hero, klaus)
	b2Page(t, "again", reply, ok, "30417-04.htm")
}

// Sir Klaus Vasper also gives the sword for three to five coins, with less
// SP: 1500 + 1920 for each coin over three.
func TestB2Q402FewerCoins(t *testing.T) {
	h := b2Load(t, b2Q402)
	klaus := &scripttest.Npc{ObjectID: 0x60000001, ID: 30417}
	for i, tt := range []struct {
		coins []int
		page  string
		event string
		sp    int64
	}{
		{[]int{1162, 1164, 1166}, "30417-10.htm", "30417-13.htm", 1500},
		{[]int{1162, 1163, 1165, 1167}, "30417-11.htm", "30417-14.htm", 3420},
		{[]int{1162, 1163, 1164, 1165, 1167}, "30417-11.htm", "30417-14.htm", 5340},
	} {
		p := &scripttest.Player{ObjectID: 0x10000001 + int32(i), Name: "Squire", Level: 19}
		h.Talk(b2Q402, p, klaus)
		h.Event(b2Q402, "30417-08.htm", p, klaus)
		for _, c := range tt.coins {
			p.Give(c, 1)
		}
		reply, ok := h.Talk(b2Q402, p, klaus)
		b2Page(t, tt.page, reply, ok, tt.page)
		b2Quest(t, tt.page, p, b2Q402, scripting.StateStarted, 1)
		h.World().ClearRecords()
		reply, ok = h.Event(b2Q402, tt.event, p, klaus)
		b2Page(t, tt.event, reply, ok, tt.event)
		b2NoQuest(t, tt.event, p, b2Q402)
		b2Items(t, tt.event, p, map[int]int64{1161: 1, 1271: 0, 1162: 0, 1163: 0, 1164: 0, 1165: 0, 1166: 0, 1167: 0})
		b2ExpSp(t, tt.event, p, 3200, tt.sp)
		b2Sounds(t, tt.event, h, p, b2Finish)
		b2Q402Social(t, tt.event, h, p)
	}
}
