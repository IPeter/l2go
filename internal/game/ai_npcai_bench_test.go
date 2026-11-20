package game

import (
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/geo/geotest"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/world"
)

// BenchmarkMonsterThinkSkills runs 10,000 fighting monsters with skills
// (Plan 13, Phase 10.4 item 4): real templates with heals, buffs, debuffs
// and nukes, in pairs that hate each other 200 apart, on flat synthetic
// geodata so every sight check walks the cells. A tenth of them think in
// each 100 ms tick: thinkAttack with the npcai steps (suicide, crowd,
// chaos, skill use with its clan scans and sight checks, the reach check
// and targetReconsider). Casts are counted, not cast, so the pairs stay
// alive. One op is one game tick of the whole loop.
func BenchmarkMonsterThinkSkills(b *testing.B) {
	const (
		monsters = 10000
		spacing  = 200
		side     = 100 // 100 × 100 monsters over 20,000 × 20,000 units
	)
	d, err := loadTestData()
	if err != nil {
		b.Fatalf("load the game data: %v", err)
	}
	sk, err := loadTestSkills()
	if err != nil {
		b.Fatalf("load the skills: %v", err)
	}
	real, err := loadDialogWorld()
	if err != nil {
		b.Fatalf("load the world data: %v", err)
	}
	_, healer := aiFindHealerMinion(b)
	nuker := aiFindNuker(b)
	wd := aiSkillWorld(b, real, data.Location{X: 1024, Y: 1024, Z: 0}, healer, nuker)

	// Flat ground under the whole grid, in cells of 16.
	cells := side*spacing/16 + 64
	row := strings.Repeat(".", cells)
	rows := make([]string, cells)
	for i := range rows {
		rows[i] = row
	}
	g := geotest.New(rows...).Load(b)

	w, err := data.NewWorld(wd)
	if err != nil {
		b.Fatalf("NewWorld: %v", err)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval = 0
	gs.World.GridsAlwaysOn = true
	clock := world.NewFakeClock(testStart)
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w, Skills: sk, Geo: g,
		Characters: newRecordingStore(), IDs: memstore.NewIDs(), Clock: clock,
		Rand: rnd.New(aiTestSeed),
	})
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	srv.ai.doAttack = func(*Npc, Creature) {}
	casts := 0
	srv.ai.x.doCast = func(*Npc, *skills.Skill) { casts++ }

	npcs := make([]*Npc, 0, monsters)
	for i := range monsters {
		id := nuker
		if (i/2)%2 == 0 {
			id = healer
		}
		pos := world.Position{
			X: 1024 + 512 + int32(i%side)*spacing,
			Y: 1024 + 512 + int32(i/side)*spacing,
			Z: 0,
		}
		n, err := srv.SpawnNpc(id, pos)
		if err != nil {
			b.Fatalf("SpawnNpc: %v", err)
		}
		npcs = append(npcs, n)
	}
	// Pairs hate each other and fight; half of them are hurt, so the heals
	// have someone to look after.
	for i := 0; i+1 < len(npcs); i += 2 {
		a, c := npcs[i], npcs[i+1]
		a.SetCurrentHP(float64(a.MaxHP()) / 2)
		srv.addDamageHate(a, c, 0, 100)
		srv.addDamageHate(c, a, 0, 100)
		srv.npcSetAttack(a, c)
		srv.npcSetAttack(c, a)
	}
	if got := srv.ThinkingNpcs(); got != monsters {
		b.Fatalf("%d NPCs think, want %d", got, monsters)
	}
	for range 50 {
		clock.Advance(100 * time.Millisecond)
		if err := srv.loop.Step(); err != nil {
			b.Fatalf("Step: %v", err)
		}
	}
	casts = 0
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		clock.Advance(100 * time.Millisecond)
		if err := srv.loop.Step(); err != nil {
			b.Fatalf("Step: %v", err)
		}
	}
	b.StopTimer()
	fighting := 0
	for _, n := range npcs {
		if n.ai.intention == NpcAttack {
			fighting++
		}
	}
	b.ReportMetric(float64(fighting), "fighting")
	b.ReportMetric(float64(casts)/float64(b.N), "casts/op")
}
