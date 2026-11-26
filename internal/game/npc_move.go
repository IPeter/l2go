package game

import (
	"math"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// Server-driven NPC movement (Plan 09 Phase 6.5, on the W7 movement data):
// Mobius Creature.moveToLocation, updatePosition, stopMove,
// broadcastMoveToLocation and setRunning for NPCs, and AbstractAI's moveTo,
// moveToPawn, clientStopMoving and clientStoppedMoving.
//
// A walking NPC holds its MoveState in Npc.Move and the walk bookkeeping in
// its AI state; the NPC movement tick advances every walking NPC by the game
// ticks since its last update, with the players' formula (walkTrack.step).
// Arrival notifies the AI (Mobius notifyActionArrived).

// Movement constants of Mobius.
const (
	// npcArrivedRevalidate is when a long walk makes its NPC think once on
	// the way (Mobius moveToLocation: a walk of more than 3 s gets
	// notifyActionArrivedRevalidate after 2 s).
	npcArrivedRevalidate     = 2 * time.Second
	npcArrivedRevalidateWalk = 3000 // milliseconds
	// npcPawnRepathTicks is how long a moveToPawn towards the same object
	// with the same offset is kept before it is sent again (Mobius
	// _moveToPawnTimeout: 1,000 ms in game ticks).
	npcPawnRepathTicks = 1000 / walkTickMillis
	// npcPawnMinOffset is the smallest offset of a moveToPawn.
	npcPawnMinOffset = 10
	// npcMaxZStep is the height step of one update that sends an attackable
	// back to its spawn (Mobius updatePosition: "prevent attackables
	// teleporting to another ground layer while moving").
	npcMaxZStep = 300
)

// npcCollision is the NPC's collision radius as Mobius's template keeps it
// for movement and ranges: cut to an int (CreatureTemplate.getCollisionRadius).
func npcCollision(n *Npc) int32 { return int32(n.Template.Collision.Radius) }

// npcFlying reports whether the NPC flies (Mobius isFlying).
func npcFlying(n *Npc) bool { return n.Template.Status.Flying }

// npcMovementDisabled is Mobius isMovementDisabled for an NPC: dead,
// immobile (canMove false), paralyzed by //para, not in the world, or
// immobilized, stunned, rooted, asleep or paralyzed by an effect.
func npcMovementDisabled(n *Npc) bool {
	return n.Dead || !n.Template.Status.CanMove || !n.InWorld() || adminParalyzed(n) ||
		(n.skill != nil && n.skill.IsMovementDisabled()) || n.script.box // Mobius Chest.isMovementDisabled
}

// NpcRunning reports whether the NPC runs (Mobius isRunning). Loop-only.
func (n *Npc) Running() bool { return n.Stats.Motion().Running }

// SetNpcRunning makes n run or walk (Mobius setRunning and setWalking): on a
// change the players around get ChangeMoveType, when the NPC can run, and
// its info again. Loop-only.
func (s *Server) SetNpcRunning(n *Npc, run bool) {
	m := n.Stats.Motion()
	if m.Running == run {
		return
	}
	m.Running = run
	n.Stats.SetMotion(m)
	if !n.InWorld() {
		return
	}
	n.statView()
	if n.cache.runSpeed != 0 {
		s.world.Broadcast(n, serverpackets.NewChangeMoveTypePacket(serverpackets.ChangeMoveType{
			ObjectID: int32(n.ID()),
			Running:  run,
		}), false)
	}
	n.FlushUpdate(world.UpdateCharInfo)
}

// npcMoveToLocation starts n walking to (x, y, z), or to offset short of it
// (Mobius Creature.moveToLocation). It sends nothing about the walk and
// reports whether a walk started. When n already stands within the offset,
// the AI hears that it arrived.
//
// While the NPC's neighbourhood is active, geodata, doors and fences clip
// the walk or route it around (pathing.go).
func (s *Server) npcMoveToLocation(n *Npc, x, y, z, offset int32) bool {
	speed := n.Stats.MoveSpeed()
	if !(speed > 0) || npcMovementDisabled(n) {
		return false
	}
	cur := n.Position()
	dx, dy, dz := walkDelta(cur, x, y, z)
	distance := math.Sqrt(dx*dx + dy*dy)
	flying := npcFlying(n)
	vertical := flying && distance == 0 && dz != 0
	if vertical {
		distance = math.Abs(dz)
	}

	var sin, cos float64
	if offset > 0 || distance < 1 {
		// Java: offset -= Math.abs(dz) on an int.
		off := int32(float64(offset) - math.Abs(dz))
		if off < 5 {
			off = 5
		}
		if distance < 1 || distance-float64(off) <= 0 {
			s.npcArrived(n)
			return false
		}
		sin, cos = dy/distance, dx/distance
		distance -= float64(off - 5)
		x = cur.X + int32(distance*cos)
		y = cur.Y + int32(distance*sin)
	} else {
		sin, cos = dy/distance, dx/distance
	}

	var route *geoRoute
	if s.world.NeighborsActive(n) && !flying {
		// pathing.go: clipped by geodata, doors and fences, or a route.
		g := geoMove{cur: cur, x: x, y: y, z: z, distance: distance, sin: sin, cos: cos, vertical: vertical}
		if !s.npcGeoMove(n, &g) {
			return false
		}
		x, y, z, distance, sin, cos, route = g.x, g.y, g.z, g.distance, g.sin, g.cos, g.route
	}
	if flying && !vertical {
		distance = math.Sqrt(distance*distance + dz*dz)
	}

	ticksToMove := int64(walkTicksPerSecond * distance / speed)
	heading := n.Position().Heading
	if !vertical {
		heading = walkHeading(cos, sin)
		n.SetHeading(heading)
	}
	now := s.Now()
	walk := &MoveState{
		Origin:      n.Position(),
		Destination: world.Position{X: x, Y: y, Z: z, Heading: heading},
		Start:       now,
		Speed:       speed,
		Running:     n.Running(),
	}
	n.Move = walk
	n.ai.walk = walkTrack{state: walk, started: gameTick(now)}
	n.ai.walk.geo.route = route
	if !n.ai.moving {
		n.ai.moving = true
		s.ai.moving = append(s.ai.moving, n)
	}
	if ticksToMove*walkTickMillis > npcArrivedRevalidateWalk {
		s.loop.After(npcArrivedRevalidate, func() { s.notifyArrivedRevalidate(n) })
	}
	return true
}

// npcMoveTick is the NPC movement tick (Mobius MovementTaskManager): every
// walking NPC advances by the game ticks since its last update.
func (s *Server) npcMoveTick(now time.Time) {
	f := &s.ai
	if len(f.moving) == 0 {
		return
	}
	tick := gameTick(now)
	batch := append(f.moveBatch[:0], f.moving...)
	for _, n := range batch {
		s.advanceNpc(n, tick)
	}
	clear(batch)
	f.moveBatch = batch[:0]
	f.moving = slices.DeleteFunc(f.moving, func(n *Npc) bool {
		if n.Move != nil && n.Move == n.ai.walk.state && n.InWorld() {
			return false
		}
		n.ai.moving = false
		return true
	})
}

// advanceNpc moves n along its walk (Mobius updatePosition) and ends the
// walk when it arrives: the players around get the finished walk's
// MoveToLocation, then the AI hears of the arrival. The NPC's own collision
// radius counts against the distance left: monsters keep no creature target
// while they chase, so Mobius never uses the target's.
func (s *Server) advanceNpc(n *Npc, tick int64) {
	walk := n.Move
	if walk == nil || walk != n.ai.walk.state {
		return
	}
	if !n.InWorld() {
		n.Move = nil
		return
	}
	speed := n.Stats.MoveSpeed()
	walk.Speed = speed
	cur := n.Position()
	flying := npcFlying(n)
	next, arrived, ok := n.ai.walk.step(cur, walk.Destination, speed, float64(npcCollision(n)), tick, flying)
	if !ok {
		return
	}
	if !arrived && !flying && n.class != nil && n.class.attackable && spawnAbs(next.Z-cur.Z) > npcMaxZStep {
		if sp := n.spawn; sp != nil {
			s.TeleportNpc(n, sp.lastPoint)
			n.clearAttackBy()
			n.ClearAggroList()
			s.npcSetIdle(n)
		}
		n.Move = nil
		return
	}
	if next != cur {
		if err := s.MoveObject(n, next); err != nil {
			s.log.Debug("game: an NPC walk left the world", "npc", n.Template.ID, "objectId", n.ID(), "pos", next, "err", err)
			n.Move = nil
			return
		}
		if n.Move != walk {
			return // the move's visibility changes stopped or replaced the walk
		}
	}
	if arrived && !s.nextNpcLeg(n) { // pathing.go: a route walks on
		s.broadcastNpcMove(n, true)
		n.Move = nil
		s.npcArrived(n)
	}
}

// broadcastNpcMove sends n's walk to the players around (Mobius
// broadcastMoveToLocation for an NPC): MoveToLocation while the NPC's
// neighbourhood is active, always when forced or for a walk that has not
// moved yet, otherwise once a second at most. Monsters keep no creature
// target, so Mobius's MoveToPawn branch does not happen.
func (s *Server) broadcastNpcMove(n *Npc, force bool) {
	walk := n.Move
	if walk == nil {
		return
	}
	tick := gameTick(s.Now())
	if !force && n.ai.walk.begun && tick-n.ai.walk.broadcast < walkBroadcastTicks {
		return
	}
	n.ai.walk.broadcast = tick
	if s.world.NeighborsActive(n) {
		s.world.Broadcast(n, moveToLocationPacket(n, walk.Destination), false)
	}
}

// stopNpc ends n's walk where it stands, or at *at, and sends StopMove to
// the players around even when it stood still (Mobius Creature.stopMove).
func (s *Server) stopNpc(n *Npc, at *world.Position) {
	n.Move = nil
	if at != nil && n.InWorld() {
		if err := s.MoveObject(n, *at); err != nil {
			s.log.Debug("game: cannot stop an NPC there", "npc", n.Template.ID, "pos", *at, "err", err)
		}
		n.SetHeading(at.Heading)
	}
	if n.InWorld() {
		s.world.Broadcast(n, npcStopMovePacket(n), false)
	}
}

// npcStopMovePacket is the StopMove of n at its position.
func npcStopMovePacket(n *Npc) []byte {
	pos := n.Position()
	return serverpackets.NewStopMovePacket(serverpackets.StopMove{
		ObjectID: int32(n.ID()),
		X:        pos.X, Y: pos.Y, Z: pos.Z,
		Heading: pos.Heading,
	})
}

// NpcMoveTo walks n to pos and shows it to the players around (Mobius
// AbstractAI.moveTo). It reports whether a walk started. Loop-only.
func (s *Server) NpcMoveTo(n *Npc, pos world.Position) bool {
	if npcMovementDisabled(n) {
		return false
	}
	n.ai.pawnOffset = 0
	started := s.npcMoveToLocation(n, pos.X, pos.Y, pos.Z, 0)
	s.broadcastNpcMove(n, false)
	return started
}

// NpcMoveToPawn walks n up to offset (at least 10) from pawn and shows it
// with MoveToPawn while n's region is active (Mobius AbstractAI.moveToPawn).
// A walk to the same pawn with the same offset is not started again within
// a second: chasing monsters follow their target once a second. Nothing
// happens while n swings. Loop-only.
func (s *Server) NpcMoveToPawn(n *Npc, pawn world.Object, offset int32) {
	if npcMovementDisabled(n) || s.AttackingNow(n) {
		return
	}
	offset = max(offset, npcPawnMinOffset)
	now := gameTick(s.Now())
	ai := &n.ai
	if n.Move != nil && ai.target == pawn {
		if ai.pawnOffset == offset && now < ai.pawnTimeout {
			return
		}
		// A route is kept 2 s at least (pathing.go).
		if ai.pawnOffset != offset && s.onRoute(n) && now < ai.pawnTimeout+npcPawnRepathTicks {
			return
		}
	}
	ai.pawnOffset = offset
	ai.target = pawn
	ai.pawnTimeout = now + npcPawnRepathTicks
	if pawn == nil {
		return
	}
	to := pawn.Position()
	s.npcMoveToLocation(n, to.X, to.Y, to.Z, offset)
	if n.Move == nil {
		return // in reach already, or geodata left nothing to walk (Mobius: !isMoving)
	}
	if _, ok := pawn.(Creature); ok {
		if s.onRoute(n) {
			// On a route the client gets the leg, not the pawn.
			s.broadcastNpcMove(n, false)
			ai.pawnOffset = 0
		} else if s.world.RegionActive(n) {
			s.world.Broadcast(n, npcMoveToPawnPacket(n, pawn, offset), false)
		}
	} else {
		s.broadcastNpcMove(n, false)
	}
}

// NpcStopMove stops n where it stands if it walks (Mobius
// clientStopMoving(null)). Loop-only.
func (s *Server) NpcStopMove(n *Npc) { s.npcClientStopMoving(n, nil) }

// npcClientStopMoving is Mobius AbstractAI.clientStopMoving.
func (s *Server) npcClientStopMoving(n *Npc, at *world.Position) {
	if n.Move != nil {
		s.stopNpc(n, at)
	}
	n.ai.pawnOffset = 0
}

// npcClientStoppedMoving is Mobius AbstractAI.clientStoppedMoving: a walk
// up to an object ends with StopMove.
func (s *Server) npcClientStoppedMoving(n *Npc) {
	if n.ai.pawnOffset > 0 {
		n.ai.pawnOffset = 0
		s.world.Broadcast(n, npcStopMovePacket(n), false)
	}
}

// TeleportNpc puts n at pos at once (Mobius Creature.teleToLocation for an
// NPC): the walk stops (StopMove), the attack ends, the AI turns ACTIVE,
// the players around the old place lose it and those around the new one
// get its info, and the NPC spawns there again for the AI (its post-spawn
// counters start over). Without geodata the height is pos's; with it, the
// ground's; Mobius adds 5. Loop-only.
func (s *Server) TeleportNpc(n *Npc, pos world.Position) {
	if !n.InWorld() {
		return
	}
	z := pos.Z
	if !npcFlying(n) {
		z = s.geo.Height(pos.X, pos.Y, pos.Z)
	}
	s.stopNpc(n, nil)
	s.AbortAttack(n)
	n.ai.target = nil
	s.npcSetActive(n)
	s.RemoveObject(n)
	heading := n.Position().Heading
	if pos.Heading != 0 {
		heading = pos.Heading
	}
	if err := n.SetPosition(world.Position{X: pos.X, Y: pos.Y, Z: z + 5, Heading: heading}); err != nil {
		s.log.Warn("game: cannot teleport an NPC", "npc", n.Template.ID, "pos", pos, "err", err)
	}
	if err := s.AddObject(n); err != nil {
		s.log.Warn("game: cannot teleport an NPC", "npc", n.Template.ID, "pos", pos, "err", err)
		return
	}
	s.npcSpawned(n, nil, nil)
	s.npcTeleported(n)           // quest_world_ai_events.go: on_teleport
	s.minionsMasterTeleported(n) // minion.go: Mobius Monster.onTeleported
}
