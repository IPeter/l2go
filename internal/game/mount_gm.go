package game

import (
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage"
)

// The ride commands (Mobius AdminRide and the user command Dismount) and the
// Lua mount calls (scripting.PlayerMount).

// userCmdDismount is /dismount (Mobius Dismount).
const userCmdDismount = 62

// registerMountCommands registers //ride_strider, //ride_wyvern,
// //unride, //unride_strider, //unride_wyvern and /dismount.
func (s *Server) registerMountCommands() []error {
	gm := func(name string, fn GMCommand) error {
		return s.RegisterGMCommand(name, storage.AccessLevelAdmin, fn)
	}
	return []error{
		gm("ride_strider", func(p *Player, _ []string) { s.gmRide(p, mountNpcStrider) }),
		gm("ride_wyvern", func(p *Player, _ []string) { s.gmRide(p, mountNpcWyvern) }),
		gm("unride", s.gmUnride),
		gm("unride_strider", s.gmUnride),
		gm("unride_wyvern", s.gmUnride),
		s.RegisterUserCommand(userCmdDismount, func(p *Player) {
			if s.mounted(p) {
				s.dismount(p)
			}
		}),
	}
}

// gmRideTarget is AdminRide.getRideTarget: the GM's player target, else the
// GM.
func gmRideTarget(p *Player) *Player {
	if t, ok := p.Target.(*Player); ok && t != p {
		return t
	}
	return p
}

// gmRide is //ride_strider and //ride_wyvern: the target rides the NPC at
// its level without feed, unless it rides or has a summon already.
func (s *Server) gmRide(p *Player, npcID int) {
	t := gmRideTarget(p)
	if s.mounted(t) || s.SummonOf(t) != nil {
		gmSysMessage(p, "Target already have a summon.")
		return
	}
	s.mountNpc(t, npcID, 0, false)
}

// gmUnride is //unride and its two aliases: the target gets off.
func (s *Server) gmUnride(p *Player, _ []string) { s.dismount(gmRideTarget(p)) }

// ----- Lua (scripting.PlayerMount) -----

var _ scripting.PlayerMount = (*Player)(nil)

// MountType is getMountType().ordinal(). Loop-only.
func (p *Player) MountType() int { return int(p.mount.typ) }

// MountNpcID is getMountNpcId. Loop-only.
func (p *Player) MountNpcID() int { return p.mount.npcID }

// MountLevel is getMountLevel. Loop-only.
func (p *Player) MountLevel() int { return p.mount.level }

// Dismount is Mobius dismount. Loop-only.
func (p *Player) Dismount() bool {
	if p.srv == nil || p.mount.typ == serverpackets.MountNone {
		return p.mount.typ == serverpackets.MountNone
	}
	return p.srv.dismount(p)
}

// Mount is Mobius mount(npcId, controlItemObjId, useFood). Loop-only.
func (p *Player) Mount(npcID int, controlItem int32, useFood bool) bool {
	if p.srv == nil {
		return false
	}
	return p.srv.mountNpc(p, npcID, controlItem, useFood)
}
