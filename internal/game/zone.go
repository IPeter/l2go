package game

import (
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// Zones (decision N8): every zone of the world data is indexed by world
// region; a player's zones are worked out again when it has moved 100 units
// or changed region since the last time (Mobius revalidateZone), and on
// entering the world and after a teleport. Entering and leaving a zone
// counts its Mobius ZoneId flags in Player.Zones. In Milestone 4 only water
// changes behaviour (swim speeds), together with the swamp's slowness and
// the danger icon of effect zones, which come for free, and the client's
// compass (ExSetCompassZoneCode).

// zoneRevalidateDistance is how far a player moves before its zones are
// worked out again (Mobius Player.revalidateZone).
const zoneRevalidateDistance = 100

// zoneCellsX and zoneCellsY are the size of the zone index: one cell per
// world region of world.RegionSize units.
const (
	zoneCellsX = (world.WorldXMax - world.WorldXMin) >> world.RegionShift
	zoneCellsY = (world.WorldYMax - world.WorldYMin) >> world.RegionShift
)

// zoneRule is what a zone does to the players inside it, worked out once
// from its Mobius zone type and parameters.
type zoneRule struct {
	zone  *data.Zone
	flags []ZoneFlag
	// water: a WaterZone (swim speeds). swamp: the move bonus of a
	// SwampZone, 0 for other zones. danger: an EffectZone showing the danger
	// icon.
	water  bool
	swamp  float64
	danger bool
	// check is Mobius's _checkAffected: the zone has a parameter of the
	// zone base class, so who it affects is checked.
	check     bool
	enabled   bool
	minLevel  int
	maxLevel  int
	races     []int
	classes   []int
	classType int    // 0 everyone, 1 fighters, 2 mystics
	target    string // the Mobius InstanceType the zone affects
}

// zoneIndex finds the zones around a point.
type zoneIndex struct {
	rules []zoneRule      // by index in World.Zones
	byID  map[int]int     // zone ID → index
	cells map[int][]int32 // cell → indexes of the zones whose bounds touch it
}

// newZoneIndex indexes the zones of w.
func newZoneIndex(w *data.World) *zoneIndex {
	ix := &zoneIndex{
		rules: make([]zoneRule, len(w.Zones)),
		byID:  make(map[int]int, len(w.Zones)),
		cells: make(map[int][]int32),
	}
	for i := range w.Zones {
		z := &w.Zones[i]
		ix.rules[i] = newZoneRule(z)
		ix.byID[z.ID] = i
		minX, minY, maxX, maxY := z.Shape.Bounds()
		x0, y0 := zoneCellXY(int32(minX), int32(minY)) //nolint:gosec // world coordinates fit in int32
		x1, y1 := zoneCellXY(int32(maxX), int32(maxY)) //nolint:gosec // world coordinates fit in int32
		for cx := x0; cx <= x1; cx++ {
			for cy := y0; cy <= y1; cy++ {
				c := cx*zoneCellsY + cy
				ix.cells[c] = append(ix.cells[c], int32(i)) //nolint:gosec // fewer zones than 2^31
			}
		}
	}
	return ix
}

// zoneCellXY returns the cell of a point, clamped to the world.
func zoneCellXY(x, y int32) (int, int) {
	cx := int((x - world.WorldXMin) >> world.RegionShift)
	cy := int((y - world.WorldYMin) >> world.RegionShift)
	return min(max(cx, 0), zoneCellsX-1), min(max(cy, 0), zoneCellsY-1)
}

// zoneCellOf returns the index cell of a position.
func zoneCellOf(pos world.Position) int {
	cx, cy := zoneCellXY(pos.X, pos.Y)
	return cx*zoneCellsY + cy
}

// at calls fn for every zone that contains pos, in zone ID order, until fn
// returns false.
func (ix *zoneIndex) at(pos world.Position, fn func(r *zoneRule) bool) {
	for _, i := range ix.cells[zoneCellOf(pos)] {
		r := &ix.rules[i]
		if r.zone.Shape.Contains(int(pos.X), int(pos.Y), int(pos.Z)) && !fn(r) {
			return
		}
	}
}

// zoneAt returns the first zone of the Mobius type typ (or a type extending
// it) that contains pos, as Mobius ZoneManager.getZone(loc, class) does, or
// nil.
func (s *Server) zoneAt(pos world.Position, typ string) *data.Zone {
	if s.dialog.zones == nil {
		return nil
	}
	var found *data.Zone
	s.dialog.zones.at(pos, func(r *zoneRule) bool {
		if s.worldData.ZoneTypeIs(r.zone.Type, typ) {
			found = r.zone
			return false
		}
		return true
	})
	return found
}

// onCastleGround reports whether pos is inside a castle's siege zone
// (Mobius CastleManager.getCastle(object) != null).
func (s *Server) onCastleGround(pos world.Position) bool {
	if s.dialog.zones == nil {
		return false
	}
	found := false
	s.dialog.zones.at(pos, func(r *zoneRule) bool {
		if r.zone.Type == "SiegeZone" {
			if v, ok := r.zone.Param("castleId"); ok {
				if id, err := strconv.Atoi(v); err == nil && id > 0 {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

// zoneBaseParams are the parameters of the Mobius ZoneType base class
// (ZoneType.setParameter); any of them makes the zone check whom it
// affects.
var zoneBaseParams = map[string]bool{
	"name": true, "affectedLvlMin": true, "affectedLvlMax": true,
	"affectedRace": true, "affectedClassId": true, "affectedClassType": true,
	"targetClass": true, "allowStore": true, "default_enabled": true, "instanceId": true,
}

// zoneBool parses a value like Java's Boolean.parseBoolean.
func zoneBool(v string) bool { return strings.EqualFold(v, "true") }

// newZoneRule works out what z does to players, following the onEnter of
// its Mobius zone type with the Mobius default settings (PeaceZoneMode 0,
// JailIsPvp and JailDisableTransaction off) and the state of a world
// without sieges or olympiad games: siege zones and the swamps of castles
// are quiet. Side effects of other features come with them: the combat
// zone messages, the olympiad stadium sending intruders to town, the
// mother tree messages and regeneration, effect zone skills.
func newZoneRule(z *data.Zone) zoneRule {
	r := zoneRule{zone: z, enabled: true, maxLevel: 0xFF, target: "Creature"}
	allowStore := true
	for _, prm := range z.Params {
		if !zoneBaseParams[prm.Name] {
			continue
		}
		r.check = true
		v := prm.Value
		switch prm.Name {
		case "affectedLvlMin":
			r.minLevel, _ = strconv.Atoi(v)
		case "affectedLvlMax":
			r.maxLevel, _ = strconv.Atoi(v)
		case "affectedRace":
			if n, err := strconv.Atoi(v); err == nil {
				r.races = append(r.races, n)
			}
		case "affectedClassId":
			if n, err := strconv.Atoi(v); err == nil {
				r.classes = append(r.classes, n)
			}
		case "affectedClassType":
			r.classType = 2
			if v == "Fighter" {
				r.classType = 1
			}
		case "targetClass":
			r.target = v
		case "allowStore":
			allowStore = zoneBool(v)
		case "default_enabled":
			r.enabled = zoneBool(v)
		}
	}
	param := func(name string) (string, bool) {
		for _, prm := range z.Params {
			if strings.EqualFold(prm.Name, name) {
				return prm.Value, true
			}
		}
		return "", false
	}

	switch z.Type {
	case "WaterZone":
		r.flags = []ZoneFlag{ZoneWater}
		r.water = true
	case "PeaceZone":
		r.flags = []ZoneFlag{ZonePeace}
		if !allowStore {
			r.flags = append(r.flags, ZoneNoStore)
		}
	case "TownZone":
		r.flags = []ZoneFlag{ZoneTown}
	case "NoPvPZone":
		r.flags = []ZoneFlag{ZoneNoPvP}
	case "NoStoreZone":
		r.flags = []ZoneFlag{ZoneNoStore}
	case "ArenaZone":
		r.flags = []ZoneFlag{ZonePvP}
	case "CastleZone":
		r.flags = []ZoneFlag{ZoneCastle}
	case "ClanHallZone", "SiegableHallZone":
		r.flags = []ZoneFlag{ZoneClanHall}
	case "JailZone":
		r.flags = []ZoneFlag{ZoneJail, ZoneNoSummonFriend}
	case "MotherTreeZone":
		r.flags = []ZoneFlag{ZoneMotherTree}
	case "NoLandingZone":
		r.flags = []ZoneFlag{ZoneNoLanding}
	case "NoRestartZone":
		r.flags = []ZoneFlag{ZoneNoRestart}
	case "NoSummonFriendZone", "ResidenceTeleportZone":
		r.flags = []ZoneFlag{ZoneNoSummonFriend}
	case "ScriptZone":
		r.flags = []ZoneFlag{ZoneScript}
	case "HqZone":
		r.flags = []ZoneFlag{ZoneHQ}
	case "DerbyTrackZone":
		r.flags = []ZoneFlag{ZoneMonsterTrack}
	case "SwampZone":
		if v, ok := param("castleId"); ok {
			if id, err := strconv.Atoi(v); err == nil && id > 0 {
				break // a castle's swamp works only during its siege
			}
		}
		r.flags = []ZoneFlag{ZoneSwamp}
		r.swamp = 0.5
		if v, ok := param("move_bonus"); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				r.swamp = f
			}
		}
	case "EffectZone":
		// EffectZone affects playable creatures only by default.
		r.check = true
		if !zoneHasParam(z, "targetClass") {
			r.target = "Playable"
		}
		r.flags = []ZoneFlag{ZoneAltered}
		r.danger = true
		if v, ok := param("showDangerIcon"); ok {
			r.danger = zoneBool(v)
		}
		if r.danger {
			r.flags = append(r.flags, ZoneDangerArea)
		}
	case "ConditionZone":
		if v, ok := param("NoBookmark"); ok && zoneBool(v) {
			r.flags = append(r.flags, ZoneNoBookmark)
		}
		if v, ok := param("NoItemDrop"); ok && zoneBool(v) {
			r.flags = append(r.flags, ZoneNoItemDrop)
		}
	case "OlympiadStadiumZone":
		r.flags = []ZoneFlag{ZonePvP, ZoneNoSummonFriend, ZoneNoLanding, ZoneNoRestart, ZoneNoBookmark}
	}
	return r
}

// zoneHasParam reports whether z has the parameter name.
func zoneHasParam(z *data.Zone, name string) bool {
	_, ok := z.Param(name)
	return ok
}

// affects reports whether the zone affects a player of the given level,
// class and race (Mobius ZoneType.isAffected, without instances).
func (r *zoneRule) affects(level, classID, race int) bool {
	if !r.enabled {
		return false
	}
	if level < r.minLevel || level > r.maxLevel {
		return false
	}
	switch r.target {
	case "Player", "Playable", "Creature", "WorldObject":
	default:
		return false
	}
	if r.classType != 0 {
		mage := data.PlayerClass(classID).IsMage()
		if (r.classType == 1 && mage) || (r.classType == 2 && !mage) {
			return false
		}
	}
	if r.races != nil && !slices.Contains(r.races, race) {
		return false
	}
	if r.classes != nil && !slices.Contains(r.classes, classID) {
		return false
	}
	return true
}

// zoneTrack is where a player's zones were last worked out.
type zoneTrack struct {
	done  bool
	pos   world.Position
	cell  int
	force bool // the next move works them out again (after a teleport)
	// compass is the last ExSetCompassZoneCode sent, 0 before the first.
	compass serverpackets.CompassZone
}

// zonesMoved is the OnPlayerMoved hook: the zones are worked out again on
// entering the world (from == to), after a teleport, and when the player
// moved far enough or into another region.
func (s *Server) zonesMoved(p *Player, from, to world.Position) {
	st := s.dialogState(p)
	force := from == to || st.zones.force
	st.zones.force = false
	s.revalidateZones(p, force)
}

// revalidateZones works out p's zones at its position (Mobius
// ZoneRegion.revalidateZones): it leaves the zones that no longer contain
// it and enters those that do and affect it. A zone that contains p but
// does not affect it changes nothing, as in Mobius. Without force, nothing
// happens unless p moved zoneRevalidateDistance or changed region since the
// last time.
func (s *Server) revalidateZones(p *Player, force bool) {
	ix := s.dialog.zones
	if ix == nil || !p.InWorld() {
		return
	}
	st := s.dialogState(p)
	pos := p.Position()
	cell := zoneCellOf(pos)
	if !force && st.zones.done && cell == st.zones.cell &&
		pos.DistanceSq3D(st.zones.pos) < zoneRevalidateDistance*zoneRevalidateDistance {
		return
	}
	st.zones = zoneTrack{done: true, pos: pos, cell: cell, compass: st.zones.compass}

	x, y, z := int(pos.X), int(pos.Y), int(pos.Z)
	for _, id := range slices.Clone(p.Zones.IDs()) {
		i, ok := ix.byID[id]
		if !ok || !ix.rules[i].zone.Shape.Contains(x, y, z) {
			s.exitZone(p, id)
		}
	}
	for _, i := range ix.cells[cell] {
		r := &ix.rules[i]
		if p.Zones.In(r.zone.ID) || !r.zone.Shape.Contains(x, y, z) {
			continue
		}
		if r.check && !r.affects(p.Level(), p.ClassID(), p.Race()) {
			continue
		}
		s.enterZone(p, r)
	}
	if c := compassZone(p); c != st.zones.compass {
		st.zones.compass = c
		p.Send(serverpackets.NewExSetCompassZoneCodePacket(c))
	}
}

// compassZone is the compass code of p's zones, in the order of Mobius
// Player.revalidateZone (the Seven Signs dungeon is not there yet).
func compassZone(p *Player) serverpackets.CompassZone {
	switch {
	case p.Zones.Inside(ZoneAltered):
		return serverpackets.CompassAltered
	case p.Zones.Inside(ZoneSiege):
		return serverpackets.CompassSiegeWar2
	case p.Zones.Inside(ZonePvP):
		return serverpackets.CompassPvP
	case p.Zones.Inside(ZonePeace), p.Zones.Inside(ZoneNoPvP):
		return serverpackets.CompassPeace
	}
	return serverpackets.CompassGeneral
}

// enterZone records that p entered r's zone and does what the zone does on
// entering.
func (s *Server) enterZone(p *Player, r *zoneRule) {
	p.Zones.Add(r.zone.ID)
	for _, f := range r.flags {
		p.Zones.Enter(f)
	}
	if r.water || r.swamp != 0 {
		// Mobius WaterZone and SwampZone onEnter: broadcastUserInfo with the
		// new speeds.
		s.updateMotion(p)
		p.BroadcastUserInfo()
	}
	if r.danger {
		p.Send(etcStatusPacket(p))
	}
	s.questZone(p, r.zone.ID, r.zone.Type, true) // quest_events.go
	s.bossZoneEntered(p, r.zone.ID)              // bosszone.go
	s.residenceZoneEntered(p, r.zone)            // siege_zone.go
	s.hallZoneEntered(p, r.zone)                 // clanhall.go
	s.olympiadZoneEntered(p, r.zone.Type)        // olympiad_rules.go
	s.mountZoneEntered(p, r.zone)                // mount.go
}

// exitZone records that p left the zone id and does what the zone does on
// leaving.
func (s *Server) exitZone(p *Player, id int) {
	if !p.Zones.Remove(id) {
		return
	}
	i, ok := s.dialog.zones.byID[id]
	if !ok {
		return
	}
	r := &s.dialog.zones.rules[i]
	wasSwamp := p.Zones.Inside(ZoneSwamp)
	for _, f := range r.flags {
		p.Zones.Leave(f)
	}
	if r.water || (r.swamp != 0 && wasSwamp) {
		s.updateMotion(p)
		if !p.Teleporting() {
			p.BroadcastUserInfo()
		}
	}
	if r.danger && !p.Zones.Inside(ZoneDangerArea) {
		p.Send(etcStatusPacket(p))
	}
	s.questZone(p, id, r.zone.Type, false) // quest_events.go
	s.bossZoneExited(p, id, false)         // bosszone.go
	s.residenceZoneExited(p, id)           // siege_zone.go
	s.mountZoneExited(p, r.zone)           // mount.go
}

// updateMotion sets the stat set's water and swamp state from p's zones:
// in water the swim speeds apply; in a swamp, the move bonus of the first
// swamp zone p entered.
func (s *Server) updateMotion(p *Player) {
	m := p.Stats().Motion()
	m.InWater = p.Zones.Inside(ZoneWater)
	m.Swamp = 0
	if p.Zones.Inside(ZoneSwamp) {
		for _, id := range p.Zones.IDs() {
			if i, ok := s.dialog.zones.byID[id]; ok && s.dialog.zones.rules[i].swamp != 0 {
				m.Swamp = s.dialog.zones.rules[i].swamp
				break
			}
		}
	}
	p.Stats().SetMotion(m)
}
