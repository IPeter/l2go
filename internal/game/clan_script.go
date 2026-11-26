package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/scripting"
)

// The clan as the scripts see it (scripting.PlayerClan, ClanRef):
// player:clan() and player:is_clan_leader().

var _ scripting.PlayerClan = (*Player)(nil)

// ScriptClan returns p's clan for the scripts, nil without one
// (scripting.PlayerClan).
func (p *Player) ScriptClan() scripting.ClanRef {
	c := p.clan.clan
	if c == nil {
		return nil
	}
	return clanScriptRef{s: p.srv, c: c}
}

// clanScriptRef is a clan for the scripts.
type clanScriptRef struct {
	s *Server
	c *Clan
}

var _ scripting.ClanRef = clanScriptRef{}

func (r clanScriptRef) ClanID() int32      { return r.c.ID }
func (r clanScriptRef) ClanName() string   { return r.c.Name }
func (r clanScriptRef) ClanLevel() int     { return r.c.Level }
func (r clanScriptRef) Reputation() int    { return r.c.Reputation }
func (r clanScriptRef) LeaderID() int32    { return r.c.LeaderID }
func (r clanScriptRef) LeaderName() string { return r.c.LeaderName() }

// LeaderPlayer is the leader in the game, or nil.
func (r clanScriptRef) LeaderPlayer() scripting.PlayerRef {
	if l := r.c.Leader(); l != nil && l.Online() {
		return l.player
	}
	return nil
}

// AddReputation is Clan.addReputationScore.
func (r clanScriptRef) AddReputation(points int) { r.s.AddClanReputation(r.c, points) }

// BroadcastPledgeInfo sends PledgeShowInfoUpdate to the members online.
func (r clanScriptRef) BroadcastPledgeInfo() { r.s.broadcastClanInfo(r.c) }

// OnlineMembers returns the members in the game (Mobius getOnlineMembers).
func (r clanScriptRef) OnlineMembers() []scripting.PlayerRef {
	var out []scripting.PlayerRef
	for _, p := range r.c.OnlineMembers() {
		out = append(out, p)
	}
	return out
}

var (
	_ scripting.PlayerOnline     = (*Player)(nil)
	_ scripting.AbnormalChecker  = (*ScriptOutput)(nil)
	_ scripting.NpcReturnControl = (*ScriptOutput)(nil)
)

// IsOnline reports whether the player is in the game (Mobius isOnline).
func (p *Player) IsOnline() bool { return p.state == stateInGame }

// HasAbnormalType reports whether a player or NPC has an effect of the
// abnormal type (Mobius getBuffInfoByAbnormalType != null); an unknown type
// has none.
func (o *ScriptOutput) HasAbnormalType(obj any, abnormalType string) bool {
	t, ok := data.ParseAbnormalType(abnormalType)
	if !ok {
		return false
	}
	switch c := creatureOfRef(obj).(type) {
	case *Player:
		return c.skill != nil && c.Effects().ByAbnormal(t) != nil
	case *Npc:
		return c.skill != nil && c.Effects().ByAbnormal(t) != nil
	}
	return false
}

// NpcSetCanReturnToSpawnPoint is Mobius Attackable.setCanReturnToSpawnPoint:
// with false the NPC no longer walks home when it has nothing to fight. As
// on Mobius, the choice lasts as long as the NPC, respawns included.
func (o *ScriptOutput) NpcSetCanReturnToSpawnPoint(npc scripting.NpcRef, can bool) {
	if n := npcOfRef(npc); n != nil {
		n.script.noReturn = !can
	}
}
