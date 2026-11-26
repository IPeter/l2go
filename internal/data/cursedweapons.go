package data

// Cursed weapon data (Plan 14 M11m): the two weapons of Mobius
// CursedWeapons.xml, Demonic Sword Zariche and Blood Sword Akamanah, in the
// cursedweapons domain of Load.

import (
	"fmt"
	"io/fs"
)

// CursedWeaponsFile is the path of the cursed weapon file, relative to the
// data folder.
const CursedWeaponsFile = "cursedweapons/cursedweapons.json"

func init() {
	registerDomain(domain{name: "cursedweapons", load: loadCursedWeapons, link: linkCursedWeapons})
}

// CursedWeapon is one <item> of CursedWeapons.xml (Mobius
// CursedWeaponsManager.load).
type CursedWeapon struct {
	// ItemID is the weapon's item.
	ItemID int    `json:"itemId"`
	Name   string `json:"name"`
	// SkillID is the weapon's passive skill, which gains a level every
	// StageKills kills.
	SkillID int `json:"skillId"`
	// DropRate is the chance of a drop per kill, out of 100,000.
	DropRate int `json:"dropRate"`
	// Duration is the weapon's life in minutes, counted from its drop.
	Duration int `json:"duration"`
	// DurationLost is how many minutes each kill takes off the life; the
	// life is checked every DurationLost × 12 seconds.
	DurationLost int `json:"durationLost"`
	// DisappearChance is the percent chance (Rnd.get(100) <= chance) that
	// the weapon vanishes when its holder dies (Mobius disapearChance).
	DisappearChance int `json:"disappearChance"`
	// StageKills is the kills per skill level.
	StageKills int `json:"stageKills"`
}

// loadCursedWeapons reads and checks the cursed weapon file.
func loadCursedWeapons(fsys fs.FS, d *Data) error {
	var list []CursedWeapon
	if err := readJSON(fsys, CursedWeaponsFile, &list); err != nil {
		return err
	}
	if err := checkCursedWeapons(list); err != nil {
		return err
	}
	d.CursedWeapons = list
	return nil
}

// CheckCursedWeapons checks cursed weapon records as the loader does: item
// IDs positive and unique, a skill, and positive durations and stage kills
// (Mobius divides by both).
func CheckCursedWeapons(list []CursedWeapon) error { return checkCursedWeapons(list) }

func checkCursedWeapons(list []CursedWeapon) error {
	var p problems
	seen := make(map[int]bool, len(list))
	for i, w := range list {
		where := fmt.Sprintf("%s: weapon %d (item %d)", CursedWeaponsFile, i, w.ItemID)
		switch {
		case w.ItemID <= 0:
			p.addf("%s: no item ID", where)
		case seen[w.ItemID]:
			p.addf("%s: item listed twice", where)
		}
		seen[w.ItemID] = true
		if w.SkillID <= 0 {
			p.addf("%s: no skill ID", where)
		}
		if w.DropRate < 0 || w.DisappearChance < 0 {
			p.addf("%s: negative drop rate or disappear chance", where)
		}
		if w.Duration <= 0 || w.DurationLost <= 0 || w.StageKills <= 0 {
			p.addf("%s: duration, durationLost and stageKills must be positive", where)
		}
	}
	return p.err()
}

// linkCursedWeapons checks that the weapons are items, when the items are
// loaded.
func linkCursedWeapons(d *Data) error {
	if !d.Loaded("items") {
		return nil
	}
	var p problems
	for _, w := range d.CursedWeapons {
		if _, ok := d.ItemTemplate(w.ItemID); !ok {
			p.addf("%s: item %d is not an item", CursedWeaponsFile, w.ItemID)
		}
	}
	return p.err()
}

// CursedWeapon returns the cursed weapon of an item, false if the item is
// none.
func (d *Data) CursedWeapon(itemID int) (*CursedWeapon, bool) {
	for i := range d.CursedWeapons {
		if d.CursedWeapons[i].ItemID == itemID {
			return &d.CursedWeapons[i], true
		}
	}
	return nil, false
}
