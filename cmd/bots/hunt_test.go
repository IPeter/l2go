package main

import (
	"bytes"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

func TestNearestMonster(t *testing.T) {
	home := testclient.Position{}
	me := testclient.Position{X: 1000}
	mob := func(id, x int32) testclient.Object {
		return testclient.Object{ObjectID: id, Kind: testclient.KindNpc, Attackable: true, Motion: testclient.Motion{Pos: testclient.Position{X: x}}}
	}
	dead := mob(1, 1010)
	dead.Dead = true
	guard := mob(2, 1020)
	guard.Attackable = false
	player := mob(3, 1000)
	player.Kind = testclient.KindPlayer
	objs := []testclient.Object{
		dead, guard, player,
		mob(4, 4000), // nearest to the bot, but too far from home
		mob(6, 500),
		mob(5, 1500),
		mob(7, -1000),
	}
	if o, ok := nearestMonster(me, home, objs, 3000, 0); !ok || o.ObjectID != 5 {
		t.Errorf("nearestMonster = %d, %v; want 5 (the lower ID of two at 500 from the bot)", o.ObjectID, ok)
	}
	if o, ok := nearestMonster(me, home, objs, 3000, 5); !ok || o.ObjectID != 6 {
		t.Errorf("nearestMonster avoiding 5 = %d, %v; want 6", o.ObjectID, ok)
	}
	if _, ok := nearestMonster(me, home, objs[:4], 3000, 0); ok {
		t.Error("a dead monster, a guard, a player or a far monster was picked")
	}
}

func TestMissingBuffs(t *testing.T) {
	effects := []testclient.Effect{{SkillID: 1068, Level: 3, Duration: 1200}, {SkillID: 4416, Level: 1, Duration: -1}}
	if got := missingBuffs(effects, huntBuffs[:3]); !slices.Equal(got, []int32{1040, 1204}) {
		t.Errorf("missingBuffs = %v, want Shield and Wind Walk", got)
	}
	if got := missingBuffs(effects, huntBuffs[:1]); len(got) != 0 {
		t.Errorf("missingBuffs with Might held = %v", got)
	}
	if len(huntBuffs) != 10 || len(slices.Compact(slices.Sorted(slices.Values(huntBuffs)))) != 10 {
		t.Errorf("huntBuffs = %v, want ten different buffs", huntBuffs)
	}
}

func TestEnoughMana(t *testing.T) {
	tests := []struct {
		mp, maxMP int32
		want      bool
	}{
		{100, 100, true},
		{25, 100, true},
		{24, 100, false},
		{0, 0, false},
	}
	for _, tt := range tests {
		if got := enoughMana(testclient.Me{MP: tt.mp, MaxMP: tt.maxMP}); got != tt.want {
			t.Errorf("enoughMana(%d/%d) = %v, want %v", tt.mp, tt.maxMP, got, tt.want)
		}
	}
}

func TestBotClass(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	stats := NewStats(1, time.Now(), 1)
	cfg := config{prefix: "bot", n: 50, class: classMystic}
	if b := newBot(7, cfg, stats, logger); b.account != "bot0007" || b.char != "Bot0007M" {
		t.Errorf("mystic bot: %s, %s", b.account, b.char)
	}
	cfg.class = classFighter
	if b := newBot(7, cfg, stats, logger); b.char != "Bot0007" {
		t.Errorf("fighter bot: %s", b.char)
	}
	if r := createRequest(classMystic, "Bot0007M"); r.ClassID != 10 || r.INT != 41 || r.Name != "Bot0007M" {
		t.Errorf("mystic CharacterCreate = %+v", r)
	}
	if r := createRequest(classFighter, "Bot0007"); r.ClassID != 0 || r.STR != 40 {
		t.Errorf("fighter CharacterCreate = %+v", r)
	}
}

func TestCheckKill(t *testing.T) {
	stats := NewStats(1, time.Now(), 1)
	b := newBot(0, config{prefix: "bot", n: 1}, stats, slog.New(slog.DiscardHandler))
	w := testclient.NewWorld()
	start := time.Now()
	b.target, b.attacking, b.fightStart = 0x60000005, 0x60000005, start

	// The monster is not in sight: the hunt ends without a kill.
	b.checkKill(w, start.Add(time.Second))
	if b.target != 0 || b.attacking != 0 {
		t.Errorf("after losing the monster: target %d, attacking %d", b.target, b.attacking)
	}
	if r := stats.Report(start.Add(time.Second), false); len(r.Events) != 0 {
		t.Errorf("events after losing the monster: %v", r.Events)
	}

	// The next monster comes into sight, is hunted and dies: a kill.
	w.Apply(testclient.NpcInfo{ObjectID: 0x60000006, NpcTypeID: 1020481, Attackable: true, X: 100})
	b.start = testclient.Position{}
	b.cfg.huntRange = 3000
	o, ok := b.prey(w, start)
	if !ok || o.ObjectID != 0x60000006 || b.target != 0x60000006 || !b.fightStart.Equal(start) {
		t.Fatalf("prey = %d, %v; target %d", o.ObjectID, ok, b.target)
	}
	b.checkKill(w, start.Add(time.Second))
	if b.target != 0x60000006 {
		t.Error("a living monster ended the hunt")
	}
	w.Apply(testclient.Die{ObjectID: 0x60000006})
	b.checkKill(w, start.Add(3*time.Second))
	r := stats.Report(start.Add(3*time.Second), false)
	if b.target != 0 || !slices.Equal(r.Events, []ErrorCount{{"kill", 1}}) {
		t.Errorf("after the death: target %d, events %v", b.target, r.Events)
	}
	if len(r.Latency) != 1 || r.Latency[0].Op != "kill" || r.Latency[0].Max != 3*time.Second {
		t.Errorf("kill latency = %+v", r.Latency)
	}
}

func TestStatsEvents(t *testing.T) {
	t0 := time.Unix(1000, 0)
	s := NewStats(1, t0, 1)
	s.Count("kill")
	s.Count("kill")
	s.Count("death")
	r := s.Report(t0.Add(10*time.Second), false)
	if want := []ErrorCount{{"death", 1}, {"kill", 2}}; !slices.Equal(r.Events, want) {
		t.Errorf("events = %v, want %v", r.Events, want)
	}
	var buf bytes.Buffer
	if err := r.Write(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "events: death=1 kill=2") {
		t.Errorf("report:\n%s", buf.String())
	}
	s.Count("cast")
	if r := s.Report(t0.Add(20*time.Second), false); !slices.Equal(r.Events, []ErrorCount{{"cast", 1}}) {
		t.Errorf("second interval events = %v", r.Events)
	}
	if f := s.Report(t0.Add(30*time.Second), true); len(f.Events) != 3 {
		t.Errorf("final events = %v", f.Events)
	}
}
