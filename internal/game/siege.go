package game

import (
	"math"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/world"
)

// The castle siege engine on the loop (Mobius Siege): the start task that
// closes registration 24 hours before and starts the siege, the end task
// with its countdown, the towers, guards and headquarters, the mid victory
// when the Seal of Ruler is engraved, and the end with the reputation
// changes and the next date, saved at once with the clans it changed (X4).

// siegeHQNpc is the headquarters NPC (HeadquarterCreate.HQ_NPC_ID).
const siegeHQNpc = 35062

// castleSiege is a castle's siege (Mobius Siege). Loop-only.
type castleSiege struct {
	reg residence.Registry
	// stored are the registrations read at startup, until the start task
	// loads them into reg (loadSiegeClan).
	stored []residence.Registration
	loaded bool

	inProgress       bool
	registrationOver bool // Mobius _isRegistrationOver
	endDate          time.Time
	firstOwner       int32

	controlTowers, flameTowers []*Npc
	ctCount                    int
	guards                     []*siegeGuard

	startTimer, endTimer world.TimerID
}

// rows returns the registrations to store.
func (cs *castleSiege) rows() []residence.Registration {
	if !cs.loaded {
		return cs.stored
	}
	return cs.reg.Rows()
}

// siegeGuard is a siege guard spawn (a Mobius Spawn of SiegeGuardManager):
// the guard respawns after its delay until its control tower falls.
type siegeGuard struct {
	npcID   int
	pos     world.Position
	heading int32
	respawn time.Duration
	npc     *Npc
	tower   *Npc
	stopped bool
	timer   world.TimerID
}

// ----- schedule -----

// schedule returns the castle's SiegeSchedule.xml line.
func (c *Castle) schedule() residence.Schedule {
	d := c.Data
	return residence.Schedule{Enabled: d.SiegeEnabled, Day: d.SiegeWeekday(), Hour: d.SiegeHour, MaxConcurrent: d.MaxConcurrent}
}

// setNextSiegeDate is Mobius Siege.setNextSiegeDate.
func (s *Server) setNextSiegeDate(c *Castle) {
	concurrent := func(t time.Time) int {
		n := 0
		for _, o := range s.residence.castles {
			if o != c && o.Data.SiegeEnabled && !o.siegeDate.IsZero() && math.Abs(float64(o.siegeDate.Sub(t).Milliseconds())) < 1000 {
				n++
			}
		}
		return n
	}
	date, open, ok := residence.NextSiegeDate(s.calendar, s.Now(), c.siegeDate, c.schedule(), s.residence.cfg.cycle, concurrent)
	if !ok {
		return
	}
	c.siegeDate = date
	if open {
		s.broadcastSysMsg(residence.MsgAnnouncedSiegeTime, castleParam(c))
	}
	c.siege.registrationOver = !open
}

// startCastleAutoTask is Mobius Siege.startAutoTask: a past siege date moves
// to the next one (correctSiegeDateTime), the registrations load and the
// start task runs in a second.
func (s *Server) startCastleAutoTask(c *Castle) {
	if !c.Data.SiegeEnabled {
		return
	}
	if c.siegeDate.Before(s.Now()) {
		s.setNextSiegeDate(c)
		c.save()
	}
	s.log.Info("game: siege of "+c.Name, "date", c.siegeDate)
	if !c.siege.loaded {
		c.siege.reg.Load(c.ownerID, c.siege.stored)
		c.siege.stored, c.siege.loaded = nil, true
	}
	s.scheduleSiegeStart(c, time.Second)
}

// scheduleSiegeStart (re)arms the start task.
func (s *Server) scheduleSiegeStart(c *Castle, d time.Duration) {
	if c.siege.startTimer != 0 {
		s.loop.Cancel(c.siege.startTimer)
	}
	c.siege.startTimer = s.loop.After(d, func() { s.siegeStartTask(c) })
}

// siegeStartTask is Mobius ScheduleStartSiegeTask.run.
func (s *Server) siegeStartTask(c *Castle) {
	c.siege.startTimer = 0
	if c.siege.inProgress {
		return
	}
	st := residence.NextStartStep(s.Now(), c.siegeDate, c.regTimeEnd, c.regTimeOver)
	if st.EndTimeRegistration {
		c.regTimeOver = true
	}
	if st.CloseRegistration {
		s.broadcastSysMsg(residence.MsgRegistrationTermEnded, castleParam(c))
		c.siege.registrationOver = true
		c.siege.reg.ClearWaiting()
		c.save()
	}
	if st.Start {
		s.StartSiege(c)
		return
	}
	s.scheduleSiegeStart(c, st.Wait)
}

// ----- start -----

// StartSiege is Mobius Siege.startSiege: without attackers the siege is
// cancelled and the next one set; else the zone turns into a battlefield.
// Loop-only.
func (s *Server) StartSiege(c *Castle) {
	cs := &c.siege
	if cs.inProgress {
		return
	}
	if !cs.loaded {
		cs.reg.Load(c.ownerID, cs.stored)
		cs.stored, cs.loaded = nil, true
	}
	cs.firstOwner = c.ownerID
	if len(cs.reg.Attackers()) == 0 {
		if cs.firstOwner <= 0 {
			s.broadcastSysMsg(residence.MsgSiegeCanceledNoInterest, castleParam(c))
		} else {
			s.broadcastSysMsg(residence.MsgSiegeCanceledNoClans, castleParam(c))
		}
		s.saveCastleSiege(c)
		c.save()
		return
	}
	cs.inProgress = true
	s.residence.sieging = true
	cs.reg.Load(c.ownerID, cs.reg.Rows())
	s.updateSiegeStates(c, false)
	s.teleportSiegePlayers(c, teleportNotOwner, false)
	cs.ctCount = 0
	s.spawnTowers(c)
	c.spawnDoors(false)
	s.spawnSiegeGuards(c)
	s.deleteMercTickets(c) // castle_merc.go
	s.siegeZoneStatus(c)   // siege_zone.go
	cs.endDate = s.Now().Add(time.Duration(s.residence.cfg.length) * time.Minute)
	cs.endTimer = s.loop.After(time.Second, func() { s.siegeEndTask(c) })
	s.broadcastSysMsg(residence.MsgSiegeStarted, castleParam(c))
}

// siegeEndTask is Mobius ScheduleEndSiegeTask.run.
func (s *Server) siegeEndTask(c *Castle) {
	c.siege.endTimer = 0
	if !c.siege.inProgress {
		return
	}
	st := residence.NextEndStep(s.Now(), c.siege.endDate)
	if st.Msg != 0 {
		s.announceSiege(c, clanSysMsg(serverpackets.SystemMessageID(st.Msg), serverpackets.NumberParam(int32(st.Param))), true) //nolint:gosec // small
	}
	if st.End {
		s.EndSiege(c)
		return
	}
	c.siege.endTimer = s.loop.After(st.Wait, func() { s.siegeEndTask(c) })
}

// announceSiege is Siege.announceToPlayer: the defenders' members online,
// and the attackers' with bothSides.
func (s *Server) announceSiege(c *Castle, body []byte, bothSides bool) {
	send := func(list []*residence.SiegeClan) {
		for _, sc := range list {
			if cl := s.Clan(sc.ClanID); cl != nil {
				broadcastToClan(cl, body)
			}
		}
	}
	send(c.siege.reg.Defenders())
	if bothSides {
		send(c.siege.reg.Attackers())
	}
}

// ----- end -----

// EndSiege is Mobius Siege.endSiege: the result to everyone, the reputation
// changes, the battlefield cleared, the next date set, and the castle saved
// with every clan it changed in one transaction. Loop-only.
func (s *Server) EndSiege(c *Castle) {
	cs := &c.siege
	if !cs.inProgress {
		return
	}
	s.broadcastSysMsg(residence.MsgSiegeFinished, castleParam(c))
	owner := c.Owner()
	if owner != nil {
		s.broadcastSysMsg(residence.MsgVictoriousOverSiege, serverpackets.TextParam(owner.Name), serverpackets.TextParam(c.Name))
		if owner.ID != cs.firstOwner {
			c.ticketBuyCount = 0
			for _, p := range owner.OnlineMembers() {
				if p.IsNoble() {
					s.HeroCastleTaken(p, int(c.ID))
				}
			}
		}
	} else {
		s.broadcastSysMsg(residence.MsgSiegeDraw, castleParam(c))
	}
	changed := s.updateClansReputation(c)
	s.removeSiegeFlags(cs.reg.RemoveFlags())
	s.teleportSiegePlayers(c, teleportNotOwner, false)
	cs.inProgress = false
	s.updateSiegeStates(c, true)
	s.clearSiegeClans(c)
	s.saveCastleSiege(c)
	s.removeTowers(c)
	s.unspawnSiegeGuards(c)
	if c.ownerID > 0 {
		c.hired = nil
	}
	c.spawnDoors(false)
	c.firstMidVictory = false
	s.siegeZoneStatus(c)
	if cs.endTimer != 0 {
		s.loop.Cancel(cs.endTimer)
		cs.endTimer = 0
	}
	s.residence.sieging = s.anySiege()
	c.save(changed...)
}

// anySiege reports whether a siege is in progress somewhere.
func (s *Server) anySiege() bool {
	for _, c := range s.residence.castles {
		if c.siege.inProgress {
			return true
		}
	}
	return false
}

// updateClansReputation is Castle.updateClansReputation: the changes apply,
// the members are told, and the changed clans are returned for the save.
func (s *Server) updateClansReputation(c *Castle) []*Clan {
	former := c.formerOwner
	score := 0
	if cl := s.Clan(former); cl != nil {
		score = cl.Reputation
	} else {
		former = 0
	}
	var changed []*Clan
	for _, ch := range residence.SiegeReputation(former, score, c.ownerID, s.residence.cfg.points) {
		cl := s.Clan(ch.ClanID)
		if cl == nil {
			continue
		}
		s.setClanReputation(cl, ch.Delta)
		broadcastToClan(cl, clanSysMsg(serverpackets.SystemMessageID(ch.Msg), serverpackets.NumberParam(int32(ch.Param)))) //nolint:gosec // points
		changed = append(changed, cl)
	}
	return changed
}

// saveCastleSiege is Mobius saveCastleSiege: the next siege date, and a day
// for the lord to choose another time; the start task starts again.
func (s *Server) saveCastleSiege(c *Castle) {
	s.setNextSiegeDate(c)
	c.regTimeEnd = s.Now().Add(24 * time.Hour)
	c.regTimeOver = false
	s.scheduleSiegeStart(c, time.Second)
}

// clearSiegeClans is Siege.clearSiegeClan: every registration of the castle
// goes, and the owner's registrations for other castles.
func (s *Server) clearSiegeClans(c *Castle) {
	c.siege.reg.Clear()
	if c.ownerID <= 0 {
		return
	}
	for _, o := range s.residence.castles {
		if o == c || !o.siege.reg.Registered(c.ownerID) || o.siege.inProgress {
			continue
		}
		o.siege.reg.Remove(c.ownerID)
		o.save()
	}
}

// ----- mid victory -----

// midVictory is Mobius Siege.midVictory, after the castle changed hands.
func (s *Server) midVictory(c *Castle) {
	cs := &c.siege
	if !cs.inProgress {
		return
	}
	if c.ownerID > 0 {
		c.hired = nil
	}
	allyOf := func(id int32) int32 {
		if cl := s.Clan(id); cl != nil {
			return cl.AllyID
		}
		return 0
	}
	var allies []int32
	if a := allyOf(c.ownerID); a != 0 {
		for _, cl := range s.Clans() {
			if cl.AllyID == a {
				allies = append(allies, cl.ID)
			}
		}
	}
	if cs.reg.MidVictory(c.ownerID, allyOf, allies) {
		s.EndSiege(c)
		return
	}
	if c.ownerID <= 0 {
		return
	}
	c.firstMidVictory = true
	s.teleportSiegePlayers(c, teleportAttacker, true)
	s.teleportSiegePlayers(c, teleportSpectator, false)
	s.removeSiegeFlags(cs.reg.RemoveDefenderFlags())
	c.removeDoorUpgrades() // door_upgrade.go
	c.trapUpgrades = make(map[int32]int32)
	c.functions = make(map[int]residence.Function)
	c.spawnDoors(true)
	s.removeTowers(c)
	cs.ctCount = 0
	s.spawnTowers(c)
	s.updateSiegeStates(c, false)
	c.save()
}

// ----- players -----

// Who a siege teleport moves (Mobius SiegeTeleportWhoType).
const (
	teleportNotOwner = iota
	teleportAttacker
	teleportSpectator
)

// teleportSiegePlayers is Siege.teleportPlayer: who goes to town, or with
// toFlag to its clan's headquarters (TeleportWhereType.SIEGEFLAG). GMs stay.
func (s *Server) teleportSiegePlayers(c *Castle, who int, toFlag bool) {
	var list []*Player
	switch who {
	case teleportNotOwner:
		for _, p := range c.playersInZone() {
			if p.ClanID() == 0 || p.ClanID() != c.ownerID {
				list = append(list, p)
			}
		}
	case teleportAttacker:
		for _, sc := range c.siege.reg.Attackers() {
			if cl := s.Clan(sc.ClanID); cl != nil {
				for _, p := range cl.OnlineMembers() {
					if s.inSiege(p) {
						list = append(list, p)
					}
				}
			}
		}
	case teleportSpectator:
		for _, p := range c.playersInZone() {
			if !s.inSiege(p) {
				list = append(list, p)
			}
		}
	}
	for _, p := range list {
		if p.IsGM() {
			continue
		}
		if toFlag {
			if loc, ok := s.siegeFlagLocation(p); ok {
				s.teleportAround(p, loc, maxOffsetOnTeleport)
				continue
			}
		}
		s.teleportToTown(p)
	}
}

// teleportToTown is teleToLocation(TeleportWhereType.TOWN).
func (s *Server) teleportToTown(p *Player) {
	if loc, ok := s.restartLocation(p); ok {
		s.teleportAround(p, loc, maxOffsetOnTeleport)
	}
}

// updateSiegeStates is Siege.updatePlayerSiegeStateFlags: the members online
// of the attackers get siege state 1, of the defenders 2 (none with clear),
// with UserInfo and RelationChanged around them.
func (s *Server) updateSiegeStates(c *Castle, clear bool) {
	apply := func(list []*residence.SiegeClan, state int) {
		for _, sc := range list {
			cl := s.Clan(sc.ClanID)
			if cl == nil {
				continue
			}
			for _, p := range cl.OnlineMembers() {
				if clear {
					delete(s.residence.sides, p)
				} else {
					s.residence.sides[p] = playerSiege{state: state, side: c.ID}
				}
				p.UpdateUserInfo()
				s.sendRelations(p)
			}
		}
	}
	apply(c.siege.reg.Attackers(), 1)
	apply(c.siege.reg.Defenders(), 2)
}

// ----- towers -----

// spawnTowers is spawnControlTower and spawnFlameTower.
func (s *Server) spawnTowers(c *Castle) {
	cs := &c.siege
	for _, t := range c.Data.ControlTowers {
		if n := s.spawnSiegeNpc(t.NpcID, world.Position{X: int32(t.X), Y: int32(t.Y), Z: int32(t.Z)}, 0, ""); n != nil { //nolint:gosec // world coordinates
			cs.controlTowers = append(cs.controlTowers, n)
		}
	}
	cs.ctCount = len(cs.controlTowers)
	for _, t := range c.Data.FlameTowers {
		if n := s.spawnSiegeNpc(t.NpcID, world.Position{X: int32(t.X), Y: int32(t.Y), Z: int32(t.Z)}, 0, ""); n != nil { //nolint:gosec // world coordinates
			cs.flameTowers = append(cs.flameTowers, n)
		}
	}
}

// removeTowers deletes the towers.
func (s *Server) removeTowers(c *Castle) {
	for _, n := range append(c.siege.flameTowers, c.siege.controlTowers...) {
		if n.InWorld() || !n.Dead {
			s.DespawnNpc(n)
		}
	}
	c.siege.flameTowers, c.siege.controlTowers = nil, nil
}

// spawnSiegeNpc puts a siege NPC (tower, guard, headquarters) into the
// world, or logs why it cannot.
func (s *Server) spawnSiegeNpc(npcID int, pos world.Position, heading int32, title string) *Npc {
	t, err := s.spawnableTemplate(npcID)
	if err != nil {
		s.log.Warn("game: cannot spawn a siege NPC", "npc", npcID, "err", err)
		return nil
	}
	n, err := s.newNpc(t)
	if err != nil {
		s.log.Warn("game: cannot spawn a siege NPC", "npc", npcID, "err", err)
		return nil
	}
	n.Title = title
	pos.Heading = heading
	if err := s.placeNpc(n, pos, nil); err != nil {
		s.releaseNpcID(n)
		s.log.Warn("game: cannot spawn a siege NPC", "npc", npcID, "err", err)
		return nil
	}
	return n
}

// ----- guards -----

// spawnSiegeGuards is SiegeGuardManager.spawnSiegeGuard and the control
// tower registration of Siege.spawnSiegeGuard: an owned castle gets its
// hired mercenaries (up to its limit, no respawn), an unowned one the
// data's guards.
func (s *Server) spawnSiegeGuards(c *Castle) {
	cs := &c.siege
	if c.ownerID > 0 {
		for i, h := range c.hired {
			if i >= c.Data.MaxMercenaries {
				break
			}
			cs.guards = append(cs.guards, &siegeGuard{npcID: int(h.NpcID), pos: world.Position{X: h.X, Y: h.Y, Z: h.Z}, heading: h.Heading})
		}
	} else if d := s.residence.data; d != nil {
		for _, g := range d.GuardsOf(int(c.ID)) {
			cs.guards = append(cs.guards, &siegeGuard{npcID: g.NpcID, pos: world.Position{X: int32(g.X), Y: int32(g.Y), Z: int32(g.Z)}, //nolint:gosec // world coordinates
				heading: int32(g.Heading), respawn: time.Duration(g.RespawnDelay) * time.Second}) //nolint:gosec // heading
		}
	}
	for _, g := range cs.guards {
		g.npc = s.spawnSiegeNpc(g.npcID, g.pos, g.heading, "")
		best := math.MaxFloat64
		for _, ct := range cs.controlTowers {
			if d := ct.Position().Distance3D(g.pos); d < best {
				best, g.tower = d, ct
			}
		}
	}
}

// unspawnSiegeGuards deletes the guards and stops their respawns.
func (s *Server) unspawnSiegeGuards(c *Castle) {
	for _, g := range c.siege.guards {
		g.stopped = true
		if g.timer != 0 {
			s.loop.Cancel(g.timer)
		}
		if g.npc != nil && g.npc.InWorld() {
			s.DespawnNpc(g.npc)
		}
	}
	c.siege.guards = nil
}

// ----- deaths -----

// residenceDie is the OnDie hook: a control tower that falls stops its
// guards' respawns (ControlTower.doDie), a headquarters leaves its clan
// (SiegeFlag.doDie), a guard respawns after its delay.
func (s *Server) residenceDie(victim, _ Creature) {
	n, ok := victim.(*Npc)
	if !ok {
		return
	}
	if s.hallSiegeDie(n) { // hallsiege.go
		return
	}
	for _, c := range s.residence.castles {
		cs := &c.siege
		if !cs.inProgress {
			continue
		}
		for _, ct := range cs.controlTowers {
			if ct == n {
				cs.ctCount = max(cs.ctCount-1, 0)
				for _, g := range cs.guards {
					if g.tower == n {
						g.stopped = true
					}
				}
				return
			}
		}
		if sc := cs.reg.FlagOwner(int32(n.ID())); sc != nil {
			sc.RemoveFlag(int32(n.ID()))
			return
		}
		for _, g := range cs.guards {
			if g.npc == n {
				s.guardDied(c, g)
				return
			}
		}
	}
}

// guardDied schedules a guard's respawn.
func (s *Server) guardDied(c *Castle, g *siegeGuard) {
	if g.stopped || g.respawn <= 0 {
		return
	}
	g.timer = s.loop.After(g.respawn, func() {
		g.timer = 0
		if g.stopped || !c.siege.inProgress {
			return
		}
		if g.npc != nil && g.npc.InWorld() {
			s.DespawnNpc(g.npc)
		}
		g.npc = s.spawnSiegeNpc(g.npcID, g.pos, g.heading, "")
	})
}

// ----- headquarters -----

// removeSiegeFlags deletes headquarters NPCs.
func (s *Server) removeSiegeFlags(ids []int32) {
	for _, id := range ids {
		if n := s.Npc(world.ObjectID(id)); n != nil {
			s.DespawnNpc(n)
		}
	}
}

// siegeFlagLocation is getTeleToLocation(SIEGEFLAG): a headquarters of the
// player's clan at the siege it stands in.
func (s *Server) siegeFlagLocation(p *Player) (data.Location, bool) {
	c := s.CastleAt(p.Position())
	if c == nil || !c.siege.inProgress || p.ClanID() == 0 {
		return data.Location{}, false
	}
	sc := c.siege.reg.Attacker(p.ClanID())
	if sc == nil || len(sc.Flags) == 0 {
		return data.Location{}, false
	}
	n := s.Npc(world.ObjectID(sc.Flags[0]))
	if n == nil {
		return data.Location{}, false
	}
	pos := n.Position()
	return data.Location{X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z)}, true
}

// buildHeadquarters is HeadquarterCreate.onStart: a clan leader's
// headquarters, titled with the clan's name, 50 above the leader.
func (s *Server) buildHeadquarters(p *Player) {
	if !p.IsClanLeader() {
		return
	}
	c := s.CastleAt(p.Position())
	if c == nil || !c.siege.inProgress {
		return
	}
	sc := c.siege.reg.Attacker(p.ClanID())
	if sc == nil {
		return
	}
	pos := p.Position()
	pos.Z += 50
	n := s.spawnSiegeNpc(siegeHQNpc, pos, pos.Heading, p.Clan().Name)
	if n != nil {
		sc.Flags = append(sc.Flags, int32(n.ID()))
	}
}
