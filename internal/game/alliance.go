package game

import (
	"errors"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
)

// Alliances (Plan 12 Phase 8, decision T2): no alliance document; the
// alliance ID is the leader clan's ID and every member clan stores the
// alliance's ID, name and crest and its own alliance penalty. Creation and
// dissolution at the village master (Mobius Clan.createAlly,
// Clan.dissolveAlly), invitations (RequestJoinAlly, RequestAnswerJoinAlly with
// the request partner slot, Clan.checkAllyJoinCondition), leaving
// (AllyLeave), dismissal (AllyDismiss), dissolution (RequestDismissAlly), the
// alliance window (RequestAllyInfo) and alliance chat (ChatAlliance). The
// alliance crest is in alliance_crest.go.

func init() {
	registerFeature("clanally", setupAlliance)
	registerVillageMasterCommand("create_ally", func(s *Server, p *Player, _ *Npc, args []string) {
		if len(args) == 0 || args[0] == "" {
			return
		}
		if p.clan.clan == nil {
			p.SendSystemMessage(allyMsgOnlyLeadersCreate)
			return
		}
		s.CreateAlly(p, args[0])
	})
	registerVillageMasterCommand("dissolve_ally", func(s *Server, p *Player, _ *Npc, _ []string) {
		if p.clan.clan == nil {
			return // Mobius throws on the missing clan
		}
		s.DissolveAlly(p)
	})
}

// System messages of the alliances (Mobius SystemMessageId).
const (
	allyMsgOnlyAllyLeaders       serverpackets.SystemMessageID = 464
	allyMsgNotAllied             serverpackets.SystemMessageID = 465
	allyMsgLimitExceeded         serverpackets.SystemMessageID = 466
	allyMsgNoAcceptAfterExpel    serverpackets.SystemMessageID = 467
	allyMsgNoEnterAfterExpelled  serverpackets.SystemMessageID = 468
	allyMsgNoAllyAtWar           serverpackets.SystemMessageID = 469
	allyMsgOnlyLeaderWithdraws   serverpackets.SystemMessageID = 470
	allyMsgLeaderCannotWithdraw  serverpackets.SystemMessageID = 471
	allyMsgDifferentAlliance     serverpackets.SystemMessageID = 473
	allyMsgClanDoesNotExist      serverpackets.SystemMessageID = 474
	allyMsgCrestSize             serverpackets.SystemMessageID = 476
	allyMsgInviteCancelled       serverpackets.SystemMessageID = 477
	allyMsgEntranceCancelled     serverpackets.SystemMessageID = 478
	allyMsgInfoHeader            serverpackets.SystemMessageID = 491
	allyMsgInfoName              serverpackets.SystemMessageID = 492
	allyMsgInfoConnection        serverpackets.SystemMessageID = 493
	allyMsgInfoLeader            serverpackets.SystemMessageID = 494
	allyMsgInfoClans             serverpackets.SystemMessageID = 495
	allyMsgInfoClanHeader        serverpackets.SystemMessageID = 496
	allyMsgInfoClanName          serverpackets.SystemMessageID = 497
	allyMsgInfoClanLeader        serverpackets.SystemMessageID = 498
	allyMsgInfoClanLevel         serverpackets.SystemMessageID = 499
	allyMsgInfoSeparator         serverpackets.SystemMessageID = 500
	allyMsgInfoFooter            serverpackets.SystemMessageID = 501
	allyMsgAlreadyInAlliance     serverpackets.SystemMessageID = 502
	allyMsgOnlyLeadersCreate     serverpackets.SystemMessageID = 504
	allyMsgCreateAfterDissolve   serverpackets.SystemMessageID = 505
	allyMsgIncorrectName         serverpackets.SystemMessageID = 506
	allyMsgIncorrectNameLength   serverpackets.SystemMessageID = 507
	allyMsgNameExists            serverpackets.SystemMessageID = 508
	allyMsgAccepted              serverpackets.SystemMessageID = 517
	allyMsgWithdrawn             serverpackets.SystemMessageID = 519
	allyMsgExpelSucceeded        serverpackets.SystemMessageID = 521
	allyMsgDissolved             serverpackets.SystemMessageID = 523
	allyMsgInvitedFriend         serverpackets.SystemMessageID = 525
	allyMsgRequested             serverpackets.SystemMessageID = 527
	allyMsgCreateNeedsLevel5     serverpackets.SystemMessageID = 549
	allyMsgCreateWhileDissolving serverpackets.SystemMessageID = 550
	allyMsgClanInOtherAlliance   serverpackets.SystemMessageID = 691
	allyMsgDissolveInSiege       serverpackets.SystemMessageID = 722
	allyMsgOpposingInSiege       serverpackets.SystemMessageID = 723
	allyMsgJoinAfterLeaving      serverpackets.SystemMessageID = 761
)

// allyMinClanLevel is the clan level an alliance needs (Clan.createAlly).
const allyMinClanLevel = 5

// allyInvite is the request partner value of an alliance invitation (Mobius
// RequestJoinAlly as getRequestPacket).
type allyInvite struct{}

// setupAlliance registers the alliance client packets.
func setupAlliance(s *Server) error {
	var errs []error
	for _, r := range []struct {
		op byte
		h  Handler
	}{
		{clientpackets.OpRequestJoinAlly, s.handleJoinAlly},
		{clientpackets.OpRequestAnswerJoinAlly, s.handleAnswerJoinAlly},
		{clientpackets.OpAllyLeave, s.handleAllyLeave},
		{clientpackets.OpAllyDismiss, s.handleAllyDismiss},
		{clientpackets.OpRequestDismissAlly, s.handleDismissAlly},
		{clientpackets.OpRequestAllyInfo, s.handleAllyInfo},
		{clientpackets.OpRequestSetAllyCrest, s.handleSetAllyCrest},
		{clientpackets.OpRequestAllyCrest, s.handleAllyCrest},
	} {
		errs = append(errs, s.Register(r.op, r.h))
	}
	return errors.Join(errs...)
}

// AllyID returns the alliance of the player's clan, 0 without one (Mobius
// Player.getAllyId). Loop-only.
func (p *Player) AllyID() int32 {
	if c := p.clan.clan; c != nil {
		return c.AllyID
	}
	return 0
}

// IsAllyLeader reports whether the clan leads its alliance.
func (c *Clan) IsAllyLeader() bool { return c.AllyID != 0 && c.AllyID == c.ID }

// AllyClans returns the clans of the alliance in ID order, none for 0
// (Mobius ClanTable.getClanAllies). Loop-only.
func (s *Server) AllyClans(allyID int32) []*Clan {
	if allyID == 0 {
		return nil
	}
	var list []*Clan
	for _, c := range sortedClans(s.clan.clans) {
		if c.AllyID == allyID {
			list = append(list, c)
		}
	}
	return list
}

// allyNameTaken is ClanTable.isAllyExists: a clan has an alliance of that
// name, ignoring case.
func (s *Server) allyNameTaken(name string) bool {
	key := social.NameKey(name)
	for _, c := range s.clan.clans {
		if c.AllyName != "" && social.NameKey(c.AllyName) == key {
			return true
		}
	}
	return false
}

// broadcastToAlly sends body to the members online of every clan of the
// alliance (Clan.broadcastToOnlineAllyMembers).
func (s *Server) broadcastToAlly(allyID int32, body []byte) {
	for _, c := range s.AllyClans(allyID) {
		broadcastToClan(c, body)
	}
}

// leaveAlly takes c out of its alliance with the alliance penalty typ
// (social.AllyPenalty*, AllyPenaltyNone for none). The caller changes the
// alliance crest (applyAllyCrest), which saves c.
func (s *Server) leaveAlly(c *Clan, typ int) {
	c.AllyID, c.AllyName = 0, ""
	c.AllyPenalty = storage.AllyPenalty{Expiry: s.clan.days.AllyPenalty(s.Now().UnixMilli(), typ), Type: int32(typ)} //nolint:gosec // small
}

// ----- creation and dissolution -----

// CreateAlly creates an alliance led by p's clan (Mobius Clan.createAlly):
// the leader of a clan of level 5 or more without an alliance, without the
// penalty of a dissolved alliance and not dissolving, with a name of 2 to 16
// letters or digits that no alliance has. It reports whether the alliance
// was created. Loop-only.
func (s *Server) CreateAlly(p *Player, name string) bool {
	c := p.clan.clan
	now := s.Now().UnixMilli()
	switch {
	case c == nil || !p.IsClanLeader():
		p.SendSystemMessage(allyMsgOnlyLeadersCreate)
		return false
	case c.AllyID != 0:
		p.SendSystemMessage(allyMsgAlreadyInAlliance)
		return false
	case c.Level < allyMinClanLevel:
		p.SendSystemMessage(allyMsgCreateNeedsLevel5)
		return false
	case social.Active(c.AllyPenalty.Expiry, now) && c.AllyPenalty.Type == social.AllyPenaltyDissolved:
		p.SendSystemMessage(allyMsgCreateAfterDissolve)
		return false
	case social.Active(c.DissolvingExpiry, now):
		p.SendSystemMessage(allyMsgCreateWhileDissolving)
		return false
	}
	switch social.CheckAllyName(name) {
	case social.NameInvalid:
		p.SendSystemMessage(allyMsgIncorrectName)
		return false
	case social.NameBadLength:
		p.SendSystemMessage(allyMsgIncorrectNameLength)
		return false
	}
	if s.allyNameTaken(name) {
		p.SendSystemMessage(allyMsgNameExists)
		return false
	}
	c.AllyID, c.AllyName = c.ID, name
	c.AllyPenalty = storage.AllyPenalty{}
	s.saveClan(c)
	p.UpdateUserInfo()
	// Mobius: "TODO: Need correct message id".
	p.SendMessage("Alliance " + name + " has been created.")
	s.log.Info("alliance created", append(p.logArgs(), "clan", c.Name, "alliance", name)...)
	return true
}

// DissolveAlly dissolves the alliance p's clan leads (Mobius
// Clan.dissolveAlly): the members online of every clan are told, the other
// clans leave it without a penalty, the leader clan gets the penalty of a
// dissolved alliance and the alliance crest is deleted. Loop-only.
//
// Deviation: Mobius resets the alliance crest only in the database (and
// there for every clan without an alliance); here every former member clan
// loses it in memory too, and their members online broadcast their
// UserInfo.
func (s *Server) DissolveAlly(p *Player) {
	c := p.clan.clan
	switch {
	case c == nil:
		return
	case c.AllyID == 0:
		p.SendSystemMessage(allyMsgNotAllied)
		return
	case !p.IsClanLeader() || c.ID != c.AllyID:
		p.SendSystemMessage(allyMsgOnlyAllyLeaders)
		return
	case p.Zones.Inside(ZoneSiege):
		p.SendSystemMessage(allyMsgDissolveInSiege)
		return
	}
	allyID, name := c.AllyID, c.AllyName
	s.broadcastToAlly(allyID, clanSysMsg(allyMsgDissolved))
	var b storage.Batch
	s.dropCrest(c.AllyCrestID, &b)
	clans := s.AllyClans(allyID)
	for _, o := range clans {
		if o == c {
			s.leaveAlly(o, social.AllyPenaltyDissolved)
		} else {
			s.leaveAlly(o, social.AllyPenaltyNone)
		}
		o.AllyCrestID = 0
		b.Clans = append(b.Clans, o.document())
	}
	s.SaveJoint(nil, b)
	for _, o := range clans {
		broadcastClanUserInfo(o)
	}
	s.log.Info("alliance dissolved", append(p.logArgs(), "clan", c.Name, "alliance", name)...)
}

// handleDismissAlly handles RequestDismissAlly (0x86): the alliance leader
// dissolves the alliance.
func (s *Server) handleDismissAlly(p *Player, _ []byte) {
	if !p.IsClanLeader() {
		p.SendSystemMessage(allyMsgOnlyAllyLeaders)
		return
	}
	s.DissolveAlly(p)
}

// ----- invitations -----

// handleJoinAlly handles RequestJoinAlly (0x82): the alliance leader invites
// another clan leader (AskJoinAlly), through the request partner slot.
func (s *Server) handleJoinAlly(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestJoinAlly(body)
	if err != nil {
		s.log.Debug("game: bad RequestJoinAlly", append(p.logArgs(), "err", err)...)
		return
	}
	target, ok := s.onlinePlayer(req.TargetID)
	if !ok {
		p.SendSystemMessage(clanMsgWrongTarget)
		return
	}
	c := p.clan.clan
	if c == nil {
		p.SendSystemMessage(clanMsgNotClanMember)
		return
	}
	if !s.checkAllyJoin(p, target) || !s.SetRequest(p, target, allyInvite{}) {
		return
	}
	target.SendSystemMessage(allyMsgRequested, serverpackets.TextParam(c.AllyName), serverpackets.TextParam(p.Name()))
	target.Send(serverpackets.NewAskJoinAllyPacket(p.ObjectID(), c.AllyName))
}

// checkAllyJoin is Mobius Clan.checkAllyJoinCondition: p, the alliance
// leader, may invite target's clan.
func (s *Server) checkAllyJoin(p, target *Player) bool {
	c := p.clan.clan
	now := s.Now().UnixMilli()
	if c == nil || c.AllyID == 0 || !p.IsClanLeader() || c.ID != c.AllyID {
		p.SendSystemMessage(allyMsgOnlyAllyLeaders)
		return false
	}
	if social.Active(c.AllyPenalty.Expiry, now) && c.AllyPenalty.Type == social.AllyPenaltyDismissedClan {
		p.SendSystemMessage(allyMsgNoAcceptAfterExpel)
		return false
	}
	switch {
	case target == nil:
		p.SendSystemMessage(clanMsgWrongTarget)
		return false
	case p == target:
		p.SendSystemMessage(clanMsgCannotAskYourself)
		return false
	case target.clan.clan == nil:
		p.SendSystemMessage(clanMsgTargetMustBeMember)
		return false
	case !target.IsClanLeader():
		p.SendSystemMessage(clanMsgS1IsNotClanLeader, serverpackets.TextParam(target.Name()))
		return false
	}
	tc := target.clan.clan
	if tc.AllyID != 0 {
		p.SendSystemMessage(allyMsgClanInOtherAlliance, serverpackets.TextParam(tc.Name), serverpackets.TextParam(tc.AllyName))
		return false
	}
	if social.Active(tc.AllyPenalty.Expiry, now) {
		switch tc.AllyPenalty.Type {
		case social.AllyPenaltyLeft:
			p.SendSystemMessage(allyMsgJoinAfterLeaving, serverpackets.TextParam(tc.Name), serverpackets.TextParam(tc.AllyName))
			return false
		case social.AllyPenaltyDismissed:
			p.SendSystemMessage(allyMsgNoEnterAfterExpelled)
			return false
		}
	}
	switch {
	case p.Zones.Inside(ZoneSiege) && target.Zones.Inside(ZoneSiege):
		p.SendSystemMessage(allyMsgOpposingInSiege)
		return false
	case c.AtWarWith(tc.ID):
		p.SendSystemMessage(allyMsgNoAllyAtWar)
		return false
	case len(s.AllyClans(c.AllyID)) >= s.gs.Ally.MaxClans:
		p.SendSystemMessage(allyMsgLimitExceeded)
		return false
	}
	return true
}

// handleAnswerJoinAlly handles RequestAnswerJoinAlly (0x83): 0 declines;
// any other answer joins the alliance when the conditions still hold. The
// clan takes the alliance's ID, name and crest and loses its penalty.
func (s *Server) handleAnswerJoinAlly(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestAnswerJoinAlly(body)
	if err != nil {
		s.log.Debug("game: bad RequestAnswerJoinAlly", append(p.logArgs(), "err", err)...)
		return
	}
	requestor, packet := s.RequestPartner(p)
	if requestor == nil {
		return
	}
	if req.Answer == 0 {
		p.SendSystemMessage(allyMsgEntranceCancelled)
		requestor.SendSystemMessage(allyMsgInviteCancelled)
	} else {
		if _, ok := packet.(allyInvite); !ok {
			return // Mobius: "hax"
		}
		rc, c := requestor.clan.clan, p.clan.clan
		if rc != nil && c != nil && s.checkAllyJoin(requestor, p) {
			// Mobius: "TODO: Need correct message id".
			requestor.SendSystemMessage(allyMsgInvitedFriend)
			p.SendSystemMessage(allyMsgAccepted)
			c.AllyID, c.AllyName = rc.AllyID, rc.AllyName
			c.AllyPenalty = storage.AllyPenalty{}
			s.applyAllyCrest(c, rc.AllyCrestID, true, storage.Batch{})
			s.log.Info("clan joined an alliance", append(p.logArgs(), "clan", c.Name, "alliance", c.AllyName)...)
		}
	}
	s.OnRequestResponse(p)
}

// ----- leaving and dismissal -----

// handleAllyLeave handles AllyLeave (0x84): the leader of a member clan,
// not the alliance leader, takes its clan out, with the penalty of a clan
// that left.
func (s *Server) handleAllyLeave(p *Player, _ []byte) {
	c := p.clan.clan
	switch {
	case c == nil:
		p.SendSystemMessage(clanMsgNotClanMember)
		return
	case !p.IsClanLeader():
		p.SendSystemMessage(allyMsgOnlyLeaderWithdraws)
		return
	case c.AllyID == 0:
		p.SendSystemMessage(allyMsgNotAllied)
		return
	case c.ID == c.AllyID:
		p.SendSystemMessage(allyMsgLeaderCannotWithdraw)
		return
	}
	name := c.AllyName
	s.leaveAlly(c, social.AllyPenaltyLeft)
	s.applyAllyCrest(c, 0, true, storage.Batch{})
	p.SendSystemMessage(allyMsgWithdrawn)
	s.log.Info("clan left its alliance", append(p.logArgs(), "clan", c.Name, "alliance", name)...)
}

// handleAllyDismiss handles AllyDismiss (0x85): the alliance leader expels a
// member clan; the clan gets the penalty of a dismissed clan, the leader's
// clan that of having dismissed one.
func (s *Server) handleAllyDismiss(p *Player, body []byte) {
	req, err := clientpackets.ParseAllyDismiss(body)
	if err != nil {
		s.log.Debug("game: bad AllyDismiss", append(p.logArgs(), "err", err)...)
		return
	}
	c := p.clan.clan
	switch {
	case c == nil:
		p.SendSystemMessage(clanMsgNotClanMember)
		return
	case c.AllyID == 0:
		p.SendSystemMessage(allyMsgNotAllied)
		return
	case !p.IsClanLeader() || c.ID != c.AllyID:
		p.SendSystemMessage(allyMsgOnlyAllyLeaders)
		return
	}
	t := s.ClanByName(req.ClanName)
	switch {
	case t == nil:
		p.SendSystemMessage(allyMsgClanDoesNotExist)
		return
	case t == c:
		p.SendSystemMessage(allyMsgLeaderCannotWithdraw)
		return
	case t.AllyID != c.AllyID:
		p.SendSystemMessage(allyMsgDifferentAlliance)
		return
	}
	now := s.Now().UnixMilli()
	c.AllyPenalty = storage.AllyPenalty{Expiry: s.clan.days.AllyPenalty(now, social.AllyPenaltyDismissedClan), Type: social.AllyPenaltyDismissedClan}
	s.leaveAlly(t, social.AllyPenaltyDismissed)
	s.applyAllyCrest(t, 0, true, storage.Batch{Clans: []storage.Clan{c.document()}})
	p.SendSystemMessage(allyMsgExpelSucceeded)
	s.log.Info("clan dismissed from an alliance", append(p.logArgs(), "clan", t.Name, "alliance", c.AllyName)...)
}

// ----- the alliance window and chat -----

// handleAllyInfo handles RequestAllyInfo (0x8E): AllianceInfo, then the
// alliance as system messages, in Mobius's order (a separator between the
// clans, none after the last).
func (s *Server) handleAllyInfo(p *Player, _ []byte) {
	allyID := p.AllyID()
	leader := s.Clan(allyID)
	if allyID <= 0 || leader == nil {
		p.SendSystemMessage(allyMsgNotAllied)
		return
	}
	clans := s.AllyClans(allyID)
	info := serverpackets.AllianceInfo{Name: leader.AllyName, LeaderClan: leader.Name, LeaderName: leader.LeaderName()}
	var total, online int32
	for _, c := range clans {
		ac := serverpackets.AllianceClan{
			Name:       c.Name,
			Level:      int32(c.Level), //nolint:gosec // small
			LeaderName: c.LeaderName(),
			Members:    int32(c.MemberCount()),        //nolint:gosec // small
			Online:     int32(len(c.OnlineMembers())), //nolint:gosec // small
		}
		total += ac.Members
		online += ac.Online
		info.Clans = append(info.Clans, ac)
	}
	p.Send(serverpackets.NewAllianceInfoPacket(info))
	text, num := serverpackets.TextParam, serverpackets.NumberParam
	p.SendSystemMessage(allyMsgInfoHeader)
	p.SendSystemMessage(allyMsgInfoName, text(info.Name))
	p.SendSystemMessage(allyMsgInfoLeader, text(info.LeaderClan), text(info.LeaderName))
	p.SendSystemMessage(allyMsgInfoConnection, num(online), num(total))
	p.SendSystemMessage(allyMsgInfoClans, num(int32(len(clans)))) //nolint:gosec // at most MaxClans
	for i, ac := range info.Clans {
		if i == 0 {
			p.SendSystemMessage(allyMsgInfoClanHeader)
		} else {
			p.SendSystemMessage(allyMsgInfoSeparator)
		}
		p.SendSystemMessage(allyMsgInfoClanName, text(ac.Name))
		p.SendSystemMessage(allyMsgInfoClanLeader, text(ac.LeaderName))
		p.SendSystemMessage(allyMsgInfoClanLevel, num(ac.Level))
		p.SendSystemMessage(allyMsgInfoConnection, num(ac.Online), num(ac.Members))
	}
	p.SendSystemMessage(allyMsgInfoFooter)
}

// chatAlliance sends alliance chat (Mobius ChatAlliance): to the members
// online of every clan of the alliance. As in Mobius, block lists do not
// apply (Clan.broadcastToOnlineAllyMembers).
func (s *Server) chatAlliance(p *Player, text string) {
	c := p.clan.clan
	if c == nil || c.AllyID == 0 {
		p.SendMessage("You are not in an alliance.")
		return
	}
	s.broadcastToAlly(c.AllyID, chatSay(p, serverpackets.ChatAlliance, p.Name(), text))
}

// sameClanOrAlly reports whether a and b are in the same clan, or in clans
// of the same alliance (Player.isAutoAttackable's clan and alliance rule).
func sameClanOrAlly(a, b *Player) bool {
	ac, bc := a.clan.clan, b.clan.clan
	if ac == nil || bc == nil {
		return false
	}
	return ac == bc || (ac.AllyID != 0 && ac.AllyID == bc.AllyID)
}
