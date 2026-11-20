package stats

// Weapon is a creature's active weapon (Mobius getActiveWeaponItem) as the
// attack code reads it: its item, type and reuse delay. The game layer
// builds it from the item data.
type Weapon struct {
	// ItemID is the item template ID.
	ItemID int
	// Type is weapon_type: "SWORD", "BOW", "FIST", ...
	Type string
	// ReuseDelay is reuse_delay (bows), 0 without one.
	ReuseDelay int
}

// FistsItemID is Player.findFistsWeaponItem: the fists item (244-252) a
// player of classID holds without a weapon. Mobius picks it from the active
// class when the character loads, by class ID range; IDs outside the
// ranges (third classes among them) get the Human Fighter's fists (246).
func FistsItemID(classID int) int {
	switch {
	case classID >= 0x00 && classID <= 0x09:
		return 246 // Human Fighter Fist
	case classID >= 0x0a && classID <= 0x11:
		return 251 // Human Mystic Fist
	case classID >= 0x12 && classID <= 0x18:
		return 244 // Elven Fighter Fist
	case classID >= 0x19 && classID <= 0x1e:
		return 249 // Elven Mystic Fist
	case classID >= 0x1f && classID <= 0x25:
		return 245 // Dark Fighter Fist
	case classID >= 0x26 && classID <= 0x2b:
		return 250 // Dark Mystic Fist
	case classID >= 0x2c && classID <= 0x30:
		return 248 // Orc Fighter Fist
	case classID >= 0x31 && classID <= 0x34:
		return 252 // Orc Shaman Fist
	case classID >= 0x35 && classID <= 0x39:
		return 247 // Dwarven Fighter Fist
	}
	return 246
}

// PlayerWeapon is Player.getActiveWeaponItem: the equipped weapon, or
// without one (nil) the class fists of FistsItemID, type FIST and no reuse.
// A player always has an active weapon. The fists are never equipped, so
// their stats (randomDamage 5 among them) do not apply.
func PlayerWeapon(classID int, equipped *Weapon) Weapon {
	if equipped != nil {
		return *equipped
	}
	return Weapon{ItemID: FistsItemID(classID), Type: "FIST"}
}

// AttackType is Creature.getAttackType: the active weapon's type, or the
// template's base attack type without one (w nil). An NPC's active weapon
// is its template's rhand item when that item is a weapon; players always
// have one (PlayerWeapon).
func AttackType(w *Weapon, templateAttackType string) string {
	if w != nil {
		return w.Type
	}
	return templateAttackType
}

// Arm sets the attack fields of f from its active weapon w (nil for none)
// and its template's attack type: Armed, the attack shape of AttackType and
// the weapon's reuse delay (Creature.calculateReuseTime reads the weapon's,
// never the NPC template's reuseDelay). player selects Mobius's player rule
// that sends fists down the dual path.
func (f *Fighter) Arm(w *Weapon, templateAttackType string, player bool) {
	f.Armed = w != nil
	f.Attack = AttackShapeOf(AttackType(w, templateAttackType), player)
	f.ReuseDelay = 0
	if w != nil {
		f.ReuseDelay = w.ReuseDelay
	}
}

// AttackCountMax is the number of targets a pole hits
// (Creature.doAttackHitByPole): the atkCountMax modifiers on 1, cut to an
// int. Poles add targets only when it is above 1.
func AttackCountMax(c Combatant) int {
	return int(javaInt(c.CalcFor(AtkCountMax, 1, Context{})))
}
