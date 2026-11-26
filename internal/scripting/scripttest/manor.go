package scripttest

import "github.com/IPeter/l2go/internal/scripting"

// The castle manor of the fake world (scripting.Manor). Set World.Manor's
// fields between calls.

var _ scripting.Manor = (*World)(nil)

// ManorState is the fake manor.
type ManorState struct {
	// Disabled turns the manor off (manor.is_enabled() false).
	Disabled bool
	// ManorIDs are the manor_id parameters by NPC template ID; an NPC not
	// listed has none (-1).
	ManorIDs map[int]int
	// ClanCastles are the castles the clans own, by clan ID.
	ClanCastles map[int32]int
	// SeedIncomes are the seed incomes by castle ID.
	SeedIncomes map[int]int64
}

// ManorEnabled reports !Manor.Disabled.
func (w *World) ManorEnabled() bool { return !w.Manor.Disabled }

// ManorID returns the NPC's entry of Manor.ManorIDs, -1 without one.
func (w *World) ManorID(npc scripting.NpcRef) int {
	n := npcOf(npc)
	if n == nil {
		return -1
	}
	if id, ok := w.Manor.ManorIDs[n.ID]; ok {
		return id
	}
	return -1
}

// IsManorLord reports a player who is no GM, leads its clan, and whose clan
// owns (Manor.ClanCastles) the castle of the NPC's manor_id.
func (w *World) IsManorLord(npc scripting.NpcRef, p scripting.PlayerRef) bool {
	pl := playerOf(p)
	if pl == nil || pl.GM || pl.Clan == nil || pl.Clan.Leader != pl {
		return false
	}
	castle, ok := w.Manor.ClanCastles[pl.Clan.ID]
	return ok && castle == w.ManorID(npc)
}

// SeedIncome returns the castle's entry of Manor.SeedIncomes.
func (w *World) SeedIncome(castleID int) int64 { return w.Manor.SeedIncomes[castleID] }
