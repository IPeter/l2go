package effects

import (
	"slices"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
)

// restorationRandom is Mobius RestorationRandom, the extractable items:
// one roll of 100 × nextDouble picks the first group whose chance range
// holds it (the ranges are laid end to end); its items go to the effected
// player, each enchanted to Between(min, max) when its max enchant is above
// 0. Nothing picked is "There was nothing found inside of that.". Then one
// message per item instance and one inventory update.
//
// RateExtractable is 1 (Rates.ini), so the counts are the data's. Mobius
// sends the messages in HashMap order, which has no fixed order; they go
// in the order the items were made here.
type restorationRandom struct {
	skills.Base
	groups []data.RestorationGroup
}

func init() {
	register(data.EffectRestorationRandom, func(def *data.SkillEffect) (skills.Effect, error) {
		groups := slices.Clone(def.Items)
		for i := range groups {
			// ExtractableProductItem keeps the chance as a float.
			groups[i].Chance = float64(float32(groups[i].Chance))
			groups[i].Items = slices.Clone(groups[i].Items)
		}
		return &restorationRandom{Base: skills.NewBase(def), groups: groups}, nil
	})
}

func (*restorationRandom) Instant() bool { return true }

func (e *restorationRandom) OnStart(info *skills.BuffInfo) {
	env, effector, effected := info.Env(), info.Effector(), info.Effected()
	rndNum := 100 * env.Rand.Float64()
	chanceFrom := 0.0
	var creation []data.RestorationItem
	for _, g := range e.groups {
		if rndNum >= chanceFrom && rndNum <= g.Chance+chanceFrom {
			creation = append(creation, g.Items...)
			break
		}
		chanceFrom += g.Chance
	}
	player := effected.AsPlayer()
	if player == nil {
		return
	}
	if len(creation) == 0 {
		skills.Send(effected, msgNothingFoundInside)
		return
	}
	type extracted struct {
		item  skills.CreatedItem
		count int
	}
	var got []extracted
	for _, it := range creation {
		if it.ID <= 0 || it.Count <= 0 {
			continue
		}
		count := int(javaInt(float64(it.Count)))
		enchant := 0
		if it.MaxEnchant > 0 {
			enchant = env.Rand.Between(it.MinEnchant, it.MaxEnchant)
		}
		item, ok := player.AddItem(it.ID, count, enchant, effector, false)
		if !ok {
			continue
		}
		i := slices.IndexFunc(got, func(x extracted) bool { return x.item.ObjectID == item.ObjectID })
		if i >= 0 {
			got[i].count += count
			got[i].item = item
		} else {
			got = append(got, extracted{item: item, count: count})
		}
	}
	if len(got) == 0 {
		return
	}
	ids := make([]int, 0, len(got))
	for _, x := range got {
		if !slices.Contains(ids, x.item.ItemID) {
			ids = append(ids, x.item.ItemID)
		}
		itemName := sp.ItemNameParam(int32(x.item.ItemID)) //nolint:gosec // item IDs are small
		switch {
		case x.count > 1:
			skills.Send(effected, msgObtainedS2S1, itemName, sp.NumberParam(int32(x.count))) //nolint:gosec // counts are ints in Mobius
		case x.item.Enchant > 0:
			skills.Send(effected, msgObtainedEnchantedS1S2, sp.NumberParam(int32(x.item.Enchant)), itemName) //nolint:gosec // enchant levels are small
		default:
			skills.Send(effected, msgObtainedS1, itemName)
		}
	}
	player.SendItemsUpdate(ids)
}
