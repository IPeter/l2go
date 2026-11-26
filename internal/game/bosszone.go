package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// Boss zones (Mobius BossZone): a player may stay in a boss zone only when
// a script let it in (allowPlayerEntry) and it entered in time, or when it
// logged out inside and comes back within the zone's InvadeTime, or when
// the server started less than InvadeTime ago and the player was allowed
// before the restart (the allowed players are stored with the grand
// bosses). Anyone else is sent to the zone's oust point, or to the nearest
// town without one. GMs come and go as they like. When the last player
// leaves, the raid bosses inside that strayed more than 150 from their
// spawn walk home.

// bossZoneReturnRange is how far from its spawn a raid boss may stand in an
// emptied zone (Mobius isInsideRadius2D(spawn, 150)).
const bossZoneReturnRange = 150

// bossZone is the state of one boss zone (Mobius BossZone.Settings).
type bossZone struct {
	zone   *data.Zone
	params *data.BossZone // nil when the boss data has none: no invade time, no oust point
	// allowed are the players let in, in the order they were (Mobius
	// _playersAllowed).
	allowed []int32
	// reentry holds, per player, until when it may come in (Mobius
	// _playerAllowedReEntryTimes).
	reentry map[int32]time.Time
}

// invadeTime is the zone's InvadeTime.
func (z *bossZone) invadeTime() time.Duration {
	if z.params == nil {
		return 0
	}
	return time.Duration(z.params.InvadeTime) * time.Millisecond
}

// isAllowed reports whether the player is on the allowed list.
func (z *bossZone) isAllowed(id int32) bool { return slices.Contains(z.allowed, id) }

// disallow takes the player off the allowed list.
func (z *bossZone) disallow(id int32) {
	z.allowed = slices.DeleteFunc(z.allowed, func(o int32) bool { return o == id })
}

// setupBossZones makes the state of every BossZone of the world data, with
// the allowed players stored at the last stop (Mobius initZones).
func (s *Server) setupBossZones(rows []storage.BossZonePlayer) {
	f := &s.bosses
	w := s.worldData
	if w == nil {
		return
	}
	for i := range w.Zones {
		z := &w.Zones[i]
		if !w.ZoneTypeIs(z.Type, "BossZone") {
			continue
		}
		bz := &bossZone{zone: z, reentry: make(map[int32]time.Time)}
		if f.data != nil {
			bz.params, _ = f.data.Zone(z.ID)
		}
		f.zones[z.ID] = bz
		f.zoneOrder = append(f.zoneOrder, z.ID)
	}
	slices.Sort(f.zoneOrder)
	for _, r := range rows {
		if bz := f.zones[int(r.Zone)]; bz != nil && !bz.isAllowed(r.Player) {
			bz.allowed = append(bz.allowed, r.Player)
		}
	}
}

// bossZonePlayerRows returns the allowed players of every zone, for the
// store.
func (s *Server) bossZonePlayerRows() []storage.BossZonePlayer {
	var rows []storage.BossZonePlayer
	for _, id := range s.bosses.zoneOrder {
		for _, p := range s.bosses.zones[id].allowed {
			rows = append(rows, storage.BossZonePlayer{Zone: int32(id), Player: p}) //nolint:gosec // zone IDs are small
		}
	}
	return rows
}

// bossZoneEntered is Mobius BossZone.onEnter for a player (enterZone): a
// GM is let in; an allowed player is let in when the server started less
// than InvadeTime ago and it has no re-entry time, or when its re-entry
// time has not passed (which it uses up); anyone else loses its place and
// is sent out, once the zone change is done.
func (s *Server) bossZoneEntered(p *Player, zoneID int) {
	z := s.bosses.zones[zoneID]
	if z == nil || p.IsGM() {
		return
	}
	id := p.ObjectID()
	now := s.Now()
	if z.isAllowed(id) {
		exp, ok := z.reentry[id]
		if !ok {
			if s.bosses.started.After(now.Add(-z.invadeTime())) {
				return
			}
		} else {
			delete(z.reentry, id)
			if exp.After(now) {
				return
			}
		}
		z.disallow(id)
	}
	s.loop.Post(func() {
		if p.InGame() {
			s.oustFromBossZone(p, z)
		}
	})
}

// bossZoneExited is Mobius BossZone.onExit for a player (exitZone, and a
// player leaving the game inside the zone with offline set): an allowed
// player that logs out inside gets InvadeTime to come back; one that walks
// out loses its place. When no player is left inside, the raid bosses in
// the zone that strayed more than 150 from their spawn walk home.
func (s *Server) bossZoneExited(p *Player, zoneID int, offline bool) {
	z := s.bosses.zones[zoneID]
	if z == nil || p.IsGM() { // Mobius returns for a GM before the raid check too
		return
	}
	id := p.ObjectID()
	if offline && z.isAllowed(id) {
		z.reentry[id] = s.Now().Add(z.invadeTime())
	} else {
		z.disallow(id)
		delete(z.reentry, id)
	}
	// Mobius takes the creature off the zone's list before onExit.
	if slices.ContainsFunc(s.BossZonePlayers(zoneID), func(o *Player) bool { return o != p }) {
		return
	}
	for _, n := range s.bossZoneRaids(z) {
		home, _, ok := s.npcHome(n)
		if ok && !n.Dead && !within2D(n.Position(), home, bossZoneReturnRange) {
			s.returnHome(n)
		}
	}
}

// bossZonesLeave is the OnPlayerLeave hook: a player leaving the game
// leaves its boss zones offline (Mobius onExit with isOnline false).
func (s *Server) bossZonesLeave(p *Player) {
	for _, id := range s.bosses.zoneOrder {
		if p.Zones.In(id) {
			s.bossZoneExited(p, id, true)
		}
	}
}

// bossZoneRaids returns the raid and grand bosses with a spawn inside the
// zone (Mobius's raid list of onExit: the zone's attackable raids).
func (s *Server) bossZoneRaids(z *bossZone) []*Npc {
	var out []*Npc
	in := func(n *Npc) bool {
		if n == nil || !n.InWorld() || !creatureRaid(n) {
			return false
		}
		pos := n.Position()
		return z.zone.Shape.Contains(int(pos.X), int(pos.Y), int(pos.Z))
	}
	for _, e := range s.bosses.raids {
		if in(e.boss) {
			out = append(out, e.boss)
		}
	}
	for _, id := range s.bosses.grandOrder {
		if n := s.bosses.grand[id].boss; in(n) && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}

// oustFromBossZone sends p to the zone's oust point when it has one, to the
// nearest town otherwise (Mobius teleToLocation(TeleportWhereType.TOWN),
// with the random offset).
func (s *Server) oustFromBossZone(p *Player, z *bossZone) {
	if z.params != nil {
		if loc, ok := z.params.OustPoint(); ok {
			if err := s.Teleport(p, world.Position{X: int32(loc.X), Y: int32(loc.Y), Z: int32(loc.Z)}); err != nil { //nolint:gosec // world coordinates fit
				s.log.Warn("game: cannot oust a player from a boss zone", append(p.logArgs(), "zone", z.zone.ID, "err", err)...)
			}
			return
		}
	}
	if loc, ok := s.restartLocation(p); ok {
		s.teleportAround(p, loc, maxOffsetOnTeleport)
	}
}

// bossZoneOf returns the zone, or an error for one that is not a boss zone.
func (s *Server) bossZoneOf(zoneID int) (*bossZone, error) {
	if z := s.bosses.zones[zoneID]; z != nil {
		return z, nil
	}
	return nil, errBossUnknown
}

// BossZonePlayers returns the players in the game inside a boss zone
// (Mobius getPlayersInside), in no particular order. Loop-only.
func (s *Server) BossZonePlayers(zoneID int) []*Player {
	var out []*Player
	for p := range s.Players() {
		if p.InGame() && p.Zones.In(zoneID) {
			out = append(out, p)
		}
	}
	return out
}

// BossZoneAllowPlayer lets p into a boss zone for d (Mobius
// allowPlayerEntry; GMs need not be). Loop-only.
func (s *Server) BossZoneAllowPlayer(zoneID int, p *Player, d time.Duration) error {
	z, err := s.bossZoneOf(zoneID)
	if err != nil || p.IsGM() {
		return err
	}
	id := p.ObjectID()
	if !z.isAllowed(id) {
		z.allowed = append(z.allowed, id)
	}
	z.reentry[id] = s.Now().Add(d)
	return nil
}

// BossZoneRemovePlayer takes p's place in a boss zone (Mobius
// removePlayer). Loop-only.
func (s *Server) BossZoneRemovePlayer(zoneID int, p *Player) error {
	z, err := s.bossZoneOf(zoneID)
	if err != nil || p.IsGM() {
		return err
	}
	z.disallow(p.ObjectID())
	delete(z.reentry, p.ObjectID())
	return nil
}

// BossZoneIsPlayerAllowed reports whether p may be in a boss zone, and
// sends it out when not (Mobius isPlayerAllowed). Loop-only.
func (s *Server) BossZoneIsPlayerAllowed(zoneID int, p *Player) bool {
	z, err := s.bossZoneOf(zoneID)
	if err != nil {
		return false
	}
	if p.IsGM() || z.isAllowed(p.ObjectID()) {
		return true
	}
	s.oustFromBossZone(p, z)
	return false
}

// BossZoneAllowedPlayers returns the object IDs of the players allowed into
// a boss zone, in the order they were (Mobius getAllowedPlayers).
// Loop-only.
func (s *Server) BossZoneAllowedPlayers(zoneID int) []int32 {
	if z := s.bosses.zones[zoneID]; z != nil {
		return slices.Clone(z.allowed)
	}
	return nil
}

// BossZoneMovePlayers teleports the players inside a boss zone to pos
// (Mobius movePlayersTo). Loop-only.
func (s *Server) BossZoneMovePlayers(zoneID int, pos world.Position) {
	for _, p := range s.BossZonePlayers(zoneID) {
		if err := s.Teleport(p, pos); err != nil {
			s.log.Warn("game: cannot move a player in a boss zone", append(p.logArgs(), "zone", zoneID, "err", err)...)
		}
	}
}

// BossZoneOustAll sends every player inside a boss zone out and forgets the
// allowed players and their re-entry times (Mobius oustAllPlayers).
// Loop-only.
func (s *Server) BossZoneOustAll(zoneID int) {
	z := s.bosses.zones[zoneID]
	if z == nil {
		return
	}
	players := s.BossZonePlayers(zoneID)
	if len(players) == 0 {
		return
	}
	for _, p := range players {
		s.oustFromBossZone(p, z)
	}
	clear(z.reentry)
	z.allowed = nil
}

// SetBossZoneEnabled turns a boss zone on or off; a change sends everyone
// inside out first (Mobius BossZone.setEnabled). Loop-only.
func (s *Server) SetBossZoneEnabled(zoneID int, on bool) {
	ix := s.dialog.zones
	if ix == nil || s.bosses.zones[zoneID] == nil {
		return
	}
	i, ok := ix.byID[zoneID]
	if !ok {
		return
	}
	if ix.rules[i].enabled != on {
		s.BossZoneOustAll(zoneID)
	}
	ix.rules[i].enabled = on
}

// BossZoneAt returns the ID of the boss zone containing pos, or 0 (Mobius
// GrandBossManager.getZone(x, y, z)). Loop-only.
func (s *Server) BossZoneAt(pos world.Position) int {
	for _, id := range s.bosses.zoneOrder {
		if s.bosses.zones[id].zone.Shape.Contains(int(pos.X), int(pos.Y), int(pos.Z)) {
			return id
		}
	}
	return 0
}
