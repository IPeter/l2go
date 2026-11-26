package cursedweapon

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

// The two weapons of Mobius CursedWeapons.xml, in file order.
var mobiusDefs = []data.CursedWeapon{
	{ItemID: 8190, Name: "Demonic Sword Zariche", SkillID: 3603, DropRate: 1, Duration: 300, DurationLost: 3, DisappearChance: 50, StageKills: 10},
	{ItemID: 8689, Name: "Blood Sword Akamanah", SkillID: 3629, DropRate: 1, Duration: 300, DurationLost: 3, DisappearChance: 50, StageKills: 10},
}

func newMobiusSet() *Set { return NewSet(mobiusDefs, func(int) int { return 10 }) }

var t0 = time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)

// Mobius's HashMap puts 8689 (bucket 1) before 8190 (bucket 14).
func TestSetOrder(t *testing.T) {
	s := newMobiusSet()
	var ids []int32
	for _, w := range s.Weapons() {
		ids = append(ids, w.ItemID())
	}
	if !slices.Equal(ids, []int32{8689, 8190}) {
		t.Errorf("order %v, want [8689 8190]", ids)
	}
	if w := s.Get(8190); w == nil || w.SkillMaxLevel != 10 || w.SkillID() != 3603 {
		t.Errorf("Get(8190) = %+v", w)
	}
	if s.IsCursed(57) || !s.IsCursed(8689) {
		t.Error("IsCursed is wrong")
	}
}

func TestHashMapOrder(t *testing.T) {
	// 1 and 17 share bucket 1 and keep their insertion order; 16 and 32 are
	// in bucket 0; the high half is mixed in (0x10001 is in bucket 0,
	// 0x10002 in bucket 3); 13 keys make 32 buckets.
	for _, c := range []struct {
		keys []int32
		want []int
	}{
		{nil, []int{}},
		{[]int32{8190, 8689}, []int{1, 0}},
		{[]int32{1, 17, 2}, []int{0, 1, 2}},
		{[]int32{16, 3, 32}, []int{0, 2, 1}},
		{[]int32{0x10001, 1}, []int{0, 1}},
		{[]int32{0x10002, 1}, []int{1, 0}},
		{[]int32{13, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0}},
	} {
		if got := HashMapOrder(c.keys); !slices.Equal(got, c.want) {
			t.Errorf("HashMapOrder(%v) = %v, want %v", c.keys, got, c.want)
		}
	}
}

func TestExcluded(t *testing.T) {
	for _, c := range []struct {
		types []string // the NPC's type and its supertypes
		want  bool
	}{
		{[]string{"Monster", "Attackable"}, false},
		{[]string{"RaidBoss", "Monster", "Attackable"}, false},
		{[]string{"GrandBoss", "RaidBoss", "Monster"}, true},
		{[]string{"Guard", "Attackable"}, true},
		{[]string{"Defender", "Attackable"}, true},
		{[]string{"RiftInvader", "Monster"}, true},
		{[]string{"FestivalMonster", "Monster"}, true},
		{[]string{"FeedableBeast", "Monster"}, true},
	} {
		got := Excluded(func(typ string) bool { return slices.Contains(c.types, typ) })
		if got != c.want {
			t.Errorf("Excluded(%v) = %v, want %v", c.types, got, c.want)
		}
	}
}

// CheckDrop rolls Rnd.get(100000) < 1 for each weapon not in the world, in
// map order, and stops at the first drop.
func TestCheckDrop(t *testing.T) {
	s := newMobiusSet()
	// No weapon drops: both roll.
	if w := s.CheckDrop(rnd.NewScripted(t, rnd.Get(DropBound, 1), rnd.Get(DropBound, 0)), false); w == nil || w.ItemID() != 8190 {
		t.Errorf("second roll 0 dropped %+v, want Zariche", w)
	}
	if w := s.CheckDrop(rnd.NewScripted(t, rnd.Get(DropBound, 0)), false); w == nil || w.ItemID() != 8689 {
		t.Errorf("first roll 0 dropped %+v, want Akamanah", w)
	}
	if w := s.CheckDrop(rnd.NewScripted(t, rnd.Get(DropBound, 1), rnd.Get(DropBound, 99999)), false); w != nil {
		t.Errorf("rolls 1 and 99999 dropped %+v", w)
	}
	// An excluded NPC rolls nothing.
	if w := s.CheckDrop(rnd.NewScripted(t), true); w != nil {
		t.Errorf("an excluded NPC dropped %+v", w)
	}
	// A weapon in the world does not roll.
	s.Get(8689).DroppedByMonster(t0)
	if w := s.CheckDrop(rnd.NewScripted(t, rnd.Get(DropBound, 0)), false); w == nil || w.ItemID() != 8190 {
		t.Errorf("with Akamanah on the ground dropped %+v, want Zariche", w)
	}
	s.Get(8190).Activate(1, 0, 0)
	if w := s.CheckDrop(rnd.NewScripted(t), false); w != nil {
		t.Errorf("with both in the world dropped %+v", w)
	}
}

// A seeded run drops at about 1 in 100,000 per weapon.
func TestDropRate(t *testing.T) {
	s := newMobiusSet()
	r := rnd.New(14)
	const kills = 5_000_000
	drops := 0
	for range kills {
		if s.CheckDrop(r, false) != nil {
			drops++
		}
	}
	// Two weapons roll per kill: 100 drops expected.
	if drops < 70 || drops > 130 {
		t.Errorf("%d drops in %d kills, want about 100", drops, kills)
	}
}

// Rnd.get(100) <= 50 vanishes: 0–50 vanish (51 values), 51–99 drop.
func TestRollDisappear(t *testing.T) {
	w := newMobiusSet().Get(8190)
	for _, c := range []struct {
		roll int
		want bool
	}{{0, true}, {50, true}, {51, false}, {99, false}} {
		if got := w.RollDisappear(rnd.NewScripted(t, rnd.Get(DisappearBound, c.roll))); got != c.want {
			t.Errorf("roll %d: vanishes %v, want %v", c.roll, got, c.want)
		}
	}
	r := rnd.New(3)
	n := 0
	for range 100_000 {
		if w.RollDisappear(r) {
			n++
		}
	}
	if n < 50_000 || n > 52_000 {
		t.Errorf("%d of 100000 vanished, want about 51000", n)
	}
}

// The skill level steps every 10 kills up to level 10; the level UserInfo
// shows is kills / 10, capped once past 100 kills.
func TestLevels(t *testing.T) {
	w := newMobiusSet().Get(8190)
	w.DroppedByMonster(t0)
	w.Activate(7, 100, 3)
	for _, c := range []struct {
		kills           int32
		skill           int
		level           int32
		levelUpReaching bool
	}{
		{0, 1, 0, false},
		{9, 1, 0, false},
		{10, 2, 1, true},
		{19, 2, 1, false},
		{20, 3, 2, true},
		{89, 9, 8, false},
		{90, 10, 9, true},
		{99, 10, 9, false},
		{100, 10, 10, false},
		{101, 10, 10, false},
		{150, 10, 10, false},
	} {
		w.NbKills = c.kills
		if got := w.SkillLevel(); got != c.skill {
			t.Errorf("%d kills: skill level %d, want %d", c.kills, got, c.skill)
		}
		if got := w.Level(); got != c.level {
			t.Errorf("%d kills: level %d, want %d", c.kills, got, c.level)
		}
		if c.kills > 0 {
			w.NbKills = c.kills - 1
			if got := w.AddKill(); got != c.levelUpReaching {
				t.Errorf("kill %d: level up %v, want %v", c.kills, got, c.levelUpReaching)
			}
		}
	}
}

// Each kill takes 3 minutes off the 300-minute life; the life is checked
// every 36 s.
func TestLife(t *testing.T) {
	w := newMobiusSet().Get(8689)
	if got := w.CheckPeriod(); got != 36*time.Second {
		t.Errorf("check period %v, want 36s", got)
	}
	w.DroppedByMonster(t0)
	if !w.Dropped || w.Activated || !w.Active() {
		t.Errorf("after the drop: %+v", w)
	}
	if got, want := w.EndTime, t0.Add(300*time.Minute).UnixMilli(); got != want {
		t.Errorf("end time %d, want %d", got, want)
	}
	if got := w.MinutesLeft(t0); got != 300 {
		t.Errorf("minutes left %d, want 300", got)
	}
	w.Activate(5, 0, 0)
	w.AddKill()
	w.AddKill()
	if got := w.MinutesLeft(t0); got != 294 {
		t.Errorf("after two kills: %d minutes left, want 294", got)
	}
	if got := w.MinutesLeft(t0.Add(30 * time.Second)); got != 293 {
		t.Errorf("30 s later: %d minutes left, want 293 (cut down)", got)
	}
	end := t0.Add(294 * time.Minute)
	if w.Expired(end.Add(-time.Millisecond)) || !w.Expired(end) {
		t.Error("Expired is wrong at the end time")
	}
	w.Reset()
	if w.Active() || w.EndTime != 0 || w.PlayerID != 0 || w.NbKills != 0 {
		t.Errorf("after Reset: %+v", w)
	}
}

// The holder of one weapon who picks up the other gets a bonus stage.
func TestBonusStage(t *testing.T) {
	w := newMobiusSet().Get(8190)
	w.StartLife(t0)
	w.Activate(5, 0, 0)
	w.NbKills = 3
	if !w.AddBonusStage() {
		t.Error("the bonus stage gives no level")
	}
	if w.NbKills != 10 || w.SkillLevel() != 2 {
		t.Errorf("after the bonus: %d kills, skill level %d; want 10, 2", w.NbKills, w.SkillLevel())
	}
	if got := w.MinutesLeft(t0); got != 297 {
		t.Errorf("after the bonus: %d minutes left, want 297", got)
	}
}

func TestHeldBy(t *testing.T) {
	s := newMobiusSet()
	if s.HeldBy(0) != nil || s.HeldBy(5) != nil {
		t.Error("an empty set has a holder")
	}
	s.Get(8190).Activate(5, 10, 2)
	if w := s.HeldBy(5); w == nil || w.ItemID() != 8190 || w.PlayerKarma != 10 || w.PlayerPKKills != 2 {
		t.Errorf("HeldBy(5) = %+v", w)
	}
	s.Get(8190).DroppedByHolder()
	if s.HeldBy(5) != nil {
		t.Error("a dropped weapon still has its holder")
	}
}

func TestFind(t *testing.T) {
	s := newMobiusSet()
	for _, c := range []struct {
		param string
		want  int32 // 0 for none
	}{
		{"8190", 8190},
		{"8689", 8689},
		{"57", 0},
		{"zariche", 8190},
		{"Blood_Sword", 8689},
		{"SWORD", 8689},
		{"excalibur", 0},
	} {
		w, ok := s.Find(c.param)
		switch {
		case c.want == 0 && ok:
			t.Errorf("Find(%q) = %d, want none", c.param, w.ItemID())
		case c.want != 0 && (!ok || w.ItemID() != c.want):
			t.Errorf("Find(%q) = %+v, %v; want %d", c.param, w, ok, c.want)
		}
	}
}
