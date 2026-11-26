package game

import (
	"errors"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/social"
)

// The siege windows (Mobius RequestSiegeInfo, RequestSiegeAttackerList,
// RequestSiegeDefenderList, RequestJoinSiege, RequestConfirmSiegeWaitingList
// and SiegeInfo, SiegeAttackerList, SiegeDefenderList) and the registration
// rules of Siege.registerAttacker, registerDefender and removeSiegeClan.

// siegeHourList is Feature.ini SiegeHourList, the hours a lord may pick.
var siegeHourList = []int{16, 20}

// Messages of the registration.
const (
	siegeMsgNotAuthorized serverpackets.SystemMessageID = 794
	siegeMsgDissolving    serverpackets.SystemMessageID = 1114
)

// registerSiegePackets registers the siege packets.
func (s *Server) registerSiegePackets() error {
	return errors.Join(
		// RequestSiegeInfo has an empty runImpl in Mobius.
		s.Register(clientpackets.OpRequestSiegeInfo, func(*Player, []byte) {}),
		s.Register(clientpackets.OpRequestSiegeAttackerList, s.handleSiegeAttackerList),
		s.Register(clientpackets.OpRequestSiegeDefenderList, s.handleSiegeDefenderList),
		s.Register(clientpackets.OpRequestJoinSiege, s.handleJoinSiege),
		s.Register(clientpackets.OpRequestConfirmSiegeWaitingList, s.handleConfirmSiegeWaitingList),
	)
}

// siegeClanLine is a clan's line of the attacker and defender lists.
func (s *Server) siegeClanLine(id int32, typ int32) (serverpackets.SiegeClan, bool) {
	cl := s.Clan(id)
	if cl == nil {
		return serverpackets.SiegeClan{}, false
	}
	return serverpackets.SiegeClan{ClanID: cl.ID, ClanName: cl.Name, LeaderName: cl.LeaderName(), CrestID: cl.CrestID,
		Type: typ, AllyID: cl.AllyID, AllyName: cl.AllyName, AllyCrestID: cl.AllyCrestID}, true
}

// siegeInfoPacket is SiegeInfo for a castle seen by p.
func (s *Server) siegeInfoPacket(c *Castle, p *Player) []byte {
	info := serverpackets.SiegeInfo{ResidenceID: c.ID, Now: int32(s.Now().Unix()), SiegeDate: int32(c.siegeDate.Unix())} //nolint:gosec // Unix seconds fit until 2038
	if o := c.Owner(); o != nil {
		info.OwnerClanID, info.OwnerClanName, info.OwnerLeaderName = o.ID, o.Name, o.LeaderName()
		info.OwnerAllyID, info.OwnerAllyName = o.AllyID, o.AllyName
		info.OwnerLeader = p.ClanID() == o.ID && p.IsClanLeader()
		if info.OwnerLeader && !c.regTimeOver {
			info.ChooseTime = true
			info.TimeChoices = serverpackets.SiegeHourChoices(s.calendar.In(c.siegeDate), siegeHourList)
		}
	}
	return serverpackets.NewSiegeInfoPacket(info)
}

// ShowSiegeInfo sends p the siege window of a castle (Siege.listRegisterClan).
func (s *Server) ShowSiegeInfo(p *Player, c *Castle) { p.Send(s.siegeInfoPacket(c, p)) }

// handleSiegeAttackerList answers RequestSiegeAttackerList.
func (s *Server) handleSiegeAttackerList(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestSiegeAttackerList(body)
	if err != nil {
		return
	}
	if c := s.CastleByID(req.CastleID); c != nil {
		s.sendAttackerList(p, c)
	} else if h := s.hallSiegeOf(req.CastleID); h != nil {
		s.sendHallAttackerList(p, h) // hallsiege.go
	}
}

// sendAttackerList sends SiegeAttackerList.
func (s *Server) sendAttackerList(p *Player, c *Castle) {
	l := serverpackets.SiegeAttackerList{ResidenceID: c.ID}
	for _, sc := range c.siege.reg.Attackers() {
		if line, ok := s.siegeClanLine(sc.ClanID, 0); ok {
			l.Clans = append(l.Clans, line)
		}
	}
	p.Send(serverpackets.NewSiegeAttackerListPacket(l))
}

// handleSiegeDefenderList answers RequestSiegeDefenderList.
func (s *Server) handleSiegeDefenderList(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestSiegeDefenderList(body)
	if err != nil {
		return
	}
	if c := s.CastleByID(req.CastleID); c != nil {
		s.sendDefenderList(p, c)
	}
}

// sendDefenderList sends SiegeDefenderList: the owner and the defenders,
// then the clans waiting for approval.
func (s *Server) sendDefenderList(p *Player, c *Castle) {
	l := serverpackets.SiegeDefenderList{CastleID: c.ID}
	for _, sc := range c.siege.reg.Defenders() {
		typ := int32(serverpackets.SiegeDefenderTypeApproved)
		if sc.Type == residence.Owner {
			typ = serverpackets.SiegeDefenderTypeOwner
		}
		if line, ok := s.siegeClanLine(sc.ClanID, typ); ok {
			l.Clans = append(l.Clans, line)
		}
	}
	for _, sc := range c.siege.reg.Waiting() {
		if line, ok := s.siegeClanLine(sc.ClanID, serverpackets.SiegeDefenderTypeWaiting); ok {
			l.Clans = append(l.Clans, line)
		}
	}
	p.Send(serverpackets.NewSiegeDefenderListPacket(l))
}

// handleJoinSiege is RequestJoinSiege for a castle.
func (s *Server) handleJoinSiege(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestJoinSiege(body)
	if err != nil {
		return
	}
	if !p.HasClanAccess(social.AccessCastleSiege) {
		p.SendSystemMessage(siegeMsgNotAuthorized)
		return
	}
	cl := p.Clan()
	c := s.CastleByID(req.CastleID)
	if h := s.hallSiegeOf(req.CastleID); c == nil && h != nil && cl != nil {
		s.hallJoinSiege(p, cl, h, req.IsJoining == 1) // hallsiege.go
		return
	}
	if cl == nil || c == nil {
		return
	}
	if req.IsJoining == 1 {
		if s.Now().UnixMilli() < cl.DissolvingExpiry {
			p.SendSystemMessage(siegeMsgDissolving)
			return
		}
		if req.IsAttacker == 1 {
			s.RegisterAttacker(p, c, false)
		} else {
			s.RegisterDefender(p, c, false)
		}
	} else {
		if c.siege.inProgress || c.siege.registrationOver {
			p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgNotTimeToRegister))
			return
		}
		s.RemoveSiegeClan(c, cl)
	}
	s.ShowSiegeInfo(p, c)
}

// handleConfirmSiegeWaitingList is RequestConfirmSiegeWaitingList: the
// owner's leader approves or removes a defender before registration closes.
func (s *Server) handleConfirmSiegeWaitingList(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestConfirmSiegeWaitingList(body)
	if err != nil || p.Clan() == nil {
		return
	}
	c := s.CastleByID(req.CastleID)
	if c == nil || c.ownerID != p.ClanID() || !p.IsClanLeader() {
		return
	}
	cl := s.Clan(req.ClanID)
	if cl == nil {
		return
	}
	if !c.siege.registrationOver {
		if req.Approved == 1 {
			if c.siege.reg.IsWaiting(cl.ID) {
				c.siege.reg.Approve(cl.ID)
				c.save()
			} else {
				return
			}
		} else if c.siege.reg.IsWaiting(cl.ID) || c.siege.reg.IsDefender(cl.ID) {
			s.RemoveSiegeClan(c, cl)
		}
	}
	s.sendDefenderList(p, c)
}

// isRegisteredFor is SiegeManager.checkIsRegistered: the clan owns a castle
// or has a registration for this one.
func isRegisteredFor(c *Castle, cl *Clan) bool {
	return cl.CastleID > 0 || slicesContainsRow(c.siege.reg.Rows(), cl.ID)
}

func slicesContainsRow(rows []residence.Registration, id int32) bool {
	for _, r := range rows {
		if r.ClanID == id {
			return true
		}
	}
	return false
}

// sameDayRegistered is checkIfAlreadyRegisteredForSameDay.
func (s *Server) sameDayRegistered(c *Castle, id int32) bool {
	day := s.calendar.In(c.siegeDate).Weekday()
	for _, o := range s.residence.castles {
		if o != c && s.calendar.In(o.siegeDate).Weekday() == day && o.siege.reg.Registered(id) {
			return true
		}
	}
	return false
}

// registerCheck fills residence.RegisterCheck for p's clan.
func (s *Server) registerCheck(p *Player, c *Castle, attacker bool) residence.RegisterCheck {
	cfg := s.residence.cfg
	chk := residence.RegisterCheck{RegistrationOver: c.siege.registrationOver, InProgress: c.siege.inProgress,
		OwnerID: c.ownerID, Attacker: attacker, MinLevel: cfg.minLevel, MaxAttackers: cfg.maxAttackers, MaxDefenders: cfg.maxDefenders}
	if cl := p.Clan(); cl != nil {
		chk.ClanID, chk.ClanLevel, chk.ClanCastleID = cl.ID, cl.Level, cl.CastleID
		chk.Registered = isRegisteredFor(c, cl)
		chk.SameDay = s.sameDayRegistered(c, cl.ID)
	}
	return chk
}

// RegisterAttacker is Siege.registerAttacker: an ally of the owner may not
// attack; force (a GM) skips the other checks. Loop-only.
func (s *Server) RegisterAttacker(p *Player, c *Castle, force bool) {
	cl := p.Clan()
	if cl == nil {
		return
	}
	if o := c.Owner(); o != nil && o.AllyID != 0 && cl.AllyID == o.AllyID && !force {
		p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgAllyOfOwner))
		return
	}
	if force {
		if isRegisteredFor(c, cl) {
			p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgAlreadyRequested))
			return
		}
	} else if msg, ok := c.siege.reg.CheckRegister(s.registerCheck(p, c, true)); !ok {
		s.sendRegisterRefusal(p, c, msg)
		return
	}
	s.saveSiegeClan(c, cl, residence.RegAttacker)
}

// RegisterDefender is Siege.registerDefender: only for a castle with an
// owner; the clan waits for the owner's approval. Loop-only.
func (s *Server) RegisterDefender(p *Player, c *Castle, force bool) {
	cl := p.Clan()
	if c.ownerID <= 0 {
		p.SendMessage("You cannot register as a defender because " + c.Name + " is owned by NPC.")
		return
	}
	if cl == nil {
		return
	}
	if force {
		if isRegisteredFor(c, cl) {
			p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgAlreadyRequested))
			return
		}
	} else if msg, ok := c.siege.reg.CheckRegister(s.registerCheck(p, c, false)); !ok {
		s.sendRegisterRefusal(p, c, msg)
		return
	}
	s.saveSiegeClan(c, cl, residence.RegDefenderPending)
}

// sendRegisterRefusal sends a refusal, with the castle's name where the
// message has one.
func (s *Server) sendRegisterRefusal(p *Player, c *Castle, msg int) {
	if msg == residence.MsgRegistrationDeadline {
		p.SendSystemMessage(serverpackets.SystemMessageID(msg), castleParam(c))
		return
	}
	p.SendSystemMessage(serverpackets.SystemMessageID(msg)) //nolint:gosec // message IDs
}

// saveSiegeClan is Siege.saveSiegeClan for a new registration.
func (s *Server) saveSiegeClan(c *Castle, cl *Clan, typ int) {
	if cl.CastleID > 0 {
		return
	}
	cfg := s.residence.cfg
	if !c.siege.reg.CanSave(typ == residence.RegAttacker, cfg.maxAttackers, cfg.maxDefenders) {
		return
	}
	switch typ {
	case residence.RegAttacker:
		c.siege.reg.AddAttacker(cl.ID)
	case residence.RegDefender:
		c.siege.reg.AddDefender(cl.ID, residence.Defender)
	case residence.RegDefenderPending:
		c.siege.reg.AddWaiting(cl.ID)
	}
	c.save()
}

// RemoveSiegeClan is Siege.removeSiegeClan(Clan): not the owner, only a
// registered clan. Loop-only.
func (s *Server) RemoveSiegeClan(c *Castle, cl *Clan) {
	if cl == nil || cl.CastleID == c.ID || !isRegisteredFor(c, cl) {
		return
	}
	c.siege.reg.Remove(cl.ID)
	c.save()
}
