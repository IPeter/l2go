package game

import (
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/skills"
)

// The siege skills (skills.ResidenceWorld): Seal of Ruler (246, TakeCastle
// on an artefact, condition canTakeCastle) and Build Headquarters (247,
// HeadquarterCreate, condition canCreateBase).

var _ skills.ResidenceWorld = skillWorld{}

// Messages of the siege skills.
const (
	siegeMsgInvalidTarget  serverpackets.SystemMessageID = 109
	siegeMsgUnsuitable     serverpackets.SystemMessageID = 113
	siegeMsgEngraveStarted serverpackets.SystemMessageID = 287
	siegeMsgNoBaseHere     serverpackets.SystemMessageID = 290
	siegeMsgTooFar         serverpackets.SystemMessageID = 748
)

// artefactCastle returns the castle of an artefact NPC (Artefact.onSpawn
// registers it with the castle it stands in), or nil.
func (s *Server) artefactCastle(n *Npc) *Castle {
	if n == nil || !s.npcIs(n, "Artefact") {
		return nil
	}
	return s.CastleAt(n.Position())
}

// IsArtefact implements skills.ResidenceWorld.
func (w skillWorld) IsArtefact(c skills.Creature) bool {
	n, ok := w.s.gameCreature(c).(*Npc)
	return ok && w.s.npcIs(n, "Artefact")
}

// CanTakeCastle implements skills.ResidenceWorld
// (ConditionPlayerCanTakeCastle).
func (w skillWorld) CanTakeCastle(c, target skills.Creature, sk *skills.Skill) bool {
	s := w.s
	p, ok := s.gameCreature(c).(*Player)
	if !ok || p.IsDead() || p.CursedWeaponEquipped() || !p.IsClanLeader() {
		return false
	}
	castle := s.CastleAt(p.Position())
	if castle == nil || !castle.siege.inProgress || castle.siege.reg.Attacker(p.ClanID()) == nil {
		p.SendSystemMessage(siegeMsgUnsuitable, gmSkillNameParam(sk))
		return false
	}
	n, _ := s.gameCreature(target).(*Npc)
	if s.artefactCastle(n) != castle {
		p.SendSystemMessage(siegeMsgInvalidTarget)
		return false
	}
	pp, np := p.Position(), n.Position()
	if pp.Distance3D(np) > 200+float64(creatureCollision(p)+creatureCollision(n)) || pp.Z < np.Z || abs32(pp.Z-np.Z) > 45 {
		p.SendSystemMessage(siegeMsgTooFar)
		return false
	}
	s.announceSiege(castle, clanSysMsg(siegeMsgEngraveStarted), false)
	return true
}

// CanCreateBase implements skills.ResidenceWorld
// (ConditionPlayerCanCreateBase).
func (w skillWorld) CanCreateBase(c skills.Creature, sk *skills.Skill) bool {
	s := w.s
	p, ok := s.gameCreature(c).(*Player)
	if !ok {
		return false
	}
	can := !p.IsDead() && !p.CursedWeaponEquipped() && p.Clan() != nil
	castle := s.CastleAt(p.Position())
	var sc *residence.SiegeClan
	if castle != nil {
		sc = castle.siege.reg.Attacker(p.ClanID())
	}
	switch {
	case castle == nil, !castle.siege.inProgress, sc == nil, !p.IsClanLeader(), len(sc.Flags) >= s.residence.cfg.maxFlags:
		p.SendSystemMessage(siegeMsgUnsuitable, gmSkillNameParam(sk))
		can = false
	case !p.Zones.Inside(ZoneHQ):
		p.SendSystemMessage(siegeMsgNoBaseHere)
		can = false
	}
	return can
}

// TakeCastle implements skills.ResidenceWorld (TakeCastle.onStart and
// Castle.engrave).
func (w skillWorld) TakeCastle(effector, effected skills.Creature) {
	s := w.s
	p, ok := s.gameCreature(effector).(*Player)
	if !ok || p.Clan() == nil {
		return
	}
	n, _ := s.gameCreature(effected).(*Npc)
	castle := s.artefactCastle(n)
	if castle == nil || !castle.siege.inProgress {
		return
	}
	s.EngraveCastle(castle, p.Clan())
}

// EngraveCastle is Castle.engrave: the clan becomes the owner and everyone
// hears it. Loop-only.
func (s *Server) EngraveCastle(c *Castle, clan *Clan) {
	s.setCastleOwner(c, clan)
	s.broadcastSysMsg(residence.MsgEngravedRuler, serverpackets.TextParam(clan.Name))
}

// CreateHeadquarters implements skills.ResidenceWorld.
func (w skillWorld) CreateHeadquarters(effector skills.Creature, _ bool) {
	if p, ok := w.s.gameCreature(effector).(*Player); ok {
		w.s.buildHeadquarters(p)
	}
}
