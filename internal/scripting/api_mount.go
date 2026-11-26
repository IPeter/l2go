package scripting

import lua "github.com/yuin/gopher-lua"

// The mount calls (Mobius Player.isMounted, getMountType, getMountNpcId,
// getMountLevel, dismount and mount(npcId, controlItemObjId, useFood)): the
// wyvern managers trade a strider for a wyvern, the bosses hinder striders.

// Mount types (Mobius MountType ordinals), as player:mount_type() answers.
const (
	MountNone    = 0
	MountStrider = 1
	MountWyvern  = 2
)

// PlayerMount is the part of a PlayerRef behind the mount calls. Without it
// the player rides nothing and cannot mount.
type PlayerMount interface {
	// MountType is MountNone, MountStrider or MountWyvern.
	MountType() int
	// MountNpcID and MountLevel are the ridden NPC and its level, 0 on foot.
	MountNpcID() int
	MountLevel() int
	// Dismount is Mobius dismount: false when the player may not get off
	// here.
	Dismount() bool
	// Mount is Mobius mount(npcId, controlItemObjId, useFood): the player
	// rides the NPC at its own level; false when it cannot take its weapon
	// off.
	Mount(npcID int, controlItem int32, useFood bool) bool
}

// playerMountMethods are the Player methods of this file.
func playerMountMethods() map[string]lua.LGFunction {
	mountOf := func(L *lua.LState, method string) PlayerMount {
		m, _ := checkPlayer(L, method).(PlayerMount)
		return m
	}
	number := func(method string, get func(PlayerMount) int) lua.LGFunction {
		return func(L *lua.LState) int {
			n := 0
			if m := mountOf(L, method); m != nil {
				n = get(m)
			}
			L.Push(lua.LNumber(n))
			return 1
		}
	}
	return map[string]lua.LGFunction{
		"is_mounted": func(L *lua.LState) int {
			m := mountOf(L, "is_mounted")
			L.Push(lua.LBool(m != nil && m.MountType() != MountNone))
			return 1
		},
		"mount_type":   number("mount_type", PlayerMount.MountType),
		"mount_npc_id": number("mount_npc_id", PlayerMount.MountNpcID),
		"mount_level":  number("mount_level", PlayerMount.MountLevel),
		"dismount": func(L *lua.LState) int {
			m := mountOf(L, "dismount")
			L.Push(lua.LBool(m != nil && m.Dismount()))
			return 1
		},
		"mount": func(L *lua.LState) int {
			m := mountOf(L, "mount")
			npcID := int(checkInt(L, 2, "player:mount"))
			control := int32(L.OptInt(3, 0)) //nolint:gosec // object IDs fit in int32
			useFood := L.OptBool(4, false)
			L.Push(lua.LBool(m != nil && m.Mount(npcID, control, useFood)))
			return 1
		},
	}
}
