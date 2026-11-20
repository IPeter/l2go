package scripting_test

import (
	"context"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const (
	soundAccept  = "ItemSound.quest_accept"
	soundMiddle  = "ItemSound.quest_middle"
	soundFinish  = "ItemSound.quest_finish"
	soundItemGet = "ItemSound.quest_itemget"
)

// wantPage checks that a talk or event answered with a page.
func wantPage(t *testing.T, what string, reply scripting.Reply, ok bool, page string) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyPage || reply.Content != page {
		t.Fatalf("%s: reply = %+v, %v; want page %s", what, reply, ok, page)
	}
}

// wantHTML checks that a talk answered with inline HTML.
func wantHTML(t *testing.T, what string, reply scripting.Reply, ok bool, html string) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyHTML || reply.Content != html {
		t.Fatalf("%s: reply = %+v, %v; want %q", what, reply, ok, html)
	}
}

// wantQuest checks a quest state's state and cond.
func wantQuest(t *testing.T, what string, p *scripttest.Player, quest string, state, cond int) {
	t.Helper()
	st := p.Quest(quest)
	if st == nil {
		t.Fatalf("%s: no %s state", what, quest)
	}
	if st.State() != state || st.Cond() != cond {
		t.Fatalf("%s: state %d cond %d, want %d and %d (vars %v)", what, st.State(), st.Cond(), state, cond, st.Vars())
	}
}

// wantItems checks item counts (0 means none).
func wantItems(t *testing.T, what string, p *scripttest.Player, counts map[int]int64) {
	t.Helper()
	for id, n := range counts {
		if got := p.Items[id]; got != n {
			t.Errorf("%s: item %d count %d, want %d", what, id, got, n)
		}
	}
}

const q001 = "Q00001_LettersOfLove"

// Letters of Love, the whole way: Darin, Roxxy, Darin, Baulro, Darin.
func TestQ001Walkthrough(t *testing.T) {
	WalkQ001(t, scripttest.Load(t, scriptsDir, "quests/Q00001_LettersOfLove/Q00001_LettersOfLove.lua"))
}

// WalkQ001 plays Q001 in h (the hand port, or questport's translation).
func WalkQ001(t *testing.T, h *scripttest.Harness) {
	w := h.World()
	darin := &scripttest.Npc{ObjectID: 0x60000001, ID: 30048, Name: "Darin"}
	roxxy := &scripttest.Npc{ObjectID: 0x60000002, ID: 30006, Name: "Roxxy"}
	baulro := &scripttest.Npc{ObjectID: 0x60000003, ID: 30033, Name: "Baulro"}

	if got := h.Engine().StartQuests(30048); !slices.Equal(got, []string{q001}) {
		t.Errorf("StartQuests(Darin) = %v", got)
	}
	for _, id := range []int{30048, 30006, 30033} {
		if got := h.Engine().TalkScripts(id); !slices.Equal(got, []string{q001}) {
			t.Errorf("TalkScripts(%d) = %v", id, got)
		}
	}
	info, ok := h.Engine().QuestByID(1)
	if !ok || info.Name != q001 || info.Description != "Letters of Love" || !slices.Equal(info.QuestItems, []int{687, 688, 1079, 1080}) {
		t.Errorf("QuestByID(1) = %+v, %v", info, ok)
	}

	// A level 1 character is too young; a new state stays CREATED.
	newbie := &scripttest.Player{ObjectID: 0x10000001, Name: "Newbie", Level: 1}
	reply, ok := h.Talk(q001, newbie, darin)
	wantPage(t, "level 1", reply, ok, "30048-01.htm")
	wantQuest(t, "level 1", newbie, q001, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 5}
	// Pages of the dialog before the quest exists just show.
	reply, ok = h.Event(q001, "30048-03.htm", hero, darin)
	wantPage(t, "before the quest", reply, ok, "30048-03.htm")

	reply, ok = h.Talk(q001, hero, darin)
	wantPage(t, "offer", reply, ok, "30048-02.htm")
	// Baulro has nothing to say yet.
	reply, ok = h.Talk(q001, hero, baulro)
	wantPage(t, "Baulro before the start", reply, ok, "30048-02.htm") // Mobius: a CREATED state gets Darin's offer anywhere

	reply, ok = h.Event(q001, "30048-06.htm", hero, darin)
	wantPage(t, "accept", reply, ok, "30048-06.htm")
	wantQuest(t, "accept", hero, q001, scripting.StateStarted, 1)
	wantItems(t, "accept", hero, map[int]int64{687: 1})

	reply, ok = h.Talk(q001, hero, darin)
	wantPage(t, "Darin cond 1", reply, ok, "30048-07.htm")
	reply, ok = h.Talk(q001, hero, baulro)
	wantHTML(t, "Baulro cond 1", reply, ok, scripting.DefaultNoQuestMsg)

	reply, ok = h.Talk(q001, hero, roxxy)
	wantPage(t, "Roxxy", reply, ok, "30006-01.htm")
	wantQuest(t, "Roxxy", hero, q001, scripting.StateStarted, 2)
	wantItems(t, "Roxxy", hero, map[int]int64{687: 0, 688: 1})
	reply, ok = h.Talk(q001, hero, roxxy)
	wantPage(t, "Roxxy again", reply, ok, "30006-02.htm")

	reply, ok = h.Talk(q001, hero, darin)
	wantPage(t, "Darin cond 2", reply, ok, "30048-08.htm")
	wantQuest(t, "Darin cond 2", hero, q001, scripting.StateStarted, 3)
	wantItems(t, "Darin cond 2", hero, map[int]int64{688: 0, 1079: 1})
	reply, ok = h.Talk(q001, hero, darin)
	wantPage(t, "Darin cond 3", reply, ok, "30048-09.htm")
	reply, ok = h.Talk(q001, hero, roxxy)
	wantPage(t, "Roxxy cond 3", reply, ok, "30006-03.htm")

	reply, ok = h.Talk(q001, hero, baulro)
	wantPage(t, "Baulro", reply, ok, "30033-01.htm")
	wantQuest(t, "Baulro", hero, q001, scripting.StateStarted, 4)
	wantItems(t, "Baulro", hero, map[int]int64{1079: 0, 1080: 1})
	reply, ok = h.Talk(q001, hero, baulro)
	wantPage(t, "Baulro again", reply, ok, "30033-02.htm")

	reply, ok = h.Talk(q001, hero, darin)
	wantPage(t, "finish", reply, ok, "30048-10.htm")
	wantQuest(t, "finish", hero, q001, scripting.StateCompleted, 4)
	wantItems(t, "finish", hero, map[int]int64{1080: 0, 906: 1})
	if vars := hero.Quest(q001).Vars(); len(vars) != 0 {
		t.Errorf("a completed quest keeps variables %v", vars)
	}

	// Completed: Darin says so, from the HTML folder when it has the page.
	reply, ok = h.Talk(q001, hero, darin)
	wantHTML(t, "completed", reply, ok, scripting.DefaultAlreadyCompletedMsg)
	w.SetPage("alreadycompleted.htm", "<html><body>Done.</body></html>")
	reply, ok = h.Talk(q001, hero, darin)
	wantHTML(t, "completed page", reply, ok, "<html><body>Done.</body></html>")

	if got, want := w.SoundsOf(hero), []string{soundAccept, soundMiddle, soundMiddle, soundMiddle, soundFinish}; !slices.Equal(got, want) {
		t.Errorf("sounds = %v, want %v", got, want)
	}
	// One quest mark per step (cond 1 to 4), QuestList on each change.
	if got := len(w.QuestMarks); got != 4 {
		t.Errorf("quest marks = %v, want 4", w.QuestMarks)
	}
	if got := w.QuestLists[hero.ObjectID]; got != 6 {
		t.Errorf("QuestList sent %d times, want 6 (4 conds, started, completed)", got)
	}
}

const q401 = "Q00401_PathOfTheWarrior"

// Path to a Warrior, with forced drops: Auron, Simplon, ten Rusted Bronze
// Swords, Simplon, Auron, twenty spider legs with the sword in hand, Auron.
func TestQ401Walkthrough(t *testing.T) {
	WalkQ401(t, scripttest.Load(t, scriptsDir, "quests/Q00401_PathOfTheWarrior/Q00401_PathOfTheWarrior.lua"))
}

// WalkQ401 plays Q401 in h (the hand port, or questport's translation).
func WalkQ401(t *testing.T, h *scripttest.Harness) {
	w := h.World()
	auron := &scripttest.Npc{ObjectID: 0x60000001, ID: 30010, Name: "Auron"}
	simplon := &scripttest.Npc{ObjectID: 0x60000002, ID: 30253, Name: "Simplon"}
	skeleton := &scripttest.Npc{ObjectID: 0x60000003, ID: 20035, Name: "Tracker Skeleton"}
	leader := &scripttest.Npc{ObjectID: 0x60000004, ID: 20042, Name: "Tracker Skeleton Leader"}
	spider := &scripttest.Npc{ObjectID: 0x60000005, ID: 20038, Name: "Venomous Spider"}
	arachnid := &scripttest.Npc{ObjectID: 0x60000006, ID: 20043, Name: "Arachnid Tracker"}

	// The refusals of 30010-05.htm.
	for _, tt := range []struct {
		name         string
		class, level int
		medallion    bool
		page         string
	}{
		{"mage", 10, 19, false, "30010-02b.htm"},
		{"warrior already", 1, 20, false, "30010-03.htm"},
		{"too young", 0, 18, false, "30010-02.htm"},
		{"medallion", 0, 19, true, "30010-04.htm"},
		{"fits", 0, 19, false, "30010-05.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000100, Name: tt.name, Level: tt.level, ClassID: tt.class}
		if tt.medallion {
			p.Give(1145, 1)
		}
		reply, ok := h.Talk(q401, p, auron)
		wantPage(t, tt.name+" talk", reply, ok, "30010-01.htm")
		reply, ok = h.Event(q401, "30010-05.htm", p, auron)
		wantPage(t, tt.name, reply, ok, tt.page)
	}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 0}
	reply, ok := h.Talk(q401, hero, auron)
	wantPage(t, "offer", reply, ok, "30010-01.htm")
	reply, ok = h.Event(q401, "30010-06.htm", hero, auron)
	wantPage(t, "accept", reply, ok, "30010-06.htm")
	wantQuest(t, "accept", hero, q401, scripting.StateStarted, 1)
	wantItems(t, "accept", hero, map[int]int64{1138: 1})

	reply, ok = h.Talk(q401, hero, auron)
	wantPage(t, "Auron cond 1", reply, ok, "30010-07.htm")
	reply, ok = h.Talk(q401, hero, simplon)
	wantPage(t, "Simplon cond 1", reply, ok, "30253-01.htm")
	reply, ok = h.Event(q401, "30253-02.htm", hero, simplon)
	wantPage(t, "Simplon letter", reply, ok, "30253-02.htm")
	wantQuest(t, "Simplon letter", hero, q401, scripting.StateStarted, 2)
	wantItems(t, "Simplon letter", hero, map[int]int64{1138: 0, 1139: 1})
	reply, ok = h.Talk(q401, hero, simplon)
	wantPage(t, "no swords", reply, ok, "30253-03.htm")

	// Skeletons drop a sword when rnd(10) < 4: 3 drops, 4 does not.
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 3), rnd.Get(10, 4)))
	h.Kill(hero, skeleton)
	h.Kill(hero, leader)
	wantItems(t, "first kills", hero, map[int]int64{1140: 1})
	reply, ok = h.Talk(q401, hero, simplon)
	wantPage(t, "one sword", reply, ok, "30253-03b.htm")
	// Spiders give nothing before cond 5, and draw nothing.
	h.Kill(hero, spider)

	draws := make([]rnd.Draw, 9)
	for i := range draws {
		draws[i] = rnd.Get(10, 0)
	}
	h.SetRand(rnd.NewScripted(t, draws...))
	for range 9 {
		h.Kill(hero, skeleton)
	}
	wantQuest(t, "ten swords", hero, q401, scripting.StateStarted, 3)
	wantItems(t, "ten swords", hero, map[int]int64{1140: 10})
	// At cond 3 kills draw nothing (the script would fail on a draw).
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, skeleton)
	wantItems(t, "after ten", hero, map[int]int64{1140: 10})

	reply, ok = h.Talk(q401, hero, auron)
	wantPage(t, "Auron cond 3", reply, ok, "30010-08.htm")
	reply, ok = h.Talk(q401, hero, simplon)
	wantPage(t, "Simplon swords", reply, ok, "30253-04.htm")
	wantQuest(t, "Simplon swords", hero, q401, scripting.StateStarted, 4)
	wantItems(t, "Simplon swords", hero, map[int]int64{1140: 0, 1139: 0, 1141: 1, 1143: 1})
	reply, ok = h.Talk(q401, hero, simplon)
	wantPage(t, "Simplon cond 4", reply, ok, "30253-05.htm")
	reply, ok = h.Talk(q401, hero, auron)
	wantPage(t, "Auron cond 4", reply, ok, "30010-09.htm")
	reply, ok = h.Event(q401, "30010-11.htm", hero, auron)
	wantPage(t, "Auron letter", reply, ok, "30010-11.htm")
	wantQuest(t, "Auron letter", hero, q401, scripting.StateStarted, 5)
	wantItems(t, "Auron letter", hero, map[int]int64{1141: 0, 1143: 0, 1142: 1})
	reply, ok = h.Talk(q401, hero, auron)
	wantPage(t, "Auron cond 5", reply, ok, "30010-12.htm")

	// Spider legs only with the Rusted Bronze Sword in the right hand.
	h.Kill(hero, spider)
	wantItems(t, "bare hands", hero, map[int]int64{1144: 0})
	const rhand = 7 // Paperdoll.RHAND
	hero.Equip(rhand, 1142)
	for i := range 20 {
		if i%2 == 0 {
			h.Kill(hero, spider)
		} else {
			h.Kill(hero, arachnid)
		}
	}
	wantQuest(t, "twenty legs", hero, q401, scripting.StateStarted, 6)
	wantItems(t, "twenty legs", hero, map[int]int64{1144: 20})

	w.ClearRecords()
	reply, ok = h.Talk(q401, hero, auron)
	wantPage(t, "medallion", reply, ok, "30010-13.htm")
	wantItems(t, "medallion", hero, map[int]int64{1144: 0, 1142: 0, 1145: 1})
	if hero.Equipped[rhand] != 0 {
		t.Error("the sword stays equipped after it was taken")
	}
	if hero.Exp != 3200 || hero.Sp != 1500 {
		t.Errorf("exp, sp = %d, %d; want 3200, 1500", hero.Exp, hero.Sp)
	}
	if want := []scripttest.Action{{Kind: "social", Object: hero.ObjectID, Value: 3}}; !slices.Equal(w.Actions, want) {
		t.Errorf("actions = %+v, want %+v", w.Actions, want)
	}
	if got := w.SoundsOf(hero); !slices.Equal(got, []string{soundFinish}) {
		t.Errorf("sounds = %v", got)
	}
	// exit_quest(true) deletes the state: the quest can be offered again,
	// and the medallion now refuses it.
	if hero.Quest(q401) != nil {
		t.Errorf("the state survives exit_quest(true): %+v", hero.Quest(q401))
	}
	reply, ok = h.Talk(q401, hero, auron)
	wantPage(t, "again", reply, ok, "30010-01.htm")
	reply, ok = h.Event(q401, "30010-05.htm", hero, auron)
	wantPage(t, "again with medallion", reply, ok, "30010-04.htm")
}

// The middle of Q401's kills: the item sound on each sword but the tenth,
// which moves the quest on with the middle sound instead.
func TestQ401KillSounds(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "quests/Q00401_PathOfTheWarrior/Q00401_PathOfTheWarrior.lua")
	auron := &scripttest.Npc{ObjectID: 0x60000001, ID: 30010}
	simplon := &scripttest.Npc{ObjectID: 0x60000002, ID: 30253}
	skeleton := &scripttest.Npc{ObjectID: 0x60000003, ID: 20035}
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19}
	h.Talk(q401, hero, auron)
	h.Event(q401, "30010-06.htm", hero, auron)
	h.Event(q401, "30253-02.htm", hero, simplon)
	hero.Give(1140, 8)
	h.World().ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 1), rnd.Get(10, 2)))
	h.Kill(hero, skeleton)
	h.Kill(hero, skeleton)
	if got := h.World().SoundsOf(hero); !slices.Equal(got, []string{soundItemGet, soundMiddle}) {
		t.Errorf("sounds = %v", got)
	}
	wantQuest(t, "tenth", hero, q401, scripting.StateStarted, 3)
}

// A talk event must cost less than 50 µs (Plan 11, Phase Q3).
func BenchmarkQuestTalk(b *testing.B) {
	h := scripttest.Load(b, scriptsDir, "quests/Q00001_LettersOfLove/Q00001_LettersOfLove.lua")
	darin := &scripttest.Npc{ObjectID: 0x60000001, ID: 30048, Name: "Darin"}
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 5}
	h.Talk(q001, hero, darin)
	h.Event(q001, "30048-06.htm", hero, darin)
	e := h.Engine()
	ctx := context.Background()
	p, n := hero.Ref(), darin.Ref()
	for b.Loop() {
		if reply, ok := e.Talk(ctx, q001, 30048, p, n); !ok || reply.Content != "30048-07.htm" {
			b.Fatalf("reply = %+v, %v", reply, ok)
		}
	}
}
