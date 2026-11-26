package game

import (
	"time"

	"github.com/IPeter/l2go/internal/olympiad"
	"github.com/IPeter/l2go/internal/world"
)

// The match making (Mobius OlympiadManager.run), a loop timer from the
// competition's start: every 30 s, with enough nobles waiting, the free
// stadiums get matches (olympiad.Config.MakeMatches) and the new games
// start one second apart. After the competition it waits for the games to
// end, then empties the waiting lists.

// startOlympiadManager starts the manager's rounds (a new thread per
// competition in Mobius).
func (s *Server) startOlympiadManager() {
	f := &s.olympiad
	if f.state.Period != olympiad.Competition {
		return
	}
	f.managerOn = true
	s.cancelTimer(&f.managerTimer)
	f.managerTimer = s.loop.After(0, s.olympiadManagerRound)
}

// olympiadManagerRound is one pass of OlympiadManager.run's loop.
func (s *Server) olympiadManagerRound() {
	f := &s.olympiad
	f.managerTimer = 0
	if !f.managerOn {
		return
	}
	if !f.inComp {
		s.olympiadManagerDrain()
		return
	}
	if f.nobles.Len() == 0 {
		f.managerTimer = s.loop.After(olympiadIdlePeriod, s.olympiadManagerRound)
		return
	}
	next := olympiadManagerPeriod
	pairings := f.cfg.MakeMatches(f.rnd, &f.queues, func(i int) bool { return i < len(f.stadiums) && f.games[i] == nil })
	if len(pairings) > 0 {
		for _, pr := range pairings {
			s.newOlympiadGame(pr) // olympiad_game.go
		}
		// Mobius starts the games of the stadiums 0 to size-1 of its queue,
		// one second apart, and waits a second for every index.
		size := 0
		for _, g := range f.games {
			if g != nil {
				size++
			}
		}
		for i := 0; i < size; i++ {
			if g := f.games[i]; g != nil && !g.started {
				g.started = true
				delay := time.Duration(i) * olympiadStartGap
				g.timer = s.loop.After(delay, g.run)
			}
		}
		next += time.Duration(size) * olympiadStartGap
	}
	f.managerTimer = s.loop.After(next, s.olympiadManagerRound)
}

// olympiadManagerDrain is the end of OlympiadManager.run: once every game
// ended, the waiting lists empty and no battle runs any more.
func (s *Server) olympiadManagerDrain() {
	f := &s.olympiad
	for _, g := range f.games {
		if g != nil {
			f.managerTimer = s.loop.After(olympiadManagerPeriod, s.olympiadManagerRound)
			return
		}
	}
	f.queues.Clear()
	f.battleStarted = false
	f.managerOn = false
}

// olympiadStadiumPos returns the centre of a stadium.
func (s *Server) olympiadStadiumPos(i int) world.Position {
	st := s.olympiad.stadiums[i]
	return world.Position{X: int32(st.X), Y: int32(st.Y), Z: int32(st.Z)} //nolint:gosec // world coordinates
}
