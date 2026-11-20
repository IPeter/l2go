package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q217 = "Q00217_TestimonyOfTrust"

// b2Spawned217 returns the one NPC the script spawned since the last
// ClearRecords, which must be of template id.
func b2Spawned217(t *testing.T, h *scripttest.Harness, id int) *scripttest.Npc {
	t.Helper()
	sp := h.World().Spawned
	if len(sp) != 1 || sp[0].ID != id {
		t.Fatalf("spawned %v, want one %d", sp, id)
	}
	return sp[0]
}

// Testimony of Trust: the elves' and dark elves' trust (Asterios's spirits,
// Thifiell and Clayton's hunt), Seresin, the orcs' and dwarves' trust
// (Kakai and Manakia, Lockirin and Nikola), Hollint's recommendation, the
// Mark of Trust from Biotin.
func TestB2Q217Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q217)
	var (
		hollint  = b2Npc214(30191)
		asterios = b2Npc214(30154)
		thifiell = b2Npc214(30358)
		clayton  = b2Npc214(30464)
		seresin  = b2Npc214(30657)
		kakai    = b2Npc214(30565)
		manakia  = b2Npc214(30515)
		lockirin = b2Npc214(30531)
		nikola   = b2Npc214(30621)
		biotin   = b2Npc214(30031)
	)

	b2Refused214(t, h, b2Q217, hollint, &scripttest.Player{ObjectID: 0x10000001, Name: "Fighter", ClassID: 0, Level: 40}, "30191-01a.htm")
	b2Refused214(t, h, b2Q217, hollint, &scripttest.Player{ObjectID: 0x10000002, Name: "Elf", ClassID: 19, Race: 1, Level: 40}, "30191-02.htm")
	b2Refused214(t, h, b2Q217, hollint, &scripttest.Player{ObjectID: 0x10000003, Name: "Young", ClassID: 4, Level: 36}, "30191-01.htm")

	w := &b2Walk214{t: t, h: h, q: b2Q217, p: &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", ClassID: 4, Level: 37}}
	w.talk(hollint, "30191-03.htm")
	h.World().ClearRecords()
	w.event(hollint, "30191-04.htm", "30191-04a.htm")
	w.cond("accept", 1)
	b2Sounds(t, "accept", h, w.p, b2Accept)
	w.items("accept", map[int]int64{2735: 1, 2736: 1, 7562: 96})
	if got := w.p.Vars["secondClassChange37"]; got != "true" {
		t.Errorf("secondClassChange37 = %q, want true", got)
	}
	w.talk(hollint, "30191-08.htm")

	// Asterios: the spirits of the dryads and the lireins.
	w.talk(asterios, "30154-01.htm")
	w.event(asterios, "30154-03.htm", "30154-03.htm")
	w.cond("Asterios", 2)
	w.items("Asterios", map[int]int64{2735: 0, 2745: 1})
	w.rand(rnd.Get(100, 33), rnd.Get(100, 32), rnd.Get(100, 0))
	dryad := b2Npc214(20013)
	w.kill(dryad, 1)
	h.World().ClearRecords()
	w.kill(dryad, 1)
	b2Sounds(t, "Actea", h, w.p, "ItemSound.quest_before_battle")
	actea := b2Spawned217(t, h, 27121)
	w.kill(actea, 1)
	w.items("seed", map[int]int64{2747: 1})
	w.cond("seed", 2)
	h.World().ClearRecords()
	w.kill(b2Npc214(20036), 1)
	luell := b2Spawned217(t, h, 27120)
	w.kill(luell, 1)
	w.items("breath", map[int]int64{2746: 1})
	w.cond("breath", 3)
	w.talk(asterios, "30154-05.htm")
	w.cond("elf trust", 4)
	w.items("elf trust", map[int]int64{2746: 0, 2747: 0, 2745: 0, 2741: 1})

	// Thifiell and Clayton.
	w.talk(thifiell, "30358-01.htm")
	w.event(thifiell, "30358-02.htm", "30358-02.htm")
	w.cond("Thifiell", 5)
	w.items("Thifiell", map[int]int64{2736: 0, 2748: 1})
	w.talk(clayton, "30464-01.htm")
	w.cond("Clayton", 6)
	w.items("Clayton", map[int]int64{2748: 0, 2755: 1})
	w.kill(b2Npc214(20157), 10)
	w.items("ichor", map[int]int64{2751: 0, 2754: 1})
	w.kill(b2Npc214(20082), 10)
	w.items("honey dew", map[int]int64{2750: 0, 2753: 1})
	w.kill(b2Npc214(20550), 9)
	w.cond("one blood short", 6)
	w.kill(b2Npc214(20550), 1)
	w.cond("plasma", 7)
	w.items("plasma", map[int]int64{2749: 0, 2752: 1})
	w.talk(clayton, "30464-03.htm")
	w.cond("Clayton done", 8)
	w.items("Clayton done", map[int]int64{2755: 0})
	w.talk(thifiell, "30358-03.htm")
	w.cond("dark elf trust", 9)
	w.items("dark elf trust", map[int]int64{2752: 0, 2753: 0, 2754: 0, 2740: 1})
	w.talk(hollint, "30191-05.htm")
	w.cond("Seresin letter", 10)
	w.items("Seresin letter", map[int]int64{2740: 0, 2741: 0, 2739: 1})

	// Seresin wants level 38.
	w.talk(seresin, "30657-01.htm")
	w.event(seresin, "30657-03.htm", "30657-02.htm")
	w.cond("level 37", 11)
	w.p.Level = 38
	w.event(seresin, "30657-03.htm", "30657-03.htm")
	w.cond("Seresin", 12)
	w.items("Seresin", map[int]int64{2739: 0, 2737: 1, 2738: 1})

	// Kakai and Manakia: the parasites of Lota.
	w.talk(kakai, "30565-01.htm")
	w.event(kakai, "30565-02.htm", "30565-02.htm")
	w.cond("Kakai", 13)
	w.items("Kakai", map[int]int64{2738: 0, 2757: 1})
	w.talk(manakia, "30515-01.htm")
	w.event(manakia, "30515-02.htm", "30515-02.htm")
	w.cond("Manakia", 14)
	w.items("Manakia", map[int]int64{2757: 0})
	w.rand(append([]rnd.Draw{rnd.Bool(false)}, b2Bools214(10)...)...)
	w.kill(b2Npc214(20553), 10)
	w.cond("9 parasites", 14)
	w.kill(b2Npc214(20553), 1)
	w.cond("parasites", 15)
	w.items("parasites", map[int]int64{2756: 10})
	w.talk(manakia, "30515-04.htm")
	w.cond("Manakia letter", 16)
	w.items("Manakia letter", map[int]int64{2756: 0, 2758: 1})
	w.talk(kakai, "30565-04.htm")
	w.cond("orc trust", 17)
	w.items("orc trust", map[int]int64{2758: 0, 2743: 1})

	// Lockirin and Nikola: the heartstones of Porta.
	w.talk(lockirin, "30531-01.htm")
	w.event(lockirin, "30531-02.htm", "30531-02.htm")
	w.cond("Lockirin", 18)
	w.items("Lockirin", map[int]int64{2737: 0, 2759: 1})
	w.talk(nikola, "30621-01.htm")
	w.event(nikola, "30621-02.htm", "30621-02.htm")
	w.cond("Nikola", 19)
	w.items("Nikola", map[int]int64{2759: 0, 2760: 1})
	w.kill(b2Npc214(20213), 10)
	w.cond("heartstones", 20)
	w.talk(nikola, "30621-04.htm")
	w.cond("Nikola done", 21)
	w.items("Nikola done", map[int]int64{2761: 0, 2760: 0})
	w.talk(lockirin, "30531-04.htm")
	w.cond("dwarf trust", 22)
	w.items("dwarf trust", map[int]int64{2742: 1})
	w.talk(hollint, "30191-06.htm")
	w.cond("recommendation", 23)
	w.items("recommendation", map[int]int64{2742: 0, 2743: 0, 2744: 1})
	w.talk(hollint, "30191-07.htm")

	h.World().ClearRecords()
	w.talk(biotin, "30031-01.htm")
	b2Quest(t, "mark", w.p, b2Q217, scripting.StateCompleted, 23)
	w.items("mark", map[int]int64{2744: 0, 2734: 1})
	b2ExpSp(t, "mark", w.p, 39571, 2500)
	b2Sounds(t, "mark", h, w.p, b2Finish)
	w.social("mark")
	reply, ok := h.Talk(b2Q217, w.p, biotin)
	b2Completed(t, "completed", reply, ok)
}
