package effects_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/skilltest"
)

// fishWorld is a World that fishes, recording the calls.
type fishWorld struct {
	skills.World
	calls []string
}

func (w *fishWorld) StartFishing(c skills.Creature) {
	w.calls = append(w.calls, fmt.Sprintf("start %d", c.ObjectID()))
}

func (w *fishWorld) Pump(c skills.Creature, sk *skills.Skill, power float64) {
	w.calls = append(w.calls, fmt.Sprintf("pump %d skill %d power %v", c.ObjectID(), sk.ID, power))
}

func (w *fishWorld) Reel(c skills.Creature, sk *skills.Skill, power float64) {
	w.calls = append(w.calls, fmt.Sprintf("reel %d skill %d power %v", c.ObjectID(), sk.ID, power))
}

var _ skills.Fisher = (*fishWorld)(nil)

// Fishing, Pumping and Reeling hand a player caster to the world's Fisher,
// with their types (FISHING_START, FISHING); NPCs and worlds that do not
// fish are left alone.
func TestFishingEffects(t *testing.T) {
	for _, tt := range []struct {
		effect string
		params map[string]string
		typ    skills.EffectType
		want   string
	}{
		{"Fishing", nil, skills.TypeFishingStart, "start 1"},
		{"Pumping", map[string]string{"power": "0.9375"}, skills.TypeFishing, "pump 1 skill 1 power 0.9375"},
		{"Reeling", map[string]string{"power": "5"}, skills.TypeFishing, "reel 1 skill 1 power 5"},
	} {
		env, _, sk := setup(t, map[string]string{"isMagic": "2"}, effect(tt.effect, tt.params))
		if !sk.HasEffectType(tt.typ) {
			t.Errorf("%s: not of type %v", tt.effect, tt.typ)
		}
		w := &fishWorld{World: env.World}
		env.World = w
		p := skilltest.NewPlayer(env, 1)
		sk.ApplyTo(env, p, p)
		npc := skilltest.NewMonster(env, 2)
		sk.ApplyTo(env, npc, npc)
		if !slices.Equal(w.calls, []string{tt.want}) {
			t.Errorf("%s: calls %q, want [%q]", tt.effect, w.calls, tt.want)
		}
		env.World = w.World
		sk.ApplyTo(env, p, p) // no Fisher: nothing happens
	}
}
