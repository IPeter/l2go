package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The noblesse quests of M11e on an active subclass (Possessor of a
// Precious Soul 1-4), the chain of letters between them, and Q247's
// noblesse. Every name starts with m11e: other batches share the package.

const (
	m11eQ241 = "Q00241_PossessorOfAPreciousSoul1"
	m11eQ242 = "Q00242_PossessorOfAPreciousSoul2"
	m11eQ246 = "Q00246_PossessorOfAPreciousSoul3"
	m11eQ247 = "Q00247_PossessorOfAPreciousSoul4"

	// The letters that chain the quests.
	m11eVirgilsLetter    = 7677 // Q241's reward, Q242's start
	m11eCaradineLetter2  = 7678 // Q242's reward, Q246's start
	m11eCaradineLetter3  = 7679 // Q246's reward, Q247's start
	m11eNoblessTiara     = 7694
	m11eSubclassFirst    = 1
	m11eFatesWhisperName = "Q00234_FatesWhisper"
)

// m11eOnSubclass is a player on its first subclass at a level.
func m11eOnSubclass(id int32, level int) *scripttest.Player {
	return &scripttest.Player{ObjectID: id, Name: "Subclass", Level: level, ClassID: 12, BaseClass: 2, ClassIndex: m11eSubclassFirst, SubClasses: 1}
}

func TestM11eQ241OnSubclass(t *testing.T) {
	q := b3aStart(t, m11eQ241)
	b3cRequires(t, q.h, q.name, "subclass")
	npc := func(id int32, npcID int) *scripttest.Npc {
		return &scripttest.Npc{ObjectID: 0x60000000 + id, ID: npcID, Name: "npc"}
	}
	talien, gabrielle, gilmore, kantabilon := npc(1, 31739), npc(2, 30753), npc(3, 30754), npc(4, 31042)
	stedmiel, virgil, ogmar, rahorakti := npc(5, 30692), npc(6, 31742), npc(7, 31744), npc(8, 31336)
	kassandra, caradine, noel := npc(9, 31743), npc(10, 31740), npc(11, 31272)
	baraham, succubus, stakato := npc(20, 27113), npc(21, 20244), npc(22, 21508)

	// Below level 50 Talien refuses even on a subclass.
	young := m11eOnSubclass(0x10000002, 49)
	q.talk(young, talien, "31739-02.htm")

	p := m11eOnSubclass(0x10000001, 50)
	q.talk(p, talien, "31739-01.htm")
	q.event(p, talien, "31739-03.htm", "31739-03.htm")
	q.want("accepted", p, scripting.StateStarted, 1)
	q.talk(p, talien, "31739-04.htm")
	q.talk(p, gabrielle, "30753-01.htm")
	q.event(p, gabrielle, "30753-02.htm", "30753-02.htm")
	q.talk(p, gilmore, "30754-01.htm")
	q.event(p, gilmore, "30754-02.htm", "30754-02.htm")
	q.want("Gilmore", p, scripting.StateStarted, 3)

	// Back on the base class nobody talks and kills count for nothing.
	p.ClassIndex = 0
	q.talk(p, talien, b3aNoQuest)
	q.kill(p, baraham)
	q.want("on the base class", p, scripting.StateStarted, 3)
	p.ClassIndex = m11eSubclassFirst

	q.kill(p, baraham)
	q.want("Baraham", p, scripting.StateStarted, 4)
	b3aWantItems(t, "Baraham", p, map[int]int64{7587: 1})
	q.talk(p, talien, "31739-06.htm")
	q.event(p, talien, "31739-07.htm", "31739-07.htm")
	b3aWantItems(t, "legend", p, map[int]int64{7587: 0})
	q.talk(p, kantabilon, "31042-01.htm")
	q.event(p, kantabilon, "31042-02.htm", "31042-02.htm")

	// Malruk succubi: 60 >= rnd(100) gives a claw, ten of them cond 7.
	q.kill(p, succubus, rnd.Get(100, 61))
	b3aWantItems(t, "a missed claw", p, map[int]int64{7597: 0})
	for range 10 {
		q.kill(p, succubus, rnd.Get(100, 60))
	}
	q.want("claws", p, scripting.StateStarted, 7)
	q.kill(p, succubus) // cond 7: no member at cond 6, no draw
	b3aWantItems(t, "claws", p, map[int]int64{7597: 10})
	q.talk(p, kantabilon, "31042-04.htm")
	q.event(p, kantabilon, "31042-05.htm", "31042-05.htm")
	b3aWantItems(t, "echo crystal", p, map[int]int64{7597: 0, 7589: 1})
	q.talk(p, talien, "31739-09.htm")
	q.event(p, talien, "31739-10.htm", "31739-10.htm")
	q.talk(p, stedmiel, "30692-01.htm")
	q.event(p, stedmiel, "30692-02.htm", "30692-02.htm")
	b3aWantItems(t, "poetry book", p, map[int]int64{7589: 0, 7588: 1})
	q.talk(p, talien, "31739-12.htm")
	q.event(p, talien, "31739-13.htm", "31739-13.htm")
	q.talk(p, virgil, "31742-01.htm")
	q.event(p, virgil, "31742-02.htm", "31742-02.htm")
	q.talk(p, ogmar, "31744-01.htm")
	q.event(p, ogmar, "31744-02.htm", "31744-02.htm")
	q.talk(p, rahorakti, "31336-01.htm")
	q.event(p, rahorakti, "31336-02.htm", "31336-02.htm")

	// Splinter stakatos: 30 >= rnd(100) gives crimson moss, five cond 15.
	for range 5 {
		q.kill(p, stakato, rnd.Get(100, 30))
	}
	q.want("moss", p, scripting.StateStarted, 15)
	q.talk(p, rahorakti, "31336-04.htm")
	q.event(p, rahorakti, "31336-05.htm", "31336-05.htm")
	b3aWantItems(t, "medicine", p, map[int]int64{7598: 0, 7599: 1})
	q.talk(p, kassandra, "31743-01.htm")
	q.event(p, kassandra, "31743-02.htm", "31743-02.htm")
	q.talk(p, virgil, "31742-04.htm")
	q.event(p, virgil, "31742-05.htm", "31742-05.htm")
	q.talk(p, caradine, "31740-01.htm")
	q.event(p, caradine, "31740-02.htm", "31740-02.htm")
	q.talk(p, noel, "31272-01.htm")
	q.event(p, noel, "31272-02.htm", "31272-02.htm")
	q.talk(p, noel, "31272-03.htm")
	p.Give(6033, 1)
	p.Give(6029, 5)
	q.talk(p, noel, "31272-04.htm")
	q.event(p, noel, "31272-05.htm", "31272-05.htm")
	q.want("Noel", p, scripting.StateStarted, 21)
	q.talk(p, caradine, "31740-04.htm")
	q.event(p, caradine, "31740-05.htm", "31740-05.htm")
	q.want("done", p, scripting.StateCompleted, 21)
	b3aWantItems(t, "reward", p, map[int]int64{m11eVirgilsLetter: 1, 6033: 0, 6029: 0})
	b3aWantExp(t, "reward", p, 263043, 0)
	q.talk(p, talien, b3aCompleted)
}

func TestM11eQ242OnSubclass(t *testing.T) {
	q := b3aStart(t, m11eQ242)
	b3cRequires(t, q.h, q.name, "subclass")
	w := q.h.World()
	npc := func(id int32, npcID int) *scripttest.Npc {
		return &scripttest.Npc{ObjectID: 0x60000000 + id, ID: npcID, Name: "npc"}
	}
	virgil, kassandra, ogmar, knight := npc(1, 31742), npc(2, 31743), npc(3, 31744), npc(4, 31751)
	kalis, matild, fallen := npc(5, 30759), npc(6, 30738), npc(7, 31746)
	restrainer := npc(20, 27317)

	p := m11eOnSubclass(0x10000001, 59)
	q.talk(p, virgil, b3aNoQuest) // no letter
	p.Give(m11eVirgilsLetter, 1)
	q.talk(p, virgil, "31742-02.htm") // level 59
	p.Level = 60
	q.talk(p, virgil, "31742-03.htm")
	q.want("accepted", p, scripting.StateStarted, 1)
	b3aWantItems(t, "letter", p, map[int]int64{m11eVirgilsLetter: 0})
	q.talk(p, virgil, "31742-04.htm")
	q.talk(p, kassandra, "31743-01.htm")
	q.event(p, kassandra, "31743-05.htm", "31743-05.htm")
	q.talk(p, virgil, "31742-05.htm")
	q.talk(p, kassandra, "31743-06.htm")
	q.talk(p, ogmar, "31744-01.htm")
	q.event(p, ogmar, "31744-02.htm", "31744-02.htm")
	q.talk(p, knight, "31751-01.htm")
	q.event(p, knight, "31751-02.htm", "31751-02.htm")
	q.want("knight", p, scripting.StateStarted, 4)

	// Four angel corpses give the golden hair; each dies.
	for i := range 4 {
		corpse := npc(int32(10+i), 31752)
		want := "31752-01.htm"
		if i == 3 {
			want = "31752-02.htm"
		}
		q.talk(p, corpse, want)
	}
	q.want("hair", p, scripting.StateStarted, 5)
	b3aWantItems(t, "hair", p, map[int]int64{7590: 1})
	if dies := fixActions(w, "die"); len(dies) != 4 {
		t.Errorf("corpses died %d times, want 4", len(dies))
	}
	q.talk(p, knight, "31751-04.htm")
	q.want("knight again", p, scripting.StateStarted, 6)
	q.talk(p, kalis, "30759-01.htm")
	q.event(p, kalis, "30759-02.htm", "30759-02.htm")
	q.talk(p, kalis, "30759-03.htm")
	q.talk(p, matild, "30738-01.htm")
	q.event(p, matild, "30738-02.htm", "30738-02.htm")
	q.talk(p, matild, "30738-03.htm")
	q.talk(p, kalis, "30759-04.htm")
	q.event(p, kalis, "30759-05.htm", "30759-05.htm")
	q.want("Kalis", p, scripting.StateStarted, 9)
	b3aWantItems(t, "Kalis", p, map[int]int64{7590: 0, 7596: 0})

	// The restrainers of glory give four orbs of binding, no more.
	for range 5 {
		q.kill(p, restrainer)
	}
	b3aWantItems(t, "orbs", p, map[int]int64{7595: 4})
	for i := range 4 {
		stone := npc(int32(30+i), 31748)
		q.talk(p, stone, "31748-02.htm")
		q.event(p, stone, "31748-03.htm", "31748-03.htm")
	}
	q.want("cornerstones", p, scripting.StateStarted, 10)
	b3aWantItems(t, "cornerstones", p, map[int]int64{7595: 0})

	// The fallen unicorn becomes the pure one, and back after the talk.
	q.talk(p, fallen, "31746-02.htm")
	q.h.Advance(3 * time.Second)
	pure := fixSpawnedOf(w, 31747)
	if len(pure) != 1 {
		t.Fatalf("pure unicorns %d, want 1", len(pure))
	}
	q.talk(p, pure[0], "31747-01.htm")
	q.want("pure unicorn", p, scripting.StateStarted, 11)
	q.h.Advance(5 * time.Second)
	if got := fixSpawnedOf(w, 31746); len(got) != 1 {
		t.Errorf("fallen unicorns spawned %d, want 1", len(got))
	}
	w.ClearRecords()
	q.talk(p, kassandra, "31743-07.htm")
	q.want("done", p, scripting.StateCompleted, 11)
	b3aWantItems(t, "reward", p, map[int]int64{m11eCaradineLetter2: 1})
	b3aWantExp(t, "reward", p, 455764, 0)
	fixWantAction(t, "reward", w, "social", p.ObjectID, 0, 3)
}

func TestM11eQ246OnSubclass(t *testing.T) {
	q := b3aStart(t, m11eQ246)
	b3cRequires(t, q.h, q.name, "subclass")
	caradine := &scripttest.Npc{ObjectID: 0x60000001, ID: 31740, Name: "Caradine"}
	ossian := &scripttest.Npc{ObjectID: 0x60000002, ID: 31741, Name: "Ossian"}
	ladd := &scripttest.Npc{ObjectID: 0x60000003, ID: 30721, Name: "Ladd"}
	pilgrim := &scripttest.Npc{ObjectID: 0x60000004, ID: 21541, Name: "Pilgrim"}
	judge := &scripttest.Npc{ObjectID: 0x60000005, ID: 21544, Name: "Judge"}
	barakiel := &scripttest.Npc{ObjectID: 0x60000006, ID: 25325, Name: "Barakiel"}

	young := m11eOnSubclass(0x10000002, 64)
	young.Give(m11eCaradineLetter2, 1)
	q.talk(young, caradine, "31740-02.htm")

	p := m11eOnSubclass(0x10000001, 65)
	p.Give(m11eCaradineLetter2, 1)
	q.talk(p, caradine, "31740-01.htm")
	q.event(p, caradine, "31740-04.htm", "31740-04.htm")
	q.want("accepted", p, scripting.StateStarted, 1)
	b3aWantItems(t, "letter", p, map[int]int64{m11eCaradineLetter2: 0})
	q.talk(p, caradine, "31740-05.htm")
	q.talk(p, ossian, "31741-01.htm")
	q.event(p, ossian, "31741-02.htm", "31741-02.htm")
	q.talk(p, ossian, "31741-03.htm")

	// rnd(10) < 2: the pilgrims' Waterbinder and the judges' Evergreen.
	q.kill(p, pilgrim, rnd.Get(10, 2))
	b3aWantItems(t, "a miss", p, map[int]int64{7591: 0})
	q.kill(p, pilgrim, rnd.Get(10, 1))
	q.kill(p, pilgrim, rnd.Get(10, 0)) // has it already
	q.kill(p, judge, rnd.Get(10, 0))
	q.want("jewels", p, scripting.StateStarted, 3)
	b3aWantItems(t, "jewels", p, map[int]int64{7591: 1, 7592: 1})
	q.talk(p, ossian, "31741-04.htm")
	q.event(p, ossian, "31741-05.htm", "31741-05.htm")
	q.want("Ossian", p, scripting.StateStarted, 4)

	q.kill(p, barakiel)
	q.want("Barakiel", p, scripting.StateStarted, 5)
	q.talk(p, ossian, "31741-07.htm")
	q.event(p, ossian, "31741-08.htm", "31741-08.htm")
	q.talk(p, ossian, "31741-09.htm")
	q.talk(p, ladd, "30721-01.htm")
	q.event(p, ladd, "30721-02.htm", "30721-02.htm")
	q.want("done", p, scripting.StateCompleted, 6)
	b3aWantItems(t, "reward", p, map[int]int64{7594: 0, m11eCaradineLetter3: 1})
	b3aWantExp(t, "reward", p, 719843, 0)
}

// Possessor of a Precious Soul - 4 ends in noblesse at level 75 on a
// subclass, with the Noblesse Tiara.
func TestM11eQ247Noblesse(t *testing.T) {
	q := b3aStart(t, m11eQ247)
	b3cRequires(t, q.h, q.name, "subclass")
	w := q.h.World()
	caradine := &scripttest.Npc{ObjectID: 0x60000001, ID: 31740, Name: "Caradine"}
	lady := &scripttest.Npc{ObjectID: 0x60000002, ID: 31745, Name: "Lady of the Lake"}

	p := m11eOnSubclass(0x10000001, 74)
	q.talk(p, caradine, b3aNoQuest)
	p.Give(m11eCaradineLetter3, 1)
	q.talk(p, caradine, "31740-02.htm")
	// Too low: the event shows its page but starts nothing.
	q.event(p, caradine, "31740-03.htm", "31740-03.htm")
	if st := p.Quest(q.name); st == nil || st.State() != scripting.StateCreated {
		t.Fatalf("started below level 75: %+v", st)
	}
	p.Level = 75
	q.talk(p, caradine, "31740-01.htm")
	q.event(p, caradine, "31740-03.htm", "31740-03.htm")
	q.want("accepted", p, scripting.StateStarted, 1)
	b3aWantItems(t, "letter", p, map[int]int64{m11eCaradineLetter3: 0})
	q.talk(p, caradine, "31740-04.htm")

	q.h.Event(q.name, "TELEPORT", p, caradine)
	q.want("teleport", p, scripting.StateStarted, 2)
	tps := fixActions(w, "teleport")
	if len(tps) != 1 || tps[0].Loc.X != 143209 || tps[0].Loc.Y != 43968 || tps[0].Loc.Z != -3038 {
		t.Errorf("teleports %+v, want the Lady of the Lake", tps)
	}
	q.talk(p, caradine, "31740-06.htm")
	p.Level = 74
	q.talk(p, lady, "31745-06.htm")
	// On the base class nothing happens.
	p.Level, p.ClassIndex = 75, 0
	q.event(p, lady, "31745-05.htm", "31745-05.htm")
	if p.Noble {
		t.Fatal("noble on the base class")
	}
	p.ClassIndex = m11eSubclassFirst
	q.talk(p, lady, "31745-01.htm")
	w.ClearRecords()
	q.event(p, lady, "31745-05.htm", "31745-05.htm")
	q.want("done", p, scripting.StateCompleted, 2)
	if !p.Noble {
		t.Error("not a noble after the Lady of the Lake")
	}
	b3aWantItems(t, "reward", p, map[int]int64{m11eNoblessTiara: 1})
	b3aWantExp(t, "reward", p, 93836, 0)
	fixWantAction(t, "reward", w, "social", p.ObjectID, 0, 3)
	q.talk(p, caradine, b3aCompleted)
}

// The six quests of M11e load with their gates: subclass for the four
// Possessor quests, grand bosses for Fate's Whisper (Baium), none for
// Mimir's Elixir.
func TestM11eQuestGates(t *testing.T) {
	for _, tt := range []struct {
		name string
		want []string
	}{
		{m11eQ241, []string{"subclass"}},
		{m11eQ242, []string{"subclass"}},
		{m11eQ246, []string{"subclass"}},
		{m11eQ247, []string{"subclass"}},
		{m11eFatesWhisperName, []string{"grand_bosses"}},
		{"Q00235_MimirsElixir", nil},
	} {
		h := b3aLoad(t, tt.name)
		b3cRequires(t, h, tt.name, tt.want...)
	}
}
