package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// RecipeBookOpener is what a player adapter implements to open its recipe
// books (Plan 14 M11b; the game's adapter checks the store mode and sends
// the book, Mobius RecipeManager.requestBookOpen). An adapter without it
// ignores the effect.
type RecipeBookOpener interface {
	OpenRecipeBook(dwarven bool)
}

// openRecipeBook is Mobius OpenDwarfRecipeBook (Dwarven Craft, 1321) and
// OpenCommonRecipeBook (Common Craft, 1322): a player caster opens the book.
type openRecipeBook struct {
	skills.Base
	dwarven bool
}

func init() {
	for name, dwarven := range map[data.EffectName]bool{
		data.EffectOpenDwarfRecipeBook:  true,
		data.EffectOpenCommonRecipeBook: false,
	} {
		register(name, func(def *data.SkillEffect) (skills.Effect, error) {
			return &openRecipeBook{Base: skills.NewBase(def), dwarven: dwarven}, nil
		})
	}
}

func (*openRecipeBook) Instant() bool { return true }

func (e *openRecipeBook) OnStart(info *skills.BuffInfo) {
	effector := info.Effector()
	if !isPlayer(effector) {
		return
	}
	if o, ok := effector.AsPlayer().(RecipeBookOpener); ok {
		o.OpenRecipeBook(e.dwarven)
	}
}
