package olympiad

import (
	"maps"
	"slices"
	"testing"
)

// Stored points are clamped to [0, 10 × done + 12] (Mobius load), and the
// weekly points too: a noble without a match falls from 18 to 12.
func TestNoblePoints(t *testing.T) {
	c := DefaultConfig()
	tab := NewNobles(c, []Noble{
		{CharID: 1, ClassID: 88, Name: "Fresh", Points: 18},
		{CharID: 2, ClassID: 88, Name: "Busy", Points: 50, Done: 10},
		{CharID: 3, ClassID: 90, Name: "Broke", Points: -4, Done: 2},
	}, nil)
	if got := []int{tab.Points(1), tab.Points(2), tab.Points(3), tab.Points(9)}; !slices.Equal(got, []int{12, 50, 0, 0}) {
		t.Errorf("loaded points %v, want 12 50 0 0", got)
	}
	tab.Get(1).Points = 18 // as a registration in this period gives
	tab.AddWeeklyPoints(c, Competition)
	if got := []int{tab.Points(1), tab.Points(2), tab.Points(3)}; !slices.Equal(got, []int{12, 53, 3}) {
		t.Errorf("after the weekly points %v, want 12 53 3", got)
	}
	tab.AddWeeklyPoints(c, Validation)
	if tab.Points(2) != 53 {
		t.Errorf("weekly points in the validation period: %d", tab.Points(2))
	}
	// A new noble starts at 18; an old one stays.
	if n := tab.Add(c, 7, 92, "New"); n.Points != 18 || n.ClassID != 92 || tab.Len() != 4 {
		t.Errorf("Add = %+v, %d nobles", n, tab.Len())
	}
	if n := tab.Add(c, 2, 92, "Busy"); n.Points != 53 || n.ClassID != 88 {
		t.Errorf("Add of an old noble = %+v", n)
	}
	tab.CopyToMonthEnd()
	tab.Clear()
	if tab.Len() != 0 || tab.LastPoints(2) != 53 || tab.LastPoints(9) != 0 || len(tab.MonthEnd()) != 4 {
		t.Errorf("after the month end: %d nobles, last points %d", tab.Len(), tab.LastPoints(2))
	}
}

// Heroes: per class the best noble with at least 9 matches and a win, by
// points, then matches, then wins; the soulhounds share one hero.
func TestHeroesToBe(t *testing.T) {
	c := DefaultConfig()
	tab := NewNobles(c, []Noble{
		{CharID: 1, ClassID: 88, Name: "A", Points: 40, Done: 9, Won: 5},
		{CharID: 2, ClassID: 88, Name: "B", Points: 60, Done: 8, Won: 8},  // too few matches
		{CharID: 3, ClassID: 88, Name: "C", Points: 40, Done: 12, Won: 1}, // ties A on points, more matches
		{CharID: 4, ClassID: 93, Name: "D", Points: 90, Done: 20, Won: 0}, // no win
		{CharID: 5, ClassID: 93, Name: "E", Points: 30, Done: 9, Won: 3},
		{CharID: 6, ClassID: 117, Name: "F", Points: 30, Done: 9, Won: 3},
		{CharID: 7, ClassID: 132, Name: "G", Points: 50, Done: 9, Won: 3},
		{CharID: 8, ClassID: 133, Name: "H", Points: 50, Done: 10, Won: 3},
		{CharID: 9, ClassID: 2, Name: "I", Points: 99, Done: 30, Won: 30}, // no third class
	}, nil)
	got := tab.HeroesToBe()
	want := []Candidate{{3, 88, "C"}, {5, 93, "E"}, {6, 117, "F"}, {8, 133, "H"}}
	if !slices.Equal(got, want) {
		t.Errorf("HeroesToBe = %+v, want %+v", got, want)
	}
}

func TestClassLeadersAndRanks(t *testing.T) {
	c := DefaultConfig()
	var nobles []Noble
	for i := range 20 {
		nobles = append(nobles, Noble{CharID: int32(100 + i), ClassID: 88, Name: string(rune('a' + i)), Points: 100 - i, Done: 9 + i%2})
	}
	nobles = append(nobles, Noble{CharID: 200, ClassID: 88, Name: "short", Points: 500, Done: 3})
	tab := NewNobles(c, nobles, nil)
	if got := tab.ClassLeaders(88, false); len(got) != 10 || got[0] != "a" || got[9] != "j" {
		t.Errorf("ClassLeaders running = %v", got)
	}
	if got := tab.ClassLeaders(88, true); len(got) != 0 {
		t.Errorf("ClassLeaders monthly before a month end = %v", got)
	}
	tab.CopyToMonthEnd()
	if got := tab.ClassLeaders(88, true); len(got) != 10 {
		t.Errorf("ClassLeaders monthly = %v", got)
	}
	// 20 classified: 1 % rounds to 0, so the first is rank 1 and the bounds
	// move up by one: 2 (10 %), 5 (25 %), 10 (50 %).
	ranks := tab.Ranks()
	want := map[int32]int{}
	for i := range 20 {
		place := i + 1
		r := 5
		switch {
		case place <= 1:
			r = 1
		case place <= 3:
			r = 2
		case place <= 6:
			r = 3
		case place <= 11:
			r = 4
		}
		want[int32(100+i)] = r
	}
	if !maps.Equal(ranks, want) {
		t.Errorf("Ranks = %v, want %v", ranks, want)
	}
	// Settling: only heroes earn passes (HeroPoints × GPPerPoint); every
	// ranked noble with points loses them; the unranked keep theirs.
	passes := tab.SettlePasses(c, Validation, ranks, func(id int32) bool { return id == 101 })
	if !maps.Equal(passes, map[int32]int{101: 100000}) {
		t.Errorf("SettlePasses = %v", passes)
	}
	// (200's 500 points were clamped to 10 × 3 + 12 at load.)
	if tab.Points(100) != 0 || tab.Points(101) != 0 || tab.Points(200) != 42 {
		t.Errorf("points after settling: %d %d %d", tab.Points(100), tab.Points(101), tab.Points(200))
	}
	if got := tab.SettlePasses(c, Competition, ranks, func(int32) bool { return true }); len(got) != 0 {
		t.Errorf("SettlePasses in the competition = %v", got)
	}
}
