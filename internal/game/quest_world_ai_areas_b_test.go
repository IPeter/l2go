package game

import (
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/scripting"
)

// The second half of the area scripts' calls on the game: an NPC's swing,
// skill reuse and ranges, template skills, a player's combat stance, sight,
// teleport lists and skill effect types.
func TestScriptAreaCalls(t *testing.T) {
	start := npcTestStart(t)
	wd := aiTestWorld(start)
	for i := range wd.Npcs {
		if wd.Npcs[i].ID == aiKeltir {
			wd.Npcs[i].Parameters.Skills = map[string]data.NpcSkill{"PhysicalSpecial1": {ID: 4001, Level: 2}}
		}
	}
	e := newAIEnv(t, wd, newAIRand(), nil, false)
	_, p := statusPlayer(t, e, "alice", "Alice")
	k := aiSpawn(e, aiKeltir, start, 100, 0)
	o := scriptOutputOf(e)
	ref := k.ScriptRef()

	var attacking, unknownDisabled, inCombat, sees, list, effect bool
	var aggro, wantAggro, help int
	e.onLoop(func() {
		attacking = o.NpcIsAttackingNow(ref)
		unknownDisabled = o.NpcIsSkillDisabled(ref, scripting.Skill{ID: 999999, Level: 1})
		aggro, wantAggro = o.NpcAggroRange(ref), e.srv.npcAggroRange(k)
		help = o.NpcClanHelpRange(ref)
		inCombat = o.PlayerIsInCombat(p)
		sees = o.CanSeeTarget(p, ref)
		list = o.ShowTeleportList(p, ref, "list9")
		effect = o.SkillHasEffectType(scripting.Skill{ID: 999999, Level: 1}, "AGGRESSION")
	})
	if attacking || !unknownDisabled || inCombat || !sees || list || effect {
		t.Errorf("attacking %v, unknown skill disabled %v, in combat %v, sees %v, unknown list shown %v, unknown skill has AGGRESSION %v; want false, true, false, true, false, false",
			attacking, unknownDisabled, inCombat, sees, list, effect)
	}
	if aggro != wantAggro || help != k.Template.AI.ClanHelpRange {
		t.Errorf("aggro range %d, clan help range %d; want %d, %d", aggro, help, wantAggro, k.Template.AI.ClanHelpRange)
	}

	ts := ref.(scripting.NpcTemplateSkills)
	sk, ok := ts.TemplateSkill("PhysicalSpecial1")
	_, missing := ts.TemplateSkill("PhysicalSpecial2")
	if !ok || sk != (scripting.Skill{ID: 4001, Level: 2}) || missing {
		t.Errorf("template skill %+v (%v), missing one found %v; want 4001 level 2", sk, ok, missing)
	}
}
