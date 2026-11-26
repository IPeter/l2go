package game

import (
	"errors"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/social"
)

// The party packets: invitations (RequestJoinParty, RequestAnswerJoinParty),
// leaving, expelling and the leader change, and the answers decision T10
// gives the party rooms, command channels and duels of M11.

// handleJoinParty handles RequestJoinParty (0x29) with Mobius's checks in
// order (RequestJoinParty.java 59–177): the target is online, has a
// client and is visible; it is in no party, has not blocked p and is not
// p; neither is jailed; p and the target are not busy; only a leader
// invites. Then p gets "You have invited $s1 to your party." and the
// target AskJoinParty: with the loot mode p chose when the invitation
// creates the party, else with the party's (a full party, a pending
// invitation and a busy target are refused). A cursed weapon holder on
// either side is an invalid target. Party bans (M12), events, the Olympiad
// and the Dimensional Rift do not exist yet.
func (s *Server) handleJoinParty(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestJoinParty(body)
	if err != nil {
		s.log.Debug("game: bad RequestJoinParty", append(p.logArgs(), "err", err)...)
		return
	}
	target, ok := s.PlayerByName(req.Name)
	if !ok || target.state != stateInGame {
		p.SendSystemMessage(partyMsgSelectUser)
		return
	}
	if target.session == nil {
		p.SendMessage("Player is in offline mode.")
		return
	}
	if target.Invisible() && !p.IsGM() {
		p.SendSystemMessage(partyMsgIncorrectTarget)
		return
	}
	name := serverpackets.TextParam(target.Name())
	switch {
	case target.InParty():
		p.SendSystemMessage(partyMsgInAnotherParty, name)
		return
	case target.Blocks(p):
		p.SendSystemMessage(partyMsgIgnoreList, name)
		return
	case target == p:
		p.SendSystemMessage(partyMsgWrongTarget)
		return
	case target.CursedWeaponEquipped() || p.CursedWeaponEquipped(): // cursedweapon.go
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	case target.jailed() || p.jailed():
		p.SendMessage("You cannot invite a player while is in Jail.")
		return
	case p.IsProcessingRequest():
		p.SendSystemMessage(partyMsgWaitingReply)
		return
	case target.IsProcessingRequest():
		p.SendSystemMessage(partyMsgBusy, name)
		return
	}
	pt := p.party.party
	if pt != nil && !pt.IsLeader(p) {
		p.SendSystemMessage(partyMsgOnlyLeaderInvites)
		return
	}
	p.SendSystemMessage(partyMsgInvited, name)
	if pt == nil {
		s.inviteToNewParty(p, target, social.LootMode(req.Loot))
		return
	}
	s.inviteToParty(p, target, pt)
}

// inviteToNewParty is RequestJoinParty.createNewParty: an unknown loot mode
// stops here; the mode is kept for the party the answer creates.
func (s *Server) inviteToNewParty(p, target *Player, mode social.LootMode) {
	if !mode.Valid() {
		return
	}
	if target.IsProcessingRequest() {
		p.SendSystemMessage(partyMsgWaitingReply)
		return
	}
	target.Send(serverpackets.NewAskJoinPartyPacket(p.Name(), serverpackets.PartyLoot(mode))) //nolint:gosec // loot modes are 0-4
	target.SetActiveRequester(p)
	p.OnTransactionRequest(target)
	p.party.lootMode = mode
}

// inviteToParty is RequestJoinParty.addTargetToParty: the party's own loot
// mode, one pending invitation at a time.
func (s *Server) inviteToParty(p, target *Player, pt *Party) {
	switch {
	case !pt.IsLeader(p):
		p.SendSystemMessage(partyMsgOnlyLeaderInvites)
	case len(pt.members) >= partyMembersLimit:
		p.SendSystemMessage(partyMsgFull)
	case pt.pendingInvite && !s.inviteExpired(pt):
		p.SendSystemMessage(partyMsgWaitingReply)
	case target.IsProcessingRequest():
		p.SendSystemMessage(partyMsgBusy, serverpackets.TextParam(target.Name()))
	default:
		p.OnTransactionRequest(target)
		target.Send(serverpackets.NewAskJoinPartyPacket(p.Name(), serverpackets.PartyLoot(pt.loot.Mode))) //nolint:gosec // loot modes are 0-4
		s.setPendingInvite(pt, true)
	}
}

// handleAnswerJoinParty handles RequestAnswerJoinParty (0x2A, Mobius
// RequestAnswerJoinParty): without an open invitation nothing happens.
// The inviter gets JoinParty with the answer; -1 (the client refuses
// invitations) also tells it so in text. An acceptance creates the party
// with the inviter as leader and the mode chosen at the invitation, or
// joins the inviter's party; a full party refuses both and leaves the
// request open, as Mobius does.
func (s *Server) handleAnswerJoinParty(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestAnswerJoinParty(body)
	if err != nil {
		s.log.Debug("game: bad RequestAnswerJoinParty", append(p.logArgs(), "err", err)...)
		return
	}
	requester := p.ActiveRequester()
	if requester == nil {
		return
	}
	requester.Send(serverpackets.NewJoinPartyPacket(req.Response))
	switch req.Response {
	case clientpackets.PartyAnswerRefusalMode:
		requester.SendMessage(p.Name() + " is set to refuse party requests and cannot receive a party request.")
	case clientpackets.PartyAnswerAccepted:
		if pt := requester.party.party; pt != nil {
			if len(pt.members) >= partyMembersLimit {
				full := sysMsgPacket(partyMsgFull)
				p.Send(full)
				requester.Send(full)
				return
			}
			if !p.InParty() {
				s.addPartyMember(pt, p)
			}
		} else if !p.InParty() {
			pt := newParty(requester, requester.party.lootMode)
			s.addPartyMember(pt, p)
		}
	}
	if pt := requester.party.party; pt != nil {
		s.setPendingInvite(pt, false)
	}
	p.SetActiveRequester(nil)
	requester.OnTransactionResponse()
}

// handleWithdrawParty handles RequestWithDrawalParty (0x2B): p leaves its
// party.
func (s *Server) handleWithdrawParty(p *Player, _ []byte) {
	if pt := p.party.party; pt != nil {
		s.removePartyMember(pt, p, partyLeaveLeft)
	}
}

// handleOustPartyMember handles RequestOustPartyMember (0x2C): the leader
// expels the member with the name; an unknown name does nothing.
func (s *Server) handleOustPartyMember(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestOustPartyMember(body)
	if err != nil {
		s.log.Debug("game: bad RequestOustPartyMember", append(p.logArgs(), "err", err)...)
		return
	}
	pt := p.party.party
	if pt == nil || !pt.IsLeader(p) {
		return
	}
	if m := pt.MemberByName(req.Name); m != nil {
		s.removePartyMember(pt, m, partyLeaveExpelled)
	}
}

// handleChangePartyLeader handles RequestChangePartyLeader (D0:04): the
// leader hands the lead to a member; others are told that only the leader
// may.
func (s *Server) handleChangePartyLeader(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestChangePartyLeader(body)
	if err != nil {
		s.log.Debug("game: bad RequestChangePartyLeader", append(p.logArgs(), "err", err)...)
		return
	}
	pt := p.party.party
	if pt == nil {
		return
	}
	m := pt.MemberByName(req.Name)
	switch {
	case pt.IsLeader(p) && m != nil:
		s.setPartyLeader(pt, m)
	case pt.IsLeader(p):
		p.SendSystemMessage(partyMsgTransferToMember)
	case m != nil:
		p.SendSystemMessage(partyMsgOnlyLeaderTransfers)
	}
}

// The party matching, command channel and duel packets that answer
// ActionFailed until M11 (decision T10): RequestPartyMatchList (0x70) and
// RequestPartyMatchDetail (0x71), and the Ex packets of the party rooms
// (D0:01–03, 14–17), the command channel (D0:0D–0F, 26) and duels (D0:27,
// 28, 30).
var (
	partyExtraOpcodes   = []byte{0x70, 0x71}
	partyExtraExOpcodes = []uint16{0x01, 0x02, 0x03, 0x0d, 0x0e, 0x0f, 0x14, 0x15, 0x16, 0x17, 0x26, 0x27, 0x28, 0x30}
)

// registerPartyExtras registers RequestPartyMatchConfig and the T10
// answers.
func (s *Server) registerPartyExtras() error {
	errs := []error{s.Register(clientpackets.OpRequestPartyMatchConfig, s.handlePartyMatchConfig)}
	failed := func(p *Player, _ []byte) { p.SendActionFailed() }
	for _, op := range partyExtraOpcodes {
		errs = append(errs, s.Register(op, failed))
	}
	for _, op := range partyExtraExOpcodes {
		errs = append(errs, s.RegisterEx(op, failed))
	}
	return errors.Join(errs...)
}

// handlePartyMatchConfig handles RequestPartyMatchConfig (0x6F): a member
// who does not lead gets Mobius's refusal; anyone else the room list, which
// is empty until party rooms come with M11 (decision T10).
func (s *Server) handlePartyMatchConfig(p *Player, body []byte) {
	if _, err := clientpackets.ParseRequestPartyMatchConfig(body); err != nil {
		s.log.Debug("game: bad RequestPartyMatchConfig", append(p.logArgs(), "err", err)...)
		return
	}
	if pt := p.party.party; pt != nil && !pt.IsLeader(p) {
		p.SendSystemMessage(partyMsgRoomsLeaderOnly)
		p.SendActionFailed()
		return
	}
	p.Send(serverpackets.NewListPartyWatingPacket(nil))
}
