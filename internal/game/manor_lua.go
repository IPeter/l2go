package game

import "github.com/IPeter/l2go/internal/scripting"

// The manor as the scripts see it (scripting.Manor): the manor managers'
// first talk and the chamberlain's vault page. Everything runs on the loop,
// while a handler runs.

var _ scripting.Manor = (*ScriptOutput)(nil)

// ManorEnabled is Mobius GeneralConfig.ALLOW_MANOR.
func (o *ScriptOutput) ManorEnabled() bool {
	s := o.srv.Load()
	return s != nil && s.manor.enabled
}

// ManorID is the NPC's manor_id template parameter, -1 without one.
func (o *ScriptOutput) ManorID(npc scripting.NpcRef) int {
	n := npcOfRef(npc)
	if n == nil || n.Template == nil {
		return -1
	}
	return int(manorIDOf(n))
}

// IsManorLord is ManorManager.onFirstTalk's lord test.
func (o *ScriptOutput) IsManorLord(npc scripting.NpcRef, p scripting.PlayerRef) bool {
	_, pl, ok := o.player(p)
	n := npcOfRef(npc)
	if !ok || n == nil || n.Template == nil || pl.IsGM() || !pl.IsClanLeader() {
		return false
	}
	c := pl.clan.clan
	return c != nil && c.CastleID != 0 && c.CastleID == manorIDOf(n)
}

// SeedIncome is the seed income of the castle's running period.
func (o *ScriptOutput) SeedIncome(castleID int) int64 {
	s := o.srv.Load()
	if s == nil {
		return 0
	}
	mc := s.manorCastle(int32(castleID)) //nolint:gosec // castle IDs
	if mc == nil {
		return 0
	}
	return mc.SeedIncome()
}
