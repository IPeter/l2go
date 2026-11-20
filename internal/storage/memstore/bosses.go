package memstore

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/IPeter/l2go/internal/storage"
)

// Bosses is an in-memory storage.BossStore. The zero value is ready to use
// and it is safe for concurrent use.
type Bosses struct {
	mu     sync.Mutex
	raids  map[int32]storage.RaidBossState
	grands map[int32]storage.GrandBossState
	zones  []storage.BossZonePlayer
	points map[[2]int32]int32 // {character, boss} → points

	// failNext, if set, fails the next write before it changes anything.
	// Tests in this package use it to run the shared failed-write tests.
	failNext error
}

var _ storage.BossStore = (*Bosses)(nil)

// NewBosses returns an empty store.
func NewBosses() *Bosses { return &Bosses{} }

// FailNext makes the next write fail with err before it changes anything.
// For tests of the callers.
func (s *Bosses) FailNext(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = err
}

// write runs fn under the lock unless ctx ended or a failure is armed.
func (s *Bosses) write(ctx context.Context, fn func()) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failNext; err != nil {
		s.failNext = nil
		return err
	}
	fn()
	return nil
}

// read runs fn under the lock unless ctx ended.
func (s *Bosses) read(ctx context.Context, fn func()) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fn()
	return nil
}

// sortedValues returns the values of m in key order, nil for none.
func sortedValues[V any](m map[int32]V) []V {
	if len(m) == 0 {
		return nil
	}
	out := make([]V, 0, len(m))
	for _, k := range slices.Sorted(maps.Keys(m)) {
		out = append(out, m[k])
	}
	return out
}

// LoadRaidBosses implements storage.BossStore.
func (s *Bosses) LoadRaidBosses(ctx context.Context) (out []storage.RaidBossState, err error) {
	err = s.read(ctx, func() { out = sortedValues(s.raids) })
	return out, err
}

// SaveRaidBosses implements storage.BossStore.
func (s *Bosses) SaveRaidBosses(ctx context.Context, bosses []storage.RaidBossState) error {
	return s.write(ctx, func() {
		if s.raids == nil {
			s.raids = make(map[int32]storage.RaidBossState)
		}
		for _, b := range bosses {
			s.raids[b.BossID] = b
		}
	})
}

// SeedGrandBosses implements storage.BossStore.
func (s *Bosses) SeedGrandBosses(ctx context.Context, seeds []storage.GrandBossState) error {
	return s.write(ctx, func() {
		if s.grands == nil {
			s.grands = make(map[int32]storage.GrandBossState)
		}
		for _, g := range seeds {
			if _, ok := s.grands[g.BossID]; !ok {
				s.grands[g.BossID] = g
			}
		}
	})
}

// LoadGrandBosses implements storage.BossStore.
func (s *Bosses) LoadGrandBosses(ctx context.Context) (out []storage.GrandBossState, err error) {
	err = s.read(ctx, func() { out = sortedValues(s.grands) })
	return out, err
}

// SaveGrandBossStatus implements storage.BossStore.
func (s *Bosses) SaveGrandBossStatus(ctx context.Context, bossID, status int32) error {
	return s.write(ctx, func() {
		if g, ok := s.grands[bossID]; ok {
			g.Status = status
			s.grands[bossID] = g
		}
	})
}

// SaveGrandBosses implements storage.BossStore.
func (s *Bosses) SaveGrandBosses(ctx context.Context, bosses []storage.GrandBossState) error {
	return s.write(ctx, func() {
		if s.grands == nil {
			s.grands = make(map[int32]storage.GrandBossState)
		}
		for _, g := range bosses {
			s.grands[g.BossID] = g
		}
	})
}

// LoadBossZonePlayers implements storage.BossStore.
func (s *Bosses) LoadBossZonePlayers(ctx context.Context) (out []storage.BossZonePlayer, err error) {
	err = s.read(ctx, func() { out = slices.Clone(s.zones) })
	return out, err
}

// SaveBossZonePlayers implements storage.BossStore.
func (s *Bosses) SaveBossZonePlayers(ctx context.Context, rows []storage.BossZonePlayer) error {
	sorted := slices.Clone(rows)
	slices.SortFunc(sorted, func(a, b storage.BossZonePlayer) int {
		return cmp.Or(cmp.Compare(a.Zone, b.Zone), cmp.Compare(a.Player, b.Player))
	})
	sorted = slices.Compact(sorted)
	if len(sorted) == 0 {
		sorted = nil
	}
	return s.write(ctx, func() { s.zones = sorted })
}

// LoadRaidPoints implements storage.BossStore.
func (s *Bosses) LoadRaidPoints(ctx context.Context) (out []storage.RaidPoints, err error) {
	err = s.read(ctx, func() {
		keys := slices.SortedFunc(maps.Keys(s.points), func(a, b [2]int32) int {
			return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1]))
		})
		for _, k := range keys {
			out = append(out, storage.RaidPoints{CharID: k[0], BossID: k[1], Points: s.points[k]})
		}
	})
	return out, err
}

// SaveRaidPoints implements storage.BossStore.
func (s *Bosses) SaveRaidPoints(ctx context.Context, rows []storage.RaidPoints) error {
	return s.write(ctx, func() {
		if s.points == nil {
			s.points = make(map[[2]int32]int32)
		}
		for _, r := range rows {
			s.points[[2]int32{r.CharID, r.BossID}] = r.Points
		}
	})
}

// ClearRaidPoints implements storage.BossStore.
func (s *Bosses) ClearRaidPoints(ctx context.Context) error {
	return s.write(ctx, func() { s.points = nil })
}
