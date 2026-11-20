package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Helpers of quest batch 3a (Q011–Q053, Q109–Q127, Q171). Every top-level
// name starts with b3a: other batches share this package.

const (
	b3aScriptsDir = "../../scripts"

	b3aSoundAccept  = "ItemSound.quest_accept"
	b3aSoundMiddle  = "ItemSound.quest_middle"
	b3aSoundFinish  = "ItemSound.quest_finish"
	b3aSoundItemGet = "ItemSound.quest_itemget"

	b3aAdena = 57
)

// b3aLoad loads one quest script, quests/<name>/<name>.lua.
func b3aLoad(t *testing.T, name string) *scripttest.Harness {
	t.Helper()
	return scripttest.Load(t, b3aScriptsDir, "quests/"+name+"/"+name+".lua")
}

// b3aQuest is one quest under test: its harness and name.
type b3aQuest struct {
	t    *testing.T
	h    *scripttest.Harness
	name string
}

// b3aStart loads the quest name.
func b3aStart(t *testing.T, name string) *b3aQuest {
	t.Helper()
	return &b3aQuest{t: t, h: b3aLoad(t, name), name: name}
}

// talk talks to npc and wants page.
func (q *b3aQuest) talk(p *scripttest.Player, npc *scripttest.Npc, page string) {
	q.t.Helper()
	reply, ok := q.h.Talk(q.name, p, npc)
	b3aWantReply(q.t, "talk to "+npc.Name, reply, ok, page)
}

// event sends event at npc and wants page.
func (q *b3aQuest) event(p *scripttest.Player, npc *scripttest.Npc, event, page string) {
	q.t.Helper()
	reply, ok := q.h.Event(q.name, event, p, npc)
	b3aWantReply(q.t, "event "+event, reply, ok, page)
}

// kill kills npc after setting the random draws (none: no draw allowed).
func (q *b3aQuest) kill(p *scripttest.Player, npc *scripttest.Npc, draws ...rnd.Draw) {
	q.t.Helper()
	q.h.SetRand(rnd.NewScripted(q.t, draws...))
	q.h.Kill(p, npc)
}

// want checks the player's quest state and cond.
func (q *b3aQuest) want(what string, p *scripttest.Player, state, cond int) {
	q.t.Helper()
	b3aWantQuest(q.t, what, p, q.name, state, cond)
}

// b3aNoQuest stands for the default "no quest" reply in talk and event.
const b3aNoQuest = "<noquest>"

// b3aCompleted stands for the default "already completed" reply.
const b3aCompleted = "<completed>"

// b3aWantReply checks a reply: a page name (ends in .htm/.html), the
// b3aNoQuest or b3aCompleted markers, or "" for no reply at all.
func b3aWantReply(t *testing.T, what string, reply scripting.Reply, ok bool, want string) {
	t.Helper()
	switch {
	case want == b3aNoQuest:
		if !ok || reply.Kind != scripting.ReplyHTML || reply.Content != scripting.DefaultNoQuestMsg {
			t.Fatalf("%s: reply = %+v, %v; want the no-quest message", what, reply, ok)
		}
	case want == b3aCompleted:
		if !ok || reply.Kind != scripting.ReplyHTML || reply.Content != scripting.DefaultAlreadyCompletedMsg {
			t.Fatalf("%s: reply = %+v, %v; want the already-completed message", what, reply, ok)
		}
	case want == "":
		if ok && reply.Content != "" {
			t.Fatalf("%s: reply = %+v; want none", what, reply)
		}
	default:
		if !ok || reply.Kind != scripting.ReplyPage || reply.Content != want {
			t.Fatalf("%s: reply = %+v, %v; want page %s", what, reply, ok, want)
		}
	}
}

// b3aWantQuest checks a quest state's state and cond.
func b3aWantQuest(t *testing.T, what string, p *scripttest.Player, quest string, state, cond int) {
	t.Helper()
	st := p.Quest(quest)
	if st == nil {
		t.Fatalf("%s: no %s state", what, quest)
	}
	if st.State() != state || st.Cond() != cond {
		t.Fatalf("%s: state %d cond %d, want %d and %d (vars %v)", what, st.State(), st.Cond(), state, cond, st.Vars())
	}
}

// b3aWantItems checks item counts (0 means none).
func b3aWantItems(t *testing.T, what string, p *scripttest.Player, counts map[int]int64) {
	t.Helper()
	for id, n := range counts {
		if got := p.Items[id]; got != n {
			t.Errorf("%s: item %d count %d, want %d", what, id, got, n)
		}
	}
}

// b3aWantExp checks the exp and sp a player got.
func b3aWantExp(t *testing.T, what string, p *scripttest.Player, exp, sp int64) {
	t.Helper()
	if p.Exp != exp || p.Sp != sp {
		t.Errorf("%s: exp, sp = %d, %d; want %d, %d", what, p.Exp, p.Sp, exp, sp)
	}
}

// b3aDraws repeats one draw n times.
func b3aDraws(n int, d rnd.Draw) []rnd.Draw {
	out := make([]rnd.Draw, n)
	for i := range out {
		out[i] = d
	}
	return out
}
