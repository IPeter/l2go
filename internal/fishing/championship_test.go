package fishing

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/calendar"
	"github.com/IPeter/l2go/internal/rnd"
)

var utc = calendar.New(time.UTC)

// TestEndOfWeek checks setEndOfChamp from a Tuesday end and from the first
// start, with Java's Sunday weeks and with Monday weeks.
func TestEndOfWeek(t *testing.T) {
	tue := time.Date(2026, 11, 24, 19, 0, 0, 123_000_000, time.UTC) // a Tuesday
	monday := utc.WithFirstDayOfWeek(time.Monday)
	for _, c := range []struct {
		name string
		cal  calendar.Calendar
		from time.Time
		want time.Time
	}{
		{"a week after the end", utc, tue, tue.AddDate(0, 0, 7)},
		{"first start on a Thursday", utc, time.Date(2026, 11, 26, 13, 37, 5, 0, time.UTC), time.Date(2026, 12, 1, 19, 0, 0, 0, time.UTC)},
		// Monday + 6 days is a Sunday: the Tuesday of its (Sunday) week is
		// eight days on.
		{"first start on a Monday", utc, time.Date(2026, 11, 23, 13, 0, 0, 0, time.UTC), time.Date(2026, 12, 1, 19, 0, 0, 0, time.UTC)},
		{"Monday weeks: Monday start", monday, time.Date(2026, 11, 23, 13, 0, 0, 0, time.UTC), time.Date(2026, 11, 24, 19, 0, 0, 0, time.UTC)},
	} {
		if got := EndOfWeek(c.cal, c.from); !got.Equal(c.want) {
			t.Errorf("%s: EndOfWeek(%v) = %v, want %v", c.name, c.from, got, c.want)
		}
	}
}

// catch draws a fish of length whole + frac/1000 for name.
func catch(t *testing.T, c *Championship, name string, lure, whole, frac int, extra ...int) Catch {
	t.Helper()
	draws := []rnd.Draw{rnd.Between(60, 89, whole), rnd.Between(0, 1000, frac)}
	if len(extra) > 0 {
		draws = append(draws, rnd.Between(0, 3000, extra[0]))
	}
	r := rnd.NewScripted(t, draws...)
	got, ok := c.NewFish(r, name, lure)
	if !ok {
		t.Fatal("NewFish: the championship is off")
	}
	r.Done()
	return got
}

// TestChampionshipRanking checks newFish: the first five enter, a fisher
// keeps the longer fish, a longer fish replaces the shortest, and the
// prize lures add up to 3.
func TestChampionshipRanking(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	c, finished := NewChampionship(DefaultChampionshipConfig(), ChampionshipState{End: now.Add(time.Hour)}, now, utc)
	if finished {
		t.Fatal("a running week finished")
	}
	for i, name := range []string{"Ann", "Bob", "Cid", "Dan", "Eve"} {
		if got := catch(t, c, name, 6519, 70+i, 0); !got.Registered || got.Length != float64(70+i) {
			t.Errorf("%s: %+v, want registered", name, got)
		}
	}
	if got := catch(t, c, "ann", 6519, 60, 0); got.Registered {
		t.Error("a shorter fish of a ranked fisher registered")
	}
	if got := catch(t, c, "ANN", 6519, 80, 500); !got.Registered {
		t.Error("a longer fish of a ranked fisher did not register")
	}
	if got := catch(t, c, "Fay", 6519, 65, 0); got.Registered {
		t.Error("a fish shorter than the shortest ranked one registered")
	}
	// Bob (71) is the shortest now; Fay's prize-lure fish of 72.5 + 1 beats him.
	if got := catch(t, c, "Fay", 8484, 72, 500, 1000); !got.Registered || got.Length != 73.5 {
		t.Errorf("Fay: %+v, want 73.5 and registered", got)
	}
	running, _ := c.Ranking()
	var names []string
	for _, f := range running {
		names = append(names, f.Name)
	}
	if want := []string{"Ann", "Cid", "Dan", "Eve", "Fay"}; !slices.Equal(names, want) {
		t.Errorf("ranked %v, want %v", names, want)
	}

	// The first look refreshes (fish_event003); the second shows the table.
	if _, stale := c.MidResult(); !stale {
		t.Error("the first MidResult did not refresh")
	}
	table, stale := c.MidResult()
	if stale || !strings.HasPrefix(table, "null<tr><td width=70 align=center>1</td><td width=110 align=center>Ann</td><td width=80 align=center>80.5</td></tr>") ||
		!strings.Contains(table, "<td width=110 align=center>Fay</td><td width=80 align=center>73.5</td>") {
		t.Errorf("MidResult = %q, %v", table, stale)
	}

	// Disabled: nothing is drawn.
	off := DefaultChampionshipConfig()
	off.Enabled = false
	c2, _ := NewChampionship(off, ChampionshipState{End: now.Add(time.Hour)}, now, utc)
	if _, ok := c2.NewFish(rnd.NewScripted(t), "Ann", 6519); ok {
		t.Error("NewFish of a disabled championship")
	}
}

// TestChampionshipWeekAndRewards runs the weekly end on a fake week and a
// restart: the ranking becomes the winners, they claim once, and a restart
// after a missed end finishes at once.
func TestChampionshipWeekAndRewards(t *testing.T) {
	start := time.Date(2026, 11, 26, 13, 0, 0, 0, time.UTC) // a Thursday
	c, finished := NewChampionship(DefaultChampionshipConfig(), ChampionshipState{}, start, utc)
	if !finished {
		t.Fatal("a first start without an end did not finish the (empty) week")
	}
	if want := time.Date(2026, 12, 1, 19, 0, 0, 0, time.UTC); !c.End().Equal(want) {
		t.Fatalf("End = %v, want %v", c.End(), want)
	}
	catch(t, c, "Ann", 6519, 70, 0)
	catch(t, c, "Bob", 6519, 85, 250)
	if c.IsWinner("Bob") {
		t.Error("a fisher of the running week is a winner")
	}
	if got := c.MinutesLeft(start); got != 5*24*60+6*60 {
		t.Errorf("MinutesLeft = %d", got)
	}

	// The week ends: Bob first, Ann second; the next end is a week later.
	c.Finish(utc)
	if want := time.Date(2026, 12, 8, 19, 0, 0, 0, time.UTC); !c.End().Equal(want) {
		t.Errorf("End after the week = %v, want %v", c.End(), want)
	}
	if !c.IsWinner("Bob") || c.IsWinner("bob") {
		t.Error("isWinner compares names exactly")
	}
	if !strings.Contains(c.WinnerTable(), "<td width=110 align=center>Bob</td><td width=80 align=center>85.25</td></tr><tr><td width=70 align=center>2</td><td width=110 align=center>Ann</td>") {
		t.Errorf("WinnerTable = %q", c.WinnerTable())
	}

	// A restart keeps the claims; Ann claims after it.
	if got := c.Reward("bob"); !slices.Equal(got, []int{800000}) {
		t.Errorf("Reward(bob) = %v, want [800000]", got)
	}
	if got := c.Reward("Bob"); got != nil {
		t.Errorf("a second claim gave %v", got)
	}
	st := c.State()
	if len(st.Fishers) != 2 || st.Fishers[0] != (Fisher{Name: "Bob", Length: 85.25, Reward: RewardTaken}) || st.Fishers[1].Reward != RewardDue {
		t.Fatalf("State = %+v", st)
	}
	restart := start.Add(48 * time.Hour)
	c2, finished := NewChampionship(DefaultChampionshipConfig(), st, restart, utc)
	if finished {
		t.Fatal("a restart before the end finished the week")
	}
	if got := c2.Reward("Ann"); !slices.Equal(got, []int{500000}) {
		t.Errorf("Reward(Ann) after a restart = %v, want [500000]", got)
	}
	if got := c2.Reward("Bob"); got != nil {
		t.Errorf("Bob claimed again after a restart: %v", got)
	}

	// The server was down over the end: it finishes at start, from then.
	catch(t, c2, "Cid", 6519, 61, 0)
	late := c2.End().Add(3 * 24 * time.Hour) // Friday
	c3, finished := NewChampionship(DefaultChampionshipConfig(), c2.State(), late, utc)
	if !finished || !c3.IsWinner("Cid") || c3.IsWinner("Ann") {
		t.Errorf("missed end: finished %v, Cid winner %v, Ann winner %v", finished, c3.IsWinner("Cid"), c3.IsWinner("Ann"))
	}
	if want := time.Date(2026, 12, 15, 19, 0, 0, 0, time.UTC); !c3.End().Equal(want) {
		t.Errorf("End after a missed week = %v, want %v", c3.End(), want)
	}
}
