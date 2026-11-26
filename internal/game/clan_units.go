package game

import (
	"strconv"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
)

// The clan units (Plan 12 Phase 7 item 2): the village master commands
// that create the academy, Royal Guard and Knight units, rename them and
// appoint their captains (Mobius VillageMaster createSubPledge,
// renameSubPledge, assignSubPledgeLeader and Clan.createSubPledge).

// System messages of the units.
const (
	unitMsgTitleTooLong         serverpackets.SystemMessageID = 80   // "Your title cannot exceed 16 characters in length. Please try again."
	unitMsgAcademyNeedsLevel5   serverpackets.SystemMessageID = 1730 // "To establish a Clan Academy, your clan must be Level 5 or higher."
	unitMsgEnterAcademyName     serverpackets.SystemMessageID = 1740 // "Please enter the name of the Clan Academy."
	unitMsgAcademyCreated       serverpackets.SystemMessageID = 1741 // "Congratulations! The $s1's Clan Academy has been created."
	unitMsgConditionsNotMet     serverpackets.SystemMessageID = 1791 // "The conditions necessary to create a military unit have not been met."
	unitMsgCaptainSelected      serverpackets.SystemMessageID = 1793 // "$s1 has been selected as the captain of $s2."
	unitMsgKnightsCreated       serverpackets.SystemMessageID = 1794 // "The Knights of $s1 have been created."
	unitMsgRoyalGuardCreated    serverpackets.SystemMessageID = 1795 // "The Royal Guard of $s1 have been created."
	unitMsgKnightCaptainRefused serverpackets.SystemMessageID = 1850 // "The Captain of the Order of Knights cannot be appointed."
	unitMsgRoyalCaptainRefused  serverpackets.SystemMessageID = 1851 // "The Captain of the Royal Guard cannot be appointed."
	unitMsgNameTaken            serverpackets.SystemMessageID = 1855 // "Another military unit is already using that name. Please enter a different name."
	unitMsgReputationTooLow     serverpackets.SystemMessageID = 1860 // "The Clan Reputation Score is too low."
)

func init() {
	registerVillageMasterCommand("create_academy", func(s *Server, p *Player, _ *Npc, args []string) {
		if len(args) == 0 || args[0] == "" {
			p.SendSystemMessage(unitMsgEnterAcademyName)
			return
		}
		s.CreateClanUnit(p, args[0], "", social.UnitAcademy)
	})
	registerVillageMasterCommand("create_royal", func(s *Server, p *Player, _ *Npc, args []string) {
		if len(args) == 0 || args[0] == "" {
			return
		}
		s.CreateClanUnit(p, args[0], vmArg(args, 1), social.UnitRoyal1)
	})
	registerVillageMasterCommand("create_knight", func(s *Server, p *Player, _ *Npc, args []string) {
		if len(args) == 0 || args[0] == "" {
			return
		}
		s.CreateClanUnit(p, args[0], vmArg(args, 1), social.UnitKnight1)
	})
	registerVillageMasterCommand("assign_subpl_leader", func(s *Server, p *Player, _ *Npc, args []string) {
		if len(args) == 0 || args[0] == "" {
			return
		}
		s.AssignClanUnitLeader(p, args[0], vmArg(args, 1))
	})
	registerVillageMasterCommand("rename_pledge", func(s *Server, p *Player, _ *Npc, args []string) {
		if vmArg(args, 0) == "" || vmArg(args, 1) == "" {
			return
		}
		unit, err := strconv.Atoi(args[0])
		if err != nil {
			return // Mobius fails on Integer.parseInt
		}
		s.RenameClanUnit(p, unit, args[1])
	})
}

// vmArg is the i-th word after a village master command, empty when there
// is none (Mobius cmdParams, cmdParams2).
func vmArg(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

// unitCaptainRefused tells p that unit's captain cannot be appointed (the
// Knights or the Royal Guard message; nothing for the academy).
func unitCaptainRefused(p *Player, unit int) {
	switch {
	case unit >= social.UnitKnight1:
		p.SendSystemMessage(unitMsgKnightCaptainRefused)
	case unit >= social.UnitRoyal1:
		p.SendSystemMessage(unitMsgRoyalCaptainRefused)
	}
}

// unitCaptainCandidate returns the member of c named name if it may become
// a captain: a member of the main clan who leads no unit yet (Mobius
// VillageMaster), or nil.
func unitCaptainCandidate(c *Clan, name string) *ClanMember {
	m := c.MemberByName(name)
	if m == nil || m.PledgeType() != social.UnitMain || c.UnitLedBy(m.ObjectID()) > 0 {
		return nil
	}
	return m
}

// unitByName returns c's unit named name, ignoring case (Mobius
// getSubPledge(String)), or nil.
func unitByName(c *Clan, name string) *storage.ClanUnit {
	for i := range c.Units {
		if social.NameKey(c.Units[i].Name) == social.NameKey(name) {
			return &c.Units[i]
		}
	}
	return nil
}

// checkUnitName is the name check of the unit commands: the template,
// letters and digits, at least 2 (CLAN_NAME_IS_INVALID), at most 16
// (CLAN_NAME_S_LENGTH_IS_INCORRECT).
func (s *Server) checkUnitName(p *Player, name string) bool {
	switch social.CheckClanName(name, s.clan.nameTemplate) {
	case social.NameInvalid:
		p.SendSystemMessage(clanMsgNameInvalid)
		return false
	case social.NameBadLength:
		p.SendSystemMessage(clanMsgNameLength)
		return false
	}
	return true
}

// CreateClanUnit creates a unit of p's clan (Mobius
// VillageMaster.createSubPledge and Clan.createSubPledge): kind is
// social.UnitAcademy, UnitRoyal1 or UnitKnight1, and the unit gets the next
// free ID of that kind (two Royal Guards, four Knights, one academy). Only
// the leader, with clan level 5 (academy), 6 (Royal Guard) or 7 (Knights);
// a name no unit of any clan has; a captain (captainName) from the main
// clan who leads no unit and is not the clan leader; the reputation the
// unit costs (none, RoyalGuardCost, KnightUnitCost). The members online
// get PledgeShowInfoUpdate and PledgeReceiveSubPledgeCreated, p the created
// message, the captain its new pledge class. The clan is saved. It reports
// whether the unit was created. Loop-only.
func (s *Server) CreateClanUnit(p *Player, name, captainName string, kind int) bool {
	c := p.clan.clan
	if c == nil || !p.IsClanLeader() {
		p.SendSystemMessage(clanMsgNotAuthorized)
		return false
	}
	if c.Level < social.UnitMinClanLevel(kind) {
		if kind == social.UnitAcademy {
			p.SendSystemMessage(unitMsgAcademyNeedsLevel5)
		} else {
			p.SendSystemMessage(unitMsgConditionsNotMet)
		}
		return false
	}
	if !s.checkUnitName(p, name) {
		return false
	}
	for _, o := range sortedClans(s.clan.clans) {
		if unitByName(o, name) != nil {
			if kind == social.UnitAcademy {
				p.SendSystemMessage(clanMsgS1AlreadyExists, serverpackets.TextParam(name))
			} else {
				p.SendSystemMessage(unitMsgNameTaken)
			}
			return false
		}
	}
	var captain *ClanMember
	if kind != social.UnitAcademy {
		if captain = unitCaptainCandidate(c, captainName); captain == nil {
			unitCaptainRefused(p, kind)
			return false
		}
	}
	// Clan.createSubPledge.
	unit := social.AvailableUnit(kind, func(u int) bool { return c.Unit(u) != nil })
	if unit == 0 {
		// Mobius's academy message here is unreachable (it tests the unit
		// after it became 0).
		p.SendMessage("You can't create any more sub-units of this type")
		return false
	}
	var captainID int32
	if captain != nil {
		captainID = captain.ObjectID()
	}
	if captainID != 0 && captainID == c.LeaderID {
		p.SendMessage("Leader is not correct")
		return false
	}
	cost := s.clan.rules.UnitCost(unit)
	if c.Reputation < cost {
		p.SendSystemMessage(unitMsgReputationTooLow)
		return false
	}
	u := storage.ClanUnit{Type: int32(unit), Name: name, LeaderID: captainID} //nolint:gosec // unit IDs are small
	c.Units = append(c.Units, u)
	if cost > 0 {
		s.setClanReputation(c, -cost)
	}
	s.saveClan(c)
	broadcastToClan(c, serverpackets.NewPledgeShowInfoUpdatePacket(clanInfo(c)))
	broadcastToClan(c, serverpackets.NewPledgeReceiveSubPledgeCreatedPacket(unitPacket(c, u)))

	msg := unitMsgAcademyCreated
	switch {
	case unit >= social.UnitKnight1:
		msg = unitMsgKnightsCreated
	case unit >= social.UnitRoyal1:
		msg = unitMsgRoyalGuardCreated
	}
	p.SendSystemMessage(msg, serverpackets.TextParam(c.Name))
	if captain != nil {
		if q := captain.Player(); q != nil {
			s.refreshPledgeClass(q)
			q.UpdateUserInfo()
		}
	}
	s.log.Info("clan unit created", append(p.logArgs(), "clan", c.Name, "unit", unit, "name", name, "captain", captainName)...)
	return true
}

// unitPacket is a unit as PledgeReceiveSubPledgeCreated shows it: the
// academy without a captain.
func unitPacket(c *Clan, u storage.ClanUnit) serverpackets.SubPledge {
	leader := ""
	if u.Type != storage.UnitAcademy && u.LeaderID != 0 {
		if m := c.member(u.LeaderID); m != nil {
			leader = m.Name()
		}
	}
	return serverpackets.SubPledge{ID: u.Type, Name: u.Name, LeaderName: leader}
}

// RenameClanUnit renames a unit of p's clan (Mobius
// VillageMaster.renameSubPledge): only the leader, a unit the clan has, a
// valid name (Mobius does not look for the name in other units). The clan
// is saved and its members online get the new clan window. Loop-only.
func (s *Server) RenameClanUnit(p *Player, unit int, name string) bool {
	c := p.clan.clan
	if c == nil || !p.IsClanLeader() {
		p.SendSystemMessage(clanMsgNotAuthorized)
		return false
	}
	u := c.Unit(unit)
	if u == nil {
		p.SendMessage("Pledge don't exists.")
		return false
	}
	if !s.checkUnitName(p, name) {
		return false
	}
	u.Name = name
	s.saveClan(c)
	s.broadcastClanStatus(c)
	p.SendMessage("Pledge name changed.")
	return true
}

// AssignClanUnitLeader makes the member named captainName the captain of
// p's unit named unitName (Mobius VillageMaster.assignSubPledgeLeader): only
// the leader; a name of at most 16 characters that is not the leader's own;
// a Royal Guard or Knight unit of the clan; a main clan member who leads no
// unit. The clan is saved; the new captain gets its pledge class, the
// members online the new clan window and "$s1 has been selected as the
// captain of $s2.". Loop-only.
//
// Deviation: the captain it replaces gets its pledge class back at once;
// Mobius leaves the old one until it logs in again.
func (s *Server) AssignClanUnitLeader(p *Player, unitName, captainName string) bool {
	c := p.clan.clan
	if c == nil || !p.IsClanLeader() {
		p.SendSystemMessage(clanMsgNotAuthorized)
		return false
	}
	if len([]rune(captainName)) > 16 {
		p.SendSystemMessage(unitMsgTitleTooLong)
		return false
	}
	if p.Name() == captainName {
		p.SendSystemMessage(unitMsgRoyalCaptainRefused)
		return false
	}
	u := unitByName(c, unitName)
	if u == nil || u.Type == storage.UnitAcademy {
		p.SendSystemMessage(clanMsgNameInvalid)
		return false
	}
	m := unitCaptainCandidate(c, captainName)
	if m == nil {
		unitCaptainRefused(p, int(u.Type))
		return false
	}
	old := c.member(u.LeaderID)
	u.LeaderID = m.ObjectID()
	s.saveClan(c)
	for _, x := range []*ClanMember{old, m} {
		if x == nil {
			continue
		}
		if q := x.Player(); q != nil {
			s.refreshPledgeClass(q)
			q.UpdateUserInfo()
		}
	}
	s.broadcastClanStatus(c)
	broadcastToClan(c, clanSysMsg(unitMsgCaptainSelected, serverpackets.TextParam(captainName), serverpackets.TextParam(unitName)))
	s.log.Info("clan unit captain appointed", append(p.logArgs(), "clan", c.Name, "unit", u.Type, "captain", captainName)...)
	return true
}
