package residence

import (
	"reflect"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/calendar"
)

var berlin = mustLoad("Europe/Berlin")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func TestNextSiegeDate(t *testing.T) {
	cal := calendar.New(berlin)
	sun16 := Schedule{Enabled: true, Day: time.Sunday, Hour: 16, MaxConcurrent: 5}
	at := func(y int, m time.Month, d, h, min int) time.Time { return time.Date(y, m, d, h, min, 0, 0, berlin) }
	for _, tc := range []struct {
		name          string
		now, current  time.Time
		concurrent    int
		want          time.Time
		open          bool
		sched         Schedule
		cycle         int
		ok            bool
		keepsMillisOf time.Time
	}{
		// Java weeks start on Sunday: on a Thursday, "Sunday" is the one
		// before, which has passed, so the siege is two weeks after it.
		{name: "Thursday: the Sunday of the week has passed", now: at(2026, 11, 26, 12, 0), want: at(2026, 12, 6, 16, 0), open: true, ok: true},
		{name: "Sunday before the hour", now: at(2026, 11, 29, 9, 0), want: at(2026, 11, 29, 16, 0), open: true, ok: true},
		{name: "Sunday after the hour: two weeks on", now: at(2026, 11, 29, 17, 0), want: at(2026, 12, 13, 16, 0), open: true, ok: true},
		{name: "a stored future date moves to its week's Sunday", now: at(2026, 11, 26, 12, 0), current: at(2026, 12, 17, 9, 30), want: at(2026, 12, 13, 16, 0), open: true, ok: true},
		{name: "full: the date is set, registration closed", now: at(2026, 11, 26, 12, 0), concurrent: 5, want: at(2026, 12, 6, 16, 0), ok: true},
		{name: "across the end of daylight saving time", now: at(2026, 10, 25, 17, 0), want: at(2026, 11, 8, 16, 0), open: true, ok: true},
		{name: "disabled", now: at(2026, 11, 26, 12, 0), sched: Schedule{Day: time.Sunday, Hour: 16}, ok: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := sun16
			if tc.sched != (Schedule{}) {
				s = tc.sched
			}
			cycle := 2
			date, open, ok := NextSiegeDate(cal, tc.now, tc.current, s, cycle, func(time.Time) int { return tc.concurrent })
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !ok {
				return
			}
			if !date.Equal(tc.want) || open != tc.open {
				t.Errorf("NextSiegeDate = %v, %v; want %v, %v", date, open, tc.want, tc.open)
			}
		})
	}
}

func TestNextStartStep(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	siege := now.Add(72 * time.Hour)
	for _, tc := range []struct {
		name    string
		siege   time.Time
		regEnd  time.Time
		regOver bool
		want    StartStep
	}{
		{"the lord may still choose", siege, now.Add(time.Hour), false, StartStep{Wait: time.Hour}},
		{"choice over: wait for the 24 h mark", siege, now.Add(-time.Hour), false, StartStep{EndTimeRegistration: true, Wait: 48 * time.Hour}},
		{"24 h mark: registration closes", now.Add(24 * time.Hour), time.Time{}, true, StartStep{CloseRegistration: true, Wait: 24*time.Hour - startLastHour}},
		{"the Mobius hour", now.Add(startLastHour), time.Time{}, true, StartStep{Wait: startLastHour - 10*time.Minute}},
		{"10 minutes", now.Add(10 * time.Minute), time.Time{}, true, StartStep{Wait: 5 * time.Minute}},
		{"5 minutes", now.Add(5 * time.Minute), time.Time{}, true, StartStep{Wait: 5*time.Minute - 10*time.Second}},
		{"10 seconds", now.Add(10 * time.Second), time.Time{}, true, StartStep{Wait: 10 * time.Second}},
		{"start", now, time.Time{}, true, StartStep{Start: true}},
		{"late start", now.Add(-time.Minute), time.Time{}, true, StartStep{Start: true}},
	} {
		if got := NextStartStep(now, tc.siege, tc.regEnd, tc.regOver); got != tc.want {
			t.Errorf("%s: NextStartStep = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestNextEndStep(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		left time.Duration
		want EndStep
	}{
		{120 * time.Minute, EndStep{Msg: MsgHoursUntilConclusion, Param: 2, Wait: time.Hour}},
		{time.Hour, EndStep{Msg: MsgMinutesUntilConclusion, Param: 60, Wait: 50 * time.Minute}},
		{10 * time.Minute, EndStep{Msg: MsgMinutesUntilConclusion, Param: 10, Wait: 5 * time.Minute}},
		{5 * time.Minute, EndStep{Msg: MsgMinutesUntilConclusion, Param: 5, Wait: 5*time.Minute - 10*time.Second}},
		{10 * time.Second, EndStep{Msg: MsgSecondsLeft, Param: 10, Wait: 10 * time.Second}},
		{0, EndStep{End: true}},
	} {
		if got := NextEndStep(now, now.Add(tc.left)); got != tc.want {
			t.Errorf("%v left: NextEndStep = %+v, want %+v", tc.left, got, tc.want)
		}
	}
}

func TestCheckRegister(t *testing.T) {
	ok := RegisterCheck{ClanID: 7, ClanLevel: 4, Attacker: true, MinLevel: 4, MaxAttackers: 1, MaxDefenders: 1}
	var full Registry
	full.AddAttacker(9)
	full.AddWaiting(10)
	for _, tc := range []struct {
		name string
		r    *Registry
		mod  func(*RegisterCheck)
		want int
	}{
		{"ok", &Registry{}, func(*RegisterCheck) {}, 0},
		{"deadline", &Registry{}, func(c *RegisterCheck) { c.RegistrationOver = true }, MsgRegistrationDeadline},
		{"in progress", &Registry{}, func(c *RegisterCheck) { c.InProgress = true }, MsgNotTimeToRegister},
		{"no clan", &Registry{}, func(c *RegisterCheck) { c.ClanID = 0 }, MsgClanLevelTooLow},
		{"level 3", &Registry{}, func(c *RegisterCheck) { c.ClanLevel = 3 }, MsgClanLevelTooLow},
		{"the owner", &Registry{}, func(c *RegisterCheck) { c.OwnerID = 7 }, MsgOwnerIsDefender},
		{"owns a castle", &Registry{}, func(c *RegisterCheck) { c.ClanCastleID = 3 }, MsgOwnsAnotherCastle},
		{"registered", &Registry{}, func(c *RegisterCheck) { c.Registered = true }, MsgAlreadyRequested},
		{"same day", &Registry{}, func(c *RegisterCheck) { c.SameDay = true }, MsgAnotherSiegeSameDay},
		{"attackers full", &full, func(*RegisterCheck) {}, MsgNoMoreAttackers},
		{"defenders full", &full, func(c *RegisterCheck) { c.Attacker = false }, MsgNoMoreDefenders},
	} {
		c := ok
		tc.mod(&c)
		msg, good := tc.r.CheckRegister(c)
		if msg != tc.want || good != (tc.want == 0) {
			t.Errorf("%s: CheckRegister = %d, %v; want %d", tc.name, msg, good, tc.want)
		}
	}
}

func ids(list []*SiegeClan) []int32 {
	var out []int32
	for _, c := range list {
		out = append(out, c.ClanID)
	}
	return out
}

func TestRegistryLoadRows(t *testing.T) {
	var r Registry
	r.Load(1, []Registration{{ClanID: 2, Type: RegAttacker}, {ClanID: 3, Type: RegDefender}, {ClanID: 4, Type: RegDefenderPending}})
	if got := ids(r.Defenders()); !reflect.DeepEqual(got, []int32{1, 3}) {
		t.Errorf("defenders = %v, want the owner first", got)
	}
	if !r.Approve(4) || r.IsWaiting(4) || !r.IsDefender(4) {
		t.Errorf("Approve(4) did not move the clan")
	}
	want := []Registration{{ClanID: 2, Type: RegAttacker}, {ClanID: 3, Type: RegDefender}, {ClanID: 4, Type: RegDefender}}
	if got := r.Rows(); !reflect.DeepEqual(got, want) {
		t.Errorf("Rows = %v, want %v (the owner is not stored)", got, want)
	}
	r.Remove(3)
	if r.Registered(3) {
		t.Errorf("Remove(3) left it registered")
	}
}

func TestMidVictory(t *testing.T) {
	allies := map[int32]int32{10: 100, 11: 100, 12: 0, 13: 100}
	allyOf := func(id int32) int32 { return allies[id] }

	t.Run("one attacker against the NPCs ends the siege", func(t *testing.T) {
		var r Registry
		r.AddAttacker(10)
		if !r.MidVictory(10, allyOf, nil) {
			t.Fatal("the siege goes on")
		}
		if d := r.Defenders(); len(d) != 1 || d[0].ClanID != 10 || d[0].Type != Owner {
			t.Errorf("defenders = %+v, want clan 10 as owner", d)
		}
	})
	t.Run("only the new owner's alliance attacks", func(t *testing.T) {
		var r Registry
		r.AddAttacker(10)
		r.AddAttacker(11)
		if !r.MidVictory(10, allyOf, []int32{10, 11, 13}) {
			t.Fatal("the siege goes on")
		}
	})
	t.Run("sides swap", func(t *testing.T) {
		var r Registry
		r.AddDefender(1, Owner)
		r.AddDefender(2, Defender)
		r.AddAttacker(10)
		r.AddAttacker(11)
		r.AddAttacker(12)
		r.Attacker(12).Flags = []int32{500}
		if r.MidVictory(10, allyOf, []int32{10, 11, 13}) {
			t.Fatal("the siege ends")
		}
		if got := ids(r.Attackers()); !reflect.DeepEqual(got, []int32{12, 1, 2}) {
			t.Errorf("attackers = %v, want 12 and the old defenders", got)
		}
		if got := ids(r.Defenders()); !reflect.DeepEqual(got, []int32{10, 11}) {
			t.Errorf("defenders = %v, want the new owner and its ally", got)
		}
		if r.Defender(10).Type != Owner || r.Defender(11).Type != Defender || r.Attacker(1).Type != Attacker {
			t.Errorf("types not updated")
		}
		if fc := r.FlagOwner(500); fc == nil || fc.ClanID != 12 {
			t.Errorf("the flag lost its clan")
		}
	})
}

func TestSiegeReputation(t *testing.T) {
	pts := DefaultReputationPoints
	for _, tc := range []struct {
		name        string
		former      int32
		formerScore int
		owner       int32
		want        []ReputationChange
	}{
		{"taken from the NPCs", 0, 0, 5, []ReputationChange{{ClanID: 5, Delta: 1500, Msg: MsgReputationWon, Param: 1500}}},
		{"taken from a clan", 4, 900, 5, []ReputationChange{{ClanID: 4, Delta: -3000, Msg: MsgReputationLost, Param: 3000}, {ClanID: 5, Delta: 900, Msg: MsgReputationWon, Param: 900}}},
		{"taken from a clan in debt", 4, -10, 5, []ReputationChange{{ClanID: 4, Delta: -3000, Msg: MsgReputationLost, Param: 3000}, {ClanID: 5, Delta: 0, Msg: MsgReputationWon, Param: 0}}},
		{"defended", 4, 900, 4, []ReputationChange{{ClanID: 4, Delta: 750, Msg: MsgReputationWon, Param: 750}}},
		{"draw", 0, 0, 0, nil},
	} {
		if got := SiegeReputation(tc.former, tc.formerScore, tc.owner, pts); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: SiegeReputation = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestTreasuryShares(t *testing.T) {
	castles := map[string]TreasuryState{
		"aden":       {ID: 5, Name: "Aden", OwnerID: 50, TaxRate: 0.1},
		"rune":       {ID: 8, Name: "Rune", OwnerID: 0, TaxRate: 0.2},
		"gludio":     {ID: 1, Name: "Gludio", OwnerID: 10},
		"schuttgart": {ID: 9, Name: "Schuttgart", OwnerID: 90},
	}
	byName := func(n string) (TreasuryState, bool) { c, ok := castles[n]; return c, ok }
	if got, want := TreasuryShares(castles["gludio"], 1000, byName), map[int]int64{5: 100, 1: 900}; !reflect.DeepEqual(got, want) {
		t.Errorf("Gludio = %v, want %v", got, want)
	}
	// Rune has no owner: it takes its 20 % but keeps nothing.
	if got, want := TreasuryShares(castles["schuttgart"], 1000, byName), map[int]int64{9: 800}; !reflect.DeepEqual(got, want) {
		t.Errorf("Schuttgart = %v, want %v", got, want)
	}
	if got := TreasuryShares(TreasuryState{ID: 2, Name: "Dion"}, 1000, byName); len(got) != 0 {
		t.Errorf("an unowned castle keeps %v", got)
	}
	if v, ok := AddNoTax(true, MaxAdena-5, 10); !ok || v != MaxAdena {
		t.Errorf("AddNoTax over the cap = %d, %v", v, ok)
	}
	if v, ok := AddNoTax(true, 5, -10); ok || v != 5 {
		t.Errorf("AddNoTax below 0 = %d, %v", v, ok)
	}
}
