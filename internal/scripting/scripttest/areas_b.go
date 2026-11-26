package scripttest

import (
	"slices"

	"github.com/IPeter/l2go/internal/scripting"
)

// The fake world of the second half of the area scripts (Plan 13, Phase
// 10.7 group B, scripting.AreaAI): an NPC's swing, skill reuse and ranges
// (Npc fields), the template's skill parameters, a player's combat stance
// (Player.InCombat), teleport lists, sight and skill effect types.

var (
	_ scripting.AreaAI            = (*World)(nil)
	_ scripting.NpcTemplateSkills = npcRef{}
)

// areasBWorld is the World's state of these calls.
type areasBWorld struct {
	sight       func(from, to any) bool
	effectTypes map[int][]string
	teleports   map[int][]string
}

// SetSight makes world.can_see_target answer with fn, which gets the fake
// *Player or *Npc on each side; without it everyone sees everyone.
func (w *World) SetSight(fn func(from, to any) bool) { w.areasB.sight = fn }

// SetSkillEffectTypes gives a skill its effect types (skill:has_effect_type);
// other skills have none.
func (w *World) SetSkillEffectTypes(skillID int, types ...string) {
	if w.areasB.effectTypes == nil {
		w.areasB.effectTypes = make(map[int][]string)
	}
	w.areasB.effectTypes[skillID] = types
}

// AddTeleportList gives NPC template npcID teleport lists of those names.
func (w *World) AddTeleportList(npcID int, lists ...string) {
	if w.areasB.teleports == nil {
		w.areasB.teleports = make(map[int][]string)
	}
	w.areasB.teleports[npcID] = append(w.areasB.teleports[npcID], lists...)
}

// NpcIsAttackingNow reads AttackingNow.
func (w *World) NpcIsAttackingNow(npc scripting.NpcRef) bool {
	return npcFlag(npc, func(n *Npc) bool { return n.AttackingNow })
}

// NpcIsSkillDisabled looks in DisabledSkills.
func (w *World) NpcIsSkillDisabled(npc scripting.NpcRef, skill scripting.Skill) bool {
	return npcFlag(npc, func(n *Npc) bool { return slices.Contains(n.DisabledSkills, skill.ID) })
}

// NpcAggroRange reads AggroRange.
func (w *World) NpcAggroRange(npc scripting.NpcRef) int {
	if n := npcOf(npc); n != nil {
		return n.AggroRange
	}
	return 0
}

// NpcClanHelpRange reads ClanHelpRange.
func (w *World) NpcClanHelpRange(npc scripting.NpcRef) int {
	if n := npcOf(npc); n != nil {
		return n.ClanHelpRange
	}
	return 0
}

// PlayerIsInCombat reads InCombat.
func (w *World) PlayerIsInCombat(p scripting.PlayerRef) bool {
	pl := playerOf(p)
	return pl != nil && pl.InCombat
}

// ShowTeleportList records the list (action "teleport_list", Object the NPC,
// Target the player, Text the list) when the NPC's template has it.
func (w *World) ShowTeleportList(p scripting.PlayerRef, npc scripting.NpcRef, list string) bool {
	if !slices.Contains(w.areasB.teleports[npc.ID()], list) {
		return false
	}
	w.act(Action{Kind: "teleport_list", Object: npc.ObjectID(), Target: p.ObjectID(), Text: list})
	return true
}

// CanSeeTarget is the SetSight function, else true.
func (w *World) CanSeeTarget(from, to any) bool {
	if fn := w.areasB.sight; fn != nil {
		return fn(fakeOf(from), fakeOf(to))
	}
	return true
}

// SkillHasEffectType looks in the SetSkillEffectTypes types.
func (w *World) SkillHasEffectType(skill scripting.Skill, effectType string) bool {
	return slices.Contains(w.areasB.effectTypes[skill.ID], effectType)
}

// TemplateSkill reads ParamSkills.
func (r npcRef) TemplateSkill(name string) (scripting.Skill, bool) {
	sk, ok := r.n.ParamSkills[name]
	return sk, ok
}
