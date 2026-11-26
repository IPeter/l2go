package game

import (
	"strings"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/skills"
)

// The second half of the area scripts' calls (Plan 13, Phase 10.7 group B,
// scripting.AreaAI on ScriptOutput): an NPC's swing, skill reuse and
// ranges, the template's skill parameters, a player's combat stance, the
// teleport lists, sight and a skill's effect types. Loop-only, while a
// handler runs or a timer fires.

var (
	_ scripting.AreaAI            = (*ScriptOutput)(nil)
	_ scripting.NpcTemplateSkills = npcRef{}
)

// NpcIsAttackingNow is Mobius isAttackingNow.
func (o *ScriptOutput) NpcIsAttackingNow(npc scripting.NpcRef) bool {
	s, n, ok := o.anyNpc(npc)
	return ok && s.AttackingNow(n)
}

// NpcIsSkillDisabled is Mobius isSkillDisabled; a skill the data lacks is
// disabled.
func (o *ScriptOutput) NpcIsSkillDisabled(npc scripting.NpcRef, skill scripting.Skill) bool {
	s, n, ok := o.anyNpc(npc)
	if !ok {
		return true
	}
	sk, found := s.Skill(skill.ID, skill.Level)
	return !found || s.SkillDisabled(n, sk)
}

// NpcAggroRange is Mobius getAggroRange.
func (o *ScriptOutput) NpcAggroRange(npc scripting.NpcRef) int {
	if s, n, ok := o.anyNpc(npc); ok {
		return s.npcAggroRange(n)
	}
	return 0
}

// NpcClanHelpRange is Mobius getTemplate().getClanHelpRange().
func (o *ScriptOutput) NpcClanHelpRange(npc scripting.NpcRef) int {
	if n := npcOfRef(npc); n != nil {
		return n.Template.AI.ClanHelpRange
	}
	return 0
}

// PlayerIsInCombat is Mobius isInCombat for a player.
func (o *ScriptOutput) PlayerIsInCombat(p scripting.PlayerRef) bool {
	s, pl := o.srv.Load(), playerOfRef(p)
	return s != nil && pl != nil && s.InCombat(pl)
}

// ShowTeleportList is Mobius TeleportHolder.showTeleportList for the NPC's
// list of that name; false (and nothing shown) when there is none.
func (o *ScriptOutput) ShowTeleportList(p scripting.PlayerRef, npc scripting.NpcRef, list string) bool {
	s, n, ok := o.npc(npc)
	pl := playerOfRef(p)
	if !ok || pl == nil || s.worldData == nil {
		return false
	}
	if _, found := s.worldData.TeleportList(n.Template.ID, list); !found {
		return false
	}
	s.showTeleports(pl, n, list)
	return true
}

// CanSeeTarget is Mobius GeoEngine.canSeeTarget: doors, fences and
// geodata.
func (o *ScriptOutput) CanSeeTarget(from, to any) bool {
	s := o.srv.Load()
	a, b := creatureOfRef(from), creatureOfRef(to)
	return s != nil && a != nil && b != nil && s.canSee(a, b)
}

// SkillHasEffectType is Mobius skill.hasEffectType by the type's name;
// false for an unknown skill or type.
func (o *ScriptOutput) SkillHasEffectType(skill scripting.Skill, effectType string) bool {
	s := o.srv.Load()
	if s == nil {
		return false
	}
	sk, ok := s.Skill(skill.ID, skill.Level)
	if !ok {
		return false
	}
	for t := skills.EffectType(0); !strings.HasPrefix(t.String(), "EffectType("); t++ {
		if t.String() == effectType {
			return sk.HasEffectType(t)
		}
	}
	return false
}

// TemplateSkill is a skill of the template parameters (Mobius
// getParameters().getObject(name, SkillHolder.class)).
func (r npcRef) TemplateSkill(name string) (scripting.Skill, bool) {
	sk, ok := r.n.Template.Parameters.Skills[name]
	if !ok {
		return scripting.Skill{}, false
	}
	return scripting.Skill{ID: sk.ID, Level: sk.Level}, true
}
