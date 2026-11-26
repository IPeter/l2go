package game

import (
	"errors"
	"slices"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/social"
)

// Clan wars (Plan 12 Phase 8, decision T15), one-sided as in Mobius: a clan
// declares war on another (RequestStartPledgeWar, ClanTable.storeClanWars);
// the declaration is stored on the declaring clan (WarsDeclared) and the
// other clan lists it among its Attackers. Both clans declaring on each
// other is a mutual war, which the combat rules use: members attack each
// other without Ctrl, a kill counts as PvP and moves reputation, and the
// death costs a quarter of the exp. The war list (RequestPledgeWarList) and
// the war list user commands 88-90 (ClanWarsList) show them.

func init() { registerFeature("clanwar", setupClanWar) }

// System messages of the clan wars (Mobius SystemMessageId).
const (
	warMsgSurrendered        serverpackets.SystemMessageID = 218
	warMsgNotInClan          serverpackets.SystemMessageID = 238
	warMsgListEnd            serverpackets.SystemMessageID = 490
	warMsgDeclaredAgain      serverpackets.SystemMessageID = 628
	warMsgClanWithAlliance   serverpackets.SystemMessageID = 1200
	warMsgClanNoAlliance     serverpackets.SystemMessageID = 1202
	warMsgDeclaredOnUs       serverpackets.SystemMessageID = 1561
	warMsgDeclared           serverpackets.SystemMessageID = 1562
	warMsgConditions         serverpackets.SystemMessageID = 1564
	warMsgNoSuchClan         serverpackets.SystemMessageID = 1565
	warMsgOtherStopped       serverpackets.SystemMessageID = 1566
	warMsgStopped            serverpackets.SystemMessageID = 1567
	warMsgAlliedClan         serverpackets.SystemMessageID = 1569
	warMsgListDeclared       serverpackets.SystemMessageID = 1571
	warMsgListUnderAttack    serverpackets.SystemMessageID = 1572
	warMsgListWar            serverpackets.SystemMessageID = 1612
	warMsgNoCeaseFireInFight serverpackets.SystemMessageID = 1677
)

// The war list user commands (ClanWarsList).
const (
	userCmdAttackList      = 88 // the clans this clan declared war on, without an answer
	userCmdUnderAttackList = 89 // the clans that declared war on this clan, without an answer
	userCmdWarList         = 90 // the mutual wars
)

// warMinClanLevel is the clan level both clans of a war need
// (RequestStartPledgeWar).
const warMinClanLevel = 3

// setupClanWar registers the war packets and user commands.
func setupClanWar(s *Server) error {
	var errs []error
	for _, r := range []struct {
		op byte
		h  Handler
	}{
		{clientpackets.OpRequestStartPledgeWar, s.handleStartPledgeWar},
		{clientpackets.OpRequestReplyStartPledgeWar, s.handleReplyPledgeWar},
		{clientpackets.OpRequestStopPledgeWar, s.handleStopPledgeWar},
		{clientpackets.OpRequestReplyStopPledgeWar, s.handleReplyPledgeWar},
		{clientpackets.OpRequestSurrenderPledgeWar, s.handleSurrenderPledgeWar},
		{clientpackets.OpRequestReplySurrenderPledgeWar, s.handleReplyPledgeWar},
	} {
		errs = append(errs, s.Register(r.op, r.h))
	}
	errs = append(errs,
		s.RegisterEx(clientpackets.ExOpRequestPledgeWarList, s.handlePledgeWarList),
		s.RegisterUserCommand(userCmdAttackList, func(p *Player) { s.userWarList(p, userCmdAttackList) }),
		s.RegisterUserCommand(userCmdUnderAttackList, func(p *Player) { s.userWarList(p, userCmdUnderAttackList) }),
		s.RegisterUserCommand(userCmdWarList, func(p *Player) { s.userWarList(p, userCmdWarList) }),
	)
	return errors.Join(errs...)
}

// MutualWar reports whether the clans a and b declared war on each other.
func MutualWar(a, b *Clan) bool {
	return a != nil && b != nil && a.AtWarWith(b.ID) && b.AtWarWith(a.ID)
}

// ----- declaring and ending -----

// failWar sends p the message and ActionFailed.
func failWar(p *Player, id serverpackets.SystemMessageID, params ...serverpackets.SystemMessageParam) {
	p.SendSystemMessage(id, params...)
	p.SendActionFailed()
}

// handleStartPledgeWar handles RequestStartPledgeWar (0x4D): a member with
// the war privilege declares war on a clan. Both clans need level 3 and
// clan.membersForWar members; allied clans and clans already declared on
// are refused.
func (s *Server) handleStartPledgeWar(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestStartPledgeWar(body)
	if err != nil {
		s.log.Debug("game: bad RequestStartPledgeWar", append(p.logArgs(), "err", err)...)
		return
	}
	c := p.clan.clan
	if c == nil {
		return
	}
	minMembers := s.gs.Clan.MembersForWar
	if c.Level < warMinClanLevel || c.MemberCount() < minMembers {
		failWar(p, warMsgConditions)
		return
	}
	if !p.HasClanAccess(social.AccessWarDeclaration) {
		failWar(p, clanMsgNotAuthorized)
		return
	}
	t := s.ClanByName(req.ClanName)
	switch {
	case t == nil:
		failWar(p, warMsgNoSuchClan)
		return
	case t == c:
		// Deviation: Mobius lets a clan declare war on itself, which makes
		// its members attack each other without Ctrl.
		p.SendActionFailed()
		return
	case c.AllyID != 0 && c.AllyID == t.AllyID:
		failWar(p, warMsgAlliedClan)
		return
	case t.Level < warMinClanLevel || t.MemberCount() < minMembers:
		failWar(p, warMsgConditions)
		return
	case c.AtWarWith(t.ID):
		failWar(p, warMsgDeclaredAgain, serverpackets.TextParam(t.Name))
		return
	}
	s.StartClanWar(c, t)
	s.log.Info("clan war declared", append(p.logArgs(), "clan", c.Name, "on", t.Name)...)
}

// StartClanWar stores c's war declaration on t (Mobius
// ClanTable.storeClanWars and the rebroadcast of RequestStartPledgeWar): both
// clans get their clan window, c's members are told the war began, t's that
// c declared it, and the members online of both broadcast their UserInfo,
// which sends the new relations. c is saved. Loop-only.
func (s *Server) StartClanWar(c, t *Clan) {
	if !slices.Contains(c.WarsDeclared, t.ID) {
		c.WarsDeclared = append(c.WarsDeclared, t.ID)
	}
	if !slices.Contains(t.Attackers, c.ID) {
		t.Attackers = append(t.Attackers, c.ID)
	}
	s.broadcastClanStatus(c)
	s.broadcastClanStatus(t)
	s.saveClan(c)
	broadcastToClan(c, clanSysMsg(warMsgDeclared, serverpackets.TextParam(t.Name)))
	broadcastToClan(t, clanSysMsg(warMsgDeclaredOnUs, serverpackets.TextParam(c.Name)))
	broadcastClanUserInfo(c)
	broadcastClanUserInfo(t)
}

// StopClanWar ends c's war declaration on t (Mobius
// ClanTable.deleteClanWars and the rebroadcast of RequestStopPledgeWar):
// both clans get their clan window and are told, and the members online of
// both broadcast their UserInfo. c is saved. Loop-only.
//
// Deviation: Mobius deletes the stored declarations of both clans but only
// c's in memory, so t's declaration on c lasts until the next restart;
// here t's declaration stays, in memory and stored.
func (s *Server) StopClanWar(c, t *Clan) {
	c.WarsDeclared = clanDeleteID(c.WarsDeclared, t.ID)
	t.Attackers = clanDeleteID(t.Attackers, c.ID)
	s.broadcastClanStatus(c)
	s.broadcastClanStatus(t)
	s.saveClan(c)
	broadcastToClan(c, clanSysMsg(warMsgStopped, serverpackets.TextParam(t.Name)))
	broadcastToClan(t, clanSysMsg(warMsgOtherStopped, serverpackets.TextParam(c.Name)))
	broadcastClanUserInfo(c)
	broadcastClanUserInfo(t)
}

// handleStopPledgeWar handles RequestStopPledgeWar (0x4F): a member with
// the war privilege ends the clan's war declaration, not while a member
// online is in combat stance.
func (s *Server) handleStopPledgeWar(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestStopPledgeWar(body)
	if err != nil {
		s.log.Debug("game: bad RequestStopPledgeWar", append(p.logArgs(), "err", err)...)
		return
	}
	c, t, ok := s.warTarget(p, req.ClanName)
	if !ok {
		return
	}
	if !p.HasClanAccess(social.AccessWarDeclaration) {
		p.SendSystemMessage(clanMsgNotAuthorized)
		return
	}
	for _, q := range c.OnlineMembers() {
		if s.InAttackStance(q) {
			p.SendSystemMessage(warMsgNoCeaseFireInFight)
			return
		}
	}
	s.StopClanWar(c, t)
	s.log.Info("clan war stopped", append(p.logArgs(), "clan", c.Name, "on", t.Name)...)
}

// handleSurrenderPledgeWar handles RequestSurrenderPledgeWar (0x51): any
// member ends the clan's war declaration with "You have surrendered to the
// $s1 clan.". Mobius asks for no privilege here.
//
// Deviation: Mobius does not rebroadcast the members' UserInfo after a
// surrender, so the war icons stay until something else refreshes them;
// here they go (StopClanWar).
func (s *Server) handleSurrenderPledgeWar(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestSurrenderPledgeWar(body)
	if err != nil {
		s.log.Debug("game: bad RequestSurrenderPledgeWar", append(p.logArgs(), "err", err)...)
		return
	}
	c, t, ok := s.warTarget(p, req.ClanName)
	if !ok {
		return
	}
	p.SendSystemMessage(warMsgSurrendered, serverpackets.TextParam(req.ClanName))
	s.StopClanWar(c, t)
	s.log.Info("clan war surrendered", append(p.logArgs(), "clan", c.Name, "to", t.Name)...)
}

// warTarget finds the clan of a stop or surrender: p's clan and the clan
// named, on which p's clan declared war.
func (s *Server) warTarget(p *Player, name string) (c, t *Clan, ok bool) {
	c = p.clan.clan
	if c == nil {
		return nil, nil, false
	}
	t = s.ClanByName(name)
	if t == nil {
		p.SendMessage("No such clan.")
		p.SendActionFailed()
		return nil, nil, false
	}
	if !c.AtWarWith(t.ID) {
		p.SendMessage("You aren't at war with this clan.")
		p.SendActionFailed()
		return nil, nil, false
	}
	return c, t, true
}

// handleReplyPledgeWar handles the answers to StartPledgeWar,
// StopPledgeWar and SurrenderPledgeWar (0x4E, 0x50, 0x52). Mobius acts on
// them only with an active requester, which none of the war packets sets:
// the client never sends them for a war. Deviation: an answer that comes
// while another request (a party or trade invitation) is open is ignored
// here; Mobius would start or end a war between the two clans without any
// check.
func (s *Server) handleReplyPledgeWar(p *Player, _ []byte) {
	if r := p.ActiveRequester(); r != nil {
		s.log.Debug("game: clan war answer without a war request", append(p.logArgs(), "requester", r.Name())...)
	}
}

// ----- the war lists -----

// handlePledgeWarList handles RequestPledgeWarList (D0:1E): the clans p's
// clan declared war on (tab 0), or that declared war on it (any other tab).
func (s *Server) handlePledgeWarList(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestPledgeWarList(body)
	if err != nil {
		s.log.Debug("game: bad RequestPledgeWarList", append(p.logArgs(), "err", err)...)
		return
	}
	c := p.clan.clan
	if c == nil {
		return
	}
	ids := c.WarsDeclared
	if req.Tab != serverpackets.WarListDeclared {
		ids = c.Attackers
	}
	var names []string
	for _, id := range ids {
		if o := s.Clan(id); o != nil {
			names = append(names, o.Name)
		}
	}
	p.Send(serverpackets.NewPledgeReceiveWarListPacket(req.Tab, names))
}

// userWarList is the war list user commands (Mobius ClanWarsList): a header,
// a line per clan with its alliance, and a closing line. 88 lists the clans
// p's clan declared war on that did not declare back, 89 the clans that
// declared war on it without an answer, 90 the mutual wars.
func (s *Server) userWarList(p *Player, cmd int) {
	c := p.clan.clan
	if c == nil {
		p.SendSystemMessage(warMsgNotInClan)
		return
	}
	var header serverpackets.SystemMessageID
	var list []*Clan
	switch cmd {
	case userCmdAttackList:
		header = warMsgListDeclared
		for _, id := range c.WarsDeclared {
			if o := s.Clan(id); o != nil && !o.AtWarWith(c.ID) {
				list = append(list, o)
			}
		}
	case userCmdUnderAttackList:
		header = warMsgListUnderAttack
		for _, id := range c.Attackers {
			if o := s.Clan(id); o != nil && !c.AtWarWith(o.ID) {
				list = append(list, o)
			}
		}
	default:
		header = warMsgListWar
		for _, id := range c.WarsDeclared {
			if o := s.Clan(id); o != nil && o.AtWarWith(c.ID) {
				list = append(list, o)
			}
		}
	}
	p.SendSystemMessage(header)
	for _, o := range list {
		if o.AllyID > 0 {
			p.SendSystemMessage(warMsgClanWithAlliance, serverpackets.TextParam(o.Name), serverpackets.TextParam(o.AllyName))
		} else {
			p.SendSystemMessage(warMsgClanNoAlliance, serverpackets.TextParam(o.Name))
		}
	}
	p.SendSystemMessage(warMsgListEnd)
}

// ----- combat (T15) -----

// warMember is p as the war rules see it.
func warMember(p *Player) social.WarMember {
	return social.WarMember{
		InClan:     p.clan.clan != nil,
		Unit:       p.PledgeType(),
		Academy:    p.IsAcademyMember(),
		WantsPeace: p.char.WantsPeace,
	}
}

// warAutoAttackable is Player.isAutoAttackable's war rule (Player.java
// 8435): attacker attacks target without Ctrl in a mutual war.
func warAutoAttackable(target, attacker *Player) bool {
	return social.WarAutoAttackable(warMember(target), warMember(attacker), MutualWar(target.clan.clan, attacker.clan.clan))
}

// warPvPKill is onKillUpdatePvPKarma's war rule (Player.java 5400-5405):
// killing a member of a clan at mutual war is a PvP kill.
func warPvPKill(killer, victim *Player) bool {
	return social.WarPvPKill(warMember(killer), warMember(victim), MutualWar(killer.clan.clan, victim.clan.clan))
}

// atWarWith is Mobius Player.isAtWarWith(target): p's clan declared war on
// the clan of target, a player, and neither is an academy member. A player
// killed by someone it is at war with loses a quarter of the death exp
// (Player.java 5168, 5654-5658).
func atWarWith(p *Player, target Creature) bool {
	t, ok := target.(*Player)
	if !ok || t == nil || p.clan.clan == nil || t.clan.clan == nil {
		return false
	}
	return social.AtWar(warMember(p), warMember(t), p.clan.clan.AtWarWith(t.clan.clan.ID))
}

// clanWarKill is the reputation part of Player.doDie (Player.java
// 5143-5160) for victim killed by killer: outside PvP and siege zones, both
// in clans at mutual war and neither in the academy, the killer's clan gains
// ReputationScorePerKill while the victim's clan's score is above 0, then
// the victim's clan loses it while the killer's clan's score is above 0.
// Sieges come with M11; the anti-feed check is off in Mobius's config.
func (s *Server) clanWarKill(victim *Player, killer Creature) {
	pk, ok := killer.(*Player)
	if !ok || pk == nil || pk == victim {
		return
	}
	if victim.Zones.Inside(ZonePvP) || victim.Zones.Inside(ZoneSiege) {
		return
	}
	vc, kc := victim.clan.clan, pk.clan.clan
	if vc == nil || kc == nil || victim.IsAcademyMember() || pk.IsAcademyMember() || !MutualWar(vc, kc) {
		return
	}
	kd, vd := social.WarKillReputation(s.clan.rules.ReputationPerKill, kc.Reputation, vc.Reputation)
	if kd != 0 {
		s.AddClanReputation(kc, kd)
	}
	if vd != 0 {
		s.AddClanReputation(vc, vd)
	}
}
