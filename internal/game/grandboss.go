package game

import (
	"context"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// Grand boss state (Mobius GrandBossManager): each grand boss has a status,
// whose meaning its script decides, and stored info (location, heading,
// respawn time, HP and MP) the script reads and writes. The game seeds the
// stored state once from the data (bosses/grandbosses.json), writes the
// status at once when it changes, the whole boss when a script stores its
// info, and everything (with the boss zones' allowed players) every 5
// minutes and at shutdown. Scripts spawn the bosses and register them with
// AddGrandBoss, which makes the stored info follow the live boss.

// grandBossStoreInterval is GrandBossManager's storeMe period.
const grandBossStoreInterval = 5 * time.Minute

// GrandBossInfo is the stored info of a grand boss (Mobius's StatSet of
// loc_x, loc_y, loc_z, heading, respawn_time, currentHP and currentMP).
type GrandBossInfo struct {
	X, Y, Z, Heading int32
	// RespawnTime is when the boss comes back, as its script set it.
	RespawnTime  time.Time
	CurHP, CurMP float64
}

// grandBossEntry is one grand boss of the manager.
type grandBossEntry struct {
	info   GrandBossInfo
	status int32
	boss   *Npc // Mobius BOSSES: the live boss a script registered, or nil
}

// grandBossSeeds converts the grand boss seeds of the data.
func (s *Server) grandBossSeeds() []storage.GrandBossState {
	if s.bosses.data == nil {
		return nil
	}
	var out []storage.GrandBossState
	for _, g := range s.bosses.data.GrandBosses {
		out = append(out, storage.GrandBossState{
			BossID: int32(g.BossID), X: int32(g.X), Y: int32(g.Y), Z: int32(g.Z), Heading: int32(g.Heading), //nolint:gosec // IDs and coordinates fit
			RespawnTime: g.RespawnTime, CurHP: g.HP, CurMP: g.MP, Status: int32(g.Status), //nolint:gosec // statuses are small
		})
	}
	return out
}

// setupGrandBosses keeps the stored grand bosses and starts the 5 minute
// store task (Mobius GrandBossManager.init).
func (s *Server) setupGrandBosses(stored []storage.GrandBossState) {
	f := &s.bosses
	for _, g := range stored {
		e := &grandBossEntry{status: g.Status, info: GrandBossInfo{
			X: g.X, Y: g.Y, Z: g.Z, Heading: g.Heading, CurHP: g.CurHP, CurMP: g.CurMP,
		}}
		if g.RespawnTime != 0 {
			e.info.RespawnTime = time.UnixMilli(g.RespawnTime)
		}
		f.grand[g.BossID] = e
		f.grandOrder = append(f.grandOrder, g.BossID)
	}
	slices.Sort(f.grandOrder)
	if len(stored) > 0 {
		s.log.Info("grand bosses loaded", "bosses", len(stored))
	}
	f.storeTimer = s.loop.Every(grandBossStoreInterval, s.grandBossStoreMe)
}

// GrandBossStatus returns a grand boss's status (Mobius getStatus), false
// for a boss the manager does not know. Loop-only.
func (s *Server) GrandBossStatus(bossID int) (int, bool) {
	e := s.bosses.grand[int32(bossID)] //nolint:gosec // NPC IDs are small
	if e == nil {
		return 0, false
	}
	return int(e.status), true
}

// SetGrandBossStatus changes a grand boss's status and writes it at once
// (Mobius setStatus). Loop-only.
func (s *Server) SetGrandBossStatus(bossID, status int) error {
	id := int32(bossID) //nolint:gosec // NPC IDs are small
	e := s.bosses.grand[id]
	if e == nil {
		return errBossUnknown
	}
	e.status = int32(status) //nolint:gosec // statuses are small
	s.log.Info("grand boss status updated", "boss", bossID, "status", status)
	s.grandBossUpdateDB(id, true)
	return nil
}

// GrandBossInfoOf returns a grand boss's stored info (Mobius getStatSet),
// false for a boss the manager does not know. Loop-only.
func (s *Server) GrandBossInfoOf(bossID int) (GrandBossInfo, bool) {
	e := s.bosses.grand[int32(bossID)] //nolint:gosec // NPC IDs are small
	if e == nil {
		return GrandBossInfo{}, false
	}
	return e.info, true
}

// SetGrandBossInfo replaces a grand boss's stored info and writes the boss
// (Mobius setStatSet): with a live boss registered, its location, heading
// and HP and MP (full when it is dead) with the info's respawn time; without
// one, only the status, as Mobius does. Loop-only.
func (s *Server) SetGrandBossInfo(bossID int, info GrandBossInfo) error {
	id := int32(bossID) //nolint:gosec // NPC IDs are small
	e := s.bosses.grand[id]
	if e == nil {
		return errBossUnknown
	}
	e.info = info
	s.grandBossUpdateDB(id, false)
	return nil
}

// AddGrandBoss registers the live grand boss a script spawned (Mobius
// addBoss): the stores follow it from now on. Loop-only.
func (s *Server) AddGrandBoss(n *Npc) {
	if n == nil {
		return
	}
	id := int32(n.Template.ID) //nolint:gosec // NPC IDs are small
	e := s.bosses.grand[id]
	if e == nil {
		// Mobius keeps a boss without stored info too; storeMe skips it.
		e = &grandBossEntry{}
		s.bosses.grand[id] = e
		s.bosses.grandOrder = append(s.bosses.grandOrder, id)
		slices.Sort(s.bosses.grandOrder)
		e.status = -1 // unknown to the store
	}
	e.boss = n
}

// GrandBoss returns the live grand boss registered with AddGrandBoss, or
// nil (Mobius getBoss). Loop-only.
func (s *Server) GrandBoss(bossID int) *Npc {
	if e := s.bosses.grand[int32(bossID)]; e != nil { //nolint:gosec // NPC IDs are small
		return e.boss
	}
	return nil
}

// grandBossRow is the whole stored row of a boss with a live boss
// registered (Mobius UPDATE_GRAND_BOSS_DATA): the live boss's location and
// HP, full when it is dead.
func grandBossRow(id int32, e *grandBossEntry) storage.GrandBossState {
	n := e.boss
	pos := n.Position()
	hp, mp := n.CurHP, n.CurMP
	if n.Dead {
		hp, mp = float64(n.MaxHP()), float64(n.MaxMP())
	}
	row := storage.GrandBossState{BossID: id, X: pos.X, Y: pos.Y, Z: pos.Z, Heading: pos.Heading, CurHP: hp, CurMP: mp, Status: e.status}
	if !e.info.RespawnTime.IsZero() {
		row.RespawnTime = e.info.RespawnTime.UnixMilli()
	}
	return row
}

// grandBossUpdateDB is Mobius GrandBossManager.updateDb: the status only,
// or, with a live boss, the whole row.
func (s *Server) grandBossUpdateDB(id int32, statusOnly bool) {
	f := &s.bosses
	e := f.grand[id]
	store := f.store
	if e == nil || e.status < 0 || store == nil {
		return
	}
	if statusOnly || e.boss == nil {
		status := e.status
		f.writer.enqueue("grand boss status", func(ctx context.Context) error { return store.SaveGrandBossStatus(ctx, id, status) })
		return
	}
	row := grandBossRow(id, e)
	f.writer.enqueue("grand boss", func(ctx context.Context) error {
		return store.SaveGrandBosses(ctx, []storage.GrandBossState{row})
	})
}

// grandBossStoreMe is Mobius GrandBossManager.storeMe: the boss zones'
// allowed players replace the stored ones, and every grand boss is written,
// whole with a live boss, its status otherwise.
func (s *Server) grandBossStoreMe() {
	f := &s.bosses
	store := f.store
	if store == nil {
		return
	}
	players := s.bossZonePlayerRows()
	type status struct{ id, status int32 }
	var statuses []status
	var rows []storage.GrandBossState
	for _, id := range f.grandOrder {
		e := f.grand[id]
		switch {
		case e.status < 0:
		case e.boss == nil:
			statuses = append(statuses, status{id, e.status})
		default:
			rows = append(rows, grandBossRow(id, e))
		}
	}
	f.writer.enqueue("grand bosses", func(ctx context.Context) error {
		if err := store.SaveBossZonePlayers(ctx, players); err != nil {
			return err
		}
		for _, st := range statuses {
			if err := store.SaveGrandBossStatus(ctx, st.id, st.status); err != nil {
				return err
			}
		}
		if len(rows) == 0 {
			return nil
		}
		return store.SaveGrandBosses(ctx, rows)
	})
}
