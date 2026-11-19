package gameserver

import (
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/storage"
)

// TestPaperdollEquip equips items in order and checks where each one ends
// up. The rules come from Mobius Inventory.equipItem and BodyPart.
func TestPaperdollEquip(t *testing.T) {
	const (
		none    = storage.PaperdollNone
		weapon  = data.ItemTypeWeapon
		armor   = data.ItemTypeArmor
		etcItem = data.ItemTypeEtc
	)
	type wear struct {
		bodyPart string
		typ      string
		want     storage.PaperdollSlot // after all items are equipped
		ok       bool                  // what equip returns
	}
	tests := []struct {
		name  string
		items []wear
	}{
		{"right hand", []wear{{"rhand", weapon, storage.PaperdollRHand, true}}},
		{"two hands go to the right hand", []wear{{"lrhand", weapon, storage.PaperdollRHand, true}}},
		{"weapon and shield", []wear{
			{"rhand", weapon, storage.PaperdollRHand, true},
			{"lhand", armor, storage.PaperdollLHand, true},
		}},
		{"two-handed weapon takes off the shield", []wear{
			{"lhand", armor, none, true},
			{"lrhand", weapon, storage.PaperdollRHand, true},
		}},
		{"shield takes off the two-handed weapon", []wear{
			{"lrhand", weapon, none, true},
			{"lhand", armor, storage.PaperdollLHand, true},
		}},
		{"arrows stay with the bow", []wear{
			{"lrhand", weapon, storage.PaperdollRHand, true},
			{"lhand", etcItem, storage.PaperdollLHand, true},
		}},
		{"second weapon replaces the first", []wear{
			{"rhand", weapon, none, true},
			{"lrhand", weapon, storage.PaperdollRHand, true},
		}},
		{"armor set", []wear{
			{"chest", armor, storage.PaperdollChest, true},
			{"legs", armor, storage.PaperdollLegs, true},
			{"head", armor, storage.PaperdollHead, true},
			{"gloves", armor, storage.PaperdollGloves, true},
			{"feet", armor, storage.PaperdollFeet, true},
			{"back", armor, storage.PaperdollCloak, true},
			{"neck", armor, storage.PaperdollNeck, true},
			{"underwear", armor, storage.PaperdollUnder, true},
		}},
		{"shirt is underwear", []wear{{"shirt", armor, storage.PaperdollUnder, true}}},
		{"full armor takes off the legs", []wear{
			{"legs", armor, none, true},
			{"fullarmor", armor, storage.PaperdollChest, true},
		}},
		{"legs take off full armor", []wear{
			{"onepiece", armor, none, true},
			{"legs", armor, storage.PaperdollLegs, true},
		}},
		{"legs keep a normal chest", []wear{
			{"chest", armor, storage.PaperdollChest, true},
			{"legs", armor, storage.PaperdollLegs, true},
		}},
		{"earrings fill left, then right, then replace left", []wear{
			{"rear", armor, none, true},
			{"lear", armor, storage.PaperdollREar, true},
			{"rear;lear", armor, storage.PaperdollLEar, true},
		}},
		{"rings fill left, then right, then replace left", []wear{
			{"rfinger;lfinger", armor, none, true},
			{"rfinger", armor, storage.PaperdollRFinger, true},
			{"lfinger", armor, storage.PaperdollLFinger, true},
		}},
		{"hair accessories", []wear{
			{"hair", armor, storage.PaperdollHair, true},
			{"hair2", armor, storage.PaperdollHair2, true},
		}},
		{"full hair accessory takes both", []wear{
			{"hair", armor, none, true},
			{"hair2", armor, none, true},
			{"hairall", armor, storage.PaperdollHair, true},
		}},
		{"hair replaces a full hair accessory", []wear{
			{"dhair", armor, none, true},
			{"hair", armor, storage.PaperdollHair, true},
		}},
		{"hair2 takes off a full hair accessory", []wear{
			{"hairall", armor, none, true},
			{"hair2", armor, storage.PaperdollHair2, true},
		}},
		{"formal wear takes off armor and weapons", []wear{
			{"rhand", weapon, none, true},
			{"lhand", armor, none, true},
			{"legs", armor, none, true},
			{"head", armor, none, true},
			{"neck", armor, storage.PaperdollNeck, true},
			{"alldress", armor, storage.PaperdollChest, true},
		}},
		{"formal wear refuses armor", []wear{
			{"alldress", armor, storage.PaperdollChest, true},
			{"legs", armor, none, false},
			{"feet", armor, none, false},
		}},
		{"a weapon takes off formal wear", []wear{
			{"alldress", armor, none, true},
			{"rhand", weapon, storage.PaperdollRHand, true},
		}},
		{"items without a body part stay in the inventory", []wear{
			{"", etcItem, none, false},
			{"none", etcItem, none, false},
			{"wolf", armor, none, false},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := make([]storage.Item, len(tt.items))
			for i := range items {
				items[i].ObjectID = int32(i + 1)
			}
			doll := newPaperdoll(items)
			for i, w := range tt.items {
				if ok := doll.equip(i, data.Item{ID: i + 1, Type: w.typ, BodyPart: w.bodyPart}); ok != w.ok {
					t.Errorf("equip %q = %v, want %v", w.bodyPart, ok, w.ok)
				}
			}
			for i, w := range tt.items {
				if items[i].Slot != w.want {
					t.Errorf("item %d (%q): slot %d, want %d", i, w.bodyPart, items[i].Slot, w.want)
				}
			}
		})
	}
}

// Items start unequipped even though the zero slot is PaperdollUnder.
func TestNewPaperdollUnequipsEverything(t *testing.T) {
	items := make([]storage.Item, 3)
	newPaperdoll(items)
	for i, it := range items {
		if it.Slot != storage.PaperdollNone {
			t.Errorf("item %d: slot %d, want PaperdollNone", i, it.Slot)
		}
	}
}
