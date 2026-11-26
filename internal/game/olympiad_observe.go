package game

import (
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/olympiad"
)

// Watching a match (Mobius Olympiad.addSpectator, OlympiadStadium and
// Player.enterOlympiadObserverMode / leaveOlympiadObserverMode): from a
// Grand Olympiad Manager's arena list the player becomes an invisible,
// invulnerable observer at the stadium's centre (ExOlympiadMode 3); the
// fighters' bars show while the fight runs. RequestOlympiadMatchList shows
// the arenas, "OlympiadArenaChange <n>" moves to another one, and
// RequestOlympiadObserverEnd brings the player back.

// The pages of the arena lists.
const (
	olympiadObservePage = "olympiad/olympiad_observe1.htm"
	olympiadArenaPage   = "olympiad/olympiad_observe2.htm"
)

// addSpectator is Mobius Olympiad.addSpectator: during the competition, a
// player not waiting for a match watches stadium id; from the manager
// (storeCoords) it remembers where it was and sees the fighters' bars.
func (s *Server) addSpectator(id int, p *Player, storeCoords bool) {
	f := &s.olympiad
	switch {
	case !f.inComp:
		p.SendSystemMessage(serverpackets.SystemMessageID(olympiad.MsgNotInProgress))
		return
	case s.olympiadRegisteredInComp(p):
		p.SendSystemMessage(serverpackets.SystemMessageID(olympiad.MsgNoObserveWhileWaiting))
		return
	case id < 0 || id >= len(f.stadiums):
		return
	}
	s.enterOlympiadObserverMode(p, id, storeCoords)
	f.spectators[id] = append(f.spectators[id], p)
	if storeCoords {
		s.spectatorEnter(id, p)
	}
}

// enterOlympiadObserverMode is Mobius enterOlympiadObserverMode.
func (s *Server) enterOlympiadObserverMode(p *Player, id int, storeCoords bool) {
	s.Unsummon(p)
	s.StopCubics(p) // cubic.go
	s.LeaveParty(p)
	st := s.olyState(p)
	st.gameID = id
	if p.Sitting {
		s.standUp(p)
	}
	if storeCoords {
		st.last, st.hasLast = p.Position(), true
	}
	st.observer = true
	s.SetTarget(p, nil)
	p.admin.invul = true
	s.setInvisible(p, true)
	pos := s.olympiadStadiumPos(id)
	pos.Heading = p.Position().Heading
	if err := s.Teleport(p, pos); err != nil {
		s.log.Warn("game: olympiad observer teleport failed", append(p.logArgs(), "err", err)...)
	}
	p.Send(serverpackets.NewExOlympiadModePacket(serverpackets.OlympiadModeSpectator))
	p.BroadcastUserInfo()
}

// spectatorEnter is Mobius handleSpectatorEnter: with the fight running,
// the new spectator gets both fighters' bars.
func (s *Server) spectatorEnter(id int, p *Player) {
	g := s.olympiad.games[id]
	if g == nil {
		return
	}
	for _, q := range g.players {
		st := s.olyOf(q)
		if q == nil || st == nil || !st.inMode || !st.start {
			return
		}
	}
	p.Send(olympiadUserInfo(g.players[0], 1))
	p.Send(olympiadUserInfo(g.players[1], 2))
}

// removeSpectator takes p off a stadium's spectators.
func (s *Server) removeSpectator(id int, p *Player) {
	if id < 0 || id >= olympiad.Stadiums {
		return
	}
	list := s.olympiad.spectators[id]
	if i := slices.Index(list, p); i >= 0 {
		s.olympiad.spectators[id] = slices.Delete(list, i, i+1)
	}
}

// spectatorArena is Mobius getSpectatorArena: the stadium p watches, -1 for
// none.
func (s *Server) spectatorArena(p *Player) int {
	for i, list := range s.olympiad.spectators {
		if slices.Contains(list, p) {
			return i
		}
	}
	return -1
}

// leaveOlympiadObserverMode is Mobius leaveOlympiadObserverMode: back where
// the player was (ExOlympiadMode 0), visible and mortal again unless a GM.
func (s *Server) leaveOlympiadObserverMode(p *Player) {
	st := s.olyOf(p)
	if st == nil || st.gameID == -1 {
		return
	}
	s.removeSpectator(st.gameID, p)
	st.gameID = -1
	st.observer = false
	s.SetTarget(p, nil)
	p.Send(serverpackets.NewExOlympiadModePacket(serverpackets.OlympiadModeNone))
	last := st.last
	st.hasLast = false
	if !p.IsGM() {
		s.setInvisible(p, false)
		p.admin.invul = false
	}
	s.teleportAround(p, data.Location{X: int(last.X), Y: int(last.Y), Z: int(last.Z)}, maxOffsetOnTeleport)
	p.BroadcastUserInfo()
}

// handleObserverEnd handles RequestOlympiadObserverEnd (D0 12).
func (s *Server) handleObserverEnd(p *Player, _ []byte) {
	if p.InObserverMode() {
		s.leaveOlympiadObserverMode(p)
	}
}

// handleOlympiadMatchList handles RequestOlympiadMatchList (D0 13): an
// observer's arena list (Mobius sendMatchList).
func (s *Server) handleOlympiadMatchList(p *Player, _ []byte) {
	if p.InObserverMode() {
		s.sendOlympiadArenaList(p)
	}
}

// olympiadTitles is Mobius getAllTitles: "one / two" of the games whose
// nobles went to the stadium, by stadium.
func (s *Server) olympiadTitles() map[int]string {
	titles := map[int]string{}
	for i, g := range s.olympiad.games {
		if g != nil && g.ported {
			titles[i] = g.names[0] + " / " + g.names[1]
		}
	}
	return titles
}

// sendOlympiadArenaList is Mobius sendMatchList: each arena's state
// ("Initial State", "Standby", "Playing": the last two by the shared
// _gameIsStarted, as Mobius) and its fighters.
func (s *Server) sendOlympiadArenaList(p *Player) {
	pg := s.pageFile(p, 0, olympiadArenaPage)
	titles := s.olympiadTitles()
	for i := range len(s.olympiad.stadiums) {
		state, players := "Initial State", "&nbsp;"
		if t, ok := titles[i]; ok {
			state = "Standby"
			if s.olympiad.gameIsStarted {
				state = "Playing"
			}
			players = t
		}
		n := strconv.Itoa(i + 1)
		pg.replace("%state"+n+"%", state)
		pg.replace("%players"+n+"%", players)
	}
	s.sendPage(p, pg)
}

// olympiadArenaChange is the "OlympiadArenaChange <n>" bypass (Mobius
// bypassChangeArena): an observer moves to another stadium.
func (s *Server) olympiadArenaChange(p *Player, command string) {
	if !p.InObserverMode() {
		return
	}
	parts := strings.Fields(command)
	if len(parts) < 2 {
		return
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return
	}
	arena := s.spectatorArena(p)
	if arena < 0 {
		return
	}
	s.removeSpectator(arena, p)
	p.Send(serverpackets.NewExOlympiadMatchEndPacket())
	s.addSpectator(id, p, false)
	s.spectatorEnter(id, p)
}
