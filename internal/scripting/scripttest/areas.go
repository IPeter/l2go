package scripttest

import (
	"slices"

	"github.com/IPeter/l2go/internal/scripting"
)

// The fake world of the area scripts (Plan 13, Phase 10.7): skill kinds,
// cast checks, effect levels, the weight penalty, parties, and zones as
// the area scripts query them.

var (
	_ scripting.SkillKinds   = (*World)(nil)
	_ scripting.NpcCasting   = (*World)(nil)
	_ scripting.EffectLevels = (*World)(nil)
	_ scripting.ZoneData     = (*World)(nil)

	_ scripting.PlayerWeight = playerRef{}
	_ scripting.PlayerParty  = playerRef{}
)

// Area is a rectangle of the ground: zones.at finds a zone at the points
// inside it, whatever their height.
type Area struct{ MinX, MinY, MaxX, MaxY int }

func (a *Area) contains(loc scripting.Location) bool {
	return a != nil && loc.X >= a.MinX && loc.X <= a.MaxX && loc.Y >= a.MinY && loc.Y <= a.MaxY
}

// SetSkillMagic makes skill:is_magic() true for the skill IDs.
func (w *World) SetSkillMagic(ids ...int) {
	a := w.aiState()
	if a.magic == nil {
		a.magic = make(map[int]bool)
	}
	for _, id := range ids {
		a.magic[id] = true
	}
}

// SkillIsMagic reports the skills set with SetSkillMagic.
func (w *World) SkillIsMagic(s scripting.Skill) bool { return w.aiState().magic[s.ID] }

// NpcCheckDoCastConditions is true unless the NPC is CannotCast.
func (w *World) NpcCheckDoCastConditions(npc scripting.NpcRef, _ scripting.Skill) bool {
	return !npcFlag(npc, func(n *Npc) bool { return n.CannotCast })
}

// NpcStopSkillEffects takes the skill from the NPC's Effects (action
// "stop_effects", Value the skill ID).
func (w *World) NpcStopSkillEffects(npc scripting.NpcRef, skillID int) {
	w.npcDo(npc, "stop_effects", skillID, func(n *Npc) {
		n.Effects = slices.DeleteFunc(n.Effects, func(id int) bool { return id == skillID })
	})
}

// EffectLevel finds the skill in a player's Effects, at its EffectLevels
// level (1 when missing); NPCs' effects have level 1.
func (w *World) EffectLevel(obj any, skillID int) (int, bool) {
	e := effectsOf(obj)
	if e == nil || !slices.Contains(*e, skillID) {
		return 0, false
	}
	if p, ok := obj.(playerRef); ok {
		if l, ok := p.p.EffectLevels[skillID]; ok {
			return l, true
		}
	}
	return 1, true
}

// ZoneNpcs returns the zone's Npcs that are not deleted.
func (w *World) ZoneNpcs(id int) []scripting.NpcRef {
	var out []scripting.NpcRef
	for _, n := range w.zone(id).Npcs {
		if !n.Deleted {
			out = append(out, n.Ref())
		}
	}
	return out
}

// ZonesAt returns the zones whose Area contains loc, by ID.
func (w *World) ZonesAt(loc scripting.Location) []scripting.Zone {
	var out []scripting.Zone
	for _, z := range w.aiState().zones {
		if z.Area.contains(loc) {
			out = append(out, scripting.Zone{ID: z.ID, Type: z.Type})
		}
	}
	slices.SortFunc(out, func(a, b scripting.Zone) int { return a.ID - b.ID })
	return out
}

// ZoneParam reads Params.
func (w *World) ZoneParam(id int, name string) (string, bool) {
	v, ok := w.zone(id).Params[name]
	return v, ok
}

// SetZoneEnabled sets Disabled (action "zone_enabled", Value the zone,
// Level 1 for enabled).
func (w *World) SetZoneEnabled(id int, enabled bool) {
	w.zone(id).Disabled = !enabled
	w.act(Action{Kind: "zone_enabled", Value: id, Level: flag(enabled)})
}

// MakeParty puts the players into one party, the first one leading.
func MakeParty(members ...*Player) {
	party := slices.Clone(members)
	for _, m := range members {
		m.Party = party
	}
}

func (r playerRef) WeightPenalty() int32 { return r.p.Weight }

// PartyMembers returns Party's refs, nil without a party.
func (r playerRef) PartyMembers() []scripting.PlayerRef {
	if len(r.p.Party) == 0 {
		return nil
	}
	out := make([]scripting.PlayerRef, len(r.p.Party))
	for i, m := range r.p.Party {
		out[i] = m.Ref()
	}
	return out
}

// PartyLootMode is 0 (finders keepers) in a party, -1 alone.
func (r playerRef) PartyLootMode() int {
	if len(r.p.Party) == 0 {
		return -1
	}
	return 0
}

// QuestRangeTarget is the player's Target, or nil.
func (r playerRef) QuestRangeTarget() any { return r.TargetObject() }

// SameInstance is always true: the fake world has no instances.
func (r playerRef) SameInstance(scripting.PlayerRef) bool { return true }
