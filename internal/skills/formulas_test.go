package skills_test

import (
	"math"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/skilltest"
	"github.com/IPeter/l2go/internal/stats"
)

// oneSkill compiles one skill of the values and returns it.
func oneSkill(t *testing.T, f *fixture, values map[string]string, effects map[data.EffectScope][]data.EffectDef) *skills.Skill {
	t.Helper()
	cat := f.catalog(t, skilltest.Template(1, values, effects))
	return skilltest.Skill(t, cat, 1)
}

func mod(st stats.Stat, op stats.Op, v float64) stats.Modifier {
	return stats.NewModifier(st, op, v, stats.Owner{Kind: stats.OwnerItem, ID: 1})
}

func TestCastTime(t *testing.T) {
	tests := []struct {
		name          string
		values        map[string]string
		pSpd, mSpd    float64
		sps, bss      bool
		want          int
		wantReuseTime int
	}{
		// (4000.0 / 333) × 333 is 3999.9999999999995 in doubles, in Java
		// too: Wind Strike casts in 3999 ms at the base casting speed.
		{"magic at 333", map[string]string{"isMagic": "1", "hitTime": "4000", "reuseDelay": "6000"}, 300, 333, false, false, 3999, 6000},
		{"magic at 213", map[string]string{"isMagic": "1", "hitTime": "4000", "reuseDelay": "6000"}, 300, 213, false, false, 6253, 9380},
		{"magic with a spiritshot", map[string]string{"isMagic": "1", "hitTime": "4000"}, 300, 213, true, false, 4377, 0},
		{"magic with a blessed spiritshot", map[string]string{"isMagic": "1", "hitTime": "4000"}, 300, 213, false, true, 4377, 0},
		{"magic over 550 floors at 550", map[string]string{"isMagic": "1", "hitTime": "600"}, 300, 1665, false, false, 550, 0},
		{"magic of 500 floors at 500", map[string]string{"isMagic": "1", "hitTime": "500"}, 300, 666, false, false, 500, 0},
		{"physical", map[string]string{"hitTime": "1080", "reuseDelay": "13000"}, 320, 333, false, false, 1012, 13528},
		{"physical floors at 500", map[string]string{"hitTime": "400", "coolTime": "100"}, 1280, 333, false, false, 500, 0},
		{"physical under 500 is not floored", map[string]string{"hitTime": "400"}, 1280, 333, false, false, 93, 0},
		{"dance", map[string]string{"isMagic": "3", "hitTime": "2000", "reuseDelay": "6000"}, 320, 333, false, false, 1875, 6243},
		{"static", map[string]string{"isMagic": "2", "hitTime": "300", "reuseDelay": "6000"}, 1280, 1665, false, false, 300, 6000},
		{"static reuse", map[string]string{"isMagic": "1", "hitTime": "1000", "reuseDelay": "6000", "staticReuse": "true"}, 300, 213, false, false, 1563, 6000},
		{"channeling with a channeling skill", map[string]string{"operateType": "CA1", "hitTime": "1000", "channelingSkillId": "5"}, 1280, 333, false, false, 1000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			sk := oneSkill(t, f, tt.values, nil)
			c := skilltest.NewPlayer(f.env, 1)
			c.BasePAtkSpd, c.BaseMAtkSpd = tt.pSpd, tt.mSpd
			c.Shots[skills.Spiritshot], c.Shots[skills.BlessedSpiritshot] = tt.sps, tt.bss
			if got := skills.CastTime(c, sk); got != tt.want {
				t.Errorf("CastTime = %d, want %d", got, tt.want)
			}
			if got := skills.ReuseDelay(c, sk); got != tt.wantReuseTime {
				t.Errorf("ReuseDelay = %d, want %d", got, tt.wantReuseTime)
			}
		})
	}
}

func TestReuseRates(t *testing.T) {
	f := newFixture(t)
	magic := oneSkill(t, f, map[string]string{"isMagic": "1", "reuseDelay": "6000"}, nil)
	c := skilltest.NewPlayer(f.env, 1)
	c.BaseMAtkSpd = 213
	if err := c.AddModifiers(mod(stats.MReuse, stats.OpMul, 0.5)); err != nil {
		t.Fatal(err)
	}
	// 3000 × (333f / 213) in Java floats.
	if got := skills.ReuseDelay(c, magic); got != 4690 {
		t.Fatalf("ReuseDelay = %d, want 4690", got)
	}
}

func TestMPConsume(t *testing.T) {
	f := newFixture(t)
	dance := func(id int) data.SkillTemplate {
		return buffSkill(id, "NONE", 0, "1", map[string]string{"isMagic": "3", "mpConsume": "30"})
	}
	cat := f.catalog(t, dance(10), dance(11), dance(12),
		skilltest.Template(20, map[string]string{"isMagic": "1", "mpConsume": "7", "mpInitialConsume": "2"}, nil))
	c := skilltest.NewPlayer(f.env, 1)
	d := skilltest.Skill(t, cat, 12)
	if got := skills.MPConsume(f.env, c, d); got != 30 {
		t.Fatalf("first dance MP = %d, want 30", got)
	}
	skilltest.Skill(t, cat, 10).ApplyTo(f.env, c, c)
	skilltest.Skill(t, cat, 11).ApplyTo(f.env, c, c)
	if got := skills.MPConsume(f.env, c, d); got != 60 {
		t.Fatalf("third dance MP = %d, want 30 + 2 × 15", got)
	}
	f.env.Config.DanceConsumeAdditionalMP = false
	if got := skills.MPConsume(f.env, c, d); got != 30 {
		t.Fatalf("without DanceConsumeAdditionalMP MP = %d, want 30", got)
	}
	ws := skilltest.Skill(t, cat, 20)
	if err := c.AddModifiers(mod(stats.MPConsume, stats.OpMul, 0.5)); err != nil {
		t.Fatal(err)
	}
	if got := skills.MPConsume(f.env, c, ws); got != 3 {
		t.Fatalf("MP at the hit = %d, want (int) 3.5", got)
	}
	if got := skills.MPInitialConsume(c, ws); got != 1 {
		t.Fatalf("MP at the start = %d, want 1", got)
	}
}

func TestSkillMasteryAndAbnormalTime(t *testing.T) {
	t.Run("STR mastery", func(t *testing.T) {
		f := newFixture(t, rnd.Get(100, 1), rnd.Get(100, 2))
		sk := oneSkill(t, f, map[string]string{"operateType": "A2", "abnormalTime": "1200"}, nil)
		p := skilltest.NewPlayer(f.env, 1)
		p.Bonus = map[data.BaseStat]float64{data.BaseSTR: 1.5}
		if err := p.AddModifiers(mod(stats.SkillCritical, stats.OpAdd, 1)); err != nil {
			t.Fatal(err)
		}
		if got := skills.AbnormalTime(f.env, p, p, sk); got != 2400 {
			t.Errorf("abnormal time with mastery = %d, want 2400", got)
		}
		if got := skills.AbnormalTime(f.env, p, p, sk); got != 1200 {
			t.Errorf("abnormal time without mastery = %d, want 1200", got)
		}
	})
	t.Run("no draw", func(t *testing.T) {
		f := newFixture(t)
		cat := f.catalog(t,
			skilltest.Template(1, map[string]string{"operateType": "A2", "abnormalTime": "1200"}, nil),
			skilltest.Template(2, map[string]string{"operateType": "P"}, nil),
			skilltest.Template(3, map[string]string{"operateType": "T", "abnormalTime": "5"}, nil),
			skilltest.Template(4, map[string]string{"operateType": "A2", "abnormalTime": "10", "abnormalInstant": "true"}, nil),
			skilltest.Template(5, map[string]string{"operateType": "A2", "abnormalTime": "10", "isMagic": "2"}, nil))
		npc := skilltest.NewMonster(f.env, 1)
		if err := npc.AddModifiers(mod(stats.SkillCritical, stats.OpAdd, 1)); err != nil {
			t.Fatal(err)
		}
		servitor := skilltest.New(f.env, 2, data.InstanceServitor)
		for _, tt := range []struct {
			id     int
			target skills.Creature
			want   int
		}{{1, npc, 1200}, {2, npc, -1}, {3, npc, -1}, {4, servitor, 5}, {4, npc, 10}} {
			if got := skills.AbnormalTime(f.env, npc, tt.target, skilltest.Skill(t, cat, tt.id)); got != tt.want {
				t.Errorf("skill %d: abnormal time %d, want %d", tt.id, got, tt.want)
			}
		}
		p := skilltest.NewPlayer(f.env, 3)
		if err := p.AddModifiers(mod(stats.SkillCritical, stats.OpAdd, 1)); err != nil {
			t.Fatal(err)
		}
		if skills.SkillMastery(f.env, p, skilltest.Skill(t, cat, 5)) {
			t.Errorf("static skills never roll mastery")
		}
	})
}

// debuffValues are a debuff's values for the success tests.
func debuffValues(magic, rate, maxChance string) map[string]string {
	return map[string]string{
		"operateType": "A2", "isDebuff": "true", "isMagic": magic, "abnormalTime": "10",
		"activateRate": rate, "magicLevel": "20", "lvlBonusRate": "1", "basicProperty": "MEN",
		"minChance": "10", "maxChance": maxChance, "trait": "SLEEP",
	}
}

func TestEffectSuccess(t *testing.T) {
	tests := []struct {
		name    string
		values  map[string]string
		setup   func(a, d *skilltest.Creature)
		draws   []rnd.Draw
		want    bool
		resists bool
	}{
		// ((20 − 20 + 3) × 1 + 80 + 30 − 20) × 1.01 = 93.93, kept at 90.
		{"physical within max", debuffValues("0", "80", "90"), nil, []rnd.Draw{rnd.Get(100, 89)}, true, false},
		{"physical at max", debuffValues("0", "80", "90"), nil, []rnd.Draw{rnd.Get(100, 90)}, false, true},
		// (3 + 30 + 30 − 20) × sqrt(100) / 100 × 11 × 1.01 = 47.773.
		{"magic lands", debuffValues("1", "30", "99"), nil, []rnd.Draw{rnd.Get(100, 47)}, true, false},
		{"magic resisted", debuffValues("1", "30", "99"), nil, []rnd.Draw{rnd.Get(100, 48)}, false, true},
		// A blessed spiritshot: sqrt(400) → × 2.2, 95.546.
		{"blessed spiritshot", debuffValues("1", "30", "99"), func(a, _ *skilltest.Creature) { a.Shots[skills.BlessedSpiritshot] = true }, []rnd.Draw{rnd.Get(100, 95)}, true, false},
		{"debuff immunity", debuffValues("0", "80", "90"), func(_, d *skilltest.Creature) {
			_ = d.AddModifiers(mod(stats.DebuffImmunity, stats.OpAdd, 1))
		}, nil, false, true},
		{"trait invulnerability", debuffValues("0", "80", "90"), func(_, d *skilltest.Creature) {
			d.TraitsV.AddInvul(data.TraitSleep)
		}, []rnd.Draw{rnd.Get(100, 0)}, false, true},
		{"no activate rate", map[string]string{"operateType": "A2", "isDebuff": "true"}, nil, nil, true, false},
		{"no basic property", map[string]string{"operateType": "A2", "activateRate": "50"}, nil, nil, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, tt.draws...)
			sk := oneSkill(t, f, tt.values, nil)
			a, d := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
			d.Lvl = 20
			d.Base = map[data.BaseStat]int{data.BaseMEN: 20}
			if tt.setup != nil {
				tt.setup(a, d)
			}
			if got := skills.EffectSuccess(f.env, a, d, sk); got != tt.want {
				t.Errorf("EffectSuccess = %v, want %v", got, tt.want)
			}
			resisted := slices.Contains(a.MessageIDs(), skills.MsgS1HasResistedYourS2)
			if resisted != tt.resists {
				t.Errorf("resist message %v, want %v", resisted, tt.resists)
			}
		})
	}
	t.Run("doors", func(t *testing.T) {
		f := newFixture(t)
		sk := oneSkill(t, f, map[string]string{"operateType": "A2"}, nil)
		if skills.EffectSuccess(f.env, skilltest.NewPlayer(f.env, 1), skilltest.New(f.env, 2, data.InstanceDoor), sk) {
			t.Errorf("a door took a continuous effect")
		}
	})
}

func TestMagicSuccess(t *testing.T) {
	tests := []struct {
		name          string
		power, mLevel string
		aLvl, tLvl    int
		draws         []rnd.Draw
		want          bool
	}{
		// 100 − round(1.3^4) = 97.
		{"lands", "12", "1", 1, 5, []rnd.Draw{rnd.Get(100, 96)}, true},
		{"fails", "12", "1", 1, 5, []rnd.Draw{rnd.Get(100, 97)}, false},
		// Without a magic level the attacker's level counts: 1.3^-5 rounds to 0.
		{"attacker level", "12", "0", 10, 5, []rnd.Draw{rnd.Get(100, 99)}, true},
		{"power -1 never fails", "-1", "1", 1, 80, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, tt.draws...)
			sk := oneSkill(t, f, map[string]string{"isMagic": "1", "power": tt.power, "magicLevel": tt.mLevel}, nil)
			a, d := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
			a.Lvl, d.Lvl = tt.aLvl, tt.tLvl
			if got := skills.MagicSuccess(f.env, a, d, sk); got != tt.want {
				t.Errorf("MagicSuccess = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMagicDamage(t *testing.T) {
	values := map[string]string{"isMagic": "1", "power": "12", "magicLevel": "1"}
	tests := []struct {
		name       string
		tLvl       int
		shld       stats.ShieldResult
		sps, bss   bool
		mcrit      bool
		draws      []rnd.Draw
		want       int
		attackerID []sp.SystemMessageID
	}{
		{"plain", 1, stats.ShieldFailed, false, false, false, []rnd.Draw{rnd.Get(100, 0)}, 109, nil},
		{"spiritshot", 1, stats.ShieldFailed, true, false, false, []rnd.Draw{rnd.Get(100, 0)}, 154, nil},
		{"blessed spiritshot", 1, stats.ShieldFailed, false, true, false, []rnd.Draw{rnd.Get(100, 0)}, 218, nil},
		{"critical on a monster", 1, stats.ShieldFailed, false, false, true, []rnd.Draw{rnd.Get(100, 0)}, 327, nil},
		{"perfect block", 1, stats.ShieldPerfect, false, false, false, nil, 1, nil},
		{"failed, halved", 5, stats.ShieldFailed, false, false, false, []rnd.Draw{rnd.Get(100, 98), rnd.Get(100, 0)}, 54,
			[]sp.SystemMessageID{skills.MsgYourAttackHasFailed}},
		{"failed, resisted", 15, stats.ShieldFailed, false, false, false, []rnd.Draw{rnd.Get(100, 70), rnd.Get(100, 0)}, 1,
			[]sp.SystemMessageID{skills.MsgS1HasResistedYourS2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, tt.draws...)
			sk := oneSkill(t, f, values, nil)
			a, d := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
			d.Lvl = tt.tLvl
			got := skills.MagicDamage(f.env, a, d, sk, tt.shld, tt.sps, tt.bss, tt.mcrit)
			if int(got) != tt.want {
				t.Errorf("MagicDamage = %v, want %d", got, tt.want)
			}
			if ids := a.MessageIDs(); !slices.Equal(ids, tt.attackerID) {
				t.Errorf("attacker messages = %v, want %v", ids, tt.attackerID)
			}
		})
	}
}

func TestPhysSkillDamage(t *testing.T) {
	tests := []struct {
		name    string
		heading int // the target's; the attacker stands at its -x side
		shld    stats.ShieldResult
		crit    bool
		ss      bool
		want    int
	}{
		{"behind", 0, stats.ShieldFailed, false, false, 114},
		{"in front", 32768, stats.ShieldFailed, false, false, 95},
		{"side", 16384, stats.ShieldFailed, false, false, 104},
		{"soulshot", 0, stats.ShieldFailed, false, true, 205},
		{"critical", 0, stats.ShieldFailed, true, false, 228},
		{"shield", 32768, stats.ShieldSucceed, false, false, 47},
		{"perfect block", 0, stats.ShieldPerfect, false, false, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			sk := oneSkill(t, f, map[string]string{"power": "25"}, nil)
			a, d := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
			a.Loc = skills.Location{X: -100}
			d.Head = tt.heading
			d.ShieldDefV = 100
			got := skills.PhysSkillDamage(f.env, a, d, sk, tt.shld, tt.crit, tt.ss)
			if int(got) != tt.want {
				t.Errorf("PhysSkillDamage = %v, want %d", got, tt.want)
			}
		})
	}
}

func TestAtkBreak(t *testing.T) {
	f := newFixture(t, rnd.Get(100, 51), rnd.Get(100, 52))
	c := skilltest.NewPlayer(f.env, 1)
	if skills.AtkBreak(f.env, c, 100) {
		t.Fatalf("a creature that does not cast broke")
	}
	c.Break.Casting = true
	// 15 + sqrt(1300) = 51.06.
	if !skills.AtkBreak(f.env, c, 100) {
		t.Errorf("Get(100) = 51 did not break")
	}
	if skills.AtkBreak(f.env, c, 100) {
		t.Errorf("Get(100) = 52 broke")
	}
	c.Raid = true
	if skills.AtkBreak(f.env, c, 100) {
		t.Errorf("a raid broke")
	}
}

func TestTraitBonuses(t *testing.T) {
	f := newFixture(t)
	a, d := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
	if got := skills.GeneralTraitBonus(a, d, data.TraitBugWeakness, false); got != 1 {
		t.Errorf("weakness without traits = %v, want 1", got)
	}
	a.TraitsV.AddAttack(data.TraitBugWeakness, 1.5)
	d.TraitsV.AddDefence(data.TraitBugWeakness, 1.1)
	want := float64(float32(1.5)-float32(1.1)) + 1
	if got := skills.GeneralTraitBonus(a, d, data.TraitBugWeakness, true); got != want {
		t.Errorf("weakness = %v, want %v", got, want)
	}
	d.TraitsV.AddDefence(data.TraitSleep, 0.8)
	if got, want := skills.GeneralTraitBonus(a, d, data.TraitSleep, false), float64(float32(1)-float32(0.8))+1; got != want {
		t.Errorf("effect trait = %v, want %v", got, want)
	}
	if got := skills.GeneralTraitBonus(a, d, data.TraitSleep, true); got != 1 {
		t.Errorf("ignored resistance = %v, want 1", got)
	}
	d.TraitsV.AddInvul(data.TraitSleep)
	if got := skills.GeneralTraitBonus(a, d, data.TraitSleep, false); got != 0 {
		t.Errorf("invulnerable = %v, want 0", got)
	}
	a.Attack = data.WeaponSword
	d.TraitsV.AddDefence(data.TraitSword, 0.9)
	weapon := 1 - (float64(float32(0.9)) - 1)
	if got, want := skills.AttackTraitBonus(a, d), weapon*want; math.Abs(got-want) > 1e-12 {
		t.Errorf("attack trait bonus = %v, want %v", got, want)
	}
}

func TestAttributeBonus(t *testing.T) {
	f := newFixture(t)
	wind := oneSkill(t, f, map[string]string{"element": "2", "elementPower": "20"}, nil)
	a, d := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
	up := 1.025 + math.Sqrt(8000.0/2)*0.0001
	if got := skills.AttributeBonus(a, d, wind); math.Abs(got-up) > 1e-12 {
		t.Errorf("wind 20 = %v, want %v", got, up)
	}
	if got := skills.AttributeBonus(a, d, nil); got != 1 {
		t.Errorf("no element = %v, want 1", got)
	}
	_ = a.AddModifiers(mod(stats.FirePower, stats.OpAdd, 10))
	_ = d.AddModifiers(mod(stats.FireRes, stats.OpAdd, 30))
	down := 0.975 - math.Sqrt(8000.0/2)*0.0001
	if got := skills.AttributeBonus(a, d, nil); math.Abs(got-down) > 1e-12 {
		t.Errorf("fire 10 against 30 = %v, want %v", got, down)
	}
}

func TestShieldUse(t *testing.T) {
	tests := []struct {
		name     string
		attacker skills.Location
		draws    []rnd.Draw
		want     stats.ShieldResult
		msg      sp.SystemMessageID
	}{
		{"perfect", skills.Location{X: 100}, []rnd.Draw{rnd.Get(100, 91)}, stats.ShieldPerfect, skills.MsgExcellentShieldDefense},
		{"block", skills.Location{X: 100}, []rnd.Draw{rnd.Get(100, 50), rnd.Get(100, 19)}, stats.ShieldSucceed, skills.MsgShieldDefenseSucceeded},
		{"no block", skills.Location{X: 100}, []rnd.Draw{rnd.Get(100, 50), rnd.Get(100, 20)}, stats.ShieldFailed, 0},
		{"from behind", skills.Location{X: -100}, nil, stats.ShieldFailed, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, tt.draws...)
			sk := oneSkill(t, f, nil, nil)
			a, d := skilltest.NewMonster(f.env, 1), skilltest.NewPlayer(f.env, 2)
			a.Loc = tt.attacker
			d.Shield = true
			_ = d.AddModifiers(mod(stats.RShld, stats.OpAdd, 20))
			if got := skills.ShieldUse(f.env, a, d, sk, true); got != tt.want {
				t.Errorf("ShieldUse = %v, want %v", got, tt.want)
			}
			var want []sp.SystemMessageID
			if tt.msg != 0 {
				want = []sp.SystemMessageID{tt.msg}
			}
			if got := d.MessageIDs(); !slices.Equal(got, want) {
				t.Errorf("messages = %v, want %v", got, want)
			}
		})
	}
	t.Run("ignores shields", func(t *testing.T) {
		f := newFixture(t)
		sk := oneSkill(t, f, map[string]string{"ignoreShld": "true"}, nil)
		a, d := skilltest.NewMonster(f.env, 1), skilltest.NewPlayer(f.env, 2)
		d.Shield = true
		_ = d.AddModifiers(mod(stats.RShld, stats.OpAdd, 20))
		if got := skills.ShieldUse(f.env, a, d, sk, true); got != stats.ShieldFailed {
			t.Errorf("ShieldUse = %v", got)
		}
	})
}

func TestSkillRolls(t *testing.T) {
	// The critical rate is 5 × 10 × 1.2 = 60, the evasion 10, the
	// reflection 30.
	f := newFixture(t,
		rnd.Get(1000, 59), rnd.Get(1000, 60),
		rnd.Get(100, 9),
		rnd.Get(100, 29), rnd.Get(100, 30),
	)
	cat := f.catalog(t,
		skilltest.Template(1, map[string]string{"baseCritRate": "5"}, nil),
		skilltest.Template(2, map[string]string{"isMagic": "1"}, nil),
		skilltest.Template(3, map[string]string{"isMagic": "1", "isDebuff": "true", "activateRate": "50"}, nil))
	phys, magic, debuff := skilltest.Skill(t, cat, 1), skilltest.Skill(t, cat, 2), skilltest.Skill(t, cat, 3)
	a, d := skilltest.NewPlayer(f.env, 1), skilltest.NewPlayer(f.env, 2)
	a.Bonus = map[data.BaseStat]float64{data.BaseSTR: 1.2}
	if !skills.SkillCrit(f.env, a, d, phys) || skills.SkillCrit(f.env, a, d, phys) {
		t.Errorf("skill critical rolls wrong")
	}
	_ = d.AddModifiers(mod(stats.PSkillEvas, stats.OpAdd, 10), mod(stats.ReflectSkillMagic, stats.OpAdd, 30))
	if skills.PhysicalSkillEvasion(f.env, a, d, magic) {
		t.Errorf("magic was dodged")
	}
	if !skills.PhysicalSkillEvasion(f.env, a, d, phys) {
		t.Errorf("Get(100) = 9 was not dodged")
	}
	if !slices.Equal(a.MessageIDs(), []sp.SystemMessageID{skills.MsgS1DodgesTheAttack}) || !slices.Equal(d.MessageIDs(), []sp.SystemMessageID{skills.MsgYouDodgeS1sAttack}) {
		t.Errorf("dodge messages %v and %v", a.MessageIDs(), d.MessageIDs())
	}
	if skills.BuffDebuffReflection(f.env, d, magic) {
		t.Errorf("a non-debuff was reflected")
	}
	if !skills.BuffDebuffReflection(f.env, d, debuff) || skills.BuffDebuffReflection(f.env, d, debuff) {
		t.Errorf("reflection rolls wrong")
	}
}
