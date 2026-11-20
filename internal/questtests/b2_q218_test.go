package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q218 = "Q00218_TestimonyOfLife"

// Testimony of Life: Asterios's moonflower charm, Pushkin's mithril cup,
// Arkenia and Adonius's stardust, Isael's Talin's spear, the unicorn's
// tears, the water of life, the camomile charm, the Mark of Life.
func TestB2Q218Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q218)
	var (
		cardien  = b2Npc214(30460)
		asterios = b2Npc214(30154)
		thalia   = b2Npc214(30371)
		pushkin  = b2Npc214(30300)
		arkenia  = b2Npc214(30419)
		adonius  = b2Npc214(30375)
		isael    = b2Npc214(30655)
	)

	b2Refused214(t, h, b2Q218, cardien, &scripttest.Player{ObjectID: 0x10000001, Name: "Human", ClassID: 4, Level: 40}, "30460-01.htm")
	b2Refused214(t, h, b2Q218, cardien, &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 19, Race: 1, Level: 36}, "30460-02.htm")
	b2Refused214(t, h, b2Q218, cardien, &scripttest.Player{ObjectID: 0x10000003, Name: "Novice", ClassID: 18, Race: 1, Level: 40}, "30460-02.htm")

	w := &b2Walk214{t: t, h: h, q: b2Q218, p: &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", ClassID: 19, Race: 1, Level: 37}}
	w.talk(cardien, "30460-03.htm")
	h.World().ClearRecords()
	w.event(cardien, "30460-04.htm", "30460-04a.htm")
	w.cond("accept", 1)
	b2Sounds(t, "accept", h, w.p, b2Accept)
	w.items("accept", map[int]int64{3141: 1, 7562: 102})
	if got := w.p.Vars["secondClassChange37"]; got != "true" {
		t.Errorf("secondClassChange37 = %q, want true", got)
	}
	w.talk(cardien, "30460-05.htm")

	w.talk(asterios, "30154-01.htm")
	w.event(asterios, "30154-07.htm", "30154-07.htm")
	w.cond("Asterios", 2)
	w.items("Asterios", map[int]int64{3141: 0, 3143: 1, 3144: 1})
	w.talk(thalia, "30371-01.htm")
	w.event(thalia, "30371-03.htm", "30371-03.htm")
	w.cond("diagram", 3)
	w.items("diagram", map[int]int64{3143: 0, 3145: 1})

	// Pushkin's mithril cup.
	w.talk(pushkin, "30300-01.htm")
	w.event(pushkin, "30300-06.htm", "30300-06.htm")
	w.cond("Pushkin", 4)
	w.items("Pushkin", map[int]int64{3145: 0, 3149: 1})
	draws := b2Bools214(30)
	for range 20 {
		draws = append(draws, rnd.Get(10, 7))
	}
	w.rand(draws...)
	w.kill(b2Npc214(20550), 10)
	w.kill(b2Npc214(20550), 1)
	w.kill(b2Npc214(20176), 20)
	w.kill(b2Npc214(20082), 19)
	w.cond("one acid short", 4)
	w.kill(b2Npc214(20088), 1)
	w.cond("Pushkin's list done", 5)
	w.items("Pushkin's list done", map[int]int64{3161: 10, 3163: 20, 3162: 20})
	w.talk(pushkin, "30300-08.htm")
	w.event(pushkin, "30300-10.htm", "30300-10.htm")
	w.cond("cup", 6)
	w.items("cup", map[int]int64{3149: 0, 3161: 0, 3163: 0, 3162: 0, 3150: 1})
	w.talk(thalia, "30371-06.htm")
	w.cond("Thalia letter", 7)
	w.items("Thalia letter", map[int]int64{3150: 0, 3146: 1})

	// Arkenia and Adonius: the stardust.
	w.talk(arkenia, "30419-01.htm")
	w.event(arkenia, "30419-04.htm", "30419-04.htm")
	w.cond("Arkenia", 8)
	w.items("Arkenia", map[int]int64{3146: 0, 3151: 1, 3152: 1})
	w.talk(adonius, "30375-01.htm")
	w.event(adonius, "30375-02.htm", "30375-02.htm")
	w.cond("Adonius", 9)
	w.items("Adonius", map[int]int64{3152: 0, 3153: 1})
	w.rand(b2Bools214(40)...)
	w.kill(b2Npc214(20233), 20)
	w.kill(b2Npc214(20145), 19)
	w.cond("one down short", 9)
	w.kill(b2Npc214(20145), 1)
	w.cond("Adonius's list done", 10)
	w.talk(adonius, "30375-04.htm")
	w.cond("scripture", 11)
	w.items("scripture", map[int]int64{3153: 0, 3164: 0, 3165: 0, 3154: 1})
	w.talk(arkenia, "30419-06.htm")
	w.cond("stardust", 12)
	w.items("stardust", map[int]int64{3154: 0, 3151: 0, 3155: 1})

	// Thalia wants level 38.
	w.talk(thalia, "30371-09.htm")
	w.event(thalia, "30371-11.htm", "30371-10.htm")
	w.cond("level 37", 13)
	w.items("level 37", map[int]int64{3155: 0, 3148: 1})
	w.talk(thalia, "30371-12.htm")
	w.p.Level = 38
	w.talk(thalia, "30371-13.htm")
	w.cond("level 38", 14)
	w.items("level 38", map[int]int64{3148: 0, 3147: 1})

	// Isael: the six pieces of Talin's spear.
	w.talk(isael, "30655-01.htm")
	w.event(isael, "30655-02.htm", "30655-02.htm")
	w.cond("Isael", 15)
	w.items("Isael", map[int]int64{3147: 0, 3156: 1})
	w.rand(append([]rnd.Draw{rnd.Bool(false)}, b2Bools214(7)...)...)
	w.kill(b2Npc214(20581), 7)
	w.cond("six pieces", 15)
	w.items("six pieces", map[int]int64{3166: 1, 3167: 1, 3168: 1, 3169: 1, 3170: 1, 3171: 1})
	w.kill(b2Npc214(20582), 1)
	w.cond("all pieces", 16)
	w.talk(isael, "30655-04.htm")
	w.cond("spear", 17)
	w.items("spear", map[int]int64{3166: 0, 3171: 0, 3156: 0, 3157: 1, 3026: 1})
	w.talk(thalia, "30371-16.htm")
	w.cond("grail", 18)
	w.items("grail", map[int]int64{3157: 0, 3158: 1})

	// The unicorn falls to Talin's spear only.
	unicorn := b2Npc214(27077)
	w.kill(unicorn, 1)
	w.cond("no spear in hand", 18)
	w.p.Equip(7, 3026)
	w.kill(unicorn, 1)
	w.cond("tears", 19)
	w.items("tears", map[int]int64{3158: 0, 3026: 0, 3159: 1})
	w.talk(thalia, "30371-18.htm")
	w.cond("water of life", 20)
	w.items("water of life", map[int]int64{3159: 0, 3160: 1})
	w.talk(asterios, "30154-09.htm")
	w.cond("camomile", 21)
	w.items("camomile", map[int]int64{3144: 0, 3160: 0, 3142: 1})

	h.World().ClearRecords()
	w.talk(cardien, "30460-07.htm")
	b2Quest(t, "mark", w.p, b2Q218, scripting.StateCompleted, 21)
	w.items("mark", map[int]int64{3142: 0, 3140: 1})
	b2ExpSp(t, "mark", w.p, 104591, 11250)
	b2Sounds(t, "mark", h, w.p, b2Finish)
	w.social("mark")
	reply, ok := h.Talk(b2Q218, w.p, cardien)
	b2Completed(t, "completed", reply, ok)
}
