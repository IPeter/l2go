package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q220 = "Q00220_TestimonyOfGlory"

// b2Says220 checks what NPCs broadcast since the last ClearRecords.
func b2Says220(t *testing.T, h *scripttest.Harness, what string, want ...string) {
	t.Helper()
	var got []string
	for _, a := range h.World().Actions {
		if a.Kind == "say" {
			got = append(got, a.Text)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("%s: said %q, want %q", what, got, want)
	}
}

// b2SpawnedN220 returns the NPCs spawned since the last ClearRecords, which
// must be n, of the templates ids (in order).
func b2SpawnedN220(t *testing.T, h *scripttest.Harness, ids ...int) []*scripttest.Npc {
	t.Helper()
	sp := h.World().Spawned
	got := make([]int, len(sp))
	for i, n := range sp {
		got[i] = n.ID
	}
	if !slices.Equal(got, ids) {
		t.Fatalf("spawned %v, want %v", got, ids)
	}
	return sp
}

// Testimony of Glory: Vokian's trophies, Chianta's five scepters (Voltar's
// sons, Kepra's Enku overlords, Burai's Makum thugs, Harak, Driko's
// stakato husks), the Timak and Tamlin trophies, Tanapi's scepter of
// Tantos, the Mark of Glory from Kakai.
func TestB2Q220Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q220)
	var (
		vokian  = b2Npc214(30514)
		chianta = b2Npc214(30642)
		kasman  = b2Npc214(30501)
		manakia = b2Npc214(30515)
		voltar  = b2Npc214(30615)
		kepra   = b2Npc214(30616)
		burai   = b2Npc214(30617)
		harak   = b2Npc214(30618)
		driko   = b2Npc214(30619)
		tanapi  = b2Npc214(30571)
		kakai   = b2Npc214(30565)
	)

	b2Refused214(t, h, b2Q220, vokian, &scripttest.Player{ObjectID: 0x10000001, Name: "Human", ClassID: 4, Level: 40}, "30514-01.htm")
	b2Refused214(t, h, b2Q220, vokian, &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 45, Race: 3, Level: 36}, "30514-02.htm")
	b2Refused214(t, h, b2Q220, vokian, &scripttest.Player{ObjectID: 0x10000003, Name: "Novice", ClassID: 44, Race: 3, Level: 40}, "30514-01a.htm")

	w := &b2Walk214{t: t, h: h, q: b2Q220, p: &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", ClassID: 45, Race: 3, Level: 37}}
	w.talk(vokian, "30514-03.htm")
	h.World().ClearRecords()
	w.event(vokian, "30514-05.htm", "30514-05a.htm")
	w.cond("accept", 1)
	b2Sounds(t, "accept", h, w.p, b2Accept)
	w.items("accept", map[int]int64{3204: 1, 7562: 109})
	if got := w.p.Vars["secondClassChange37"]; got != "true" {
		t.Errorf("secondClassChange37 = %q, want true", got)
	}
	w.talk(vokian, "30514-06.htm")

	// Vokian's trophies.
	draws := []rnd.Draw{rnd.Get(10, 5)}
	for range 20 {
		draws = append(draws, rnd.Get(10, 4))
	}
	for range 10 {
		draws = append(draws, rnd.Get(100, 74))
	}
	w.rand(draws...)
	w.kill(b2Npc214(20192), 11)
	w.kill(b2Npc214(20550), 10)
	w.kill(b2Npc214(20563), 9)
	w.cond("one shard short", 1)
	w.kill(b2Npc214(20563), 1)
	w.cond("trophies", 2)
	w.items("trophies", map[int]int64{3205: 10, 3206: 10, 3207: 10})
	w.talk(vokian, "30514-08.htm")
	w.cond("necklace", 3)
	w.items("necklace", map[int]int64{3205: 0, 3206: 0, 3207: 0, 3204: 0, 3209: 1, 3208: 1})
	w.talk(vokian, "30514-09.htm")
	w.talk(chianta, "30642-01.htm")
	w.event(chianta, "30642-03.htm", "30642-03.htm")
	w.cond("Chianta", 4)
	w.items("Chianta", map[int]int64{3208: 0, 3210: 1})
	w.talk(chianta, "30642-04.htm")

	// Kasman's and Manakia's letters.
	w.talk(kasman, "30501-01.htm")
	w.event(kasman, "30501-02.htm", "30501-03.htm")
	w.event(kasman, "30501-02.htm", "30501-04.htm")
	w.event(kasman, "30501-05.htm", "30501-06.htm")
	w.event(kasman, "30501-08.htm", "30501-09.htm")
	w.items("Kasman letters", map[int]int64{3230: 1, 3231: 1, 3232: 1})
	w.talk(manakia, "30515-01.htm")
	w.event(manakia, "30515-02.htm", "30515-03.htm")
	w.event(manakia, "30515-05.htm", "30515-06.htm")
	w.items("Manakia letters", map[int]int64{3228: 1, 3229: 1})

	// Voltar's sons: one group at a time.
	w.talk(voltar, "30615-02.htm")
	h.World().ClearRecords()
	w.event(voltar, "30615-04.htm", "30615-04.htm")
	w.items("glove of Voltar", map[int]int64{3228: 0, 3223: 1})
	sons := b2SpawnedN220(t, h, 27080, 27081)
	h.World().ClearRecords()
	w.talk(voltar, "30615-05.htm")
	b2SpawnedN220(t, h)
	w.kill(sons[0], 1)
	sons[0].Dead = true
	w.items("Pashika", map[int]int64{3221: 1})
	w.kill(sons[1], 1)
	sons[1].Dead = true
	w.items("Vultus", map[int]int64{3222: 1, 3223: 0})
	w.talk(voltar, "30615-06.htm")
	w.items("scepter of Breka", map[int]int64{3221: 0, 3222: 0, 3211: 1})
	w.talk(voltar, "30615-07.htm")

	// Kepra's Enku overlords.
	w.talk(kepra, "30616-02.htm")
	h.World().ClearRecords()
	w.event(kepra, "30616-05.htm", "30616-05.htm")
	w.items("glove of Kepra", map[int]int64{3229: 0, 3225: 1})
	for _, n := range b2SpawnedN220(t, h, 27082, 27082, 27082, 27082) {
		w.kill(n, 1)
		n.Dead = true
	}
	w.items("Enku heads", map[int]int64{3224: 4, 3225: 0})
	w.talk(kepra, "30616-06.htm")
	w.items("scepter of Enku", map[int]int64{3224: 0, 3212: 1})

	// Burai's Makum thugs.
	w.talk(burai, "30617-02.htm")
	h.World().ClearRecords()
	w.event(burai, "30617-04.htm", "30617-04.htm")
	w.items("glove of Burai", map[int]int64{3231: 0, 3227: 1})
	for _, n := range b2SpawnedN220(t, h, 27083, 27083) {
		w.kill(n, 1)
		n.Dead = true
	}
	w.items("Makum heads", map[int]int64{3226: 2, 3227: 0})
	w.talk(burai, "30617-05.htm")
	w.items("scepter of Turek", map[int]int64{3226: 0, 3214: 1})

	// Harak.
	w.talk(harak, "30618-02.htm")
	w.event(harak, "30618-03.htm", "30618-03.htm")
	w.items("scepter of Tunath", map[int]int64{3232: 0, 3215: 1})
	w.cond("four scepters", 4)

	// Driko's stakato drone husks: the fifth scepter.
	w.talk(driko, "30619-02.htm")
	w.event(driko, "30619-03.htm", "30619-03.htm")
	w.items("contract", map[int]int64{3230: 0, 3233: 1})
	w.talk(driko, "30619-04.htm")
	draws = []rnd.Draw{rnd.Get(100, 75)}
	for range 30 {
		draws = append(draws, rnd.Get(100, 74))
	}
	w.rand(draws...)
	w.kill(b2Npc214(20234), 32)
	w.items("husks", map[int]int64{3234: 30})
	w.talk(driko, "30619-05.htm")
	w.cond("five scepters", 5)
	w.items("five scepters", map[int]int64{3233: 0, 3234: 0, 3213: 1})

	// Chianta's third order.
	w.talk(chianta, "30642-05.htm")
	w.event(chianta, "30642-07.htm", "30642-07.htm")
	w.cond("Chianta order 3", 6)
	w.items("Chianta order 3", map[int]int64{3210: 0, 3230: 0, 3228: 0, 3229: 0, 3211: 0, 3212: 0, 3213: 0, 3214: 0, 3215: 0, 3217: 1})
	w.talk(chianta, "30642-10.htm")
	draws = []rnd.Draw{rnd.Get(1000000, 500000)}
	for range 40 {
		draws = append(draws, rnd.Get(1000000, 499999))
	}
	w.rand(draws...)
	w.kill(b2Npc214(20583), 21)
	w.items("Timak heads", map[int]int64{3219: 20})
	w.kill(b2Npc214(20601), 20)
	w.cond("trophies of the orcs", 7)
	w.items("trophies of the orcs", map[int]int64{3218: 20})
	w.talk(chianta, "30642-11.htm")
	w.cond("scepter box", 8)
	w.items("scepter box", map[int]int64{3217: 0, 3209: 0, 3218: 0, 3219: 0, 3220: 1})
	w.talk(chianta, "30642-12.htm")
	w.talk(vokian, "30514-10.htm")
	w.talk(kakai, "30565-01.htm")

	// Tanapi: the Ragna orcs and the revenant of the Tantos chief.
	w.talk(tanapi, "30571-01.htm")
	w.event(tanapi, "30571-03.htm", "30571-03.htm")
	w.cond("Tanapi", 9)
	w.items("Tanapi", map[int]int64{3220: 0, 3235: 1})
	ragna := b2Npc214(20778)
	h.World().ClearRecords()
	h.Attack(w.p, ragna, 100)
	h.Attack(w.p, ragna, 100)
	b2Says220(t, h, "Ragna attacked", "Is it a lackey of Kakai?!")
	h.World().ClearRecords()
	w.kill(ragna, 1)
	b2Says220(t, h, "Ragna killed", "Too late!")
	chief := b2Spawned217(t, h, 27086)
	chief.HP, chief.MaxHP = 1000, 1000
	h.World().ClearRecords()
	h.Attack(w.p, chief, 100)
	h.Attack(w.p, chief, 100)
	b2Says220(t, h, "chief attacked", "How regretful! Unjust dishonor!")
	// Below a third of its HP the chief cries out once.
	h.World().ClearRecords()
	chief.HP = 340
	h.Attack(w.p, chief, 100)
	b2Says220(t, h, "chief at 34 %")
	chief.HP = 320
	h.Attack(w.p, chief, 100)
	h.Attack(w.p, chief, 100)
	b2Says220(t, h, "chief at 32 %", "Indignant and unfair death!")
	h.World().ClearRecords()
	w.kill(chief, 1)
	b2Says220(t, h, "chief killed", "I'll get revenge someday!!")
	w.cond("scepter of Tantos", 10)
	w.items("scepter of Tantos", map[int]int64{3236: 1})
	w.talk(tanapi, "30571-05.htm")
	w.cond("ritual box", 11)
	w.items("ritual box", map[int]int64{3236: 0, 3235: 0, 3237: 1})
	w.talk(tanapi, "30571-06.htm")

	h.World().ClearRecords()
	w.talk(kakai, "30565-02.htm")
	b2Quest(t, "mark", w.p, b2Q220, scripting.StateCompleted, 11)
	w.items("mark", map[int]int64{3237: 0, 3203: 1})
	b2ExpSp(t, "mark", w.p, 91457, 2500)
	b2Sounds(t, "mark", h, w.p, b2Finish)
	w.social("mark")
	reply, ok := h.Talk(b2Q220, w.p, kakai)
	b2Completed(t, "completed", reply, ok)
}
