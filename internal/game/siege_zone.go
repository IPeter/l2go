package game

import (
	"strconv"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/residence"
)

// Siege zones and the players in a siege: Mobius SiegeZone (PvP, siege and
// no-summon-friend while the siege runs), the siege bits of
// Player.getRelation and isAutoAttackable, the Die options and the restart
// points of RequestRestartPoint and MapRegionData.getTeleToLocation.

// siegeZoneActive reports whether a siege zone is a battlefield now.
func (s *Server) siegeZoneActive(z *data.Zone) bool {
	for _, c := range s.residence.castles {
		if c.siegeZone == z {
			return c.siege.inProgress
		}
	}
	if hs := s.hallSiegeByZone(z.ID); hs != nil { // hallsiege.go
		return hs.zoneActive
	}
	return false
}

// residenceZoneEntered is SiegeZone.onEnter (zone.go enterZone): inside an
// active siege zone a player is in a PvP and siege zone.
func (s *Server) residenceZoneEntered(p *Player, z *data.Zone) {
	if z.Type != "SiegeZone" || !s.siegeZoneActive(z) {
		return
	}
	s.applySiegeZone(p, z.ID)
}

// applySiegeZone sets the zone flags of an active siege zone once.
func (s *Server) applySiegeZone(p *Player, zoneID int) {
	f := &s.residence
	m := f.zoneFlags[p]
	if m[zoneID] {
		return
	}
	if m == nil {
		m = make(map[int]bool)
		f.zoneFlags[p] = m
	}
	m[zoneID] = true
	p.Zones.Enter(ZonePvP)
	p.Zones.Enter(ZoneSiege)
	p.Zones.Enter(ZoneNoSummonFriend)
	p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgEnteredCombatZone))
	s.mountSiegeZoneEntered(p, zoneID) // mount.go
}

// residenceZoneExited is SiegeZone.onExit: the flags go, and leaving an
// active battlefield starts the PvP flag.
func (s *Server) residenceZoneExited(p *Player, zoneID int) {
	if !s.leaveSiegeZone(p, zoneID) {
		return
	}
	p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgLeftCombatZone))
	if p.PvPFlag == 0 {
		s.startPvPFlag(p)
	}
}

// leaveSiegeZone takes a siege zone's flags off; false if it had none.
func (s *Server) leaveSiegeZone(p *Player, zoneID int) bool {
	m := s.residence.zoneFlags[p]
	if !m[zoneID] {
		return false
	}
	delete(m, zoneID)
	if len(m) == 0 {
		delete(s.residence.zoneFlags, p)
	}
	p.Zones.Leave(ZonePvP)
	p.Zones.Leave(ZoneSiege)
	p.Zones.Leave(ZoneNoSummonFriend)
	s.mountSiegeZoneExited(p) // mount.go
	return true
}

// siegeZoneStatus is SiegeZone.updateZoneStatusForCharactersInside after the
// siege started or ended.
func (s *Server) siegeZoneStatus(c *Castle) {
	z := c.siegeZone
	if z == nil {
		return
	}
	for _, p := range c.playersInZone() {
		if !p.Zones.In(z.ID) {
			continue
		}
		if c.siege.inProgress {
			s.applySiegeZone(p, z.ID)
		} else if s.leaveSiegeZone(p, z.ID) {
			p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgLeftCombatZone))
		}
		p.UpdateUserInfo()
		s.sendRelations(p)
	}
}

// inSiege is Player.isInSiege: registered on a siege in progress and inside
// its zone.
func (s *Server) inSiege(p *Player) bool {
	st, ok := s.residence.sides[p]
	if !ok {
		return false
	}
	sg := s.siegableByID(st.side)
	return sg != nil && sg.InProgress() && sg.Contains(p.Position())
}

// siegeState returns p's siege state: 1 attacker, 2 defender, 0 none.
func (s *Server) siegeState(p *Player) int {
	if s == nil {
		return 0
	}
	return s.residence.sides[p].state
}

// residencePlayerEntered gives a player who enters during a siege its siege
// state, as Mobius EnterWorld does for the clan of a siege in progress.
func (s *Server) residencePlayerEntered(p *Player) {
	id := p.ClanID()
	if id == 0 {
		return
	}
	for _, c := range s.residence.castles {
		if !c.siege.inProgress {
			continue
		}
		switch {
		case c.siege.reg.IsAttacker(id):
			s.residence.sides[p] = playerSiege{state: 1, side: c.ID}
		case c.siege.reg.IsDefender(id):
			s.residence.sides[p] = playerSiege{state: 2, side: c.ID}
		}
	}
}

// residencePlayerLeft forgets a player's siege state; a player who logs out
// inside a siege zone of a castle its clan does not own goes to town first
// (SiegeZone.onPlayerLogoutInside).
func (s *Server) residencePlayerLeft(p *Player) {
	delete(s.residence.sides, p)
	delete(s.residence.zoneFlags, p)
}

// siegeRelationBits are the siege bits of Player.getRelation.
func (s *Server) siegeRelationBits(p, v *Player) int32 {
	st := s.siegeState(p) // 0 for a nil s
	if st == 0 {
		return 0
	}
	var r int32 = serverpackets.RelationInSiege
	if st != s.siegeState(v) {
		r |= serverpackets.RelationEnemy
	} else {
		r |= serverpackets.RelationAlly
	}
	if st == 1 {
		r |= serverpackets.RelationAttacker
	}
	return r
}

// siegeAutoAttackable is the siege part of Player.isAutoAttackable for a
// player attacker v of p: decided is false when the siege rules say
// nothing.
func (s *Server) siegeAutoAttackable(p, v *Player) (attackable, decided bool) {
	pc, vc := p.Clan(), v.Clan()
	if s == nil || pc == nil || vc == nil || pc == vc {
		return false, false
	}
	sg := s.siegeAt(p.Position())
	if sg == nil {
		return false, false
	}
	reg := sg.Clans()
	if reg.IsDefender(vc.ID) && reg.IsDefender(pc.ID) {
		return false, true
	}
	if reg.IsAttacker(vc.ID) && reg.IsAttacker(pc.ID) {
		c, ok := sg.(*Castle)
		return ok && s.CastleByID(s.residence.sides[p].side) == c && c.firstMidVictory, true
	}
	return false, false
}

// siegeNpcAutoAttackable is isAutoAttackable of the siege NPCs for a player:
// a Defender by anyone but the defenders of its siege (Defender), a control
// or flame tower by the attackers (Tower), a headquarters always
// (SiegeFlag). decided is false for other NPCs.
func (s *Server) siegeNpcAutoAttackable(n *Npc, p *Player) (attackable, decided bool) {
	switch {
	case s.npcIs(n, "Defender"):
		sg := s.siegeAt(n.Position())
		if sg == nil || p == nil {
			return false, true
		}
		st := s.residence.sides[p]
		return !(st.state == 2 && st.side == sg.ResidenceID()), true
	case s.npcIs(n, "ControlTower"), s.npcIs(n, "FlameTower"):
		c := s.CastleAt(n.Position())
		return c != nil && c.siege.inProgress && p != nil && c.siege.reg.IsAttacker(p.ClanID()), true
	case s.npcIs(n, "SiegeFlag"):
		return true, true
	}
	return false, false
}

// siegeDoorAttackable is the castle part of Door.isAutoAttackable: during
// the castle's siege, by anyone but the owner clan.
func (s *Server) siegeDoorAttackable(d *Door, p *Player) bool {
	c := s.CastleAt(d.Position())
	return c != nil && c.siege.inProgress && (p == nil || p.ClanID() == 0 || p.ClanID() != c.ownerID)
}

// residenceDieOptions are the clan options of Die (Mobius Die.writeImpl):
// to castle for an owner or a defender of a siege in progress, to the
// headquarters for an attacker with one.
func (s *Server) residenceDieOptions(p *Player, d *serverpackets.Die) {
	cl := p.Clan()
	if cl == nil {
		return
	}
	defense := false
	var sc *residence.SiegeClan
	if c := s.CastleAt(p.Position()); c != nil && c.siege.inProgress {
		sc = c.siege.reg.Attacker(cl.ID)
		defense = sc == nil && c.siege.reg.IsDefender(cl.ID)
	}
	d.ToClanHall = d.ToClanHall || cl.HideoutID > 0
	d.ToCastle = cl.CastleID > 0 || defense
	d.ToSiegeHQ = sc != nil && !defense && len(sc.Flags) > 0
}

// residenceRestart handles the castle and headquarters choices of
// RequestRestartPoint; it reports whether it handled the request. An
// attacker who died on its battlefield waits AttackerRespawn first.
func (s *Server) residenceRestart(p *Player, point int32) bool {
	switch point {
	case clientpackets.RestartPointCastle:
		loc, ok := s.castleRestartLocation(p)
		if !ok {
			return true
		}
		s.restartAt(p, loc)
		return true
	case clientpackets.RestartPointSiegeHQ:
		loc, ok := s.siegeFlagLocation(p)
		if !ok {
			s.log.Warn("game: RequestRestartPoint to a siege headquarters without one", p.logArgs()...)
			return true
		}
		s.restartAt(p, loc)
		return true
	}
	return false
}

// restartAt revives p and teleports it near loc, after the attacker's delay
// on a battlefield.
func (s *Server) restartAt(p *Player, loc data.Location) {
	port := func() {
		if !p.dead || !p.InWorld() {
			return
		}
		s.RevivePlayer(p)
		s.teleportAround(p, loc, maxOffsetOnTeleport)
	}
	if c := s.CastleAt(p.Position()); c != nil && c.siege.inProgress && c.siege.reg.IsAttacker(p.ClanID()) && s.residence.cfg.attackerRespawn > 0 {
		p.SendMessage("You will be re-spawned in " + strconv.Itoa(int(s.residence.cfg.attackerRespawn.Seconds())) + " seconds")
		s.loop.After(s.residence.cfg.attackerRespawn, port)
		return
	}
	port()
}

// castleRestartLocation is the "to castle" choice: a defender of a siege in
// progress, or the owner's members otherwise, go to the castle zone's spawn
// (chaotic with karma); an attacker to town.
func (s *Server) castleRestartLocation(p *Player) (data.Location, bool) {
	cl := p.Clan()
	c := s.CastleAt(p.Position())
	if c != nil && c.siege.inProgress {
		switch {
		case cl != nil && c.siege.reg.IsDefender(cl.ID):
		case cl != nil && c.siege.reg.IsAttacker(cl.ID):
			return s.restartLocation(p)
		default:
			s.log.Warn("game: RequestRestartPoint to a castle without one", p.logArgs()...)
			return data.Location{}, false
		}
	} else if cl == nil || cl.CastleID == 0 {
		return data.Location{}, false
	}
	target := s.CastleByOwner(cl)
	if target == nil {
		target = c
	}
	if target == nil {
		return s.restartLocation(p)
	}
	kind := data.ZoneSpawnNormal
	if p.Karma() > 0 {
		kind = data.ZoneSpawnChaotic
	}
	if loc, ok := target.spawnLoc(kind); ok {
		return loc, true
	}
	return s.restartLocation(p)
}
