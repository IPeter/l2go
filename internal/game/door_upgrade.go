package game

import "github.com/IPeter/l2go/internal/scripting"

// Castle door upgrades (Mobius Castle.setDoorUpgrade, loadDoorUpgrade and
// removeDoorUpgrade, and DoorStat's HP ratio): the lord buys a door a
// higher maximum HP from the chamberlain; the ratio is stored with the
// castle (storage.Castle.DoorUpgrades) and taken again at startup; a mid
// victory takes every upgrade away.

// The startup half of the upgrades runs after the castles are made: the
// features run in name order, "doors" and "residence" first.
func init() { registerFeature("residence_doors", setupCastleDoorUpgrades) }

var _ scripting.CastleDoorUpgrades = castleScriptRef{}

// setupCastleDoorUpgrades is Castle.loadDoorUpgrade for every castle: the
// stored ratios apply, each door at its new full HP.
func setupCastleDoorUpgrades(s *Server) error {
	for _, c := range s.residence.castles {
		for _, id := range sortedKeys(c.doorUpgrades) {
			c.setDoorUpgrade(id, c.doorUpgrades[id], false)
		}
	}
	return nil
}

// door returns the castle's door with the door ID (Castle.getDoor), nil for
// another door or an ID of 0 or less.
func (c *Castle) door(doorID int32) *Door {
	if doorID <= 0 {
		return nil
	}
	d, ok := c.srv.Door(int(doorID))
	if !ok || !c.Contains(d.Position()) {
		return nil
	}
	return d
}

// setDoorUpgrade is Castle.setDoorUpgrade: the castle's door (any door when
// the castle has none) gets the HP ratio and its new full HP; with save the
// ratio is stored with the castle.
func (c *Castle) setDoorUpgrade(doorID, ratio int32, save bool) {
	d := c.door(doorID)
	if d == nil && len(c.doors()) == 0 && doorID > 0 {
		d, _ = c.srv.Door(int(doorID))
	}
	if d == nil {
		return
	}
	d.setHPRatio(ratio)
	c.srv.setDoorHP(d, float64(d.maxHP), true)
	if save {
		c.doorUpgrades[doorID] = d.HPRatio()
		c.save()
	}
}

// removeDoorUpgrades is Castle.removeDoorUpgrade: every door of the castle
// goes back to ratio 1, its HP cut to the new maximum, and the stored
// upgrades go (the caller saves the castle).
func (c *Castle) removeDoorUpgrades() {
	for _, d := range c.doors() {
		d.setHPRatio(1)
		c.srv.setDoorHP(d, d.curHP, true)
	}
	c.doorUpgrades = make(map[int32]int32)
}

// DoorUpgradeLevel is DoorStat.getUpgradeHpRatio of the castle's door with
// the ID; false for another door.
func (r castleScriptRef) DoorUpgradeLevel(doorID int) (int, bool) {
	d := r.c.door(int32(doorID)) //nolint:gosec // door IDs fit
	if d == nil {
		return 0, false
	}
	return int(d.HPRatio()), true
}

// SetDoorUpgrade is Castle.setDoorUpgrade(doorId, level, true): the door's
// HP ratio, stored with the castle.
func (r castleScriptRef) SetDoorUpgrade(doorID, level int) {
	if level < 1 || level > 100 {
		return
	}
	r.c.setDoorUpgrade(int32(doorID), int32(level), true) //nolint:gosec // checked
}
