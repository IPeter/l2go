package game

import (
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The GM command feature of Milestone 3 (decision E8): //teleport (Mobius
// //move_to), //loc, //gmspeed, //heal, //invis, //vis and //online, for
// access level 1 and up. Its state is in movementFeature.
func init() { registerFeature("gm", setupGM) }

const (
	// gmSysMsgInvalidTarget is "Invalid target." (Mobius INVALID_TARGET).
	gmSysMsgInvalidTarget serverpackets.SystemMessageID = 109
	// gmMaxSpeed is the highest //gmspeed (Mobius's own limit).
	gmMaxSpeed = 10
)

// setupGM registers the GM commands, with Mobius's aliases.
func setupGM(s *Server) error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("teleport", admin, s.gmTeleport("teleport")),
		s.RegisterGMCommand("move_to", admin, s.gmTeleport("move_to")),
		s.RegisterGMCommand("loc", admin, s.gmLoc),
		s.RegisterGMCommand("gmspeed", admin, s.gmSpeed),
		s.RegisterGMCommand("heal", admin, s.gmHeal),
		s.RegisterGMCommand("invis", admin, s.gmInvisible),
		s.RegisterGMCommand("invisible", admin, s.gmInvisible),
		s.RegisterGMCommand("vis", admin, s.gmVisible),
		s.RegisterGMCommand("visible", admin, s.gmVisible),
		s.RegisterGMCommand("online", admin, s.gmOnline),
	)
}

// gmTeleport is //teleport x y z, and //move_to x y z, its Mobius name
// (AdminTeleport.teleportTo): the GM teleports there, keeping its heading,
// and is told "You have been teleported to x y z".
func (s *Server) gmTeleport(name string) GMCommand {
	return func(p *Player, args []string) {
		if len(args) < 3 {
			p.SendMessage("Wrong or no Coordinates given.")
			return
		}
		var c [3]int32
		for i := range c {
			v, err := strconv.ParseInt(args[i], 10, 32)
			if err != nil {
				p.SendMessage("Usage: //" + name + " <x> <y> <z>")
				return
			}
			c[i] = int32(v)
		}
		if err := s.Teleport(p, world.Position{X: c[0], Y: c[1], Z: c[2]}); err != nil {
			s.log.Info("game: GM teleport refused", append(p.logArgs(), "err", err)...)
			if errors.Is(err, world.ErrOutOfWorld) {
				p.SendMessage("Those coordinates are outside the world.")
			}
			return
		}
		p.SendMessage("You have been teleported to " + strings.Join(args, " "))
	}
}

// gmLoc is //loc: the GM's position and heading. (Mobius has no such admin
// command; the /loc user command names the map region.)
func (s *Server) gmLoc(p *Player, _ []string) {
	pos := p.Position()
	p.SendMessage(fmt.Sprintf("Current location: %d, %d, %d (heading %d)", pos.X, pos.Y, pos.Z, pos.Heading))
}

// gmSpeed is //gmspeed n (Mobius AdminGmSpeed): the GM's target creature,
// or the GM, moves n times as fast, 0 to 10 (0 is normal speed again). A
// player's UserInfo and CharInfo, or an NPC's info, go out again.
func (s *Server) gmSpeed(p *Player, args []string) {
	const usage = "//gmspeed [0...10]"
	if len(args) == 0 {
		p.SendMessage(usage)
		return
	}
	v, err := strconv.ParseFloat(args[0], 64)
	if err != nil || math.IsNaN(v) || v < 0 || v > gmMaxSpeed {
		p.SendMessage(usage)
		return
	}
	mult := v
	if !(v > 0) {
		mult = 1
	}
	name := p.Name()
	switch t := p.Target.(type) {
	case *Npc:
		name = t.Name()
		if t.Stats != nil {
			t.Stats.SetGMSpeed(mult)
		}
		t.BroadcastInfo()
	case *Player:
		name = t.Name()
		t.stats.SetGMSpeed(mult)
		t.BroadcastUserInfo()
	default:
		p.stats.SetGMSpeed(mult)
		p.BroadcastUserInfo()
	}
	// Java appends "0" to the double: 2 shows as "[200.00]%".
	p.SendMessage("[" + name + "] speed is [" + gmJavaDouble(v*100) + "0]% fast.")
}

// gmJavaDouble formats v as Java's Double.toString does for the numbers
// //gmspeed prints: the shortest decimal that reads back as v, with at
// least one digit after the point.
func gmJavaDouble(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

// gmHeal is //heal (Mobius AdminHeal): the GM's target, or the GM, gets full
// HP and MP, and a player full CP. //heal <name> heals that player, and
// //heal <radius> every creature the GM sees: Mobius ignores the radius
// itself and leaves the GM out. A target that is no creature gets "Invalid
// target."
func (s *Server) gmHeal(p *Player, args []string) {
	target := p.Target
	if len(args) > 0 {
		arg := strings.Join(args, " ")
		if other, ok := s.PlayerByName(arg); ok {
			target = other
		} else if radius, err := strconv.ParseInt(arg, 10, 32); err == nil {
			var seen []world.Object
			for o := range s.world.Visible(p) {
				seen = append(seen, o)
			}
			for _, o := range seen {
				s.healCreature(o)
			}
			p.SendMessage(fmt.Sprintf("Healed within %d unit radius.", radius))
			return
		}
	}
	if target == nil {
		target = p
	}
	if !s.healCreature(target) {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
	}
}

// healCreature gives o full HP and MP, and a player full CP too, and reports
// false when o is no creature. A player gets its UserInfo and the
// StatusUpdate of its HP, MP and CP (Mobius updateUserInfo and
// broadcastStatusUpdate), and the players targeting it its HP bar; an NPC's
// HP bar goes to the players targeting it.
func (s *Server) healCreature(o world.Object) bool {
	switch c := o.(type) {
	case *Player:
		v := c.stats.View()
		c.CurHP, c.CurMP, c.CurCP = float64(v.MaxHP), float64(v.MaxMP), float64(v.MaxCP)
		c.StopRegen() // all full, as the setters would leave it
		c.sendUserInfo()
		c.Send(vitalsStatusUpdate(c))
		s.BroadcastHP(c)
	case *Npc:
		if c.Stats != nil {
			c.CurMP = float64(c.Stats.MaxMP())
		}
		c.SetCurrentHP(float64(c.MaxHP()))
	default:
		return false
	}
	return true
}

// gmInvisible is //invis and //invisible (Mobius AdminEffects): the GM
// becomes invisible to players below GM level.
func (s *Server) gmInvisible(p *Player, _ []string) {
	s.setInvisible(p, true)
	p.SendMessage("Now, you cannot be seen.")
}

// gmVisible is //vis and //visible: the GM becomes visible again.
func (s *Server) gmVisible(p *Player, _ []string) {
	s.setInvisible(p, false)
	p.SendMessage("Now, you can be seen.")
}

// setInvisible changes p's invisibility the way Mobius setInvisible and
// AdminEffects do. The players around that no longer see p get DeleteObject
// and forget p (the OnForget hooks run, which drop p as their target); the
// players that see p get its info again (CharInfo, RelationChanged and its
// walk, with the "Invisible" title for GMs); and p gets its UserInfo while
// they get its CharInfo at the end of the tick.
func (s *Server) setInvisible(p *Player, invisible bool) {
	p.SetInvisible(invisible)
	if p.InWorld() {
		var viewers []*Player
		for v := range s.world.VisiblePlayers(p) {
			if vp, ok := v.(*Player); ok {
				viewers = append(viewers, vp)
			}
		}
		var del []byte
		for _, v := range viewers {
			if s.world.CanSee(v, p) {
				for _, body := range p.InfoFor(v) {
					v.Send(body)
				}
				continue
			}
			if del == nil {
				del = serverpackets.NewDeleteObjectPacket(p.ObjectID())
			}
			v.Send(del)
			forgetRelation(v, p)
			s.runForget(v, p)
		}
	}
	p.BroadcastUserInfo()
}

// gmOnline is //online (Mobius AdminOnline): a report of the players in the
// game. Offline traders, instances and combat do not exist yet, so most of
// those lines count nobody.
func (s *Server) gmOnline(p *Player, _ []string) {
	ips := make(map[string]bool)
	var total, online, peace, instanced, combat int
	for v := range s.Players() {
		total++
		if sess := v.session; sess != nil {
			online++
			ips[gmRemoteHost(sess.RemoteAddr())] = true
		}
		if v.Zones.Inside(ZonePeace) {
			peace++
		}
		if v.InstanceID() != 0 {
			instanced++
		}
		if v.PvPFlag > 0 || v.Zones.Inside(ZonePvP) || v.Zones.Inside(ZoneSiege) {
			combat++
		}
	}
	for _, line := range []string{
		"Online Player Report",
		"Total count: " + strconv.Itoa(total),
		"Total online: " + strconv.Itoa(online),
		"Total offline: 0",
		"Max connected: " + strconv.Itoa(max(s.movement.maxOnline, total)),
		"Unique IPs: " + strconv.Itoa(len(ips)),
		"In peace zone: " + strconv.Itoa(peace),
		"Not in peace zone: " + strconv.Itoa(total-peace),
		"In instances: " + strconv.Itoa(instanced),
		"In combat: " + strconv.Itoa(combat),
	} {
		p.SendMessage(line)
	}
}

// gmRemoteHost is the host part of a session's remote address.
func gmRemoteHost(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}
