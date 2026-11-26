package game

import (
	"math"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/world"
)

// Restarting after death: Mobius RequestRestartPoint, Creature.doRevive,
// Player.doRevive and restoreExp, MapRegionData.getTeleToLocation for the
// town, and the random offset of teleToLocation(loc, true).

// maxOffsetOnTeleport is Mobius MaxOffsetOnTeleport (Player.ini, 50): the
// random offset of a restart in town.
const maxOffsetOnTeleport = 50

// handleRestartPoint handles RequestRestartPoint (0x6d) from a dead player
// (Mobius RequestRestartPoint). The village (0, and every unknown value)
// revives the player with RespawnRestoreHP/CP/MP and teleports it to a
// restart point of its town (restartLocation), 50 units around it. Fixed
// (4) revives a GM where it lies and gives back the exp it lost. The clan
// hall, castle and siege headquarters need clans, which do not exist yet;
// the agathion and jail points do nothing, as in Mobius for a player who is
// not jailed.
func (s *Server) handleRestartPoint(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestRestartPoint(body)
	if err != nil {
		s.log.Debug("game: bad RequestRestartPoint", append(p.logArgs(), "err", err)...)
		return
	}
	if !p.dead {
		return
	}
	if s.residenceRestart(p, req.Point) { // siege_zone.go: castle and headquarters
		return
	}
	switch req.Point {
	case clientpackets.RestartPointClanHall:
		s.hallRestart(p) // clanhall.go
	case clientpackets.RestartPointCastle:
		// Mobius returns quietly for a player without a clan.
	case clientpackets.RestartPointSiegeHQ:
		s.log.Warn("game: RequestRestartPoint to a siege headquarters without one", p.logArgs()...)
	case clientpackets.RestartPointFixed:
		if !p.IsGM() {
			s.log.Warn("game: RequestRestartPoint fixed without the right", p.logArgs()...)
			return
		}
		s.RevivePlayer(p)
		s.RestoreExp(p, 100)
	case clientpackets.RestartPointAgathion, clientpackets.RestartPointJail:
	default:
		loc, ok := s.restartLocation(p)
		if !ok {
			s.log.Error("game: no restart point for a dead player", append(p.logArgs(), "pos", p.Position())...)
			return
		}
		s.RevivePlayer(p)
		s.teleportAround(p, loc, maxOffsetOnTeleport)
	}
}

// RevivePlayer brings a dead player back (Mobius Creature.doRevive and
// Player.doRevive): CP, HP and MP go up to RespawnRestoreCP/HP/MP percent of
// their maximum when below it (65 % HP by default), Revive goes to p and the
// players around, and p gets EtcStatusUpdate. A living player is left
// alone. Loop-only.
func (s *Server) RevivePlayer(p *Player) {
	if !p.dead {
		return
	}
	p.Revive()
	c := s.combatSettings()
	v := p.stats.View()
	if pct := c.RespawnRestoreCP / 100; pct > 0 && p.CurCP < float64(v.MaxCP)*pct {
		p.SetCurrentCP(float64(v.MaxCP) * pct)
	}
	if pct := c.RespawnRestoreHP / 100; pct > 0 && p.CurHP < float64(v.MaxHP)*pct {
		p.SetCurrentHP(float64(v.MaxHP) * pct)
	}
	if pct := c.RespawnRestoreMP / 100; pct > 0 && p.CurMP < float64(v.MaxMP)*pct {
		p.SetCurrentMP(float64(v.MaxMP) * pct)
	}
	if p.InWorld() {
		s.world.Broadcast(p, serverpackets.NewRevivePacket(p.ObjectID()), true)
	} else {
		p.Send(serverpackets.NewRevivePacket(p.ObjectID()))
	}
	p.Send(etcStatusPacket(p))
	s.mountRevived(p) // mount.go: a rider's mount eats again
}

// RestoreExp gives p back percent of the exp its last death took (Mobius
// Player.restoreExp, for resurrections): Math.round((exp before death −
// exp) × percent / 100) through AddExp, then UserInfo. Nothing happens
// without a recorded loss. Loop-only.
func (s *Server) RestoreExp(p *Player, percent float64) {
	if p.char.ExpBeforeDeath <= 0 {
		return
	}
	s.AddExp(p, mathRound(float64(p.char.ExpBeforeDeath-p.char.Exp)*percent/100))
	p.char.ExpBeforeDeath = 0
	p.UpdateUserInfo()
}

// restartLocation is Mobius MapRegionData.getTeleToLocation(player, TOWN)
// without clans, sieges and instances: a player with karma goes to a
// chaotic point (getNearestKarmaRespawn), others to a normal one
// (getNearestTownRespawn). Inside a respawn zone the zone names the region
// for the player's race, following banned races; elsewhere the map region
// of the position, or the region its bans send the race to; without either
// the default region. The point is random with RandomRespawnInTown, else
// the first.
func (s *Server) restartLocation(p *Player) (data.Location, bool) {
	w := s.worldData
	if w == nil {
		return data.Location{}, false
	}
	if loc, ok := s.dawnSiegeRespawn(p); ok { // sevensigns_castle.go: the Seal of Strife
		return loc, true
	}
	race := data.Race(p.Race()) //nolint:gosec // player races are 0-4
	pos := p.Position()
	var region *data.MapRegion
	if z := s.zoneAt(pos, "RespawnZone"); z != nil {
		region = restartRegion(w, race, z.RacePoints[race])
	} else if r, ok := w.MapRegionAt(int(pos.X), int(pos.Y)); ok {
		region = r
		if alt, banned := r.BannedRaces[race]; banned {
			if a, ok := w.MapRegion(alt); ok {
				region = a
			}
		}
	} else {
		region = w.DefaultRegion()
	}
	if region == nil {
		return data.Location{}, false
	}
	points := region.RespawnPoints
	if p.Karma() > 0 && len(region.ChaoticRespawnPoints) > 0 {
		points = region.ChaoticRespawnPoints
	}
	if len(points) == 0 {
		return data.Location{}, false
	}
	if s.combatSettings().RandomRespawnInTown {
		return points[s.rnd.Get(len(points))], true
	}
	return points[0], true
}

// restartRegion is the region of Mobius getRestartRegionSpawnLoc (and its
// chaotic twin): the named region, or the default one; a region that bans
// the race leads to the region it names for the race. A loop of bans falls
// back to the default region, as Mobius does after its stack overflows.
func restartRegion(w *data.World, race data.Race, point string) *data.MapRegion {
	for range 8 {
		r := w.RespawnRegion(point)
		if r == nil {
			return nil
		}
		next, banned := r.BannedRaces[race]
		if !banned {
			return r
		}
		point = next
	}
	return w.DefaultRegion()
}

// teleportAround teleports p near loc as Mobius teleToLocation(loc, offset)
// does: x and y move by Rnd.get(-offset, offset) each, drawn again (up to
// 100 times) while the point is closer to loc than twice p's collision
// radius (at most offset) or cannot be seen or reached from it; then the
// geodata's valid location between them. The heading stays.
func (s *Server) teleportAround(p *Player, loc data.Location, offset int) {
	x0, y0, z0 := int32(loc.X), int32(loc.Y), int32(loc.Z) //nolint:gosec // world coordinates
	z := s.geo.Height(x0, y0, z0)
	x, y := x0, y0
	if offset > 0 {
		draw := func() {
			x = x0 + int32(s.rnd.Between(-offset, offset)) //nolint:gosec // small offset
			y = y0 + int32(s.rnd.Between(-offset, offset)) //nolint:gosec // small offset
		}
		draw()
		collision := math.Min(p.stats.View().CollisionRadius*2, float64(offset))
		from := geo.Point{X: x0, Y: y0, Z: z0}
		for count := 0; count < 100; count++ {
			to := geo.Point{X: x, Y: y, Z: z}
			dx, dy, dz := float64(x0-x), float64(y0-y), float64(z0-z)
			if math.Sqrt(dx*dx+dy*dy+dz*dz) >= collision && s.geo.CanSeeTarget(from, to) && s.geo.CanMoveToTarget(from, to) {
				break
			}
			draw()
		}
		valid := s.geo.ValidLocation(geo.Point{X: x0, Y: y0, Z: z}, geo.Point{X: x, Y: y, Z: z})
		x, y, z = valid.X, valid.Y, valid.Z
	}
	if err := s.Teleport(p, world.Position{X: x, Y: y, Z: z}); err != nil {
		s.log.Warn("game: restart teleport failed", append(p.logArgs(), "x", x, "y", y, "z", z, "err", err)...)
	}
}
