package game

import (
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/world"
)

// The AI scripts' zones, grand bosses and cinematics (Plan 13, Phase 10.6,
// on ScriptOutput): zones over the zone index and the boss zones
// (bosszone.go), the grand boss store over GrandBossManager (grandboss.go),
// and SpecialCamera, Earthquake, PlaySound and ExShowScreenMessage for one
// player, the players around an object or the players inside a zone.

var (
	_ scripting.ZoneControl    = (*ScriptOutput)(nil)
	_ scripting.GrandBossStore = (*ScriptOutput)(nil)
	_ scripting.Cinematics     = (*ScriptOutput)(nil)
)

// ----- ZoneControl -----

// ZonePlayers is ZoneType.getPlayersInside: the players in the game inside
// the zone.
func (o *ScriptOutput) ZonePlayers(zoneID int) []scripting.PlayerRef {
	s := o.srv.Load()
	if s == nil {
		return nil
	}
	players := s.BossZonePlayers(zoneID)
	out := make([]scripting.PlayerRef, len(players))
	for i, p := range players {
		out[i] = p
	}
	return out
}

// ZoneIsInside is ZoneType.isInsideZone(object): the object's position is
// inside the zone's shape.
func (o *ScriptOutput) ZoneIsInside(zoneID int, obj any) bool {
	s := o.srv.Load()
	if s == nil || s.dialog.zones == nil {
		return false
	}
	var x, y, z int
	if loc, isLoc := obj.(scripting.Location); isLoc {
		x, y, z = loc.X, loc.Y, loc.Z // Mobius isInsideZone(x, y, z)
	} else {
		c := scanCenter(obj)
		if c == nil {
			return false
		}
		pos := c.Position()
		x, y, z = int(pos.X), int(pos.Y), int(pos.Z)
	}
	i, ok := s.dialog.zones.byID[zoneID]
	if !ok {
		return false
	}
	return s.dialog.zones.rules[i].zone.Shape.Contains(x, y, z)
}

// ZoneMovePlayers is BossZone.movePlayersTo.
func (o *ScriptOutput) ZoneMovePlayers(zoneID int, loc scripting.Location) {
	if s := o.srv.Load(); s != nil {
		s.BossZoneMovePlayers(zoneID, worldPosition(loc))
	}
}

// ZoneOustAll is BossZone.oustAllPlayers, or ZoneType.oustAllPlayers (to
// town) for other zones.
func (o *ScriptOutput) ZoneOustAll(zoneID int) {
	if s := o.srv.Load(); s != nil {
		if s.bosses.zones[zoneID] == nil {
			s.oustToTown(zoneID) // quest_world_ai_areas.go
			return
		}
		s.BossZoneOustAll(zoneID)
	}
}

// ZoneAllowEntry is BossZone.allowPlayerEntry.
func (o *ScriptOutput) ZoneAllowEntry(zoneID int, p scripting.PlayerRef, d time.Duration) {
	if s, pl, ok := o.player(p); ok {
		if err := s.BossZoneAllowPlayer(zoneID, pl, d); err != nil {
			s.log.Debug("game: a script let a player into a zone that is no boss zone", append(pl.logArgs(), "zone", zoneID)...)
		}
	}
}

// ZoneRemovePlayer is BossZone.removePlayer.
func (o *ScriptOutput) ZoneRemovePlayer(zoneID int, p scripting.PlayerRef) {
	if s, pl, ok := o.player(p); ok {
		if err := s.BossZoneRemovePlayer(zoneID, pl); err != nil {
			s.log.Debug("game: a script removed a player from a zone that is no boss zone", append(pl.logArgs(), "zone", zoneID)...)
		}
	}
}

// worldPosition converts a script location.
func worldPosition(loc scripting.Location) world.Position {
	return world.Position{X: int32(loc.X), Y: int32(loc.Y), Z: int32(loc.Z), Heading: int32(loc.Heading)} //nolint:gosec // coordinates fit in int32
}

// ----- GrandBossStore -----

// BossStatus is GrandBossManager.getStatus.
func (o *ScriptOutput) BossStatus(id int) (int, bool) {
	if s := o.srv.Load(); s != nil {
		return s.GrandBossStatus(id)
	}
	return 0, false
}

// SetBossStatus is GrandBossManager.setStatus.
func (o *ScriptOutput) SetBossStatus(id, status int) {
	if s := o.srv.Load(); s != nil {
		if err := s.SetGrandBossStatus(id, status); err != nil {
			s.log.Warn("game: a script set the status of an unknown grand boss", "boss", id, "status", status)
		}
	}
}

// BossInfo is GrandBossManager.getStatSet.
func (o *ScriptOutput) BossInfo(id int) (scripting.BossInfo, bool) {
	s := o.srv.Load()
	if s == nil {
		return scripting.BossInfo{}, false
	}
	g, ok := s.GrandBossInfoOf(id)
	if !ok {
		return scripting.BossInfo{}, false
	}
	info := scripting.BossInfo{X: int(g.X), Y: int(g.Y), Z: int(g.Z), Heading: int(g.Heading), HP: g.CurHP, MP: g.CurMP}
	if !g.RespawnTime.IsZero() {
		info.RespawnTime = g.RespawnTime.UnixMilli()
	}
	return info, true
}

// SetBossInfo is GrandBossManager.setStatSet.
func (o *ScriptOutput) SetBossInfo(id int, info scripting.BossInfo) {
	s := o.srv.Load()
	if s == nil {
		return
	}
	//nolint:gosec // coordinates and headings fit in int32
	g := GrandBossInfo{X: int32(info.X), Y: int32(info.Y), Z: int32(info.Z), Heading: int32(info.Heading), CurHP: info.HP, CurMP: info.MP}
	if info.RespawnTime != 0 {
		g.RespawnTime = time.UnixMilli(info.RespawnTime)
	}
	if err := s.SetGrandBossInfo(id, g); err != nil {
		s.log.Warn("game: a script stored the info of an unknown grand boss", "boss", id)
	}
}

// LiveBoss is GrandBossManager.getBoss while the boss is in the world.
func (o *ScriptOutput) LiveBoss(id int) scripting.NpcRef {
	if s := o.srv.Load(); s != nil {
		if n := s.GrandBoss(id); n != nil && n.InWorld() {
			return n.ScriptRef()
		}
	}
	return nil
}

// AddBoss is GrandBossManager.addBoss.
func (o *ScriptOutput) AddBoss(npc scripting.NpcRef) {
	if s, n, ok := o.anyNpc(npc); ok {
		s.AddGrandBoss(n)
	}
}

// ----- Cinematics -----

// show sends body to the audience: a player, the players around an object
// (Mobius broadcastPacket, the player itself included), or the players
// inside a zone (ZoneType.broadcastPacket).
func (s *Server) show(to scripting.Audience, body []byte) {
	switch {
	case to.Player != nil:
		if p := playerOfRef(to.Player); p != nil && p.InWorld() {
			p.Send(body)
		}
	case to.Around != nil:
		if c := scanCenter(to.Around); c != nil {
			_, self := c.(*Player)
			s.world.Broadcast(c, body, self)
		}
	case to.Zone != 0:
		for _, p := range s.BossZonePlayers(to.Zone) {
			p.Send(body)
		}
	}
}

// SpecialCamera sends SpecialCamera at target. Mobius writes the duration,
// not the range.
func (o *ScriptOutput) SpecialCamera(to scripting.Audience, target any, cam scripting.Camera) {
	s, t := o.srv.Load(), creatureOfRef(target)
	if s == nil || t == nil {
		return
	}
	//nolint:gosec // camera values are small script numbers
	s.show(to, serverpackets.NewSpecialCameraPacket(serverpackets.SpecialCamera{
		ObjectID: int32(t.ID()),
		Force:    int32(cam.Force), Angle1: int32(cam.Angle1), Angle2: int32(cam.Angle2),
		Time: int32(cam.Time), Duration: int32(cam.Duration),
		RelYaw: int32(cam.RelYaw), RelPitch: int32(cam.RelPitch), Wide: int32(cam.Wide),
		RelAngle: int32(cam.RelAngle), Unknown: int32(cam.Unk),
	}))
}

// Earthquake sends Earthquake at loc.
func (o *ScriptOutput) Earthquake(to scripting.Audience, loc scripting.Location, intensity, duration int) {
	if s := o.srv.Load(); s != nil {
		//nolint:gosec // coordinates and small script numbers
		s.show(to, serverpackets.NewEarthquakePacket(serverpackets.Earthquake{
			X: int32(loc.X), Y: int32(loc.Y), Z: int32(loc.Z), Intensity: int32(intensity), Duration: int32(duration),
		}))
	}
}

// PlaySoundFrom sends PlaySound: new PlaySound(sound) without a source,
// new PlaySound(1, sound, 1, objectId, x, y, z) with one.
func (o *ScriptOutput) PlaySoundFrom(to scripting.Audience, sound string, source any) {
	s := o.srv.Load()
	if s == nil {
		return
	}
	body := serverpackets.NewQuestSoundPacket(sound)
	if c := creatureOfRef(source); c != nil {
		pos := c.Position()
		body = serverpackets.NewPlaySoundPacket(serverpackets.PlaySound{
			Type: 1, Sound: sound, Bound: 1, ObjectID: int32(c.ID()), X: pos.X, Y: pos.Y, Z: pos.Z, //nolint:gosec // object IDs fit in int32
		})
	}
	s.show(to, body)
}

// ScreenMessage sends ExShowScreenMessage(text, position, ms).
func (o *ScriptOutput) ScreenMessage(to scripting.Audience, text string, position, ms int) {
	if s := o.srv.Load(); s != nil {
		m := serverpackets.ScreenText(text, int32(ms)) //nolint:gosec // milliseconds fit
		m.Position = int32(position)                   //nolint:gosec // small
		s.show(to, serverpackets.NewExShowScreenMessagePacket(m))
	}
}
