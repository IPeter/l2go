package game

import (
	"context"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The contestable clan hall sieges on the loop (Mobius ClanHallSiegeEngine
// and SiegableHall), on the siege machinery of the castles (Siegable,
// siege_zone.go): an hour before the siege the registration closes and the
// owner becomes an attacker (PrepareOwner); then the doors close, the
// guards of clanhall_siege_guards stand, the siege zone turns into a
// battlefield and the attackers fight for an hour, or until the hall's
// script ends the siege. Each hall has a Lua script (scripts/conquerablehalls/)
// that decides the winner and may replace a step (scripting: on_hall_siege).
// The siege's end, with the new owner, is saved at once.

// Hall siege script events (scripting.HallSiegeEvent names).
const (
	hallEventPrepareOwner = "prepare_owner"
	hallEventStartSiege   = "start_siege"
	hallEventSiegeStarts  = "siege_starts"
	hallEventSiegeEnds    = "siege_ends"
	hallEventEndSiege     = "end_siege"
	hallEventWinner       = "winner"
)

// fortressOfResistance has no attacker list: its siege starts without
// attackers (ClanHallSiegeEngine.startSiege).
const fortressOfResistance = 21

// hallSiege is a contestable hall's siege (Mobius ClanHallSiegeEngine with
// SiegableHall's siege fields). Loop-only.
type hallSiege struct {
	h         *ClanHall
	status    residence.HallSiegeStatus
	nextSiege time.Time
	reg       residence.Registry // attackers only
	// vars are the values the hall's script keeps, saved with the hall.
	vars map[string]string

	task                world.TimerID // Mobius _siegeTask
	missionAccomplished bool
	zoneActive          bool

	guards []*siegeGuard
	// spawns are the NPCs the script placed for the siege (hall:spawn),
	// removed at its end.
	spawns []*siegeGuard
	// teleTasks are the running ResidenceHallTeleportZone tasks by zone.
	teleTasks map[int]world.TimerID
}

var _ Siegable = (*hallSiege)(nil)

func newHallSiege(h *ClanHall) *hallSiege {
	return &hallSiege{h: h, teleTasks: make(map[int]world.TimerID)}
}

// restore takes the stored siege state.
func (hs *hallSiege) restore(st storage.ClanHall) {
	hs.nextSiege = msTime(st.NextSiege)
	for _, id := range st.Attackers {
		hs.reg.AddAttacker(id)
	}
	hs.vars = hallVarsClone(st.Vars)
}

// document adds the siege state to the hall's stored state.
func (hs *hallSiege) document(st *storage.ClanHall) {
	st.NextSiege = timeMs(hs.nextSiege)
	for _, sc := range hs.reg.Attackers() {
		st.Attackers = append(st.Attackers, sc.ClanID)
	}
	st.Vars = hallVarsClone(hs.vars)
}

// ResidenceID implements Siegable.
func (hs *hallSiege) ResidenceID() int32 { return hs.h.ID }

// InProgress implements Siegable: the siege runs (SiegableHall.isInSiege).
func (hs *hallSiege) InProgress() bool { return hs.status == residence.HallRunning }

// Clans implements Siegable.
func (hs *hallSiege) Clans() *residence.Registry { return &hs.reg }

// Contains implements Siegable: pos is in the hall's siege zone.
func (hs *hallSiege) Contains(pos world.Position) bool {
	z := hs.h.siegeZone
	return z != nil && z.Shape.Contains(int(pos.X), int(pos.Y), int(pos.Z))
}

// hallSiegeByZone returns the hall siege whose siege zone is z, or nil.
func (s *Server) hallSiegeByZone(zoneID int) *hallSiege {
	for _, h := range s.halls.halls {
		if h.siege != nil && h.siegeZone != nil && h.siegeZone.ID == zoneID {
			return h.siege
		}
	}
	return nil
}

// ----- schedule -----

// startHallSiege is the SiegableHall constructor with the engine's: a date
// less than an hour away moves to the next one, and PrepareOwner is armed
// an hour before the siege. The hall is registered as a Siegable.
func (s *Server) startHallSiege(h *ClanHall, b *storage.Batch) {
	hs := h.siege
	s.registerSiegable(hs)
	if hs.nextSiege.IsZero() || residence.HallSiegeDue(s.Now(), hs.nextSiege) {
		hs.nextSiege = residence.NextHallSiege(s.calendar, s.Now(), h.Data.Schedule)
		b.ClanHalls = append(b.ClanHalls, h.document())
	}
	s.scheduleHallPrepare(h)
	s.log.Info("game: siege of "+h.Name, "date", hs.nextSiege)
}

// scheduleHallTask (re)arms the siege task (Mobius _siegeTask).
func (s *Server) scheduleHallTask(h *ClanHall, d time.Duration, fn func()) {
	hs := h.siege
	if hs.task != 0 {
		s.loop.Cancel(hs.task)
	}
	hs.task = s.loop.After(max(d, 0), func() {
		hs.task = 0
		fn()
	})
}

// cancelHallTask is cancelSiegeTask.
func (s *Server) cancelHallTask(h *ClanHall) {
	if hs := h.siege; hs.task != 0 {
		s.loop.Cancel(hs.task)
		hs.task = 0
	}
}

// scheduleHallPrepare arms PrepareOwner an hour before the next siege.
func (s *Server) scheduleHallPrepare(h *ClanHall) {
	d := h.siege.nextSiege.Sub(s.Now()) - residence.HallPrepareBefore
	s.scheduleHallTask(h, d, func() { s.hallPrepareOwner(h) })
}

// updateHallNextSiege is SiegableHall.updateNextSiege (not saved).
func (s *Server) updateHallNextSiege(h *ClanHall) {
	h.siege.nextSiege = residence.NextHallSiege(s.calendar, s.Now(), h.Data.Schedule)
}

// ----- steps -----

// hallPrepareOwner is ClanHallSiegeEngine.prepareOwner, unless the hall's
// script replaces it: the owner becomes an attacker and loses the hall,
// strangers leave it, and the registration closes.
func (s *Server) hallPrepareOwner(h *ClanHall) {
	if s.hallScriptReplaces(h, hallEventPrepareOwner) {
		return
	}
	hs := h.siege
	var changed []*Clan
	if h.ownerID > 0 {
		if !hs.reg.IsAttacker(h.ownerID) {
			hs.reg.AddAttacker(h.ownerID)
		}
		if old := s.freeHallState(h); old != nil {
			changed = append(changed, old)
		}
	}
	s.banishForeigners(h)
	s.closeHallRegistration(h)
	h.save(changed...)
}

// closeHallRegistration is the end of prepareOwner: "The registration term
// for <hall> has ended." to everyone, the status waiting for the battle,
// and the siege an hour later.
func (s *Server) closeHallRegistration(h *ClanHall) {
	s.broadcastSysMsg(residence.MsgRegistrationTermEnded, hallParam(h))
	h.siege.status = residence.HallWaitingBattle
	s.scheduleHallTask(h, residence.HallPrepareBefore, func() { s.hallStartSiege(h) })
}

// hallStartSiege is ClanHallSiegeEngine.startSiege, unless the hall's
// script replaces it.
func (s *Server) hallStartSiege(h *ClanHall) {
	if s.hallScriptReplaces(h, hallEventStartSiege) {
		return
	}
	s.hallStartSiegeDefault(h)
}

// hallStartSiegeDefault is the engine's startSiege: without attackers (but
// at Fortress of Resistance) the siege is cancelled; else the doors close,
// the guards stand, the zone turns into a battlefield, the attackers get
// their siege state, and the siege ends after its length.
func (s *Server) hallStartSiegeDefault(h *ClanHall) {
	hs := h.siege
	if len(hs.reg.Attackers()) == 0 && h.ID != fortressOfResistance {
		s.hallScriptEvent(h, hallEventSiegeEnds)
		s.cancelHallSiege(h)
		return
	}
	s.spawnHallDoors(h, false)
	s.spawnHallGuards(h)
	s.setHallZoneActive(h, true)
	hs.status = residence.HallRunning
	s.residence.sieging = true
	s.updateHallSiegeStates(h, false)
	s.hallScriptEvent(h, hallEventSiegeStarts)
	s.scheduleHallTask(h, time.Duration(h.Data.SiegeLength)*time.Millisecond, func() { s.hallEndSiege(h) })
}

// cancelHallSiege is the "lack of interest" branch of startSiege: the
// attackers are forgotten, the next date set and announced. Deviation:
// PrepareOwner is armed for the next siege and the registration opens
// (Mobius passes the absolute date as the delay and stays waiting).
func (s *Server) cancelHallSiege(h *ClanHall) {
	hs := h.siege
	hs.reg.Clear()
	s.updateHallNextSiege(h)
	hs.status = residence.HallRegistering
	s.scheduleHallPrepare(h)
	s.broadcastSysMsg(residence.MsgSiegeCanceledNoInterest, hallParam(h))
	h.save()
}

// hallEndSiege is ClanHallSiegeEngine.endSiege (with the script's end
// step first): the result to everyone, the winner the new owner when the
// mission was accomplished, the battlefield cleared, the next date set; the
// hall is saved with the clans it changed.
func (s *Server) hallEndSiege(h *ClanHall) {
	hs := h.siege
	s.cancelHallTask(h)
	s.hallScriptEvent(h, hallEventEndSiege)
	s.broadcastSysMsg(residence.MsgSiegeFinished, hallParam(h))
	var changed []*Clan
	winner := s.Clan(s.hallScriptWinner(h))
	if hs.missionAccomplished && winner != nil {
		if old := h.Owner(); old != nil && old != winner {
			if o := s.freeHallState(h); o != nil {
				changed = append(changed, o)
			}
		}
		if prev := s.ClanHallByOwner(winner); prev != nil && prev != h {
			s.freeHallState(prev)
			prev.save()
		}
		h.ownerID = winner.ID
		winner.HideoutID = h.ID
		changed = append(changed, winner)
		s.broadcastClanInfo(winner)
		s.broadcastSysMsg(residence.MsgVictoriousOverSiege, serverpackets.TextParam(winner.Name), hallParam(h))
	} else {
		s.broadcastSysMsg(residence.MsgSiegeDraw, hallParam(h))
	}
	hs.missionAccomplished = false
	s.setHallZoneActive(h, false)
	s.updateHallNextSiege(h)
	s.spawnHallDoors(h, false)
	s.banishForeigners(h)
	s.updateHallSiegeStates(h, true)
	for _, p := range s.hallSiegeZonePlayers(h) {
		s.startPvPFlag(p)
	}
	hs.reg.Clear()
	s.hallScriptEvent(h, hallEventSiegeEnds)
	s.scheduleHallPrepare(h)
	hs.status = residence.HallRegistering
	s.unspawnHallGuards(h)
	s.residence.sieging = s.anySiege() || s.anyHallSiege()
	h.save(changed...)
}

// anyHallSiege reports whether a hall siege is running.
func (s *Server) anyHallSiege() bool {
	for _, h := range s.halls.halls {
		if h.siege != nil && h.siege.status == residence.HallRunning {
			return true
		}
	}
	return false
}

// ----- the battlefield -----

// setHallZoneActive turns the hall's siege zone into a battlefield or back
// (SiegableHall.updateSiegeZone).
func (s *Server) setHallZoneActive(h *ClanHall, on bool) {
	hs := h.siege
	if hs.zoneActive == on {
		return
	}
	hs.zoneActive = on
	z := h.siegeZone
	if z == nil {
		return
	}
	for _, p := range s.hallSiegeZonePlayers(h) {
		if !p.Zones.In(z.ID) {
			continue
		}
		if on {
			s.applySiegeZone(p, z.ID)
		} else if s.leaveSiegeZone(p, z.ID) {
			p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgLeftCombatZone))
		}
		p.UpdateUserInfo()
		s.sendRelations(p)
	}
}

// hallSiegeZonePlayers returns the players in the game inside the hall's
// siege zone.
func (s *Server) hallSiegeZonePlayers(h *ClanHall) []*Player {
	if h.siegeZone == nil {
		return nil
	}
	return s.luaPlayersInZone(h.siegeZone)
}

// updateHallSiegeStates gives the attackers' members online siege state 1
// (none with clear), with UserInfo and the relations around them.
func (s *Server) updateHallSiegeStates(h *ClanHall, clear bool) {
	for _, sc := range h.siege.reg.Attackers() {
		cl := s.Clan(sc.ClanID)
		if cl == nil {
			continue
		}
		for _, p := range cl.OnlineMembers() {
			if clear {
				delete(s.residence.sides, p)
			} else {
				s.residence.sides[p] = playerSiege{state: 1, side: h.ID}
			}
			p.UpdateUserInfo()
			s.sendRelations(p)
		}
	}
}

// hallSiegePlayerEntered gives a member of a clan attacking a running hall
// siege its siege state.
func (s *Server) hallSiegePlayerEntered(p *Player) {
	id := p.ClanID()
	for _, h := range s.halls.halls {
		if h.siege != nil && h.siege.status == residence.HallRunning && h.siege.reg.IsAttacker(id) {
			s.residence.sides[p] = playerSiege{state: 1, side: h.ID}
		}
	}
}

// spawnHallDoors is SiegableHall.spawnDoor: dead doors come back (at half
// HP when weak) and every door closes.
func (s *Server) spawnHallDoors(h *ClanHall, weak bool) {
	for _, d := range s.hallDoors(h) {
		if d.IsDead() {
			s.reviveDoor(d)
			hp := float64(d.MaxHP())
			if weak {
				hp = float64(d.MaxHP() / 2)
			}
			s.setDoorHP(d, hp, true)
		}
		if d.Open() {
			s.closeDoor(d)
		}
	}
}

// spawnHallGuards is loadGuards and spawnSiegeGuards: the hall's
// clanhall_siege_guards rows, respawning after their delay while the siege
// runs (never with a delay of 0).
func (s *Server) spawnHallGuards(h *ClanHall) {
	hs := h.siege
	s.unspawnHallGuards(h)
	for _, g := range s.halls.data.GuardsOf(int(h.ID)) {
		sg := &siegeGuard{npcID: g.NpcID, pos: world.Position{X: int32(g.X), Y: int32(g.Y), Z: int32(g.Z)}, //nolint:gosec // world coordinates
			heading: int32(g.Heading), respawn: time.Duration(g.RespawnDelay) * time.Second} //nolint:gosec // heading
		sg.npc = s.spawnSiegeNpc(sg.npcID, sg.pos, sg.heading, "")
		hs.guards = append(hs.guards, sg)
		s.notifyHallSpawn(sg.npc)
	}
}

// unspawnHallGuards removes the guards and the script's NPCs and stops their
// respawns (unSpawnSiegeGuards).
func (s *Server) unspawnHallGuards(h *ClanHall) {
	hs := h.siege
	for _, g := range append(hs.guards, hs.spawns...) {
		g.stopped = true
		if g.timer != 0 {
			s.loop.Cancel(g.timer)
		}
		if g.npc != nil && g.npc.InWorld() {
			s.DespawnNpc(g.npc)
		}
	}
	hs.guards, hs.spawns = nil, nil
}

// hallSiegeDie is the OnDie hook of the hall sieges: a guard or a script NPC
// with a respawn delay comes back after it while the siege runs, a
// headquarters leaves its clan. It reports whether the NPC was the hall's.
func (s *Server) hallSiegeDie(n *Npc) bool {
	for _, h := range s.halls.halls {
		hs := h.siege
		if hs == nil {
			continue
		}
		if sc := hs.reg.FlagOwner(int32(n.ID())); sc != nil {
			sc.RemoveFlag(int32(n.ID()))
			return true
		}
		for _, g := range append(slices.Clone(hs.guards), hs.spawns...) {
			if g.npc == n {
				s.hallGuardDied(h, g)
				return true
			}
		}
	}
	return false
}

// hallGuardDied schedules a hall guard's or script NPC's respawn.
func (s *Server) hallGuardDied(h *ClanHall, g *siegeGuard) {
	if g.stopped || g.respawn <= 0 {
		return
	}
	g.timer = s.loop.After(g.respawn, func() {
		g.timer = 0
		if g.stopped || (h.siege.status != residence.HallRunning && !slices.Contains(h.siege.spawns, g)) {
			return
		}
		if g.npc != nil && g.npc.InWorld() {
			s.DespawnNpc(g.npc)
		}
		g.npc = s.spawnSiegeNpc(g.npcID, g.pos, g.heading, "")
		s.notifyHallSpawn(g.npc)
	})
}

// notifyHallSpawn runs the scripts' on_spawn for an NPC the siege placed.
func (s *Server) notifyHallSpawn(n *Npc) {
	if n == nil || s.scripts == nil || !s.scripts.Listens("spawn", n.Template.ID) {
		return
	}
	if err := s.scripts.Spawn(context.Background(), npcRef{n: n}); err != nil {
		s.log.Warn("game: on_spawn of a hall siege NPC failed", "npc", n.Template.ID, "err", err)
	}
}

// hallSpawnNpc is hall:spawn: an NPC for the siege, respawning after
// respawn while the siege is on until it is despawned.
func (s *Server) hallSpawnNpc(h *ClanHall, npcID int, pos world.Position, respawn time.Duration) *Npc {
	g := &siegeGuard{npcID: npcID, pos: pos, heading: pos.Heading, respawn: respawn}
	g.npc = s.spawnSiegeNpc(npcID, pos, pos.Heading, "")
	if g.npc == nil {
		return nil
	}
	h.siege.spawns = append(h.siege.spawns, g)
	s.notifyHallSpawn(g.npc)
	return g.npc
}

// hallDespawnNpc is hall:despawn: the script NPC leaves for good.
func (s *Server) hallDespawnNpc(h *ClanHall, objectID world.ObjectID) {
	hs := h.siege
	for i, g := range hs.spawns {
		if g.npc == nil || g.npc.ID() != objectID {
			continue
		}
		g.stopped = true
		if g.timer != 0 {
			s.loop.Cancel(g.timer)
		}
		if g.npc.InWorld() {
			s.DespawnNpc(g.npc)
		}
		hs.spawns = slices.Delete(hs.spawns, i, i+1)
		return
	}
	if n := s.Npc(objectID); n != nil && n.InWorld() {
		s.DespawnNpc(n)
	}
}

// banishNonParticipants is SiegableHallZone.banishNonSiegeParticipants:
// everyone in the hall zone who is not in the siege goes to the banish
// point.
func (s *Server) banishNonParticipants(h *ClanHall) {
	if h.zone == nil {
		return
	}
	loc, ok := s.hallSpawnLoc(h.zone, "banish")
	if !ok {
		return
	}
	for _, p := range s.luaPlayersInZone(h.zone) {
		if st, in := s.residence.sides[p]; !in || st.side != h.ID {
			s.teleportAround(p, loc, maxOffsetOnTeleport)
		}
	}
}

// checkHallTeleportZone is ResidenceHallTeleportZone.checkTeleportTask:
// 30 seconds later everyone in the zone goes to one of its spawns.
func (s *Server) checkHallTeleportZone(h *ClanHall, index int) {
	hs := h.siege
	z := h.teleZones[index]
	if z == nil || hs.teleTasks[index] != 0 {
		return
	}
	hs.teleTasks[index] = s.loop.After(30*time.Second, func() {
		delete(hs.teleTasks, index)
		pts := zoneSpawns(z, "")
		if len(pts) == 0 {
			return
		}
		loc := pts[0]
		if len(pts) > 1 {
			loc = pts[s.rnd.Get(len(pts))]
		}
		for _, p := range s.luaPlayersInZone(z) {
			s.teleportAround(p, loc, 0)
		}
	})
}

// ----- registration -----

// hallJoinSiege is the hall branch of RequestJoinSiege
// (CHSiegeManager.registerClan and unRegisterClan), then SiegeInfo.
func (s *Server) hallJoinSiege(p *Player, cl *Clan, h *ClanHall, joining bool) {
	hs := h.siege
	if joining {
		if s.Now().UnixMilli() < cl.DissolvingExpiry {
			p.SendSystemMessage(siegeMsgDissolving)
			return
		}
		other := false
		for _, o := range s.halls.halls {
			if o.siege != nil && o.siege.reg.IsAttacker(cl.ID) {
				other = true
			}
		}
		cfg := s.halls.siegeCfg
		res := residence.CheckHallRegister(residence.HallRegisterCheck{ClanID: cl.ID, ClanLevel: cl.Level, ClanCastleID: cl.CastleID,
			ClanHideoutID: cl.HideoutID, OwnerID: h.ownerID, Status: hs.status, Attacker: hs.reg.IsAttacker(cl.ID), OtherHall: other,
			Attackers: len(hs.reg.Attackers()), MinLevel: cfg.MinClanLevel, MaxAttackers: cfg.MaxAttackers,
			MinLevelMessage: "Only clans of level " + idText(int32(cfg.MinClanLevel)) + " or higher may register for a castle siege"}) //nolint:gosec // small
		switch {
		case res.Text != "":
			p.SendMessage(res.Text)
		case res.WithArg:
			p.SendSystemMessage(serverpackets.SystemMessageID(res.SysMsg), hallParam(h)) //nolint:gosec // message IDs
		case res.SysMsg != 0:
			p.SendSystemMessage(serverpackets.SystemMessageID(res.SysMsg)) //nolint:gosec // message IDs
		default:
			hs.reg.AddAttacker(cl.ID)
			h.save()
		}
	} else if hs.status == residence.HallRegistering && hs.reg.IsAttacker(cl.ID) {
		hs.reg.Remove(cl.ID)
		h.save()
	}
	s.showHallSiegeInfo(p, h)
}

// showHallSiegeInfo sends SiegeInfo for a hall (SiegableHall.showSiegeInfo).
func (s *Server) showHallSiegeInfo(p *Player, h *ClanHall) {
	info := serverpackets.SiegeInfo{ResidenceID: h.ID, Now: int32(s.Now().Unix()), SiegeDate: int32(h.siege.nextSiege.Unix())} //nolint:gosec // Unix seconds fit until 2038
	if o := h.Owner(); o != nil {
		info.OwnerClanID, info.OwnerClanName, info.OwnerLeaderName = o.ID, o.Name, o.LeaderName()
		info.OwnerAllyID, info.OwnerAllyName = o.AllyID, o.AllyName
		info.OwnerLeader = p.ClanID() == o.ID && p.IsClanLeader()
	}
	p.Send(serverpackets.NewSiegeInfoPacket(info))
}

// sendHallAttackerList sends SiegeAttackerList for a hall.
func (s *Server) sendHallAttackerList(p *Player, h *ClanHall) {
	l := serverpackets.SiegeAttackerList{ResidenceID: h.ID}
	for _, sc := range h.siege.reg.Attackers() {
		if line, ok := s.siegeClanLine(sc.ClanID, 0); ok {
			l.Clans = append(l.Clans, line)
		}
	}
	p.Send(serverpackets.NewSiegeAttackerListPacket(l))
}

// hallSiegeOf returns the contestable hall with the ID, or nil.
func (s *Server) hallSiegeOf(id int32) *ClanHall {
	if h := s.halls.byID[id]; h != nil && h.siege != nil {
		return h
	}
	return nil
}
