package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Helpers of the batch 3b walkthroughs (quests 234–386 outside the newbie
// areas). Every name starts with b3b: other batches share the package.

// b3bScriptsDir is the scripts folder, relative to this package.
const b3bScriptsDir = "../../scripts"

// b3bLoad loads one quest script, quests/<name>/<name>.lua.
func b3bLoad(t *testing.T, name string) *scripttest.Harness {
	t.Helper()
	return scripttest.Load(t, b3bScriptsDir, "quests/"+name+"/"+name+".lua")
}

// b3bPage checks that a talk or event answered with a page.
func b3bPage(t *testing.T, what string, reply scripting.Reply, ok bool, page string) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyPage || reply.Content != page {
		t.Fatalf("%s: reply = %+v, %v; want page %s", what, reply, ok, page)
	}
}

// b3bHTML checks that a talk or event answered with inline HTML.
func b3bHTML(t *testing.T, what string, reply scripting.Reply, ok bool, html string) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyHTML || reply.Content != html {
		t.Fatalf("%s: reply = %+v, %v; want %q", what, reply, ok, html)
	}
}

// b3bQuest checks a quest state's state and cond.
func b3bQuest(t *testing.T, what string, p *scripttest.Player, quest string, state, cond int) {
	t.Helper()
	st := p.Quest(quest)
	if st == nil {
		t.Fatalf("%s: no %s state", what, quest)
	}
	if st.State() != state || st.Cond() != cond {
		t.Fatalf("%s: state %d cond %d, want %d and %d (vars %v)", what, st.State(), st.Cond(), state, cond, st.Vars())
	}
}

// b3bNoQuest checks that the player has no state of the quest (exit_quest(true)).
func b3bNoQuest(t *testing.T, what string, p *scripttest.Player, quest string) {
	t.Helper()
	if st := p.Quest(quest); st != nil {
		t.Fatalf("%s: %s state left: state %d cond %d vars %v", what, quest, st.State(), st.Cond(), st.Vars())
	}
}

// b3bItems checks item counts (0 means none).
func b3bItems(t *testing.T, what string, p *scripttest.Player, counts map[int]int64) {
	t.Helper()
	for id, n := range counts {
		if got := p.Items[id]; got != n {
			t.Errorf("%s: item %d count %d, want %d", what, id, got, n)
		}
	}
}

// b3bExpSp checks the exp and SP a player got.
func b3bExpSp(t *testing.T, what string, p *scripttest.Player, exp, sp int64) {
	t.Helper()
	if p.Exp != exp || p.Sp != sp {
		t.Errorf("%s: exp, sp = %d, %d; want %d, %d", what, p.Exp, p.Sp, exp, sp)
	}
}
