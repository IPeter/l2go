package quest

import (
	"maps"
	"math"
	"slices"
	"strconv"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// TestCondFlags pins Mobius QuestState.set("cond") and its private
// setCond(cond, old) case by case: the __compltdStateFlags value, the step,
// and the packets. The expected flags were worked out by hand from the Java
// code, with Java's int rules (wrap-around, shift distances mod 32).
func TestCondFlags(t *testing.T) {
	const absent = "absent"
	flags := func(bits uint32) string { return strconv.Itoa(int(u(bits))) }
	mark := []string{"QuestList", "QuestMark 401"}
	listOnly := []string{"QuestList"}
	tests := []struct {
		name      string
		cond      string // the stored step before; "" for none
		flags     string // the stored flags before; "" for none
		set       string // the new step, as set
		wantFlags string // the flags after, or absent
		wantCond  int32
		events    []string
	}{
		// No step skipped: the flags stay absent.
		{"first step from none", "", "", "1", absent, 1, mark},
		{"next step 1 to 2", "1", "", "2", absent, 2, mark},
		{"next step 2 to 3", "2", "", "3", absent, 3, mark},
		{"next step 3 to 4", "3", "", "4", absent, 4, mark},
		{"back 4 to 2 without skips", "4", "", "2", absent, 2, mark},

		// The first skip starts the flags: top bit, step 1, every step up
		// to old, and the new step.
		{"skip 1 to 3", "1", "", "3", flags(0x80000005), 3, mark},
		{"skip 0 to 2 (cond below 3 still gets flags)", "", "", "2", flags(0x80000003), 2, mark},
		{"skip 0 to 3", "", "", "3", flags(0x80000005), 3, mark},
		{"skip 3 to 5", "3", "", "5", flags(0x80000017), 5, mark},
		{"skip 0 to 31", "", "", "31", flags(0xC0000001), 31, mark},

		// Forward after a skip: the new step's bit is added.
		{"forward 3 to 4 after a skip", "3", flags(0x80000005), "4", flags(0x8000000D), 4, mark},
		{"skip again 3 to 6", "3", flags(0x80000005), "6", flags(0x80000025), 6, mark},
		{"positive flags move forward too", "3", "5", "4", "13", 4, mark},

		// Back to an earlier step: the bits after it go; if no earlier
		// step was skipped, the flags go too.
		{"back 4 to 3 keeps the earlier skip", "4", flags(0x8000000D), "3", flags(0x80000005), 3, mark},
		{"back 5 to 3 before the skip", "5", flags(0x80000017), "3", absent, 3, mark},
		{"back 6 to 4, a skipped step", "6", flags(0x80000025), "4", flags(0x80000005), 4, mark},
		{"back 4 to 2 clears the flags", "4", flags(0x8000000D), "2", absent, 2, mark},
		{"forward 1 to 2 clears the flags", "1", flags(0x80000005), "2", absent, 2, mark},
		{"back to 0: no mark", "3", flags(0x80000005), "0", absent, 0, listOnly},
		{"negative step: no mark", "3", flags(0x80000005), "-1", absent, -1, listOnly},

		// Steps above 31 have no flags of their own; the shift wraps.
		{"31 to 32", "31", "", "32", absent, 32, mark},
		{"skip 1 to 32 (1 << 31)", "1", "", "32", flags(0x80000001), 32, mark},
		{"skip 1 to 33 (1 << 32 is 1)", "1", "", "33", flags(0x80000001), 33, mark},
		{"skip 31 to 33 ((1 << 31) - 1 wraps)", "31", "", "33", flags(0xFFFFFFFF), 33, mark},
		{"skip 3 to 40 drops the old flags", "3", flags(0x80000005), "40", flags(0x80000087), 40, mark},
		{"from -5 to 3 (1 << -5 is 1 << 27)", "-5", "", "3", flags(0x87FFFFFF), 3, mark},
		{"from MaxInt32 to 5 (old + 1 wraps)", strconv.Itoa(math.MaxInt32), "", "5", flags(0xFFFFFFFF), 5, mark},

		// Values that are not ints count as 0.
		{"old step not a number", "x", "", "3", flags(0x80000005), 3, mark},
		{"new step not a number", "2", "", "abc", absent, 0, listOnly},
		{"flags not a number", "3", "zz", "5", flags(0x80000017), 5, mark},
		{"step with a plus sign", "1", "", "+3", flags(0x80000005), 3, mark},

		// The same step: nothing but the variable.
		{"same step", "3", flags(0x80000005), "3", flags(0x80000005), 3, nil},
		{"same step, other spelling", "3", "", "03", absent, 3, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := storage.Quest{Name: warrior, Vars: vars("<state>", "Started")}
			if tt.cond != "" {
				doc.Vars[VarCond] = tt.cond
			}
			if tt.flags != "" {
				doc.Vars[VarFlags] = tt.flags
			}
			b, r := loaded(t, doc)
			st := b.Get(warrior)
			st.Set(VarCond, tt.set)

			got, ok := st.Get(VarFlags)
			if !ok {
				got = absent
			}
			if got != tt.wantFlags {
				t.Errorf("flags = %s, want %s", got, tt.wantFlags)
			}
			if st.Cond() != tt.wantCond || !st.IsCond(tt.wantCond) {
				t.Errorf("Cond() = %d, want %d", st.Cond(), tt.wantCond)
			}
			checkEvents(t, r, tt.events...)

			// The stored document follows: the step as set, the flags as
			// computed.
			ch := b.TakeChanges()
			if len(ch.Upserts) != 1 || len(ch.Deleted) != 0 {
				t.Fatalf("TakeChanges = %+v, want the quest's document", ch)
			}
			want := vars("<state>", "Started", VarCond, tt.set)
			if tt.wantFlags != absent {
				want[VarFlags] = tt.wantFlags
			}
			if !maps.Equal(ch.Upserts[0].Vars, want) {
				t.Errorf("document %v, want %v", ch.Upserts[0].Vars, want)
			}

			// QuestList sends positive flags, else the step.
			status := tt.wantCond
			if n, err := strconv.Atoi(tt.wantFlags); err == nil && n > 0 {
				status = int32(n)
			}
			if a := b.Active(); len(a) != 1 || a[0] != (Entry{ID: 401, Status: status}) {
				t.Errorf("Active() = %+v, want 401 with status %d", a, status)
			}
		})
	}
}

// TestCondFlagsWalk runs a whole quest with skips and resets, each step
// against the flags Mobius stores.
func TestCondFlagsWalk(t *testing.T) {
	b, r := loaded(t)
	st := b.State(warrior, true)
	st.StartQuest()
	r.take()
	steps := []struct {
		cond  int32
		flags string // "" for none
	}{
		{2, ""},
		{4, strconv.Itoa(int(u(0x8000000B)))}, // 1, 2, 4
		{5, strconv.Itoa(int(u(0x8000001B)))}, // 1, 2, 4, 5
		{7, strconv.Itoa(int(u(0x8000005B)))}, // 1, 2, 4, 5, 7
		{3, strconv.Itoa(int(u(0x80000003)))}, // back to 3: 1, 2 and the top bit
		{4, strconv.Itoa(int(u(0x8000000B)))}, // 1, 2, 4
		{1, ""},                               // reset
		{3, strconv.Itoa(int(u(0x80000005)))}, // 1, 3
		{2, ""},                               // below 3: gone
	}
	for _, s := range steps {
		st.SetCond(s.cond, false)
		got, _ := st.Get(VarFlags)
		if got != s.flags || st.Cond() != s.cond {
			t.Errorf("after SetCond(%d): cond %d, flags %q; want flags %q", s.cond, st.Cond(), got, s.flags)
		}
	}
}

// TestCondPackets checks the packets of a step change by quest kind.
func TestCondPackets(t *testing.T) {
	tests := []struct {
		quest  string
		events []string
	}{
		{warrior, []string{"QuestList", "QuestMark 401"}},
		{custom, []string{"QuestList"}}, // custom quests get no mark
		{script, []string{"QuestList", "QuestMark -1"}},
		{gone, []string{"QuestList"}}, // Mobius: getQuest() is null, the mark throws
	}
	for _, tt := range tests {
		now := testNow
		b, r := newTestBook(&now)
		st := b.New(tt.quest)
		st.Set(VarCond, "2")
		if got := r.take(); !slices.Equal(got, tt.events) {
			t.Errorf("%s: events %q, want %q", tt.quest, got, tt.events)
		}
	}
}

func TestBit(t *testing.T) {
	tests := []struct {
		n    int32
		want int32
	}{
		{0, 1}, {1, 2}, {30, 1 << 30}, {31, math.MinInt32}, {32, 1}, {33, 2}, {-1, math.MinInt32}, {-5, 1 << 27},
	}
	for _, tt := range tests {
		if got := bit(tt.n); got != tt.want {
			t.Errorf("bit(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		s    string
		want int32
		ok   bool
	}{
		{"0", 0, true}, {"42", 42, true}, {"-7", -7, true}, {"+7", 7, true}, {"007", 7, true},
		{"2147483647", math.MaxInt32, true}, {"-2147483648", math.MinInt32, true},
		{"2147483648", 0, false}, {"", 0, false}, {" 1", 0, false}, {"1 ", 0, false},
		{"1_000", 0, false}, {"0x10", 0, false}, {"1.5", 0, false}, {"abc", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseInt(tt.s)
		if got != tt.want || ok != tt.ok {
			t.Errorf("parseInt(%q) = %d, %v; want %d, %v", tt.s, got, ok, tt.want, tt.ok)
		}
	}
}
