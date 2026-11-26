package scripttest

import "github.com/IPeter/l2go/internal/scripting"

var _ scripting.CastleDoorUpgrades = castleRef{}

// DoorUpgradeLevel is the ratio in DoorUpgrades of a door in Doors, 1
// without one; false for a door not in Doors.
func (r castleRef) DoorUpgradeLevel(doorID int) (int, bool) {
	if _, ok := r.c.Doors[doorID]; !ok {
		return 0, false
	}
	if l, ok := r.c.DoorUpgrades[doorID]; ok {
		return l, true
	}
	return 1, true
}

// SetDoorUpgrade records the ratio of a door in Doors in DoorUpgrades.
func (r castleRef) SetDoorUpgrade(doorID, level int) {
	if _, ok := r.c.Doors[doorID]; !ok {
		return
	}
	if r.c.DoorUpgrades == nil {
		r.c.DoorUpgrades = make(map[int]int)
	}
	r.c.DoorUpgrades[doorID] = level
}
