package game

import (
	"math"

	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/world"
)

// The look-ahead of a walking player (Plan 13, decision A4): Mobius
// Creature.updatePosition checks, at every step of a player that does not
// fly, the point 10 × speed / 100 units ahead in its heading:
//   - with the arrow keys, whether it can walk there (terrain, doors and
//     fences); if not it stops where it stands;
//   - after a click farther than 3,000 units away, the same while the
//     destination is that far; if not it turns idle;
//   - while it attacks, the same; if not it stops and the attack goes on;
//   - otherwise, when its region holds doors or fences, whether one stands
//     between; if so it turns idle and stops.
//
// The first three run at every step, as in Mobius: they are short line
// walks. The door case is computed once, when a leg starts (planDoorStop)
// or a door or fence nearby changes (GeoBlockersChanged): the distance
// along the leg at which a door or fence comes within the look-ahead.

// lookAheadScale is Mobius's look-ahead: 10 × speed / 100 units.
const lookAheadScale = 10.0 / 100

// doorSearchChunk is how much of a leg one door query covers: Mobius looks
// doors up from the region of the walker, so a long leg is asked in pieces
// that stay within the region and its neighbours.
const doorSearchChunk = 512

// lookAheadPoint is the point Mobius's updatePosition looks at: front units
// ahead of pos in its heading, both at pos's height.
func lookAheadPoint(pos world.Position, front float64) geo.Point {
	radian := float64(pos.Heading) / headingPerDegree * (math.Pi / 180)
	course := math.Pi // Math.toRadians(180)
	x1 := int32(math.Cos(math.Pi+radian+course) * front)
	y1 := int32(math.Sin(math.Pi+radian+course) * front)
	return geo.Point{X: pos.X + x1, Y: pos.Y + y1, Z: pos.Z}
}

// walkLookAhead runs the look-ahead of p's walk before its step at tick. It
// reports true when it stopped the walk.
func (s *Server) walkLookAhead(p *Player, m *motion, cur world.Position, tick int64) bool {
	walk := p.Move
	updated := m.walk.updated
	if !m.walk.begun {
		updated = m.walk.started
	}
	if walk == nil || updated == tick {
		return false // no step this tick
	}
	front := lookAheadScale * p.stats.MoveSpeed()
	if m.keyboard {
		if !s.geo.CanMoveToTarget(geoPoint(cur), lookAheadPoint(cur, front)) {
			s.stopWalk(p, m, &cur)
			return true
		}
		return false
	}
	x, y := float64(cur.X), float64(cur.Y)
	if m.walk.begun {
		x, y = m.walk.x, m.walk.y
	}
	dx, dy := float64(walk.Destination.X)-x, float64(walk.Destination.Y)-y
	ai := s.attack.players[p]
	switch {
	case math.Sqrt(dx*dx+dy*dy) > pathClickMax:
		if !s.geo.CanMoveToTarget(geoPoint(cur), lookAheadPoint(cur, front)) {
			s.dropRoute(m)
			s.playerIntendIdle(p)
			return true
		}
	case ai != nil && ai.intention == playerAttacking:
		if !s.geo.CanMoveToTarget(geoPoint(cur), lookAheadPoint(cur, front)) {
			// Mobius stops the walk and lets the attack think again.
			s.dropRoute(m)
			s.broadcastStop(p)
			p.Move = nil
			s.notifyArrived(p, m)
			return true
		}
	case m.walk.geo.stop && cur.Distance2D(walk.Origin) >= m.walk.geo.stopAt:
		// A door or fence ahead: idle (which stops the walk with StopMove),
		// then Mobius's stopMove(null) sends StopMove once more.
		s.dropRoute(m)
		s.playerIntendIdle(p)
		if p.state == stateInGame && p.InWorld() {
			s.stopWalk(p, m, nil)
		}
		return true
	}
	return false
}

// dropRoute gives up the route of p's walk (onGeodataPathIndex = -1).
func (s *Server) dropRoute(m *motion) {
	if r := m.walk.geo.route; r != nil {
		r.index = -1
	}
}

// planDoorStop finds where the door look-ahead stops p's current mouse
// walk leg: the first point of the leg from which a door or fence lies
// within the look-ahead. Without blockers nothing ever stops it.
func (s *Server) planDoorStop(p *Player, m *motion) {
	g := &m.walk.geo
	g.stop = false
	walk := p.Move
	if s.geo.blockers == nil || walk == nil || walk != m.walk.state || m.keyboard {
		return
	}
	cur := p.Position()
	dest := walk.Destination
	dx, dy := float64(dest.X)-float64(cur.X), float64(dest.Y)-float64(cur.Y)
	left := math.Sqrt(dx*dx + dy*dy)
	if left < 1 {
		return
	}
	front := lookAheadScale * p.stats.MoveSpeed()
	ux, uy := dx/left, dy/left
	at := func(d float64) geo.Point {
		return geo.Point{X: cur.X + int32(ux*d), Y: cur.Y + int32(uy*d), Z: cur.Z}
	}
	blocked := func(a, b geo.Point) bool {
		return s.geo.doorsBetween(a, b, false) || s.geo.fencesBetween(a, b)
	}
	reach := left + front
	for a := 0.0; a < reach; a += doorSearchChunk {
		b := math.Min(a+doorSearchChunk, reach)
		pa := at(a)
		if !blocked(pa, at(b)) {
			continue
		}
		lo, hi := a, b // blocked(pa, at(hi)) holds, blocked(pa, at(lo)) does not
		for hi-lo > 1 {
			mid := (lo + hi) / 2
			if blocked(pa, at(mid)) {
				hi = mid
			} else {
				lo = mid
			}
		}
		g.stop = true
		g.stopAt = cur.Distance2D(walk.Origin) + hi - front
		return
	}
}

// GeoBlockersChanged tells the walking players that a door or fence opened,
// closed, appeared or went (Phase 10.3 calls it): the look-ahead of every
// mouse walk is computed again from where its player stands. Loop-only.
func (s *Server) GeoBlockersChanged() {
	for _, p := range s.movement.walking {
		if m := s.movement.players[p]; m != nil && p.Move != nil && p.Move == m.walk.state {
			s.planDoorStop(p, m)
		}
	}
}
