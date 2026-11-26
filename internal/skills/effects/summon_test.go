package effects_test

import (
	"io"
	"log/slog"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/effects"
	"github.com/IPeter/l2go/internal/skills/skilltest"
)

// summonWorld records what the summoning effects ask of a SummonWorld.
type summonWorld struct {
	*skilltest.World
	can       bool
	servitors []skills.ServitorParams
	pets      int
	cubics    []skills.CubicParams
	maxCubics []int
	unsummons int
}

func (w *summonWorld) CanSummon(skills.Creature) bool { return w.can }
func (w *summonWorld) SummonServitor(_ skills.Creature, _ *skills.Skill, p skills.ServitorParams) {
	w.servitors = append(w.servitors, p)
}
func (w *summonWorld) SummonPet(_, _ skills.Creature, _ *skills.Skill) { w.pets++ }
func (w *summonWorld) SummonCubic(_, _ skills.Creature, _ *skills.Skill, p skills.CubicParams) {
	w.cubics = append(w.cubics, p)
}
func (w *summonWorld) SetMaxCubics(_ skills.Creature, n int) { w.maxCubics = append(w.maxCubics, n) }
func (w *summonWorld) Unsummon(skills.Creature)              { w.unsummons++ }

// summonSetup compiles one skill with the effect over a summonWorld.
func summonSetup(t *testing.T, values map[string]string, name string, params map[string]string, draws ...rnd.Draw) (*skills.Env, *summonWorld, *skills.Skill) {
	t.Helper()
	w := &summonWorld{World: &skilltest.World{}, can: true}
	env, err := skills.NewEnv(rnd.NewScripted(t, draws...), skilltest.NewSched(), w, skills.DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewEnv: %v", err)
	}
	cat := skilltest.Catalog(t, effects.NewRegistry(), skilltest.Template(1, values, effect(name, params)))
	sk := skilltest.Skill(t, cat, 1)
	if r := sk.Refusal(); r != "" {
		t.Fatalf("skill refused: %s", r)
	}
	return env, w, sk
}

func TestSummonEffects(t *testing.T) {
	t.Run("Summon", func(t *testing.T) {
		env, w, sk := summonSetup(t, nil, "Summon", map[string]string{"npcId": "14111", "consumeItemId": "1458", "lifeTime": "1200", "expMultiplier": "0.70"})
		p := skilltest.NewPlayer(env, 1)
		sk.ApplyTo(env, p, p)
		want := skills.ServitorParams{NpcID: 14111, ExpMultiplier: 0.7, ConsumeItemID: 1458, ConsumeItemCount: 1, LifeTime: 1200}
		if len(w.servitors) != 1 || w.servitors[0] != want {
			t.Fatalf("servitors %+v, want %+v", w.servitors, want)
		}
		// A monster does not summon.
		m := skilltest.NewMonster(env, 2)
		sk.ApplyTo(env, m, m)
		if len(w.servitors) != 1 {
			t.Fatal("a monster summoned")
		}
	})
	t.Run("SummonPet", func(t *testing.T) {
		env, w, sk := summonSetup(t, nil, "SummonPet", nil)
		p := skilltest.NewPlayer(env, 1)
		sk.ApplyTo(env, p, p)
		if w.pets != 1 || !sk.HasEffectType(skills.TypeSummonPet) {
			t.Fatalf("pets %d", w.pets)
		}
	})
	t.Run("SummonCubic", func(t *testing.T) {
		env, w, sk := summonSetup(t, nil, "SummonCubic", map[string]string{"cubicId": "1", "cubicDelay": "10", "cubicSkillChance": "12", "cubicPower": "282"})
		p := skilltest.NewPlayer(env, 1)
		sk.ApplyTo(env, p, p)
		want := skills.CubicParams{ID: 1, Power: 282, Delay: 10, MaxCount: -1, SkillChance: 12}
		if len(w.cubics) != 1 || w.cubics[0] != want {
			t.Fatalf("cubics %+v, want %+v", w.cubics, want)
		}
	})
	t.Run("CubicMastery", func(t *testing.T) {
		env, w, sk := summonSetup(t, map[string]string{"operateType": "P"}, "CubicMastery", map[string]string{"cubicCount": "2"})
		p := skilltest.NewPlayer(env, 1)
		sk.AddPassive(env, p)
		if len(w.maxCubics) != 1 || w.maxCubics[0] != 2 {
			t.Fatalf("max cubics %v", w.maxCubics)
		}
		p.EL.StopSkill(skills.FinishRemoved, 1)
		if len(w.maxCubics) != 2 || w.maxCubics[1] != 1 {
			t.Fatalf("max cubics after removal %v", w.maxCubics)
		}
	})
	t.Run("Unsummon", func(t *testing.T) {
		// 100 × nextDouble() = 50 under the chance of 100.
		env, w, sk := summonSetup(t, nil, "Unsummon", nil, rnd.Float64(0.5))
		p := skilltest.NewPlayer(env, 1)
		s := skilltest.New(env, 2, data.InstanceServitor)
		sk.ApplyTo(env, p, s)
		if w.unsummons != 1 {
			t.Fatalf("unsummons %d", w.unsummons)
		}
	})
}

func TestCubicTemplateLevel(t *testing.T) {
	for _, tt := range []struct{ cubicLvl, skillLvl, want int }{
		{0, 3, 3}, {5, 3, 5}, {0, 101, 8}, {0, 115, 10},
	} {
		if got := skills.CubicTemplateLevel(tt.cubicLvl, tt.skillLvl); got != tt.want {
			t.Errorf("CubicTemplateLevel(%d, %d) = %d, want %d", tt.cubicLvl, tt.skillLvl, got, tt.want)
		}
	}
}
