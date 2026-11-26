package game

import (
	"cmp"
	"slices"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/skills"
)

// The calls of the area scripts (Plan 13, Phase 10.7) on ScriptOutput:
// skill kinds, NPC cast checks and effect removal, effect levels, and the
// zones as scripts query them. Loop-only, like the rest of ScriptOutput.

var (
	_ scripting.SkillKinds   = (*ScriptOutput)(nil)
	_ scripting.NpcCasting   = (*ScriptOutput)(nil)
	_ scripting.EffectLevels = (*ScriptOutput)(nil)
	_ scripting.ZoneData     = (*ScriptOutput)(nil)
	_ scripting.PlayerWeight = (*Player)(nil)
)

// SkillIsMagic is Mobius Skill.isMagic; false for an unknown skill.
func (o *ScriptOutput) SkillIsMagic(skill scripting.Skill) bool {
	s := o.srv.Load()
	if s == nil {
		return false
	}
	sk, ok := s.Skill(skill.ID, skill.Level)
	return ok && sk.IsMagic()
}

// NpcCheckDoCastConditions is Mobius checkDoCastConditions for an NPC.
func (o *ScriptOutput) NpcCheckDoCastConditions(npc scripting.NpcRef, skill scripting.Skill) bool {
	s, n, ok := o.npc(npc)
	if !ok || n.Dead {
		return false
	}
	sk, found := s.Skill(skill.ID, skill.Level)
	b := s.skillBaseOf(n)
	if !found || b == nil {
		return false
	}
	return s.checkDoCastConditions(n, b, sk)
}

// NpcStopSkillEffects is Mobius stopSkillEffects(REMOVED, skillId).
func (o *ScriptOutput) NpcStopSkillEffects(npc scripting.NpcRef, skillID int) {
	if _, n, ok := o.npc(npc); ok && n.skill != nil {
		n.skill.el.StopSkill(skills.FinishRemoved, skillID)
	}
}

// EffectLevel is the level of the skill's effect on a player or NPC.
func (o *ScriptOutput) EffectLevel(obj any, skillID int) (int, bool) {
	var el *skills.EffectList
	switch c := creatureOfRef(obj).(type) {
	case *Player:
		if c.skill != nil {
			el = c.skill.el
		}
	case *Npc:
		if c.skill != nil {
			el = c.skill.el
		}
	}
	if el == nil {
		return 0, false
	}
	b := el.BySkillID(skillID)
	if b == nil {
		return 0, false
	}
	return b.Skill().Level, true
}

// ZoneNpcs returns the NPCs in the world whose position is inside the
// zone's shape, by object ID.
func (o *ScriptOutput) ZoneNpcs(zoneID int) []scripting.NpcRef {
	s := o.srv.Load()
	if s == nil || s.dialog.zones == nil {
		return nil
	}
	i, ok := s.dialog.zones.byID[zoneID]
	if !ok {
		return nil
	}
	shape := s.dialog.zones.rules[i].zone.Shape
	var found []*Npc
	for x := range s.world.Objects() {
		if n, ok := x.(*Npc); ok {
			pos := n.Position()
			if shape.Contains(int(pos.X), int(pos.Y), int(pos.Z)) {
				found = append(found, n)
			}
		}
	}
	slices.SortFunc(found, func(a, b *Npc) int { return cmp.Compare(a.ID(), b.ID()) })
	out := make([]scripting.NpcRef, len(found))
	for i, n := range found {
		out[i] = n.ScriptRef()
	}
	return out
}

// ZonesAt returns the zones containing loc, by zone ID.
func (o *ScriptOutput) ZonesAt(loc scripting.Location) []scripting.Zone {
	s := o.srv.Load()
	if s == nil || s.dialog.zones == nil {
		return nil
	}
	var out []scripting.Zone
	s.dialog.zones.at(worldPosition(loc), func(r *zoneRule) bool {
		out = append(out, scripting.Zone{ID: r.zone.ID, Type: r.zone.Type})
		return true
	})
	slices.SortFunc(out, func(a, b scripting.Zone) int { return cmp.Compare(a.ID, b.ID) })
	return out
}

// ZoneParam returns a parameter of the zone's data.
func (o *ScriptOutput) ZoneParam(zoneID int, name string) (string, bool) {
	s := o.srv.Load()
	if s == nil || s.dialog.zones == nil {
		return "", false
	}
	i, ok := s.dialog.zones.byID[zoneID]
	if !ok {
		return "", false
	}
	return s.dialog.zones.rules[i].zone.Param(name)
}

// SetZoneEnabled is ZoneType.setEnabled: a disabled zone affects nobody
// who enters it (a boss zone sends everyone inside out first).
func (o *ScriptOutput) SetZoneEnabled(zoneID int, enabled bool) {
	s := o.srv.Load()
	if s == nil || s.dialog.zones == nil {
		return
	}
	if s.bosses.zones[zoneID] != nil {
		s.SetBossZoneEnabled(zoneID, enabled)
		return
	}
	if i, ok := s.dialog.zones.byID[zoneID]; ok {
		s.dialog.zones.rules[i].enabled = enabled
	}
}

// oustToTown is ZoneType.oustAllPlayers for a zone that is no boss zone:
// every player inside goes to the nearest town.
func (s *Server) oustToTown(zoneID int) {
	for _, p := range s.BossZonePlayers(zoneID) {
		if loc, ok := s.restartLocation(p); ok {
			s.teleportAround(p, loc, maxOffsetOnTeleport)
		}
	}
}
