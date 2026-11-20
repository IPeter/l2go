package effects_test

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/effects"
	"github.com/IPeter/l2go/internal/skills/skilltest"
	"github.com/IPeter/l2go/internal/stats"
)

// repoData is the data folder of the repository.
const repoData = "../../../data"

// waveA are the handlers of Wave A.
var waveA = []data.EffectName{
	data.EffectBuff, data.EffectPhysicalDamage, data.EffectMagicalDamage, data.EffectDebuff,
	data.EffectDefenceTrait, data.EffectHeal, data.EffectHealOverTime, data.EffectHealPercent,
	data.EffectAttackTrait, data.EffectEscape, data.EffectCpHeal, data.EffectManaHealOverTime,
	data.EffectManaHealByLevel, data.EffectManaHeal, data.EffectManaHealPercent, data.EffectLucky,
}

func TestPortedHasWaveA(t *testing.T) {
	got := effects.Ported()
	for _, n := range waveA {
		if !slices.Contains(got, n) {
			t.Errorf("%s is not ported", n)
		}
	}
}

// setup returns an Env over a scripted source and compiles one skill.
func setup(t *testing.T, values map[string]string, defs map[data.EffectScope][]data.EffectDef, draws ...rnd.Draw) (*skills.Env, *skilltest.Sched, *skills.Skill) {
	t.Helper()
	env, sched := skilltest.NewEnv(t, rnd.NewScripted(t, draws...), &skilltest.World{})
	cat := skilltest.Catalog(t, effects.NewRegistry(), skilltest.Template(1, values, defs))
	sk := skilltest.Skill(t, cat, 1)
	if r := sk.Refusal(); r != "" {
		t.Fatalf("skill refused: %s", r)
	}
	return env, sched, sk
}

func effect(name string, params map[string]string) map[data.EffectScope][]data.EffectDef {
	return skilltest.General(data.EffectDef{Name: name, Params: params})
}

func TestHeal(t *testing.T) {
	tests := []struct {
		name     string
		values   map[string]string
		setup    func(caster *skilltest.Creature)
		self     bool
		want     float64
		messages []sp.SystemMessageID
	}{
		// 30 + sqrt(2 × 100).
		{"plain", map[string]string{"isMagic": "1"}, nil, true, 44, []sp.SystemMessageID{skills.MsgS1HPRestored}},
		{"by another", map[string]string{"isMagic": "1"}, nil, false, 44, []sp.SystemMessageID{skills.MsgS2HPRestoredByS1}},
		// A mystic with a blessed spiritshot: 30 + 10 × 2.4 + sqrt(4 × 100).
		{"mystic with blessed spiritshot", map[string]string{"isMagic": "1", "mpConsume": "10"}, func(c *skilltest.Creature) {
			c.Mage = true
			c.Shots[skills.BlessedSpiritshot] = true
		}, true, 74, []sp.SystemMessageID{skills.MsgS1HPRestored}},
		// A fighter's blessed spiritshot only multiplies M.Atk by 4.
		{"fighter with blessed spiritshot", map[string]string{"isMagic": "1", "mpConsume": "10"}, func(c *skilltest.Creature) {
			c.Shots[skills.BlessedSpiritshot] = true
		}, true, 50, []sp.SystemMessageID{skills.MsgS1HPRestored}},
		{"static", map[string]string{"isMagic": "2"}, nil, true, 30, []sp.SystemMessageID{skills.MsgS1HPRestored}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, _, sk := setup(t, tt.values, effect("Heal", map[string]string{"power": "30"}))
			caster := skilltest.NewPlayer(env, 1)
			target := caster
			if !tt.self {
				target = skilltest.NewPlayer(env, 2)
			}
			if tt.setup != nil {
				tt.setup(caster)
			}
			target.HP = 500
			sk.ApplyTo(env, caster, target)
			if got := int(target.HP - 500); got != int(tt.want) {
				t.Errorf("healed %v, want %v", target.HP-500, tt.want)
			}
			if got := target.MessageIDs(); !slices.Equal(got, tt.messages) {
				t.Errorf("messages = %v, want %v", got, tt.messages)
			}
		})
	}
	t.Run("overheal and rejuvenation", func(t *testing.T) {
		env, _, _ := setup(t, nil, nil)
		cat := skilltest.Catalog(t, effects.NewRegistry(), skilltest.Template(4051, map[string]string{"isMagic": "2"}, effect("Heal", map[string]string{"power": "300"})))
		p := skilltest.NewPlayer(env, 1)
		p.HP = 900
		skilltest.Skill(t, cat, 4051).ApplyTo(env, p, p)
		if p.HP != 1000 || !slices.Equal(p.MessageIDs(), []sp.SystemMessageID{skills.MsgRejuvenatingHP}) {
			t.Errorf("HP %v, messages %v", p.HP, p.MessageIDs())
		}
	})
}

func TestHealPercentAndCp(t *testing.T) {
	env, _, half := setup(t, nil, effect("HealPercent", map[string]string{"power": "50"}))
	p := skilltest.NewPlayer(env, 1)
	p.HP = 200
	half.ApplyTo(env, p, p)
	if p.HP != 700 || p.Messages[0].Params[0] != sp.NumberParam(500) {
		t.Fatalf("HP %v, message %+v", p.HP, p.Messages)
	}
	p.HP = 900
	half.ApplyTo(env, p, p)
	if p.HP != 1000 {
		t.Fatalf("HP %v, want 1000", p.HP)
	}

	env, _, cp := setup(t, nil, effect("CpHeal", map[string]string{"power": "100"}))
	a, b := skilltest.NewPlayer(env, 1), skilltest.NewPlayer(env, 2)
	b.CP = 0
	cp.ApplyTo(env, a, b)
	if b.CP != 100 || !slices.Equal(b.MessageIDs(), []sp.SystemMessageID{skills.MsgS1RestoresS2CP}) {
		t.Fatalf("CP %v, messages %v", b.CP, b.MessageIDs())
	}
}

func TestManaHeals(t *testing.T) {
	t.Run("ManaHeal", func(t *testing.T) {
		env, _, sk := setup(t, nil, effect("ManaHeal", map[string]string{"power": "100"}))
		p := skilltest.NewPlayer(env, 1)
		p.MP = 0
		_ = p.AddModifiers(stats.NewModifier(stats.ManaCharge, stats.OpMul, 1.5, stats.Owner{Kind: stats.OwnerItem, ID: 1}))
		sk.ApplyTo(env, p, p)
		if p.MP != 150 {
			t.Fatalf("MP %v, want 150", p.MP)
		}
	})
	t.Run("ManaHealPercent", func(t *testing.T) {
		env, _, sk := setup(t, nil, effect("ManaHealPercent", map[string]string{"power": "10"}))
		p := skilltest.NewPlayer(env, 1)
		p.MP = 0
		sk.ApplyTo(env, p, p)
		if p.MP != 100 || !slices.Equal(p.MessageIDs(), []sp.SystemMessageID{skills.MsgS1MPRestored}) {
			t.Fatalf("MP %v, messages %v", p.MP, p.MessageIDs())
		}
	})
	t.Run("ManaHealByLevel", func(t *testing.T) {
		for _, tt := range []struct {
			level int
			want  float64
		}{{20, 100}, {25, 100}, {26, 90}, {27, 80}, {34, 10}, {35, 0}} {
			env, _, sk := setup(t, map[string]string{"magicLevel": "20"}, effect("ManaHealByLevel", map[string]string{"power": "100"}))
			a, p := skilltest.NewPlayer(env, 1), skilltest.NewPlayer(env, 2)
			p.Lvl = tt.level
			p.MP = 0
			sk.ApplyTo(env, a, p)
			if p.MP < tt.want-1e-9 || p.MP > tt.want+1e-9 {
				t.Errorf("level %d: MP %v, want %v", tt.level, p.MP, tt.want)
			}
			if !slices.Equal(p.MessageIDs(), []sp.SystemMessageID{skills.MsgS2MPRestoredByS1}) {
				t.Errorf("messages %v", p.MessageIDs())
			}
		}
	})
}

func TestHealOverTimeTicks(t *testing.T) {
	env, sched, sk := setup(t,
		map[string]string{"operateType": "A2", "abnormalType": "HP_RECOVER", "abnormalLevel": "1", "abnormalTime": "15"},
		skilltest.General(data.EffectDef{Name: "HealOverTime", Ticks: "5", Params: map[string]string{"power": "10"}}))
	p := skilltest.NewPlayer(env, 1)
	p.HP = 100
	sk.ApplyTo(env, p, p)
	if len(p.RegenMax) != 1 || p.RegenMax[0] != [3]float64{15, 5, 10} {
		t.Fatalf("ExRegenMax = %v", p.RegenMax)
	}
	// 5 ticks of 666 ms; each heals 10 × 3.33.
	sched.Advance(3329 * time.Millisecond)
	if p.HP != 100 {
		t.Fatalf("healed before the first tick")
	}
	sched.Advance(time.Millisecond)
	perTick := float64(float32(5*666) / 1000)
	if want := 100 + float64(10*perTick); p.HP != want {
		t.Fatalf("HP after one tick %v, want %v", p.HP, want)
	}
	p.HP = 1000
	sched.Advance(3330 * time.Millisecond)
	if p.HP != 1000 {
		t.Fatalf("HP over the maximum: %v", p.HP)
	}
}

func TestManaHealOverTimeKeepsGoingAtFull(t *testing.T) {
	env, sched, sk := setup(t,
		map[string]string{"operateType": "A2", "abnormalTime": "20"},
		skilltest.General(data.EffectDef{Name: "ManaHealOverTime", Ticks: "1", Params: map[string]string{"power": "10"}}))
	p := skilltest.NewPlayer(env, 1)
	p.MP = 995
	sk.ApplyTo(env, p, p)
	sched.Advance(666 * time.Millisecond)
	if p.MP != 1000 {
		t.Fatalf("MP %v, want 1000", p.MP)
	}
	sched.Advance(666 * time.Millisecond)
	if p.EL.BySkillID(1) == nil {
		t.Fatalf("the effect ended at full MP")
	}
}

func TestTraits(t *testing.T) {
	env, _, sk := setup(t, map[string]string{"operateType": "P"},
		skilltest.General(
			data.EffectDef{Name: "DefenceTrait", Params: map[string]string{"SWORD": "-15", "POISON": "100", "BLEED": "0", "NOT_A_TRAIT": "5"}},
			data.EffectDef{Name: "AttackTrait", Params: map[string]string{"BUG_WEAKNESS": "20"}}))
	m := skilltest.NewMonster(env, 1)
	sk.AddPassive(env, m)
	tr := m.Traits()
	if got, want := tr.Defence(data.TraitSword), (float32(-15)+100)/100; got != want || !tr.HasDefence(data.TraitSword) {
		t.Errorf("SWORD defence %v, want %v", got, want)
	}
	if !tr.Invul(data.TraitPoison) || tr.HasDefence(data.TraitBleed) {
		t.Errorf("POISON invul %v, BLEED %v", tr.Invul(data.TraitPoison), tr.HasDefence(data.TraitBleed))
	}
	if got, want := tr.Attack(data.TraitBugWeakness), (float32(20)+100)/100; got != want {
		t.Errorf("BUG_WEAKNESS attack %v, want %v", got, want)
	}
	m.EL.StopSkill(skills.FinishNormal, 1)
	if tr.Defence(data.TraitSword) != 1 || tr.Invul(data.TraitPoison) || tr.HasAttack(data.TraitBugWeakness) {
		t.Errorf("traits left after the passive ended")
	}
}

func TestBadParameters(t *testing.T) {
	cat := skilltest.Catalog(t, effects.NewRegistry(),
		skilltest.Template(1, nil, effect("AttackTrait", map[string]string{"NOT_A_TRAIT": "5"})),
		skilltest.Template(2, nil, effect("Escape", map[string]string{"escapeType": "NOWHERE"})),
		skilltest.Template(3, nil, effect("Heal", map[string]string{"power": "lots"})))
	for id := 1; id <= 3; id++ {
		if skilltest.Skill(t, cat, id).Refusal() == "" {
			t.Errorf("skill %d with bad parameters is not refused", id)
		}
	}
	if n := len(cat.Report().Problems); n != 3 {
		t.Errorf("%d problems, want 3", n)
	}
}

func TestEscapeLuckyBuffDebuff(t *testing.T) {
	env, _, escape := setup(t, nil, effect("Escape", map[string]string{"escapeType": "TOWN"}))
	p := skilltest.NewPlayer(env, 1)
	escape.ApplyTo(env, p, p)
	if !slices.Equal(p.Escapes, []skills.TeleportWhere{skills.TeleportTown}) {
		t.Fatalf("escapes = %v", p.Escapes)
	}

	env, _, lucky := setup(t, map[string]string{"operateType": "P"}, effect("Lucky", nil))
	p = skilltest.NewPlayer(env, 1)
	m := skilltest.NewMonster(env, 2)
	lucky.AddPassive(env, p)
	lucky.AddPassive(env, m)
	if len(p.EL.BySkillID(1).Effects()) != 1 || len(m.EL.BySkillID(1).Effects()) != 0 {
		t.Fatalf("Lucky must start on players only")
	}

	env, _, debuff := setup(t, map[string]string{"operateType": "A2", "isDebuff": "true", "abnormalTime": "5"},
		skilltest.General(data.EffectDef{Name: "Debuff", Funcs: []data.StatFuncDef{{Stat: data.StatPowerDefence, Op: data.OpMul, Order: 20, Value: "0.5"}}}))
	m = skilltest.NewMonster(env, 2)
	debuff.ApplyTo(env, skilltest.NewPlayer(env, 1), m)
	if got := m.PDef(nil); got != 50 || !debuff.HasEffectType(skills.TypeDebuff) {
		t.Fatalf("P.Def %v under the debuff", got)
	}
}

func TestPhysicalDamage(t *testing.T) {
	values := map[string]string{"operateType": "A1", "power": "25", "targetType": "ONE", "effectPoint": "-100"}
	t.Run("hit", func(t *testing.T) {
		// Draws: the skill evasion, then the counter-attack.
		env, _, sk := setup(t, values, effect("PhysicalDamage", nil), rnd.Get(100, 50), rnd.Get(100, 50))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		a.Loc = skills.Location{X: -100}
		a.Shots[skills.Soulshot] = true
		sk.ApplyTo(env, a, m)
		// (100 × 2 + 25) × 76 × 1.2 / 100 from behind.
		if !slices.Equal(m.Damage, []float64{205}) || !slices.Equal(a.DamageMsgs, []int{205}) {
			t.Fatalf("damage %v, messages %v", m.Damage, a.DamageMsgs)
		}
		if !sk.UseSoulShot() {
			t.Fatalf("a physical attack skill uses soulshots")
		}
	})
	t.Run("dodged", func(t *testing.T) {
		env, _, sk := setup(t, values, effect("PhysicalDamage", nil), rnd.Get(100, 5))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		_ = m.AddModifiers(stats.NewModifier(stats.PSkillEvas, stats.OpAdd, 10, stats.Owner{Kind: stats.OwnerItem, ID: 1}))
		sk.ApplyTo(env, a, m)
		if len(m.Damage) != 0 || !slices.Equal(a.MessageIDs(), []sp.SystemMessageID{skills.MsgS1DodgesTheAttack}) {
			t.Fatalf("damage %v, messages %v", m.Damage, a.MessageIDs())
		}
	})
	t.Run("critical and counter-attack", func(t *testing.T) {
		v := map[string]string{"operateType": "A1", "power": "25", "baseCritRate": "10", "castRange": "40"}
		env, _, sk := setup(t, v, effect("PhysicalDamage", nil), rnd.Get(100, 50), rnd.Get(1000, 0), rnd.Get(100, 0))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		m.Head = 32768 // facing the attacker
		a.Loc = skills.Location{X: -100}
		_ = m.AddModifiers(stats.NewModifier(stats.VengeancePdam, stats.OpAdd, 10, stats.Owner{Kind: stats.OwnerItem, ID: 1}))
		sk.ApplyTo(env, a, m)
		// 2 × 125 × 76 / 100; then 100 × 10 × 70 / 100, twice for a critical hit.
		if !slices.Equal(m.Damage, []float64{190}) || !slices.Equal(a.Damage, []float64{700, 700}) {
			t.Fatalf("target damage %v, attacker damage %v", m.Damage, a.Damage)
		}
	})
}

func TestMagicalDamage(t *testing.T) {
	values := map[string]string{"operateType": "A1", "isMagic": "1", "power": "12", "magicLevel": "1", "targetType": "ONE", "effectPoint": "-100"}
	t.Run("hit and break", func(t *testing.T) {
		// Draws: the magic critical roll, the magic success, the attack
		// break, Shield Deflect Magic.
		env, _, sk := setup(t, values, effect("MagicalDamage", nil),
			rnd.Get(1000, 500), rnd.Get(100, 0), rnd.Get(100, 0), rnd.Get(100, 0))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		a.Shots[skills.Spiritshot] = true
		m.Break.Casting = true
		sk.ApplyTo(env, a, m)
		if !slices.Equal(m.Damage, []float64{154}) || m.Breaks != 1 {
			t.Fatalf("damage %v, breaks %d", m.Damage, m.Breaks)
		}
	})
	t.Run("deflected", func(t *testing.T) {
		env, _, sk := setup(t, values, effect("MagicalDamage", nil),
			rnd.Get(1000, 500), rnd.Get(100, 0), rnd.Get(100, 10))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		_ = m.AddModifiers(stats.NewModifier(stats.VengeanceMdam, stats.OpAdd, 20, stats.Owner{Kind: stats.OwnerItem, ID: 1}))
		sk.ApplyTo(env, a, m)
		if len(m.Damage) != 0 || !slices.Equal(a.Damage, []float64{109}) || len(a.DamageMsgs) != 0 {
			t.Fatalf("target %v, attacker %v", m.Damage, a.Damage)
		}
	})
}

func TestActivateSpendsShotsAndAppliesSelf(t *testing.T) {
	values := map[string]string{"operateType": "A1", "isMagic": "1", "power": "12", "targetType": "ONE", "effectPoint": "-100"}
	defs := map[data.EffectScope][]data.EffectDef{
		data.ScopeGeneral: {{Name: "MagicalDamage"}},
		data.ScopeSelf:    {{Name: "HealPercent", Params: map[string]string{"power": "10"}}},
	}
	env, _, sk := setup(t, values, defs, rnd.Get(1000, 500), rnd.Get(100, 0), rnd.Get(100, 50))
	a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
	a.HP = 500
	a.Shots[skills.BlessedSpiritshot] = true
	sk.Activate(env, a, []skills.Creature{m})
	if a.Shots[skills.BlessedSpiritshot] {
		t.Errorf("the blessed spiritshot was not spent")
	}
	if a.HP != 600 {
		t.Errorf("self effect: HP %v, want 600", a.HP)
	}
	if !slices.Equal(m.Damage, []float64{218}) {
		t.Errorf("damage %v", m.Damage)
	}
}

func TestMobiusData(t *testing.T) {
	d, err := data.LoadSkills(repoData)
	if err != nil {
		t.Fatalf("LoadSkills: %v", err)
	}
	cat, err := skills.NewCatalog(d, effects.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	r := cat.Report()
	for _, n := range waveA {
		if r.EffectSkills[n] == 0 || !r.Ported[n] {
			t.Errorf("%s: used by %d skills, ported %v", n, r.EffectSkills[n], r.Ported[n])
		}
	}
	for _, p := range r.Problems {
		t.Errorf("problem: %s", p)
	}
	t.Logf("%d skills, %d levels, %d stub names, %d skills with stubs, %d refused levels",
		r.Skills, r.Levels, len(r.Stubs()), r.StubSkills, r.RefusedLevels)

	env, _ := skilltest.NewEnv(t, rnd.NewScripted(t), nil)
	p := skilltest.NewPlayer(env, 1)
	for _, lvl := range []struct{ level, mp, initial int }{{1, 7, 2}, {3, 8, 2}, {5, 12, 3}} {
		ws, ok := cat.Skill(1177, lvl.level)
		if !ok || ws.Refusal() != "" {
			t.Fatalf("Wind Strike %d: %v %q", lvl.level, ok, ws.Refusal())
		}
		if skills.MPConsume(env, p, ws) != lvl.mp || skills.MPInitialConsume(p, ws) != lvl.initial {
			t.Errorf("Wind Strike %d: MP %d + %d", lvl.level, skills.MPInitialConsume(p, ws), skills.MPConsume(env, p, ws))
		}
		if got := skills.CastTime(p, ws); got != 3999 {
			t.Errorf("Wind Strike %d: cast time %d at 333", lvl.level, got)
		}
		if got := skills.ReuseDelay(p, ws); got != 6000 {
			t.Errorf("Wind Strike %d: reuse %d at 333", lvl.level, got)
		}
	}
	for lvl, rate := range []float64{1.08, 1.12, 1.15} {
		might, ok := cat.Skill(1068, lvl+1)
		if !ok || might.Refusal() != "" || might.AbnormalTime != 1200 {
			t.Fatalf("Might %d: %v %q", lvl+1, ok, might.Refusal())
		}
		target := skilltest.NewPlayer(env, 2)
		might.ApplyTo(env, p, target)
		base := 100.0
		if got := target.PAtk(nil); got != base*rate {
			t.Errorf("Might %d: P.Atk %v, want %v", lvl+1, got, base*rate)
		}
	}
	for lvl := 1; lvl <= cat.MaxLevel(4416); lvl++ {
		if sk, ok := cat.Skill(4416, lvl); !ok || sk.Refusal() != "" {
			t.Errorf("4416 level %d: %v", lvl, ok)
		}
	}
}
