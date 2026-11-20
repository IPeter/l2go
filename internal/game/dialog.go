package game

import (
	"errors"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/world"
)

// The dialog feature: NPC dialogs (dialog_npc.go), the HTML pages
// (html.go), bypasses and links (bypass.go), the Lua NPC hooks
// (dialog_script.go), teleporters (teleporter.go), zones (zone.go) and map
// regions (mapregion.go).
func init() { registerFeature("dialog", setupDialog) }

// dialogFeature is the dialog, bypass, teleporter, zone and map region
// feature's state on the Server (Server.dialog). Loop-only.
type dialogFeature struct {
	html *htmlLoader
	// zones indexes the zones of the world data, nil without world data.
	zones *zoneIndex
	// players is the feature's state per player in the game.
	players map[world.ObjectID]*dialogPlayer
}

// dialogPlayer is the dialog feature's state of one player in the game.
type dialogPlayer struct {
	p *Player
	// lastQuestNpc is the NPC the player last talked to through a script's
	// talk (Mobius lastQuestNpcObject): script links still reach it when the
	// last NPC clicked is out of range.
	lastQuestNpc world.ObjectID

	zones zoneTrack
}

func setupDialog(s *Server) error {
	s.dialog.html = newHTMLLoader(s.settings.HTMLDir)
	s.dialog.players = make(map[world.ObjectID]*dialogPlayer)
	if s.worldData != nil {
		s.dialog.zones = newZoneIndex(s.worldData)
	}
	s.checkHTMLDir()

	s.OnNpcInteract(s.npcInteract)
	s.OnPlayerMoved(s.zonesMoved)
	s.OnPlayerEnter(func(p *Player) { s.revalidateZones(p, false) })
	s.OnTeleport(func(p *Player, _ world.Position) { s.dialogState(p).zones.force = true })
	s.OnPlayerLeave(s.dialogLeave)
	return errors.Join(
		s.Register(clientpackets.OpRequestBypassToServer, s.handleBypass),
		s.Register(clientpackets.OpRequestLinkHtml, s.handleLinkHtml),
		s.RegisterUserCommand(clientpackets.UserCmdLoc, s.userCmdLoc),
	)
}

// dialogState returns the dialog feature's state of p, made on first use.
func (s *Server) dialogState(p *Player) *dialogPlayer {
	st := s.dialog.players[p.ID()]
	if st == nil || st.p != p {
		st = &dialogPlayer{p: p}
		s.dialog.players[p.ID()] = st
	}
	return st
}

// dialogLeave drops the state of a player that leaves the game.
func (s *Server) dialogLeave(p *Player) {
	st := s.dialog.players[p.ID()]
	if st == nil || st.p != p {
		return
	}
	delete(s.dialog.players, p.ID())
}
