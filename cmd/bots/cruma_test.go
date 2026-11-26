package main

import (
	"io"
	"testing"

	"github.com/IPeter/l2go/internal/testclient"
)

func TestParseFlagsCruma(t *testing.T) {
	cfg, err := parseFlags([]string{"-mode", "cruma"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.mode != modeCruma || cfg.class != classMystic || cfg.skill != skillWindStrike || cfg.level != crumaLevel || cfg.huntRange != 3000 {
		t.Errorf("cruma defaults: mode %q, class %q, skill %d, level %d, hunt range %v", cfg.mode, cfg.class, cfg.skill, cfg.level, cfg.huntRange)
	}
	tests := []struct {
		args  []string
		level int32
	}{
		{[]string{"-mode", "cruma", "-level", "60"}, 60},
		{[]string{"-mode", "cruma", "-level", "0"}, 0},
		// The other modes keep the characters' levels unless told.
		{nil, 0},
		{[]string{"-mode", "hunt"}, 0},
		{[]string{"-mode", "hunt", "-level", "20"}, 20},
	}
	for _, tt := range tests {
		cfg, err := parseFlags(tt.args, io.Discard)
		if err != nil || cfg.level != tt.level {
			t.Errorf("parseFlags(%q): level %d, %v; want %d", tt.args, cfg.level, err, tt.level)
		}
	}
	if cfg, err := parseFlags([]string{"-mode", "cruma", "-class", "fighter"}, io.Discard); err != nil || cfg.class != classFighter || cfg.skill != 0 {
		t.Errorf("a fighter cruma bot: class %q, skill %d, %v", cfg.class, cfg.skill, err)
	}
	for _, bad := range []string{"-2", "81"} {
		if _, err := parseFlags([]string{"-mode", "cruma", "-level", bad}, io.Discard); err == nil {
			t.Errorf("-level %s was accepted", bad)
		}
	}
}

func TestNextCrumaStep(t *testing.T) {
	spot := crumaSpots[0]
	at := func(p testclient.Position, level int32) testclient.Me {
		me := testclient.Me{HasUserInfo: true}
		me.Pos, me.UserInfo.Level = p, level
		return me
	}
	village := testclient.Position{X: 15670, Y: 142983, Z: -2705} // Dion
	near := testclient.Position{X: spot.X + 2500, Y: spot.Y, Z: spot.Z}
	dead := at(spot, 40)
	dead.Dead = true
	unknown := at(village, 0)
	unknown.HasUserInfo = false
	tests := []struct {
		name  string
		me    testclient.Me
		level int32
		want  crumaStep
	}{
		{"dead", dead, 40, crumaRestart},
		{"level 1 in the village", at(village, 1), 40, crumaLevelUp},
		{"level 40 in the village", at(village, 40), 40, crumaTeleport},
		{"no level wanted", at(village, 1), 0, crumaTeleport},
		{"before UserInfo", unknown, 40, crumaTeleport},
		{"at the spot", at(spot, 40), 40, crumaFight},
		{"chasing within the leash", at(near, 41), 40, crumaFight},
	}
	for _, tt := range tests {
		if got := nextCrumaStep(tt.me, tt.level, spot); got != tt.want {
			t.Errorf("%s: %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestCrumaSpots(t *testing.T) {
	seen := map[testclient.Position]int{}
	for id := range 2 * len(crumaSpots) {
		seen[crumaSpot(id)]++
	}
	if len(seen) != len(crumaSpots) {
		t.Errorf("bots use %d spots, want %d", len(seen), len(crumaSpots))
	}
	for p, n := range seen {
		if n != 2 {
			t.Errorf("spot %v has %d of %d bots", p, n, 2*len(crumaSpots))
		}
	}
	// Every spot is on the first floor (around -12,085) of the tower.
	for _, p := range crumaSpots {
		if p.Z < -12200 || p.Z > -12000 || p.X < 11000 || p.X > 24000 || p.Y < 116000 || p.Y > 121000 {
			t.Errorf("spot %v is not on Cruma Tower's first floor", p)
		}
	}
	// The leash is past the hunt range, so a chase never ends in a teleport.
	if crumaLeash <= 3000 {
		t.Errorf("crumaLeash %d is within the default hunt range", crumaLeash)
	}
}
