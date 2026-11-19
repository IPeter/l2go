package game

import (
	"fmt"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
)

// locMessageHasCoords reports whether the /loc system message id has the
// three coordinate parameters ("Current location : $s1, $s2, $s3 (Near
// ...)"). Mobius adds them only when its message table says so; the
// region IDs the Interlude client does not know get none.
func locMessageHasCoords(id int) bool {
	switch id {
	case 1222, 1537, 1538, 1643, 1659, 1714, 1924:
		return true
	}
	return id >= 910 && id <= 926
}

// userCmdLoc answers /loc (BypassUserCmd 0; Mobius user command Loc): the
// region's "Current location" system message with the player's
// coordinates, or a plain text message where no region is known.
func (s *Server) userCmdLoc(p *Player) {
	pos := p.Position()
	if region := s.locRegionID(p); region > 0 {
		var params []serverpackets.SystemMessageParam
		if locMessageHasCoords(region) {
			params = []serverpackets.SystemMessageParam{
				serverpackets.NumberParam(pos.X),
				serverpackets.NumberParam(pos.Y),
				serverpackets.NumberParam(pos.Z),
			}
		}
		p.SendSystemMessage(serverpackets.SystemMessageID(region), params...) //nolint:gosec // location IDs are small
		return
	}
	p.SendMessage(fmt.Sprintf("Current location: %d, %d, %d", pos.X, pos.Y, pos.Z))
}

// locRegionID returns the location ID /loc names for p: inside a respawn
// zone, the ID of the region its human players restart in (following the
// banned races); elsewhere the ID of the map region of p's position.
func (s *Server) locRegionID(p *Player) int {
	w := s.worldData
	if w == nil {
		return 0
	}
	pos := p.Position()
	if z := s.zoneAt(pos, "RespawnZone"); z != nil {
		return restartRegionLocID(w, data.Race(p.Race()), z.RacePoints[data.RaceHuman]) //nolint:gosec // player races are 0-4
	}
	return w.MapRegionLocID(int(pos.X), int(pos.Y))
}

// restartRegionLocID returns the location ID of the restart region named
// point for a player of race (Mobius MapRegionData.getRestartRegionLocId):
// a missing region gives the default region's ID, and a region that bans
// the race gives the ID of the region it sends the race to.
func restartRegionLocID(w *data.World, race data.Race, point string) int {
	def := 0
	if r := w.DefaultRegion(); r != nil {
		def = r.LocID
	}
	// Banned races lead to another region; a loop of them would make Mobius
	// fail and fall back to the default, as here after a few steps.
	for range 8 {
		r, ok := w.MapRegion(point)
		if !ok {
			return def
		}
		next, banned := r.BannedRaces[race]
		if !banned {
			return r.LocID
		}
		point = next
	}
	return def
}
