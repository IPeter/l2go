package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q214 = "Q00214_TrialOfTheScholar"

// b2Walk214 plays one quest for one player; the walkthroughs of quests 214
// to 222 share it.
type b2Walk214 struct {
	t *testing.T
	h *scripttest.Harness
	q string
	p *scripttest.Player
}

// b2Npc214 is an NPC of template id, with an object ID made from it.
func b2Npc214(id int) *scripttest.Npc {
	return &scripttest.Npc{ObjectID: int32(0x60000000 + id), ID: id} //nolint:gosec // template IDs are small
}

// talk talks to npc and expects page ("" for the no-quest answer).
func (w *b2Walk214) talk(npc *scripttest.Npc, page string) {
	w.t.Helper()
	reply, ok := w.h.Talk(w.q, w.p, npc)
	if page == "" {
		b2NoQuestMsg(w.t, "talk", reply, ok)
		return
	}
	b2Page(w.t, "talk to "+page, reply, ok, page)
}

// event clicks event at npc and expects page.
func (w *b2Walk214) event(npc *scripttest.Npc, event, page string) {
	w.t.Helper()
	reply, ok := w.h.Event(w.q, event, w.p, npc)
	b2Page(w.t, "event "+event, reply, ok, page)
}

// cond checks that the quest is started at cond.
func (w *b2Walk214) cond(what string, cond int) {
	w.t.Helper()
	b2Quest(w.t, what, w.p, w.q, scripting.StateStarted, cond)
}

// items checks item counts.
func (w *b2Walk214) items(what string, counts map[int]int64) {
	w.t.Helper()
	b2Items(w.t, what, w.p, counts)
}

// kill kills npc n times.
func (w *b2Walk214) kill(npc *scripttest.Npc, n int) {
	w.t.Helper()
	for range n {
		w.h.Kill(w.p, npc)
	}
}

// rand forces the next draws.
func (w *b2Walk214) rand(draws ...rnd.Draw) {
	w.t.Helper()
	b2Rand(w.t, w.h, draws...)
}

// b2Bools214 is n draws of rnd.bool() returning true.
func b2Bools214(n int) []rnd.Draw {
	out := make([]rnd.Draw, n)
	for i := range out {
		out[i] = rnd.Bool(true)
	}
	return out
}

// social checks that the player made social action 3 (the reward).
func (w *b2Walk214) social(what string) {
	w.t.Helper()
	for _, a := range w.h.World().Actions {
		if a.Kind == "social" && a.Object == w.p.ObjectID && a.Value == 3 {
			return
		}
	}
	w.t.Errorf("%s: no social action 3 in %v", what, w.h.World().Actions)
}

// b2Refused214 checks that a player who may not start the quest gets page at
// npc.
func b2Refused214(t *testing.T, h *scripttest.Harness, quest string, npc *scripttest.Npc, p *scripttest.Player, page string) {
	t.Helper()
	reply, ok := h.Talk(quest, p, npc)
	b2Page(t, p.Name, reply, ok, page)
}

// Trial of the Scholar: Mirien's three sigils, Sylvain and Maria's errands,
// Jurek's hunt, Cronos's scripture chapters, the Mark of Scholar.
func TestB2Q214Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q214)
	var (
		mirien  = b2Npc214(30461)
		sylvain = b2Npc214(30070)
		maria   = b2Npc214(30608)
		lucas   = b2Npc214(30071)
		creta   = b2Npc214(30609)
		jurek   = b2Npc214(30115)
		cronos  = b2Npc214(30610)
		dieter  = b2Npc214(30111)
		edroc   = b2Npc214(30230)
		raut    = b2Npc214(30316)
		triff   = b2Npc214(30611)
		valkon  = b2Npc214(30103)
		poitan  = b2Npc214(30458)
		casian  = b2Npc214(30612)
	)

	b2Refused214(t, h, b2Q214, mirien, &scripttest.Player{ObjectID: 0x10000001, Name: "Mage", ClassID: 10, Level: 40}, "30461-01.htm")
	b2Refused214(t, h, b2Q214, mirien, &scripttest.Player{ObjectID: 0x10000002, Name: "Fighter", ClassID: 1, Level: 40}, "30461-01.htm")
	b2Refused214(t, h, b2Q214, mirien, &scripttest.Player{ObjectID: 0x10000003, Name: "Young", ClassID: 11, Level: 34}, "30461-02.htm")

	// A second class quest done before: no Dimensional Diamonds.
	again := &b2Walk214{t: t, h: h, q: b2Q214, p: &scripttest.Player{ObjectID: 0x10000004, Name: "Again", ClassID: 26, Level: 35, Race: 1,
		Vars: map[string]string{"secondClassChange35": "true"}}}
	again.talk(mirien, "30461-03.htm")
	again.event(mirien, "30461-04.htm", "30461-04.htm")
	again.cond("again", 1)
	again.items("again", map[int]int64{2675: 1, 7562: 0})

	w := &b2Walk214{t: t, h: h, q: b2Q214, p: &scripttest.Player{ObjectID: 0x10000005, Name: "Hero", ClassID: 11, Level: 35}}
	w.talk(mirien, "30461-03.htm")
	h.World().ClearRecords()
	w.event(mirien, "30461-04.htm", "30461-04a.htm")
	w.cond("accept", 1)
	b2Sounds(t, "accept", h, w.p, b2Accept)
	w.items("accept", map[int]int64{2675: 1, 7562: 168})
	if got := w.p.Vars["secondClassChange35"]; got != "true" {
		t.Errorf("secondClassChange35 = %q, want true", got)
	}
	w.talk(mirien, "30461-05.htm")

	// Sylvain sends the hero to Maria, Lucas and Creta.
	w.talk(sylvain, "30070-01.htm")
	w.event(sylvain, "30070-02.htm", "30070-02.htm")
	w.cond("Sylvain", 2)
	w.items("Sylvain", map[int]int64{2689: 1, 2692: 1})
	w.talk(maria, "30608-01.htm")
	w.event(maria, "30608-02.htm", "30608-02.htm")
	w.cond("Maria letter", 3)
	w.items("Maria letter", map[int]int64{2692: 0, 2679: 1})
	w.talk(lucas, "30071-01.htm")
	w.cond("Lucas letter", 4)
	w.items("Lucas letter", map[int]int64{2679: 0, 2681: 1})
	w.talk(maria, "30608-04.htm")
	w.cond("Maria letter 2", 5)
	w.items("Maria letter 2", map[int]int64{2681: 0, 2680: 1})
	w.talk(creta, "30609-01.htm")
	w.event(creta, "30609-05.htm", "30609-05.htm")
	w.cond("Creta letter", 6)
	w.items("Creta letter", map[int]int64{2680: 0, 2683: 1})
	w.talk(maria, "30608-06.htm")
	w.event(maria, "30608-08.htm", "30608-08.htm")
	w.cond("handbag", 7)
	w.items("handbag", map[int]int64{2683: 0, 2682: 1})
	w.talk(creta, "30609-07.htm")
	w.event(creta, "30609-09.htm", "30609-09.htm")
	w.cond("painting 1", 8)
	w.items("painting 1", map[int]int64{2682: 0, 2684: 1})
	w.talk(maria, "30608-10.htm")
	w.cond("painting 2", 9)
	w.items("painting 2", map[int]int64{2684: 0, 2685: 1})
	w.talk(lucas, "30071-03.htm")
	w.event(lucas, "30071-04.htm", "30071-04.htm")
	w.cond("painting 3", 10)
	w.items("painting 3", map[int]int64{2685: 0, 2686: 1})
	w.talk(maria, "30608-12.htm")
	w.cond("scraps", 11)

	// Five brown scroll scraps from the Leto lizardman warriors, half the
	// time.
	leto := b2Npc214(20580)
	w.rand(append([]rnd.Draw{rnd.Bool(false)}, b2Bools214(5)...)...)
	w.kill(leto, 5)
	w.items("4 scraps", map[int]int64{2687: 4})
	w.cond("4 scraps", 11)
	h.World().ClearRecords()
	w.kill(leto, 1)
	w.cond("5 scraps", 12)
	// set_cond(12, true) and the script's own sound, as in Mobius.
	b2Sounds(t, "5 scraps", h, w.p, b2Middle, b2Middle)
	w.talk(maria, "30608-13.htm")
	w.event(maria, "30608-14.htm", "30608-14.htm")
	w.cond("crystal", 13)
	w.items("crystal", map[int]int64{2687: 0, 2686: 0, 2688: 1})
	w.talk(sylvain, "30070-04.htm")
	w.cond("symbol of Sylvain", 14)
	w.items("symbol of Sylvain", map[int]int64{2688: 0, 2689: 0, 2693: 1})
	w.talk(mirien, "30461-06.htm")
	w.cond("sigil 2", 15)
	w.items("sigil 2", map[int]int64{2675: 0, 2693: 0, 2676: 1})

	// Jurek's list.
	w.talk(jurek, "30115-01.htm")
	w.event(jurek, "30115-03.htm", "30115-03.htm")
	w.cond("Jurek", 16)
	w.items("Jurek", map[int]int64{2690: 1, 2694: 1})
	w.rand(b2Bools214(12)...)
	w.kill(b2Npc214(20235), 1)
	w.kill(b2Npc214(20279), 1)
	w.kill(b2Npc214(20068), 5)
	w.kill(b2Npc214(20269), 4)
	w.cond("11 parts", 16)
	w.kill(b2Npc214(20269), 1)
	w.cond("Jurek's list done", 17)
	w.items("Jurek's list done", map[int]int64{2697: 2, 2695: 5, 2696: 5})
	w.talk(jurek, "30115-05.htm")
	w.cond("symbol of Jurek", 18)
	w.items("symbol of Jurek", map[int]int64{2690: 0, 2694: 0, 2697: 0, 2695: 0, 2696: 0, 2698: 1})

	// Level 36 for the third sigil.
	w.talk(mirien, "30461-08.htm")
	w.event(mirien, "30461-09.htm", "30461-09.htm")
	w.items("instruction", map[int]int64{2678: 1})
	w.talk(mirien, "30461-11.htm")
	w.cond("level 35", 18)
	w.p.Level = 36
	w.talk(mirien, "30461-12.htm")
	w.cond("sigil 3", 19)
	w.items("sigil 3", map[int]int64{2678: 0, 2676: 0, 2698: 0, 2677: 1})

	// Cronos's scripture chapters.
	w.talk(cronos, "30610-01.htm")
	w.event(cronos, "30610-10.htm", "30610-10.htm")
	w.cond("Cronos", 20)
	w.items("Cronos", map[int]int64{2699: 1, 2691: 1})
	w.talk(dieter, "30111-01.htm")
	w.event(dieter, "30111-05.htm", "30111-05.htm")
	w.cond("Dieter key", 21)
	w.items("Dieter key", map[int]int64{2699: 0, 2700: 1})
	w.talk(creta, "30609-12.htm")
	w.event(creta, "30609-14.htm", "30609-14.htm")
	w.cond("Creta letter 2", 22)
	w.items("Creta letter 2", map[int]int64{2700: 0, 2701: 1})
	w.talk(dieter, "30111-07.htm")
	w.event(dieter, "30111-09.htm", "30111-09.htm")
	w.cond("diary", 23)
	w.items("diary", map[int]int64{2701: 0, 2703: 1, 2702: 1})
	w.talk(edroc, "30230-01.htm")
	w.event(edroc, "30230-02.htm", "30230-02.htm")
	w.cond("Edroc", 24)
	w.items("Edroc", map[int]int64{2702: 0, 2704: 1})
	w.talk(raut, "30316-01.htm")
	w.event(raut, "30316-02.htm", "30316-02.htm")
	w.cond("chapter 1", 25)
	w.items("chapter 1", map[int]int64{2704: 0, 2706: 1, 2713: 1})
	w.talk(triff, "30611-01.htm")
	w.event(triff, "30611-04.htm", "30611-04.htm")
	w.cond("Triff ring", 26)
	w.items("Triff ring", map[int]int64{2713: 0, 2705: 1})

	// Chapter 2: Valkon's request, Maria's crystal.
	w.talk(valkon, "30103-01.htm")
	w.event(valkon, "30103-04.htm", "30103-04.htm")
	w.items("request", map[int]int64{2710: 1})
	w.talk(valkon, "30103-05.htm")
	w.talk(maria, "30608-18.htm")
	w.items("crystal 2", map[int]int64{2710: 0, 2714: 1})
	w.talk(valkon, "30103-06.htm")
	w.items("chapter 2", map[int]int64{2714: 0, 2707: 1})
	w.talk(valkon, "30103-07.htm")

	// Chapter 3: a Grandis, 30%.
	grandis := b2Npc214(20554)
	w.rand(rnd.Get(100, 30), rnd.Get(100, 29))
	w.kill(grandis, 2)
	w.items("chapter 3", map[int]int64{2708: 1})
	w.rand()
	w.kill(grandis, 1)

	// Chapter 4: Poitan's notes and Casian's list.
	w.talk(poitan, "30458-01.htm")
	w.items("notes", map[int]int64{2711: 1})
	w.talk(poitan, "30458-02.htm")
	w.talk(casian, "30612-02.htm")
	w.event(casian, "30612-04.htm", "30612-04.htm")
	w.cond("Casian", 28)
	w.items("Casian", map[int]int64{2715: 1})
	w.kill(b2Npc214(20201), 10)
	w.kill(b2Npc214(20158), 12)
	w.kill(b2Npc214(20552), 5)
	w.kill(b2Npc214(20567), 4)
	w.cond("one nail short", 28)
	w.kill(b2Npc214(20567), 1)
	w.cond("Casian's list done", 29)
	w.items("Casian's list done", map[int]int64{2716: 10, 2717: 12, 2718: 5, 2719: 5})
	w.talk(casian, "30612-06.htm")
	w.event(casian, "30612-07.htm", "30612-07.htm")
	w.cond("chapter 4", 30)
	w.items("chapter 4", map[int]int64{2715: 0, 2716: 0, 2717: 0, 2718: 0, 2719: 0, 2711: 0, 2709: 1})
	w.talk(cronos, "30610-12.htm")
	w.event(cronos, "30610-14.htm", "30610-14.htm")
	w.cond("symbol of Cronos", 31)
	w.items("symbol of Cronos", map[int]int64{2691: 0, 2703: 0, 2706: 0, 2707: 0, 2708: 0, 2709: 0, 2705: 0, 2720: 1})

	// The mark.
	h.World().ClearRecords()
	w.talk(mirien, "30461-14.htm")
	b2Quest(t, "mark", w.p, b2Q214, scripting.StateCompleted, 31)
	w.items("mark", map[int]int64{2677: 0, 2720: 0, 2674: 1})
	b2ExpSp(t, "mark", w.p, 80265, 30000)
	b2Sounds(t, "mark", h, w.p, b2Finish)
	w.social("mark")
	reply, ok := h.Talk(b2Q214, w.p, mirien)
	b2Completed(t, "completed", reply, ok)
}
