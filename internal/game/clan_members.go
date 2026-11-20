package game

import (
	"strings"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
)

// Clan membership: invitations (RequestJoinPledge, RequestAnswerJoinPledge
// with the request partner slot), withdrawal, dismissal
// (Clan.removeClanMember), the online state and log-in notices (EnterWorld
// and Player.deleteMe), clan chat and the member list updates on level and
// class changes.

// clanInvite is the request partner value of a clan invitation (Mobius
// RequestJoinPledge as getRequestPacket).
type clanInvite struct {
	pledgeType int
}

// registerClanPackets registers the clan client packets.
func (s *Server) registerClanPackets() error {
	regs := []struct {
		op byte
		h  Handler
	}{
		{clientpackets.OpRequestJoinPledge, s.handleJoinPledge},
		{clientpackets.OpRequestAnswerJoinPledge, s.handleAnswerJoinPledge},
		{clientpackets.OpRequestWithdrawalPledge, s.handleWithdrawalPledge},
		{clientpackets.OpRequestOustPledgeMember, s.handleOustPledgeMember},
		{clientpackets.OpRequestPledgeMemberList, s.handlePledgeMemberList},
		{clientpackets.OpRequestPledgeInfo, s.handlePledgeInfo},
		{clientpackets.OpRequestPledgeExtendedInfo, func(*Player, []byte) {}}, // Mobius reads it and does nothing
		{clientpackets.OpRequestGiveNickName, s.handleGiveNickName},
		{clientpackets.OpRequestPledgePower, s.handlePledgePower},
	}
	for _, r := range regs {
		if err := s.Register(r.op, r.h); err != nil {
			return err
		}
	}
	exRegs := []struct {
		op uint16
		h  Handler
	}{
		{clientpackets.ExOpRequestPledgePowerGradeList, s.handlePledgePowerGradeList},
		{clientpackets.ExOpRequestPledgeMemberPowerInfo, s.handlePledgeMemberPowerInfo},
		{clientpackets.ExOpRequestPledgeSetMemberPowerGrade, s.handlePledgeSetMemberPowerGrade},
		{clientpackets.ExOpRequestPledgeMemberInfo, s.handlePledgeMemberInfo},
		{clientpackets.ExOpRequestPledgeReorganizeMember, s.handlePledgeReorganizeMember},
	}
	for _, r := range exRegs {
		if err := s.RegisterEx(r.op, r.h); err != nil {
			return err
		}
	}
	return nil
}

// ----- invitations -----

// handleJoinPledge handles RequestJoinPledge (0x24): a member with the
// invite privilege asks a player into the clan or one of its units
// (AskJoinPledge), through the request partner slot.
func (s *Server) handleJoinPledge(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestJoinPledge(body)
	if err != nil {
		s.log.Debug("game: bad RequestJoinPledge", append(p.logArgs(), "err", err)...)
		return
	}
	c := p.clan.clan
	if c == nil {
		return
	}
	target, ok := s.onlinePlayer(req.TargetID)
	if !ok {
		p.SendSystemMessage(clanMsgWrongTarget)
		return
	}
	unit := int(req.PledgeType)
	if !s.checkClanJoin(c, p, target, unit) {
		return
	}
	if !s.SetRequest(p, target, clanInvite{pledgeType: unit}) {
		return
	}
	ask := serverpackets.AskJoinPledge{RequestorID: p.ObjectID(), PledgeType: req.PledgeType, ClanName: c.Name}
	if u := c.Unit(unit); u != nil {
		ask.SubPledgeName = u.Name
	}
	target.Send(serverpackets.NewAskJoinPledgePacket(ask))
}

// checkClanJoin is Mobius Clan.checkClanJoinCondition: p may invite target
// into unit of c.
func (s *Server) checkClanJoin(c *Clan, p, target *Player, unit int) bool {
	now := s.Now().UnixMilli()
	switch {
	case !p.HasClanAccess(social.AccessInviteMember):
		p.SendSystemMessage(clanMsgNotAuthorized)
	case target == nil:
		p.SendSystemMessage(clanMsgWrongTarget)
	case p == target:
		p.SendSystemMessage(clanMsgCannotAskYourself)
	case social.Active(c.MemberPenaltyExpiry, now):
		p.SendSystemMessage(clanMsgClanWaitAfterDismiss)
	case target.char.ClanID != 0:
		p.SendSystemMessage(clanMsgS1WorkingWithAnotherClan, serverpackets.TextParam(target.Name()))
	case social.Active(target.char.ClanJoinExpiry, now):
		p.SendSystemMessage(clanMsgS1JoinPenalty, serverpackets.TextParam(target.Name()))
	case unit == social.UnitAcademy && !social.CanJoinAcademy(target.Level(), clanClassLevel(target.char.ClassID)):
		p.SendSystemMessage(clanMsgS1NotForAcademy, serverpackets.TextParam(target.Name()))
		p.SendSystemMessage(clanMsgAcademyRequirements)
	case c.UnitMemberCount(unit) >= social.MaxMembers(c.Level, unit):
		if unit == social.UnitMain {
			p.SendSystemMessage(clanMsgS1IsFull, serverpackets.TextParam(c.Name))
		} else {
			p.SendSystemMessage(clanMsgUnitFull)
		}
	default:
		return true
	}
	return false
}

// handleAnswerJoinPledge handles RequestAnswerJoinPledge (0x25): 0
// declines; any other answer joins the clan when the conditions still hold
// (rank 5, the academy rank 9 with the join level).
func (s *Server) handleAnswerJoinPledge(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestAnswerJoinPledge(body)
	if err != nil {
		s.log.Debug("game: bad RequestAnswerJoinPledge", append(p.logArgs(), "err", err)...)
		return
	}
	requestor, packet := s.RequestPartner(p)
	if requestor == nil {
		return
	}
	if req.Answer == 0 {
		p.SendSystemMessage(clanMsgYouDidNotRespond, serverpackets.TextParam(requestor.Name()))
		requestor.SendSystemMessage(clanMsgS1DidNotRespond, serverpackets.TextParam(p.Name()))
	} else {
		inv, ok := packet.(clanInvite)
		if !ok {
			return // Mobius: "hax"
		}
		if c := requestor.clan.clan; c != nil && s.checkClanJoin(c, requestor, p, inv.pledgeType) {
			s.joinClan(c, p, inv.pledgeType)
		}
	}
	s.OnRequestResponse(p)
}

// joinClan adds p to unit of c (RequestAnswerJoinPledge and
// Clan.addClanMember), with Mobius's packets in order. p is saved.
func (s *Server) joinClan(c *Clan, p *Player, unit int) {
	p.Send(serverpackets.NewJoinPledgePacket(c.ID))
	p.char.PledgeType = int32(unit) //nolint:gosec // small
	if unit == social.UnitAcademy {
		p.char.PowerGrade = social.RankAcademy
		p.char.LvlJoinedAcademy = int32(p.Level()) //nolint:gosec // small
	} else {
		p.char.PowerGrade = social.RankNewMember
	}
	// Clan.addClanMember.
	p.char.ClanID = c.ID
	p.char.Apprentice, p.char.Sponsor = 0, 0
	m := &ClanMember{data: storage.ClanMember{ObjectID: p.ObjectID()}}
	c.addMember(m)
	m.setPlayer(p)
	p.clan.clan = c
	s.refreshPledgeClass(p)
	p.Send(memberUpdatePacket(m))
	p.Send(s.pledgeSkillListPacket(c))
	s.addClanSkillEffects(p)
	// RequestAnswerJoinPledge.
	p.clan.privs = c.Ranks.Of(p.PowerGrade())
	p.SendSystemMessage(clanMsgEnteredClan)
	broadcastToClan(c, clanSysMsg(clanMsgS1JoinedClan, serverpackets.TextParam(p.Name())))
	p.SendSkillList()
	broadcastToOtherClanMembers(c, serverpackets.NewPledgeShowMemberListAddPacket(pledgeMember(m)), p)
	s.broadcastClanInfo(c)
	s.sendClanWindow(p, c)
	p.char.ClanJoinExpiry = 0
	p.BroadcastUserInfo()
	s.RequestSave(p)
	s.log.Info("clan member joined", append(p.logArgs(), "clan", c.Name, "unit", unit)...)
}

// pledgeSkillListPacket is PledgeSkillList of c (Mobius PledgeSkillList):
// the clan skills come with Phase 7b, which sets clanSkillListHook.
func (s *Server) pledgeSkillListPacket(c *Clan) []byte {
	if clanSkillListHook != nil {
		return clanSkillListHook(s, c)
	}
	return serverpackets.NewPledgeSkillListPacket(nil)
}

// Hooks for the clan skills (Phase 7b): the PledgeSkillList of a clan, and
// giving or taking a member's clan skills (Mobius addSkillEffects and
// removeSkillEffects).
var (
	clanSkillListHook   func(s *Server, c *Clan) []byte
	clanSkillEffectHook func(s *Server, p *Player, add bool)
)

func (s *Server) addClanSkillEffects(p *Player) {
	if clanSkillEffectHook != nil {
		clanSkillEffectHook(s, p, true)
	}
}

func (s *Server) removeClanSkillEffects(p *Player) {
	if clanSkillEffectHook != nil {
		clanSkillEffectHook(s, p, false)
	}
}

// ----- leaving -----

// handleWithdrawalPledge handles RequestWithdrawalPledge (0x26): a member
// but the leader leaves the clan, not in combat, with the join penalty.
func (s *Server) handleWithdrawalPledge(p *Player, _ []byte) {
	c := p.clan.clan
	switch {
	case c == nil:
		p.SendSystemMessage(clanMsgNotClanMember)
		return
	case p.IsClanLeader():
		p.SendSystemMessage(clanMsgLeaderCannotWithdraw)
		return
	case s.InCombat(p):
		p.SendSystemMessage(clanMsgNoLeaveInCombat)
		return
	}
	name := p.Name()
	expiry := social.AfterDays(s.Now().UnixMilli(), s.clan.days.JoinClan)
	patches := s.removeClanMember(c, p.ObjectID(), expiry)
	s.SaveJoint([]*Player{p}, storage.Batch{Clans: []storage.Clan{c.document()}, Patches: patches})
	broadcastToClan(c, clanSysMsg(clanMsgS1Withdrew, serverpackets.TextParam(name)))
	broadcastToClan(c, serverpackets.NewPledgeShowMemberListDeletePacket(name))
	p.SendSystemMessage(clanMsgYouWithdrew)
	p.SendSystemMessage(clanMsgWaitAfterLeaving)
	s.log.Info("clan member withdrew", append(p.logArgs(), "clan", c.Name)...)
}

// handleOustPledgeMember handles RequestOustPledgeMember (0x27): a member
// with the dismiss privilege dismisses another, not in combat; the member
// gets the join penalty and the clan the accept penalty. An offline member
// is changed with a patch.
func (s *Server) handleOustPledgeMember(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestOustPledgeMember(body)
	if err != nil {
		s.log.Debug("game: bad RequestOustPledgeMember", append(p.logArgs(), "err", err)...)
		return
	}
	c := p.clan.clan
	switch {
	case c == nil:
		p.SendSystemMessage(clanMsgNotClanMember)
		return
	case !p.HasClanAccess(social.AccessRemoveMember):
		p.SendSystemMessage(clanMsgNotAuthorized)
		return
	case strings.EqualFold(p.Name(), req.Name):
		p.SendSystemMessage(clanMsgCannotDismissYourself)
		return
	}
	m := c.MemberByName(req.Name)
	if m == nil {
		s.log.Warn("game: dismissal of a player who is not in the clan", append(p.logArgs(), "target", req.Name)...)
		return
	}
	if m.ObjectID() == c.LeaderID {
		// Deviation: Mobius lets a member with the privilege dismiss the
		// leader, which leaves the clan without one.
		p.SendSystemMessage(clanMsgNotAuthorized)
		return
	}
	target := m.Player()
	if m.Online() && s.InCombat(target) {
		p.SendSystemMessage(clanMsgNoDismissInCombat)
		return
	}
	name := m.Name()
	now := s.Now().UnixMilli()
	expiry := social.AfterDays(now, s.clan.days.JoinClan)
	patches := s.removeClanMember(c, m.ObjectID(), expiry)
	c.MemberPenaltyExpiry = expiry
	var players []*Player
	if target != nil {
		players = append(players, target)
	}
	s.SaveJoint(players, storage.Batch{Clans: []storage.Clan{c.document()}, Patches: patches})
	broadcastToClan(c, clanSysMsg(clanMsgMemberExpelled, serverpackets.TextParam(name)))
	p.SendSystemMessage(clanMsgExpelSucceeded)
	p.SendSystemMessage(clanMsgClanWaitAfterDismiss)
	broadcastToClan(c, serverpackets.NewPledgeShowMemberListDeletePacket(req.Name))
	if target != nil && target.InGame() {
		target.SendSystemMessage(clanMsgRecentlyDismissed)
	}
	s.log.Info("clan member dismissed", append(p.logArgs(), "clan", c.Name, "member", name)...)
}

// removeClanMember is Mobius Clan.removeClanMember: the member leaves c; a
// unit it led loses its leader; its apprentice or sponsor is released. A
// member online loses its title, clan skills and clan fields, gets the
// join penalty joinExpiry (none in the academy) and, as leader, the create
// penalty; it gets its skill list, UserInfo and an empty clan window, and is
// saved by the caller (in SaveJoint's players). An offline member, and an
// offline apprentice or sponsor, are changed with the patches returned; the
// caller writes them with the clan.
//
// Deviations: Mobius's academy exemption reads the unit after clearing it,
// so it never applies online; here it does, online and offline. Offline,
// Mobius also clears the create penalty of a member who is not the leader;
// here it stays.
func (s *Server) removeClanMember(c *Clan, id int32, joinExpiry int64) []storage.CharacterPatch {
	m := c.removeMember(id)
	if m == nil {
		s.log.Warn("game: clan member not found for removal", "clan", c.Name, "objectId", id)
		return nil
	}
	var patches []storage.CharacterPatch
	if unit := c.UnitLedBy(id); unit != 0 {
		c.Unit(unit).LeaderID = 0
	}
	release := func(otherID int32, apprentice bool) {
		o := c.member(otherID)
		if o == nil {
			return
		}
		if q := o.Player(); q != nil {
			if apprentice {
				q.char.Sponsor = 0
			} else {
				q.char.Apprentice = 0
			}
			s.RequestSave(q)
			return
		}
		patches = append(patches, o.patch(storage.PatchApprentice|storage.PatchSponsor, storage.PatchValues{}))
	}
	if a := m.Apprentice(); a != 0 {
		release(a, true)
	}
	if sp := m.Sponsor(); sp != 0 {
		release(sp, false)
	}
	leader := c.LeaderID == id
	academy := m.PledgeType() == social.UnitAcademy
	now := s.Now().UnixMilli()
	if p := m.Player(); p != nil {
		m.setPlayer(nil)
		if leader {
			p.char.ClanCreateExpiry = social.AfterDays(now, s.clan.days.CreateClan)
		}
		s.removeClanSkillEffects(p)
		p.SendSkillList()
		clearClanFields(p)
		p.char.WantsPeace = false
		if !academy {
			p.char.ClanJoinExpiry = joinExpiry
		}
		s.refreshPledgeClass(p)
		p.BroadcastUserInfo()
		p.Send(serverpackets.NewPledgeShowMemberListDeleteAllPacket())
		s.RequestSave(p)
		return patches
	}
	fields := storage.PatchClanMembership
	v := storage.PatchValues{ClanJoinExpiry: joinExpiry}
	if academy {
		fields &^= storage.PatchClanJoinExpiry
	}
	if leader {
		v.ClanCreateExpiry = social.AfterDays(now, s.clan.days.CreateClan)
	} else {
		fields &^= storage.PatchClanCreateExpiry
	}
	return append(patches, m.patch(fields, v))
}

// ----- the clan window -----

// handlePledgeMemberList handles RequestPledgeMemberList (0x3C): the clan
// window.
func (s *Server) handlePledgeMemberList(p *Player, _ []byte) {
	if c := p.clan.clan; c != nil {
		s.sendClanWindow(p, c)
	}
}

// handlePledgeInfo handles RequestPledgeInfo (0x66): a clan's name and
// alliance (PledgeInfo); an unknown clan gets nothing.
func (s *Server) handlePledgeInfo(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestPledgeInfo(body)
	if err != nil {
		return
	}
	c := s.Clan(req.ClanID)
	if c == nil {
		return
	}
	p.Send(serverpackets.NewPledgeInfoPacket(c.ID, c.Name, c.AllyName))
}

// ----- the online state -----

// clanEnterWorld is the clan part of EnterWorld (EnterWorld.java 252-318),
// right after the first UserInfo: PledgeSkillList, the members online learn
// that p logged in, the sponsor or apprentice is told, then the clan window
// and PledgeStatusChanged.
func (s *Server) clanEnterWorld(p *Player) {
	c := p.clan.clan
	if c == nil {
		return
	}
	m := c.member(p.ObjectID())
	if m == nil || s.clan.clans[c.ID] != c {
		// Dismissed, or the clan went, between selection and EnterWorld.
		clearClanFields(p)
		s.refreshPledgeClass(p)
		p.UpdateUserInfo()
		return
	}
	p.Send(s.pledgeSkillListPacket(c))
	// notifyClanMembers.
	m.setPlayer(p)
	s.addClanSkillEffects(p)
	broadcastToOtherClanMembers(c, clanSysMsg(clanMsgMemberLoggedIn, serverpackets.TextParam(p.Name())), p)
	broadcastToOtherClanMembers(c, memberUpdatePacket(m), p)
	// notifySponsorOrApprentice.
	if id := p.char.Sponsor; id != 0 {
		if q, ok := s.onlinePlayer(id); ok {
			q.SendSystemMessage(clanMsgApprenticeLoggedIn, serverpackets.TextParam(p.Name()))
		}
	} else if id := p.char.Apprentice; id != 0 {
		if q, ok := s.onlinePlayer(id); ok {
			q.SendSystemMessage(clanMsgSponsorLoggedIn, serverpackets.TextParam(p.Name()))
		}
	}
	s.sendClanWindow(p, c)
	p.Send(serverpackets.NewPledgeStatusChangedPacket(serverpackets.PledgeStatusChanged{
		LeaderID: c.LeaderID, ClanID: c.ID, CrestID: c.CrestID, AllyID: c.AllyID, AllyCrestID: c.AllyCrestID,
	}))
}

// clanPlayerEntered runs with the OnPlayerEnter hooks: the clan notice
// (EnterWorld 434-442).
func (s *Server) clanPlayerEntered(p *Player) {
	c := p.clan.clan
	if c == nil || !c.Notice.Enabled {
		return
	}
	pg := s.pageFile(p, 0, "clanNotice.htm")
	pg.replace("%clan_name%", c.Name)
	pg.replace("%notice_text%", strings.NewReplacer("\r\n", "<br>", "\n", "<br>").Replace(c.Notice.Text))
	s.sendPage(p, pg)
}

// clanEnterWorldLate is the clan join penalty notice near the end of
// EnterWorld (EnterWorld 493-496).
func (s *Server) clanEnterWorldLate(p *Player) {
	if social.Active(p.char.ClanJoinExpiry, s.Now().UnixMilli()) {
		p.SendSystemMessage(clanMsgRecentlyDismissed)
	}
}

// clanPlayerLeft runs when p leaves the game (Mobius Player.deleteMe): the
// member goes offline and the others online see it.
func (s *Server) clanPlayerLeft(p *Player) {
	c := p.clan.clan
	if c == nil {
		return
	}
	m := c.member(p.ObjectID())
	if m == nil || m.player != p {
		return
	}
	m.setPlayer(nil)
	broadcastToOtherClanMembers(c, memberUpdatePacket(m), p)
}

// clanLevelChanged tells the clan about p's new level (PlayerStat.addLevel:
// updateClanMember and PledgeShowMemberListUpdate to the members online).
func (s *Server) clanLevelChanged(p *Player) {
	c := p.clan.clan
	if c == nil || p.state != stateInGame {
		return
	}
	if m := c.member(p.ObjectID()); m != nil {
		broadcastToClan(c, memberUpdatePacket(m))
	}
}

// clanClassChanged tells the clan about p's new class (Player.setClassId:
// PledgeShowMemberListUpdate to the members online).
func (s *Server) clanClassChanged(p *Player) { s.clanLevelChanged(p) }

// ----- clan chat -----

// chatClan sends clan chat (Mobius ChatClan): to the members online who do
// not block p (broadcastCSToOnlineMembers).
func (s *Server) chatClan(p *Player, text string) {
	c := p.clan.clan
	if c == nil {
		p.SendMessage("You are not in a clan.")
		return
	}
	say := chatSay(p, serverpackets.ChatClan, p.Name(), text)
	for _, q := range c.OnlineMembers() {
		if !q.Blocks(p) {
			q.Send(say)
		}
	}
}
