package game

import (
	"errors"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
)

// miniMapPacket is the ShowMiniMap Mobius answers RequestShowMiniMap with:
// map 1665, and the Seven Signs period, which is the competition (the
// period of a new Mobius database) until Seven Signs exists.
var miniMapPacket = serverpackets.NewShowMiniMapPacket(serverpackets.ShowMiniMap{
	MapID:  serverpackets.MiniMapID,
	Period: serverpackets.SevenSignsCompetition,
})

// registerCore registers the packet handlers of the core: the teleport
// round trip, the command packets that dispatch to the command registries,
// the requests the client sends while entering the world, and the map.
func (s *Server) registerCore() error {
	return errors.Join(
		s.Register(clientpackets.OpAppearing, s.handleAppearing),
		s.Register(clientpackets.OpSendBypassBuildCmd, s.handleBuildCmd),
		s.Register(clientpackets.OpBypassUserCmd, s.handleUserCmd),
		s.Register(clientpackets.OpRequestQuestList, s.handleRequestQuestList),
		s.Register(clientpackets.OpRequestSkillCoolTime, s.handleRequestSkillCoolTime),
		s.Register(clientpackets.OpRequestShowMiniMap, s.handleRequestShowMiniMap),
		s.RegisterEx(clientpackets.ExOpRequestManorList, s.handleRequestManorList),
	)
}

// handleRequestShowMiniMap answers RequestShowMiniMap (0xCD), the map key,
// with ShowMiniMap, which opens the map window (Mobius RequestShowMiniMap).
func (s *Server) handleRequestShowMiniMap(p *Player, _ []byte) { p.Send(miniMapPacket) }

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
