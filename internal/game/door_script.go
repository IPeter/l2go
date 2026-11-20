package game

import "github.com/IPeter/l2go/internal/scripting"

// The doors as the scripts see them (scripting.DoorControl, the Lua doors
// module): ScriptOutput passes the calls to the connected server. They run
// on the loop, while a handler runs.

var _ scripting.DoorControl = (*ScriptOutput)(nil)

// OpenDoor opens the door (Mobius DoorData.getDoor(id).openMe()); false for
// an unknown door or without a server.
func (o *ScriptOutput) OpenDoor(id int) bool {
	s := o.srv.Load()
	return s != nil && s.OpenDoor(id)
}

// CloseDoor closes the door (Mobius closeMe); false for an unknown door or
// without a server.
func (o *ScriptOutput) CloseDoor(id int) bool {
	s := o.srv.Load()
	return s != nil && s.CloseDoor(id)
}

// DoorIsOpen reports the door's state; ok is false for an unknown door.
func (o *ScriptOutput) DoorIsOpen(id int) (open, ok bool) {
	s := o.srv.Load()
	if s == nil {
		return false, false
	}
	d, ok := s.doors.byID[id]
	if !ok {
		return false, false
	}
	return d.open, true
}
