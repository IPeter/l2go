package game

import (
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
