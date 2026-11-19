package game

import (
	"errors"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
)

// registerCore registers the packet handlers of the core: the teleport
// round trip, the command packets that dispatch to the command registries,
// and the requests the client sends while entering the world.
func (s *Server) registerCore() error {
	return errors.Join(
		s.Register(clientpackets.OpAppearing, s.handleAppearing),
		s.Register(clientpackets.OpSendBypassBuildCmd, s.handleBuildCmd),
		s.Register(clientpackets.OpBypassUserCmd, s.handleUserCmd),
		s.Register(clientpackets.OpRequestQuestList, s.handleRequestQuestList),
		s.Register(clientpackets.OpRequestSkillCoolTime, s.handleRequestSkillCoolTime),
		s.RegisterEx(clientpackets.ExOpRequestManorList, s.handleRequestManorList),
	)
}

// handleRequestManorList answers RequestManorList (D0 08), which the client
// sends after entering the world, with the castles that have a manor
// (Mobius RequestManorList).
func (s *Server) handleRequestManorList(p *Player, _ []byte) { p.Send(manorListPacket) }

// handleRequestQuestList answers RequestQuestList (0x63) with the quest list,
// empty until quests exist (Mobius RequestQuestList).
func (s *Server) handleRequestQuestList(p *Player, _ []byte) { p.Send(emptyQuestList) }

// handleRequestSkillCoolTime answers RequestSkillCoolTime (0x9D) with the
// skills on cooldown, none until skills exist (Mobius RequestSkillCoolTime).
func (s *Server) handleRequestSkillCoolTime(p *Player, _ []byte) { p.Send(emptySkillCoolTime) }
