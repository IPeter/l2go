package gameserver

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/storage"
)

// Body parts, as the item data names them (Mobius BodyPart). Some body
// parts have more than one name.
const (
	bodyUnderwear = "underwear"
	bodyShirt     = "shirt"
	bodyREar      = "rear"
	bodyRBracelet = "rbracelet" // Mobius Interlude treats it as an ear
	bodyLEar      = "lear"
	bodyLREar     = "rear;lear"
	bodyNeck      = "neck"
	bodyRFinger   = "rfinger"
	bodyLFinger   = "lfinger"
	bodyLRFinger  = "rfinger;lfinger"
	bodyHead      = "head"
	bodyRHand     = "rhand"
	bodyLHand     = "lhand"
	bodyGloves    = "gloves"
	bodyChest     = "chest"
	bodyLegs      = "legs"
	bodyFeet      = "feet"
	bodyBack      = "back"
	bodyLRHand    = "lrhand"
	bodyFullArmor = "fullarmor"
	bodyOnePiece  = "onepiece"
	bodyHair      = "hair"
	bodyAllDress  = "alldress"
	bodyHair2     = "hair2"
	bodyHairAll   = "hairall"
	bodyDHair     = "dhair"
)

// bodyKind folds the alias names of a body part into one.
func bodyKind(bodyPart string) string {
	switch bodyPart {
	case bodyShirt:
		return bodyUnderwear
	case bodyRBracelet, bodyLEar, bodyLREar:
		return bodyREar
	case bodyLFinger, bodyLRFinger:
		return bodyRFinger
	case bodyOnePiece:
		return bodyFullArmor
	case bodyDHair:
		return bodyHairAll
	}
	return bodyPart
}

// paperdoll puts the starting items of a new character into their
// equipment slots, following Mobius Inventory.equipItem.
type paperdoll struct {
	items []storage.Item
	// worn holds, per slot, the index in items plus one (0 is empty) and
	// the body part of that item.
	worn [storage.PaperdollTotalSlots]struct {
		index int
		kind  string
	}
}

// newPaperdoll returns a paperdoll for items, which it changes in place.
// Every item starts unequipped.
func newPaperdoll(items []storage.Item) *paperdoll {
	for i := range items {
		items[i].Slot = storage.PaperdollNone
	}
	return &paperdoll{items: items}
}

// set puts item index i (with body part kind) into slot, taking off what
// was there. i < 0 empties the slot.
func (p *paperdoll) set(slot storage.PaperdollSlot, i int, kind string) {
	w := &p.worn[slot]
	if w.index > 0 {
		p.items[w.index-1].Slot = storage.PaperdollNone
	}
	if i < 0 {
		w.index, w.kind = 0, ""
		return
	}
	if old := p.items[i].Slot; old.Equipped() {
		p.worn[old].index, p.worn[old].kind = 0, ""
	}
	w.index, w.kind = i+1, kind
	p.items[i].Slot = slot
}

func (p *paperdoll) clear(slot storage.PaperdollSlot) { p.set(slot, -1, "") }

func (p *paperdoll) empty(slot storage.PaperdollSlot) bool { return p.worn[slot].index == 0 }

func (p *paperdoll) holds(slot storage.PaperdollSlot, kind string) bool {
	return !p.empty(slot) && p.worn[slot].kind == kind
}

// equip wears item index i, whose template is it. It reports whether the
// item went into a slot; items without a known body part stay in the
// inventory.
func (p *paperdoll) equip(i int, it data.Item) bool {
	kind := bodyKind(it.BodyPart)

	// Formal wear (alldress) only allows weapons, which take it off.
	if p.holds(storage.PaperdollChest, bodyAllDress) {
		switch kind {
		case bodyLRHand, bodyLHand, bodyRHand:
			p.clear(storage.PaperdollChest)
		case bodyLegs, bodyFeet, bodyGloves, bodyHead:
			return false
		}
	}

	switch kind {
	case bodyLRHand:
		p.clear(storage.PaperdollLHand)
		p.set(storage.PaperdollRHand, i, kind)
	case bodyLHand:
		// A shield takes off a two-handed weapon. Mobius keeps a bow with
		// arrows and a fishing rod with lures; those are the only etc items
		// worn in the left hand.
		if p.holds(storage.PaperdollRHand, bodyLRHand) && it.Type != data.ItemTypeEtc {
			p.clear(storage.PaperdollRHand)
		}
		p.set(storage.PaperdollLHand, i, kind)
	case bodyRHand:
		p.set(storage.PaperdollRHand, i, kind)
	case bodyREar:
		p.set(p.pair(storage.PaperdollLEar, storage.PaperdollREar), i, kind)
	case bodyRFinger:
		p.set(p.pair(storage.PaperdollLFinger, storage.PaperdollRFinger), i, kind)
	case bodyNeck:
		p.set(storage.PaperdollNeck, i, kind)
	case bodyFullArmor:
		p.clear(storage.PaperdollLegs)
		p.set(storage.PaperdollChest, i, kind)
	case bodyChest:
		p.set(storage.PaperdollChest, i, kind)
	case bodyLegs:
		if p.holds(storage.PaperdollChest, bodyFullArmor) {
			p.clear(storage.PaperdollChest)
		}
		p.set(storage.PaperdollLegs, i, kind)
	case bodyFeet:
		p.set(storage.PaperdollFeet, i, kind)
	case bodyGloves:
		p.set(storage.PaperdollGloves, i, kind)
	case bodyHead:
		p.set(storage.PaperdollHead, i, kind)
	case bodyHair:
		if p.holds(storage.PaperdollHair, bodyHairAll) {
			p.clear(storage.PaperdollHair2)
		} else {
			p.clear(storage.PaperdollHair)
		}
		p.set(storage.PaperdollHair, i, kind)
	case bodyHair2:
		if p.holds(storage.PaperdollHair, bodyHairAll) {
			p.clear(storage.PaperdollHair)
		} else {
			p.clear(storage.PaperdollHair2)
		}
		p.set(storage.PaperdollHair2, i, kind)
	case bodyHairAll:
		p.clear(storage.PaperdollHair2)
		p.set(storage.PaperdollHair, i, kind)
	case bodyUnderwear:
		p.set(storage.PaperdollUnder, i, kind)
	case bodyBack:
		p.set(storage.PaperdollCloak, i, kind)
	case bodyAllDress:
		for _, s := range []storage.PaperdollSlot{
			storage.PaperdollLegs, storage.PaperdollLHand, storage.PaperdollRHand,
			storage.PaperdollHead, storage.PaperdollFeet, storage.PaperdollGloves,
		} {
			p.clear(s)
		}
		p.set(storage.PaperdollChest, i, kind)
	default:
		return false
	}
	return true
}

// pair picks the slot for an earring or a ring: first if it is free, else
// second if that is free, else first again.
func (p *paperdoll) pair(first, second storage.PaperdollSlot) storage.PaperdollSlot {
	if !p.empty(first) && p.empty(second) {
		return second
	}
	return first
}
