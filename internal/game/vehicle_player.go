package game

import (
	"errors"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// The vehicle requests (Mobius RequestGetOnVehicle, RequestGetOffVehicle,
// RequestMoveToLocationInVehicle, CannotMoveAnymoreInVehicle). A passenger
// stands where its boat is in the world; its client keeps the deck
// position, which the server only records and shows to the others.

// setupVehicleRequests registers the vehicle requests.
func (s *Server) setupVehicleRequests() error {
	return errors.Join(
		s.Register(clientpackets.OpRequestGetOnVehicle, s.handleGetOnVehicle),
		s.Register(clientpackets.OpRequestGetOffVehicle, s.handleGetOffVehicle),
		s.Register(clientpackets.OpRequestMoveToLocationInVehicle, s.handleMoveToLocationInVehicle),
		s.Register(clientpackets.OpCannotMoveAnymoreInVehicle, s.handleCannotMoveAnymoreInVehicle),
	)
}

// deckPos is a deck position of a request.
func deckPos(x, y, z int32) world.Position { return world.Position{X: x, Y: y, Z: z} }

// stopMoveInVehiclePacket is Mobius StopMoveInVehicle(player, boatId): the
// deck position and the player's heading.
func stopMoveInVehiclePacket(p *Player, boatID int32) []byte {
	v := &p.vehicle
	return serverpackets.NewStopMoveInVehiclePacket(serverpackets.InVehicleLocation{
		ObjectID: p.ObjectID(), BoatID: boatID,
		X: v.pos.X, Y: v.pos.Y, Z: v.pos.Z,
		Heading: p.Position().Heading,
	})
}

// handleGetOnVehicle is RequestGetOnVehicle: a player within 1,000 of a
// boat that stands boards it at the deck position (a passenger may only
// name its own boat). Everyone around sees GetOnVehicle; the player stands
// where the boat is, inside the peace zone.
func (s *Server) handleGetOnVehicle(p *Player, body []byte) {
	req, err := clientpackets.ParseVehicleRequest(body)
	if err != nil {
		s.log.Debug("game: bad RequestGetOnVehicle", append(p.logArgs(), "err", err)...)
		return
	}
	if !p.InWorld() {
		return
	}
	b := p.vehicle.boat
	if b != nil {
		if b.ObjectID() != req.BoatID {
			p.SendActionFailed()
			return
		}
	} else {
		var ok bool
		b, ok = s.Boat(world.ObjectID(req.BoatID)) //nolint:gosec // object IDs are positive
		if !ok || b.Moving() || !within3D(p.Position(), b.Position(), boatRange) {
			p.SendActionFailed()
			return
		}
	}
	v := &p.vehicle
	v.pos, v.hasPos = deckPos(req.X, req.Y, req.Z), true
	s.setVehicle(p, b)
	s.world.Broadcast(p, serverpackets.NewGetOnVehiclePacket(serverpackets.VehiclePassenger{
		ObjectID: p.ObjectID(), BoatID: b.ObjectID(), X: req.X, Y: req.Y, Z: req.Z,
	}), true)
	to := b.Position()
	to.Heading = p.Position().Heading
	if !s.placePlayer(p, to) {
		return
	}
	if !v.peace {
		v.peace = true
		p.Zones.Enter(ZonePeace)
	}
	s.revalidateZones(p, true)
}

// handleGetOffVehicle is RequestGetOffVehicle: a passenger of a boat that
// stands leaves it for a world point within 1,000. The others see
// StopMoveInVehicle and GetOffVehicle; the player leaves the peace zone.
func (s *Server) handleGetOffVehicle(p *Player, body []byte) {
	req, err := clientpackets.ParseVehicleRequest(body)
	if err != nil {
		s.log.Debug("game: bad RequestGetOffVehicle", append(p.logArgs(), "err", err)...)
		return
	}
	b := p.vehicle.boat
	to := world.Position{X: req.X, Y: req.Y, Z: req.Z, Heading: p.Position().Heading}
	if b == nil || b.ObjectID() != req.BoatID || b.Moving() || !p.InWorld() || !within3D(p.Position(), to, boatRange) {
		p.SendActionFailed()
		return
	}
	s.world.Broadcast(p, stopMoveInVehiclePacket(p, req.BoatID), true)
	s.setVehicle(p, nil)
	p.vehicle.hasPos = false
	p.SendActionFailed()
	s.world.Broadcast(p, serverpackets.NewGetOffVehiclePacket(serverpackets.VehiclePassenger{
		ObjectID: p.ObjectID(), BoatID: req.BoatID, X: req.X, Y: req.Y, Z: req.Z,
	}), true)
	if s.placePlayer(p, to) {
		s.revalidateZones(p, true)
	}
}

// handleMoveToLocationInVehicle is RequestMoveToLocationInVehicle: a player
// walks on a deck. Standing still answers StopMoveInVehicle; a sitting or
// immobile player gets ActionFailed. As in Mobius, the request also puts a
// player on the boat it names; an unknown boat is refused.
func (s *Server) handleMoveToLocationInVehicle(p *Player, body []byte) {
	req, err := clientpackets.ParseMoveToLocationInVehicle(body)
	if err != nil {
		s.log.Debug("game: bad RequestMoveToLocationInVehicle", append(p.logArgs(), "err", err)...)
		return
	}
	if req.TargetX == req.OriginX && req.TargetY == req.OriginY && req.TargetZ == req.OriginZ {
		p.Send(stopMoveInVehiclePacket(p, req.BoatID))
		return
	}
	if p.Sitting || !canWalk(p) || p.IsDead() {
		p.SendActionFailed()
		return
	}
	b := p.vehicle.boat
	if b == nil || b.ObjectID() != req.BoatID {
		nb, ok := s.Boat(world.ObjectID(req.BoatID)) //nolint:gosec // object IDs are positive
		if !ok {
			p.SendActionFailed()
			return
		}
		if b != nil {
			s.setVehicle(p, nil)
		}
		s.setVehicle(p, nb)
		b = nb
	}
	v := &p.vehicle
	v.pos, v.hasPos = deckPos(req.TargetX, req.TargetY, req.TargetZ), true
	s.world.Broadcast(p, serverpackets.NewMoveToLocationInVehiclePacket(serverpackets.MoveToLocationInVehicle{
		ObjectID: p.ObjectID(), BoatID: b.ObjectID(),
		DestX: req.TargetX, DestY: req.TargetY, DestZ: req.TargetZ,
		X: req.OriginX, Y: req.OriginY, Z: req.OriginZ,
	}), true)
}

// handleCannotMoveAnymoreInVehicle is CannotMoveAnymoreInVehicle: a
// passenger stopped on its deck; the others see StopMoveInVehicle there.
func (s *Server) handleCannotMoveAnymoreInVehicle(p *Player, body []byte) {
	req, err := clientpackets.ParseCannotMoveAnymoreInVehicle(body)
	if err != nil {
		s.log.Debug("game: bad CannotMoveAnymoreInVehicle", append(p.logArgs(), "err", err)...)
		return
	}
	b := p.vehicle.boat
	if b == nil || b.ObjectID() != req.BoatID {
		return
	}
	v := &p.vehicle
	v.pos, v.hasPos = deckPos(req.X, req.Y, req.Z), true
	p.SetHeading(req.Heading)
	s.world.Broadcast(p, stopMoveInVehiclePacket(p, req.BoatID), true)
}
