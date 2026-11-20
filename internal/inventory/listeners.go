package inventory

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
)

// bowListener puts arrows of the bow's grade into the left hand when a bow
// goes into the right hand, and takes off the arrows or the lure when a bow
// or a fishing rod comes off (Mobius BowCrossRodListener).
type bowListener struct{}

func (bowListener) Unequipped(inv *Inventory, slot storage.PaperdollSlot, it *Item) {
	if slot != storage.PaperdollRHand {
		return
	}
	if isWeapon(it.tmpl, data.WeaponBow) || isWeapon(it.tmpl, data.WeaponFishingrod) {
		if inv.paperdoll[storage.PaperdollLHand] != nil {
			inv.setPaperdoll(storage.PaperdollLHand, nil)
		}
	}
}

func (bowListener) Equipped(inv *Inventory, slot storage.PaperdollSlot, it *Item) {
	if slot != storage.PaperdollRHand || !isWeapon(it.tmpl, data.WeaponBow) {
		return
	}
	if arrow := inv.FindArrowForBow(it.tmpl); arrow != nil {
		inv.setPaperdoll(storage.PaperdollLHand, arrow)
	}
}

// StatSet is the part of a player's stat set the stats listener changes;
// *stats.Player implements it.
type StatSet interface {
	Equip(slot stats.Slot, owner stats.Owner, it stats.Item) error
	Unequip(slot stats.Slot)
}

// StatsListener adds a worn item's stat modifiers to the owner's stat set
// and removes them when it comes off (Mobius StatsListener). Add it to the
// inventory before loading.
type StatsListener struct {
	Stats StatSet
	// OnError, if set, is told when the stat set refuses an item (bad
	// template data).
	OnError func(it *Item, err error)
}

// Equipped adds the item's modifiers for slot.
func (l *StatsListener) Equipped(_ *Inventory, slot storage.PaperdollSlot, it *Item) {
	err := l.Stats.Equip(stats.Slot(slot), stats.ItemOwner(it.objectID), StatsItem(it.tmpl, int(it.enchant))) //nolint:gosec // a paperdoll slot from 0 up
	if err != nil && l.OnError != nil {
		l.OnError(it, err)
	}
}

// Unequipped removes the modifiers of the item in slot.
func (l *StatsListener) Unequipped(_ *Inventory, slot storage.PaperdollSlot, _ *Item) {
	l.Stats.Unequip(stats.Slot(slot)) //nolint:gosec // a paperdoll slot from 0 up
}

// StatsItem returns an item template, enchanted to enchant, as the stat
// engine needs it. The enchant functions the data lists (op "enchant" and
// "enchanthp") are left out: stats.ItemModifiers computes them from the
// enchant level.
func StatsItem(t *data.Item, enchant int) stats.Item {
	it := stats.Item{
		Type:     t.Type,
		BodyPart: t.BodyPart,
		Grade:    t.Grade.String(),
		Enchant:  enchant,
	}
	if t.Weapon != nil {
		it.WeaponType = t.Weapon.Type.String()
	}
	if t.Armor != nil {
		it.ArmorType = t.Armor.Type.String()
	}
	for _, st := range t.Stats {
		if st.Op == data.OpEnchant || st.Op == data.OpEnchantHP {
			continue
		}
		it.Stats = append(it.Stats, stats.ItemStat{Stat: st.Stat.String(), Value: st.Value})
	}
	return it
}
