package game

import (
	"strings"
	"unicode/utf16"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
)

// Ranks, privileges and titles: RequestPledgePower (0xC0), the grade list
// and member grades (D0:1A-D0:1C), member info (D0:1D), unit moves (D0:24)
// and titles (RequestGiveNickName, 0x55).

// clanTitleLevel is the clan level titles need (RequestGiveNickName).
const clanTitleLevel = 3

// handlePledgePower handles RequestPledgePower (0xC0): action 2 lets the
// leader set a rank's privileges (rank 9 keeps only the warehouse and door
// bits); action 1 shows them (ManagePledgePower).
func (s *Server) handlePledgePower(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestPledgePower(body)
	if err != nil {
		s.log.Debug("game: bad RequestPledgePower", append(p.logArgs(), "err", err)...)
		return
	}
	c := p.clan.clan
	if req.Action == clientpackets.PledgePowerSet {
		if c != nil && p.IsClanLeader() {
			s.SetRankPrivileges(c, int(req.Rank), social.RankPrivileges(int(req.Rank), social.Privileges(req.Privileges)))
		}
		return
	}
	// Mobius writes ManagePledgePower only for action 1 (and fails without a
	// clan).
	if c != nil && req.Action == clientpackets.PledgePowerShow {
		p.Send(serverpackets.NewManagePledgePowerPacket(int32(c.Ranks.Of(int(req.Rank)))))
	}
}

// SetRankPrivileges sets the privileges of a rank (Mobius
// Clan.setRankPrivs): the members online of that rank get them with their
// UserInfo, and every member online the clan window. The clan is saved.
// Ranks outside 1-9 are ignored (Mobius stores them but never loads them).
// Loop-only.
func (s *Server) SetRankPrivileges(c *Clan, rank int, privs social.Privileges) {
	if rank < 1 || rank > social.Ranks {
		return
	}
	c.Ranks.Set(rank, privs)
	s.saveClan(c)
	for _, m := range c.members {
		if m.Online() && m.PowerGrade() == rank && m.ObjectID() != c.LeaderID {
			m.player.clan.privs = privs
			m.player.UpdateUserInfo()
		}
	}
	s.broadcastClanStatus(c)
}

// handlePledgePowerGradeList handles RequestPledgePowerGradeList (D0:1A):
// the nine ranks (PledgePowerGradeList; Mobius writes their unused party
// value, 0).
func (s *Server) handlePledgePowerGradeList(p *Player, _ []byte) {
	if p.clan.clan == nil {
		return
	}
	grades := make([]serverpackets.PowerGrade, social.Ranks)
	for i := range grades {
		grades[i] = serverpackets.PowerGrade{Rank: int32(i + 1)} //nolint:gosec // 1-9
	}
	p.Send(serverpackets.NewPledgePowerGradeListPacket(grades))
}

// powerInfoPacket is PledgeReceivePowerInfo of a member: its rank and that
// rank's privileges.
func powerInfoPacket(m *ClanMember) []byte {
	grade := m.PowerGrade()
	return serverpackets.NewPledgeReceivePowerInfoPacket(int32(grade), m.Name(), int32(m.clan.Ranks.Of(grade))) //nolint:gosec // 0-9
}

// handlePledgeMemberPowerInfo handles RequestPledgeMemberPowerInfo
// (D0:1B).
func (s *Server) handlePledgeMemberPowerInfo(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestPledgeMemberPowerInfo(body)
	if err != nil {
		return
	}
	if c := p.clan.clan; c != nil {
		if m := c.MemberByName(req.Name); m != nil {
			p.Send(powerInfoPacket(m))
		}
	}
}

// handlePledgeSetMemberPowerGrade handles RequestPledgeSetMemberPowerGrade
// (D0:1C): a member with the rank privilege gives another member, not the
// leader and not in the academy, a rank; an online member gets that rank's
// privileges.
func (s *Server) handlePledgeSetMemberPowerGrade(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestPledgeSetMemberPowerGrade(body)
	if err != nil {
		return
	}
	c := p.clan.clan
	if c == nil || !p.HasClanAccess(social.AccessModifyRanks) {
		return
	}
	m := c.MemberByName(req.Name)
	if m == nil || m.ObjectID() == c.LeaderID {
		return
	}
	if m.PledgeType() == social.UnitAcademy {
		p.SendMessage("You cannot change academy member grade")
		return
	}
	grade := req.PowerGrade
	if q := m.Player(); q != nil {
		q.char.PowerGrade = grade
		q.clan.privs = c.Ranks.Of(int(grade))
		q.UpdateUserInfo()
		s.RequestSave(q)
	} else {
		s.SaveJoint(nil, storage.Batch{Patches: []storage.CharacterPatch{
			m.patch(storage.PatchPowerGrade, storage.PatchValues{PowerGrade: grade}),
		}})
	}
	s.broadcastClanStatus(c)
	p.Send(powerInfoPacket(m))
}

// handlePledgeMemberInfo handles RequestPledgeMemberInfo (D0:1D): a
// member's unit, title, rank, unit name and apprentice or sponsor.
func (s *Server) handlePledgeMemberInfo(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestPledgeMemberInfo(body)
	if err != nil {
		return
	}
	c := p.clan.clan
	if c == nil {
		return
	}
	m := c.MemberByName(req.Name)
	if m == nil {
		return
	}
	unitName := c.Name
	if unit := m.PledgeType(); unit != social.UnitMain {
		u := c.Unit(unit)
		if u == nil {
			return // Mobius throws on a unit that does not exist
		}
		unitName = u.Name
	}
	p.Send(serverpackets.NewPledgeReceiveMemberInfoPacket(serverpackets.PledgeReceiveMemberInfo{
		PledgeType:  int32(m.PledgeType()), //nolint:gosec // small
		Name:        m.Name(),
		Title:       m.Title(),
		PowerGrade:  int32(m.PowerGrade()), //nolint:gosec // 0-9
		UnitName:    unitName,
		SponsorName: m.ApprenticeOrSponsorName(),
	}))
}

// handlePledgeReorganizeMember handles RequestPledgeReorganizeMember
// (D0:24): a member with the rank privilege swaps the units of two members
// (neither the leader). The units themselves come with Phase 7b.
func (s *Server) handlePledgeReorganizeMember(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestPledgeReorganizeMember(body)
	if err != nil || req.MemberSelected == 0 {
		return
	}
	c := p.clan.clan
	if c == nil || !p.HasClanAccess(social.AccessModifyRanks) {
		return
	}
	m1 := c.MemberByName(req.MemberName)
	if m1 == nil || m1.ObjectID() == c.LeaderID {
		return
	}
	m2 := c.MemberByName(req.SelectedMember)
	if m2 == nil || m2.ObjectID() == c.LeaderID {
		return
	}
	old := m1.PledgeType()
	if old == int(req.NewPledgeType) {
		return
	}
	var b storage.Batch
	var players []*Player
	set := func(m *ClanMember, unit int) {
		if q := m.Player(); q != nil {
			q.char.PledgeType = int32(unit) //nolint:gosec // small
			players = append(players, q)
			return
		}
		b.Patches = append(b.Patches, m.patch(storage.PatchPledgeType, storage.PatchValues{PledgeType: int32(unit)})) //nolint:gosec // small
	}
	set(m1, int(req.NewPledgeType))
	set(m2, old)
	s.SaveJoint(players, b)
	s.broadcastClanStatus(c)
}

// ----- titles -----

// handleGiveNickName handles RequestGiveNickName (0x55): a noble titles
// itself; otherwise a member with the title privilege, in a clan of level 3
// or more, titles a member online. Titles over 21 characters are cut to 20
// (Creature.setTitle).
func (s *Server) handleGiveNickName(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestGiveNickName(body)
	if err != nil {
		s.log.Debug("game: bad RequestGiveNickName", append(p.logArgs(), "err", err)...)
		return
	}
	title := clanCutTitle(req.Title)
	if p.IsNoble() && strings.EqualFold(req.Target, p.Name()) {
		s.setTitle(p, title)
		return
	}
	if !p.HasClanAccess(social.AccessAssignTitle) {
		p.SendSystemMessage(clanMsgNotAuthorized)
		return
	}
	c := p.clan.clan
	if c.Level < clanTitleLevel {
		p.SendSystemMessage(clanMsgTitleNeedsLevel3)
		return
	}
	m := c.MemberByName(req.Target)
	switch {
	case m == nil:
		p.SendSystemMessage(clanMsgTargetMustBeMember)
	case m.Player() == nil:
		p.SendSystemMessage(clanMsgPlayerNotOnline)
	default:
		s.setTitle(m.Player(), title)
	}
}

// setTitle gives p a title with "Your title has been changed." and the
// title broadcast (broadcastTitleInfo); p is saved.
func (s *Server) setTitle(p *Player, title string) {
	p.char.Title = title
	p.SendSystemMessage(clanMsgTitleChanged)
	s.broadcastTitleInfo(p)
	s.RequestSave(p)
}

// clanCutTitle is Creature.setTitle's limit for players: over 21 UTF-16
// units, the first 20.
func clanCutTitle(title string) string {
	if units := utf16.Encode([]rune(title)); len(units) > adminMaxTitleLength {
		return string(utf16.Decode(units[:adminCutTitleLength]))
	}
	return title
}
