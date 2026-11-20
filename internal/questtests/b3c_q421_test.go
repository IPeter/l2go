package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ421 = "Q00421_LittleWingsBigAdventure"

// Little Wing's Big Adventure waits for pets (summons, flute levels); until
// then only Cronos's refusals that need no pet run.
func TestB3cQ421Refusals(t *testing.T) {
	h := b3cLoad(t, b3cQ421)
	b3cRequires(t, h, b3cQ421, "pets")
	cronos := &scripttest.Npc{ObjectID: 0x60000001, ID: 30610, Name: "Cronos"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 44}
	young.Give(3500, 1)
	reply, ok := h.Talk(b3cQ421, young, cronos)
	b3cPage(t, "level 44", reply, ok, "30610-01.htm")
	b3cQuest(t, "level 44", young, b3cQ421, scripting.StateCreated, 0)

	noFlute := &scripttest.Player{ObjectID: 0x10000002, Name: "NoFlute", Level: 45}
	reply, ok = h.Talk(b3cQ421, noFlute, cronos)
	b3cPage(t, "no flute", reply, ok, "30610-02.htm")

	twoFlutes := &scripttest.Player{ObjectID: 0x10000003, Name: "Two", Level: 45}
	twoFlutes.Give(3500, 1)
	twoFlutes.Give(3501, 1)
	reply, ok = h.Talk(b3cQ421, twoFlutes, cronos)
	b3cPage(t, "two flutes", reply, ok, "30610-02.htm")
	// Accepting with two flutes does not start the quest (exit_quest on a
	// CREATED state does nothing, as Mobius).
	reply, ok = h.Event(b3cQ421, "30610-06.htm", twoFlutes, cronos)
	b3cPage(t, "accept with two flutes", reply, ok, "30610-06.htm")
	b3cQuest(t, "accept with two flutes", twoFlutes, b3cQ421, scripting.StateCreated, 0)
}

// A fairy tree killed by a player (no pet needed): the tree's curse (skill
// 4243, applied as Mobius does when the killer has its effect already) and
// the twenty guardian ghosts running at the killer.
func TestB3cQ421TreeKill(t *testing.T) {
	h := b3cLoad(t, b3cQ421)
	w := h.World()
	tree := &scripttest.Npc{ObjectID: 0x60000002, ID: 27185, Name: "Fairy Tree of Wind", X: 100, Y: 200, Z: 300}
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 45, Effects: []int{4243}}

	draws := []rnd.Draw{rnd.Get(100, 29)}
	for range 20 {
		draws = append(draws, rnd.Get(65536, 0))
	}
	h.SetRand(rnd.NewScripted(t, draws...))
	h.Kill(hero, tree)
	if got := fixActions(w, "apply_effects"); len(got) != 1 || got[0].Object != tree.ObjectID || got[0].Target != hero.ObjectID || got[0].Value != 4243 {
		t.Errorf("curse: %+v", got)
	}
	ghosts := fixSpawnedOf(w, 27189)
	if len(ghosts) != 20 {
		t.Fatalf("%d ghosts, want 20", len(ghosts))
	}
	for _, g := range ghosts {
		if !g.Running || g.X != 100 || g.Y != 200 {
			t.Errorf("ghost %+v", g)
		}
		fixWantAction(t, "ghost hate", w, "hate", g.ObjectID, hero.ObjectID, 999)
		fixWantAction(t, "ghost attack", w, "attack", g.ObjectID, hero.ObjectID, 0)
	}

	// Without the effect, or on a 30 % miss, there is no curse.
	w.ClearRecords()
	hero.Effects = nil
	draws[0] = rnd.Get(100, 29)
	h.SetRand(rnd.NewScripted(t, draws...))
	h.Kill(hero, tree)
	if got := fixActions(w, "apply_effects"); len(got) != 0 {
		t.Errorf("curse without the effect: %+v", got)
	}
}
