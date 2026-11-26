package game

import (
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/skills"
)

// The monster flags of the AI scripts (scripting's api_monsters.go): lethal
// strikes (read by skillBase.IsLethalable), Silent Move (read by
// isAggressiveTowards), the world region and the template's minion lists.
// player:is_cursed_weapon_equipped() is Player.CursedWeaponEquipped
// (cursedweapon.go).

var (
	_ scripting.MonsterFlags       = (*ScriptOutput)(nil)
	_ scripting.NpcTemplateMinions = npcRef{}
	_ scripting.PlayerCursedWeapon = (*Player)(nil)
)

// NpcSetLethalable is Mobius setLethalable.
func (o *ScriptOutput) NpcSetLethalable(npc scripting.NpcRef, on bool) {
	if n := npcOfRef(npc); n != nil {
		n.script.notLethalable = !on
	}
}

// NpcSetSeeThroughSilentMove is Mobius Attackable.setSeeThroughSilentMove.
func (o *ScriptOutput) NpcSetSeeThroughSilentMove(npc scripting.NpcRef, on bool) {
	if n := npcOfRef(npc); n != nil {
		n.script.seeThroughSilentMove = on
	}
}

// silentMoveHides is the Silent Move check of Mobius
// AttackableAI.isAggressiveTowards: a player in Silent Move draws no
// aggression from a monster that is not a raid boss and cannot see through
// it.
func silentMoveHides(n *Npc, p *Player) bool {
	if n.script.seeThroughSilentMove || (n.class != nil && n.class.raid) {
		return false
	}
	return p.skill != nil && p.skill.el != nil && p.skill.el.IsAffected(skills.FlagSilentMove)
}

// NpcInActiveRegion is Mobius getWorldRegion().isActive().
func (o *ScriptOutput) NpcInActiveRegion(npc scripting.NpcRef) bool {
	s, n, ok := o.npc(npc)
	return ok && s.world.RegionActive(n)
}

// TemplateMinions is Mobius getParameters().getMinionList(name).
func (r npcRef) TemplateMinions(name string) []scripting.TemplateMinion {
	list := r.n.Template.Parameters.Minions[name]
	out := make([]scripting.TemplateMinion, 0, len(list))
	for _, m := range list {
		out = append(out, scripting.TemplateMinion{ID: m.NpcID, Count: m.Count})
	}
	return out
}
