package game

import (
	"fmt"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// teleportWatchdog puts a teleported player back into the world when its
// client never sends Appearing. Mobius has the same watchdog
// (TeleportWatchdogTimeout), off by default.
const teleportWatchdog = 5 * time.Second

// teleportZOffset is what Mobius teleToLocation adds to the height.
const teleportZOffset = 5

// teleportState is a teleport waiting for Appearing.
type teleportState struct {
	from     world.Position // where the player was before the teleport
	watchdog world.TimerID
}

// Teleport moves p to pos the way Mobius teleToLocation does. The OnTeleport
// hooks run first (they stop the movement with StopMove and drop the
// target); then p leaves the world (DeleteObject to its viewers),
// TeleportToLocation goes to p and to the players around the old position,
// and p waits outside the world at pos for its client's Appearing, which
// puts it back (handleAppearing). If Appearing does not come within 5
// seconds, p is put back anyway.
//
// The height is the geodata ground nearest to pos.Z (the given Z without
// geodata) plus 5, as in Mobius. A heading of 0 keeps p's heading. Callers
// that want Mobius's random offset around the destination add it to pos.
// Teleporting again before Appearing changes the destination. Loop-only.
func (s *Server) Teleport(p *Player, pos world.Position) error {
	if p.state != stateInGame {
		return ErrNotInWorld
	}
	pos.Z = s.geo.Height(pos.X, pos.Y, pos.Z) + teleportZOffset
	if _, ok := world.RegionOf(pos.X, pos.Y, pos.Z); !ok {
		return fmt.Errorf("game: teleport to %v: %w", pos, world.ErrOutOfWorld)
	}
	if p.teleport == nil {
		from := p.Position()
		s.runTeleport(p, pos)
		if p.state != stateInGame {
			return ErrNotInWorld // a hook made the player leave
		}
		p.Move = nil
		s.RemoveObject(p)
		p.teleport = &teleportState{from: from}
	} else {
		s.loop.Cancel(p.teleport.watchdog)
	}

	s.world.Broadcast(p, serverpackets.NewTeleportToLocationPacket(serverpackets.TeleportToLocation{
		ObjectID: p.ObjectID(),
		X:        pos.X, Y: pos.Y, Z: pos.Z,
		Heading: pos.Heading,
	}), true)
	if pos.Heading == 0 {
		pos.Heading = p.Position().Heading
	}
	_ = p.SetPosition(pos) // out of the world now

	p.teleport.watchdog = s.loop.After(teleportWatchdog, func() {
		if p.teleport != nil && p.state == stateInGame {
			s.log.Warn("game: no Appearing after a teleport, putting the player back", p.logArgs()...)
			s.finishTeleport(p)
		}
	})
	return nil
}

// finishTeleport puts a teleported player back into the world at its new
// position (Mobius onTeleported): it sees and is seen again, the players
// around get its CharInfo once more, and the OnPlayerMoved hooks run.
func (s *Server) finishTeleport(p *Player) {
	t := p.teleport
	if t == nil {
		return
	}
	p.teleport = nil
	s.loop.Cancel(t.watchdog)
	if err := s.AddObject(p); err != nil {
		// Teleport checked the position, so only a duplicate ID gets here.
		s.log.Error("game: cannot put a teleported player back into the world", append(p.logArgs(), "pos", p.Position(), "err", err)...)
		return
	}
	s.world.MarkUpdate(p, world.UpdateCharInfo)
	s.runMoved(p, t.from, p.Position())
}

// handleAppearing handles Appearing (0x30): the client has loaded the area
// after a teleport. Mobius puts the player back into the world and sends
// UserInfo (also when no teleport is pending).
func (s *Server) handleAppearing(p *Player, _ []byte) {
	if p.teleport != nil {
		s.finishTeleport(p)
	}
	p.sendUserInfo()
}
