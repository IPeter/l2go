package skills_test

import (
	"strconv"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/effects"
	"github.com/IPeter/l2go/internal/skills/skilltest"
)

// probeLog counts what probe effects did.
type probeLog struct {
	starts, ticks, exits int
}

// probe is a test effect: continuous unless its "instant" parameter is
// set, ticking when given ticks, keeping on ticking with "keep", setting
// the stunned flag.
type probe struct {
	skills.Base
	log     *probeLog
	instant bool
	keep    bool
}

func (p *probe) Instant() bool { return p.instant }

func (p *probe) Flags() skills.EffectFlags { return skills.FlagStunned.Mask() }

func (p *probe) OnStart(*skills.BuffInfo) { p.log.starts++ }

func (p *probe) OnActionTime(*skills.BuffInfo) bool {
	p.log.ticks++
	return p.keep
}

func (p *probe) OnExit(*skills.BuffInfo) { p.log.exits++ }

// probeName is the name the probe effect takes in the tests.
const probeName = "Stun"

// fixture is an Env over a scripted random source, the Wave A effects and
// the probe.
type fixture struct {
	env   *skills.Env
	sched *skilltest.Sched
	world *skilltest.World
	rand  *rnd.Scripted
	reg   *skills.Registry
	log   *probeLog
}

func newFixture(t *testing.T, draws ...rnd.Draw) *fixture {
	t.Helper()
	f := &fixture{rand: rnd.NewScripted(t, draws...), world: &skilltest.World{}, log: &probeLog{}}
	f.env, f.sched = skilltest.NewEnv(t, f.rand, f.world)
	f.reg = effects.NewRegistry()
	f.reg.Register(data.EffectStun, func(def *data.SkillEffect) (skills.Effect, error) {
		instant, _ := def.Params.Bool("instant", false)
		keep, _ := def.Params.Bool("keep", false)
		return &probe{Base: skills.NewBase(def), log: f.log, instant: instant, keep: keep}, nil
	})
	return f
}

func (f *fixture) catalog(t *testing.T, tmpls ...data.SkillTemplate) *skills.Catalog {
	t.Helper()
	return skilltest.Catalog(t, f.reg, tmpls...)
}

// mulPAtk is a Buff that multiplies P.Atk.
func mulPAtk(rate string) data.EffectDef {
	return data.EffectDef{Name: "Buff", Funcs: []data.StatFuncDef{{Stat: data.StatPowerAttack, Op: data.OpMul, Order: 20, Value: rate}}}
}

// buffSkill is an A2 buff of an abnormal type and level that multiplies
// P.Atk, lasting 1200 s unless more says otherwise.
func buffSkill(id int, abnormal string, level int, rate string, more map[string]string) data.SkillTemplate {
	v := map[string]string{
		"operateType": "A2", "targetType": "ONE", "isMagic": "1",
		"abnormalType": abnormal, "abnormalLevel": strconv.Itoa(level), "abnormalTime": "1200",
	}
	for k, x := range more {
		v[k] = x
	}
	return skilltest.Template(id, v, skilltest.General(mulPAtk(rate)))
}

// probeDef is a probe effect definition.
func probeDef(params map[string]string, ticks string) data.EffectDef {
	return data.EffectDef{Name: probeName, Params: params, Ticks: ticks}
}

func skillIDs(list []*skills.BuffInfo) []int {
	ids := make([]int, len(list))
	for i, b := range list {
		ids[i] = b.Skill().ID
	}
	return ids
}
