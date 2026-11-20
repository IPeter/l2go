package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q412 = "Q00412_PathOfTheDarkWizard"

// b2Q412Social checks that p made the social action (and only that one)
// since the last ClearRecords. The other walkthroughs of Q412-Q418 and
// Q211-Q213 use it too.
func b2Q412Social(t *testing.T, what string, h *scripttest.Harness, p *scripttest.Player, action int) {
	t.Helper()
	var got []int
	for _, a := range h.World().Actions {
		if a.Kind == "social" && a.Object == p.ObjectID {
			got = append(got, a.Value)
		}
	}
	if len(got) != 1 || got[0] != action {
		t.Errorf("%s: social actions %v, want [%d]", what, got, action)
	}
}

// Path to a Dark Wizard: Varika, then the three seeds from Charkeren,
// Annika and Arkenia, and the Jewel of Darkness.
func TestB2Q412Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q412)
	varika := &scripttest.Npc{ObjectID: 0x60000001, ID: 30421}
	charkeren := &scripttest.Npc{ObjectID: 0x60000002, ID: 30415}
	annika := &scripttest.Npc{ObjectID: 0x60000003, ID: 30418}
	arkenia := &scripttest.Npc{ObjectID: 0x60000004, ID: 30419}
	skeleton := &scripttest.Npc{ObjectID: 0x60000005, ID: 20015}
	zombie := &scripttest.Npc{ObjectID: 0x60000006, ID: 20022}
	zombie2 := &scripttest.Npc{ObjectID: 0x60000007, ID: 20517}
	marsh := &scripttest.Npc{ObjectID: 0x60000008, ID: 20045}

	// Refusals: the wrong class, a dark wizard already, too low, the jewel.
	for _, c := range []struct {
		what  string
		class int
		level int
		jewel bool
		page  string
	}{
		{"fighter", 31, 19, false, "30421-03.htm"},
		{"dark wizard", 39, 20, false, "30421-02a.htm"},
		{"level 18", 38, 18, false, "30421-02.htm"},
		{"jewel", 38, 19, true, "30421-04.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000010, Name: "No", Level: c.level, ClassID: c.class, Race: 2}
		if c.jewel {
			p.Give(1261, 1)
		}
		reply, ok := h.Talk(b2Q412, p, varika)
		b2Page(t, c.what+" offer", reply, ok, "30421-01.htm")
		reply, ok = h.Event(b2Q412, "30421-05.htm", p, varika)
		b2Page(t, c.what, reply, ok, c.page)
		b2Quest(t, c.what, p, b2Q412, scripting.StateCreated, 0)
	}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 38, Race: 2}
	reply, ok := h.Talk(b2Q412, hero, varika)
	b2Page(t, "offer", reply, ok, "30421-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q412, "30421-05.htm", hero, varika)
	b2Page(t, "accept", reply, ok, "30421-05.htm")
	b2Quest(t, "accept", hero, b2Q412, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{1254: 1})
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Event(b2Q412, "30421-07.htm", hero, varika)
	b2Page(t, "ask for anger", reply, ok, "30421-07.htm")

	// Seed of Anger: Charkeren's key, three Family Remains.
	reply, ok = h.Talk(b2Q412, hero, charkeren)
	b2Page(t, "Charkeren", reply, ok, "30415-01.htm")
	h.Event(b2Q412, "30415-03.htm", hero, charkeren)
	b2Items(t, "key", hero, map[int]int64{1277: 1})
	reply, ok = h.Event(b2Q412, "30421-07.htm", hero, varika)
	b2Page(t, "Varika key", reply, ok, "30421-08.htm")
	reply, ok = h.Talk(b2Q412, hero, charkeren)
	b2Page(t, "Charkeren no remains", reply, ok, "30415-04.htm")
	h.World().ClearRecords()
	b2Rand(t, h, rnd.Get(10, 5), rnd.Get(10, 4), rnd.Get(10, 0), rnd.Get(10, 4))
	for range 4 {
		h.Kill(hero, skeleton)
	}
	b2Items(t, "remains", hero, map[int]int64{1257: 3})
	b2Sounds(t, "remains", h, hero, b2ItemGet, b2ItemGet, b2Middle)
	b2Rand(t, h)
	h.Kill(hero, skeleton)
	b2Items(t, "remains full", hero, map[int]int64{1257: 3})
	reply, ok = h.Talk(b2Q412, hero, charkeren)
	b2Page(t, "anger", reply, ok, "30415-05.htm")
	b2Items(t, "anger", hero, map[int]int64{1257: 0, 1277: 0, 1253: 1})
	reply, ok = h.Talk(b2Q412, hero, charkeren)
	b2Page(t, "Charkeren done", reply, ok, "30415-06.htm")

	// Seed of Horror: Annika's candle, two knee bones.
	reply, ok = h.Talk(b2Q412, hero, annika)
	b2Page(t, "Annika", reply, ok, "30418-01.htm")
	h.Event(b2Q412, "30418-02.htm", hero, annika)
	b2Items(t, "candle", hero, map[int]int64{1278: 1})
	reply, ok = h.Talk(b2Q412, hero, annika)
	b2Page(t, "Annika no bones", reply, ok, "30418-03.htm")
	b2Rand(t, h, rnd.Get(10, 4), rnd.Get(10, 9), rnd.Get(10, 1))
	h.Kill(hero, zombie)
	h.Kill(hero, zombie2)
	h.Kill(hero, zombie2)
	b2Items(t, "bones", hero, map[int]int64{1259: 2})
	reply, ok = h.Event(b2Q412, "30421-10.htm", hero, varika)
	b2Page(t, "Varika bones", reply, ok, "30421-19.htm")
	reply, ok = h.Talk(b2Q412, hero, annika)
	b2Page(t, "horror", reply, ok, "30418-04.htm")
	b2Items(t, "horror", hero, map[int]int64{1259: 0, 1278: 0, 1255: 1})

	// Seed of Lunacy: Arkenia's scent, three hearts.
	reply, ok = h.Talk(b2Q412, hero, varika)
	b2Page(t, "Varika two seeds", reply, ok, "30421-17.htm")
	reply, ok = h.Talk(b2Q412, hero, arkenia)
	b2Page(t, "Arkenia", reply, ok, "30419-01.htm")
	b2Items(t, "scent", hero, map[int]int64{1279: 1})
	reply, ok = h.Talk(b2Q412, hero, arkenia)
	b2Page(t, "Arkenia no hearts", reply, ok, "30419-02.htm")
	b2Rand(t, h, rnd.Get(10, 0), rnd.Get(10, 0), rnd.Get(10, 0))
	for range 3 {
		h.Kill(hero, marsh)
	}
	reply, ok = h.Talk(b2Q412, hero, arkenia)
	b2Page(t, "lunacy", reply, ok, "30419-03.htm")
	b2Items(t, "lunacy", hero, map[int]int64{1260: 0, 1279: 0, 1256: 1})
	reply, ok = h.Event(b2Q412, "30421-13.htm", hero, varika)
	b2Page(t, "Varika lunacy", reply, ok, "30421-12.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q412, hero, varika)
	b2Page(t, "jewel", reply, ok, "30421-16.htm")
	b2Items(t, "jewel", hero, map[int]int64{1253: 0, 1254: 0, 1255: 0, 1256: 0, 1261: 1})
	b2ExpSp(t, "jewel", hero, 3200, 1650)
	b2Q412Social(t, "jewel", h, hero, 3)
	b2Sounds(t, "jewel", h, hero, b2Finish)
	b2NoQuest(t, "jewel", hero, b2Q412)
}
