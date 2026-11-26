package scripttest

import (
	"slices"

	"github.com/IPeter/l2go/internal/scripting"
)

// Clan is a fake clan, player:clan() of the players whose Clan it is. The
// player that is Leader leads it.
type Clan struct {
	ID         int32
	Name       string
	Level      int
	Reputation int
	Leader     *Player
	// Members are the members in the game, the leader included
	// (q:set_quest_to_clan_members and q:finish_quest_to_clan walk them).
	Members []*Player
	// PledgeInfoUpdates counts clan:broadcast_pledge_info_update().
	PledgeInfoUpdates int
}

var (
	_ scripting.PlayerClan       = playerRef{}
	_ scripting.PlayerOnline     = playerRef{}
	_ scripting.ClanRef          = clanRef{}
	_ scripting.AbnormalChecker  = (*World)(nil)
	_ scripting.NpcReturnControl = (*World)(nil)
)

// ScriptClan returns the player's Clan.
func (r playerRef) ScriptClan() scripting.ClanRef {
	if r.p.Clan == nil {
		return nil
	}
	return clanRef{r.p.Clan}
}

// IsClanLeader reports whether the player is its clan's Leader.
func (r playerRef) IsClanLeader() bool { return r.p.Clan != nil && r.p.Clan.Leader == r.p }

// IsOnline is the opposite of Offline.
func (r playerRef) IsOnline() bool { return !r.p.Offline }

type clanRef struct{ c *Clan }

func (r clanRef) ClanID() int32    { return r.c.ID }
func (r clanRef) ClanName() string { return r.c.Name }
func (r clanRef) ClanLevel() int   { return r.c.Level }
func (r clanRef) Reputation() int  { return r.c.Reputation }

func (r clanRef) LeaderID() int32 {
	if r.c.Leader == nil {
		return 0
	}
	return r.c.Leader.ObjectID
}

func (r clanRef) LeaderName() string {
	if r.c.Leader == nil {
		return ""
	}
	return r.c.Leader.Name
}

// LeaderPlayer is the Leader unless it is Offline.
func (r clanRef) LeaderPlayer() scripting.PlayerRef {
	if r.c.Leader == nil || r.c.Leader.Offline {
		return nil
	}
	return r.c.Leader.Ref()
}

// AddReputation adds to Reputation.
func (r clanRef) AddReputation(points int) { r.c.Reputation += points }

// BroadcastPledgeInfo counts in PledgeInfoUpdates.
func (r clanRef) BroadcastPledgeInfo() { r.c.PledgeInfoUpdates++ }

// OnlineMembers returns the Members that are not Offline.
func (r clanRef) OnlineMembers() []scripting.PlayerRef {
	var out []scripting.PlayerRef
	for _, m := range r.c.Members {
		if m != nil && !m.Offline {
			out = append(out, m.Ref())
		}
	}
	return out
}

// HasAbnormalType looks in a fake player's Abnormals.
func (w *World) HasAbnormalType(obj any, abnormalType string) bool {
	if r, ok := obj.(playerRef); ok {
		return slices.Contains(r.p.Abnormals, abnormalType)
	}
	return false
}

// NpcSetCanReturnToSpawnPoint sets the NPC's NoReturn to the opposite.
func (w *World) NpcSetCanReturnToSpawnPoint(npc scripting.NpcRef, can bool) {
	if r, ok := npc.(npcRef); ok {
		r.n.NoReturn = !can
	}
}
