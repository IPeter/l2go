package storagetest

import (
	"errors"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// TestBossStore runs the BossStore conformance tests. newStore must return an
// empty store for each call, and a function that makes the store's next write
// fail with the given error after it did all its work, just before it
// commits.
func TestBossStore(t *testing.T, newStore func(t *testing.T) (s storage.BossStore, failNext func(error))) {
	kutus := storage.RaidBossState{BossID: 25001, RespawnTime: 1795000000000, CurHP: 95986, CurMP: 514}
	// A raid boss a GM moved (//recall_npc) keeps its spawn point.
	tyrannosaurus := storage.RaidBossState{BossID: 25019, CurHP: 120000.5, CurMP: 600.25,
		Moved: true, X: 117000, Y: -16500, Z: -1900, Heading: 32768}
	queenAnt := storage.GrandBossState{BossID: 29001, X: -21610, Y: 181594, Z: -5734, CurHP: 229898.48, CurMP: 667.776}
	core := storage.GrandBossState{BossID: 29006, X: 17726, Y: 108915, Z: -6480, CurHP: 622493.58388, CurMP: 3793.536}

	raids := func(t *testing.T, s storage.BossStore) []storage.RaidBossState {
		t.Helper()
		got, err := s.LoadRaidBosses(t.Context())
		if err != nil {
			t.Fatalf("LoadRaidBosses: %v", err)
		}
		return got
	}
	grands := func(t *testing.T, s storage.BossStore) []storage.GrandBossState {
		t.Helper()
		got, err := s.LoadGrandBosses(t.Context())
		if err != nil {
			t.Fatalf("LoadGrandBosses: %v", err)
		}
		return got
	}
	zones := func(t *testing.T, s storage.BossStore) []storage.BossZonePlayer {
		t.Helper()
		got, err := s.LoadBossZonePlayers(t.Context())
		if err != nil {
			t.Fatalf("LoadBossZonePlayers: %v", err)
		}
		return got
	}
	points := func(t *testing.T, s storage.BossStore) []storage.RaidPoints {
		t.Helper()
		got, err := s.LoadRaidPoints(t.Context())
		if err != nil {
			t.Fatalf("LoadRaidPoints: %v", err)
		}
		return got
	}
	must := func(t *testing.T, what string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}

	t.Run("Empty", func(t *testing.T) {
		s, _ := newStore(t)
		if got := raids(t, s); got != nil {
			t.Errorf("LoadRaidBosses = %+v, want nil", got)
		}
		if got := grands(t, s); got != nil {
			t.Errorf("LoadGrandBosses = %+v, want nil", got)
		}
		if got := zones(t, s); got != nil {
			t.Errorf("LoadBossZonePlayers = %+v, want nil", got)
		}
		if got := points(t, s); got != nil {
			t.Errorf("LoadRaidPoints = %+v, want nil", got)
		}
		must(t, "SaveRaidBosses(nil)", s.SaveRaidBosses(t.Context(), nil))
		must(t, "SaveGrandBossStatus of a missing boss", s.SaveGrandBossStatus(t.Context(), 29001, 1))
		if got := grands(t, s); got != nil {
			t.Errorf("a status of a missing boss stored %+v, want nothing", got)
		}
		must(t, "ClearRaidPoints", s.ClearRaidPoints(t.Context()))
	})

	t.Run("RaidBosses", func(t *testing.T) {
		s, _ := newStore(t)
		must(t, "SaveRaidBosses", s.SaveRaidBosses(t.Context(), []storage.RaidBossState{tyrannosaurus, kutus}))
		if got, want := raids(t, s), []storage.RaidBossState{kutus, tyrannosaurus}; !slices.Equal(got, want) {
			t.Errorf("LoadRaidBosses = %+v, want %+v", got, want)
		}
		alive := storage.RaidBossState{BossID: 25001, CurHP: 1234.5, CurMP: 10}
		must(t, "SaveRaidBosses", s.SaveRaidBosses(t.Context(), []storage.RaidBossState{alive}))
		if got, want := raids(t, s), []storage.RaidBossState{alive, tyrannosaurus}; !slices.Equal(got, want) {
			t.Errorf("after an update: %+v, want %+v", got, want)
		}
	})

	t.Run("GrandBosses", func(t *testing.T) {
		s, _ := newStore(t)
		must(t, "SeedGrandBosses", s.SeedGrandBosses(t.Context(), []storage.GrandBossState{core, queenAnt}))
		if got, want := grands(t, s), []storage.GrandBossState{queenAnt, core}; !slices.Equal(got, want) {
			t.Fatalf("LoadGrandBosses = %+v, want %+v", got, want)
		}
		must(t, "SaveGrandBossStatus", s.SaveGrandBossStatus(t.Context(), 29001, 1))
		dead := queenAnt
		dead.Status = 1
		if got := grands(t, s); got[0] != dead || got[1] != core {
			t.Errorf("after the status: %+v, want %+v first", got, dead)
		}
		// A seed again changes nothing that is stored.
		must(t, "SeedGrandBosses", s.SeedGrandBosses(t.Context(), []storage.GrandBossState{queenAnt, core}))
		if got := grands(t, s); got[0] != dead {
			t.Errorf("a second seed overwrote the stored boss: %+v", got[0])
		}
		whole := storage.GrandBossState{BossID: 29001, X: 1, Y: 2, Z: 3, Heading: 4, RespawnTime: 1795000000000, CurHP: 5, CurMP: 6, Status: 1}
		must(t, "SaveGrandBosses", s.SaveGrandBosses(t.Context(), []storage.GrandBossState{whole}))
		if got, want := grands(t, s), []storage.GrandBossState{whole, core}; !slices.Equal(got, want) {
			t.Errorf("after a whole save: %+v, want %+v", got, want)
		}
	})

	t.Run("ZonePlayers", func(t *testing.T) {
		s, _ := newStore(t)
		must(t, "SaveBossZonePlayers", s.SaveBossZonePlayers(t.Context(), []storage.BossZonePlayer{
			{Zone: 12010, Player: 0x10000002}, {Zone: 12003, Player: 0x10000005}, {Zone: 12010, Player: 0x10000001},
			{Zone: 12010, Player: 0x10000001},
		}))
		want := []storage.BossZonePlayer{{Zone: 12003, Player: 0x10000005}, {Zone: 12010, Player: 0x10000001}, {Zone: 12010, Player: 0x10000002}}
		if got := zones(t, s); !slices.Equal(got, want) {
			t.Errorf("LoadBossZonePlayers = %+v, want %+v", got, want)
		}
		must(t, "SaveBossZonePlayers", s.SaveBossZonePlayers(t.Context(), []storage.BossZonePlayer{{Zone: 12016, Player: 7}}))
		if got, want := zones(t, s), []storage.BossZonePlayer{{Zone: 12016, Player: 7}}; !slices.Equal(got, want) {
			t.Errorf("after a replacement: %+v, want %+v", got, want)
		}
		must(t, "SaveBossZonePlayers(nil)", s.SaveBossZonePlayers(t.Context(), nil))
		if got := zones(t, s); got != nil {
			t.Errorf("after an empty save: %+v, want nil", got)
		}
	})

	t.Run("RaidPoints", func(t *testing.T) {
		s, _ := newStore(t)
		must(t, "SaveRaidPoints", s.SaveRaidPoints(t.Context(), []storage.RaidPoints{
			{CharID: 20, BossID: 25001, Points: 9}, {CharID: 10, BossID: 25019, Points: 30}, {CharID: 10, BossID: 25001, Points: 7},
		}))
		must(t, "SaveRaidPoints", s.SaveRaidPoints(t.Context(), []storage.RaidPoints{{CharID: 10, BossID: 25001, Points: 19}}))
		want := []storage.RaidPoints{{CharID: 10, BossID: 25001, Points: 19}, {CharID: 10, BossID: 25019, Points: 30}, {CharID: 20, BossID: 25001, Points: 9}}
		if got := points(t, s); !slices.Equal(got, want) {
			t.Errorf("LoadRaidPoints = %+v, want %+v", got, want)
		}
		must(t, "ClearRaidPoints", s.ClearRaidPoints(t.Context()))
		if got := points(t, s); got != nil {
			t.Errorf("after the reset: %+v, want nil", got)
		}
	})

	t.Run("FailedWrites", func(t *testing.T) {
		s, failNext := newStore(t)
		must(t, "SaveRaidBosses", s.SaveRaidBosses(t.Context(), []storage.RaidBossState{kutus}))
		must(t, "SeedGrandBosses", s.SeedGrandBosses(t.Context(), []storage.GrandBossState{queenAnt}))
		must(t, "SaveBossZonePlayers", s.SaveBossZonePlayers(t.Context(), []storage.BossZonePlayer{{Zone: 12010, Player: 1}}))
		must(t, "SaveRaidPoints", s.SaveRaidPoints(t.Context(), []storage.RaidPoints{{CharID: 1, BossID: 25001, Points: 5}}))

		for _, w := range []struct {
			name string
			fn   func() error
		}{
			{"SaveRaidBosses", func() error {
				return s.SaveRaidBosses(t.Context(), []storage.RaidBossState{{BossID: 25001}, tyrannosaurus})
			}},
			{"SeedGrandBosses", func() error { return s.SeedGrandBosses(t.Context(), []storage.GrandBossState{core}) }},
			{"SaveGrandBossStatus", func() error { return s.SaveGrandBossStatus(t.Context(), 29001, 1) }},
			{"SaveGrandBosses", func() error {
				return s.SaveGrandBosses(t.Context(), []storage.GrandBossState{{BossID: 29001, Status: 3}, core})
			}},
			{"SaveBossZonePlayers", func() error {
				return s.SaveBossZonePlayers(t.Context(), []storage.BossZonePlayer{{Zone: 12003, Player: 2}})
			}},
			{"SaveRaidPoints", func() error {
				return s.SaveRaidPoints(t.Context(), []storage.RaidPoints{{CharID: 1, BossID: 25001, Points: 50}})
			}},
			{"ClearRaidPoints", func() error { return s.ClearRaidPoints(t.Context()) }},
		} {
			failNext(errInjected)
			if err := w.fn(); !errors.Is(err, errInjected) {
				t.Fatalf("%s error = %v, want the injected failure", w.name, err)
			}
		}
		if got, want := raids(t, s), []storage.RaidBossState{kutus}; !slices.Equal(got, want) {
			t.Errorf("raid bosses after failed writes = %+v, want %+v", got, want)
		}
		if got, want := grands(t, s), []storage.GrandBossState{queenAnt}; !slices.Equal(got, want) {
			t.Errorf("grand bosses after failed writes = %+v, want %+v", got, want)
		}
		if got, want := zones(t, s), []storage.BossZonePlayer{{Zone: 12010, Player: 1}}; !slices.Equal(got, want) {
			t.Errorf("zone players after failed writes = %+v, want %+v", got, want)
		}
		if got, want := points(t, s), []storage.RaidPoints{{CharID: 1, BossID: 25001, Points: 5}}; !slices.Equal(got, want) {
			t.Errorf("raid points after failed writes = %+v, want %+v", got, want)
		}
	})
}
