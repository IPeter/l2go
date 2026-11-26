package game

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/world"
)

// The area scripts' calls on the game (Plan 13, Phase 10.7): skill kinds,
// effect levels, removing an NPC's effects, the NPC cast check, and the
// zones as scripts query them.

func TestScriptAreasSkillsAndEffects(t *testing.T) {
	e := newCastEnv(t)
	_, p := castEnter(t, e, "newbie", classHumanFighter)
	keltir := npcSpawnedOne(t, e, attackKeltir)
	o := scriptOutputOf(e)
	ref := keltir.ScriptRef()

	var magic, physical, unknownMagic bool
	var level int
	var has, hasUnknown, npcHas, npcGone, castUnknown bool
	e.onLoop(func() {
		magic, physical = o.SkillIsMagic(scripting.Skill{ID: 1177, Level: 1}), o.SkillIsMagic(scripting.Skill{ID: 3, Level: 1})
		unknownMagic = o.SkillIsMagic(scripting.Skill{ID: 1177, Level: 999})
		o.ApplyEffects(scripting.Skill{ID: 4322, Level: 1}, ref, p)
		level, has = o.EffectLevel(p, 4322)
		_, hasUnknown = o.EffectLevel(p, 4323)
		o.ApplyEffects(scripting.Skill{ID: 4322, Level: 1}, ref, ref)
		npcHas = o.IsAffectedBySkill(ref, 4322)
		o.NpcStopSkillEffects(ref, 4322)
		npcGone = !o.IsAffectedBySkill(ref, 4322)
		castUnknown = o.NpcCheckDoCastConditions(ref, scripting.Skill{ID: 1177, Level: 999})
	})
	if !magic || physical || unknownMagic {
		t.Errorf("is_magic: Wind Strike %v, Power Strike %v, an unknown level %v; want true, false, false", magic, physical, unknownMagic)
	}
	if !has || level != 1 || hasUnknown {
		t.Errorf("effect_level: %d (%v), without the effect %v; want 1 (true), false", level, has, hasUnknown)
	}
	if !npcHas || !npcGone {
		t.Errorf("the NPC had the effect %v, lost it after stop_skill_effects %v; want true, true", npcHas, npcGone)
	}
	if castUnknown {
		t.Error("check_do_cast_conditions passed an unknown skill level")
	}
}

// Quest 620 (Four Goblets) is offered once the Four Sepulchers run.
func TestScriptAreasSepulchersFeature(t *testing.T) {
	if !questAvailable(scripting.QuestInfo{Requires: []string{"sepulchers"}}) {
		t.Error("a quest requiring the sepulchers is not offered")
	}
}

func TestScriptAreasZones(t *testing.T) {
	start := npcTestStart(t)
	wd := aiTestWorld(start)
	wd.ZoneTypes = append(wd.ZoneTypes, data.MobiusClass{Name: "EffectZone", Parent: "ZoneType", Usable: true})
	wd.Zones = append(wd.Zones, data.Zone{
		ID: 300002, Name: "far_weakness", Type: "EffectZone", File: "effect.xml",
		Shape:  data.Shape{Type: data.ShapeCuboid, MinZ: -4779, MaxZ: -3779, Points: []data.Point{{X: 100400, Y: 100400}, {X: 100600, Y: 100600}}},
		Params: []data.ZoneParam{{Name: "skillIdLvl", Value: "4146-3;"}, {Name: "default_enabled", Value: "false"}},
	})
	e := newAIEnv(t, wd, newAIRand(), nil, false)
	k := aiSpawn(e, aiKeltir, start, 100, 0)
	var wet *Npc
	e.onLoop(func() {
		var err error
		wet, err = e.srv.SpawnNpc(aiWolf, world.Position{X: 100500, Y: 100500, Z: -4279})
		if err != nil {
			t.Errorf("SpawnNpc: %v", err)
		}
	})
	o := scriptOutputOf(e)

	var inside []scripting.NpcRef
	var at, nowhere []scripting.Zone
	var param, missing string
	var hasParam, hasMissing, before, after bool
	e.onLoop(func() {
		inside = o.ZoneNpcs(300001)
		at = o.ZonesAt(scripting.Location{X: 100500, Y: 100500, Z: -4279})
		nowhere = o.ZonesAt(scripting.Location{X: int(start.X), Y: int(start.Y), Z: int(start.Z)})
		param, hasParam = o.ZoneParam(300002, "skillIdLvl")
		missing, hasMissing = o.ZoneParam(300001, "skillIdLvl")
		i := e.srv.dialog.zones.byID[300002]
		before = e.srv.dialog.zones.rules[i].enabled
		o.SetZoneEnabled(300002, true)
		after = e.srv.dialog.zones.rules[i].enabled
		o.ZoneOustAll(300001) // nobody inside: nothing happens
	})
	if len(inside) != 1 || inside[0].ObjectID() != int32(wet.ID()) || slices.ContainsFunc(inside, func(n scripting.NpcRef) bool { return n.ObjectID() == int32(k.ID()) }) {
		t.Errorf("NPCs in the water: %v; want only the wolf %d", inside, wet.ID())
	}
	want := []scripting.Zone{{ID: 300001, Type: "WaterZone"}, {ID: 300002, Type: "EffectZone"}}
	if !slices.Equal(at, want) || len(nowhere) != 0 {
		t.Errorf("zones at the wolf %v, at the start %v; want %v, none", at, nowhere, want)
	}
	if !hasParam || param != "4146-3;" || hasMissing || missing != "" {
		t.Errorf("skillIdLvl %q (%v), of the water %q (%v); want \"4146-3;\", none", param, hasParam, missing, hasMissing)
	}
	if before || !after {
		t.Errorf("the effect zone enabled %v, after set_enabled(true) %v; want false, true", before, after)
	}
}
