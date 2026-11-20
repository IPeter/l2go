package game

import (
	"math"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/world"
)

// Walk constants of Mobius.
const (
	// walkTicksPerSecond is GameTimeTaskManager.TICKS_PER_SECOND: a walk
	// advances speed / 10 units per game tick.
	walkTicksPerSecond = 10
	// walkTickMillis is the length of a game tick in milliseconds.
	walkTickMillis = int64(world.GameTick / time.Millisecond)
	// walkBroadcastTicks is how often Mobius repeats the MoveToLocation of a
	// walk under way, when it is not forced (broadcastMoveToLocation).
	walkBroadcastTicks = 10
	// walkMaxSwim is the longest swim one request starts (Mobius
	// moveToLocation in water).
	walkMaxSwim = 700
	// walkGeoCheckDistance: a mouse walk shorter than this ends where
	// geodata blocks it (Mobius moveToLocation, getValidLocation).
	walkGeoCheckDistance = 3000
	// walkHeadingScale turns degrees into the client's heading units
	// (Mobius LocationUtil, 65536 / 360).
	walkHeadingScale = 182.044444444
)

// gameTick returns the game tick of t (Mobius getGameTicks): 100 ms steps
// on the same boundaries as Mobius's, which count from a midnight.
func gameTick(t time.Time) int64 { return t.UnixMilli() / walkTickMillis }

// walkTrack follows a walk the way Mobius MoveData does.
type walkTrack struct {
	// state is the walk the track belongs to.
	state *MoveState
	// started is the game tick the walk started at (moveStartTime);
	// updated is the tick of the last update (moveTimestamp), set by the
	// first one (begun).
	started, updated int64
	begun            bool
	// x and y are the exact position between whole units (xAccurate,
	// yAccurate): the integer position is what they truncate to.
	x, y float64
	// broadcast is the game tick of the last MoveToLocation sent
	// (lastBroadcastTime).
	broadcast int64
}

// step is one update of a walk towards dest (Mobius
// Creature.updatePosition, without geodata). The creature advances
// speed × Δticks / 10 units, Δticks being the game ticks since the last
// update, and arrives when that covers the distance left minus its
// collision radius: then it stands on dest. X and Y advance from the exact
// position the track keeps; Z starts from cur, which follows the client.
// The distance left is 3D, but 2D when the creature is within 100 units
// of dest and more than 50 units above or below it (client and server
// heights disagree), unless it floats (swims or flies).
//
// It returns the new position, whether the walk arrived, and ok false when
// no game tick passed since the last update.
func (w *walkTrack) step(cur, dest world.Position, speed, collision float64, tick int64, floating bool) (next world.Position, arrived, ok bool) {
	if !w.begun {
		w.begun = true
		w.updated = w.started
		w.x, w.y = float64(cur.X), float64(cur.Y)
	}
	if w.updated == tick {
		return cur, false, false
	}
	dx := float64(dest.X) - w.x
	dy := float64(dest.Y) - w.y
	dz := float64(dest.Z) - float64(cur.Z)
	delta := dx*dx + dy*dy
	if !floating && delta < 10000 && dz*dz > 2500 {
		delta = math.Sqrt(delta)
	} else {
		delta = math.Sqrt(delta + dz*dz)
	}
	delta = math.Max(0.00001, delta-collision)
	fraction := math.MaxFloat64
	if delta > 1 {
		passed := speed * float64(tick-w.updated) / walkTicksPerSecond
		fraction = passed / delta
	}
	w.updated = tick
	next = cur
	if fraction > 1 {
		next.X, next.Y, next.Z = dest.X, dest.Y, dest.Z
		return next, true, true
	}
	next.Z = cur.Z + int32(dz*fraction+0.5)
	w.x += dx * fraction
	w.y += dy * fraction
	next.X, next.Y = int32(w.x), int32(w.y)
	return next, false, true
}

// walkHeading returns the client heading of the direction (dx, dy) (Mobius
// LocationUtil.calculateHeadingFrom).
func walkHeading(dx, dy float64) int32 {
	angle := math.Atan2(dy, dx) * (180 / math.Pi)
	if angle < 0 {
		angle += 360
	}
	return int32(angle * walkHeadingScale)
}

// walkCollision is the collision radius Mobius subtracts from the distance
// left: the class template's own (male) radius, as an int
// (getTemplate().getCollisionRadius()).
func walkCollision(p *Player) float64 {
	return float64(int(p.stats.Template().Male.Radius))
}

// walkFloating reports whether p swims: in water outside a castle (Mobius;
// flying comes later).
func walkFloating(p *Player) bool {
	return p.Zones.Inside(ZoneWater) && !p.Zones.Inside(ZoneCastle)
}

// canWalk reports whether p may start a walk: it is in the game and in the
// world, and does not sit (Mobius isMovementDisabled covers teleporting,
// the REST intention sitting).
func canWalk(p *Player) bool {
	return p.state == stateInGame && p.InWorld() && !p.Teleporting() && !p.Sitting
}

// walkDelta returns (x, y, z) - cur as floats.
func walkDelta(cur world.Position, x, y, z int32) (dx, dy, dz float64) {
	return float64(x) - float64(cur.X), float64(y) - float64(cur.Y), float64(z) - float64(cur.Z)
}

// startWalk starts a walk of p to dest (Mobius Creature.moveToLocation),
// straight there, or with an offset to offset - 5 units short of it. It
// sends nothing about the walk (see MovePlayerTo) and reports whether a walk
// started: when p cannot walk it sends ActionFailed, and when p already
// stands within offset of dest it runs the arrival hooks and sends
// ActionFailed.
func (s *Server) startWalk(p *Player, m *motion, dest world.Position, offset int32) bool {
	speed := p.stats.MoveSpeed()
	if !(speed > 0) || !canWalk(p) {
		p.SendActionFailed()
		return false
	}
	cur := p.Position()
	x, y, z := dest.X, dest.Y, dest.Z
	dx, dy, dz := walkDelta(cur, x, y, z)
	distance := math.Sqrt(dx*dx + dy*dy)

	inWater := walkFloating(p)
	if inWater && distance > walkMaxSwim {
		f := walkMaxSwim / distance
		x = cur.X + int32(f*dx)
		y = cur.Y + int32(f*dy)
		z = cur.Z + int32(f*dz)
		dx, dy, dz = walkDelta(cur, x, y, z)
		distance = math.Sqrt(dx*dx + dy*dy)
	}

	var sin, cos float64
	if offset > 0 || distance < 1 {
		// The height difference counts against the offset, and at least 5
		// remain (Mobius's approximation for different heights).
		off := max(float64(offset)-math.Abs(dz), 5)
		if distance < 1 || distance-off <= 0 {
			s.notifyArrived(p, m)
			p.SendActionFailed()
			return false
		}
		sin, cos = dy/distance, dx/distance
		distance -= off - 5
		x = cur.X + int32(distance*cos)
		y = cur.Y + int32(distance*sin)
	} else {
		sin, cos = dy/distance, dx/distance
	}

	// A mouse walk on land ends where geodata blocks it; without geodata
	// the destination stays (Mobius getValidLocation). Mobius would walk
	// around the obstacle on a path (geo.FindPath); walks do not follow
	// paths yet.
	if !inWater && !m.keyboard && distance < walkGeoCheckDistance &&
		!(float64(cur.Z)-float64(z) > 300 && distance < 300) {
		to := s.geo.ValidLocation(geo.Point{X: cur.X, Y: cur.Y, Z: cur.Z}, geo.Point{X: x, Y: y, Z: z})
		x, y, z = to.X, to.Y, to.Z
		vx, vy, _ := walkDelta(cur, x, y, z)
		if math.Sqrt(vx*vx+vy*vy) < 1 {
			p.SendActionFailed()
			return false
		}
	}

	heading := walkHeading(cos, sin)
	p.SetHeading(heading)
	now := s.Now()
	walk := &MoveState{
		Origin:      p.Position(),
		Destination: world.Position{X: x, Y: y, Z: z, Heading: heading},
		Start:       now,
		Speed:       speed,
		Running:     p.Running(),
	}
	p.Move = walk
	m.walk = walkTrack{state: walk, started: gameTick(now)}
	m.pawnStop = false
	if !m.listed {
		m.listed = true
		s.movement.walking = append(s.movement.walking, p)
	}
	return true
}

// walkTick is the movement tick hook (Mobius MovementTaskManager): every
// walking player advances by the game ticks since its last update.
func (s *Server) walkTick(now time.Time) {
	mv := &s.movement
	if len(mv.walking) == 0 {
		return
	}
	tick := gameTick(now)
	batch := append(mv.batch[:0], mv.walking...)
	for _, p := range batch {
		s.advanceWalk(p, tick)
	}
	clear(batch)
	mv.batch = batch[:0]
	mv.walking = slices.DeleteFunc(mv.walking, func(p *Player) bool {
		m := mv.players[p]
		if m == nil {
			return true
		}
		if p.Move != nil && p.Move == m.walk.state && p.state == stateInGame {
			return false
		}
		m.listed = false
		return true
	})
}

// advanceWalk moves p along its walk (Mobius updatePosition) and ends the
// walk when it arrives. Each update uses the speed and the water state of
// that moment, as Mobius reads getMoveSpeed on every update: a walk/run
// toggle, water or a swamp zone (Motion.InWater, Motion.Swamp) or a
// //gmspeed change the pace of a walk under way, the way the client's
// UserInfo and CharInfo show it. MoveState.Speed follows: it holds the
// speed of the last update.
func (s *Server) advanceWalk(p *Player, tick int64) {
	m := s.movement.players[p]
	walk := p.Move
	if m == nil || walk == nil || walk != m.walk.state || p.state != stateInGame {
		return
	}
	if !p.InWorld() {
		// Mobius drops the walk of a creature that is not spawned.
		p.Move = nil
		return
	}
	speed := p.stats.MoveSpeed()
	walk.Speed = speed
	cur := p.Position()
	next, arrived, ok := m.walk.step(cur, walk.Destination, speed, walkCollision(p), tick, walkFloating(p))
	if !ok {
		return
	}
	if next != cur {
		if err := s.MoveObject(p, next); err != nil {
			// Out of the world: Mobius puts a player back at its last
			// server position (World.disposeOutOfBoundsObject).
			s.log.Debug("game: a walk left the world", append(p.logArgs(), "pos", next, "err", err)...)
			back := m.lastServer
			s.stopWalk(p, m, &back)
			return
		}
		if p.Move != walk || p.state != stateInGame {
			return // a hook stopped, replaced or ended the walk
		}
	}
	if arrived {
		s.arriveWalk(p, m)
	}
}

// arriveWalk ends a walk that reached its destination (Mobius
// updatePosition and notifyActionArrived): p and the players around get the
// MoveToLocation of the finished walk, which now starts at its
// destination; a walk with an offset gets StopMove after it; then the
// arrival hooks run.
func (s *Server) arriveWalk(p *Player, m *motion) {
	s.broadcastWalk(p, m, true)
	p.Move = nil
	if m.pawnStop {
		m.pawnStop = false
		s.broadcastStop(p)
	}
	s.notifyArrived(p, m)
}

// broadcastWalk sends the MoveToLocation of p's walk to p and the players
// around (Mobius broadcastMoveToLocation): always when forced, for a walk
// that has not moved yet, or with the arrow keys; otherwise once a second
// at most. It does nothing while p stands.
func (s *Server) broadcastWalk(p *Player, m *motion, force bool) {
	walk := p.Move
	if walk == nil {
		return
	}
	tick := gameTick(s.Now())
	if !force && !m.keyboard && m.walk.begun && tick-m.walk.broadcast < walkBroadcastTicks {
		return
	}
	m.walk.broadcast = tick
	s.world.Broadcast(p, moveToLocationPacket(p, walk.Destination), true)
}

// stopWalk ends p's walk where p stands, or at *at (heading included), and
// sends StopMove to p and the players around (Mobius Creature.stopMove,
// which sends it even when p stood still).
func (s *Server) stopWalk(p *Player, m *motion, at *world.Position) {
	p.Move = nil
	m.pawnStop = false
	if at != nil {
		if err := s.MoveObject(p, *at); err != nil {
			s.log.Debug("game: cannot stop a walk there", append(p.logArgs(), "pos", *at, "err", err)...)
		}
		if p.state != stateInGame {
			return
		}
	}
	s.broadcastStop(p)
}

// broadcastStop sends StopMove at p's position to p and the players around.
func (s *Server) broadcastStop(p *Player) {
	s.world.Broadcast(p, stopMovePacket(p), true)
}

// stopMovePacket is the StopMove of p at its position.
func stopMovePacket(p *Player) []byte {
	pos := p.Position()
	return serverpackets.NewStopMovePacket(serverpackets.StopMove{
		ObjectID: p.ObjectID(),
		X:        pos.X, Y: pos.Y, Z: pos.Z,
		Heading: pos.Heading,
	})
}

// syncWalk puts p where its client says and lets the walk go on from there
// (Mobius setSyncedXYZ).
func (s *Server) syncWalk(p *Player, m *motion, x, y, z int32) {
	pos := p.Position()
	pos.X, pos.Y, pos.Z = x, y, z
	if !s.placePlayer(p, pos) {
		return
	}
	if p.Move != nil && p.Move == m.walk.state {
		m.walk.x, m.walk.y = float64(x), float64(y)
	}
}

// placePlayer sets p's position from a client report (Mobius setXYZ): a
// walk goes on from its own exact position at the next tick, so only Z
// sticks while p walks. It reports false when pos is outside the world.
func (s *Server) placePlayer(p *Player, pos world.Position) bool {
	if err := s.MoveObject(p, pos); err != nil {
		s.log.Debug("game: client position refused", append(p.logArgs(), "pos", pos, "err", err)...)
		return false
	}
	return true
}
