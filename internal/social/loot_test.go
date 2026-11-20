package social

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
)

var (
	okC      = LootCandidate{HasSpace: true, InRange: true}
	fullC    = LootCandidate{HasSpace: false, InRange: true}
	farC     = LootCandidate{HasSpace: true, InRange: false}
	allOK3   = []LootCandidate{okC, okC, okC}
	noneOK3  = []LootCandidate{fullC, farC, fullC}
	mixedOK4 = []LootCandidate{okC, fullC, okC, farC}
)

func TestLooterModes(t *testing.T) {
	tests := []struct {
		name    string
		mode    LootMode
		members []LootCandidate
		spoil   bool
		draws   []rnd.Draw
		want    int
	}{
		{"finders keepers", FindersKeepers, allOK3, false, nil, -1},
		{"finders keepers spoil", FindersKeepers, allOK3, true, nil, -1},
		{"random", Random, mixedOK4, false, []rnd.Draw{rnd.Get(2, 1)}, 2},
		{"random first", Random, mixedOK4, false, []rnd.Draw{rnd.Get(2, 0)}, 0},
		{"random spoil stays", Random, mixedOK4, true, nil, -1},
		{"random nobody", Random, noneOK3, false, nil, -1},
		{"random one candidate draws", Random, []LootCandidate{fullC, okC}, false, []rnd.Draw{rnd.Get(1, 0)}, 1},
		{"random incl spoil", RandomIncludingSpoil, mixedOK4, true, []rnd.Draw{rnd.Get(2, 1)}, 2},
		{"random incl spoil item", RandomIncludingSpoil, allOK3, false, []rnd.Draw{rnd.Get(3, 2)}, 2},
		{"by turn spoil stays", ByTurn, allOK3, true, nil, -1},
		{"by turn first goes to second", ByTurn, allOK3, false, nil, 1},
		{"by turn incl spoil", ByTurnIncludingSpoil, allOK3, true, nil, 1},
		{"by turn nobody", ByTurn, noneOK3, false, nil, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := rnd.NewScripted(t, tt.draws...)
			l := &Loot{Mode: tt.mode}
			if got := l.Looter(src, tt.members, tt.spoil); got != tt.want {
				t.Errorf("Looter = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLooterRotation(t *testing.T) {
	tests := []struct {
		name    string
		members []LootCandidate
		spoil   []bool
		want    []int
	}{
		{"all qualify", allOK3, []bool{false, false, false, false}, []int{1, 2, 0, 1}},
		{"third full", []LootCandidate{okC, okC, fullC}, []bool{false, false, false, false}, []int{1, 0, 1, 0}},
		{"spoil keeps the turn", allOK3, []bool{false, true, false}, []int{1, -1, 2}},
		{"leader only", []LootCandidate{okC, farC, fullC}, []bool{false, false}, []int{0, 0}},
	}
	for _, tt := range tests {
		l := &Loot{Mode: ByTurn}
		var got []int
		for _, s := range tt.spoil {
			got = append(got, l.Looter(nil, tt.members, s))
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s: looters %v, want %v", tt.name, got, tt.want)
		}
	}

	// Nobody qualifies: a full round, back to the same turn.
	l := &Loot{Mode: ByTurn, Turn: 1}
	if got := l.Looter(nil, noneOK3, false); got != -1 || l.Turn != 1 {
		t.Errorf("nobody: %d, turn %d", got, l.Turn)
	}
	// The party shrank below the turn: it wraps to the leader.
	l = &Loot{Mode: ByTurn, Turn: 5}
	if got := l.Looter(nil, allOK3, false); got != 0 {
		t.Errorf("shrunk party: %d", got)
	}
}

func TestLootModeIDs(t *testing.T) {
	want := map[LootMode]int{FindersKeepers: 487, Random: 488, RandomIncludingSpoil: 798, ByTurn: 799, ByTurnIncludingSpoil: 800}
	for m, s := range want {
		if !m.Valid() || m.SysString() != s {
			t.Errorf("%v: valid %v sysstring %d", m, m.Valid(), m.SysString())
		}
	}
	if LootMode(5).Valid() || LootMode(-1).Valid() || LootMode(5).SysString() != 0 {
		t.Error("invalid mode accepted")
	}
	if ByTurnIncludingSpoil.String() != "BY_TURN_INCLUDING_SPOIL" || LootMode(7).String() != "LootMode(7)" {
		t.Error("LootMode.String")
	}
}

func TestSplitAdena(t *testing.T) {
	tests := []struct {
		name    string
		adena   int64
		inRange []bool
		draws   []rnd.Draw
		want    []int64
	}{
		{"even", 99, []bool{true, true, true}, nil, []int64{33, 33, 33}},
		{"remainder 1 skips out of range", 100, []bool{true, false, true, true}, []rnd.Draw{rnd.Get(3, 2)}, []int64{33, 0, 33, 34}},
		{"remainder 2 wraps", 101, []bool{true, true, true}, []rnd.Draw{rnd.Get(3, 2)}, []int64{34, 33, 34}},
		{"less than members", 2, []bool{true, true, true}, []rnd.Draw{rnd.Get(3, 1)}, []int64{0, 1, 1}},
		{"one in range", 7, []bool{false, true}, nil, []int64{0, 7}},
		{"nobody in range", 100, []bool{false, false}, nil, []int64{0, 0}},
		{"no adena", 0, []bool{true, true}, nil, []int64{0, 0}},
		{"remainder from 0", 5, []bool{true, true, true}, []rnd.Draw{rnd.Get(3, 0)}, []int64{2, 2, 1}},
		{"9 members", 1_000_000_007, []bool{true, true, true, true, true, true, true, true, true},
			[]rnd.Draw{rnd.Get(9, 8)},
			[]int64{111111112, 111111112, 111111112, 111111112, 111111112, 111111112, 111111112, 111111111, 111111112}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := rnd.NewScripted(t, tt.draws...)
			got := SplitAdena(src, tt.adena, tt.inRange)
			if !slices.Equal(got, tt.want) {
				t.Errorf("SplitAdena = %v, want %v", got, tt.want)
			}
			var sum int64
			for _, v := range got {
				sum += v
			}
			if in := slices.Contains(tt.inRange, true); in && tt.adena > 0 && sum != tt.adena {
				t.Errorf("split %d of %d", sum, tt.adena)
			}
		})
	}
}
