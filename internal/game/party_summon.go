package game

import (
	"fmt"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/world"
)

// Summoning players (Plan 12 Phase 3.7): the CallPc and CallParty effects
// (Mobius CallPc, CallParty, checkSummonTargetStatus), the ConfirmDlg
// question and the DlgAnswer (0xC5) dispatcher that other questions
// (resurrection, doors) can join.

// System messages of summoning. The Olympiad, observer, instance and Seven
// Signs refusals come with their features.
const (
	summonMsgS1Disappeared serverpackets.SystemMessageID = 302  // "$s1 has disappeared."
	summonMsgWishes        serverpackets.SystemMessageID = 1842 // "$s1 wishes to summon you from $s2. Do you accept?"
	summonMsgInCombat      serverpackets.SystemMessageID = 1843 // "$s1 is engaged in combat and cannot be summoned."
	summonMsgDead          serverpackets.SystemMessageID = 1844 // "$s1 is dead at the moment and cannot be summoned."
	summonMsgAreaBlocks    serverpackets.SystemMessageID = 1895 // "$s1 is in an area which blocks summoning."
	summonMsgItemRequired  serverpackets.SystemMessageID = 1897 // "$s1 is required for summoning."
	summonMsgTrading       serverpackets.SystemMessageID = 1898 // "$s1 is currently trading or operating a private store and cannot be summoned."
)

// summonConfirmTime is how long the summon question stays (CallPc: 30 s).
const summonConfirmTimeMs int32 = 30000

// DlgAnswerHandler handles a DlgAnswer to the ConfirmDlg with its message:
// answer 1 is yes, requesterID the ConfirmDlg's requester.
type DlgAnswerHandler func(p *Player, answer, requesterID int32)

// RegisterDlgAnswer sets the handler of the DlgAnswer to the ConfirmDlg
// with the message id. Call it from a feature's setup.
func (s *Server) RegisterDlgAnswer(id serverpackets.SystemMessageID, h DlgAnswerHandler) error {
	if h == nil {
		return fmt.Errorf("game: nil DlgAnswer handler for message %d", id)
	}
	if _, ok := s.party.dlg[id]; ok {
		return fmt.Errorf("%w: DlgAnswer handler of message %d", ErrDuplicate, id)
	}
	s.party.dlg[id] = h
	return nil
}

// setupSummons registers DlgAnswer and the summon answer.
func (s *Server) setupSummons() error {
	if err := s.Register(clientpackets.OpDlgAnswer, s.handleDlgAnswer); err != nil {
		return err
	}
	return s.RegisterDlgAnswer(summonMsgWishes, s.summonAnswer)
}

// handleDlgAnswer handles DlgAnswer (0xC5): the handler of the question's
// message runs; other answers are dropped.
func (s *Server) handleDlgAnswer(p *Player, body []byte) {
	req, err := clientpackets.ParseDlgAnswer(body)
	if err != nil {
		s.log.Debug("game: bad DlgAnswer", append(p.logArgs(), "err", err)...)
		return
	}
	if h := s.party.dlg[serverpackets.SystemMessageID(req.MessageID)]; h != nil {
		h(p, req.Answer, req.RequesterID)
	}
}

// summonRequest is Mobius SummonRequestHolder: who asked and where it stood
// when it asked.
type summonRequest struct {
	summonerID int32
	at         world.Position
}

// summonAnswer is DlgAnswer for "$s1 wishes to summon you from $s2": the
// request ends, and a yes to the same summoner teleports p to where the
// summoner stood, with the random offset.
func (s *Server) summonAnswer(p *Player, answer, requesterID int32) {
	req := p.party.summon
	p.party.summon = nil
	if answer != 1 || req == nil || req.summonerID != requesterID {
		return
	}
	s.teleportAround(p, summonLocation(req.at), maxOffsetOnTeleport)
}

// checkSummonTarget is CallPc.checkSummonTargetStatus: whether summoner may
// summon target, telling summoner why not. The Olympiad, festivals,
// events, observers, instances and the Seven Signs dungeons do not exist
// yet.
func (s *Server) checkSummonTarget(target, summoner *Player) bool {
	name := serverpackets.TextParam(target.Name())
	switch {
	case target == summoner:
		return false
	case target.IsDead() || target.skill.IsAlikeDead():
		summoner.SendSystemMessage(summonMsgDead, name)
		return false
	case s.inStoreMode(target):
		summoner.SendSystemMessage(summonMsgTrading, name)
		return false
	case target.skill.el.IsAffected(skills.FlagRooted) || s.InCombat(target):
		summoner.SendSystemMessage(summonMsgInCombat, name)
		return false
	case target.Zones.Inside(ZoneNoSummonFriend) || target.Zones.Inside(ZoneJail):
		summoner.SendSystemMessage(summonMsgAreaBlocks, name)
		return false
	}
	return true
}

// CallPc is the CallPc effect (skills.Summoner): a player effector asks a
// player effected to come, after checkSummonTarget, taking the item cost
// from effected first ("$s1 is required for summoning." without it); an
// NPC effector teleports effected to itself.
func (w skillWorld) CallPc(effector, effected skills.Creature, itemID, itemCount int) {
	s := w.s
	target, ok := s.gameCreature(effected).(*Player)
	if !ok {
		return
	}
	summoner, isPlayer := s.gameCreature(effector).(*Player)
	if !isPlayer {
		if c := s.gameCreature(effector); c != nil {
			prev := target.Target
			pos := c.Position()
			s.teleportAround(target, summonLocation(pos), 0)
			target.Target = prev
		}
		return
	}
	if !s.checkSummonTarget(target, summoner) {
		return
	}
	if itemID != 0 && itemCount != 0 {
		id := int32(itemID) //nolint:gosec // item IDs are small
		if target.inv.Count(id, -1, true) < int64(itemCount) {
			target.SendSystemMessage(summonMsgItemRequired, s.itemNameParam(id))
			return
		}
		s.DestroyItemByItemID(target, id, int64(itemCount), false)
		target.SendSystemMessage(summonMsgS1Disappeared, s.itemNameParam(id))
	}
	pos := summoner.Position()
	target.party.summon = &summonRequest{summonerID: summoner.ObjectID(), at: pos}
	target.Send(serverpackets.NewConfirmDlgPacket(serverpackets.ConfirmDlg{
		Message: serverpackets.SystemMessage{ID: summonMsgWishes, Params: []serverpackets.SystemMessageParam{
			serverpackets.TextParam(summoner.Name()),
			serverpackets.ZoneNameParam(pos.X, pos.Y, pos.Z),
		}},
		TimeMs:      summonConfirmTimeMs,
		RequesterID: summoner.ObjectID(),
	}))
}

// CallParty is the CallParty effect (skills.Summoner): every other member
// of the effector's party that passes checkSummonTarget comes to the
// effector at once, with the random offset.
func (w skillWorld) CallParty(effector skills.Creature) {
	s := w.s
	p, ok := s.gameCreature(effector).(*Player)
	if !ok || p.party.party == nil {
		return
	}
	pos := p.Position()
	for _, m := range p.party.party.Members() {
		if s.checkSummonTarget(m, p) && m != p {
			s.teleportAround(m, summonLocation(pos), maxOffsetOnTeleport)
		}
	}
}

// summonLocation is a world position as a data location.
func summonLocation(pos world.Position) data.Location {
	return data.Location{X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z)}
}

var _ skills.Summoner = skillWorld{}
