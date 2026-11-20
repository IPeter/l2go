package game

import (
	"errors"
	"fmt"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
)

// The geodata GM commands (Plan 13, Phase 10.2): Mobius AdminGeodata's
// //geo_pos, //geo_spawn_pos, //geo_can_move and //geo_can_see, and
// AdminPathNode's //path_find. The geodata editor commands (geosave, the
// NSWE switches, geoedit) are left out: decision A2 keeps geodata
// read-only.
func init() { registerFeature("geo", setupGeoGM) }

// geoMsgInvalidTarget is "Invalid target." (Mobius INVALID_TARGET).
const geoMsgInvalidTarget serverpackets.SystemMessageID = 109

// setupGeoGM registers the geodata GM commands.
func setupGeoGM(s *Server) error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("geo_pos", admin, s.gmGeoPos),
		s.RegisterGMCommand("geo_spawn_pos", admin, s.gmGeoPos),
		s.RegisterGMCommand("geo_can_move", admin, s.gmGeoCanMove),
		s.RegisterGMCommand("geo_can_see", admin, s.gmGeoCanSee),
		s.RegisterGMCommand("path_find", admin, s.gmPathFind),
	)
}

// gmGeoPos is //geo_pos and //geo_spawn_pos: the GM's position, its geodata
// cell and the ground height there. Mobius answers both with getHeight.
func (s *Server) gmGeoPos(p *Player, _ []string) {
	pos := p.Position()
	gx, gy := geoCellX(pos.X), geoCellY(pos.Y)
	if !s.geo.HasGeo(pos.X, pos.Y) {
		gmSysMessage(p, "There is no geodata at this position.")
		return
	}
	gmSysMessage(p, fmt.Sprintf("WorldX: %d, WorldY: %d, WorldZ: %d, GeoX: %d, GeoY: %d, GeoZ: %d",
		pos.X, pos.Y, pos.Z, gx, gy, s.geo.Height(pos.X, pos.Y, pos.Z)))
}

// gmGeoCanMove is //geo_can_move: whether the GM could walk straight to its
// target. Mobius asks canSeeTarget here, not canMoveToTarget; so do we.
func (s *Server) gmGeoCanMove(p *Player, _ []string) {
	if p.Target == nil {
		p.SendSystemMessage(geoMsgInvalidTarget)
		return
	}
	if s.canSee(p, p.Target) {
		gmSysMessage(p, "Can move beeline.")
	} else {
		gmSysMessage(p, "Can not move beeline!")
	}
}

// gmGeoCanSee is //geo_can_see: whether the GM sees its target.
func (s *Server) gmGeoCanSee(p *Player, _ []string) {
	if p.Target == nil {
		p.SendSystemMessage(geoMsgInvalidTarget)
		return
	}
	if s.canSee(p, p.Target) {
		gmSysMessage(p, "Can see target.")
	} else {
		p.SendSystemMessage(sysMsgCannotSeeTarget)
	}
}

// gmPathFind is //path_find: the waypoints of a player's path from the GM
// to its target, one line each.
func (s *Server) gmPathFind(p *Player, _ []string) {
	if !s.geo.Pathfinding() {
		gmSysMessage(p, "PathFinding is disabled.")
		return
	}
	t := p.Target
	if t == nil {
		gmSysMessage(p, "No Target!")
		return
	}
	// Mobius casts both heights to short.
	from, to := geoPoint(p.Position()), geoPoint(t.Position())
	from.Z, to.Z = int32(int16(from.Z)), int32(int16(to.Z)) //nolint:gosec // Java (short)
	path := s.geo.FindPath(from, to, true)
	if path == nil {
		gmSysMessage(p, "No Route!")
		return
	}
	for _, w := range path {
		gmSysMessage(p, fmt.Sprintf("x:%d y:%d z:%d", w.X, w.Y, w.Z))
	}
}
