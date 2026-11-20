package inventory

// The charged shots of a weapon (Mobius Item._shotsMask): soulshots and
// spiritshots charge the weapon, not its owner, so the charge goes with the
// weapon when it is taken off or traded. The charge is not saved.

// maxShotKinds is the number of shot kinds an item keeps.
const maxShotKinds = 8

// ChargedShot reports whether shot kind is charged on the item (Mobius
// Item.isChargedShot). The kinds are the caller's (skills.ShotType); kinds
// from 8 up are never charged.
func (it *Item) ChargedShot(kind uint8) bool {
	return kind < maxShotKinds && it.shots&(1<<kind) != 0
}

// SetChargedShot charges or spends shot kind on the item (Mobius
// Item.setChargedShot).
func (it *Item) SetChargedShot(kind uint8, charged bool) {
	if kind >= maxShotKinds {
		return
	}
	if charged {
		it.shots |= 1 << kind
	} else {
		it.shots &^= 1 << kind
	}
}
