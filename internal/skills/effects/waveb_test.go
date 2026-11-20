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

// waveB are the handlers of Wave B.
var waveB = []data.EffectName{
	data.EffectDamOverTime, data.EffectStun, data.EffectRestorationRandom, data.EffectHpDrain,
	data.EffectRestoration, data.EffectDispelBySlot, data.EffectRoot, data.EffectFatalBlow,
	data.EffectParalyze, data.EffectMute, data.EffectEnergyDamage, data.EffectConsumeBody,
	data.EffectMpConsumePerLevel, data.EffectTargetCancel, data.EffectFear, data.EffectSleep,
	data.EffectDispelBySlotProbability, data.EffectManaDamOverTime, data.EffectDispelByCategory,
	data.EffectPhysicalMute, data.EffectLethal, data.EffectPetrification, data.EffectMagicalDamageMp,
	data.EffectImmobileBuff, data.EffectMaxHp, data.EffectResurrection, data.EffectFocusEnergy,
	data.EffectHpByLevel, data.EffectInvincible, data.EffectSilentMove, data.EffectDeathLink,
	data.EffectDispelAll,
}

// The skill values the Wave B tests use.
var (
	bDebuff = map[string]string{"operateType": "A2", "abnormalTime": "10", "isDebuff": "true", "effectPoint": "-100", "targetType": "ONE"}
	bToggle = map[string]string{"operateType": "T"}
	bMagic  = map[string]string{"operateType": "A1", "isMagic": "1", "power": "12", "magicLevel": "1", "targetType": "ONE", "effectPoint": "-100"}
)

// bTicking returns the GENERAL effects of one ticking effect.
func bTicking(name, ticks string, params map[string]string) map[data.EffectScope][]data.EffectDef {
	return skilltest.General(data.EffectDef{Name: name, Ticks: ticks, Params: params})
}

// bTick is the period of a one-tick effect.
const bTick = 666 * time.Millisecond

// bPerTick is the ticks multiplier of a one-tick effect, a Java float.
var bPerTick = float64(float32(666) / 1000)

func bHasMessage(c *skilltest.Creature, id sp.SystemMessageID) bool {
	return slices.Contains(c.MessageIDs(), id)
}

func TestWaveBPorted(t *testing.T) {
	got := effects.Ported()
	for _, n := range waveB {
		if !slices.Contains(got, n) {
			t.Errorf("%s is not ported", n)
		}
	}
}

func TestWaveBMobiusData(t *testing.T) {
	d, err := data.LoadSkills(repoData)
	if err != nil {
		t.Fatalf("LoadSkills: %v", err)
	}
	cat, err := skills.NewCatalog(d, effects.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	r := cat.Report()
	for _, n := range waveB {
		if r.EffectSkills[n] == 0 || !r.Ported[n] {
			t.Errorf("%s: used by %d skills, ported %v", n, r.EffectSkills[n], r.Ported[n])
		}
	}
	// Hero skills cannot be cancelled; Might can.
	for _, tt := range []struct {
		id   int
		want bool
	}{{395, false}, {1374, false}, {1068, true}} {
		sk, ok := cat.Skill(tt.id, 1)
		if !ok || sk.CanBeStolen() != tt.want {
			t.Errorf("skill %d: CanBeStolen %v, want %v", tt.id, ok && sk.CanBeStolen(), tt.want)
		}
	}
}

func TestWaveBBadParameters(t *testing.T) {
	cat := skilltest.Catalog(t, effects.NewRegistry(),
		skilltest.Template(1, nil, effect("DispelBySlot", map[string]string{"dispel": "POISON"})),
		skilltest.Template(2, nil, effect("MaxHp", map[string]string{"type": "HALF", "power": "10"})),
		skilltest.Template(3, nil, effect("DamOverTime", map[string]string{"canKill": "maybe"})))
	for id := 1; id <= 3; id++ {
		if skilltest.Skill(t, cat, id).Refusal() == "" {
			t.Errorf("skill %d with bad parameters is not refused", id)
		}
	}
}

func TestDamOverTime(t *testing.T) {
	t.Run("ticks and stops at 1 HP", func(t *testing.T) {
		env, sched, sk := setup(t, bDebuff, bTicking("DamOverTime", "1", map[string]string{"power": "10"}))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		sk.ApplyTo(env, a, m)
		sched.Advance(bTick)
		if want := 10 * bPerTick; !slices.Equal(m.DOTDamage, []float64{want}) {
			t.Fatalf("damage %v, want %v", m.DOTDamage, want)
		}
		m.HP = 5
		sched.Advance(bTick)
		sched.Advance(bTick)
		if !slices.Equal(m.DOTDamage[1:], []float64{4}) || m.HP != 1 || m.Dead {
			t.Fatalf("damage %v, HP %v, dead %v", m.DOTDamage, m.HP, m.Dead)
		}
	})
	t.Run("canKill", func(t *testing.T) {
		env, sched, sk := setup(t, bDebuff, bTicking("DamOverTime", "1", map[string]string{"power": "10", "canKill": "true"}))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		m.HP = 5
		sk.ApplyTo(env, a, m)
		sched.Advance(bTick)
		if !m.Dead {
			t.Fatalf("HP %v: not killed", m.HP)
		}
	})
	t.Run("toggle without HP", func(t *testing.T) {
		env, sched, sk := setup(t, bToggle, bTicking("DamOverTime", "1", map[string]string{"power": "10"}))
		p := skilltest.NewPlayer(env, 1)
		p.HP = 5
		sk.ApplyTo(env, p, p)
		sched.Advance(bTick)
		if len(p.DOTDamage) != 0 || !bHasMessage(p, 610) || p.EL.BySkillID(1) != nil {
			t.Fatalf("damage %v, messages %v, still on: %v", p.DOTDamage, p.MessageIDs(), p.EL.BySkillID(1) != nil)
		}
	})
}

func TestStun(t *testing.T) {
	values := map[string]string{"operateType": "A2", "abnormalTime": "5", "isDebuff": "true"}
	env, sched, sk := setup(t, values, effect("Stun", nil))
	a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
	sk.ApplyTo(env, a, m)
	if !slices.Equal(m.Aborts, []string{"attack", "cast", "move"}) ||
		!slices.Equal(m.AIEvents(), []skills.AIEvent{skills.AIStunned, skills.AIIdle}) ||
		!m.EL.IsAffected(skills.FlagStunned) || m.VisualUpdates == 0 {
		t.Fatalf("aborts %v, AI %v, stunned %v", m.Aborts, m.AIEvents(), m.EL.IsAffected(skills.FlagStunned))
	}
	sched.Advance(6 * time.Second)
	if m.EL.IsAffected(skills.FlagStunned) || m.AIEvents()[2] != skills.AIThink {
		t.Fatalf("after the stun: AI %v", m.AIEvents())
	}

	raid := skilltest.NewMonster(env, 3)
	raid.Raid = true
	sk.ApplyTo(env, a, raid)
	if raid.EL.IsAffected(skills.FlagStunned) || len(raid.AI) != 0 {
		t.Fatalf("a raid was stunned")
	}

	summon := skilltest.New(env, 4, data.InstanceSummon)
	sk.ApplyTo(env, a, summon)
	if got := summon.AIEvents(); !slices.Equal(got, []skills.AIEvent{skills.AIStunned}) {
		t.Fatalf("summon AI %v: a summon does not go idle", got)
	}
	a.Dead = true
	summon.EL.StopSkill(skills.FinishRemoved, 1)
	if got := summon.AIEvents(); got[len(got)-1] != skills.AIFollowOwner {
		t.Fatalf("summon AI %v after the stun", got)
	}
}

func TestRestorationRandom(t *testing.T) {
	items := &data.RestorationGroups{All: []data.RestorationGroup{
		{Chance: 30, Items: []data.RestorationItem{{ID: 57, Count: 10}, {ID: 57, Count: 5}}},
		{Chance: 50, Items: []data.RestorationItem{{ID: 100, Count: 1, MinEnchant: 1, MaxEnchant: 3}, {ID: 101, Count: 1}}},
	}}
	defs := skilltest.General(data.EffectDef{Name: "RestorationRandom", Items: items})
	t.Run("second group", func(t *testing.T) {
		env, _, sk := setup(t, nil, defs, rnd.Float64(0.45), rnd.Between(1, 3, 2))
		p := skilltest.NewPlayer(env, 1)
		sk.ApplyTo(env, p, p)
		want := []skills.CreatedItem{{ObjectID: 1000, ItemID: 100, Enchant: 2}, {ObjectID: 1001, ItemID: 101}}
		if !slices.Equal(p.Items, want) || !slices.Equal(p.ItemMessages, []bool{false, false}) {
			t.Fatalf("items %v, messages %v", p.Items, p.ItemMessages)
		}
		wantMsgs := []sp.SystemMessage{
			{ID: 369, Params: []sp.SystemMessageParam{sp.NumberParam(2), sp.ItemNameParam(100)}},
			{ID: 30, Params: []sp.SystemMessageParam{sp.ItemNameParam(101)}},
		}
		if !slices.EqualFunc(p.Messages, wantMsgs, func(a, b sp.SystemMessage) bool {
			return a.ID == b.ID && slices.Equal(a.Params, b.Params)
		}) {
			t.Fatalf("messages %+v", p.Messages)
		}
		if len(p.ItemUpdates) != 1 || !slices.Equal(p.ItemUpdates[0], []int{100, 101}) {
			t.Fatalf("updates %v", p.ItemUpdates)
		}
	})
	t.Run("stackable", func(t *testing.T) {
		env, _, sk := setup(t, nil, defs, rnd.Float64(0.1))
		p := skilltest.NewPlayer(env, 1)
		p.ItemStackable = map[int]bool{57: true}
		sk.ApplyTo(env, p, p)
		want := sp.SystemMessage{ID: 29, Params: []sp.SystemMessageParam{sp.ItemNameParam(57), sp.NumberParam(15)}}
		if len(p.Messages) != 1 || p.Messages[0].ID != want.ID || !slices.Equal(p.Messages[0].Params, want.Params) {
			t.Fatalf("messages %+v", p.Messages)
		}
	})
	t.Run("nothing", func(t *testing.T) {
		env, _, sk := setup(t, nil, defs, rnd.Float64(0.9))
		p := skilltest.NewPlayer(env, 1)
		sk.ApplyTo(env, p, p)
		if len(p.Items) != 0 || !slices.Equal(p.MessageIDs(), []sp.SystemMessageID{1669}) {
			t.Fatalf("items %v, messages %v", p.Items, p.MessageIDs())
		}
	})
}

func TestHpDrain(t *testing.T) {
	// Draws: the magic critical roll, the magic success.
	env, _, sk := setup(t, bMagic, effect("HpDrain", map[string]string{"power": "0.2"}),
		rnd.Get(1000, 500), rnd.Get(100, 0), rnd.Get(1000, 500), rnd.Get(100, 0))
	a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
	a.HP, m.CP = 500, 0
	sk.ApplyTo(env, a, m)
	// 91 × sqrt(100) / 100 × 12 = 109; 0.2 of it heals.
	drained := 109.0
	if want := 500 + 0.2*drained; !slices.Equal(m.Damage, []float64{109}) || a.HP != want || !sk.IsDamage() {
		t.Fatalf("damage %v, caster HP %v, want %v", m.Damage, a.HP, want)
	}
	// CP takes the damage first: nothing is drained.
	m.CP = 1000
	sk.ApplyTo(env, a, m)
	if want := 500 + 0.2*drained; a.HP != want {
		t.Fatalf("caster HP %v, want %v", a.HP, want)
	}
}

func TestRestoration(t *testing.T) {
	env, _, sk := setup(t, nil, effect("Restoration", map[string]string{"itemId": "57", "itemCount": "100", "itemEnchantmentLevel": "3"}))
	p := skilltest.NewPlayer(env, 1)
	sk.ApplyTo(env, p, p)
	if !slices.Equal(p.Items, []skills.CreatedItem{{ObjectID: 1000, ItemID: 57, Enchant: 3}}) || !slices.Equal(p.ItemMessages, []bool{true}) {
		t.Fatalf("items %v, messages %v", p.Items, p.ItemMessages)
	}
	m := skilltest.NewMonster(env, 2)
	sk.ApplyTo(env, p, m)
	if len(m.Items) != 0 {
		t.Fatalf("a monster got items")
	}

	env, _, bad := setup(t, nil, effect("Restoration", map[string]string{"itemId": "0", "itemCount": "1"}))
	p = skilltest.NewPlayer(env, 1)
	bad.ApplyTo(env, p, p)
	if len(p.Items) != 0 || !slices.Equal(p.MessageIDs(), []sp.SystemMessageID{1669}) {
		t.Fatalf("items %v, messages %v", p.Items, p.MessageIDs())
	}
}

// bDispelCatalog compiles debuffs (1 POISON level 3, 2 BLEEDING level 1), a
// buff (3), and the skill under test (9).
func bDispelCatalog(t *testing.T, tested data.SkillTemplate) *skills.Catalog {
	t.Helper()
	debuff := func(id int, abnormal, level string) data.SkillTemplate {
		return skilltest.Template(id, map[string]string{
			"operateType": "A2", "abnormalType": abnormal, "abnormalLevel": level, "abnormalTime": "30", "isDebuff": "true",
		}, effect("Debuff", nil))
	}
	buff := skilltest.Template(3, map[string]string{"operateType": "A2", "abnormalTime": "30"}, effect("Buff", nil))
	return skilltest.Catalog(t, effects.NewRegistry(), debuff(1, "POISON", "3"), debuff(2, "BLEEDING", "1"), buff, tested)
}

func TestDispelBySlot(t *testing.T) {
	for _, tt := range []struct {
		dispel string
		gone   bool
	}{{"POISON,3", true}, {"POISON,2", false}, {"BLEEDING,9;POISON,-1", true}, {"NOT_A_TYPE,9", false}} {
		env, _ := skilltest.NewEnv(t, rnd.NewScripted(t), nil)
		cat := bDispelCatalog(t, skilltest.Template(9, nil, effect("DispelBySlot", map[string]string{"dispel": tt.dispel})))
		p := skilltest.NewPlayer(env, 1)
		skilltest.Skill(t, cat, 1).ApplyTo(env, p, p)
		p.Reset()
		skilltest.Skill(t, cat, 9).ApplyTo(env, p, p)
		if gone := p.EL.BySkillID(1) == nil; gone != tt.gone {
			t.Errorf("%s: poison removed %v, want %v", tt.dispel, gone, tt.gone)
		}
		if tt.gone && !bHasMessage(p, skills.MsgEffectOfS1Removed) {
			t.Errorf("%s: messages %v", tt.dispel, p.MessageIDs())
		}
		if !skilltest.Skill(t, cat, 9).HasEffectType(skills.TypeDispelBySlot) {
			t.Errorf("not a DISPEL_BY_SLOT skill")
		}
	}
}

func TestDispelBySlotProbability(t *testing.T) {
	if data.AbnormalBleeding > data.AbnormalPoison {
		t.Fatal("the test expects BLEEDING before POISON in the enum")
	}
	// The slots go in enum order: BLEEDING (60: kept), then POISON (10:
	// removed, whatever its level).
	env, _ := skilltest.NewEnv(t, rnd.NewScripted(t, rnd.Get(100, 60), rnd.Get(100, 10)), nil)
	cat := bDispelCatalog(t, skilltest.Template(9, nil, effect("DispelBySlotProbability", map[string]string{"dispel": "POISON;BLEEDING", "rate": "50"})))
	m := skilltest.NewMonster(env, 1)
	skilltest.Skill(t, cat, 1).ApplyTo(env, m, m)
	skilltest.Skill(t, cat, 2).ApplyTo(env, m, m)
	skilltest.Skill(t, cat, 9).ApplyTo(env, m, m)
	if m.EL.BySkillID(1) != nil || m.EL.BySkillID(2) == nil {
		t.Fatalf("poison %v, bleeding %v", m.EL.BySkillID(1), m.EL.BySkillID(2))
	}
	m.Raid = true
	skilltest.Skill(t, cat, 9).ApplyTo(env, m, m) // no draw
}

func TestDispelByCategory(t *testing.T) {
	t.Run("buffs", func(t *testing.T) {
		// From the last buff: 12 is static (no roll), 11 resists (30), 10
		// goes (10).
		env, _ := skilltest.NewEnv(t, rnd.NewScripted(t, rnd.Get(100, 30), rnd.Get(100, 10)), nil)
		buff := func(id int, extra ...string) data.SkillTemplate {
			v := map[string]string{"operateType": "A2", "abnormalTime": "30"}
			for i := 0; i+1 < len(extra); i += 2 {
				v[extra[i]] = extra[i+1]
			}
			return skilltest.Template(id, v, effect("Buff", nil))
		}
		cancel := skilltest.Template(9, map[string]string{"magicLevel": "40"},
			effect("DispelByCategory", map[string]string{"slot": "buff", "rate": "25", "max": "2"}))
		cat := skilltest.Catalog(t, effects.NewRegistry(), buff(10), buff(11), buff(12, "isMagic", "2"), cancel)
		a, p := skilltest.NewPlayer(env, 1), skilltest.NewPlayer(env, 2)
		for _, id := range []int{10, 11, 12} {
			skilltest.Skill(t, cat, id).ApplyTo(env, p, p)
		}
		skilltest.Skill(t, cat, 9).ApplyTo(env, a, p)
		if p.EL.BySkillID(10) != nil || p.EL.BySkillID(11) == nil || p.EL.BySkillID(12) == nil {
			t.Fatalf("buffs left: 10 %v, 11 %v, 12 %v", p.EL.BySkillID(10) != nil, p.EL.BySkillID(11) != nil, p.EL.BySkillID(12) != nil)
		}
	})
	t.Run("debuffs", func(t *testing.T) {
		env, _ := skilltest.NewEnv(t, rnd.NewScripted(t, rnd.Get(100, 50)), nil)
		cat := bDispelCatalog(t, skilltest.Template(9, nil,
			effect("DispelByCategory", map[string]string{"slot": "debuff", "rate": "50", "max": "1"})))
		p := skilltest.NewPlayer(env, 1)
		skilltest.Skill(t, cat, 1).ApplyTo(env, p, p)
		skilltest.Skill(t, cat, 2).ApplyTo(env, p, p)
		skilltest.Skill(t, cat, 9).ApplyTo(env, p, p)
		if p.EL.BySkillID(2) != nil || p.EL.BySkillID(1) == nil {
			t.Fatalf("the last debuff must go, and only it")
		}
		p.Dead = true
		skilltest.Skill(t, cat, 9).ApplyTo(env, p, p) // no draw on the dead
	})
}

func TestDispelAll(t *testing.T) {
	env, _ := skilltest.NewEnv(t, rnd.NewScripted(t), nil)
	cat := bDispelCatalog(t, skilltest.Template(9, nil, effect("DispelAll", nil)))
	p := skilltest.NewPlayer(env, 1)
	for _, id := range []int{1, 3} {
		skilltest.Skill(t, cat, id).ApplyTo(env, p, p)
	}
	skilltest.Skill(t, cat, 9).ApplyTo(env, p, p)
	if !p.EL.IsEmpty() || !skilltest.Skill(t, cat, 9).HasEffectType(skills.TypeDispel) {
		t.Fatalf("effects left: %d", len(p.EL.Effects()))
	}
}

// bControlCase is a control effect's expected start, flags and end.
type bControlCase struct {
	name       string
	flags      []skills.EffectFlag
	aborts     []string
	startAI    []skills.AIEvent
	npcExitAI  []skills.AIEvent
	playerExit []skills.AIEvent
}

func TestControlEffects(t *testing.T) {
	tests := []bControlCase{
		{"Root", []skills.EffectFlag{skills.FlagRooted}, []string{"move"}, []skills.AIEvent{skills.AIRooted}, []skills.AIEvent{skills.AIThink}, nil},
		{"Paralyze", []skills.EffectFlag{skills.FlagParalyzed}, []string{"attack", "cast", "move"},
			[]skills.AIEvent{skills.AIIdle, skills.AIParalyzed}, []skills.AIEvent{skills.AIThink}, nil},
		{"Petrification", []skills.EffectFlag{skills.FlagParalyzed, skills.FlagInvul}, []string{"attack", "cast", "move"},
			[]skills.AIEvent{skills.AIParalyzed}, []skills.AIEvent{skills.AIThink}, nil},
		{"Mute", []skills.EffectFlag{skills.FlagMuted}, []string{"cast"}, []skills.AIEvent{skills.AIMuted}, nil, nil},
		{"PhysicalMute", []skills.EffectFlag{skills.FlagPhysicalMuted}, nil, []skills.AIEvent{skills.AIMuted}, nil, nil},
		{"Sleep", []skills.EffectFlag{skills.FlagSleep}, []string{"attack", "cast", "move"},
			[]skills.AIEvent{skills.AISleeping}, []skills.AIEvent{skills.AIThink}, []skills.AIEvent{skills.AIActive}},
		{"Invincible", []skills.EffectFlag{skills.FlagInvul}, nil, nil, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, player := range []bool{false, true} {
				env, _, sk := setup(t, bDebuff, effect(tt.name, nil))
				a := skilltest.NewPlayer(env, 1)
				c := skilltest.NewMonster(env, 2)
				exit := tt.npcExitAI
				if player {
					c, exit = skilltest.NewPlayer(env, 2), tt.playerExit
				}
				sk.ApplyTo(env, a, c)
				for _, f := range tt.flags {
					if !c.EL.IsAffected(f) {
						t.Errorf("player %v: flag %d missing", player, f)
					}
				}
				if !slices.Equal(c.Aborts, tt.aborts) || !slices.Equal(c.AIEvents(), tt.startAI) {
					t.Errorf("player %v: aborts %v, AI %v", player, c.Aborts, c.AIEvents())
				}
				c.EL.StopSkill(skills.FinishRemoved, 1)
				if got := c.AIEvents()[len(tt.startAI):]; !slices.Equal(got, exit) {
					t.Errorf("player %v: AI at the end %v, want %v", player, got, exit)
				}
				if c.EL.Flags() != 0 {
					t.Errorf("player %v: flags left %b", player, c.EL.Flags())
				}
			}
		})
	}
	t.Run("PhysicalMute spares raids", func(t *testing.T) {
		env, _, sk := setup(t, bDebuff, effect("PhysicalMute", nil))
		raid := skilltest.NewMonster(env, 2)
		raid.Raid = true
		sk.ApplyTo(env, skilltest.NewPlayer(env, 1), raid)
		if raid.EL.IsAffected(skills.FlagPhysicalMuted) {
			t.Fatalf("a raid was muted")
		}
	})
}

func TestFear(t *testing.T) {
	env, sched, sk := setup(t, bDebuff, effect("Fear", nil))
	a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
	m.Casting, m.Abortable = true, true
	sk.ApplyTo(env, a, m)
	if !slices.Equal(m.Aborts, []string{"cast"}) || len(m.AI) != 1 || m.AI[0].Event != skills.AIAfraidStart || m.AI[0].Other != a {
		t.Fatalf("aborts %v, AI %v", m.Aborts, m.AI)
	}
	// Five ticks, whatever the data says.
	sched.Advance(5*bTick - time.Millisecond)
	if len(m.AI) != 1 {
		t.Fatalf("ticked early")
	}
	sched.Advance(time.Millisecond)
	if !slices.Equal(m.AIEvents(), []skills.AIEvent{skills.AIAfraidStart, skills.AIAfraid}) || !m.EL.IsAffected(skills.FlagFear) {
		t.Fatalf("AI %v", m.AIEvents())
	}
	for _, typ := range []data.InstanceType{data.InstanceDefender, data.InstanceSiegeFlag} {
		d := skilltest.New(env, 3, typ)
		sk.ApplyTo(env, a, d)
		if d.EL.IsAffected(skills.FlagFear) {
			t.Errorf("%s was afraid", typ)
		}
	}
	w := skilltest.NewMonster(env, 4)
	w.Rc = data.RaceSiegeWeapon
	sk.ApplyTo(env, a, w)
	if w.EL.IsAffected(skills.FlagFear) {
		t.Errorf("a siege weapon was afraid")
	}
}

func TestTargetCancel(t *testing.T) {
	values := map[string]string{"operateType": "A1", "magicLevel": "10", "targetType": "ONE", "effectPoint": "-100"}
	// 10 + 100 − 1 + 30 − INT 0 = 139.
	env, _, sk := setup(t, values, effect("TargetCancel", nil), rnd.Get(100, 99))
	a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
	m.Tgt = a
	sk.ApplyTo(env, a, m)
	if m.Tgt != nil || !slices.Equal(m.Aborts, []string{"attack", "cast"}) || !slices.Equal(m.AIEvents(), []skills.AIEvent{skills.AIIdle}) {
		t.Fatalf("target %v, aborts %v, AI %v", m.Tgt, m.Aborts, m.AIEvents())
	}
	// 10 − 50 − 1 + 30 = −11: never.
	env, _, sk = setup(t, values, effect("TargetCancel", map[string]string{"chance": "-50"}), rnd.Get(100, 0))
	m = skilltest.NewMonster(env, 2)
	m.Tgt = a
	sk.ApplyTo(env, a, m)
	if m.Tgt == nil {
		t.Fatalf("the target was cancelled")
	}
}

func TestManaDrainOverTime(t *testing.T) {
	t.Run("ManaDamOverTime", func(t *testing.T) {
		env, sched, sk := setup(t, bDebuff, bTicking("ManaDamOverTime", "1", map[string]string{"power": "10"}))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		sk.ApplyTo(env, a, m)
		sched.Advance(bTick)
		if want := 1000 - 10*bPerTick; m.MP != want {
			t.Fatalf("MP %v, want %v", m.MP, want)
		}
		m.MP = 5
		sched.Advance(bTick)
		if m.MP != 0 || m.EL.BySkillID(1) == nil {
			t.Fatalf("MP %v; a non-toggle keeps going", m.MP)
		}
	})
	for _, name := range []string{"ManaDamOverTime", "SilentMove", "MpConsumePerLevel"} {
		t.Run(name+" toggle without MP", func(t *testing.T) {
			env, sched, sk := setup(t, bToggle, bTicking(name, "1", map[string]string{"power": "10"}))
			p := skilltest.NewPlayer(env, 1)
			p.Lvl = 76
			p.MP = 5
			sk.ApplyTo(env, p, p)
			sched.Advance(bTick)
			if p.MP != 5 || !bHasMessage(p, 140) || p.EL.BySkillID(1) != nil {
				t.Fatalf("MP %v, messages %v, on %v", p.MP, p.MessageIDs(), p.EL.BySkillID(1) != nil)
			}
		})
	}
	t.Run("SilentMove", func(t *testing.T) {
		env, sched, sk := setup(t, bToggle, bTicking("SilentMove", "1", map[string]string{"power": "10"}))
		p := skilltest.NewPlayer(env, 1)
		sk.ApplyTo(env, p, p)
		sched.Advance(bTick)
		if want := 1000 - 10*bPerTick; p.MP != want || !p.EL.IsAffected(skills.FlagSilentMove) {
			t.Fatalf("MP %v, want %v", p.MP, want)
		}
	})
	t.Run("MpConsumePerLevel", func(t *testing.T) {
		env, sched, sk := setup(t, bToggle, bTicking("MpConsumePerLevel", "1", map[string]string{"power": "10"}))
		p := skilltest.NewPlayer(env, 1)
		p.Lvl = 16 // (16 − 1) / 7.5 = 2
		sk.ApplyTo(env, p, p)
		sched.Advance(bTick)
		if want := 1000 - 10*bPerTick*2; p.MP != want {
			t.Fatalf("MP %v, want %v", p.MP, want)
		}
	})
}

func TestFatalBlow(t *testing.T) {
	values := map[string]string{"operateType": "A1", "power": "100", "blowChance": "50", "targetType": "ONE", "effectPoint": "-100"}
	t.Run("hit from behind", func(t *testing.T) {
		// Draws: the skill evasion, the blow (50 × 2 behind), the critical
		// roll, the counter-attack.
		env, _, sk := setup(t, values, effect("FatalBlow", nil),
			rnd.Get(100, 50), rnd.Get(100, 99), rnd.Get(1000, 999), rnd.Get(100, 50))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		a.Loc = skills.Location{X: -100}
		sk.ApplyTo(env, a, m)
		// 77 × (100 + 100) / 100 × 1.2.
		base := 154.0
		if want := base * 1.2; !slices.Equal(m.Damage, []float64{want}) || !slices.Equal(a.DamageMsgs, []int{184}) {
			t.Fatalf("damage %v (want %v), messages %v", m.Damage, want, a.DamageMsgs)
		}
	})
	t.Run("blow fails in front", func(t *testing.T) {
		env, _, sk := setup(t, values, effect("FatalBlow", nil), rnd.Get(100, 50), rnd.Get(100, 50))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		a.Loc = skills.Location{X: 100}
		sk.ApplyTo(env, a, m)
		if len(m.Damage) != 0 {
			t.Fatalf("damage %v", m.Damage)
		}
	})
	t.Run("critical with soulshot", func(t *testing.T) {
		crit := map[string]string{"baseCritRate": "1"}
		for k, v := range values {
			crit[k] = v
		}
		// The critical roll: 1 × 10 × STR bonus 1 = 10 > 0.
		env, _, sk := setup(t, crit, effect("FatalBlow", nil),
			rnd.Get(100, 50), rnd.Get(100, 0), rnd.Get(1000, 0), rnd.Get(100, 50))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		a.Loc = skills.Location{X: 0, Y: -100} // at the side
		a.Shots[skills.Soulshot] = true
		_ = a.AddModifiers(stats.NewModifier(stats.CritDmgPos, stats.OpAdd, 1, stats.Owner{Kind: stats.OwnerItem, ID: 1}))
		sk.ApplyTo(env, a, m)
		// 77 × (100 + 145.8) / 100 × 1.5 (position) × 1.1 (side), doubled.
		pAtk, pos, side := 145.8, 1.5, 1.1
		want := 77 * (100 + pAtk) / 100 * pos * side * 2
		if len(m.Damage) != 1 || m.Damage[0] < want-1e-9 || m.Damage[0] > want+1e-9 {
			t.Fatalf("damage %v, want %v", m.Damage, want)
		}
	})
}

func TestEnergyDamage(t *testing.T) {
	values := map[string]string{"operateType": "A1", "chargeConsume": "2", "targetType": "ONE", "effectPoint": "-100"}
	t.Run("plain", func(t *testing.T) {
		// Draws: the skill evasion, the critical roll, the counter-attack.
		env, _, sk := setup(t, values, effect("EnergyDamage", map[string]string{"power": "50"}),
			rnd.Get(100, 50), rnd.Float64(0.5), rnd.Get(100, 50))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		sk.ApplyTo(env, a, m)
		// (100 + 50) × (1 + (0 + 2 − 1) × 0.2) × 77 / 100.
		attack, boost := 150.0, 1.2
		want := attack * boost * 77 / 100
		if !slices.Equal(m.Damage, []float64{want}) || !slices.Equal(a.DamageMsgs, []int{int(want)}) {
			t.Fatalf("damage %v, want %v", m.Damage, want)
		}
	})
	t.Run("critical", func(t *testing.T) {
		env, _, sk := setup(t, values, effect("EnergyDamage", map[string]string{"power": "50", "criticalChance": "60"}),
			rnd.Get(100, 50), rnd.Float64(0.5), rnd.Get(100, 50))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		sk.ApplyTo(env, a, m)
		attack, boost := 150.0, 1.2
		if want := attack * boost * 77 / 100 * 2; !slices.Equal(m.Damage, []float64{want}) {
			t.Fatalf("damage %v, want %v", m.Damage, want)
		}
	})
	t.Run("not a player", func(t *testing.T) {
		env, _, sk := setup(t, values, effect("EnergyDamage", map[string]string{"power": "50"}), rnd.Get(100, 50))
		a, m := skilltest.NewMonster(env, 1), skilltest.NewMonster(env, 2)
		sk.ApplyTo(env, a, m)
		if len(m.Damage) != 0 {
			t.Fatalf("damage %v", m.Damage)
		}
	})
}

func TestConsumeBody(t *testing.T) {
	env, _, sk := setup(t, nil, effect("ConsumeBody", nil))
	a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
	sk.ApplyTo(env, a, m)
	if len(m.Decays) != 0 {
		t.Fatalf("a living monster decayed")
	}
	m.Dead = true
	sk.ApplyTo(env, a, m)
	if !slices.Equal(m.Decays, []time.Duration{0}) {
		t.Fatalf("decays %v", m.Decays)
	}
	defs := map[data.EffectScope][]data.EffectDef{
		data.ScopeGeneral: {{Name: "ConsumeBody"}},
		data.ScopeStart:   {{Name: "Buff"}},
	}
	env, _, sk = setup(t, map[string]string{"hitTime": "1500", "coolTime": "0"}, defs)
	a, m = skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
	m.Dead = true
	sk.ApplyTo(env, a, m)
	// 1500 / 300 (P.Atk.Spd) × 300.
	if !slices.Equal(m.Decays, []time.Duration{1500 * time.Millisecond}) {
		t.Fatalf("decays %v", m.Decays)
	}
}

func TestLethal(t *testing.T) {
	values := map[string]string{"operateType": "A1", "magicLevel": "10", "targetType": "ONE", "effectPoint": "-100"}
	params := map[string]string{"fullLethal": "5", "halfLethal": "10"}
	tests := []struct {
		name    string
		player  bool
		roll    int
		hp, cp  float64
		msgs    []sp.SystemMessageID
		caster  []sp.SystemMessageID
		texts   int
		casterT int
	}{
		{"monster full", false, 40000, 1, 1000, nil, []sp.SystemMessageID{1668}, 0, 0},
		{"monster half", false, 100000, 500, 1000, nil, nil, 0, 1},
		{"monster miss", false, 200000, 1000, 1000, nil, nil, 0, 0},
		{"player full", true, 40000, 1, 1, []sp.SystemMessageID{1667}, []sp.SystemMessageID{1668}, 0, 0},
		{"player half", true, 100000, 1000, 1, nil, nil, 2, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, _, sk := setup(t, values, effect("Lethal", params), rnd.Get(1000000, tt.roll))
			a := skilltest.NewPlayer(env, 1)
			c := skilltest.NewMonster(env, 2)
			if tt.player {
				c = skilltest.NewPlayer(env, 2)
			}
			c.Lvl = 10
			sk.ApplyTo(env, a, c)
			if c.HP != tt.hp || c.CP != tt.cp || !slices.Equal(c.MessageIDs(), tt.msgs) || len(c.Texts) != tt.texts {
				t.Fatalf("HP %v, CP %v, messages %v, texts %v", c.HP, c.CP, c.MessageIDs(), c.Texts)
			}
			if !slices.Equal(a.MessageIDs(), tt.caster) || len(a.Texts) != tt.casterT {
				t.Fatalf("caster messages %v, texts %v", a.MessageIDs(), a.Texts)
			}
		})
	}
	t.Run("no roll", func(t *testing.T) {
		env, _, sk := setup(t, values, effect("Lethal", params))
		a := skilltest.NewPlayer(env, 1)
		high := skilltest.NewMonster(env, 2)
		high.Lvl = 17 // magic level 10 < 17 − 6
		sk.ApplyTo(env, a, high)
		immune := skilltest.NewMonster(env, 3)
		immune.NoLethal = true
		sk.ApplyTo(env, a, immune)
		a.NoDamage = true
		sk.ApplyTo(env, a, skilltest.NewMonster(env, 4))
	})
}

func TestMagicalDamageMp(t *testing.T) {
	t.Run("burn", func(t *testing.T) {
		// Draws: MagicAffected's Gaussian, the magic critical roll, the
		// magic success.
		env, _, sk := setup(t, map[string]string{"operateType": "A1", "isMagic": "1", "power": "10", "magicLevel": "1", "targetType": "ONE", "effectPoint": "-100"},
			effect("MagicalDamageMp", nil), rnd.Gaussian(0), rnd.Get(1000, 500), rnd.Get(100, 0))
		a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
		sk.ApplyTo(env, a, m)
		// sqrt(100) × 10 × 1000 / 97 / 100.
		mp := 1000.0
		want := 10 * 10 * (mp / 97) / 100
		if m.MP != 1000-want {
			t.Fatalf("MP %v, want %v", m.MP, 1000-want)
		}
		wantMsg := sp.SystemMessage{ID: 1867, Params: []sp.SystemMessageParam{sp.NumberParam(10)}}
		if len(a.Messages) != 1 || a.Messages[0].ID != wantMsg.ID || !slices.Equal(a.Messages[0].Params, wantMsg.Params) {
			t.Fatalf("caster messages %+v", a.Messages)
		}
	})
	t.Run("not affected", func(t *testing.T) {
		env, _, sk := setup(t, bMagic, effect("MagicalDamageMp", nil), rnd.Gaussian(-1))
		a, p := skilltest.NewPlayer(env, 1), skilltest.NewPlayer(env, 2)
		sk.ApplyTo(env, a, p)
		if p.MP != 1000 || !slices.Equal(a.MessageIDs(), []sp.SystemMessageID{skills.MsgYourAttackHasFailed}) || len(p.Texts) != 1 {
			t.Fatalf("MP %v, caster %v, target texts %v", p.MP, a.MessageIDs(), p.Texts)
		}
	})
}

func TestImmobileBuff(t *testing.T) {
	env, _, sk := setup(t, map[string]string{"operateType": "A2", "abnormalTime": "10"}, effect("ImmobileBuff", nil))
	p := skilltest.NewPlayer(env, 1)
	sk.ApplyTo(env, p, p)
	if !p.Immobilized || !sk.HasEffectType(skills.TypeBuff) {
		t.Fatalf("not immobilized")
	}
	p.EL.StopSkill(skills.FinishRemoved, 1)
	if p.Immobilized {
		t.Fatalf("still immobilized")
	}
}

func TestMaxHp(t *testing.T) {
	t.Run("percent with heal", func(t *testing.T) {
		env, _, sk := setup(t, map[string]string{"operateType": "A2", "abnormalTime": "10"},
			effect("MaxHp", map[string]string{"type": "PER", "power": "10", "heal": "true"}))
		p := skilltest.NewPlayer(env, 1)
		sk.ApplyTo(env, p, p)
		want := sp.SystemMessage{ID: skills.MsgS1HPRestored, Params: []sp.SystemMessageParam{sp.NumberParam(100)}}
		if p.MaxHP() != 1100 || p.HP != 1100 || !slices.ContainsFunc(p.Messages, func(m sp.SystemMessage) bool {
			return m.ID == want.ID && slices.Equal(m.Params, want.Params)
		}) {
			t.Fatalf("max HP %d, HP %v, messages %+v", p.MaxHP(), p.HP, p.Messages)
		}
		p.EL.StopSkill(skills.FinishRemoved, 1)
		if p.MaxHP() != 1000 || p.HP != 1000 {
			t.Fatalf("after: max HP %d, HP %v", p.MaxHP(), p.HP)
		}
	})
	t.Run("difference by another", func(t *testing.T) {
		env, _, sk := setup(t, map[string]string{"operateType": "A2", "abnormalTime": "10"},
			effect("MaxHp", map[string]string{"power": "200", "heal": "true"}))
		a, p := skilltest.NewPlayer(env, 1), skilltest.NewPlayer(env, 2)
		p.HP = 500
		sk.ApplyTo(env, a, p)
		want := sp.SystemMessage{ID: skills.MsgS2HPRestoredByS1, Params: []sp.SystemMessageParam{sp.TextParam("c1"), sp.NumberParam(200)}}
		if p.MaxHP() != 1200 || p.HP != 700 || !slices.ContainsFunc(p.Messages, func(m sp.SystemMessage) bool {
			return m.ID == want.ID && slices.Equal(m.Params, want.Params)
		}) {
			t.Fatalf("max HP %d, HP %v, messages %+v", p.MaxHP(), p.HP, p.Messages)
		}
	})
}

func TestResurrection(t *testing.T) {
	env, _, sk := setup(t, map[string]string{"targetType": "PC_BODY"}, effect("Resurrection", map[string]string{"power": "20"}))
	a, p := skilltest.NewPlayer(env, 1), skilltest.NewPlayer(env, 2)
	p.Dead = true
	sk.ApplyTo(env, a, p)
	if len(p.ReviveAsks) != 1 || p.ReviveAsks[0] != (skilltest.ReviveAsk{Reviver: a, Power: 20}) || !p.Dead {
		t.Fatalf("revive asks %+v", p.ReviveAsks)
	}
	for _, tt := range []struct{ wit, want float64 }{{1, 20}, {1.5, 30}, {3, 80}, {5, 90}} {
		npc, m := skilltest.NewMonster(env, 3), skilltest.NewMonster(env, 4)
		npc.Bonus = map[data.BaseStat]float64{data.BaseWIT: tt.wit}
		m.Dead = true
		sk.ApplyTo(env, npc, m)
		if !slices.Equal(m.Revives, []float64{tt.want}) || m.Dead {
			t.Errorf("WIT bonus %v: revived with %v, want %v", tt.wit, m.Revives, tt.want)
		}
	}
}

func TestFocusEnergy(t *testing.T) {
	env, _, sk := setup(t, nil, effect("FocusEnergy", map[string]string{"charge": "2"}))
	p, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
	sk.ApplyTo(env, p, p)
	sk.ApplyTo(env, p, m)
	if !slices.Equal(p.ChargeCalls, [][2]int{{1, 2}}) || p.ChargesV != 1 || m.ChargeCalls != nil {
		t.Fatalf("charge calls %v, charges %d", p.ChargeCalls, p.ChargesV)
	}
}

func TestHpByLevel(t *testing.T) {
	env, _, sk := setup(t, nil, effect("HpByLevel", map[string]string{"power": "300"}))
	a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
	a.HP, m.HP = 900, 500
	sk.ApplyTo(env, a, m)
	want := sp.SystemMessage{ID: skills.MsgS1HPRestored, Params: []sp.SystemMessageParam{sp.NumberParam(100)}}
	if a.HP != 1000 || m.HP != 500 || len(a.Messages) != 1 || !slices.Equal(a.Messages[0].Params, want.Params) {
		t.Fatalf("caster HP %v, target HP %v, messages %+v", a.HP, m.HP, a.Messages)
	}
}

func TestDeathLink(t *testing.T) {
	// Draws: the magic critical roll, the magic success, Shield Deflect
	// Magic.
	env, _, sk := setup(t, bMagic, effect("DeathLink", nil), rnd.Get(1000, 500), rnd.Get(100, 0), rnd.Get(100, 50))
	a, m := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2)
	a.HP = 250 // power × (2 − 2 × 250 / 1000) = 12 × 1.5 = 18
	sk.ApplyTo(env, a, m)
	// 91 × sqrt(100) / 100 × 18 = 163.8.
	if !slices.Equal(m.Damage, []float64{163}) || !slices.Equal(a.DamageMsgs, []int{163}) || !sk.HasEffectType(skills.TypeDeathLink) {
		t.Fatalf("damage %v, messages %v", m.Damage, a.DamageMsgs)
	}
}
