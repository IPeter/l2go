package game

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/world"
)

// BenchmarkMonsterThink runs 10,000 thinking monsters, a tenth of them
// thinking in each 100 ms tick, wandering on 1 think in 30, with every
// region active (GridsAlwaysOn) and no player around. One op is one game
// tick of the whole loop; Plan 05's tick budget is well under 10 ms.
func BenchmarkMonsterThink(b *testing.B) {
	const (
		monsters = 10000
		spacing  = 200
		side     = 100 // 100 × 100 monsters over 20,000 × 20,000 units
	)
	d, err := loadTestData()
	if err != nil {
		b.Fatalf("load the game data: %v", err)
	}
	tmpl, ok := d.Template(classHumanFighter)
	if !ok {
		b.Fatal("no Human Fighter")
	}
	start := tmpl.CreationPoints[0]
	w, err := data.NewWorld(aiTestWorld(start))
	if err != nil {
		b.Fatalf("NewWorld: %v", err)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval = 0
	gs.World.GridsAlwaysOn = true
	clock := world.NewFakeClock(testStart)
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w,
		Characters: newRecordingStore(), IDs: memstore.NewIDs(), Clock: clock,
		Rand: rnd.New(aiTestSeed),
	})
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	for i := range monsters {
		pos := world.Position{
			X: int32(start.X) - side*spacing/2 + int32(i%side)*spacing,
			Y: int32(start.Y) - side*spacing + int32(i/side)*spacing, // Talking Island is near the world's edge

			Z: int32(start.Z),
		}
		n, err := srv.SpawnNpc(aiWanderer, pos)
		if err != nil {
			b.Fatalf("SpawnNpc: %v", err)
		}
		srv.npcSetActive(n)
	}
	if got := srv.ThinkingNpcs(); got != monsters {
		b.Fatalf("%d NPCs think, want %d", got, monsters)
	}
	// Let the wanderers get going.
	for range 50 {
		clock.Advance(100 * time.Millisecond)
		if err := srv.loop.Step(); err != nil {
			b.Fatalf("Step: %v", err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		clock.Advance(100 * time.Millisecond)
		if err := srv.loop.Step(); err != nil {
			b.Fatalf("Step: %v", err)
		}
	}
	b.StopTimer()
	walking := 0
	for _, l := range srv.ai.slots {
		for _, n := range l {
			if n.Move != nil {
				walking++
			}
		}
	}
	b.ReportMetric(float64(walking), "walking")
}
