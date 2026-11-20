package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q219 = "Q00219_TestimonyOfFate"

// Testimony of Fate: Metheus's funeral jar, Ixia's belladonna, the Alder
// spirit's skull, Roa and Norman's manuscript, Kaira's recommendation,
// Arkenia's fairy dust and treant sap, the Mark of Fate from Thifiell.
func TestB2Q219Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q219)
	var (
		kaira    = b2Npc214(30476)
		metheus  = b2Npc214(30614)
		ixia     = b2Npc214(30463)
		roa      = b2Npc214(30114)
		norman   = b2Npc214(30210)
		thifiell = b2Npc214(30358)
		arkenia  = b2Npc214(30419)
		pixy     = b2Npc214(31845)
		treant   = b2Npc214(31850)
	)

	b2Refused214(t, h, b2Q219, kaira, &scripttest.Player{ObjectID: 0x10000001, Name: "Human", ClassID: 4, Level: 40}, "30476-02.htm")
	b2Refused214(t, h, b2Q219, kaira, &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 32, Race: 2, Level: 36}, "30476-01.htm")
	b2Refused214(t, h, b2Q219, kaira, &scripttest.Player{ObjectID: 0x10000003, Name: "Novice", ClassID: 31, Race: 2, Level: 40}, "30476-01.htm")

	w := &b2Walk214{t: t, h: h, q: b2Q219, p: &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", ClassID: 32, Race: 2, Level: 37}}
	w.talk(kaira, "30476-03.htm")
	h.World().ClearRecords()
	w.event(kaira, "30476-05.htm", "30476-05a.htm")
	w.cond("accept", 1)
	b2Sounds(t, "accept", h, w.p, b2Accept)
	w.items("accept", map[int]int64{3173: 1, 7562: 98})
	if got := w.p.Vars["secondClassChange37"]; got != "true" {
		t.Errorf("secondClassChange37 = %q, want true", got)
	}
	w.talk(kaira, "30476-06.htm")

	// Metheus's funeral jar and Ixia's belladonna.
	w.talk(metheus, "30614-01.htm")
	w.cond("Metheus", 2)
	w.items("Metheus", map[int]int64{3173: 0, 3174: 1})
	w.kill(b2Npc214(20144), 1)
	w.cond("remains", 3)
	w.items("remains", map[int]int64{3174: 0, 3175: 1})
	w.talk(metheus, "30614-03.htm")
	w.cond("textbook", 5)
	w.items("textbook", map[int]int64{3175: 0, 3176: 1})
	w.talk(ixia, "30463-01.htm")
	w.cond("Ixia", 6)
	w.items("Ixia", map[int]int64{3176: 0, 3177: 1})
	draws := []rnd.Draw{rnd.Get(1000000, 500000)}
	for range 10 {
		draws = append(draws, rnd.Get(1000000, 499999))
	}
	for range 10 {
		draws = append(draws, rnd.Get(1000000, 599999))
	}
	for range 30 {
		draws = append(draws, rnd.Get(1000000, 299999))
	}
	w.rand(draws...)
	w.kill(b2Npc214(20202), 11)
	w.items("dung", map[int]int64{3180: 10})
	w.kill(b2Npc214(20193), 10)
	w.kill(b2Npc214(20158), 10)
	w.kill(b2Npc214(20230), 10)
	w.kill(b2Npc214(20233), 9)
	w.cond("one fluid short", 6)
	w.kill(b2Npc214(20233), 1)
	w.cond("Ixia's list done", 7)
	w.items("Ixia's list done", map[int]int64{3181: 10, 3178: 10, 3182: 10, 3179: 10})
	w.talk(ixia, "30463-03.htm")
	w.cond("belladonna", 8)
	w.items("belladonna", map[int]int64{3177: 0, 3180: 0, 3179: 0, 3178: 0, 3182: 0, 3181: 0, 3183: 1})
	w.talk(metheus, "30614-05.htm")
	w.cond("skull", 9)
	w.items("skull", map[int]int64{3183: 0, 3184: 1})

	// The Alder spirit appears at Kaira's.
	h.World().ClearRecords()
	w.talk(kaira, "30476-09.htm")
	w.cond("Alder spirit", 10)
	w.items("Alder spirit", map[int]int64{3184: 0})
	spirit := b2Spawned217(t, h, 30613)
	w.talk(spirit, "30613-01.htm")
	w.cond("skull 2", 11)
	w.items("skull 2", map[int]int64{3185: 1})
	if !spirit.Deleted {
		t.Error("the Alder spirit was not deleted")
	}

	// Roa and Norman.
	w.talk(roa, "30114-01.htm")
	w.event(roa, "30114-04.htm", "30114-04.htm")
	w.cond("receipt", 12)
	w.items("receipt", map[int]int64{3185: 0, 3186: 1})
	w.talk(norman, "30210-01.htm")
	w.cond("manuscript", 13)
	w.items("manuscript", map[int]int64{3186: 0, 3187: 1})

	// Kaira wants level 38.
	w.talk(kaira, "30476-11.htm")
	w.event(kaira, "30476-12.htm", "30476-13.htm")
	w.cond("level 37", 14)
	w.items("level 37", map[int]int64{3188: 1})
	w.talk(kaira, "30476-14.htm")
	w.p.Level = 38
	w.talk(kaira, "30476-12.htm")
	w.cond("recommendation", 15)
	w.items("recommendation", map[int]int64{3188: 0, 3187: 0, 3189: 1})
	w.talk(thifiell, "30358-01.htm")
	w.cond("Thifiell", 16)
	w.items("Thifiell", map[int]int64{3189: 0, 3190: 1, 3191: 1})
	w.talk(arkenia, "30419-01.htm")
	w.event(arkenia, "30419-02.htm", "30419-02.htm")
	w.cond("Arkenia", 17)
	w.items("Arkenia", map[int]int64{3191: 0, 3192: 1})
	w.talk(arkenia, "30419-03.htm")

	// The pixy's red fairy dust.
	w.talk(pixy, "31845-01.htm")
	w.event(pixy, "31845-02.htm", "31845-02.htm")
	w.items("garnet", map[int]int64{3193: 1})
	w.talk(pixy, "31845-03.htm")
	w.kill(b2Npc214(20554), 11)
	w.kill(b2Npc214(20582), 10)
	w.kill(b2Npc214(20270), 10)
	w.kill(b2Npc214(20600), 10)
	w.items("skulls", map[int]int64{3194: 10, 3197: 10, 3196: 10, 3195: 10})
	w.talk(pixy, "31845-04.htm")
	w.items("fairy dust", map[int]int64{3194: 0, 3197: 0, 3196: 0, 3195: 0, 3193: 0, 3198: 1})
	w.talk(pixy, "31845-05.htm")

	// The blight treant's sap.
	w.talk(treant, "31850-01.htm")
	w.event(treant, "31850-02.htm", "31850-02.htm")
	w.items("seed", map[int]int64{3199: 1})
	w.talk(treant, "31850-03.htm")
	w.kill(b2Npc214(27079), 2)
	w.items("leaf", map[int]int64{3200: 1})
	w.talk(treant, "31850-04.htm")
	w.items("sap", map[int]int64{3200: 0, 3199: 0, 3201: 1})
	w.talk(treant, "31850-05.htm")

	w.talk(arkenia, "30419-04.htm")
	w.event(arkenia, "30419-05.htm", "30419-05.htm")
	w.cond("Arkenia letter", 18)
	w.items("Arkenia letter", map[int]int64{3192: 0, 3201: 0, 3198: 0, 3202: 1})
	w.talk(arkenia, "30419-06.htm")

	h.World().ClearRecords()
	w.talk(thifiell, "30358-04.htm")
	b2Quest(t, "mark", w.p, b2Q219, scripting.StateCompleted, 18)
	w.items("mark", map[int]int64{3190: 0, 3202: 0, 3172: 1})
	b2ExpSp(t, "mark", w.p, 68183, 1750)
	b2Sounds(t, "mark", h, w.p, b2Finish)
	w.social("mark")
	reply, ok := h.Talk(b2Q219, w.p, thifiell)
	b2Completed(t, "completed", reply, ok)
}
