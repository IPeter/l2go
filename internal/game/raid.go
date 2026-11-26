package game

import (
	"context"
	"errors"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The boss feature (Plan 13, decisions A8 and A9):
//
//   - raid.go: the feature, the raid boss spawner (Mobius
//     RaidBossSpawnManager), the RaidBoss and GrandBoss death (the success
//     message, raid points) and the raid curse test.
//   - raid_points.go: raid points and their weekly reset
//     (RaidBossPointsManager).
//   - raid_store.go: the background writer of the boss state.
//   - grandboss.go: the grand boss state (GrandBossManager).
//   - bosszone.go: the boss zones (BossZone).
//   - grandboss_gm.go: //grandboss and its commands (AdminGrandBoss).
//   - minion.go: the minions (MinionList).
//
// The boss state lives on the loop; the database is read once at startup
// and written in the background, in order (raid_store.go), never on the
// loop: a raid boss's respawn time and the HP and MP of the others when one
// dies and at shutdown; a grand boss's status at once when it changes, the
// whole boss when a script stores its state, everything every 5 minutes and
// at shutdown; raid points when they change.
func init() { registerFeature("raidboss", setupBosses) }

// System messages of the boss feature.
const (
	// sysMsgRaidSuccessful is "Congratulations. Your raid was successful."
	// (Mobius CONGRATULATIONS_YOUR_RAID_WAS_SUCCESSFUL).
	sysMsgRaidSuccessful serverpackets.SystemMessageID = 1209
)

// bossLoadTimeout bounds reading the boss state at startup.
const bossLoadTimeout = 30 * time.Second

// bossFeature is the boss feature's state on the Server (Server.bosses).
// Loop-only, except data and store, which New sets, and writer.
type bossFeature struct {
	data  *data.Bosses
	store storage.BossStore // nil keeps the state in memory only

	writer *bossWriter
	// started is when the server started (Mobius GameServer.getStartTime),
	// for the boss zones' re-entry after a restart.
	started time.Time
	// minionTimes are npcs.customMinionsRespawnTime.
	minionTimes map[int]time.Duration

	raids    []*raidEntry
	raidByID map[int]*raidEntry

	grand      map[int32]*grandBossEntry
	grandOrder []int32 // boss IDs, sorted
	storeTimer world.TimerID

	zones     map[int]*bossZone
	zoneOrder []int // zone IDs, sorted

	points      map[int32]map[int32]int32 // character → boss → points
	resetTimer  world.TimerID
	resetHooks  []func(ranking map[int32]int)
	shutdownRan bool
}

// setupBosses reads the boss state, makes the boss zones, spawns the raid
// bosses (or schedules their respawn) and registers the hooks, timers and
// GM commands.
func setupBosses(s *Server) error {
	f := &s.bosses
	f.started = s.Now()
	f.writer = newBossWriter(s.log)
	f.raidByID = make(map[int]*raidEntry)
	f.grand = make(map[int32]*grandBossEntry)
	f.zones = make(map[int]*bossZone)
	f.points = make(map[int32]map[int32]int32)
	times, err := s.settings.NPCs.MinionRespawnTimes()
	if err != nil {
		s.log.Warn("game: npcs.customMinionsRespawnTime is not valid; no custom minion respawn times", "err", err)
		times = nil
	}
	f.minionTimes = times

	st, err := s.loadBossState()
	if err != nil {
		return err
	}
	s.setupGrandBosses(st.grand)
	s.setupBossZones(st.zones)
	s.setupRaidPoints(st.points)
	s.setupRaidSpawns(st.raids)

	s.OnPlayerLeave(s.bossZonesLeave)
	return s.registerGrandBossGM()
}

// bossState is the boss state read at startup.
type bossState struct {
	raids  []storage.RaidBossState
	grand  []storage.GrandBossState
	zones  []storage.BossZonePlayer
	points []storage.RaidPoints
}

// loadBossState seeds the grand bosses (Mobius's INSERT IGNORE of
// grandboss_data) and reads the boss state, once, before the loop runs.
func (s *Server) loadBossState() (bossState, error) {
	f := &s.bosses
	var st bossState
	if f.store == nil {
		st.grand = s.grandBossSeeds()
		return st, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), bossLoadTimeout)
	defer cancel()
	var err error
	if seeds := s.grandBossSeeds(); len(seeds) > 0 {
		if err = f.store.SeedGrandBosses(ctx, seeds); err != nil {
			return st, err
		}
	}
	if st.raids, err = f.store.LoadRaidBosses(ctx); err != nil {
		return st, err
	}
	if st.grand, err = f.store.LoadGrandBosses(ctx); err != nil {
		return st, err
	}
	if st.zones, err = f.store.LoadBossZonePlayers(ctx); err != nil {
		return st, err
	}
	if st.points, err = f.store.LoadRaidPoints(ctx); err != nil {
		return st, err
	}
	return st, nil
}

// saveBossesAtShutdown writes the whole boss state when the server stops
// (Mobius Shutdown: RaidBossSpawnManager.cleanUp, GrandBossManager.cleanUp)
// and waits for the writes.
func (s *Server) saveBossesAtShutdown() {
	f := &s.bosses
	if f.writer == nil || f.shutdownRan {
		return
	}
	f.shutdownRan = true
	s.raidUpdateDB()
	s.grandBossStoreMe()
	ctx, cancel := context.WithTimeout(s.background(), shutdownTimeout)
	defer cancel()
	if err := f.writer.wait(ctx); err != nil {
		s.log.Error("game: the boss state is not saved at shutdown", "err", err)
	}
}

// ----- the raid spawner (Mobius RaidBossSpawnManager) -----

// RaidStatus is a raid boss's status (Mobius RaidBossStatus).
type RaidStatus uint8

// The raid statuses.
const (
	RaidUndefined RaidStatus = iota
	RaidAlive
	RaidDead
)

// String returns Mobius's name of the status.
func (r RaidStatus) String() string {
	switch r {
	case RaidAlive:
		return "ALIVE"
	case RaidDead:
		return "DEAD"
	}
	return "UNDEFINED"
}

// raidEntry is one raid boss of the spawner.
type raidEntry struct {
	spawn *data.RaidSpawn
	slot  *npcSpawn
	// boss is the raid boss once it spawned (Mobius _bosses).
	boss   *Npc
	status RaidStatus
	// info is Mobius _storedInfo: the HP, MP and respawn time to store;
	// hasInfo is set once the boss spawned.
	info    storage.RaidBossState
	hasInfo bool
	// timer is the pending respawn (Mobius _schedules), or 0.
	timer world.TimerID
	// moved is set once a GM moved the boss's spawn (//recall_npc): the
	// slot's point is stored with the state.
	moved bool
}

// stored returns st with the moved spawn point of e, if any.
func (e *raidEntry) stored(st storage.RaidBossState) storage.RaidBossState {
	if e.moved {
		pt := e.slot.point
		st.Moved, st.X, st.Y, st.Z, st.Heading = true, pt.X, pt.Y, pt.Z, pt.Heading
	}
	return st
}

// id is the boss's NPC ID as the store keeps it.
func (e *raidEntry) id() int32 { return int32(e.spawn.BossID) } //nolint:gosec // NPC IDs are small

// setupRaidSpawns makes a spawn slot for every raid boss of the data and
// spawns it, or schedules its stored respawn (Mobius addNewSpawn): a boss
// without a respawn time, or whose time has passed, spawns now with the
// stored HP and MP (the data's on the first start). A boss a GM moved
// (moveRaidSpawn) spawns at its stored point.
func (s *Server) setupRaidSpawns(stored []storage.RaidBossState) {
	f := &s.bosses
	if f.data == nil || s.worldData == nil {
		return
	}
	byID := make(map[int32]storage.RaidBossState, len(stored))
	for _, r := range stored {
		byID[r.BossID] = r
	}
	spawned, scheduled := 0, 0
	for i := range f.data.RaidSpawns {
		r := &f.data.RaidSpawns[i]
		t, err := s.spawnableTemplate(r.BossID)
		if err != nil {
			s.log.Warn("game: raid boss left out", "boss", r.BossID, "err", err)
			continue
		}
		lo, hi := r.RespawnWindow()
		sp := &npcSpawn{tmpl: t, respawnMin: lo, respawnMax: hi, index: -1,
			point: world.Position{X: int32(r.X), Y: int32(r.Y), Z: int32(r.Z), Heading: int32(r.Heading)}} //nolint:gosec // world coordinates fit
		st, ok := byID[int32(r.BossID)] //nolint:gosec // NPC IDs are small
		if !ok {
			st = storage.RaidBossState{BossID: int32(r.BossID), CurHP: r.HP, CurMP: r.MP} //nolint:gosec // NPC IDs are small
		}
		if st.Moved {
			sp.point = world.Position{X: st.X, Y: st.Y, Z: st.Z, Heading: st.Heading}
		}
		s.addSpawnSlot(sp)
		e := &raidEntry{spawn: r, slot: sp, moved: st.Moved}
		f.raids = append(f.raids, e)
		f.raidByID[r.BossID] = e
		now := s.Now().UnixMilli()
		if st.RespawnTime == 0 || now > st.RespawnTime {
			if n := s.spawnSlot(sp); n != nil {
				n.boss.raid = e
				n.SetCurrentHPMP(st.CurHP, st.CurMP)
				e.boss, e.status = n, RaidAlive
				e.info, e.hasInfo = storage.RaidBossState{BossID: st.BossID, CurHP: st.CurHP, CurMP: st.CurMP}, true
				spawned++
			}
			continue
		}
		s.scheduleRaidSpawn(e, time.Duration(st.RespawnTime-now)*time.Millisecond)
		scheduled++
	}
	s.log.Info("raid bosses loaded", "bosses", len(f.raids), "spawned", spawned, "scheduled", scheduled)
}

// scheduleRaidSpawn is Mobius SpawnSchedule: after d the boss spawns with
// full HP and MP and is ALIVE, with no respawn time to store.
func (s *Server) scheduleRaidSpawn(e *raidEntry, d time.Duration) {
	e.timer = s.loop.After(d, func() {
		e.timer = 0
		if e.slot.deleted() {
			return
		}
		n := s.spawnSlot(e.slot)
		if n == nil {
			return
		}
		n.boss.raid = e
		e.boss, e.status = n, RaidAlive
		e.info, e.hasInfo = storage.RaidBossState{BossID: e.id(), CurHP: n.CurHP, CurMP: n.CurMP}, true
		s.log.Info("spawning raid boss", "boss", n.Template.ID, "name", n.Name())
	})
}

// raidBossDied is Mobius RaidBossSpawnManager.updateStatus(boss, true):
// the boss is DEAD and comes back after a uniform delay between the
// respawn window's ends, each times its multiplier, in milliseconds; the
// state of every raid boss is written.
func (s *Server) raidBossDied(n *Npc) {
	e := n.boss.raid
	if e == nil || !e.hasInfo {
		return
	}
	e.status = RaidDead
	lo, hi := e.slot.respawnMin, e.slot.respawnMax
	minMs := int(float64(lo.Milliseconds()) * s.settings.NPCs.RaidMinRespawnMultiplier)
	maxMs := int(float64(hi.Milliseconds()) * s.settings.NPCs.RaidMaxRespawnMultiplier)
	delay := time.Duration(s.rnd.Between(minMs, maxMs)) * time.Millisecond
	at := s.Now().Add(delay)
	e.info = storage.RaidBossState{BossID: e.id(), RespawnTime: at.UnixMilli(), CurHP: float64(n.MaxHP()), CurMP: float64(n.MaxMP())}
	if e.timer == 0 && (minMs > 0 || maxMs > 0) {
		s.log.Info("raid boss respawn time updated", "boss", n.Template.ID, "name", n.Name(), "respawn", at)
		s.scheduleRaidSpawn(e, delay)
		s.raidUpdateDB()
	}
}

// raidUpdateDB is Mobius RaidBossSpawnManager.updateDb: the state of every
// raid boss that spawned since the start, a living one with its current HP
// and MP, a dead one with full HP and MP, written in the background.
func (s *Server) raidUpdateDB() {
	var rows []storage.RaidBossState
	for _, e := range s.bosses.raids {
		n := e.boss
		if n == nil || !e.hasInfo || e.slot.deleted() {
			continue // a GM's //delete is Mobius deleteSpawn
		}
		if e.status == RaidAlive {
			e.info = storage.RaidBossState{BossID: e.id(), CurHP: n.CurHP, CurMP: n.CurMP}
		}
		row := e.stored(e.info)
		if n.Dead {
			row.CurHP, row.CurMP = float64(n.MaxHP()), float64(n.MaxMP())
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 || s.bosses.store == nil {
		return
	}
	store := s.bosses.store
	s.bosses.writer.enqueue("raid bosses", func(ctx context.Context) error { return store.SaveRaidBosses(ctx, rows) })
}

// RaidBossStatus returns the status of a raid boss of the spawner (Mobius
// getRaidBossStatusId): its own once it spawned, DEAD while its first
// respawn waits, UNDEFINED for a boss the spawner does not know. Loop-only.
func (s *Server) RaidBossStatus(bossID int) RaidStatus {
	e := s.bosses.raidByID[bossID]
	switch {
	case e == nil:
		return RaidUndefined
	case e.boss != nil:
		return e.status
	case e.timer != 0:
		return RaidDead
	}
	return RaidUndefined
}

// RaidBoss returns the raid boss of the spawner with the given NPC ID once
// it spawned (alive or dead), or nil. Loop-only.
func (s *Server) RaidBoss(bossID int) *Npc {
	if e := s.bosses.raidByID[bossID]; e != nil {
		return e.boss
	}
	return nil
}

// RaidBossRespawnAt returns when a dead raid boss of the spawner comes
// back, and false when none is pending. Loop-only.
func (s *Server) RaidBossRespawnAt(bossID int) (time.Time, bool) {
	e := s.bosses.raidByID[bossID]
	if e == nil || e.timer == 0 || e.info.RespawnTime == 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(e.info.RespawnTime), true
}

// moveRaidSpawn is the RaidBoss branch of Mobius AdminTeleport.recallNPC: the
// spawner's spawn of n is deleted (RaidBossSpawnManager.deleteSpawn: a
// pending respawn is cancelled) and a new one at pos spawns the boss there at
// once with n's HP and MP (addNewSpawn with no respawn time), ALIVE, and is
// stored with its point (Mobius's raidboss_spawnlist row), so the boss
// spawns there after a restart too. It returns the new boss, or nil when n
// is not a boss of the spawner or the boss could not be placed.
//
// Unlike Mobius, the old boss leaves the world (Mobius leaves it standing,
// no longer the spawner's, beside the new one), a dead boss comes back with
// full HP and MP rather than none, and the new spawn keeps the boss's
// respawn window: Mobius sets 43200 and 129600 "seconds" that its spawn
// reads as milliseconds, and after a restart its table's default of 36 h ±
// 24 h, the data's window of most bosses. Loop-only.
func (s *Server) moveRaidSpawn(n *Npc, pos world.Position) *Npc {
	e := n.boss.raid
	if e == nil || e.slot.deleted() || e.slot != n.spawn {
		return nil
	}
	hp, mp := n.CurHP, n.CurMP
	if n.Dead {
		hp, mp = float64(n.MaxHP()), float64(n.MaxMP())
	}
	old := e.slot
	if e.timer != 0 {
		s.loop.Cancel(e.timer)
		e.timer = 0
	}
	s.deleteSpawn(old)
	sp := &npcSpawn{tmpl: old.tmpl, respawnMin: old.respawnMin, respawnMax: old.respawnMax, point: pos, index: -1}
	s.addSpawnSlot(sp)
	e.slot, e.boss, e.status, e.info, e.hasInfo, e.moved = sp, nil, RaidUndefined, storage.RaidBossState{}, false, true
	b := s.spawnSlot(sp)
	if b == nil {
		return nil
	}
	b.boss.raid = e
	b.SetCurrentHPMP(hp, mp)
	e.boss, e.status = b, RaidAlive
	e.info, e.hasInfo = storage.RaidBossState{BossID: e.id(), CurHP: b.CurHP, CurMP: b.CurMP}, true
	if store := s.bosses.store; store != nil {
		row := e.stored(e.info)
		s.bosses.writer.enqueue("raid boss spawn", func(ctx context.Context) error {
			return store.SaveRaidBosses(ctx, []storage.RaidBossState{row})
		})
	}
	return b
}

// ----- RaidBoss and GrandBoss -----

// bossNpcDied is the end of Mobius doDie for the bosses: the minion hooks
// of Attackable.doDie, then for a RaidBoss or a GrandBoss killed by a
// player the success message to the killer's party (or the killer) and raid
// points to each of them, and for a raid boss of the spawner its respawn.
func (s *Server) bossNpcDied(n *Npc, killer Creature) {
	s.minionsOnDie(n)
	if n.class == nil || !n.class.raid {
		return
	}
	if p, ok := killer.(*Player); ok && p != nil {
		msg := sysMsgPacket(sysMsgRaidSuccessful)
		if pt := p.Party(); pt != nil {
			pt.Broadcast(msg)
			for _, m := range pt.Members() {
				s.AddRaidPoints(m, n.Template.ID, n.Level()/2+s.rnd.Between(-5, 5))
			}
		} else {
			p.Send(msg)
			s.AddRaidPoints(p, n.Template.ID, n.Level()/2+s.rnd.Between(-5, 5))
		}
		// Hero.setRBkilled for nobles comes with the Olympiad.
	}
	s.raidBossDied(n)
}

// raidCurses reports whether c curses those too far above it (Mobius
// isRaid() && giveRaidCurse() && !RAID_DISABLE_CURSE): a raid or grand
// boss whose curse a script did not turn off, or a raid minion of such a
// master.
func (s *Server) raidCurses(c Creature) bool {
	if s.settings.NPCs.DisableRaidCurse || !creatureRaid(c) {
		return false
	}
	n, _ := c.(*Npc)
	return giveRaidCurse(n)
}

// giveRaidCurse is Mobius giveRaidCurse: a raid minion asks its master.
func giveRaidCurse(n *Npc) bool {
	if n == nil {
		return true
	}
	if n.boss.raidMinion && n.boss.leader != nil {
		return giveRaidCurse(n.boss.leader)
	}
	if n.class != nil && n.class.raid {
		return !n.boss.noRaidCurse
	}
	return true
}

// SetUseRaidCurse turns a raid or grand boss's curse on or off (Mobius
// setUseRaidCurse). Loop-only.
func (s *Server) SetUseRaidCurse(n *Npc, on bool) { n.boss.noRaidCurse = !on }

// errBossUnknown means a boss ID the boss data does not know.
var errBossUnknown = errors.New("game: unknown boss")
