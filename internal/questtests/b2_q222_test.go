package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q222 = "Q00222_TestOfTheDuelist"

// Test of the Duelist: Kaien's five orders (ten trophies of ten monsters),
// the final order (three trophies of five more), the Mark of Duelist.
func TestB2Q222Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q222)
	kaien := b2Npc214(30623)

	b2Refused214(t, h, b2Q222, kaien, &scripttest.Player{ObjectID: 0x10000001, Name: "Knight", ClassID: 4, Level: 40}, "30623-02.htm")
	b2Refused214(t, h, b2Q222, kaien, &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 1, Level: 38}, "30623-01.htm")

	w := &b2Walk214{t: t, h: h, q: b2Q222, p: &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", ClassID: 47, Race: 3, Level: 39}}
	w.talk(kaien, "30623-03.htm")
	w.event(kaien, "30623-04.htm", "30623-05.htm")
	h.World().ClearRecords()
	w.event(kaien, "30623-07.htm", "30623-07a.htm")
	w.cond("accept", 2)
	b2Sounds(t, "accept", h, w.p, b2Accept)
	w.items("accept", map[int]int64{2763: 1, 2764: 1, 2765: 1, 2766: 1, 2767: 1, 7562: 72})
	if got := w.p.Vars["secondClassChange39"]; got != "true" {
		t.Errorf("secondClassChange39 = %q, want true", got)
	}
	w.talk(kaien, "30623-07a.htm")
	w.event(kaien, "30623-16.htm", "30623-16.htm")
	w.cond("too early", 2)
	w.items("too early", map[int]int64{2778: 0})

	// Ten trophies each.
	first := []int{20085, 20090, 20234, 20202, 20270, 20552, 20582, 20564, 20601, 20602}
	for _, id := range first[:9] {
		w.kill(b2Npc214(id), 11)
	}
	w.kill(b2Npc214(20602), 9)
	w.cond("one arrow short", 2)
	w.kill(b2Npc214(20602), 1)
	w.cond("orders done", 3)
	w.items("orders done", map[int]int64{2768: 10, 2769: 10, 2770: 10, 2771: 10, 2772: 10, 2773: 10, 2774: 10, 2775: 10, 2776: 10, 2777: 10})
	w.talk(kaien, "30623-13.htm")
	w.event(kaien, "30623-16.htm", "30623-16.htm")
	w.cond("final order", 4)
	w.items("final order", map[int]int64{2763: 0, 2764: 0, 2765: 0, 2766: 0, 2767: 0,
		2768: 0, 2769: 0, 2770: 0, 2771: 0, 2772: 0, 2773: 0, 2774: 0, 2775: 0, 2776: 0, 2777: 0, 2778: 1})
	w.talk(kaien, "30623-17.htm")

	// Three trophies each.
	w.kill(b2Npc214(20214), 4)
	w.kill(b2Npc214(20217), 3)
	w.kill(b2Npc214(20554), 3)
	w.kill(b2Npc214(20588), 3)
	w.kill(b2Npc214(20604), 2)
	w.cond("one mace short", 4)
	w.kill(b2Npc214(20604), 1)
	w.cond("final order done", 5)
	w.items("final order done", map[int]int64{2779: 3, 2780: 3, 2781: 3, 2782: 3, 2783: 3})

	h.World().ClearRecords()
	w.talk(kaien, "30623-18.htm")
	b2Quest(t, "mark", w.p, b2Q222, scripting.StateCompleted, 5)
	w.items("mark", map[int]int64{2778: 0, 2779: 0, 2780: 0, 2781: 0, 2782: 0, 2783: 0, 2762: 1})
	b2ExpSp(t, "mark", w.p, 47015, 20000)
	b2Sounds(t, "mark", h, w.p, b2Finish)
	w.social("mark")
	reply, ok := h.Talk(b2Q222, w.p, kaien)
	b2Completed(t, "completed", reply, ok)
}
