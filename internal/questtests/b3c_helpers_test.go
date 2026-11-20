package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Shared helpers of the batch 3c walkthroughs (quests 419 and up). Every
// top-level name in the b3c files starts with b3c: other batches write to
// this package too.

const b3cScriptsDir = "../../scripts"

const (
	b3cSoundAccept  = "ItemSound.quest_accept"
	b3cSoundMiddle  = "ItemSound.quest_middle"
	b3cSoundFinish  = "ItemSound.quest_finish"
	b3cSoundItemGet = "ItemSound.quest_itemget"
	b3cAdena        = 57
)

// b3cLoad loads one quest script, quests/<name>/<name>.lua.
func b3cLoad(t *testing.T, name string) *scripttest.Harness {
	t.Helper()
	return scripttest.Load(t, b3cScriptsDir, "quests/"+name+"/"+name+".lua")
}

// b3cPage checks that a talk or event answered with a page.
func b3cPage(t *testing.T, what string, reply scripting.Reply, ok bool, page string) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyPage || reply.Content != page {
		t.Fatalf("%s: reply = %+v, %v; want page %s", what, reply, ok, page)
	}
}

// b3cHTML checks that a talk answered with inline HTML (e.g.
// scripting.DefaultNoQuestMsg).
func b3cHTML(t *testing.T, what string, reply scripting.Reply, ok bool, html string) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyHTML || reply.Content != html {
		t.Fatalf("%s: reply = %+v, %v; want %q", what, reply, ok, html)
	}
}

// b3cText checks that a talk or event answered with a text message.
func b3cText(t *testing.T, what string, reply scripting.Reply, ok bool, text string) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyText || reply.Content != text {
		t.Fatalf("%s: reply = %+v, %v; want text %q", what, reply, ok, text)
	}
}

// b3cQuest checks a quest state's state and cond.
func b3cQuest(t *testing.T, what string, p *scripttest.Player, quest string, state, cond int) {
	t.Helper()
	st := p.Quest(quest)
	if st == nil {
		t.Fatalf("%s: no %s state", what, quest)
	}
	if st.State() != state || st.Cond() != cond {
		t.Fatalf("%s: state %d cond %d, want %d and %d (vars %v)", what, st.State(), st.Cond(), state, cond, st.Vars())
	}
}

// b3cNoQuest checks that the player has no state of the quest (never
// created, or deleted by exit_quest(true)).
func b3cNoQuest(t *testing.T, what string, p *scripttest.Player, quest string) {
	t.Helper()
	if st := p.Quest(quest); st != nil {
		t.Fatalf("%s: %s state exists: state %d cond %d vars %v", what, quest, st.State(), st.Cond(), st.Vars())
	}
}

// b3cItems checks item counts (0 means none).
func b3cItems(t *testing.T, what string, p *scripttest.Player, counts map[int]int64) {
	t.Helper()
	for id, n := range counts {
		if got := p.Items[id]; got != n {
			t.Errorf("%s: item %d count %d, want %d", what, id, got, n)
		}
	}
}

// b3cSounds checks the sounds a player heard since the last ClearRecords.
func b3cSounds(t *testing.T, what string, w *scripttest.World, p *scripttest.Player, want ...string) {
	t.Helper()
	if got := w.SoundsOf(p); !slices.Equal(got, want) {
		t.Errorf("%s: sounds = %v, want %v", what, got, want)
	}
}

// b3cRequires checks a quest's q:requires features.
func b3cRequires(t *testing.T, h *scripttest.Harness, quest string, want ...string) {
	t.Helper()
	info, ok := h.Engine().Quest(quest)
	if !ok {
		t.Fatalf("quest %s not loaded", quest)
	}
	if !slices.Equal(info.Requires, want) {
		t.Errorf("%s requires %v, want %v", quest, info.Requires, want)
	}
}
