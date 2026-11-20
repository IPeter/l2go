package effects

import (
	"log/slog"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// restoration is Mobius Restoration: it gives the effected player itemCount
// of itemId (with the obtain message), enchanted to itemEnchantmentLevel
// when above 0. Without a valid item and count the playable gets "There was
// nothing found inside of that." and the server logs a warning.
//
// Pets get the item in their own inventory in Mobius; pets come with M11,
// so a pet gets nothing yet.
type restoration struct {
	skills.Base
	itemID, count, enchant int
}

func init() {
	register(data.EffectRestoration, func(def *data.SkillEffect) (skills.Effect, error) {
		id, err := def.Params.Int("itemId", 0)
		if err != nil {
			return nil, err
		}
		count, err := def.Params.Int("itemCount", 0)
		if err != nil {
			return nil, err
		}
		enchant, err := def.Params.Int("itemEnchantmentLevel", 0)
		if err != nil {
			return nil, err
		}
		return &restoration{Base: skills.NewBase(def), itemID: id, count: count, enchant: enchant}, nil
	})
}

func (*restoration) Instant() bool { return true }

func (e *restoration) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	if effected == nil || !skills.IsPlayable(effected) {
		return
	}
	if e.itemID <= 0 || e.count <= 0 {
		skills.Send(effected, msgNothingFoundInside)
		log := info.Env().Log
		if log == nil {
			log = slog.Default()
		}
		log.Warn("skills: Restoration effect with a wrong item ID or count", "skill", info.Skill().String(), "item", e.itemID, "count", e.count)
		return
	}
	if isPlayer(effected) {
		effected.AsPlayer().AddItem(e.itemID, e.count, e.enchant, info.Effector(), true)
	}
}
