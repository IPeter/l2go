package scripttest

import (
	"slices"

	"github.com/IPeter/l2go/internal/scripting"
)

// The fakes of the monster flags (scripting's api_monsters.go).

var (
	_ scripting.MonsterFlags       = (*World)(nil)
	_ scripting.NpcTemplateMinions = npcRef{}
	_ scripting.PlayerCursedWeapon = playerRef{}
)

// NpcSetLethalable sets NotLethalable to the opposite (action "lethalable",
// Value 1 for lethalable).
func (w *World) NpcSetLethalable(npc scripting.NpcRef, on bool) {
	w.npcDo(npc, "lethalable", flag(on), func(n *Npc) { n.NotLethalable = !on })
}

// NpcSetSeeThroughSilentMove sets SeeThroughSilentMove.
func (w *World) NpcSetSeeThroughSilentMove(npc scripting.NpcRef, on bool) {
	w.npcDo(npc, "see_through_silent_move", flag(on), func(n *Npc) { n.SeeThroughSilentMove = on })
}

// NpcInActiveRegion is !InactiveRegion.
func (w *World) NpcInActiveRegion(npc scripting.NpcRef) bool {
	return npcFlag(npc, func(n *Npc) bool { return !n.InactiveRegion })
}

// TemplateMinions returns ParamMinions[name].
func (r npcRef) TemplateMinions(name string) []scripting.TemplateMinion {
	return slices.Clone(r.n.ParamMinions[name])
}

// CursedWeaponEquipped is CursedWeapon.
func (r playerRef) CursedWeaponEquipped() bool { return r.p.CursedWeapon }
