package questtests

import (
	"slices"
	"strconv"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Helpers of the batch 4 tests (the sagas, scripts/lib/saga.lua). Every
// name starts with b4: other batches share the package.

// b4ScriptsDir is the scripts folder, relative to this package.
const b4ScriptsDir = "../../scripts"

// b4Sagas are the 31 saga quests, Q070 to Q100.
var b4Sagas = []string{
	"Q00070_SagaOfThePhoenixKnight", "Q00071_SagaOfEvasTemplar", "Q00072_SagaOfTheSwordMuse",
	"Q00073_SagaOfTheDuelist", "Q00074_SagaOfTheDreadnought", "Q00075_SagaOfTheTitan",
	"Q00076_SagaOfTheGrandKhavatari", "Q00077_SagaOfTheDominator", "Q00078_SagaOfTheDoomcryer",
	"Q00079_SagaOfTheAdventurer", "Q00080_SagaOfTheWindRider", "Q00081_SagaOfTheGhostHunter",
	"Q00082_SagaOfTheSagittarius", "Q00083_SagaOfTheMoonlightSentinel", "Q00084_SagaOfTheGhostSentinel",
	"Q00085_SagaOfTheCardinal", "Q00086_SagaOfTheHierophant", "Q00087_SagaOfEvasSaint",
	"Q00088_SagaOfTheArchmage", "Q00089_SagaOfTheMysticMuse", "Q00090_SagaOfTheStormScreamer",
	"Q00091_SagaOfTheArcanaLord", "Q00092_SagaOfTheElementalMaster", "Q00093_SagaOfTheSpectralMaster",
	"Q00094_SagaOfTheSoultaker", "Q00095_SagaOfTheHellKnight", "Q00096_SagaOfTheSpectralDancer",
	"Q00097_SagaOfTheShillienTemplar", "Q00098_SagaOfTheShillienSaint", "Q00099_SagaOfTheFortuneSeeker",
	"Q00100_SagaOfTheMaestro",
}

// Q070, the saga the walkthroughs play: a level 76 Paladin (class 5)
// becomes a Phoenix Knight (90).
const (
	b4Q070    = "Q00070_SagaOfThePhoenixKnight"
	b4Paladin = 5

	// NPCs (saga npc[1..12]): Sedrick is npc[1] and npc[4], Hardin's
	// Academy npc[3] and npc[12].
	b4Sedrick   = 30849
	b4Npc2      = 31624
	b4Npc3      = 31277
	b4Npc5      = 31631 // spawned at step 10, owns its first talk
	b4Tablet1   = 31646
	b4Tablet2   = 31647
	b4Tablet3   = 31650
	b4Tablet4   = 31654
	b4Tablet5   = 31655
	b4Tablet6   = 31657
	b4Mob1      = 27286
	b4Archon    = 27219
	b4Mob3      = 27278
	b4Guardian  = 27214
	b4Minion    = 21646
	b4HalishaNm = 18212

	// Items (saga items[1..12]).
	b4IceCrystal  = 7080
	b4Item2       = 7534
	b4DivineStone = 7081
	b4HalishaMark = 7485
	b4Item5       = 7268
	b4Item6       = 7299
	b4Item7       = 7330
	b4Item8       = 7361
	b4Item9       = 7392
	b4Item10      = 7423
	b4Item11      = 7093
	b4Item12      = 6482
)

// The saga lines Q070 uses (all sagas share Mobius's English lines).
const (
	b4Text1  = "PLAYERNAME! Pursued to here! However, I jumped out of the Banshouren boundaries! You look at the giant as the sign of power!"
	b4Text2  = "... Oh ... good! So it was ... let's begin!"
	b4Text5  = "Now, my soul freed from the shackles of the millennium, Halixia, to the back side I come ..."
	b4Text6  = "Why do you interfere others' battles?"
	b4Text7  = "This is a waste of time.. Say goodbye...!"
	b4Text8  = "...That is the enemy"
	b4Text13 = "let..."
	b4Text14 = "Only engaged in the battle to bar their choice. Perhaps you should regret."
	b4Text15 = "The human nation was foolish to try and fight a giant's strength."
	b4Text18 = "....! Fight...Defeat...It...Fight...Defeat...It..."
)

// b4File is the script path of a saga.
func b4File(name string) string { return "quests/" + name + "/" + name + ".lua" }

// b4Load loads the given sagas.
func b4Load(t *testing.T, names ...string) *scripttest.Harness {
	t.Helper()
	files := make([]string, len(names))
	for i, n := range names {
		files[i] = b4File(n)
	}
	return scripttest.Load(t, b4ScriptsDir, files...)
}

// b4Page checks that a talk or event answered with a page.
func b4Page(t *testing.T, what string, reply scripting.Reply, ok bool, page string) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyPage || reply.Content != page {
		t.Fatalf("%s: reply = %+v, %v; want page %s", what, reply, ok, page)
	}
}

// b4NoQuestMsg checks the default "no quest" answer.
func b4NoQuestMsg(t *testing.T, what string, reply scripting.Reply, ok bool) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyHTML || reply.Content != scripting.DefaultNoQuestMsg {
		t.Fatalf("%s: reply = %+v, %v; want the no-quest message", what, reply, ok)
	}
}

// b4Quest checks a quest state's state and cond.
func b4Quest(t *testing.T, what string, p *scripttest.Player, quest string, state, cond int) {
	t.Helper()
	st := p.Quest(quest)
	if st == nil {
		t.Fatalf("%s: no %s state", what, quest)
	}
	if st.State() != state || st.Cond() != cond {
		t.Fatalf("%s: state %d cond %d, want %d and %d (vars %v)", what, st.State(), st.Cond(), state, cond, st.Vars())
	}
}

// b4Var checks a quest variable.
func b4Var(t *testing.T, what string, p *scripttest.Player, quest, name, want string) {
	t.Helper()
	got, _ := p.Quest(quest).Get(name)
	if got != want {
		t.Fatalf("%s: %s = %q, want %q", what, name, got, want)
	}
}

// b4Items checks item counts (0 means none).
func b4Items(t *testing.T, what string, p *scripttest.Player, counts map[int]int64) {
	t.Helper()
	for id, n := range counts {
		if got := p.Items[id]; got != n {
			t.Errorf("%s: item %d count %d, want %d", what, id, got, n)
		}
	}
}

// b4Said returns what NPCs said (npc:broadcast_say) since the last
// ClearRecords, in order.
func b4Said(h *scripttest.Harness) []string {
	var out []string
	for _, a := range h.World().Actions {
		if a.Kind == "say" {
			out = append(out, a.Text)
		}
	}
	return out
}

// b4WantSaid checks what NPCs said since the last ClearRecords.
func b4WantSaid(t *testing.T, what string, h *scripttest.Harness, want ...string) {
	t.Helper()
	if got := b4Said(h); !slices.Equal(got, want) {
		t.Errorf("%s: said %q, want %q", what, got, want)
	}
}

// b4NoTimerErrors checks that no timer handler failed.
func b4NoTimerErrors(t *testing.T, h *scripttest.Harness) {
	t.Helper()
	for _, r := range h.World().Replies {
		if r.Reply.Err != nil {
			t.Errorf("a timer handler failed: %v", r.Reply.Err)
		}
	}
}

// b4Last returns the last NPC with the template id the scripts spawned
// since the last ClearRecords; the test fails if there is none.
func b4Last(t *testing.T, h *scripttest.Harness, id int) *scripttest.Npc {
	t.Helper()
	s := b4Spawned(h, id)
	if len(s) == 0 {
		t.Fatalf("no NPC %d spawned (spawned %+v)", id, h.World().Spawned)
	}
	return s[len(s)-1]
}

// b4Spawned returns the NPCs with the template id the scripts spawned
// since the last ClearRecords.
func b4Spawned(h *scripttest.Harness, id int) []*scripttest.Npc {
	var out []*scripttest.Npc
	for _, n := range h.World().Spawned {
		if n.ID == id {
			out = append(out, n)
		}
	}
	return out
}

// b4Start makes p take Q070 at Sedrick and moves the quest to cond.
func b4Start(t *testing.T, h *scripttest.Harness, p *scripttest.Player, cond int) {
	t.Helper()
	sedrick := &scripttest.Npc{ObjectID: 0x6000ffff, ID: b4Sedrick}
	h.Talk(b4Q070, p, sedrick)
	h.Event(b4Q070, "accept", p, sedrick)
	if cond != 1 {
		p.Quest(b4Q070).Set("cond", strconv.Itoa(cond))
	}
	b4Quest(t, "start", p, b4Q070, scripting.StateStarted, cond)
}
