package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
)

// Academy graduation (Plan 12 Phase 7 item 7, Mobius Player.setPlayerClass
// lines 2346-2373): an academy member who changes into a class of the
// third class group (the second class transfer) earns its clan reputation
// and leaves the clan without a penalty, with the Academy Circlet.

const (
	// academyMsgGraduated is "Congratulations! You will now graduate from
	// the Clan Academy and leave your current clan. ...".
	academyMsgGraduated serverpackets.SystemMessageID = 1749
	// academyCirclet is the graduation gift, the Academy Circlet.
	academyCirclet = 8181
)

// academyGraduation graduates p from its clan's academy if it joined the
// academy and classID is in the third class group (Mobius CategoryData
// THIRD_CLASS_GROUP, the classes of the second transfer). ChangeClass calls
// it before the class changes. The clan gains AcademyMaxPoints if p joined
// at level 16 or below, AcademyMinPoints from level 39, AcademyMaxPoints −
// (join level − 16) × 20 between; the members online get "Clan member $s1
// has been expelled." and PledgeShowMemberListDelete; p leaves the clan
// without a join penalty, is told it graduated and gets the circlet. The
// clan, p and a released offline sponsor are written jointly. Loop-only.
func (s *Server) academyGraduation(p *Player, classID int) {
	c := p.clan.clan
	if p.char.LvlJoinedAcademy == 0 || c == nil || !s.thirdClassGroup(classID) {
		return
	}
	points := s.clan.rules.AcademyGraduationPoints(int(p.char.LvlJoinedAcademy))
	s.setClanReputation(c, points)
	p.char.LvlJoinedAcademy = 0
	name := p.Name()
	broadcastToClan(c, clanSysMsg(clanMsgMemberExpelled, serverpackets.TextParam(name)))
	broadcastToClan(c, serverpackets.NewPledgeShowMemberListDeletePacket(name))
	patches := s.removeClanMember(c, p.ObjectID(), 0)
	p.SendSystemMessage(academyMsgGraduated)
	// Mobius adds the circlet without a message or an InventoryUpdate; the
	// update is sent here so that the client shows it at once.
	if _, err := s.AddItem(p, academyCirclet, 1, -1, false, false); err != nil {
		s.log.Warn("game: no academy circlet for a graduate", append(p.logArgs(), "err", err)...)
	}
	s.SaveJoint([]*Player{p}, storage.Batch{Clans: []storage.Clan{c.document()}, Patches: patches})
	s.log.Info("academy graduation", append(p.logArgs(), "clan", c.Name, "reputation", points)...)
}

// thirdClassGroup reports whether the class is in Mobius's
// THIRD_CLASS_GROUP category; without category data, the classes of the
// second transfer (class level 2), which is what the category holds.
func (s *Server) thirdClassGroup(classID int) bool {
	if cats := s.items.categories; cats != nil {
		return cats.InCategory(data.CategoryThirdClassGroup, classID)
	}
	return data.PlayerClass(int32(classID)).Level() == 2 //nolint:gosec // class IDs are small
}
