package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Helpers of the batch 1 walkthroughs (Talking Island and the newbie
// areas). Every name starts with b1: other batches share the package.

// b1ScriptsDir is the scripts folder, relative to this package.
const b1ScriptsDir = "../../scripts"

// Sounds the batch 1 quests play.
const (
	b1Accept  = "ItemSound.quest_accept"
	b1Middle  = "ItemSound.quest_middle"
	b1Finish  = "ItemSound.quest_finish"
	b1ItemGet = "ItemSound.quest_itemget"
	b1Giveup  = "ItemSound.quest_giveup"
)

// b1Load loads one quest script, quests/<name>/<name>.lua.
func b1Load(t *testing.T, name string) *scripttest.Harness {
	t.Helper()
	return scripttest.Load(t, b1ScriptsDir, "quests/"+name+"/"+name+".lua")
}

// b1Rand forces the next random draws; the test fails if the script draws
// anything else or leaves draws unused.
func b1Rand(t *testing.T, h *scripttest.Harness, draws ...rnd.Draw) {
	t.Helper()
	h.SetRand(rnd.NewScripted(t, draws...))
}

// b1Page checks that a talk or event answered with a page.
func b1Page(t *testing.T, what string, reply scripting.Reply, ok bool, page string) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyPage || reply.Content != page {
		t.Fatalf("%s: reply = %+v, %v; want page %s", what, reply, ok, page)
	}
}

// b1HTML checks that a talk or event answered with inline HTML.
func b1HTML(t *testing.T, what string, reply scripting.Reply, ok bool, html string) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyHTML || reply.Content != html {
		t.Fatalf("%s: reply = %+v, %v; want %q", what, reply, ok, html)
	}
}

// b1NoQuestMsg checks the default "no quest" answer.
func b1NoQuestMsg(t *testing.T, what string, reply scripting.Reply, ok bool) {
	t.Helper()
	b1HTML(t, what, reply, ok, scripting.DefaultNoQuestMsg)
}

// b1Completed checks the default "already completed" answer.
func b1Completed(t *testing.T, what string, reply scripting.Reply, ok bool) {
	t.Helper()
	b1HTML(t, what, reply, ok, scripting.DefaultAlreadyCompletedMsg)
}

// b1Quest checks a quest state's state and cond.
func b1Quest(t *testing.T, what string, p *scripttest.Player, quest string, state, cond int) {
	t.Helper()
	st := p.Quest(quest)
	if st == nil {
		t.Fatalf("%s: no %s state", what, quest)
	}
	if st.State() != state || st.Cond() != cond {
		t.Fatalf("%s: state %d cond %d, want %d and %d (vars %v)", what, st.State(), st.Cond(), state, cond, st.Vars())
	}
}

// b1NoQuest checks that the player has no state of the quest (exit_quest(true)).
func b1NoQuest(t *testing.T, what string, p *scripttest.Player, quest string) {
	t.Helper()
	if st := p.Quest(quest); st != nil {
		t.Fatalf("%s: %s state left: state %d cond %d vars %v", what, quest, st.State(), st.Cond(), st.Vars())
	}
}

// b1Items checks item counts (0 means none).
func b1Items(t *testing.T, what string, p *scripttest.Player, counts map[int]int64) {
	t.Helper()
	for id, n := range counts {
		if got := p.Items[id]; got != n {
			t.Errorf("%s: item %d count %d, want %d", what, id, got, n)
		}
	}
}

// b1ExpSp checks the exp and SP a player got.
func b1ExpSp(t *testing.T, what string, p *scripttest.Player, exp, sp int64) {
	t.Helper()
	if p.Exp != exp || p.Sp != sp {
		t.Errorf("%s: exp, sp = %d, %d; want %d, %d", what, p.Exp, p.Sp, exp, sp)
	}
}

// b1Sounds checks the sounds a player heard since the last ClearRecords.
func b1Sounds(t *testing.T, what string, h *scripttest.Harness, p *scripttest.Player, want ...string) {
	t.Helper()
	if got := h.World().SoundsOf(p); !slices.Equal(got, want) {
		t.Errorf("%s: sounds = %v, want %v", what, got, want)
	}
}
