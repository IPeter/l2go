package social

import "testing"

func TestLevelUpCost(t *testing.T) {
	r := DefaultClanRules()
	tests := []struct {
		level int
		want  LevelUpCost
		ok    bool
	}{
		{0, LevelUpCost{SP: 20000, Adena: 650000}, true},
		{1, LevelUpCost{SP: 100000, Adena: 2500000}, true},
		{2, LevelUpCost{SP: 350000, ItemID: 1419}, true},
		{3, LevelUpCost{SP: 1000000, ItemID: 3874}, true},
		{4, LevelUpCost{SP: 2500000, ItemID: 3870}, true},
		{5, LevelUpCost{Reputation: 5000, Members: 30}, true},
		{6, LevelUpCost{Reputation: 10000, Members: 50}, true},
		{7, LevelUpCost{Reputation: 20000, Members: 80}, true},
		{8, LevelUpCost{}, false},
		{-1, LevelUpCost{}, false},
	}
	for _, tt := range tests {
		got, ok := r.LevelUpCost(tt.level)
		if got != tt.want || ok != tt.ok {
			t.Errorf("LevelUpCost(%d) = %+v %v, want %+v %v", tt.level, got, ok, tt.want, tt.ok)
		}
	}
	r.LevelCost[1], r.LevelMembers[1] = 1, 2
	if got, _ := r.LevelUpCost(6); got.Reputation != 1 || got.Members != 2 {
		t.Errorf("configured level 7 cost: %+v", got)
	}
}

func TestLevelUpMet(t *testing.T) {
	r := DefaultClanRules()
	c0, _ := r.LevelUpCost(0)
	c2, _ := r.LevelUpCost(2)
	c5, _ := r.LevelUpCost(5)
	tests := []struct {
		name string
		cost LevelUpCost
		have LevelUpHave
		want bool
	}{
		{"0 exact", c0, LevelUpHave{SP: 20000, Adena: 650000}, true},
		{"0 sp short", c0, LevelUpHave{SP: 19999, Adena: 650000}, false},
		{"0 adena short", c0, LevelUpHave{SP: 20000, Adena: 649999}, false},
		{"2 item", c2, LevelUpHave{SP: 350000, HasItem: true}, true},
		{"2 no item", c2, LevelUpHave{SP: 350000}, false},
		{"5 exact", c5, LevelUpHave{Reputation: 5000, Members: 30}, true},
		{"5 members short", c5, LevelUpHave{Reputation: 5000, Members: 29}, false},
		{"5 reputation short", c5, LevelUpHave{Reputation: 4999, Members: 80}, false},
	}
	for _, tt := range tests {
		if got := tt.cost.Met(tt.have); got != tt.want {
			t.Errorf("%s: Met = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestMaxMembers(t *testing.T) {
	tests := []struct{ level, unit, want int }{
		{0, UnitMain, 10}, {1, UnitMain, 15}, {2, UnitMain, 20}, {3, UnitMain, 30},
		{4, UnitMain, 40}, {5, UnitMain, 40}, {8, UnitMain, 40},
		{5, UnitAcademy, 20}, {8, UnitAcademy, 20},
		{6, UnitRoyal1, 20}, {8, UnitRoyal2, 20}, {11, UnitRoyal1, 30},
		{7, UnitKnight1, 10}, {8, UnitKnight4, 10}, {9, UnitKnight2, 25}, {11, UnitKnight3, 25},
		{8, 300, 0},
	}
	for _, tt := range tests {
		if got := MaxMembers(tt.level, tt.unit); got != tt.want {
			t.Errorf("MaxMembers(%d, %d) = %d, want %d", tt.level, tt.unit, got, tt.want)
		}
	}
}

func TestUnits(t *testing.T) {
	r := DefaultClanRules()
	for unit, want := range map[int][2]int{
		UnitAcademy: {0, 5}, UnitRoyal1: {5000, 6}, UnitRoyal2: {5000, 6},
		UnitKnight1: {10000, 7}, UnitKnight4: {10000, 7}, UnitMain: {5000, 0},
	} {
		if got := r.UnitCost(unit); unit != UnitMain && got != want[0] {
			t.Errorf("UnitCost(%d) = %d, want %d", unit, got, want[0])
		}
		if got := UnitMinClanLevel(unit); got != want[1] {
			t.Errorf("UnitMinClanLevel(%d) = %d, want %d", unit, got, want[1])
		}
	}

	has := func(units ...int) func(int) bool {
		return func(u int) bool {
			for _, x := range units {
				if x == u {
					return true
				}
			}
			return false
		}
	}
	tests := []struct {
		req   int
		units []int
		want  int
	}{
		{UnitAcademy, nil, UnitAcademy},
		{UnitAcademy, []int{UnitAcademy}, 0},
		{UnitRoyal1, nil, UnitRoyal1},
		{UnitRoyal1, []int{UnitRoyal1}, UnitRoyal2},
		{UnitRoyal1, []int{UnitRoyal1, UnitRoyal2}, 0},
		{UnitKnight1, []int{UnitKnight1}, UnitKnight2},
		{UnitKnight1, []int{UnitKnight1, UnitKnight2, UnitKnight3}, UnitKnight4},
		{UnitKnight1, []int{UnitKnight1, UnitKnight2, UnitKnight3, UnitKnight4}, 0},
		{UnitKnight1, []int{UnitKnight2}, UnitKnight1},
	}
	for _, tt := range tests {
		if got := AvailableUnit(tt.req, has(tt.units...)); got != tt.want {
			t.Errorf("AvailableUnit(%d, %v) = %d, want %d", tt.req, tt.units, got, tt.want)
		}
	}
}

func TestPledgeClass(t *testing.T) {
	member := func(level, unit int) PledgeMember { return PledgeMember{InClan: true, ClanLevel: level, Unit: unit} }
	leader := func(level int) PledgeMember { m := member(level, UnitMain); m.Leader = true; return m }
	captain := func(level, leads int) PledgeMember { m := member(level, UnitMain); m.LeadsUnit = leads; return m }
	tests := []struct {
		name string
		m    PledgeMember
		want int
	}{
		{"no clan", PledgeMember{}, 0},
		{"no clan noble", PledgeMember{Noble: true}, 5},
		{"no clan hero", PledgeMember{Hero: true}, 8},
		{"level 0 member", member(0, UnitMain), 1},
		{"level 3 leader", leader(3), 1},
		{"level 4 leader", leader(4), 3},
		{"level 4 member", member(4, UnitMain), 0},
		{"level 4 member noble", PledgeMember{InClan: true, ClanLevel: 4, Noble: true}, 5},
		{"level 5 leader", leader(5), 4},
		{"level 5 member", member(5, UnitMain), 2},
		{"level 5 academy", member(5, UnitAcademy), 2},
		{"level 6 leader", leader(6), 5},
		{"level 6 member", member(6, UnitMain), 3},
		{"level 6 academy", member(6, UnitAcademy), 1},
		{"level 6 royal", member(6, UnitRoyal1), 2},
		{"level 6 royal captain", captain(6, UnitRoyal2), 4},
		{"level 6 knight", member(6, UnitKnight1), 0},
		{"level 6 knight captain", captain(6, UnitKnight1), 3},
		{"level 7 leader", leader(7), 7},
		{"level 7 member", member(7, UnitMain), 4},
		{"level 7 academy", member(7, UnitAcademy), 1},
		{"level 7 royal", member(7, UnitRoyal2), 3},
		{"level 7 knight", member(7, UnitKnight3), 2},
		{"level 7 royal captain", captain(7, UnitRoyal1), 6},
		{"level 7 knight captain", captain(7, UnitKnight4), 5},
		{"level 8 leader", leader(8), 8},
		{"level 8 member", member(8, UnitMain), 5},
		{"level 8 academy", member(8, UnitAcademy), 1},
		{"level 8 royal", member(8, UnitRoyal1), 4},
		{"level 8 knight", member(8, UnitKnight2), 3},
		{"level 8 royal captain", captain(8, UnitRoyal2), 7},
		{"level 8 knight captain", captain(8, UnitKnight1), 6},
		{"level 8 academy hero", PledgeMember{InClan: true, ClanLevel: 8, Unit: UnitAcademy, Hero: true}, 8},
		{"level 8 leader hero", PledgeMember{InClan: true, ClanLevel: 8, Leader: true, Hero: true}, 8},
		{"level 11 leader", leader(11), 11},
		{"level 9 member", member(9, UnitMain), 6},
		{"level 12", member(12, UnitMain), 1},
	}
	for _, tt := range tests {
		if got := PledgeClass(tt.m); got != tt.want {
			t.Errorf("%s: PledgeClass = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestAcademy(t *testing.T) {
	r := DefaultClanRules()
	for lvl, want := range map[int]int{10: 650, 16: 650, 17: 630, 30: 370, 38: 210, 39: 190, 76: 190} {
		if got := r.AcademyGraduationPoints(lvl); got != want {
			t.Errorf("AcademyGraduationPoints(%d) = %d, want %d", lvl, got, want)
		}
	}
	tests := []struct {
		level, classLevel int
		want              bool
	}{{40, 1, true}, {41, 0, false}, {20, 2, false}, {1, 0, true}}
	for _, tt := range tests {
		if got := CanJoinAcademy(tt.level, tt.classLevel); got != tt.want {
			t.Errorf("CanJoinAcademy(%d, %d) = %v", tt.level, tt.classLevel, got)
		}
	}
}
