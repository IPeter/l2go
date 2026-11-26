package summon

import "testing"

func TestKind(t *testing.T) {
	for _, tt := range []struct {
		typ   string
		kind  Kind
		pet   bool
		stype int32
	}{
		{"Servitor", Servitor, false, 1},
		{"Pet", Pet, true, 2},
		{"BabyPet", BabyPet, true, 2},
	} {
		k, ok := KindOfType(tt.typ)
		if !ok || k != tt.kind || k.IsPet() != tt.pet || k.SummonType() != tt.stype || k.String() != tt.typ {
			t.Errorf("%s: kind %v ok %v pet %v type %d", tt.typ, k, ok, k.IsPet(), k.SummonType())
		}
	}
	if _, ok := KindOfType("Monster"); ok {
		t.Error("a monster is a summon")
	}
	if !IsPassive(12564) || !IsPassive(14736) || IsPassive(12077) {
		t.Error("passive summons")
	}
}

func TestLife(t *testing.T) {
	// Kat the Cat level 3: 1 crystal every 4 intervals... here an upkeep
	// every 10 s and a 20 s life.
	l := NewLife(20, 1458, 1, 10, false)
	steps := []LifeStep{LifeGoesOn, LifeConsume}
	for i, want := range steps {
		if got := l.Tick(); got != want {
			t.Fatalf("tick %d: %v, want %v", i, got, want)
		}
	}
	if !l.Consumed(true) || l.IntervalLeft != 10000 {
		t.Fatalf("after the upkeep: %+v", l)
	}
	if got := l.Tick(); got != LifeGoesOn {
		t.Fatalf("tick 3: %v", got)
	}
	// 20 s are used up at the fourth tick (remaining 0), the fifth goes
	// under 0.
	if got := l.Tick(); got != LifeConsume || l.Remaining != 0 {
		t.Fatalf("tick 4: %v, %+v", got, l)
	}
	if l.Consumed(false) {
		t.Fatal("a failed upkeep keeps the servitor")
	}
	if got := l.Tick(); got != LifeExpired {
		t.Fatalf("tick 5: %v", got)
	}

	def := NewLife(3600, 0, 1, 0, false)
	if def.Interval != 240000 || def.Total != 3600000 {
		t.Fatalf("defaults %+v", def)
	}
	if siege := NewLife(3600, 0, 1, 0, true); siege.Interval != 60000 {
		t.Fatalf("siege weapon interval %d", siege.Interval)
	}
	// No item: never due.
	for range 100 {
		if def.Tick() == LifeConsume {
			t.Fatal("an upkeep without an item")
		}
	}
}

func TestHunger(t *testing.T) {
	for _, tt := range []struct {
		fed, max, limit int
		hungry          bool
	}{
		{137, 248, 55, false}, // 136.4 is the limit
		{136, 248, 55, true},
		{0, 248, 55, true},
		{248, 248, 55, false},
	} {
		if got := Hungry(tt.fed, tt.max, tt.limit); got != tt.hungry {
			t.Errorf("Hungry(%d, %d, %d) = %v", tt.fed, tt.max, tt.limit, got)
		}
	}
	if !Uncontrollable(0) || Uncontrollable(1) {
		t.Error("uncontrollable")
	}
	if AfterFeedTick(5, 2) != 3 || AfterFeedTick(2, 2) != 0 || AfterFeedTick(1, 2) != 0 {
		t.Error("feed tick")
	}
	if FeedConsume(true, 4, 2) != 4 || FeedConsume(false, 4, 2) != 2 {
		t.Error("feed consume")
	}
	if ClampFed(300, 248) != 248 || ClampFed(10, 248) != 10 {
		t.Error("clamp")
	}
	if CanSendBack(99, 248) || !CanSendBack(100, 248) {
		t.Error("send back at 40 %")
	}
}

// curve is a small pet curve: levels 1-5, level n starting at 100 × (n−1)².
func curve() ExpCurve {
	exp := []int64{0, 0, 100, 400, 900, 1600, 2500}
	return ExpCurve{MinLevel: 1, MaxLevel: 5, ExpFor: func(l int) int64 {
		return exp[min(max(l, 0), len(exp)-1)]
	}}
}

func TestExpCurve(t *testing.T) {
	c := curve()
	for _, tt := range []struct {
		name      string
		exp       int64
		level     int
		add       int64
		wantExp   int64
		wantLevel int
	}{
		{"within the level", 0, 1, 50, 50, 1},
		{"one level", 50, 1, 60, 110, 2},
		{"two levels", 0, 1, 500, 500, 3},
		{"up to max level − 1", 0, 1, 1000, 1000, 4},
		{"capped under the top exp", 0, 1, 5000, 1599, 4},
		{"top of the table", 1599, 4, 10, 1599, 4},
		{"loss under 0", 50, 1, -60, 50, 1},
		{"loss to a lower level", 450, 3, -100, 350, 2},
	} {
		exp, level := c.AddExp(tt.exp, tt.level, tt.add)
		if exp != tt.wantExp || level != tt.wantLevel {
			t.Errorf("%s: AddExp(%d, %d, %d) = %d, %d; want %d, %d", tt.name, tt.exp, tt.level, tt.add, exp, level, tt.wantExp, tt.wantLevel)
		}
	}
	// Death at level 2: (−0.14 + 6.5) % of 300 = 19.08 → 19.
	if got := c.DeathPenalty(2); got != 19 {
		t.Errorf("death penalty %d", got)
	}
	if got := RestoredExp(200, 100, 70); got != 70 {
		t.Errorf("restored %d", got)
	}
	if got := RestoredExp(0, 100, 70); got != 0 {
		t.Errorf("restored without a death %d", got)
	}
}

func TestOwnerShare(t *testing.T) {
	oe, os, pe, ps := OwnerShare(1000, 100, 73)
	if oe < 729.99 || oe > 730.01 || pe < 269.99 || pe > 270.01 || os < 72.99 || ps < 26.99 {
		t.Fatalf("share %v %v %v %v", oe, os, pe, ps)
	}
	if oe, _, pe, _ := OwnerShare(1000, 0, 150); oe != 1000 || pe != 0 {
		t.Fatalf("over 100 %%: %v %v", oe, pe)
	}
	if PetExpGain(269.5, 1) != 270 {
		t.Fatal("rounding")
	}
}

func TestSkillLevels(t *testing.T) {
	maxLevel := func(int) int { return 12 }
	for _, tt := range []struct{ pet, want int }{{1, 1}, {9, 1}, {35, 3}, {69, 6}, {70, 7}, {79, 8}, {86, 10}} {
		if got := SkillLevel(tt.pet, 0); got != tt.want {
			t.Errorf("SkillLevel(%d) = %d, want %d", tt.pet, got, tt.want)
		}
	}
	if SkillLevel(99, 8) != 8 {
		t.Error("cap")
	}
	rows := []SkillRow{{SkillID: 4710, SkillLevel: 1, MinLevel: 1}, {SkillID: 4710, SkillLevel: 2, MinLevel: 30}, {SkillID: 5000, SkillLevel: 0}}
	for _, tt := range []struct{ skill, level, want int }{{4710, 10, 1}, {4710, 30, 2}, {5000, 45, 4}, {6000, 50, 0}} {
		if got := PetAvailableLevel(rows, tt.skill, tt.level, maxLevel); got != tt.want {
			t.Errorf("PetAvailableLevel(%d, %d) = %d, want %d", tt.skill, tt.level, got, tt.want)
		}
	}
	// Listed but not reached: 1.
	if got := PetAvailableLevel([]SkillRow{{SkillID: 1, SkillLevel: 3, MinLevel: 50}}, 1, 10, maxLevel); got != 1 {
		t.Errorf("not reached: %d", got)
	}
	if got := SummonAvailableLevel([]SkillRow{{SkillID: 4260, SkillLevel: 0}}, 4260, 53, maxLevel); got != 5 {
		t.Errorf("servitor by level %d", got)
	}
	if got := SummonAvailableLevel([]SkillRow{{SkillID: 4260, SkillLevel: 3}, {SkillID: 4260, SkillLevel: 1}}, 4260, 53, maxLevel); got != 3 {
		t.Errorf("servitor highest %d", got)
	}
	if KnownSkillLevel(75) != 8 || KnownSkillLevel(55) != 5 {
		t.Error("known skill level")
	}
}

func TestActions(t *testing.T) {
	if d := Actions[16]; d.Kind != ActAttack || !d.Pet || !d.ValidFor(Pet) || !d.ValidFor(Servitor) {
		t.Errorf("16: %+v", d)
	}
	if d := Actions[22]; d.Kind != ActAttack || d.ValidFor(Pet) || !d.ValidFor(Servitor) {
		t.Errorf("22: %+v", d)
	}
	if d := Actions[45]; d.SkillName != "HealMagic" || !d.OnOwner {
		t.Errorf("45: %+v", d)
	}
	if _, ok := Actions[0]; ok {
		t.Error("sit is a player action")
	}
}
