package scripttest

import "github.com/IPeter/l2go/internal/scripting"

var _ scripting.PlayerMount = playerRef{}

// Mount is what a fake player rides (player:mount_type() and friends).
type Mount struct {
	// Type is scripting.MountStrider or MountWyvern.
	Type         int
	NpcID, Level int
	// ControlItem and UseFood are the arguments of the player:mount call
	// that put the player on it.
	ControlItem int32
	UseFood     bool
}

// mountTypeOf is the fake world's Mobius MountType.findByNpcId.
func mountTypeOf(npcID int) int {
	switch npcID {
	case 12526, 12527, 12528:
		return scripting.MountStrider
	case 12621:
		return scripting.MountWyvern
	}
	return scripting.MountNone
}

func (r playerRef) MountType() int {
	if r.p.Mount == nil {
		return scripting.MountNone
	}
	return r.p.Mount.Type
}

func (r playerRef) MountNpcID() int {
	if r.p.Mount == nil {
		return 0
	}
	return r.p.Mount.NpcID
}

func (r playerRef) MountLevel() int {
	if r.p.Mount == nil {
		return 0
	}
	return r.p.Mount.Level
}

// Dismount always succeeds in the fake world.
func (r playerRef) Dismount() bool {
	r.p.Mount = nil
	return true
}

// Mount rides the NPC at the player's level.
func (r playerRef) Mount(npcID int, controlItem int32, useFood bool) bool {
	r.p.Mount = &Mount{Type: mountTypeOf(npcID), NpcID: npcID, Level: r.p.Level, ControlItem: controlItem, UseFood: useFood}
	return true
}
