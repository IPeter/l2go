package scripting_test

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The scripted monsters of Plan 13 group A, end to end on the fake world.

func TestChestsScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/monsters/chests.lua")
	p := &scripttest.Player{ObjectID: 1, Name: "Hero"}
	box := &scripttest.Npc{ObjectID: 2, ID: 18265, Types: []string{"Monster", "Chest"}, Box: true}
	mimic := &scripttest.Npc{ObjectID: 3, ID: 18266, Types: []string{"Monster", "Chest"}}

	// The first hit blows a box up, once.
	h.Attack(p, box, 10)
	h.Attack(p, box, 10)
	if !box.Bombed || !box.Interacted {
		t.Errorf("box = %+v, want bombed", box)
	}
	bombs := 0
	for _, a := range h.World().Actions {
		if a.Kind == "treasure_bomb" {
			bombs++
		}
	}
	if bombs != 1 {
		t.Errorf("%d bombs, want 1", bombs)
	}
	// A mimic fights back instead.
	h.Attack(p, mimic, 10)
	if mimic.Bombed || mimic.Interacted {
		t.Errorf("mimic = %+v, want untouched", mimic)
	}

	// Skills: only an offensive skill aimed at the box itself.
	other := &scripttest.Npc{ObjectID: 4, ID: 18267, Types: []string{"Monster", "Chest"}, Box: true}
	h.World().SetSkillTargetType(27, "UNLOCKABLE")
	h.SkillSee(p, other, scripting.Skill{ID: 1177, Level: 1}, p)
	h.SkillSee(p, other, scripting.Skill{ID: 27, Level: 1}, other)
	if other.Bombed {
		t.Fatalf("a skill at someone else or an unlock skill blew the box up")
	}
	h.SkillSee(p, other, scripting.Skill{ID: 1177, Level: 1}, p, other)
	if !other.Bombed {
		t.Errorf("an offensive skill at the box did not blow it up")
	}
	// Not a chest: nothing.
	plain := &scripttest.Npc{ObjectID: 5, ID: 18268, Types: []string{"Monster"}, Box: true}
	h.Attack(p, plain, 10)
	if plain.Bombed {
		t.Errorf("a non-chest NPC exploded")
	}
}

func TestFleeMonstersScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/monsters/flee_monsters.lua")
	p := &scripttest.Player{ObjectID: 1, X: 900, Y: 1000}
	rabbit := &scripttest.Npc{ObjectID: 2, ID: 20002, X: 1000, Y: 1000, Z: -50}

	h.Attack(p, rabbit, 5)
	if !rabbit.CoreAIDisabled || !rabbit.Running {
		t.Errorf("rabbit = %+v, want its AI off and running", rabbit)
	}
	if want := (scripting.Location{X: 1500, Y: 1000, Z: -50}); rabbit.Dest == nil || *rabbit.Dest != want {
		t.Errorf("dest = %v, want %v (500 away from the attacker)", rabbit.Dest, want)
	}

	// From the summon when the summon hit; geodata clips the run.
	p.Summon = &scripttest.Npc{ObjectID: 3, ID: 12077, X: 1000, Y: 900}
	h.World().Add(p.Summon)
	h.World().SetGeo(func(from, to scripting.Location) scripting.Location {
		to.Y = (from.Y + to.Y) / 2
		return to
	})
	h.AttackWith(p, rabbit, 5, true, nil)
	if want := (scripting.Location{X: 1000, Y: 1250, Z: -50}); rabbit.Dest == nil || *rabbit.Dest != want {
		t.Errorf("dest = %v, want %v (away from the summon, clipped)", rabbit.Dest, want)
	}
}

func TestTurekOrcsScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/monsters/turek_orcs.lua")
	p := &scripttest.Player{ObjectID: 1, Name: "Hero", HP: 500, MaxHP: 1000, X: 1050, Y: 1000}
	spawn := scripting.Location{X: 1000, Y: 1000}
	orc := &scripttest.Npc{
		ObjectID: 2, ID: 20495, Name: "Turek Orc Warlord", X: 1000, Y: 1000, HP: 400, MaxHP: 1000,
		Spawn: &spawn, AIData: map[string]int{"fleeX": 2000, "fleeY": 2000, "fleeZ": 0},
	}
	helper := &scripttest.Npc{ObjectID: 3, ID: 20499, X: 2100, Y: 2000, HP: 900, MaxHP: 900}
	far := &scripttest.Npc{ObjectID: 4, ID: 20499, X: 2500, Y: 2000, HP: 900, MaxHP: 900}
	h.World().Add(helper, far)

	// The first hit only marks the orc; the second makes it flee (10 %).
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 5), rnd.Get(21, 2)))
	h.Attack(p, orc, 10)
	if len(h.World().Actions) != 0 {
		t.Fatalf("first hit did %+v", h.World().Actions)
	}
	h.Attack(p, orc, 10)
	want := []scripttest.Action{
		{Kind: "say", Object: 2, Value: 0, Text: "Retreat!"},
		{Kind: "core_ai", Object: 2, Value: 1},
		{Kind: "run", Object: 2},
		{Kind: "move", Object: 2, Loc: scripting.Location{X: 2000, Y: 2000}},
	}
	if got := h.World().Actions; !slices.Equal(got, want) {
		t.Fatalf("flee =\n%+v\nwant\n%+v", got, want)
	}
	h.World().ClearRecords()

	// At the flee point it warns the orcs within 400; they attack the
	// player after the orc's handler.
	h.Arrive(orc)
	want = []scripttest.Action{
		{Kind: "core_ai", Object: 2, Value: 0},
		{Kind: "run", Object: 3},
		{Kind: "hate", Object: 3, Target: 1, Value: 99999},
		{Kind: "attack", Object: 3, Target: 1},
	}
	if got := h.World().Actions; !slices.Equal(got, want) {
		t.Fatalf("warning =\n%+v\nwant\n%+v", got, want)
	}
	if helper.Intention != "ATTACK" || far.Intention != "" || h.Engine().QueuedEvents() != 0 {
		t.Errorf("helper %q, far %q, queued %d", helper.Intention, far.Intention, h.Engine().QueuedEvents())
	}
	h.World().ClearRecords()

	// 15 s later, healed above 70 %, it goes home and its AI comes back.
	orc.HP = 800
	h.Advance(15*time.Second - time.Millisecond)
	if len(h.World().Actions) != 0 {
		t.Fatalf("check before 15 s: %+v", h.World().Actions)
	}
	h.Advance(time.Millisecond)
	if orc.Dest == nil || *orc.Dest != spawn {
		t.Fatalf("after the check: dest %v, want home %v", orc.Dest, spawn)
	}
	h.Arrive(orc)
	if orc.CoreAIDisabled {
		t.Errorf("the orc's AI is still off at home")
	}
	// The state is gone: hit twice more, it may flee again.
	orc.HP = 400
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 50)))
	h.Attack(p, orc, 10)
}
