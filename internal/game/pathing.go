package game

import (
	"math"

	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/world"
)

// Walking with geodata (Plan 13, decision A3): the geodata part of Mobius
// Creature.moveToLocation and moveToNextRoutePoint, for players
// (movement_walk.go) and NPCs (npc_move.go).
//
// A move on the ground is clipped where the terrain, a door or a fence
// stops it (getValidLocation). When that cut more than 30 units, A* looks
// for a path to the original destination; an attackable that finds none
// tries the points of a 500-unit grid around it. With a path the creature
// walks its waypoints one leg at a time (a route), the last leg to the exact
// destination; with none it walks straight and ignores geodata. Clicks
// farther than 3,000 units, water destinations, keyboard moves and
// attacking players never run A*; boats do not exist yet. All of it runs
// only with pathfinding on (GeoEngineConfig.PATHFINDING), as in Mobius.

// Movement constants of Mobius moveToLocation.
const (
	// pathMinCut: A* runs when getValidLocation shortened the move by more
	// than this.
	pathMinCut = 30
	// pathGridStep is the spacing of the attackables' fallback grid, and
	// how far it reaches past the move's bounding box at most.
	pathGridStep = 500
	// pathClickMax: a player's click farther than this is neither clipped
	// nor pathed ("should be able to click far away and move").
	pathClickMax = 3000
	// pathFallDZ: a move this much downwards over a shorter distance is a
	// fall, which geodata never corrects.
	pathFallDZ = 300
	// pathLedgeDZ: a monster that sees its target this far up or down walks
	// to it unclipped ("monsters can move on ledges").
	pathLedgeDZ = 100
)

// geoRoute is a walk along a geodata path (Mobius MoveData geoPath,
// onGeodataPathIndex, geoPathGtx/Gty and geoPathAccurateTx/Ty).
type geoRoute struct {
	points []geo.Point
	// index is the waypoint the current leg walks to; -1 once the route was
	// given up.
	index int
	// gtx and gty are the cell of the destination the route was made for:
	// a new move to the same cell keeps the route.
	gtx, gty int32
	// tx and ty are where the last leg ends: the exact destination.
	tx, ty int32
}

// on is Mobius isOnGeodataPath: a route whose last leg has not started.
func (r *geoRoute) on() bool {
	return r != nil && r.index != -1 && r.index != len(r.points)-1
}

// nextLeg moves the route to its next leg (moveToNextRoutePoint) and
// returns where that leg ends: the next waypoint, or for the last leg the
// exact destination at the last waypoint's height.
func (r *geoRoute) nextLeg() world.Position {
	last := r.index == len(r.points)-2
	r.index++
	p := r.points[r.index]
	if last {
		return world.Position{X: r.tx, Y: r.ty, Z: p.Z}
	}
	return geoPos(p)
}

// walkGeo is the geodata state of a walk (walkTrack.geo): its route and,
// for a player, where the door look-ahead stops it (pathing_door.go).
type walkGeo struct {
	route *geoRoute
	// stop is set when a door or fence lies ahead of a mouse walk: the walk
	// stops once it covered stopAt units (2D) from its origin.
	stop   bool
	stopAt float64
}

// pathCell is the cell Mobius's moveToLocation compares routes by:
// (x - WORLD_X_MIN) >> 4.
func pathCell(x, y int32) (int32, int32) {
	return (x - world.WorldXMin) >> 4, (y - world.WorldYMin) >> 4
}

// geoMove is one moveToLocation as its geodata part sees it.
type geoMove struct {
	// cur is where the creature stands.
	cur world.Position
	// x, y, z, distance, sin and cos are the move as computed so far (the
	// destination, its 2D distance and direction); geoPlanMove updates them.
	x, y, z  int32
	distance float64
	sin, cos float64
	// vertical is a flyer's straight up or down move.
	vertical bool

	player, playable, attackable bool
	// direct is a player attacking: clipped, never pathed.
	direct bool
	// destWater is a player's destination in water.
	destWater bool
	// ledge is a monster with a visible target more than 100 up or down.
	ledge bool
	// controlBlocked is a creature under fear (isControlBlocked).
	controlBlocked bool
	// current is the route of the walk under way, nil without one.
	current *geoRoute

	// route is the new walk's route, nil when it walks straight.
	route *geoRoute
}

// measure recomputes distance from cur to the destination (Mobius:
// dz × dz for a vertical move, else the 2D distance).
func (m *geoMove) measure() {
	dx, dy, dz := walkDelta(m.cur, m.x, m.y, m.z)
	if m.vertical {
		m.distance = dz * dz
		return
	}
	m.distance = math.Sqrt(dx*dx + dy*dy)
}

// falls is Mobius's "prohibit correcting destination if character wants to
// fall".
func (m *geoMove) falls() bool {
	return float64(m.cur.Z)-float64(m.z) > pathFallDZ && m.distance < pathFallDZ
}

// geoPlanMove is the movement checks and pathfinding of Mobius
// moveToLocation, with pathfinding on. It reports true when the creature
// keeps the route it walks: the new destination lies in the cell that
// route goes to (Mobius answers ActionFailed and changes nothing).
func (s *Server) geoPlanMove(m *geoMove) (keep bool) {
	if !s.geo.Pathfinding() {
		return false
	}
	cur := m.cur
	ox, oy, oz, odist := m.x, m.y, m.z, m.distance
	gtx, gty := pathCell(ox, oy)
	if r := m.current; r.on() {
		if r.gtx == gtx && r.gty == gty {
			return true
		}
		r.index = -1
	}
	if m.direct || (!m.destWater && !(m.player && m.distance > pathClickMax) && !m.ledge && !m.falls()) {
		d := s.geo.ValidLocation(geoPoint(cur), geo.Point{X: m.x, Y: m.y, Z: m.z})
		m.x, m.y = d.X, d.Y
		if !m.player {
			m.z = d.Z
		}
		m.measure()
	}
	tried := false
	if !m.direct && odist-m.distance > pathMinCut && !m.controlBlocked {
		path := s.geo.FindPath(geoPoint(cur), geo.Point{X: ox, Y: oy, Z: oz}, m.player)
		tried = path != nil
		found := len(path) > 1
		if !found && m.attackable {
			var tx, ty int32
			if path, tx, ty, found = s.pathGrid(cur, ox, oy, oz); found {
				ox, oy = tx, ty
			}
		}
		if found {
			m.route = &geoRoute{points: path, gtx: gtx, gty: gty, tx: ox, ty: oy}
			first := path[0]
			m.x, m.y, m.z = first.X, first.Y, first.Z
			m.measure()
			if m.distance > 0 && !m.vertical {
				dx, dy, _ := walkDelta(cur, m.x, m.y, m.z)
				m.sin, m.cos = dy/m.distance, dx/m.distance
			}
		} else {
			// No path: walk straight there, ignoring geodata (Mobius,
			// "tested at retail on October 21st 2024").
			m.x, m.y, m.z, m.distance = ox, oy, oz, odist
		}
	}
	if m.playable && !tried && m.route == nil && m.distance < pathClickMax && !m.falls() {
		d := s.geo.ValidLocation(geoPoint(cur), geo.Point{X: m.x, Y: m.y, Z: m.z})
		m.x, m.y, m.z = d.X, d.Y, d.Z
		m.measure()
	}
	return false
}

// pathGrid is the attackables' fallback of moveToLocation: the points of a
// 500-unit grid over the move's bounding box, widened by its larger side
// (at most 500), are tried from the one nearest to the destination on; the
// nearest with a path wins. It returns that path and point.
func (s *Server) pathGrid(cur world.Position, ox, oy, oz int32) (path []geo.Point, tx, ty int32, found bool) {
	xMin, xMax := min(cur.X, ox), max(cur.X, ox)
	yMin, yMax := min(cur.Y, oy), max(cur.Y, oy)
	diff := min(max(xMax-xMin, yMax-yMin), pathGridStep)
	xMin, xMax, yMin, yMax = xMin-diff, xMax+diff, yMin-diff, yMax+diff
	short := math.MaxFloat64
	for sx := xMin; sx < xMax; sx += pathGridStep {
		for sy := yMin; sy < yMax; sy += pathGridStep {
			dx, dy := float64(sx)-float64(ox), float64(sy)-float64(oy)
			d := math.Sqrt(dx*dx + dy*dy)
			if d >= short {
				continue
			}
			if p := s.geo.FindPath(geoPoint(cur), geo.Point{X: sx, Y: sy, Z: oz}, false); len(p) > 1 {
				short, path, tx, ty = d, p, sx, sy
			}
		}
	}
	return path, tx, ty, path != nil
}

// playerGeoMove is the geodata part of moveToLocation for a player's mouse
// walk on land. It reports false, after ActionFailed, when no walk starts:
// the player keeps its route to the same cell, or geodata leaves nothing
// to walk (then the player turns idle unless it attacks or casts).
func (s *Server) playerGeoMove(p *Player, m *motion, g *geoMove) bool {
	ai := s.attack.players[p]
	g.player, g.playable = true, true
	g.direct = ai != nil && ai.intention == playerAttacking
	g.destWater = s.zoneAt(world.Position{X: g.x, Y: g.y, Z: g.z}, "WaterZone") != nil
	if p.Move != nil && p.Move == m.walk.state {
		g.current = m.walk.geo.route
	}
	if s.geoPlanMove(g) {
		p.SendActionFailed()
		return false
	}
	if g.distance < 1 {
		if ai == nil || (ai.intention != playerAttacking && ai.intention != playerCasting) {
			s.playerIntendIdle(p)
		}
		p.SendActionFailed()
		return false
	}
	return true
}

// npcGeoMove is the geodata part of moveToLocation for an NPC on the ground
// in an active neighbourhood. It reports false when no walk starts: the NPC
// keeps its route to the same cell, or geodata leaves nothing to walk (then
// the NPC turns idle unless it attacks or casts).
//
// Mobius lets a monster walk unclipped up or down a ledge to a target it
// sees, its creature target; monsters here keep their targets in the AI, so
// like Mobius monsters outside skill use they have none and that rule does
// not apply.
func (s *Server) npcGeoMove(n *Npc, g *geoMove) bool {
	g.attackable = n.class != nil && n.class.attackable
	g.controlBlocked = n.skill != nil && n.skill.el != nil && n.skill.el.IsAffected(skills.FlagFear)
	if n.Move != nil && n.Move == n.ai.walk.state {
		g.current = n.ai.walk.geo.route
	}
	if s.geoPlanMove(g) {
		return false
	}
	if g.distance < 1 && s.geo.Pathfinding() {
		if in := n.ai.intention; in != NpcAttack && in != NpcCast {
			s.npcSetIdle(n)
		}
		return false
	}
	return true
}

// nextPlayerLeg starts the next leg of p's route when its walk arrived at a
// waypoint (Mobius notifyActionArrived and moveToNextRoutePoint): p turns
// to it, and p and the players around get its MoveToLocation. It reports
// false when p's walk has no further leg or p cannot move on.
func (s *Server) nextPlayerLeg(p *Player, m *motion) bool {
	r := m.walk.geo.route
	if !r.on() {
		return false
	}
	speed := p.stats.MoveSpeed()
	if !(speed > 0) || !canWalk(p) || s.CastingNow(p) || s.CastingSimultaneouslyNow(p) {
		return false
	}
	dest := r.nextLeg()
	cur := p.Position()
	dx, dy, _ := walkDelta(cur, dest.X, dest.Y, dest.Z)
	heading := cur.Heading
	if dx != 0 || dy != 0 {
		heading = walkHeading(dx, dy)
		p.SetHeading(heading)
	}
	dest.Heading = heading
	now := s.Now()
	walk := &MoveState{Origin: p.Position(), Destination: dest, Start: now, Speed: speed, Running: p.Running()}
	p.Move = walk
	m.walk = walkTrack{state: walk, started: gameTick(now)}
	m.walk.geo.route = r
	s.planDoorStop(p, m)
	s.broadcastWalk(p, m, true)
	return true
}

// nextNpcLeg is nextPlayerLeg for an NPC: the next leg of its route, shown
// to the players around with MoveToLocation.
func (s *Server) nextNpcLeg(n *Npc) bool {
	r := n.ai.walk.geo.route
	if !r.on() {
		return false
	}
	speed := n.Stats.MoveSpeed()
	if !(speed > 0) || npcMovementDisabled(n) {
		return false
	}
	dest := r.nextLeg()
	cur := n.Position()
	dx, dy, _ := walkDelta(cur, dest.X, dest.Y, dest.Z)
	distance := math.Sqrt(dx*dx + dy*dy)
	heading := cur.Heading
	if distance != 0 {
		heading = walkHeading(dx, dy)
		n.SetHeading(heading)
	}
	dest.Heading = heading
	now := s.Now()
	walk := &MoveState{Origin: n.Position(), Destination: dest, Start: now, Speed: speed, Running: n.Running()}
	n.Move = walk
	n.ai.walk = walkTrack{state: walk, started: gameTick(now)}
	n.ai.walk.geo.route = r
	if int64(walkTicksPerSecond*distance/speed)*walkTickMillis > npcArrivedRevalidateWalk {
		s.loop.After(npcArrivedRevalidate, func() { s.notifyArrivedRevalidate(n) })
	}
	s.broadcastNpcMove(n, true)
	return true
}

// broadcastPawnWalk shows p's walk up to target to p and the players around
// (the broadcast of Mobius moveToPawn): MoveToPawn, or on a route the leg's
// MoveToLocation, since the client would walk straight to the pawn. A walk
// on a route is no walk up to a pawn any more: it ends without StopMove.
// It reports whether p walks a route.
func (s *Server) broadcastPawnWalk(p *Player, target world.Object, offset int32) bool {
	if m := s.movement.players[p]; m != nil && s.onRoute(p) {
		m.pawnStop = false
		s.broadcastWalk(p, m, false)
		return true
	}
	s.world.Broadcast(p, npcMoveToPawnPacket(p, target, offset), true)
	return false
}

// walkGoal returns where p's walk mv ends: the end of its route, else its
// destination. The approach checks (talking, picking up, opening a store)
// compare it, since a route's destination changes at every leg. A zero
// position for no walk.
func (s *Server) walkGoal(p *Player, mv *MoveState) world.Position {
	if mv == nil {
		return world.Position{}
	}
	if m := s.movement.players[p]; m != nil && mv == m.walk.state {
		if r := m.walk.geo.route; r != nil && r.index >= 0 {
			return world.Position{X: r.tx, Y: r.ty, Z: r.points[len(r.points)-1].Z}
		}
	}
	return mv.Destination
}

// onRoute reports whether the creature walks a route leg that is not its
// last (Mobius isOnGeodataPath).
func (s *Server) onRoute(c world.Object) bool {
	switch c := c.(type) {
	case *Player:
		m := s.movement.players[c]
		return m != nil && c.Move != nil && c.Move == m.walk.state && m.walk.geo.route.on()
	case *Npc:
		return c.Move != nil && c.Move == c.ai.walk.state && c.ai.walk.geo.route.on()
	}
	return false
}
