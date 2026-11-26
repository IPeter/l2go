package game

import (
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/sevensigns"
	"github.com/IPeter/l2go/internal/world"
)

// The Seven Signs NPCs (Mobius AutoSpawnHandler with the random_spawn
// groups, driven by SevenSigns.spawnSevenSignsNPC): the Merchant and the
// Blacksmith of Mammon move from place to place, the orators, preachers,
// crests, Lilith and Anakim stand while their seals allow it.

// Mobius AutoSpawnHandler's defaults for a delay of -1.
const (
	autoSpawnDefaultInitial = 30 * time.Second
	autoSpawnDefaultRespawn = time.Hour
	autoSpawnDefaultDespawn = time.Hour
)

// autoSpawnState is the auto spawn groups. Loop-only.
type autoSpawnState struct {
	groups []*autoSpawn
}

// autoSpawn is a running group (Mobius AutoSpawnInstance).
type autoSpawn struct {
	data    *data.AutoSpawn
	active  bool
	timer   world.TimerID
	nextAt  time.Time
	lastLoc int
	npcs    []*Npc
}

func autoDelay(ms int, def time.Duration) time.Duration {
	if ms < 0 {
		return def
	}
	return time.Duration(ms) * time.Millisecond
}

// autoSpawnGroups returns the groups of an NPC, made on first use.
func (s *Server) autoSpawnGroups(npcID int) []*autoSpawn {
	f := &s.sevenSigns
	if f.spawns.groups == nil && f.data != nil {
		for i := range f.data.AutoSpawns {
			f.spawns.groups = append(f.spawns.groups, &autoSpawn{data: &f.data.AutoSpawns[i], lastLoc: -1})
		}
	}
	var out []*autoSpawn
	for _, g := range f.spawns.groups {
		if g.data.NpcID == npcID {
			out = append(out, g)
		}
	}
	return out
}

// setAutoSpawnActive is AutoSpawnHandler.setSpawnActive for every group of
// the NPC: an active group spawns after its initial delay (and moves every
// respawn delay when it despawns), an inactive one despawns at once. Mobius
// only activates a group that is not active.
func (s *Server) setAutoSpawnActive(npcID int, active bool) {
	for _, g := range s.autoSpawnGroups(npcID) {
		if active == g.active && active {
			continue
		}
		s.cancelTimer(&g.timer)
		g.active = active
		if !active {
			s.autoDespawn(g)
			continue
		}
		s.scheduleAutoSpawn(g, autoDelay(g.data.InitialDelay, autoSpawnDefaultInitial))
	}
}

func (s *Server) scheduleAutoSpawn(g *autoSpawn, d time.Duration) {
	g.nextAt = s.Now().Add(d)
	g.timer = s.loop.After(d, func() {
		g.timer = 0
		if !g.active {
			return
		}
		s.autoSpawnRun(g)
		if g.data.DespawnDelay > 0 {
			s.scheduleAutoSpawn(g, autoDelay(g.data.RespawnDelay, autoSpawnDefaultRespawn))
		}
	})
}

// autoSpawnRun is AutoSpawner.run: the NPCs appear at the next (or a random)
// place of the group, the server is told when the group broadcasts, and they
// leave a second before the despawn delay.
func (s *Server) autoSpawnRun(g *autoSpawn) {
	locs := g.data.Locations
	if len(locs) == 0 {
		return
	}
	idx := s.rnd.Get(len(locs))
	if !g.data.RandomSpawn {
		idx = g.lastLoc + 1
		if idx >= len(locs) {
			idx = 0
		}
		g.lastLoc = idx
	}
	loc := locs[idx]
	heading := int32(loc.Heading) //nolint:gosec // headings
	if loc.Heading == -1 {
		heading = int32(s.rnd.Get(65536)) //nolint:gosec // headings
	}
	count := max(g.data.Count, 1)
	var last *Npc
	for range count {
		pos := world.Position{X: int32(loc.X), Y: int32(loc.Y), Z: int32(loc.Z)} //nolint:gosec // world coordinates
		if count > 1 {
			pos.X += int32(s.rnd.Get(50)) //nolint:gosec // small
			pos.Y += int32(s.rnd.Get(50)) //nolint:gosec // small
		}
		if n := s.spawnSiegeNpc(g.data.NpcID, pos, heading, ""); n != nil {
			g.npcs = append(g.npcs, n)
			last = n
		}
	}
	if last != nil && g.data.BroadcastSpawn {
		town := ""
		if w := s.worldData; w != nil {
			pos := last.Position()
			if r, ok := w.MapRegionAt(int(pos.X), int(pos.Y)); ok {
				town = r.Town
			}
		}
		text := "The " + last.Template.Name + " has spawned near " + town + "!"
		for p := range s.Players() {
			if p.state == stateInGame {
				p.SendMessage(text)
			}
		}
	}
	if g.data.DespawnDelay > 0 {
		s.loop.After(autoDelay(g.data.DespawnDelay, autoSpawnDefaultDespawn)-time.Second, func() { s.autoDespawn(g) })
	}
}

// autoDespawn is AutoDespawner.run.
func (s *Server) autoDespawn(g *autoSpawn) {
	for _, n := range g.npcs {
		if n.InWorld() {
			s.DespawnNpc(n)
		}
	}
	g.npcs = nil
}

// autoSpawnNpc returns the first NPC of the NPC's first group in the world,
// or nil (AutoSpawnInstance.getNpcList().peek()).
func (s *Server) autoSpawnNpc(npcID int) *Npc {
	for _, g := range s.autoSpawnGroups(npcID) {
		for _, n := range g.npcs {
			if n.InWorld() {
				return n
			}
		}
		return nil
	}
	return nil
}

// spawnSevenSignsNpcs is spawnSevenSignsNPC: in the results and seal
// validation periods the Black Marketeer stands; the Blacksmith of Mammon,
// the orators and the preachers while the winner owns the Seal of Gnosis;
// the Merchant of Mammon and the winner's crests and boss (Lilith for Dawn,
// Anakim for Dusk) while it owns the Seal of Avarice. In the other periods
// none of them.
func (s *Server) spawnSevenSignsNpcs() {
	st := s.sevenSigns.state
	set := func(active bool, ids ...int) {
		for _, id := range ids {
			s.setAutoSpawnActive(id, active)
		}
	}
	if !st.IsSealValidation() && !st.IsResults() {
		set(false, sevensigns.MammonMerchantID, sevensigns.MammonBlacksmithID, sevensigns.LilithNpcID, sevensigns.AnakimNpcID,
			sevensigns.CrestOfDawnID, sevensigns.CrestOfDuskID, sevensigns.OratorNpcID, sevensigns.PreacherNpcID, sevensigns.MammonMarketeerID)
		return
	}
	winner := st.HighestCabal()
	set(true, sevensigns.MammonMarketeerID)
	gnosis := st.SealOwner(sevensigns.SealGnosis)
	gnosisWon := gnosis == winner && gnosis != sevensigns.CabalNull
	set(gnosisWon, sevensigns.MammonBlacksmithID, sevensigns.OratorNpcID, sevensigns.PreacherNpcID)
	avarice := st.SealOwner(sevensigns.SealAvarice)
	if avarice != winner || avarice == sevensigns.CabalNull {
		set(false, sevensigns.MammonMerchantID, sevensigns.LilithNpcID, sevensigns.AnakimNpcID, sevensigns.CrestOfDawnID, sevensigns.CrestOfDuskID)
		return
	}
	set(true, sevensigns.MammonMerchantID)
	switch winner {
	case sevensigns.CabalDawn:
		set(true, sevensigns.LilithNpcID, sevensigns.CrestOfDawnID)
		set(false, sevensigns.AnakimNpcID, sevensigns.CrestOfDuskID)
	case sevensigns.CabalDusk:
		set(true, sevensigns.AnakimNpcID, sevensigns.CrestOfDuskID)
		set(false, sevensigns.LilithNpcID, sevensigns.CrestOfDawnID)
	}
}
