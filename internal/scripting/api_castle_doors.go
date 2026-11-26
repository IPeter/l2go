package scripting

import (
	lua "github.com/yuin/gopher-lua"
)

// CastleDoorUpgrades is the part of a CastleRef behind
// castle:door_upgrade_level(door_id) and castle:set_door_upgrade(door_id,
// level): the castle door upgrades the chamberlain sells (Mobius
// DoorStat.getUpgradeHpRatio and Castle.setDoorUpgrade). A CastleRef
// without it has no doors to upgrade.
type CastleDoorUpgrades interface {
	// DoorUpgradeLevel is the HP ratio of the castle's door with the ID (1
	// without an upgrade); false when the castle has no such door.
	DoorUpgradeLevel(doorID int) (int, bool)
	// SetDoorUpgrade gives the castle's door the HP ratio and its new full
	// HP, stored with the castle.
	SetDoorUpgrade(doorID, level int)
}

// castleDoorMethods adds castle:door_upgrade_level(door_id), which returns
// nil for a door that is not the castle's, and
// castle:set_door_upgrade(door_id, level) to the Lua castle t.
func castleDoorMethods(L *lua.LState, t *lua.LTable, c CastleRef) {
	du, _ := c.(CastleDoorUpgrades)
	t.RawSetString("door_upgrade_level", L.NewFunction(func(L *lua.LState) int {
		door := int(checkInt(L, 2, "castle:door_upgrade_level"))
		if du == nil {
			L.Push(lua.LNil)
			return 1
		}
		level, ok := du.DoorUpgradeLevel(door)
		if !ok {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(lua.LNumber(level))
		return 1
	}))
	t.RawSetString("set_door_upgrade", L.NewFunction(func(L *lua.LState) int {
		const name = "castle:set_door_upgrade"
		door := int(checkInt(L, 2, name))
		level := checkInt(L, 3, name)
		if level < 1 || level > 100 {
			L.ArgError(3, name+": the level must be 1 to 100")
			return 0
		}
		if du != nil {
			du.SetDoorUpgrade(door, int(level))
		}
		return 0
	}))
}
