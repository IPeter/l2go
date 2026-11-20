package game

import (
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/world"
)

// The ferry schedules (Plan 13, A14): each route the scripts declared
// (scripts/vehicles/, vehicles.add_route) is a Mobius vehicles/Boat*.java
// engine: a cycle of steps, one per case of its run(), each run by a timer
// or by the end of the boat's path.

const (
	// boatBusyRetry is how long a boat waits for a busy dock before it
	// looks again (Mobius schedule(this, 5000)).
	boatBusyRetry = 5 * time.Second
	// boatBusyShouts: the delay is announced on the first wait and every
	// 36th after (Mobius _shoutCount > 35).
	boatBusyShouts = 35
)

// boatSchedule is a running schedule (Mobius BoatTalkingGludin and its
// siblings): the step it is at and the busy-dock count.
type boatSchedule struct {
	route  scripting.VehicleRoute
	boat   *Boat
	cycle  int
	shouts int
}

// StartBoats puts one boat per route into the world, docked where its route
// says, and runs the first step after the route's start delay (Mobius
// main: getNewBoat, registerEngine, runEngine(180000), dockShip). Nothing
// happens when boats are not allowed (AllowBoat). Loop-only.
func (s *Server) StartBoats(routes []scripting.VehicleRoute) {
	if len(routes) == 0 {
		return
	}
	if !s.gs.World.AllowBoat {
		s.log.Info("boats are off", "routes", len(routes))
		return
	}
	started := 0
	for i := range routes {
		r := routes[i]
		if len(r.Steps) == 0 {
			continue
		}
		at := world.Position{X: int32(r.Start.X), Y: int32(r.Start.Y), Z: int32(r.Start.Z), Heading: int32(r.Start.Heading)} //nolint:gosec // world coordinates fit
		b := s.newBoat(r.Name, r.BoatID, at)
		if b == nil {
			continue
		}
		sc := &boatSchedule{route: r, boat: b}
		b.engine = func() { s.runBoatSchedule(sc) }
		s.boatRunEngine(b, r.StartDelay)
		if r.Dock > 0 {
			s.dockShip(r.Dock, true)
		}
		s.vehicles.schedules = append(s.vehicles.schedules, sc)
		started++
	}
	s.log.Info("boats started", "boats", started)
}

// runBoatSchedule is the run() of a Mobius boat script: the current step,
// unless it waits for a busy dock, then the next step.
func (s *Server) runBoatSchedule(sc *boatSchedule) {
	b := sc.boat
	if !b.InWorld() {
		return
	}
	st := &sc.route.Steps[sc.cycle]
	if st.Await > 0 && s.dockBusy(st.Await) {
		if sc.shouts == 0 {
			s.boatAnnounce(b, st.Busy)
		}
		sc.shouts++
		if sc.shouts > boatBusyShouts {
			sc.shouts = 0
		}
		s.loop.After(boatBusyRetry, b.engine)
		return
	}
	if st.Undock > 0 {
		s.dockShip(st.Undock, false)
	}
	if st.Dock > 0 {
		s.dockShip(st.Dock, true)
	}
	s.boatAnnounce(b, st.Announce)
	if st.Horn != nil {
		s.boatBroadcast(b, boatHorn(b, *st.Horn))
	}
	s.boatPayForRide(b, st.Pay)
	if len(st.Path) > 0 {
		s.boatExecutePath(b, st.Path)
	}
	if st.Wait > 0 {
		s.loop.After(st.Wait, b.engine)
	}
	sc.shouts = 0
	sc.cycle = (sc.cycle + 1) % len(sc.route.Steps)
}
