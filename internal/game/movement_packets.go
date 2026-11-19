package game

import (
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// Limits of Mobius MoveToLocation and ValidatePosition.
const (
	// walkMaxRequest is the farthest target a MoveToLocation may name (2D).
	walkMaxRequest = 9900
	// walkKeyboardSync: with the arrow keys the client's origin replaces the
	// server's position when it is closer than this (2D).
	walkKeyboardSync = 1000

	// validateMaxZ: a reported height beyond ±20,000 puts the player back
	// at its last server position.
	validateMaxZ = 20000
	// validateNear and validateFar bound, squared, the 2D differences that
	// get a ValidateLocation correction (500 to 600 units); farther ones
	// are left alone, as they "mess observation".
	validateNear = 500 * 500
	validateFar  = 600 * 600
	// validateMaxDZ is the height difference that gets a correction, or the
	// client's height when it is below validateTakeDZ and close to the last
	// reported one (validateClientDZ).
	validateMaxDZ    = 200
	validateTakeDZ   = 1500
	validateClientDZ = 800
	// validateWater is, squared, the 2D difference that gets a correction
	// in water (300 units).
	validateWater = 300 * 300
)

// handleMoveToLocation handles the client MoveToLocation (0x01), a click or
// an arrow key (Mobius MoveToLocation): p walks there and p and the players
// around see it. ActionFailed refuses it for a teleporting or sitting
// player, a target more than 9,900 units away and a target geodata blocks.
// A target equal to the client's origin stops p where it is.
func (s *Server) handleMoveToLocation(p *Player, body []byte) {
	req, err := clientpackets.ParseMoveToLocation(body)
	if err != nil {
		s.log.Debug("game: bad MoveToLocation", append(p.logArgs(), "err", err)...)
		return
	}
	if !p.InWorld() {
		// Teleporting (Mobius isMovementDisabled).
		p.SendActionFailed()
		return
	}
	m := s.motionOf(p)
	if req.TargetX == req.OriginX && req.TargetY == req.OriginY && req.TargetZ == req.OriginZ {
		here := p.Position()
		s.stopWalk(p, m, &here)
		p.SendActionFailed()
		return
	}
	if s.walkBlocked(req.TargetX, req.TargetY, req.TargetZ) {
		p.SendActionFailed()
		return
	}
	if req.Mode == clientpackets.MoveModeMouse {
		m.keyboard = false
	} else {
		// The arrow keys (Mobius KeyboardMovement, on by default): the
		// client's origin counts when it is close to the server's position.
		cur := p.Position()
		dx, dy := int64(req.OriginX)-int64(cur.X), int64(req.OriginY)-int64(cur.Y)
		if dx*dx+dy*dy < walkKeyboardSync*walkKeyboardSync {
			s.syncWalk(p, m, req.OriginX, req.OriginY, req.OriginZ)
		}
		m.keyboard = true
		m.lastServer = lastServerPosition(p.Position())
	}
	cur := p.Position()
	dx, dy := float64(req.TargetX)-float64(cur.X), float64(req.TargetY)-float64(cur.Y)
	if dx*dx+dy*dy > walkMaxRequest*walkMaxRequest {
		p.SendActionFailed()
		return
	}
	if p.Sitting {
		// Mobius's AI rests while the player sits.
		p.SendActionFailed()
		return
	}
	if s.startWalk(p, m, world.Position{X: req.TargetX, Y: req.TargetY, Z: req.TargetZ}, 0) {
		s.broadcastWalk(p, m, false)
	}
}

// walkBlocked reports whether geodata blocks a target cell completely
// (Mobius isCompletelyBlocked). A geo.Engine answers it with an optional
// CompletelyBlocked method; without geodata nothing is blocked.
func (s *Server) walkBlocked(x, y, z int32) bool {
	b, ok := s.geo.(interface{ CompletelyBlocked(x, y, z int32) bool })
	return ok && s.geo.HasGeo(x, y) && b.CompletelyBlocked(x, y, z)
}

// handleValidatePosition handles ValidatePosition (0x48), where the client
// shows p (Mobius ValidatePosition), which mostly trusts the client:
//   - Nothing happens while p teleports, or for a report of (0, 0).
//   - A height beyond ±20,000 teleports p back to its last server position.
//   - In water p takes the client's height; a 2D difference over 300 gets
//     a ValidateLocation correction.
//   - On land a 2D difference of 500 to 600 units, or a height difference
//     over 200, gets ValidateLocation, unless the height differs by less
//     than 1,500 and the client reported a height within 800 of it last
//     time: then p takes the client's height.
//   - A 3D difference larger than the move speed makes p take the client's
//     position; its height only when the client is not lower (without
//     geodata the server keeps its own height then).
//   - With the arrow keys p takes the client's position anyway, and p and
//     the players around get ValidateLocation.
//
// While p walks with the mouse, a new X and Y last only until the next
// tick, which goes on from the walk's own position; the height stays, so Z
// follows the client.
func (s *Server) handleValidatePosition(p *Player, body []byte) {
	req, err := clientpackets.ParseValidatePosition(body)
	if err != nil {
		s.log.Debug("game: bad ValidatePosition", append(p.logArgs(), "err", err)...)
		return
	}
	if p.Teleporting() || !p.InWorld() {
		return
	}
	known := p.Position() // Mobius realX, realY, realZ
	if req.X == 0 && req.Y == 0 && known.X != 0 {
		return
	}
	m := s.motionOf(p)
	if req.Z < -validateMaxZ || req.Z > validateMaxZ {
		if err := s.Teleport(p, m.lastServer); err != nil {
			s.log.Warn("game: cannot put a player back at its last server position", append(p.logArgs(), "pos", m.lastServer, "err", err)...)
		}
		return
	}
	dx := int64(req.X) - int64(known.X)
	dy := int64(req.Y) - int64(known.Y)
	dz := int64(req.Z) - int64(known.Z)
	diffSq := dx*dx + dy*dy
	knownZ := known.Z
	switch {
	case p.Zones.Inside(ZoneWater):
		s.placePlayer(p, world.Position{X: known.X, Y: known.Y, Z: req.Z, Heading: known.Heading})
		if diffSq > validateWater {
			p.Send(validateLocation(p))
		}
	case diffSq < validateFar && (diffSq > validateNear || validateAbs(dz) > validateMaxDZ):
		if validateAbs(dz) > validateMaxDZ && validateAbs(dz) < validateTakeDZ &&
			validateAbs(int64(req.Z)-int64(m.clientZ)) < validateClientDZ {
			s.placePlayer(p, world.Position{X: known.X, Y: known.Y, Z: req.Z, Heading: known.Heading})
			knownZ = req.Z
		} else {
			p.Send(validateLocation(p))
		}
	}

	// Out of sync: Mobius takes the client's position.
	client := world.Position{X: req.X, Y: req.Y, Z: req.Z}
	if cur := p.Position(); cur.Distance3D(client) > p.stats.MoveSpeed() {
		z := req.Z
		if knownZ > req.Z {
			z = s.geo.Height(req.X, req.Y, knownZ)
		}
		s.placePlayer(p, world.Position{X: req.X, Y: req.Y, Z: z, Heading: cur.Heading})
	}
	if p.state != stateInGame {
		return // a hook made the player leave
	}
	m.clientZ = req.Z
	// There are no doors yet, so nothing stands between the two positions.
	m.lastServer = world.Position{X: known.X, Y: known.Y, Z: knownZ}
	if m.keyboard {
		s.syncWalk(p, m, req.X, req.Y, req.Z)
		s.world.Broadcast(p, validateLocation(p), true)
	}
}

func validateAbs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// handleCannotMoveAnymore handles CannotMoveAnymore (0x36): the client
// stopped p against an obstacle (Mobius notifyActionArrivedBlocked). A
// walking p stops where the client says, and p and the players around get
// StopMove; a p that stands is left alone.
func (s *Server) handleCannotMoveAnymore(p *Player, body []byte) {
	req, err := clientpackets.ParseCannotMoveAnymore(body)
	if err != nil {
		s.log.Debug("game: bad CannotMoveAnymore", append(p.logArgs(), "err", err)...)
		return
	}
	if p.Move == nil {
		return
	}
	s.stopWalk(p, s.motionOf(p), &world.Position{X: req.X, Y: req.Y, Z: req.Z, Heading: req.Heading})
}

// handleStartRotating handles StartRotating (0x4A), turning with the
// keyboard: p and the players around get StartRotation (Mobius
// StartRotating, speed 0).
func (s *Server) handleStartRotating(p *Player, body []byte) {
	req, err := clientpackets.ParseStartRotating(body)
	if err != nil {
		s.log.Debug("game: bad StartRotating", append(p.logArgs(), "err", err)...)
		return
	}
	if !p.InWorld() {
		return
	}
	s.world.Broadcast(p, serverpackets.NewStartRotationPacket(serverpackets.StartRotation{
		ObjectID: p.ObjectID(),
		Degree:   req.Degree,
		Side:     req.Side,
	}), true)
}

// handleFinishRotating handles FinishRotating (0x4B): p faces the final
// heading, and p and the players around get StopRotation (Mobius
// FinishRotating, speed 0).
func (s *Server) handleFinishRotating(p *Player, body []byte) {
	req, err := clientpackets.ParseFinishRotating(body)
	if err != nil {
		s.log.Debug("game: bad FinishRotating", append(p.logArgs(), "err", err)...)
		return
	}
	if !p.InWorld() {
		return
	}
	p.SetHeading(req.Degree)
	s.world.Broadcast(p, serverpackets.NewStopRotationPacket(serverpackets.StopRotation{
		ObjectID: p.ObjectID(),
		Degree:   req.Degree,
	}), true)
}
