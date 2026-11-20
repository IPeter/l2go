package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q215 = "Q00215_TrialOfThePilgrim"

// Trial of the Pilgrim: Tanapi and Martankus's flame, Gauri's rumor, Dorf
// and Gerald's book, Petron's statue, Andellia and Uruha's book of darkness,
// Casian's book of sage, the Mark of Pilgrim.
func TestB2Q215Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q215)
	var (
		santiago  = b2Npc214(30648)
		tanapi    = b2Npc214(30571)
		martankus = b2Npc214(30649)
		gauri     = b2Npc214(30550)
		dorf      = b2Npc214(30651)
		gerald    = b2Npc214(30650)
		primos    = b2Npc214(30117)
		petron    = b2Npc214(30036)
		andellia  = b2Npc214(30362)
		uruha     = b2Npc214(30652)
		casian    = b2Npc214(30612)
	)

	b2Refused214(t, h, b2Q215, santiago, &scripttest.Player{ObjectID: 0x10000001, Name: "Wizard", ClassID: 11, Level: 40}, "30648-02.htm")
	b2Refused214(t, h, b2Q215, santiago, &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 15, Level: 34}, "30648-01.htm")

	w := &b2Walk214{t: t, h: h, q: b2Q215, p: &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", ClassID: 50, Race: 3, Level: 35}}
	w.talk(santiago, "30648-03.htm")
	h.World().ClearRecords()
	w.event(santiago, "30648-04.htm", "30648-04a.htm")
	w.cond("accept", 1)
	b2Sounds(t, "accept", h, w.p, b2Accept)
	w.items("accept", map[int]int64{2723: 1, 7562: 49})
	if got := w.p.Vars["secondClassChange35"]; got != "true" {
		t.Errorf("secondClassChange35 = %q, want true", got)
	}
	w.talk(santiago, "30648-09.htm")

	// The spirit of flame.
	w.talk(tanapi, "30571-01.htm")
	w.cond("Tanapi", 2)
	w.items("Tanapi", map[int]int64{2723: 0})
	w.talk(martankus, "30649-01.htm")
	w.cond("Martankus", 3)
	salamander := b2Npc214(27116)
	w.rand(rnd.Get(10, 2), rnd.Get(10, 1))
	w.kill(salamander, 1)
	w.cond("no essence", 3)
	w.kill(salamander, 1)
	w.cond("essence", 4)
	w.items("essence", map[int]int64{2725: 1})
	w.talk(martankus, "30649-03.htm")
	w.event(martankus, "30649-04.htm", "30649-04.htm")
	w.cond("spirit", 5)
	w.items("spirit", map[int]int64{2725: 0, 2724: 1})
	w.talk(tanapi, "30571-03.htm")
	w.cond("Tanapi again", 6)

	// The gray badge, with Gerald's book bought and sold back.
	w.talk(gauri, "30550-01.htm")
	w.cond("rumor", 7)
	w.items("rumor", map[int]int64{2733: 1})
	w.talk(gerald, "30650-01.htm")
	w.event(gerald, "30650-02.htm", "30650-03.htm")
	w.items("no adena", map[int]int64{2726: 0})
	w.p.Give(57, 100000)
	w.event(gerald, "30650-02.htm", "30650-02.htm")
	w.items("book", map[int]int64{57: 0, 2726: 1})
	w.talk(dorf, "30651-02.htm")
	w.cond("badge", 8)
	w.items("badge", map[int]int64{2733: 0, 2727: 1})
	w.talk(dorf, "30651-03.htm")
	w.talk(gerald, "30650-04.htm")
	w.items("book back", map[int]int64{57: 100000, 2726: 0})

	// The statue of Einhasad.
	w.talk(primos, "30117-01.htm")
	w.cond("Primos", 9)
	w.talk(petron, "30036-01.htm")
	w.cond("picture", 10)
	w.items("picture", map[int]int64{2728: 1})
	w.rand(rnd.Get(10, 1))
	w.kill(b2Npc214(27117), 1)
	w.cond("hair", 11)
	w.items("hair", map[int]int64{2729: 1})
	w.talk(petron, "30036-03.htm")
	w.cond("statue", 12)
	w.items("statue", map[int]int64{2728: 0, 2729: 0, 2730: 1})

	// The book of darkness, from level 36.
	w.talk(andellia, "30362-01a.htm")
	w.cond("level 35", 12)
	w.p.Level = 36
	w.talk(andellia, "30362-01.htm")
	w.cond("Andellia", 13)
	w.rand(rnd.Bool(true), rnd.Bool(false), rnd.Get(10, 0))
	w.talk(andellia, "30362-02.htm")
	w.talk(andellia, "30362-02a.htm")
	w.kill(b2Npc214(27118), 1)
	w.cond("debris", 14)
	w.items("debris", map[int]int64{2732: 1})
	w.talk(andellia, "30362-07.htm")
	w.talk(uruha, "30652-01.htm")
	w.event(uruha, "30652-02.htm", "30652-02.htm")
	w.cond("book of darkness", 15)
	w.items("book of darkness", map[int]int64{2732: 0, 2731: 1})
	w.talk(uruha, "30652-03.htm")
	w.talk(andellia, "30362-03.htm")
	w.event(andellia, "30362-04.htm", "30362-04.htm")
	w.cond("kept the book", 16)
	w.items("kept the book", map[int]int64{2731: 1})
	w.talk(andellia, "30362-06.htm")
	w.talk(casian, "30612-01.htm")
	w.cond("book of sage", 17)
	w.items("book of sage", map[int]int64{2731: 0, 2727: 0, 2724: 0, 2730: 0, 2722: 1})
	w.talk(casian, "30612-02.htm")

	h.World().ClearRecords()
	w.talk(santiago, "30648-10.htm")
	b2Quest(t, "mark", w.p, b2Q215, scripting.StateCompleted, 17)
	w.items("mark", map[int]int64{2722: 0, 2721: 1})
	b2ExpSp(t, "mark", w.p, 77382, 16000)
	b2Sounds(t, "mark", h, w.p, b2Finish)
	w.social("mark")
	reply, ok := h.Talk(b2Q215, w.p, santiago)
	b2Completed(t, "completed", reply, ok)
}
