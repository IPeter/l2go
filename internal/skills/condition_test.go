package skills_test

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/skilltest"
)

func compile(t *testing.T, c data.Condition) skills.Cond {
	t.Helper()
	cond, unported := skills.CompileCondition(&c)
	if len(unported) > 0 {
		t.Fatalf("unported kinds %v", unported)
	}
	return cond
}

func TestConditions(t *testing.T) {
	f := newFixture(t, rnd.Get(100, 29), rnd.Get(100, 30))
	p := skilltest.NewPlayer(f.env, 1)
	m := skilltest.NewMonster(f.env, 2)
	ctx := func(caster, target skills.Creature) skills.ConditionContext {
		return skills.ConditionContext{Env: f.env, Caster: caster, Target: target}
	}
	sword := data.WeaponSword.Mask()
	heavy, light := data.ArmorHeavy.Mask(), data.ArmorLight.Mask()

	type check struct {
		name   string
		setup  func()
		caster skills.Creature
		target skills.Creature
		want   bool
	}
	tests := []struct {
		cond   data.Condition
		checks []check
	}{
		{data.Condition{Kind: data.CondPlayerHP, Int: 50}, []check{
			{"half HP", func() { p.HP = 500 }, p, nil, true},
			{"over half", func() { p.HP = 501 }, p, nil, false},
		}},
		{data.Condition{Kind: data.CondPlayerWeight, Int: 80}, []check{
			{"light", func() { p.Load, p.MaxLoadV = 79, 100 }, p, nil, true},
			{"heavy", func() { p.Load = 80 }, p, nil, false},
			{"an NPC", nil, m, nil, true},
		}},
		{data.Condition{Kind: data.CondPlayerInvsize, Int: 10}, []check{
			{"room", func() { p.InvSize, p.InvLimit = 70, 80 }, p, nil, true},
			{"full", func() { p.InvSize = 71 }, p, nil, false},
		}},
		{data.Condition{Kind: data.CondPlayerCharges, Int: 2}, []check{
			{"charged", func() { p.ChargesV = 2 }, p, nil, true},
			{"an NPC", nil, m, nil, false},
		}},
		{data.Condition{Kind: data.CondUsingKind, Int: sword}, []check{
			{"a sword", func() { p.Worn = sword }, p, nil, true},
			{"no sword", func() { p.Worn = 0 }, p, nil, false},
			{"an NPC with a sword", func() { m.Attack = data.WeaponSword }, m, nil, true},
		}},
		{data.Condition{Kind: data.CondUsingKind, Int: heavy}, []check{
			{"heavy set", func() { p.Chest, p.Legs, p.FullArmor = heavy, heavy, false }, p, nil, true},
			{"mixed set", func() { p.Legs = light }, p, nil, false},
			{"full heavy armor", func() { p.Legs, p.FullArmor = 0, true }, p, nil, true},
			{"an NPC", nil, m, nil, false},
		}},
		{data.Condition{Kind: data.CondTargetRace, Races: []data.Race{data.RaceOrc}}, []check{
			{"an orc", func() { m.Rc = data.RaceOrc }, p, m, true},
			{"no target", nil, p, nil, false},
		}},
		{data.Condition{Kind: data.CondPlayerBehind, Bool: true}, []check{
			{"behind", func() { p.Loc, m.Loc, m.Head = skills.Location{X: -100}, skills.Location{}, 0 }, p, m, true},
			{"in front", func() { m.Head = 32768 }, p, m, false},
		}},
		{data.Condition{Kind: data.CondGameChance, Int: 30}, []check{
			{"29 < 30", nil, p, nil, true},
			{"30 < 30", nil, p, nil, false},
		}},
		{data.Condition{Kind: data.CondNot, Conds: []data.Condition{
			{Kind: data.CondOr, Conds: []data.Condition{
				{Kind: data.CondPlayerLevel, Int: 20},
				{Kind: data.CondPlayerRaces, Races: []data.Race{data.RaceElf}},
			}},
		}}, []check{
			{"neither", func() { p.Lvl, p.Rc = 19, data.RaceHuman }, p, nil, true},
			{"level", func() { p.Lvl = 20 }, p, nil, false},
			{"race", func() { p.Lvl, p.Rc = 1, data.RaceElf }, p, nil, false},
		}},
	}
	for _, tt := range tests {
		cond := compile(t, tt.cond)
		for _, c := range tt.checks {
			if c.setup != nil {
				c.setup()
			}
			if got := cond(ctx(c.caster, c.target)); got != c.want {
				t.Errorf("%s, %s: %v, want %v", tt.cond.Kind, c.name, got, c.want)
			}
		}
	}
}

func TestPreconditions(t *testing.T) {
	f := newFixture(t)
	hp := data.SkillConditions{MsgID: int(skills.MsgS1CannotBeUsedUnsuitable), AddName: true,
		Cond: data.ConditionDef{Kind: data.CondPlayerHP, Value: "50"}}
	text := data.SkillConditions{Msg: "Not now.", Cond: data.ConditionDef{Kind: data.CondPlayerLevel, Value: "10"}}
	unported := data.SkillConditions{Cond: data.ConditionDef{Kind: data.CondNot,
		Conds: []data.ConditionDef{{Kind: data.CondTargetWeight, Value: "50"}}}}
	if skills.ConditionPorted(data.CondTargetWeight) {
		t.Fatalf("pick another unported kind for this test")
	}
	t1 := skilltest.Template(1, nil, nil)
	t1.Conditions = []data.SkillConditions{hp, text}
	t2 := skilltest.Template(2, nil, nil)
	t2.Conditions = []data.SkillConditions{unported}
	cat := f.catalog(t, t1, t2)
	sk := skilltest.Skill(t, cat, 1)
	p := skilltest.NewPlayer(f.env, 1)

	p.HP = 900
	if sk.CheckCondition(f.env, p, nil, false) {
		t.Fatalf("passed at 90 percent HP")
	}
	want := sp.SystemMessage{ID: skills.MsgS1CannotBeUsedUnsuitable, Params: []sp.SystemMessageParam{sp.SkillNameParam(1, 1)}}
	if len(p.Messages) != 1 || p.Messages[0].ID != want.ID || !slices.Equal(p.Messages[0].Params, want.Params) {
		t.Fatalf("messages = %+v, want %+v", p.Messages, want)
	}
	p.HP, p.Lvl = 100, 5
	if sk.CheckCondition(f.env, p, nil, false) || !slices.Equal(p.Texts, []string{"Not now."}) {
		t.Fatalf("level 5 passed or texts %v", p.Texts)
	}
	p.Lvl = 10
	if !sk.CheckCondition(f.env, p, nil, false) {
		t.Fatalf("met conditions failed")
	}
	// A kind without a port fails the whole tree, even under a <not>.
	sk2 := skilltest.Skill(t, cat, 2)
	if sk2.CheckCondition(f.env, p, nil, false) || sk2.Refusal() == "" {
		t.Fatalf("an unported condition passed (refusal %q)", sk2.Refusal())
	}
	if cat.Report().UnportedConditions[data.CondTargetWeight] != 1 {
		t.Fatalf("the report misses the unported kind")
	}
	f.env.Config.GMSkillRestriction = false
	p.GM = true
	if !sk2.CheckCondition(f.env, p, nil, false) {
		t.Fatalf("a GM without GMSkillRestriction must pass")
	}
}
