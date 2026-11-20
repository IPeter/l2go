package game

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/world"
)

// The party feature (party*.go, Plan 12 Phase 3): Mobius Party and
// AbstractPlayerGroup on the loop.
//
//   - Membership (this file): addPartyMember, removePartyMember,
//     disbandParty and setLeader with Mobius's packets and messages in
//     order, the party window, PartyMemberPosition every 12 s (the first
//     after 6 s), party chat and /partyinfo.
//   - Invitations and the party packets (party_invite.go), the request
//     partner slot (party_request.go; the active requester is in
//     trade_request.go), the live window (party_window.go), rewards and
//     loot (party_reward.go), summoning (party_summon.go).
//
// Parties are not stored, as in Mobius: a restart clears them.
func init() { registerFeature("party", setupParty) }

// The party's timings (Mobius Party.PARTY_POSITION_BROADCAST_INTERVAL).
const (
	partyPositionInterval = 12 * time.Second
	partyPositionFirst    = partyPositionInterval / 2
)

// partyMembersLimit is the size of a full party (RequestJoinParty: 9).
const partyMembersLimit = social.MaxPartyMembers

// System messages of the party.
const (
	partyMsgInvited             serverpackets.SystemMessageID = 105  // "You have invited $s1 to your party."
	partyMsgJoined              serverpackets.SystemMessageID = 106  // "You have joined $s1's party."
	partyMsgMemberJoined        serverpackets.SystemMessageID = 107  // "$s1 has joined the party."
	partyMsgMemberLeft          serverpackets.SystemMessageID = 108  // "$s1 has left the party."
	partyMsgInvalidTarget       serverpackets.SystemMessageID = 109  // "Invalid target."
	partyMsgIncorrectTarget     serverpackets.SystemMessageID = 144  // "That is the incorrect target."
	partyMsgWrongTarget         serverpackets.SystemMessageID = 152  // "You have invited the wrong target."
	partyMsgBusy                serverpackets.SystemMessageID = 153  // "$s1 is busy. Please try again later."
	partyMsgOnlyLeaderInvites   serverpackets.SystemMessageID = 154  // "Only the leader can give out invitations."
	partyMsgFull                serverpackets.SystemMessageID = 155  // "The party is full."
	partyMsgInAnotherParty      serverpackets.SystemMessageID = 160  // "$s1 is a member of another party and cannot be invited."
	partyMsgWaitingReply        serverpackets.SystemMessageID = 164  // "Waiting for another reply."
	partyMsgSelectUser          serverpackets.SystemMessageID = 185  // "You must first select a user to invite to your party."
	partyMsgWithdrawn           serverpackets.SystemMessageID = 200  // "You have withdrawn from the party."
	partyMsgMemberExpelled      serverpackets.SystemMessageID = 201  // "$s1 was expelled from the party."
	partyMsgExpelled            serverpackets.SystemMessageID = 202  // "You have been expelled from the party."
	partyMsgDispersed           serverpackets.SystemMessageID = 203  // "The party has dispersed."
	partyMsgIgnoreList          serverpackets.SystemMessageID = 619  // "$s1 has placed you on his/her Ignore List."
	partyMsgInformation         serverpackets.SystemMessageID = 1030 // "<Party Information>"
	partyMsgLootingBase         serverpackets.SystemMessageID = 1031 // "Looting method: Finders keepers" (1031-1035 by mode)
	partyMsgNewLeader           serverpackets.SystemMessageID = 1384 // "$s1 has become the party leader."
	partyMsgRoomsLeaderOnly     serverpackets.SystemMessageID = 1396 // "The list of party rooms can only be viewed by ..."
	partyMsgOnlyLeaderTransfers serverpackets.SystemMessageID = 1399 // "Only the leader of the party can transfer party leadership to another player."
	partyMsgAlreadyLeader       serverpackets.SystemMessageID = 1401 // "Slow down, you are already the party leader."
	partyMsgTransferToMember    serverpackets.SystemMessageID = 1402 // "You may only transfer party leadership to another member of the party."
	partyMsgLeaderS1            serverpackets.SystemMessageID = 1611 // "Party Leader: $s1"
	partyMsgLine                serverpackets.SystemMessageID = 490  // "========================"
)

// partyInfoCommand is the user command /partyinfo (Mobius PartyInfo).
const partyInfoCommand = 81

// partyFeature is the party state on the Server (Server.party). Loop-only.
type partyFeature struct {
	// exp is the party exp rules of the config (decision T8).
	exp social.PartyExp
	// dlg holds the DlgAnswer handlers by message ID (party_summon.go).
	dlg map[serverpackets.SystemMessageID]DlgAnswerHandler
}

// Party is a party of 2 to 9 players (Mobius Party): the members in party
// order, the leader first, and the loot mode chosen at the first
// invitation. Loop-only.
type Party struct {
	members []*Player
	loot    social.Loot
	// pendingInvite and inviteExpires are Mobius's one pending invitation
	// per party (setPendingInvitation): 15 s by the world clock.
	pendingInvite bool
	inviteExpires time.Time
	// positionTimer is the PartyMemberPosition task, 0 without one.
	positionTimer world.TimerID
	// disbanding is set while the party disperses; it never ends.
	disbanding bool
}

// playerParty is a player's party state (Player.party).
type playerParty struct {
	// party is the player's party, nil without one.
	party *Party
	// lootMode is the loot mode the player chose for the party its
	// invitation creates (Mobius Player._partyDistributionType).
	lootMode social.LootMode
	// partner is the request partner slot (party_request.go).
	partner requestPartner
	// summon is the open CallPc request (party_summon.go), nil for none.
	summon *summonRequest
	// bars are the party window thresholds (party_window.go).
	bars partyBars
}

// partyLeave tells removePartyMember why a member goes (Mobius
// PartyMessageType).
type partyLeave int

const (
	partyLeaveNone partyLeave = iota // the party disperses
	partyLeaveExpelled
	partyLeaveLeft
	partyLeaveDisconnected
)

// setupParty registers the party packets, /partyinfo, the leave hook and
// the DlgAnswer dispatcher.
func setupParty(s *Server) error {
	s.party.exp = partyExpRules(s.gs)
	s.party.dlg = make(map[serverpackets.SystemMessageID]DlgAnswerHandler)
	s.OnPlayerLeave(s.partyLeave)
	err := errors.Join(
		s.Register(clientpackets.OpRequestJoinParty, s.handleJoinParty),
		s.Register(clientpackets.OpRequestAnswerJoinParty, s.handleAnswerJoinParty),
		s.Register(clientpackets.OpRequestWithDrawalParty, s.handleWithdrawParty),
		s.Register(clientpackets.OpRequestOustPartyMember, s.handleOustPartyMember),
		s.RegisterEx(clientpackets.ExOpRequestChangePartyLeader, s.handleChangePartyLeader),
		s.RegisterUserCommand(partyInfoCommand, s.partyInfo),
		s.registerPartyExtras(),
		s.setupSummons(),
	)
	return err
}

// partyExpRules builds the party exp rules of a game server's config: the
// party section and the party rates, Mobius's defaults where a config made
// in code left them out.
func partyExpRules(gs config.GameServer) social.PartyExp {
	c := gs.Party
	if c.XPCutoffMethod == "" {
		c = config.DefaultParty()
	}
	r := social.DefaultPartyExp()
	if m, err := social.ParseCutoffMethod(string(c.XPCutoffMethod)); err == nil {
		r.Method = m
	}
	r.CutoffLevel = c.XPCutoffLevel
	r.CutoffPercent = c.XPCutoffPercent
	if len(c.XPCutoffGaps) == len(c.XPCutoffGapPercents) {
		r.Gaps, r.GapPercents = c.XPCutoffGaps, c.XPCutoffGapPercents
	}
	if gs.Rates.PartyXP > 0 || gs.Rates.PartySP > 0 {
		r.RateXP, r.RateSP = gs.Rates.PartyXP, gs.Rates.PartySP
	}
	return r
}

// Party returns the player's party, nil without one. Loop-only.
func (p *Player) Party() *Party { return p.party.party }

// InParty reports whether the player is in a party. Loop-only.
func (p *Player) InParty() bool { return p.party.party != nil }

// Members returns the members in party order, the leader first. The slice
// is a copy. Loop-only.
func (pt *Party) Members() []*Player { return slices.Clone(pt.members) }

// Len returns the number of members.
func (pt *Party) Len() int { return len(pt.members) }

// Leader returns the leader, nil for a party that dispersed.
func (pt *Party) Leader() *Player {
	if len(pt.members) == 0 {
		return nil
	}
	return pt.members[0]
}

// IsLeader reports whether p leads the party.
func (pt *Party) IsLeader(p *Player) bool { return pt.Leader() == p && p != nil }

// Contains reports whether p is a member.
func (pt *Party) Contains(p *Player) bool { return slices.Contains(pt.members, p) }

// LootMode returns the loot mode.
func (pt *Party) LootMode() social.LootMode { return pt.loot.Mode }

// Level returns the party level: the highest member level (Mobius
// Party.getLevel, kept up to date on joins, leaves and level changes).
func (pt *Party) Level() int {
	top := 0
	for _, m := range pt.members {
		top = max(top, m.Level())
	}
	return top
}

// MemberByName returns the member with the name, ignoring case (Mobius
// getPlayerByName).
func (pt *Party) MemberByName(name string) *Player {
	for _, m := range pt.members {
		if strings.EqualFold(m.Name(), name) {
			return m
		}
	}
	return nil
}

// Broadcast sends body to every member (Mobius broadcastPacket).
func (pt *Party) Broadcast(body []byte) {
	for _, m := range pt.members {
		m.Send(body)
	}
}

// BroadcastExcept sends body to every member but p (Mobius
// broadcastToPartyMembers).
func (pt *Party) BroadcastExcept(p *Player, body []byte) {
	for _, m := range pt.members {
		if m != p {
			m.Send(body)
		}
	}
}

// broadcastMessage sends a system message to every member.
func (pt *Party) broadcastMessage(id serverpackets.SystemMessageID, params ...serverpackets.SystemMessageParam) {
	pt.Broadcast(sysMsgPacket(id, params...))
}

// index returns p's place in the party, -1 if it is not a member.
func (pt *Party) index(p *Player) int { return slices.Index(pt.members, p) }

func sysMsgPacket(id serverpackets.SystemMessageID, params ...serverpackets.SystemMessageParam) []byte {
	return serverpackets.NewSystemMessagePacket(serverpackets.SystemMessage{ID: id, Params: params})
}

// SameParty reports whether a and b are in the same party. Loop-only.
func (s *Server) SameParty(a, b *Player) bool {
	return a != nil && b != nil && a.party.party != nil && a.party.party == b.party.party
}

// partyMember is a member as the party window shows it.
func partyMember(m *Player) serverpackets.PartyMember {
	v := m.stats.View()
	return serverpackets.PartyMember{
		ObjectID: m.ObjectID(),
		Name:     m.Name(),
		CurCP:    m.CurCP, CurHP: m.CurHP, CurMP: m.CurMP,
		MaxCP: v.MaxCP, MaxHP: v.MaxHP, MaxMP: v.MaxMP,
		Level:   int32(m.Level()),   //nolint:gosec // levels are small
		ClassID: int32(m.ClassID()), //nolint:gosec // class IDs are small
		Race:    int32(m.Race()),    //nolint:gosec // races are small
	}
}

// windowAll is PartySmallWindowAll for p: the other members.
func (pt *Party) windowAll(p *Player) []byte {
	all := serverpackets.PartySmallWindowAll{LeaderID: pt.Leader().ObjectID(), Loot: serverpackets.PartyLoot(pt.loot.Mode)}
	for _, m := range pt.members {
		if m != p {
			all.Members = append(all.Members, partyMember(m))
		}
	}
	return serverpackets.NewPartySmallWindowAllPacket(all)
}

// newParty is new Party(leader, mode): the leader alone, not yet told.
func newParty(leader *Player, mode social.LootMode) *Party {
	pt := &Party{members: []*Player{leader}, loot: social.Loot{Mode: mode}}
	leader.party.party = pt
	return pt
}

// setPendingInvite is Party.setPendingInvitation: the flag and a new 15 s
// expiry.
func (s *Server) setPendingInvite(pt *Party, pending bool) {
	pt.pendingInvite = pending
	pt.inviteExpires = s.Now().Add(requestTimeout)
}

// inviteExpired is Party.isInvitationRequestExpired.
func (s *Server) inviteExpired(pt *Party) bool { return !s.Now().Before(pt.inviteExpires) }

// addPartyMember is Player.joinParty and Party.addPartyMember: p gets the
// window and "You have joined $s1's party.", the others "$s1 has joined the
// party." and PartySmallWindowAdd; every member's party icons and UserInfo
// are refreshed, and the position task starts. Loop-only.
func (s *Server) addPartyMember(pt *Party, p *Player) {
	if pt.Contains(p) {
		return
	}
	p.party.party = pt
	pt.members = append(pt.members, p)
	p.Send(pt.windowAll(p))
	p.SendSystemMessage(partyMsgJoined, serverpackets.TextParam(pt.Leader().Name()))
	pt.BroadcastExcept(p, sysMsgPacket(partyMsgMemberJoined, serverpackets.TextParam(p.Name())))
	add := serverpackets.NewPartySmallWindowAddPacket(pt.Leader().ObjectID(), serverpackets.PartyLoot(pt.loot.Mode), partyMember(p))
	pt.BroadcastExcept(p, add)
	for _, m := range pt.members {
		m.Effects().UpdateIcons(true)
		m.BroadcastUserInfo()
	}
	if pt.positionTimer == 0 {
		pt.positionTimer = s.loop.After(partyPositionFirst, func() {
			pt.positionTimer = s.loop.Every(partyPositionInterval, func() { s.partyPositions(pt) })
			s.partyPositions(pt)
		})
	}
}

// partyPositions sends PartyMemberPosition to every member (Mobius's
// position task).
func (s *Server) partyPositions(pt *Party) {
	if len(pt.members) == 0 {
		return
	}
	locs := make([]serverpackets.PartyMemberLocation, 0, len(pt.members))
	for _, m := range pt.members {
		pos := m.Position()
		locs = append(locs, serverpackets.PartyMemberLocation{ObjectID: m.ObjectID(), X: pos.X, Y: pos.Y, Z: pos.Z})
	}
	pt.Broadcast(serverpackets.NewPartyMemberPositionPacket(locs))
}

// newLeaderOnLeave is AltLeavePartyLeader.
func (s *Server) newLeaderOnLeave() bool { return s.gs.Party.NewLeaderOnLeave }

// removePartyMember is Party.removePartyMember. A party of two, or one
// whose leader leaves (unless AltLeavePartyLeader or the leader
// disconnected), disperses instead. Otherwise the member gets its message
// and PartySmallWindowDeleteAll, the others theirs and
// PartySmallWindowDelete; a leader who left hands over to the next member,
// and the last member of a dispersing party is left alone. Loop-only.
func (s *Server) removePartyMember(pt *Party, p *Player, how partyLeave) {
	i := pt.index(p)
	if i < 0 {
		return
	}
	isLeader := i == 0
	if !pt.disbanding && (len(pt.members) == 2 || (isLeader && !s.newLeaderOnLeave() && how != partyLeaveDisconnected)) {
		s.disbandParty(pt)
		return
	}
	pt.members = slices.Delete(slices.Clone(pt.members), i, i+1)
	name := serverpackets.TextParam(p.Name())
	switch how {
	case partyLeaveExpelled:
		p.SendSystemMessage(partyMsgExpelled)
		pt.broadcastMessage(partyMsgMemberExpelled, name)
	case partyLeaveLeft, partyLeaveDisconnected:
		p.SendSystemMessage(partyMsgWithdrawn)
		pt.broadcastMessage(partyMsgMemberLeft, name)
	}
	p.Send(serverpackets.NewPartySmallWindowDeleteAllPacket())
	p.party.party = nil
	pt.Broadcast(serverpackets.NewPartySmallWindowDeletePacket(p.ObjectID(), p.Name()))
	switch {
	case isLeader && len(pt.members) > 1 && (s.newLeaderOnLeave() || how == partyLeaveDisconnected):
		pt.broadcastMessage(partyMsgNewLeader, serverpackets.TextParam(pt.Leader().Name()))
		s.partyNewLeader(pt)
	case len(pt.members) == 1:
		pt.members[0].party.party = nil
		s.loop.Cancel(pt.positionTimer)
		pt.positionTimer = 0
		pt.members = nil
	}
}

// disbandParty is Party.disbandParty: "The party has dispersed." to every
// member, then each leaves without a message of its own.
func (s *Server) disbandParty(pt *Party) {
	pt.disbanding = true
	pt.broadcastMessage(partyMsgDispersed)
	for _, m := range slices.Clone(pt.members) {
		s.removePartyMember(pt, m, partyLeaveNone)
	}
}

// partyNewLeader is broadcastToPartyMembersNewLeader: every member gets
// PartySmallWindowDeleteAll and a new PartySmallWindowAll, UserInfo is
// broadcast, and the party icons are sent again.
func (s *Server) partyNewLeader(pt *Party) {
	for _, m := range pt.members {
		m.Send(serverpackets.NewPartySmallWindowDeleteAllPacket())
		m.Send(pt.windowAll(m))
		m.BroadcastUserInfo()
	}
	for _, m := range pt.members {
		m.Effects().UpdateIcons(true)
	}
}

// setPartyLeader is Party.setLeader: a member becomes the leader in the
// old leader's place, everyone is told and the windows are rebuilt.
func (s *Server) setPartyLeader(pt *Party, p *Player) {
	i := pt.index(p)
	switch {
	case i < 0:
		p.SendSystemMessage(partyMsgTransferToMember)
	case i == 0:
		p.SendSystemMessage(partyMsgAlreadyLeader)
	default:
		pt.members[0], pt.members[i] = pt.members[i], pt.members[0]
		pt.broadcastMessage(partyMsgNewLeader, serverpackets.TextParam(p.Name()))
		s.partyNewLeader(pt)
	}
}

// LeaveParty takes p out of its party as a disconnection does (Mobius
// Player.leaveParty). Loop-only.
func (s *Server) LeaveParty(p *Player) {
	if pt := p.party.party; pt != nil {
		s.removePartyMember(pt, p, partyLeaveDisconnected)
		p.party.party = nil
	}
}

// partyLeave is the leave hook: the player leaves its party as
// disconnected (Mobius deleteMe), and its request partner and summon
// request end.
func (s *Server) partyLeave(p *Player) {
	s.LeaveParty(p)
	s.clearRequestPartner(p)
	p.party.summon = nil
}

// chatParty sends party chat (Mobius ChatParty and broadcastCreatureSay):
// every member who has not blocked p gets it.
func (s *Server) chatParty(p *Player, text string) {
	pt := p.party.party
	if pt == nil {
		p.SendMessage("You are not in a party.")
		return
	}
	say := chatSay(p, serverpackets.ChatParty, p.Name(), text)
	for _, m := range pt.members {
		if !m.Blocks(p) {
			m.Send(say)
		}
	}
}

// partyInfo is /partyinfo (Mobius PartyInfo): the loot mode, the leader
// for the other members, the member count.
func (s *Server) partyInfo(p *Player) {
	p.SendSystemMessage(partyMsgInformation)
	if pt := p.party.party; pt != nil {
		if pt.loot.Mode.Valid() {
			p.SendSystemMessage(partyMsgLootingBase + serverpackets.SystemMessageID(pt.loot.Mode))
		}
		if !pt.IsLeader(p) {
			p.SendSystemMessage(partyMsgLeaderS1, serverpackets.TextParam(pt.Leader().Name()))
		}
		p.SendMessage("Members: " + strconv.Itoa(len(pt.members)) + "/9")
	}
	p.SendSystemMessage(partyMsgLine)
}
