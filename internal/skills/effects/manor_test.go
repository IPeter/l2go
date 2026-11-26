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

// manorTestWorld is a skilltest world with the manor: it records the sowings
// and harvests the effects ask for.
type manorTestWorld struct {
	*skilltest.World
	sown, harvested []int32 // the effected object IDs
}

func (w *manorTestWorld) Sow(_, effected skills.Creature) { w.sown = append(w.sown, effected.ObjectID()) }
func (w *manorTestWorld) Harvest(_, effected skills.Creature) {
	w.harvested = append(w.harvested, effected.ObjectID())
}

// Sow and Harvesting hand a player's skill on a monster to the world's
// manor: Sow on any monster, Harvesting on a dead one only.
func TestManorEffects(t *testing.T) {
	w := &manorTestWorld{World: &skilltest.World{}}
	env, err := skills.NewEnv(rnd.NewScripted(t), skilltest.NewSched(), w, skills.DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	cat := skilltest.Catalog(t, effects.NewRegistry(),
		skilltest.Template(1, nil, effect("Sow", nil)), skilltest.Template(2, nil, effect("Harvesting", nil)))
	env.Skills = cat
	sow, harvest := skilltest.Skill(t, cat, 1), skilltest.Skill(t, cat, 2)
	p, m, other := skilltest.NewPlayer(env, 1), skilltest.NewMonster(env, 2), skilltest.NewPlayer(env, 3)
	sow.ApplyTo(env, p, m)
	sow.ApplyTo(env, p, other)  // not a monster
	sow.ApplyTo(env, m, m)      // not a player
	harvest.ApplyTo(env, p, m)  // alive
	m.Dead = true
	harvest.ApplyTo(env, p, m)
	if len(w.sown) != 1 || w.sown[0] != 2 || len(w.harvested) != 1 || w.harvested[0] != 2 {
		t.Errorf("sown %v, harvested %v", w.sown, w.harvested)
	}
	// Without a manor world nothing happens.
	plain, _ := skilltest.NewEnv(t, rnd.NewScripted(t), &skilltest.World{})
	plain.Skills = cat
	sow.ApplyTo(plain, skilltest.NewPlayer(plain, 1), skilltest.NewMonster(plain, 2))
	for _, n := range []data.EffectName{data.EffectSow, data.EffectHarvesting} {
		found := false
		for _, got := range effects.Ported() {
			found = found || got == n
		}
		if !found {
			t.Errorf("%v is not ported", n)
		}
	}
}
