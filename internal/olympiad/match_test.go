package olympiad

import (
	"slices"
	"testing"
	"time"
)

func TestPointDiff(t *testing.T) {
	c := DefaultConfig()
	for _, tc := range []struct {
		typ  CompetitionType
		a, b int
		want int
	}{
		{NonClassed, 18, 30, 3},
		{Classed, 18, 30, 6},
		{Classed, 100, 90, 10},
		{NonClassed, 4, 90, 0},
	} {
		if got := c.PointDiff(tc.typ, tc.a, tc.b); got != tc.want {
			t.Errorf("PointDiff(%v, %d, %d) = %d, want %d", tc.typ, tc.a, tc.b, got, tc.want)
		}
	}
}

func sides(p1, p2 int) [2]Side {
	return [2]Side{
		{Noble: &Noble{CharID: 1, Points: p1}, Name: "One", Present: true, HP: 1000},
		{Noble: &Noble{CharID: 2, Points: p2}, Name: "Two", Present: true, HP: 900},
	}
}

func TestValidate(t *testing.T) {
	c := DefaultConfig()

	// A kill: the dead noble loses a fifth of the lower points, the winner
	// gets them, 30 gate passes and the messages.
	s := sides(18, 30)
	s[1].Dead = true
	r := c.Validate(NonClassed, s)
	if r.Kind != ResultWin || r.Winner != 0 || r.Reward != 30 || r.Fight != 1 || !r.Countdown || r.PointDiff != 3 {
		t.Errorf("kill = %+v", r)
	}
	if n := s[0].Noble; n.Points != 21 || n.Won != 1 || n.Done != 1 {
		t.Errorf("winner %+v", n)
	}
	if n := s[1].Noble; n.Points != 27 || n.Lost != 1 || n.Done != 1 {
		t.Errorf("loser %+v", n)
	}
	wantNotices := []Notice{{MsgYouWin, "One", 0, true}, {MsgEarnedPoints, "One", 3, false}, {MsgLostPoints, "Two", 3, false}}
	if !slices.Equal(r.Notices, wantNotices) {
		t.Errorf("notices %+v", r.Notices)
	}

	// Time out, both standing: more damage wins; classed gives 50 passes.
	s = sides(30, 30)
	s[1].Damage = 500
	s[0].Damage = 100
	r = c.Validate(Classed, s)
	if r.Kind != ResultWin || r.Winner != 1 || r.Reward != 50 || r.Fight != 2 || s[1].Noble.Points != 40 || s[0].Noble.Points != 20 {
		t.Errorf("damage win = %+v, nobles %+v %+v", r, s[0].Noble, s[1].Noble)
	}

	// Equal damage: a tie; each loses a fifth of its own points.
	s = sides(30, 12)
	r = c.Validate(Classed, s)
	if r.Kind != ResultTie || r.Fight != 0 || s[0].Noble.Points != 24 || s[1].Noble.Points != 10 || s[0].Noble.Drawn != 1 || s[1].Noble.Done != 1 {
		t.Errorf("tie = %+v, nobles %+v %+v", r, s[0].Noble, s[1].Noble)
	}

	// The second noble logged out after the match: the first wins.
	s = sides(30, 30)
	s[1].Present = false
	if r = c.Validate(NonClassed, s); r.Kind != ResultWin || r.Winner != 0 {
		t.Errorf("absent = %+v", r)
	}
	// Both gone: a tie without points and without a saved fight.
	s = sides(30, 30)
	s[0].Present, s[1].Present = false, false
	if r = c.Validate(NonClassed, s); r.Kind != ResultTie || r.Fight != -1 || s[0].Noble.Points != 30 || s[0].Noble.Drawn != 1 {
		t.Errorf("both absent = %+v", r)
	}

	// A crash during the fight: the other wins the points, no reward, no
	// countdown.
	s = sides(30, 30)
	s[0].Crashed = true
	r = c.Validate(Classed, s)
	if r.Kind != ResultCrash || r.Winner != 1 || r.Reward != 0 || r.Countdown || s[0].Noble.Points != 20 || s[1].Noble.Points != 40 ||
		s[0].Noble.Lost != 1 || s[1].Noble.Won != 1 || s[0].Noble.Done != 1 || s[1].Noble.Done != 1 {
		t.Errorf("crash = %+v, nobles %+v %+v", r, s[0].Noble, s[1].Noble)
	}
	s = sides(30, 30)
	s[0].Crashed, s[1].Crashed = true, true
	if r = c.Validate(Classed, s); r.Winner != -1 || s[0].Noble.Points != 20 || s[1].Noble.Points != 20 || len(r.Notices) != 0 {
		t.Errorf("both crashed = %+v", r)
	}

	// A default: the noble loses a third of its points (at most 10), no
	// match counts.
	s = sides(45, 18)
	s[0].Defaulted = true
	r = c.Validate(NonClassed, s)
	if r.Kind != ResultDefault || s[0].Noble.Points != 35 || s[1].Noble.Points != 18 || s[0].Noble.Done != 0 ||
		!slices.Equal(r.Notices, []Notice{{MsgLostPoints, "One", 10, false}}) {
		t.Errorf("default = %+v, noble %+v", r, s[0].Noble)
	}
}

func TestCountdowns(t *testing.T) {
	secs := func(ticks []Tick) (out []int, total time.Duration) {
		for _, x := range ticks {
			out = append(out, x.Seconds)
			total += x.Wait
		}
		return out, total
	}
	c := DefaultConfig()
	if got, total := secs(c.TeleportCountdown()); !slices.Equal(got, []int{120, 60, 30, 15, 10, 5, 4, 3, 2, 1}) || total != 120*time.Second {
		t.Errorf("TeleportCountdown = %v over %v", got, total)
	}
	if got, total := secs(StartCountdown()); !slices.Equal(got, []int{60, 50, 40, 30, 20, 10, 5, 4, 3, 2, 1}) || total != 60*time.Second {
		t.Errorf("StartCountdown = %v over %v", got, total)
	}
	if got, total := secs(BackCountdown()); !slices.Equal(got, []int{40, 30, 20, 10, 5, 4, 3, 2, 1}) || total != 40*time.Second {
		t.Errorf("BackCountdown = %v over %v", got, total)
	}
	c.WaitTime = 100
	if got, total := secs(c.TeleportCountdown()); !slices.Equal(got, []int{100, 40}) || total != 120*time.Second {
		t.Errorf("TeleportCountdown(100) = %v over %v", got, total)
	}
}
