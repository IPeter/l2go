package game

import (
	"slices"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/world"
)

// spawnNightEndHour is the game hour night ends at (world.GameTime: it is
// night before 06:00).
const spawnNightEndHour = 6

// spawnHeightCorrection bounds the height correction of a monster's spawn
// point to the geodata ground (Mobius Spawn.initializeNpc: "do not correct
// Z distances greater than 300").
const spawnHeightCorrection = 300

// spawnRandomHeading is Mobius's bound for a random spawn heading
// (Rnd.get(61794)).
const spawnRandomHeading = 61794

// spawnPeriod is when the NPC of a spawn slot exists.
type spawnPeriod uint8

const (
	spawnAlways    spawnPeriod = iota
	spawnDayOnly               // only by day (periodOfDay "day")
	spawnNightOnly             // only by night (periodOfDay "night")
)

// npcSpawn is a spawn slot: it keeps one NPC at its place and brings it
// back after it is gone (Mobius Spawn; SpawnData makes one Spawn of amount
// 1 for each NPC of an entry's count). The slot keeps its NPC object, and
// so its object ID, across respawns, as Mobius respawnNpc does.
type npcSpawn struct {
	tmpl *data.NpcTemplate
	// block is the spawn list block of the slot; nil for a GM spawn.
	block *data.Spawn
	// territory is where the NPC spawns at a random point with a random
	// heading; nil for a fixed point.
	territory *data.SpawnTerritory
	// point is the fixed spawn point; a Heading of -1 is a random heading.
	point world.Position
	// respawnMin and respawnMax bound the respawn delay (Mobius
	// setRespawnDelay); 0 means no respawn.
	respawnMin, respawnMax time.Duration
	period                 spawnPeriod
	// gm is set for a slot made by //spawn.
	gm bool

	// respawn is Mobius _doRespawn: a gone NPC comes back after the delay.
	respawn bool
	// npc is the slot's NPC, nil until it first spawns.
	npc *Npc
	// lastPoint is where the NPC last spawned (Mobius keeps the random
	// point of a territory spawn as the spawn's location).
	lastPoint world.Position
	// timer is the pending respawn, or 0.
	timer world.TimerID
	// index is the slot's place in npcFeature.spawns, -1 once deleted.
	index int
}

// deleted reports whether the slot was deleted for good (//delete).
func (sp *npcSpawn) deleted() bool { return sp.index < 0 }

// aiData returns the AI data of the slot's block. Mobius applies it only
// to blocks with a name (NpcPersonalAIData is keyed by the spawn name).
func (sp *npcSpawn) aiData() map[string]int {
	if sp.block == nil || sp.block.Name == "" {
		return nil
	}
	return sp.block.AIData
}

// spawnRespawnDelays turns an entry's respawn delay and random spread, in
// seconds, into the bounds of the delay (Mobius Spawn.setRespawnDelay):
// delay ± random, each at least a second; a delay of 0 means no respawn.
func spawnRespawnDelays(delay, random int) (lo, hi time.Duration) {
	if delay == 0 {
		return 0, 0
	}
	lo = time.Duration(max(1, delay-random)) * time.Second
	hi = time.Duration(max(1, delay+random)) * time.Second
	return lo, hi
}

// spawnIsCustomFile reports whether a spawn list file is one of the custom
// lists (below a "custom" folder), which spawn only with
// Settings.NPCs.SpawnCustom.
func spawnIsCustomFile(file string) bool {
	first, _, _ := strings.Cut(strings.ReplaceAll(file, `\`, "/"), "/")
	return strings.EqualFold(first, "custom")
}

// spawnCounts sums up the spawning at startup, for the log.
type spawnCounts struct {
	slots, spawned, failed int
	dayOnly, nightOnly     int
	notSpawnable, custom   int
	groups                 int
}

// spawnWorld makes the spawn slots of the world data's spawn lists and
// spawns them (Mobius SpawnData.load): every entry of a block gets count
// slots, at a random point of the block's territory or at the entry's
// fixed point. Entries of templates Mobius does not spawn (raid bosses,
// siege guards, classes that are not NPCs, traps) are left out, and so are
// the custom lists unless configured. NPCs of a period of day exist only in
// that period: the ones of the current period spawn now, and the game time
// switches them (Mobius DayNightSpawnManager).
func (s *Server) spawnWorld() {
	w := s.worldData
	if w == nil {
		return
	}
	began := time.Now()
	var c spawnCounts
	for i := range w.Spawns {
		b := &w.Spawns[i]
		if !s.settings.NPCs.SpawnCustom && spawnIsCustomFile(b.File) {
			c.custom += len(b.Npcs)
			continue
		}
		if b.Selection != "" {
			// Spawn groups (selection all/random/randomFill) are not used by
			// the Interlude lists.
			s.log.Warn("game: spawn groups are not supported; spawn block left out", "file", b.File, "name", b.Name, "selection", b.Selection)
			c.groups++
			continue
		}
		var terr *data.SpawnTerritory
		if b.Zone != "" {
			terr, _ = w.Territory(b.Zone) // data.NewWorld checked that it exists
		}
		for j := range b.Npcs {
			e := &b.Npcs[j]
			t, err := s.spawnListTemplate(e.NpcID)
			if err != nil {
				c.notSpawnable += e.Count
				continue
			}
			lo, hi := spawnRespawnDelays(e.RespawnDelay, e.RespawnRandom)
			for range e.Count {
				sp := &npcSpawn{tmpl: t, block: b, territory: terr, respawnMin: lo, respawnMax: hi, index: -1}
				if terr == nil {
					sp.point = world.Position{X: int32(e.X), Y: int32(e.Y), Z: int32(e.Z), Heading: int32(e.Heading)} //nolint:gosec // world coordinates fit
				} else {
					sp.point.Heading = -1
				}
				s.addSpawnSlot(sp)
				c.slots++
				switch e.PeriodOfDay {
				case data.PeriodDay:
					sp.period = spawnDayOnly
					s.npc.dayOnly = append(s.npc.dayOnly, sp)
					c.dayOnly++
				case data.PeriodNight:
					sp.period = spawnNightOnly
					s.npc.nightOnly = append(s.npc.nightOnly, sp)
					c.nightOnly++
				default:
					// Mobius Spawn.init: spawn, then respawn only with a delay.
					if s.spawnSlot(sp) != nil {
						c.spawned++
					} else {
						c.failed++
					}
					sp.respawn = sp.respawnMin > 0
				}
			}
		}
	}

	gt := s.world.GameTime()
	s.npc.night = gt.Hour() < spawnNightEndHour
	current := s.npc.dayOnly
	if s.npc.night {
		current = s.npc.nightOnly
	}
	for _, sp := range current {
		sp.respawn = true // Mobius startRespawn, then doSpawn
		if s.spawnSlot(sp) != nil {
			c.spawned++
		} else {
			c.failed++
		}
	}
	gt.OnDayNightChange(s.switchDayNight)

	s.log.Info("NPCs spawned", "slots", c.slots, "npcs", c.spawned, "failed", c.failed,
		"dayOnly", c.dayOnly, "nightOnly", c.nightOnly, "night", s.npc.night,
		"notSpawnable", c.notSpawnable, "custom", c.custom, "groups", c.groups,
		"templates", len(s.npc.classes), "took", time.Since(began).Round(time.Millisecond))
}

// spawnListTemplate returns the template of a spawn list entry if Mobius
// spawns it from the lists (SpawnData.checkTemplate, and Spawn's class
// rules).
func (s *Server) spawnListTemplate(npcID int) (*data.NpcTemplate, error) {
	if !s.worldData.Spawnable(npcID) {
		return nil, errNoNpcTemplate
	}
	return s.spawnableTemplate(npcID)
}

// addSpawnSlot registers a new slot.
func (s *Server) addSpawnSlot(sp *npcSpawn) {
	f := &s.npc
	f.ensureMaps()
	sp.index = len(f.spawns)
	f.spawns = append(f.spawns, sp)
	if t := sp.territory; t != nil {
		f.byTerritory[t.Name] = append(f.byTerritory[t.Name], sp)
	}
}

// spawnSlot puts the slot's NPC into the world (Mobius Spawn.doSpawn and
// initializeNpc), making the NPC on first use. It returns the NPC, or nil
// when the slot was deleted or the NPC could not be placed. An NPC already
// in the world stays where it is.
func (s *Server) spawnSlot(sp *npcSpawn) *Npc {
	if sp.deleted() {
		return nil
	}
	n := sp.npc
	if n == nil {
		var err error
		if n, err = s.newNpc(sp.tmpl); err != nil {
			s.log.Error("game: cannot make an NPC", "npc", sp.tmpl.ID, "err", err)
			return nil
		}
		n.spawn = sp
		sp.npc = n
	}
	if n.InWorld() {
		return n
	}
	pos := s.spawnPoint(sp, n)
	if err := s.placeNpc(n, pos, sp.aiData()); err != nil {
		s.log.Warn("game: cannot spawn an NPC", "npc", sp.tmpl.ID, "name", sp.tmpl.Name, "pos", pos, "err", err)
		return nil
	}
	sp.lastPoint = pos
	return n
}

// spawnPoint picks where the slot's NPC spawns (Mobius initializeNpc): a
// random point of the territory with a random heading, or the fixed point.
// A monster that does not fly gets the geodata ground height when it is
// within 300 of the point, but never above the territory. Without geodata
// the height stays the territory's middle (decision N2) or the list's Z.
// The water check of Mobius waits for zones of NPCs; it changes nothing
// without geodata.
func (s *Server) spawnPoint(sp *npcSpawn, n *Npc) world.Position {
	pos := sp.point
	if t := sp.territory; t != nil {
		loc := t.RandomPoint(s.npc.rnd.IntN)
		x, y := int32(loc.X), int32(loc.Y)                                                 //nolint:gosec // world coordinates fit
		pos = world.Position{X: x, Y: y, Z: s.geo.Height(x, y, int32(loc.Z)), Heading: -1} //nolint:gosec // world coordinates fit
	}
	if n.class.monster && !sp.tmpl.Status.Flying {
		if geoZ := s.geo.Height(pos.X, pos.Y, pos.Z); spawnAbs(pos.Z-geoZ) < spawnHeightCorrection {
			pos.Z = geoZ
		}
		if t := sp.territory; t != nil {
			pos.Z = min(pos.Z, int32(t.Shape.HighZ())) //nolint:gosec // world coordinates fit
		}
	}
	if pos.Heading == -1 {
		pos.Heading = int32(s.npc.rnd.IntN(spawnRandomHeading)) //nolint:gosec // below 61794
	}
	return pos
}

func spawnAbs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// scheduleRespawn starts the respawn timer of a slot whose NPC is gone
// (Mobius Spawn.decreaseCount): after respawnMin, or a uniform time between
// respawnMin and respawnMax in milliseconds, unless the slot does not
// respawn or a respawn is pending already.
func (s *Server) scheduleRespawn(sp *npcSpawn) {
	if !sp.respawn || sp.deleted() || sp.timer != 0 {
		return
	}
	d := sp.respawnMin
	if spread := sp.respawnMax - sp.respawnMin; spread > 0 {
		d += time.Duration(s.npc.rnd.Int64N(int64(spread/time.Millisecond)+1)) * time.Millisecond
	}
	sp.timer = s.loop.After(d, func() {
		sp.timer = 0
		delete(s.npc.pending, sp)
		// Mobius respawnNpc: only while the slot still respawns.
		if sp.respawn && !sp.deleted() {
			s.spawnSlot(sp)
		}
	})
	s.npc.ensureMaps()
	s.npc.pending[sp] = struct{}{}
}

// cancelRespawn stops a pending respawn of the slot.
func (s *Server) cancelRespawn(sp *npcSpawn) {
	if sp.timer == 0 {
		return
	}
	s.loop.Cancel(sp.timer)
	sp.timer = 0
	delete(s.npc.pending, sp)
}

// respawnSlotNow takes the slot's NPC out of the world if it is in it and
// spawns it again at once, at a new point for a territory slot (//respawn).
func (s *Server) respawnSlotNow(sp *npcSpawn) *Npc {
	s.cancelRespawn(sp)
	if n := sp.npc; n != nil && n.InWorld() {
		s.RemoveObject(n)
	}
	return s.spawnSlot(sp)
}

// respawnPending spawns every slot waiting for its respawn timer now and
// returns how many NPCs came back (//respawn without a target).
func (s *Server) respawnPending() int {
	waiting := make([]*npcSpawn, 0, len(s.npc.pending))
	for sp := range s.npc.pending {
		waiting = append(waiting, sp)
	}
	// The map order is random; spawn in slot order, so that IDs and points
	// are drawn in the same order each time.
	slices.SortFunc(waiting, func(a, b *npcSpawn) int { return a.index - b.index })
	back := 0
	for _, sp := range waiting {
		s.cancelRespawn(sp)
		if s.spawnSlot(sp) != nil {
			back++
		}
	}
	return back
}

// deleteSpawn deletes a slot for good (Mobius admin_delete: deleteMe,
// stopRespawn and SpawnData.deleteSpawn, without the XML file): its NPC
// leaves the world, its respawn stops, and its NPC's ID is released.
func (s *Server) deleteSpawn(sp *npcSpawn) {
	if sp.deleted() {
		return
	}
	sp.respawn = false
	s.cancelRespawn(sp)
	if n := sp.npc; n != nil {
		if n.InWorld() {
			s.RemoveObject(n)
		}
		s.releaseNpcID(n)
	}
	f := &s.npc
	last := len(f.spawns) - 1
	moved := f.spawns[last]
	f.spawns[sp.index] = moved
	moved.index = sp.index
	f.spawns[last] = nil
	f.spawns = f.spawns[:last]
	sp.index = -1
	if t := sp.territory; t != nil {
		list := slices.DeleteFunc(f.byTerritory[t.Name], func(o *npcSpawn) bool { return o == sp })
		if len(list) == 0 {
			delete(f.byTerritory, t.Name)
		} else {
			f.byTerritory[t.Name] = list
		}
	}
}

// deleteTerritorySpawns deletes every slot of a spawn territory for good
// and returns how many there were (Mobius admin_delete_group, which also
// catches the slots whose NPC is waiting for a respawn).
func (s *Server) deleteTerritorySpawns(name string) int {
	list := slices.Clone(s.npc.byTerritory[name])
	for _, sp := range list {
		s.deleteSpawn(sp)
	}
	return len(list)
}

// switchDayNight is the day/night hook (Mobius
// DayNightSpawnManager.notifyChangeMode): the NPCs of the period that ends
// leave without respawning, those of the period that starts spawn.
func (s *Server) switchDayNight(night bool) {
	f := &s.npc
	if night == f.night {
		return
	}
	f.night = night
	f.dayOnly = slices.DeleteFunc(f.dayOnly, (*npcSpawn).deleted)
	f.nightOnly = slices.DeleteFunc(f.nightOnly, (*npcSpawn).deleted)
	gone, come := f.dayOnly, f.nightOnly
	if !night {
		gone, come = come, gone
	}
	removed, spawned := 0, 0
	for _, sp := range gone {
		sp.respawn = false // Mobius stopRespawn
		s.cancelRespawn(sp)
		if n := sp.npc; n != nil && n.InWorld() {
			s.RemoveObject(n)
			removed++
		}
	}
	for _, sp := range come {
		sp.respawn = true // Mobius startRespawn
		if s.spawnSlot(sp) != nil {
			spawned++
		}
	}
	s.log.Info("day/night spawns switched", "night", night, "removed", removed, "spawned", spawned)
}
