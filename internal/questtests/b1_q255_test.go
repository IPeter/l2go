package questtests

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q255 = "Q00255_Tutorial"

// b1Q255Op is one tutorial packet: what and its argument.
type b1Q255Op struct {
	op  scripting.TutorialOp
	arg string
}

// b1Q255Ops returns the tutorial packets sent so far and forgets the
// records.
func b1Q255Ops(w *scripttest.World) []b1Q255Op {
	var ops []b1Q255Op
	for _, a := range w.Actions {
		if a.Kind == "tutorial" {
			ops = append(ops, b1Q255Op{scripting.TutorialOp(a.Value), a.Text})
		}
	}
	w.ClearRecords()
	return ops
}

// b1Q255Var checks one variable of the tutorial state.
func b1Q255Var(t *testing.T, what string, p *scripttest.Player, name, want string) {
	t.Helper()
	if got, _ := p.Quest(b1Q255).Get(name); got != want {
		t.Errorf("%s: %s = %q, want %q", what, name, got, want)
	}
}

// The tutorial's first steps for a new human fighter: the "UC" event on
// entering the world, the two voice timers, the client event of the first
// tutorial page, a question mark and the level 5 notice. Without the
// tutorial state (the game creates it at character creation) and from
// level 6, nothing happens.
func TestB1Q255Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q255)
	w := h.World()

	stranger := &scripttest.Player{ObjectID: 0x10000001, Name: "Stranger", Level: 1}
	reply, ok := h.Event(b1Q255, "UC", stranger, nil)
	if ok && reply.Kind != scripting.ReplyNone {
		t.Errorf("no state: reply %+v", reply)
	}
	if len(w.Actions) != 0 || w.Pending() != 0 {
		t.Errorf("no state: actions %+v, %d timers", w.Actions, w.Pending())
	}

	veteran := &scripttest.Player{ObjectID: 0x10000002, Name: "Veteran", Level: 6}
	w.NewQuestState(veteran.Ref(), b1Q255)
	h.Event(b1Q255, "UC", veteran, nil)
	if w.Pending() != 0 {
		t.Errorf("level 6: %d timers started", w.Pending())
	}

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 1}
	w.NewQuestState(hero.Ref(), b1Q255)
	h.Event(b1Q255, "UC", hero, nil)
	b1Q255Var(t, "UC", hero, "Ex", "-2")
	b1Q255Var(t, "UC", hero, "ucMemo", "0")

	// Ten seconds later the welcome voice and page, then the second voice.
	h.Advance(10 * time.Second)
	if got, want := b1Q255Ops(w), []b1Q255Op{
		{scripting.TutorialVoice, "tutorial_voice_001a"},
		{scripting.TutorialShowHTML, "tutorial_human_fighter001.htm"},
	}; !slices.Equal(got, want) {
		t.Errorf("first timer: %+v, want %+v", got, want)
	}
	b1Q255Var(t, "first timer", hero, "Ex", "-3")
	h.Advance(30 * time.Second)
	if got, want := b1Q255Ops(w), []b1Q255Op{{scripting.TutorialVoice, "tutorial_voice_002"}}; !slices.Equal(got, want) {
		t.Errorf("second timer: %+v, want %+v", got, want)
	}
	b1Q255Var(t, "second timer", hero, "Ex", "0")
	if w.Pending() != 0 {
		t.Errorf("%d timers left", w.Pending())
	}

	// The client event 8 of the first page: the radar to the Newbie Helper.
	h.Event(b1Q255, "CE8", hero, nil)
	var radar []scripttest.Action
	for _, a := range w.Actions {
		if a.Kind == "radar" {
			radar = append(radar, a)
		}
	}
	if len(radar) != 1 || radar[0].Loc != (scripting.Location{X: -71424, Y: 258336, Z: -3109}) {
		t.Errorf("radar %+v", radar)
	}
	if got, want := b1Q255Ops(w), []b1Q255Op{
		{scripting.TutorialVoice, "tutorial_voice_007"},
		{scripting.TutorialShowHTML, "tutorial_human_fighter007.htm"},
	}; !slices.Equal(got, want) {
		t.Errorf("CE8: %+v, want %+v", got, want)
	}
	b1Q255Var(t, "CE8", hero, "ucMemo", "1")
	b1Q255Var(t, "CE8", hero, "Ex", "-5")

	h.Event(b1Q255, "QM3", hero, nil)
	if got, want := b1Q255Ops(w), []b1Q255Op{
		{scripting.TutorialClientEvent, "1048576"},
		{scripting.TutorialShowHTML, "tutorial_09.htm"},
	}; !slices.Equal(got, want) {
		t.Errorf("QM3: %+v, want %+v", got, want)
	}

	// Level 5 for a fighter: the skill question mark, once.
	hero.Level = 5
	h.Event(b1Q255, "CE40", hero, nil)
	if got, want := b1Q255Ops(w), []b1Q255Op{
		{scripting.TutorialVoice, "tutorial_voice_014"},
		{scripting.TutorialQuestionMark, "9"},
	}; !slices.Equal(got, want) {
		t.Errorf("CE40: %+v, want %+v", got, want)
	}
	b1Q255Var(t, "CE40", hero, "lvl", "5")
	h.Event(b1Q255, "CE40", hero, nil)
	if got := b1Q255Ops(w); len(got) != 0 {
		t.Errorf("CE40 again: %+v", got)
	}
}
