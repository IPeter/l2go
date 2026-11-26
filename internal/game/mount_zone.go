package game

import (
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
)

// The zones' mount rules (Mobius NoLandingZone and SiegeZone, with
// Player.enteredNoLanding and exitedNoLanding): a wyvern over a no-landing
// zone, or over an active siege that is not its lord's, lands by force after
// a few seconds; a strider in an active siege zone is not allowed but for
// the lord's.

// mountZoneEntered is NoLandingZone.onEnter (zone.go enterZone): a GM is
// told the restrictions do not hold for it; a wyvern rider is warned and
// dismounted after the zone's dismountDelay (5 s by default) unless it
// leaves.
func (s *Server) mountZoneEntered(p *Player, z *data.Zone) {
	if z.Type != "NoLandingZone" {
		return
	}
	if p.IsGM() {
		p.SendMessage("You have entered a no-landing zone. Dismount restrictions are ignored for GMs.")
		return
	}
	if p.mount.typ == serverpackets.MountWyvern {
		delay := mountDismountDelay
		if v, ok := z.Param("dismountDelay"); ok {
			if n, err := strconv.Atoi(v); err == nil {
				delay = n
			}
		}
		p.SendSystemMessage(mountMsgWyvernNoEntry)
		s.mountEnteredNoLanding(p, delay)
	}
}

// mountZoneExited is NoLandingZone.onExit: the forced landing is off.
func (s *Server) mountZoneExited(p *Player, z *data.Zone) {
	if z.Type == "NoLandingZone" && p.mount.typ == serverpackets.MountWyvern {
		s.mountCancelLanding(p)
	}
}

// mountSiegeZoneEntered is the mount part of SiegeZone.onEnter in an active
// siege (siege_zone.go), with AllowRideMountsDuringSiege off: a GM is told
// the restrictions do not hold for it; a wyvern that is not the castle
// lord's is warned and lands after 5 s; any other mount but the lord's
// strider is dismounted at once.
func (s *Server) mountSiegeZoneEntered(p *Player, zoneID int) {
	if mountAllowDuringSiege {
		return
	}
	if p.IsGM() {
		p.SendMessage("You have entered a siege zone. GM dismount restrictions are ignored.")
		return
	}
	lord := false
	for _, c := range s.residence.castles {
		if c.siegeZone != nil && c.siegeZone.ID == zoneID {
			lord = p.IsClanLeader() && p.ClanID() == c.OwnerID()
			break
		}
	}
	switch {
	case p.mount.typ == serverpackets.MountWyvern:
		if !lord {
			p.SendSystemMessage(mountMsgWyvernNoEntry)
			s.mountEnteredNoLanding(p, mountDismountDelay)
		}
	case s.mounted(p):
		if !(p.mount.typ == serverpackets.MountStrider && lord) {
			s.dismount(p)
		}
	}
}

// mountSiegeZoneExited is the mount part of SiegeZone.onExit and of the end
// of a siege: the forced landing of a wyvern is off.
func (s *Server) mountSiegeZoneExited(p *Player) {
	if p.mount.typ == serverpackets.MountWyvern {
		s.mountCancelLanding(p)
	}
}

// mountEnteredNoLanding is Player.enteredNoLanding: the rider is dismounted
// after delay seconds.
func (s *Server) mountEnteredNoLanding(p *Player, delay int) {
	s.mountCancelLanding(p)
	p.mount.landTimer = s.loop.After(time.Duration(delay)*time.Second, func() {
		p.mount.landTimer = 0
		if p.InGame() && s.mounted(p) {
			s.dismount(p)
		}
	})
}

// mountCancelLanding is Player.exitedNoLanding.
func (s *Server) mountCancelLanding(p *Player) {
	if m := &p.mount; m.landTimer != 0 {
		s.loop.Cancel(m.landTimer)
		m.landTimer = 0
	}
}
