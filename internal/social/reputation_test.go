package social

import (
	"slices"
	"testing"
)

func TestReputationChange(t *testing.T) {
	tests := []struct {
		score, delta, want int
		off, on            bool
	}{
		{0, 100, 100, false, false},
		{0, -1, -1, true, false},
		{5, -5, 0, false, false},
		{-1, 1, 0, false, true},
		{-10, 5, -5, false, false},
		{99_999_990, 100, MaxReputation, false, false},
		{-99_999_990, -100, -MaxReputation, false, false},
	}
	for _, tt := range tests {
		got, off, on := ReputationChange(tt.score, tt.delta)
		if got != tt.want || off != tt.off || on != tt.on {
			t.Errorf("ReputationChange(%d, %d) = %d %v %v, want %d %v %v", tt.score, tt.delta, got, off, on, tt.want, tt.off, tt.on)
		}
	}
}

func TestWarKillReputation(t *testing.T) {
	tests := []struct {
		killer, victim         int
		wantKiller, wantVictim int
	}{
		{10, 10, 1, -1},
		{0, 5, 1, -1}, // the killer's gain lifts it above 0 first
		{5, 0, 0, -1},
		{0, 0, 0, 0},
		{-3, 5, 1, 0},
		{10, -5, 0, -1},
	}
	for _, tt := range tests {
		k, v := WarKillReputation(1, tt.killer, tt.victim)
		if k != tt.wantKiller || v != tt.wantVictim {
			t.Errorf("WarKillReputation(1, %d, %d) = %d %d, want %d %d", tt.killer, tt.victim, k, v, tt.wantKiller, tt.wantVictim)
		}
	}
}

func TestClanRanks(t *testing.T) {
	clans := []ClanStanding{
		{Level: 5, Reputation: 100},
		{Level: 5, Reputation: 100}, // tie: both ranked below each other
		{Level: 8, Reputation: 0},
		{Level: 3, Reputation: 5000},
		{Level: 2, Reputation: 99999}, // below level 3: rank 0
		{Level: 5, Reputation: 50},
	}
	want := []int{3, 3, 1, 5, 0, 4}
	got := ClanRanks(clans)
	if !slices.Equal(got, want) {
		t.Errorf("ClanRanks = %v, want %v", got, want)
	}
	for i := range clans {
		if r := ClanRank(clans, i); r != want[i] {
			t.Errorf("ClanRank(%d) = %d, want %d", i, r, want[i])
		}
	}

	// A larger pseudo-random set agrees with the direct count.
	big := make([]ClanStanding, 300)
	x := uint32(12345)
	for i := range big {
		x = x*1664525 + 1013904223
		big[i] = ClanStanding{Level: int(x>>8) % 9, Reputation: int(x>>16) % 50}
	}
	fast := ClanRanks(big)
	for i := range big {
		if r := ClanRank(big, i); r != fast[i] {
			t.Fatalf("clan %d %+v: ClanRanks %d, ClanRank %d", i, big[i], fast[i], r)
		}
	}
}
