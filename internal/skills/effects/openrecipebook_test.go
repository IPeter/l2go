package effects_test

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/effects"
	"github.com/IPeter/l2go/internal/skills/skilltest"
)

// bookPlayer is a player adapter that opens recipe books.
type bookPlayer struct {
	*skilltest.Creature
	opened []bool
}

func (b *bookPlayer) AsPlayer() skills.Player     { return b }
func (b *bookPlayer) OpenRecipeBook(dwarven bool) { b.opened = append(b.opened, dwarven) }

var _ effects.RecipeBookOpener = (*bookPlayer)(nil)

// Dwarven Craft (1321) and Common Craft (1322) open the caster's book; a
// caster that cannot open books is left alone.
func TestOpenRecipeBook(t *testing.T) {
	if got := effects.Ported(); !slices.Contains(got, data.EffectOpenDwarfRecipeBook) || !slices.Contains(got, data.EffectOpenCommonRecipeBook) {
		t.Fatal("the recipe book effects are not ported")
	}
	for _, tt := range []struct {
		effect  string
		dwarven bool
	}{{"OpenDwarfRecipeBook", true}, {"OpenCommonRecipeBook", false}} {
		env, _, sk := setup(t, map[string]string{"isMagic": "2"}, effect(tt.effect, nil))
		p := &bookPlayer{Creature: skilltest.NewPlayer(env, 1)}
		sk.ApplyTo(env, p, p)
		if !slices.Equal(p.opened, []bool{tt.dwarven}) {
			t.Errorf("%s opened %v, want [%v]", tt.effect, p.opened, tt.dwarven)
		}
		plain := skilltest.NewPlayer(env, 2)
		sk.ApplyTo(env, plain, plain) // no panic, nothing to open
	}
}
