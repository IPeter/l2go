package scripting_test

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The rest of the scripted monsters of Plan 13 group A (Mobius ai/others),
// on the fake world.

// actionsOf returns the recorded actions of one kind.
func actionsOf(w *scripttest.World, kind string) []scripttest.Action {
	var out []scripttest.Action
	for _, a := range w.Actions {
		if a.Kind == kind {
			out = append(out, a)
		}
	}
	return out
}

func TestFairyTreesScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/monsters/fairy_trees.lua")
	tree := &scripttest.Npc{ObjectID: 2, ID: 27185, X: 1000, Y: 1000, Z: -100}
	h.Spawn(tree)
	if !tree.NoRandomWalk || !tree.Immobilized {
		t.Fatalf("tree = %+v, want it standing still", tree)
	}

	// Killed from farther than 1,500: no guardians.
	far := &scripttest.Player{ObjectID: 1, X: 2501, Y: 1000, Z: -100}
	h.Kill(far, tree)
	if n := len(h.World().Spawned); n != 0 {
		t.Fatalf("%d guardians for a kill from afar", n)
	}

	// From close: 20 guardians go for the killer; a coin flip each poisons.
	p := &scripttest.Player{ObjectID: 3, X: 1100, Y: 1000, Z: -100}
	draws := make([]rnd.Draw, 20)
	for i := range draws {
		draws[i] = rnd.Bool(i%4 == 0)
	}
	h.SetRand(rnd.NewScripted(t, draws...))
	h.Kill(p, tree)
	spawned := h.World().Spawned
	if len(spawned) != 20 {
		t.Fatalf("%d guardians, want 20", len(spawned))
	}
	for _, g := range spawned {
		if g.ID != 27189 || g.X != 1000 || g.Y != 1000 || g.Z != -100 || !g.Running || g.Intention != "ATTACK" {
			t.Fatalf("guardian = %+v", g)
		}
		if len(g.Hate) != 1 || g.Hate[0].Target != p || g.Hate[0].Hate != 999 {
			t.Fatalf("guardian hate = %+v, want 999 for the killer", g.Hate)
		}
	}
	casts := actionsOf(h.World(), "cast")
	if len(casts) != 5 || len(actionsOf(h.World(), "set_target")) != 5 {
		t.Fatalf("casts = %+v, want 5", casts)
	}
	if c := casts[0]; c.Object != spawned[0].ObjectID || c.Value != 4243 || c.Level != 1 {
		t.Errorf("cast = %+v, want Venomous Poison by the first guardian", c)
	}
}

func TestGordonScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/monsters/gordon.lua")
	gordon := &scripttest.Npc{ObjectID: 2, ID: 29095}
	h.Spawn(gordon)
	if !gordon.NoReturn {
		t.Fatalf("Gordon may still walk back to his spawn point")
	}
	h.World().ClearRecords()

	p := &scripttest.Player{ObjectID: 1}
	h.CreatureSee(gordon, p)
	h.CreatureSee(gordon, &scripttest.Npc{ObjectID: 3, ID: 20001})
	if len(h.World().Actions) != 0 {
		t.Fatalf("Gordon went for %+v without a cursed weapon", h.World().Actions)
	}
	p.CursedWeapon = true
	h.CreatureSee(gordon, p)
	want := []scripttest.Action{
		{Kind: "hate", Object: 2, Target: 1, Value: 999},
		{Kind: "run", Object: 2},
		{Kind: "attack", Object: 2, Target: 1},
	}
	if got := h.World().Actions; !slices.Equal(got, want) {
		t.Errorf("actions =\n%+v\nwant\n%+v", got, want)
	}
}

func TestNonLethalableNpcsScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/monsters/non_lethalable_npcs.lua")
	gustav := &scripttest.Npc{ObjectID: 2, ID: 35410}
	h.Spawn(gustav)
	other := &scripttest.Npc{ObjectID: 3, ID: 20001}
	h.Spawn(other)
	if !gustav.NotLethalable || other.NotLethalable {
		t.Errorf("not lethalable: Gustav %v, another NPC %v; want true, false", gustav.NotLethalable, other.NotLethalable)
	}
}

func TestPolymorphingAngelScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/monsters/polymorphing_angel.lua")
	p := &scripttest.Player{ObjectID: 1}
	angel := &scripttest.Npc{ObjectID: 2, ID: 21067, X: 500, Y: 600, Z: 70, Dead: true}
	h.Kill(p, angel)
	spawned := h.World().Spawned
	if len(spawned) != 1 {
		t.Fatalf("%d spawned, want 1", len(spawned))
	}
	if a := spawned[0]; a.ID != 21068 || a.X != 500 || a.Y != 600 || a.Z != 70 || !a.Running {
		t.Errorf("new angel = %+v, want 21068 running where the other died", a)
	}
}

func TestPolymorphingOnAttackScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/monsters/polymorphing_on_attack.lua")
	p := &scripttest.Player{ObjectID: 1}
	ol := &scripttest.Npc{ObjectID: 2, ID: 21261, X: 100, Y: 200, Z: 300, Heading: 4000, HP: 900, MaxHP: 1000}

	// 20 %: the roll fails.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 20)))
	h.Attack(p, ol, 10)
	if len(h.World().Actions) != 0 {
		t.Fatalf("a failed roll did %+v", h.World().Actions)
	}

	// It passes: a line of the first group, then the next stage 20 higher,
	// running at the attacker.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 19), rnd.Get(3, 1)))
	h.Attack(p, ol, 10)
	spawned := h.World().Spawned
	if len(spawned) != 1 {
		t.Fatalf("%d spawned, want 1", len(spawned))
	}
	next := spawned[0]
	if next.ID != 21262 || next.X != 100 || next.Y != 200 || next.Z != 320 || next.Heading != 4000 {
		t.Errorf("next stage = %+v", next)
	}
	want := []scripttest.Action{
		{Kind: "delete", Object: 2},
		{Kind: "run", Object: next.ObjectID},
		{Kind: "hate", Object: next.ObjectID, Target: 1, Value: 500},
		{Kind: "attack", Object: next.ObjectID, Target: 1},
	}
	got := h.World().Actions
	if len(got) != 5 || got[0].Kind != "say" || got[0].Text != "You idiot! I've just been toying with you!" || !slices.Equal(got[1:], want) {
		t.Errorf("actions = %+v", got)
	}
	if !ol.Deleted {
		t.Errorf("the first stage is still there")
	}
	h.World().ClearRecords()

	// Above its HP threshold (66 %) a Cave Ant does not even roll; the
	// Sharp Talon Tiger's shaman says nothing; a dead monster stays dead.
	ant := &scripttest.Npc{ObjectID: 3, ID: 21271, HP: 700, MaxHP: 1000}
	h.Attack(p, ant, 10)
	shaman := &scripttest.Npc{ObjectID: 4, ID: 21258, HP: 1000, MaxHP: 1000}
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 99)))
	h.Attack(p, shaman, 10)
	if len(actionsOf(h.World(), "say")) != 0 || len(actionsOf(h.World(), "delete")) != 1 {
		t.Errorf("actions = %+v, want the shaman deleted without a word", h.World().Actions)
	}
	dead := &scripttest.Npc{ObjectID: 5, ID: 21258, Dead: true, MaxHP: 1000}
	h.Attack(p, dead, 10)
	if s := h.World().Spawned; len(s) != 1 || s[0].ID != 21259 {
		t.Errorf("spawned %+v, want only the tiger (none for the ant or the dead)", s)
	}
}

func TestSeeThroughSilentMoveScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/monsters/see_through_silent_move.lua")
	mob := &scripttest.Npc{ObjectID: 2, ID: 22199, Types: []string{"Monster", "Attackable"}}
	h.Spawn(mob)
	folk := &scripttest.Npc{ObjectID: 3, ID: 18001, Types: []string{"Folk"}}
	h.Spawn(folk)
	if !mob.SeeThroughSilentMove || folk.SeeThroughSilentMove {
		t.Errorf("see through: monster %v, folk %v; want true, false", mob.SeeThroughSilentMove, folk.SeeThroughSilentMove)
	}
}

func TestTimakOrcTroopLeaderScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/monsters/timak_orc_troop_leader.lua")
	p := &scripttest.Player{ObjectID: 1}
	leader := &scripttest.Npc{
		ObjectID: 2, ID: 20767, Types: []string{"Monster"},
		Params: map[string]string{"SummonPrivateRate": "50"},
		ParamMinions: map[string][]scripting.TemplateMinion{"Privates": {
			{ID: 20768, Count: 1}, {ID: 20769, Count: 1}, {ID: 20770, Count: 1},
		}},
	}
	first := &scripttest.Npc{ObjectID: 3, ID: 20768, Leader: leader}
	leader.Minions = []*scripttest.Npc{first}
	h.World().Add(first)

	// 51 > 50: nothing.
	h.SetRand(rnd.NewScripted(t, rnd.Between(1, 100, 51)))
	h.Attack(p, leader, 10)
	if len(leader.Minions) != 1 {
		t.Fatalf("a failed roll called a private")
	}

	// The first missing private comes, with a shout.
	h.SetRand(rnd.NewScripted(t, rnd.Between(1, 100, 50), rnd.Get(4, 2)))
	h.Attack(p, leader, 10)
	if len(leader.Minions) != 2 || leader.Minions[1].ID != 20769 {
		t.Fatalf("minions = %+v, want 20769 added", leader.Minions)
	}
	if said := actionsOf(h.World(), "say"); len(said) != 1 || said[0].Text != "Destroy the enemy, my brothers!" {
		t.Errorf("said %+v", said)
	}

	// Three alive: no more.
	h.SetRand(rnd.NewScripted(t, rnd.Between(1, 100, 50), rnd.Get(4, 0)))
	h.Attack(p, leader, 10)
	h.SetRand(rnd.NewScripted(t, rnd.Between(1, 100, 1)))
	h.Attack(p, leader, 10)
	if len(leader.Minions) != 3 || len(actionsOf(h.World(), "say")) != 2 {
		t.Errorf("minions %d, shouts %d; want 3 and 2", len(leader.Minions), len(actionsOf(h.World(), "say")))
	}

	// Not a monster: no roll.
	h.Attack(p, &scripttest.Npc{ObjectID: 9, ID: 20767}, 10)
}

func TestTownPetsScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/monsters/town_pets.lua")
	home := scripting.Location{X: 1000, Y: 1000}
	pet := &scripttest.Npc{ObjectID: 2, ID: 31202, X: 1000, Y: 1000, Spawn: &home}
	h.Spawn(pet)
	if !pet.Running {
		t.Fatalf("the pet walks")
	}

	// Every 5 s, a point up to 50 from home.
	h.SetRand(rnd.NewScripted(t, rnd.Between(-50, 50, 30), rnd.Between(-50, 50, -40)))
	h.Advance(5 * time.Second)
	if want := (scripting.Location{X: 1030, Y: 960}); pet.Dest == nil || *pet.Dest != want {
		t.Fatalf("dest = %v, want %v", pet.Dest, want)
	}
	h.Arrive(pet)

	// Within 20 of where it stands: it stays.
	h.SetRand(rnd.NewScripted(t, rnd.Between(-50, 50, 25), rnd.Between(-50, 50, -35)))
	h.Advance(5 * time.Second)
	if pet.Dest != nil {
		t.Fatalf("the pet moved %v for a short step", pet.Dest)
	}

	// Nobody near: it looks again after 15 s, without moving.
	pet.InactiveRegion = true
	h.Advance(5 * time.Second)
	pet.InactiveRegion = false
	h.Advance(15*time.Second - time.Millisecond)
	h.SetRand(rnd.NewScripted(t, rnd.Between(-50, 50, -50), rnd.Between(-50, 50, 50)))
	h.Advance(time.Millisecond)
	if want := (scripting.Location{X: 950, Y: 1050}); pet.Dest == nil || *pet.Dest != want {
		t.Fatalf("dest = %v, want %v", pet.Dest, want)
	}
	h.Arrive(pet)

	// A reload drops the timers; on_start starts them again.
	if err := h.Reload(); err != nil {
		t.Fatal(err)
	}
	h.Start()
	h.SetRand(rnd.NewScripted(t, rnd.Between(-50, 50, 0), rnd.Between(-50, 50, 0)))
	h.Advance(5 * time.Second)
	if want := home; pet.Dest == nil || *pet.Dest != want {
		t.Errorf("after a reload: dest = %v, want %v", pet.Dest, want)
	}
}

func TestVarkaKetraScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/monsters/varka_ketra.lua")
	ally := &scripttest.Player{ObjectID: 1, X: 1000, Y: 1000}
	ally.Give(7213, 1) // Ketra alliance level 3
	plain := &scripttest.Player{ObjectID: 2}
	ketra := &scripttest.Npc{ObjectID: 3, ID: 21324, X: 1100, Y: 1000}
	varka := &scripttest.Npc{ObjectID: 4, ID: 21350, X: 1100, Y: 1000}

	// Each side leaves its own allies alone.
	if h.NpcHate(ketra, ally) || !h.NpcHate(varka, ally) || !h.NpcHate(ketra, plain) {
		t.Fatalf("hate: Ketra at its ally, Varka at Ketra's ally, Ketra at a stranger; want false, true, true")
	}

	// Killing a Varka monster costs a Ketra ally nothing.
	h.Kill(ally, varka)
	if ally.Items[7213] != 1 {
		t.Fatalf("items = %v after a Varka kill", ally.Items)
	}

	// Killing a Ketra monster: one level down, and the Ketra quests go.
	for _, q := range []string{"Q00605_AllianceWithKetraOrcs", "Q00607_ProveYourCourageKetra"} {
		st, _ := h.World().NewQuestState(ally.Ref(), q).(*scripttest.QuestState)
		st.SetState(scripting.StateStarted)
	}
	h.World().NewQuestState(ally.Ref(), "Q00608_SlayTheEnemyCommanderKetra") // created only
	h.Kill(ally, ketra)
	if ally.Items[7213] != 0 || ally.Items[7212] != 1 {
		t.Errorf("items = %v, want the level 2 mark instead of level 3", ally.Items)
	}
	if got := ally.Quests(); !slices.Equal(got, []string{"Q00608_SlayTheEnemyCommanderKetra"}) {
		t.Errorf("quests = %v, want only the one not started", got)
	}

	// Out of range (1,500): nothing.
	ally.X = 3000
	h.Kill(ally, ketra)
	if ally.Items[7212] != 1 {
		t.Errorf("items = %v after a kill from afar", ally.Items)
	}

	// The level 1 mark is taken.
	ally.X = 1000
	ally.Items = map[int]int64{7211: 1}
	h.Kill(ally, ketra)
	if ally.Items[7211] != 0 || len(h.World().Given) != 1 {
		t.Errorf("items = %v, given %v; want the mark gone", ally.Items, h.World().Given)
	}
}

func TestWastedCorpseScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/monsters/wasted_corpse.lua")
	corpse := &scripttest.Npc{ObjectID: 2, ID: 18119, HP: 100, MaxHP: 100}
	h.Spawn(corpse)
	if !corpse.Dead {
		t.Errorf("the corpse is alive")
	}
}
