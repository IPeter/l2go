package quest

import (
	"errors"
	"maps"
	"strconv"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

func TestStartQuest(t *testing.T) {
	b, r := loaded(t)
	st := b.State(warrior, true)
	if !st.IsCreated() || b.Dirty() {
		t.Fatalf("new state: %v, dirty %v; want Created and nothing to save", st.State(), b.Dirty())
	}
	st.StartQuest()
	// Mobius: set("cond", "1"), setState(STARTED), then the accept sound.
	checkEvents(t, r, "QuestList", "QuestMark 401", "QuestList", "Sound "+SoundAccept)
	if !st.IsStarted() || st.Cond() != 1 {
		t.Errorf("after StartQuest: %v, cond %d; want Started, 1", st.State(), st.Cond())
	}
	if r.changed != 1 {
		t.Errorf("Changed called %d times, want once", r.changed)
	}
	ch := b.TakeChanges()
	want := storage.QuestChanges{Upserts: []storage.Quest{{Name: warrior, Vars: vars("<state>", "Started", "cond", "1")}}}
	if !changesEqual(ch, want) {
		t.Errorf("TakeChanges = %+v, want %+v", ch, want)
	}

	// Only from Created.
	st.StartQuest()
	checkEvents(t, r)
	if b.Dirty() {
		t.Error("a second StartQuest changed something")
	}
}

func TestStartQuestRefused(t *testing.T) {
	for _, name := range []string{custom, gone} {
		b, r := loaded(t)
		st := b.State(name, true)
		st.StartQuest()
		checkEvents(t, r)
		if !st.IsCreated() || b.Dirty() {
			t.Errorf("%s: StartQuest made it %v, dirty %v; want nothing", name, st.State(), b.Dirty())
		}
	}
	b, r := loaded(t, storage.Quest{Name: letters, Vars: vars("<state>", "Completed")})
	b.Get(letters).StartQuest()
	checkEvents(t, r)
	if !b.Get(letters).IsCompleted() {
		t.Error("StartQuest restarted a completed quest")
	}
}

func TestSetCond(t *testing.T) {
	b, r := loaded(t, storage.Quest{Name: warrior, Vars: vars("<state>", "Started", "cond", "1")})
	st := b.Get(warrior)
	st.SetCond(2, true)
	checkEvents(t, r, "QuestList", "QuestMark 401", "Sound "+SoundMiddle)
	st.SetCond(3, false)
	checkEvents(t, r, "QuestList", "QuestMark 401")
	// The same step plays the sound but sends no list.
	st.SetCond(3, true)
	checkEvents(t, r, "Sound "+SoundMiddle)

	// SetCond needs a started quest; Set does not.
	c := b.State(letters, true)
	c.SetCond(2, true)
	checkEvents(t, r)
	if c.IsSet(VarCond) || c.IsCond(2) {
		t.Error("SetCond changed a Created quest")
	}
	c.Set(VarCond, "2")
	checkEvents(t, r, "QuestList", "QuestMark 1")
	if c.Cond() != 0 || !c.IsCond(2) {
		t.Errorf("Created quest with cond 2: Cond() %d, IsCond(2) %v; want 0, true", c.Cond(), c.IsCond(2))
	}
}

func TestVariables(t *testing.T) {
	b, r := loaded(t)
	st := b.State(warrior, true)
	if _, ok := st.Get("x"); ok || st.IsSet("x") || st.GetInt("x") != 0 || st.Vars() != nil {
		t.Error("a new state has variables")
	}
	st.Set("x", "12")
	st.Set("empty", "")
	st.SetInt("neg", -3)
	st.Set("word", "abc")
	for name, want := range map[string]int32{"x": 12, "empty": 0, "neg": -3, "word": 0, "missing": 0} {
		if got := st.GetInt(name); got != want {
			t.Errorf("GetInt(%s) = %d, want %d", name, got, want)
		}
	}
	if v, ok := st.Get("empty"); !ok || v != "" {
		t.Errorf("Get(empty) = %q, %v; want set and empty", v, ok)
	}
	st.Unset("x")
	st.Unset("never")
	if st.IsSet("x") {
		t.Error("Unset left the variable")
	}
	want := vars("empty", "", "neg", "-3", "word", "abc")
	if got := st.Vars(); !maps.Equal(got, want) {
		t.Errorf("Vars() = %v, want %v", got, want)
	}
	checkEvents(t, r)
	// Variables of a Created state are saved without <state>.
	ch := b.TakeChanges()
	if len(ch.Upserts) != 1 || !maps.Equal(ch.Upserts[0].Vars, want) {
		t.Errorf("TakeChanges = %+v, want the variables without <state>", ch)
	}

	// Unsetting the step sets it to 0 without packets.
	s, r2 := loaded(t, storage.Quest{Name: warrior, Vars: vars("<state>", "Started", "cond", "3")})
	s.Get(warrior).Unset(VarCond)
	checkEvents(t, r2)
	if s.Get(warrior).IsCond(3) || !s.Get(warrior).IsCond(0) {
		t.Error("Unset(cond) kept the step")
	}
}

func TestMemoState(t *testing.T) {
	b, _ := loaded(t)
	st := b.State(warrior, true)
	st.SetMemoState(5)
	st.SetMemoStateEx(1, 7)
	// Created: the getters say 0, IsMemoState does not care, IsMemoStateEx
	// goes through the getter.
	if st.MemoState() != 0 || !st.IsMemoState(5) || st.MemoStateEx(1) != 0 || st.IsMemoStateEx(1, 7) || !st.IsMemoStateEx(1, 0) {
		t.Errorf("Created: MemoState %d, IsMemoState(5) %v, MemoStateEx(1) %d, IsMemoStateEx(1, 7) %v",
			st.MemoState(), st.IsMemoState(5), st.MemoStateEx(1), st.IsMemoStateEx(1, 7))
	}
	st.StartQuest()
	if st.MemoState() != 5 || st.MemoStateEx(1) != 7 || !st.IsMemoStateEx(1, 7) {
		t.Errorf("Started: MemoState %d, MemoStateEx(1) %d", st.MemoState(), st.MemoStateEx(1))
	}
	if v, _ := st.Get("memoStateEx1"); v != "7" {
		t.Errorf("memoStateEx1 = %q, want 7", v)
	}
	st.SetMemoStateEx(-2, 1)
	if !st.IsSet("memoStateEx-2") {
		t.Error("memo state ex of slot -2 is not memoStateEx-2")
	}
}

func TestExitQuestOneTime(t *testing.T) {
	b, r := loaded(t, storage.Quest{Name: warrior, Vars: vars("<state>", "Started", "cond", "3", "memoState", "2")})
	st := b.Get(warrior)
	st.ExitQuest(false, true)
	checkEvents(t, r,
		"RemoveDeathNotify "+warrior, "RemoveQuestItems "+warrior, "QuestList", "Sound "+SoundFinish)
	if !st.IsCompleted() || b.Get(warrior) != st || st.Detached() {
		t.Errorf("after ExitQuest(false): %v, in book %v", st.State(), b.Get(warrior) == st)
	}
	// The variables go; the step stays in memory, as in Mobius.
	if st.Vars() != nil || st.Cond() != 0 || !st.IsCond(3) {
		t.Errorf("after ExitQuest(false): vars %v, Cond() %d, IsCond(3) %v; want none, 0, true", st.Vars(), st.Cond(), st.IsCond(3))
	}
	want := storage.QuestChanges{Upserts: []storage.Quest{{Name: warrior, Vars: vars("<state>", "Completed")}}}
	if ch := b.TakeChanges(); !changesEqual(ch, want) {
		t.Errorf("TakeChanges = %+v, want %+v", ch, want)
	}
	// Exiting again does nothing but the death notification.
	st.ExitQuest(false, false)
	checkEvents(t, r, "RemoveDeathNotify "+warrior)
	if b.Dirty() {
		t.Error("exiting a completed quest changed something")
	}
}

func TestExitQuestRepeatable(t *testing.T) {
	b, r := loaded(t, storage.Quest{Name: warrior, Vars: vars("<state>", "Started", "cond", "3")})
	st := b.Get(warrior)
	st.ExitQuest(true, true)
	checkEvents(t, r,
		"RemoveDeathNotify "+warrior, "RemoveQuestItems "+warrior, "QuestList", "Sound "+SoundFinish)
	if b.Get(warrior) != nil || !st.Detached() {
		t.Error("ExitQuest(true) left the state in the book")
	}
	// Mobius leaves the object as it was, without variables.
	if !st.IsStarted() || st.Vars() != nil {
		t.Errorf("after ExitQuest(true): %v, vars %v; want Started without variables", st.State(), st.Vars())
	}
	want := storage.QuestChanges{Deleted: []string{warrior}}
	if ch := b.TakeChanges(); !changesEqual(ch, want) {
		t.Errorf("TakeChanges = %+v, want %+v", ch, want)
	}
	// The detached state's changes are not saved; a new state is.
	st.Set("late", "1")
	if b.Dirty() {
		t.Error("a change to a detached state was saved")
	}
	again := b.State(warrior, true)
	if again == st || !again.IsCreated() {
		t.Error("State(create) after a repeatable exit did not make a new state")
	}
	again.StartQuest()
	want = storage.QuestChanges{Upserts: []storage.Quest{{Name: warrior, Vars: vars("<state>", "Started", "cond", "1")}}}
	if ch := b.TakeChanges(); !changesEqual(ch, want) {
		t.Errorf("TakeChanges after taking the quest again = %+v, want %+v", ch, want)
	}
}

func TestExitQuestNotStarted(t *testing.T) {
	b, r := loaded(t)
	st := b.State(warrior, true)
	st.Set("x", "1")
	b.TakeChanges()
	st.ExitQuest(true, true)
	checkEvents(t, r, "RemoveDeathNotify "+warrior, "Sound "+SoundFinish)
	if b.Get(warrior) != st || !st.IsSet("x") || b.Dirty() {
		t.Error("ExitQuest of a Created quest changed it")
	}
}

func TestExitQuestType(t *testing.T) {
	tests := []struct {
		typ       Type
		wantState StateID
		inBook    bool
		events    []string
	}{
		{Repeatable, Started, false, []string{"RemoveDeathNotify " + warrior, "RemoveQuestItems " + warrior, "QuestList"}},
		{OneTime, Completed, true, []string{"RemoveDeathNotify " + warrior, "RemoveQuestItems " + warrior, "QuestList"}},
		{Daily, Completed, true, []string{"RemoveDeathNotify " + warrior, "RemoveQuestItems " + warrior, "QuestList"}},
	}
	for _, tt := range tests {
		b, r := loaded(t, storage.Quest{Name: warrior, Vars: vars("<state>", "Started", "cond", "2")})
		st := b.Get(warrior)
		st.ExitQuestType(tt.typ, false)
		checkEvents(t, r, tt.events...)
		if st.State() != tt.wantState || (b.Get(warrior) == st) != tt.inBook {
			t.Errorf("type %d: %v, in book %v", tt.typ, st.State(), b.Get(warrior) == st)
		}
		if _, ok := st.Get(VarRestart); ok != (tt.typ == Daily) {
			t.Errorf("type %d: restartTime set %v", tt.typ, ok)
		}
	}
}

func TestDailyRestart(t *testing.T) {
	cet := testNow.Location()
	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{"before the reset: today", time.Date(2026, 11, 20, 5, 10, 20, 345_678_000, cet), time.Date(2026, 11, 20, 6, 30, 20, 345_000_000, cet)},
		{"at 06:00: tomorrow", time.Date(2026, 11, 20, 6, 0, 0, 0, cet), time.Date(2026, 11, 21, 6, 30, 0, 0, cet)},
		{"at 06:29: tomorrow", time.Date(2026, 11, 20, 6, 29, 59, 999_000_000, cet), time.Date(2026, 11, 21, 6, 30, 59, 999_000_000, cet)},
		{"evening: tomorrow", time.Date(2026, 11, 20, 23, 59, 1, 0, cet), time.Date(2026, 11, 21, 6, 30, 1, 0, cet)},
		{"end of the month", time.Date(2026, 11, 30, 12, 0, 0, 0, cet), time.Date(2026, 12, 1, 6, 30, 0, 0, cet)},
		{"just after midnight: today", time.Date(2026, 12, 31, 0, 0, 5, 0, cet), time.Date(2026, 12, 31, 6, 30, 5, 0, cet)},
	}
	for _, tt := range tests {
		now := tt.now
		b, r := newTestBook(&now)
		b.Load([]storage.Quest{{Name: warrior, Vars: vars("<state>", "Started", "cond", "4")}})
		st := b.Get(warrior)
		st.ExitQuestType(Daily, true)
		r.take()
		got, _ := st.Get(VarRestart)
		if want := strconv.FormatInt(tt.want.UnixMilli(), 10); got != want {
			t.Errorf("%s: restartTime %s, want %s (%v)", tt.name, got, want, tt.want)
		}
		want := storage.QuestChanges{Upserts: []storage.Quest{{Name: warrior, Vars: vars("<state>", "Completed", VarRestart, got)}}}
		if ch := b.TakeChanges(); !changesEqual(ch, want) {
			t.Errorf("%s: TakeChanges = %+v, want %+v", tt.name, ch, want)
		}

		for _, at := range []struct {
			now  time.Time
			want bool
		}{
			{tt.want.Add(-time.Millisecond), false},
			{tt.want, true},
			{tt.want.Add(time.Hour), true},
		} {
			now = at.now
			if ok, err := st.IsNowAvailable(); ok != at.want || err != nil {
				t.Errorf("%s: IsNowAvailable at %v = %v, %v; want %v", tt.name, at.now, ok, err, at.want)
			}
		}
	}
}

func TestDailyRestartNotStarted(t *testing.T) {
	// ExitQuestType(Daily) sets the restart time even when exitQuest itself
	// does nothing.
	b, r := loaded(t)
	st := b.State(warrior, true)
	st.ExitQuestType(Daily, false)
	checkEvents(t, r, "RemoveDeathNotify "+warrior)
	ch := b.TakeChanges()
	if len(ch.Upserts) != 1 || ch.Upserts[0].Vars[VarState] != "" || ch.Upserts[0].Vars[VarRestart] == "" {
		t.Errorf("TakeChanges = %+v, want restartTime without <state>", ch)
	}
}

func TestIsNowAvailable(t *testing.T) {
	b, _ := loaded(t)
	st := b.State(warrior, true)
	if ok, err := st.IsNowAvailable(); ok || err != nil {
		t.Errorf("without restartTime: %v, %v; want false, nil", ok, err)
	}
	st.Set(VarRestart, "soon")
	ok, err := st.IsNowAvailable()
	var ve *VarError
	if ok || !errors.As(err, &ve) || ve.Var != VarRestart || ve.Quest != warrior {
		t.Errorf("with a bad restartTime: %v, %v; want a VarError", ok, err)
	}
}

func TestSetState(t *testing.T) {
	b, r := loaded(t, storage.Quest{Name: letters, Vars: vars("<state>", "Completed")})
	st := b.Get(letters)
	st.SetState(Completed)
	checkEvents(t, r)
	if b.Dirty() {
		t.Error("setting the same state changed something")
	}
	st.SetState(Created)
	checkEvents(t, r, "QuestList")
	// The state row stays, with Mobius's name for Created.
	want := storage.QuestChanges{Upserts: []storage.Quest{{Name: letters, Vars: vars("<state>", "Start")}}}
	if ch := b.TakeChanges(); !changesEqual(ch, want) {
		t.Errorf("TakeChanges = %+v, want %+v", ch, want)
	}
}

func TestStateNames(t *testing.T) {
	for _, tt := range []struct {
		s    StateID
		name string
	}{{Created, "Start"}, {Started, "Started"}, {Completed, "Completed"}, {StateID(9), "Start"}} {
		if tt.s.Name() != tt.name || tt.s.String() != tt.name {
			t.Errorf("StateID(%d).Name() = %q, want %q", tt.s, tt.s.Name(), tt.name)
		}
	}
	for name, want := range map[string]StateID{"Start": Created, "Started": Started, "Completed": Completed, "started": Created, "": Created} {
		if got := StateFromName(name); got != want {
			t.Errorf("StateFromName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestCheckVar(t *testing.T) {
	long := string(make([]rune, 21))
	tests := []struct {
		name, value string
		want        bool
	}{
		{"cond", "1", true},
		{"__compltdStateFlags", "-2147483635", true}, // 19 characters
		{long, "1", false},
		{"x", string(make([]rune, 256)), false},
	}
	for _, tt := range tests {
		if got := CheckVar(tt.name, tt.value); got != tt.want {
			t.Errorf("CheckVar(%d chars, %d chars) = %v, want %v", len(tt.name), len(tt.value), got, tt.want)
		}
	}
}

// changesEqual compares quest changes as the stores apply them.
func changesEqual(a, b storage.QuestChanges) bool {
	if len(a.Upserts) != len(b.Upserts) || len(a.Deleted) != len(b.Deleted) {
		return false
	}
	for i := range a.Upserts {
		if a.Upserts[i].Name != b.Upserts[i].Name || !maps.Equal(a.Upserts[i].Vars, b.Upserts[i].Vars) {
			return false
		}
	}
	for i := range a.Deleted {
		if a.Deleted[i] != b.Deleted[i] {
			return false
		}
	}
	return true
}
