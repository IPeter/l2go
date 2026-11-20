package game

import (
	"math"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/npcai"
	"github.com/IPeter/l2go/internal/world"
)

// Minions (Plan 13, decision A9): Mobius MinionList with the minion parts
// of Monster, Attackable, AttackableAI and Spawn.initializeNpc.
//
//   - A master is a monster whose template has a "Privates" minion list and
//     no SummonPrivateRate parameter (NpcData.getMasterMonsterIDs). Its
//     minions spawn when it spawns (placeNpc), around it: MinionList's ring
//     with the master's collision radius + 30 inside and 200 outside.
//   - A minion has its master as leader and no spawn: it follows the living
//     leader (thinkActive), helps it (onAssist: hate 1, or 10 when the
//     master was attacked, × 10 for a raid master) and a raid minion is
//     teleported next to its master every 120 s when it was dragged out of
//     the master's territory or boss zone (checkMinionLeash).
//   - A dead minion of a raid master comes back after
//     npcs.raidMinionRespawnTime (300 s) while the master lives, unless
//     npcs.customMinionsRespawnTime names its own time (0: never). Minions
//     of other masters do not come back. Mobius never reads the per-minion
//     respawnTime of the XML (MinionHolder.getRespawnTime is unused).
//   - A raid master's death deletes its minions; other masters' minions
//     stay unless npcs.forceDeleteMinions. Deleting a master (deleteMe, not
//     a decay) deletes them always. A teleported master takes its minions
//     along.
//
// The API for scripts and other features: Npc.Leader, Npc.Minions,
// Npc.HasMinions, Npc.IsRaidMinion, Server.SpawnMinion.

// MinionList constants (Mobius MinionList and AttackableAI).
const (
	minionBaseAggro       = 1                 // BASE_AGGRO_AMOUNT
	minionMasterAggro     = 10                // MASTER_AGGRO_AMOUNT
	minionRaidAggroFactor = 10                // RAID_AGGRO_MULTIPLIER
	minionLeashInterval   = 120 * time.Second // MINION_LEASH_INTERVAL
	minionLeashOffset     = 100               // MINION_LEASH_OFFSET
	// minionRespawnDefault is MinionList's INVALID_RESPAWN_TIME: the default
	// respawn of the master's kind.
	minionRespawnDefault time.Duration = -1
	// minionMasterList is the minion list Spawn.initializeNpc spawns.
	minionMasterList = "Privates"
)

// npcBoss is the minion and boss state of an NPC (Npc.boss). Loop-only.
type npcBoss struct {
	// leader is the master of a minion (Mobius Monster._master).
	leader *Npc
	// minions is a master's list, nil until it spawned minions (Mobius
	// hasMinions).
	minions *minionList
	// raidMinion is Mobius isRaidMinion: a minion of a raid master.
	raidMinion bool
	// leashAt is the next leash check of a raid minion (AttackableAI
	// _minionLeashTime), zero before its first think.
	leashAt time.Time
	// comeBack is set while a respawn task waits for a dead minion: its
	// object ID and passive skills stay when its corpse decays.
	comeBack bool
	// noRaidCurse is Mobius setUseRaidCurse(false).
	noRaidCurse bool
	// raid is the raid spawner's entry of a raid boss it spawned (raid.go).
	raid *raidEntry
}

// minionList is Mobius MinionList: a master's living minions and the
// respawn tasks of its dead ones.
type minionList struct {
	master *Npc
	// spawned are the minions in the world and alive, in spawn order.
	spawned []*Npc
	// respawns are the pending respawn tasks, by minion.
	respawns map[*Npc]world.TimerID
}

// Leader returns the master of a minion, nil for an NPC that is none
// (Mobius getLeader). Loop-only.
func (n *Npc) Leader() *Npc { return n.boss.leader }

// HasMinions reports whether n is a master with a minion list (Mobius
// hasMinions). Loop-only.
func (n *Npc) HasMinions() bool { return n.boss.minions != nil }

// Minions returns the living minions of a master, in spawn order (Mobius
// getSpawnedMinions). Loop-only.
func (n *Npc) Minions() []*Npc {
	if n.boss.minions == nil {
		return nil
	}
	return slices.Clone(n.boss.minions.spawned)
}

// IsRaidMinion reports a minion of a raid master (Mobius isRaidMinion).
// Loop-only.
func (n *Npc) IsRaidMinion() bool { return n.boss.raidMinion }

// IsRaid reports a raid boss, a grand boss or a raid minion (Mobius
// isRaid). Loop-only.
func (n *Npc) IsRaid() bool { return creatureRaid(n) }

// minionListOf returns n's minion list, making it on first use (Mobius
// getMinionList).
func minionListOf(n *Npc) *minionList {
	if n.boss.minions == nil {
		n.boss.minions = &minionList{master: n, respawns: make(map[*Npc]world.TimerID)}
	}
	return n.boss.minions
}

// isMasterTemplate is NpcData's master test: a "Privates" list and no
// SummonPrivateRate parameter.
func isMasterTemplate(t *data.NpcTemplate) bool {
	if len(t.Parameters.Minions[minionMasterList]) == 0 {
		return false
	}
	_, summons := t.Parameters.Params["SummonPrivateRate"]
	return !summons
}

// minionCount is MinionHolder.getCount: Rnd.get(count, max) when max is
// above count, the count otherwise.
func (s *Server) minionCount(h data.NpcMinion) int {
	if h.Max > h.Count {
		return s.rnd.Between(h.Count, h.Max)
	}
	return h.Count
}

// spawnMinions spawns the minions a master lacks after it spawned (Mobius
// Spawn.initializeNpc and MinionList.spawnMinions): for each entry of its
// Privates list, the count less the living minions of that template. A
// minion itself never spawns minions this way.
func (s *Server) spawnMinions(n *Npc) {
	if n.boss.leader != nil || !n.IsMonster() || !isMasterTemplate(n.Template) || n.Dead {
		return
	}
	l := minionListOf(n)
	for _, h := range n.Template.Parameters.Minions[minionMasterList] {
		want := s.minionCount(h) - l.countByID(h.NpcID)
		for range want {
			s.SpawnMinion(n, h.NpcID)
		}
	}
}

// countByID counts the living minions of a template.
func (l *minionList) countByID(id int) int {
	c := 0
	for _, m := range l.spawned {
		if m.Template.ID == id {
			c++
		}
	}
	return c
}

// remove takes a minion off the living list.
func (l *minionList) remove(m *Npc) {
	l.spawned = slices.DeleteFunc(l.spawned, func(o *Npc) bool { return o == m })
}

// SpawnMinion spawns a new minion of template id around master (Mobius
// MinionList.spawnMinion) and returns it, or nil when the template cannot
// be spawned. Loop-only.
func (s *Server) SpawnMinion(master *Npc, id int) *Npc {
	if id == 0 {
		return nil
	}
	t, err := s.spawnableTemplate(id)
	if err != nil {
		s.log.Warn("game: cannot spawn a minion", "master", master.Template.ID, "minion", id, "err", err)
		return nil
	}
	m, err := s.newNpc(t)
	if err != nil {
		s.log.Error("game: cannot make a minion", "master", master.Template.ID, "minion", id, "err", err)
		return nil
	}
	if !s.placeMinion(master, m) {
		s.releaseNpcID(m)
		return nil
	}
	return m
}

// placeMinion is Mobius MinionList.initializeNpc and the minion part of
// Monster.onSpawn: full HP and MP (placeNpc), the master's heading, the
// master as leader, a point of the spawn ring at the master's height; the
// minion joins the master's list as a raid minion when the master is a
// raid.
func (s *Server) placeMinion(master, m *Npc) bool {
	mp := master.Position()
	pt := npcai.SpawnPoint(s.rnd, npcai.Point{X: mp.X, Y: mp.Y, Z: mp.Z}, master.Template.Collision.Radius)
	m.boss.leader = master
	m.boss.raidMinion = creatureRaid(master)
	l := minionListOf(master)
	l.spawned = append(l.spawned, m)
	if err := s.placeNpc(m, world.Position{X: pt.X, Y: pt.Y, Z: mp.Z, Heading: mp.Heading}, nil); err != nil {
		s.log.Warn("game: cannot place a minion", "master", master.Template.ID, "minion", m.Template.ID, "err", err)
		l.remove(m)
		m.boss.leader = nil
		return false
	}
	return true
}

// minionsOnDie is the minion part of Mobius Attackable.doDie: a dead minion
// leaves its master's list (and may come back), a dead master's minions go
// for a raid master.
func (s *Server) minionsOnDie(n *Npc) {
	if !n.IsMonster() {
		return
	}
	if l := n.boss.leader; l != nil && l.boss.minions != nil {
		respawn := minionRespawnDefault
		if d, ok := s.bosses.minionTimes[n.Template.ID]; ok {
			respawn = d
		}
		s.onMinionDie(l.boss.minions, n, respawn)
	}
	if n.boss.minions != nil {
		s.onMasterDie(n.boss.minions, false)
	}
}

// onMinionDie is Mobius MinionList.onMinionDie: respawn 0 ends the minion's
// tie to its master; a negative one is the default, RaidMinionRespawnTime
// for a raid master and none otherwise. A minion with a respawn time comes
// back after it, while the master lives.
func (s *Server) onMinionDie(l *minionList, m *Npc, respawn time.Duration) {
	if respawn == 0 {
		m.boss.leader = nil
	}
	l.remove(m)
	if respawn < 0 {
		respawn = 0
		if creatureRaid(l.master) {
			respawn = time.Duration(s.settings.NPCs.RaidMinionRespawnTime)
		}
	}
	if respawn <= 0 || l.master.Dead {
		return
	}
	if old, ok := l.respawns[m]; ok {
		s.loop.Cancel(old)
	}
	m.boss.comeBack = true
	l.respawns[m] = s.loop.After(respawn, func() { s.minionRespawn(l, m) })
}

// minionRespawn is Mobius MinionRespawnTask: a minion out of the world
// comes back around its living master in the world and takes over the
// master's hate list, attacking the first enemy on it.
func (s *Server) minionRespawn(l *minionList, m *Npc) {
	delete(l.respawns, m)
	m.boss.comeBack = false
	master := l.master
	if master.Dead || !master.InWorld() || m.InWorld() {
		if !m.InWorld() {
			s.releaseNpcID(m) // gone for good
		}
		return
	}
	if !s.placeMinion(master, m) {
		s.releaseNpcID(m)
		return
	}
	if len(master.ai.hate) == 0 {
		return
	}
	for _, a := range master.ai.hate {
		if a.Attacker == nil || a.Attacker == Creature(m) {
			continue
		}
		if o := m.aggroOf(a.Attacker); o != nil {
			*o = a
		} else {
			m.ai.hate = append(m.ai.hate, a)
		}
	}
	if first := master.ai.hate[0].Attacker; first != nil && first != Creature(m) {
		s.npcSetAttack(m, first)
	}
}

// onMasterDie is Mobius MinionList.onMasterDie: the minions of a raid
// master (any master with force or npcs.forceDeleteMinions) are deleted
// and their respawn tasks end.
func (s *Server) onMasterDie(l *minionList, force bool) {
	if !creatureRaid(l.master) && !force && !s.settings.NPCs.ForceDeleteMinions {
		return
	}
	minions := l.spawned
	l.spawned = nil
	for _, m := range minions {
		m.boss.leader = nil
		s.DespawnNpc(m) // Mobius deleteMe
	}
	for m, id := range l.respawns {
		s.loop.Cancel(id)
		m.boss.comeBack = false
		if !m.InWorld() {
			s.releaseNpcID(m) // its corpse is gone already
		}
	}
	clear(l.respawns)
}

// minionsDeleted is Mobius Monster.deleteMe for n leaving the world for
// good (a script's or a GM's delete, not a decay): its minions are deleted
// and a minion leaves its master's list for good.
func (s *Server) minionsDeleted(n *Npc) {
	if l := n.boss.minions; l != nil {
		s.onMasterDie(l, true)
	}
	if m := n.boss.leader; m != nil && m.boss.minions != nil {
		s.onMinionDie(m.boss.minions, n, 0)
	}
}

// minionsAssist is the onAssist call of Mobius reduceCurrentHp,
// notifyActionAttacked and notifyActionAggression: n's own minions, and the
// minions of n's master, come for attacker.
func (s *Server) minionsAssist(n *Npc, attacker Creature) {
	if attacker == nil || !n.IsMonster() {
		return
	}
	if l := n.boss.minions; l != nil {
		s.minionAssist(l, n, attacker)
	}
	if m := n.boss.leader; m != nil && m.boss.minions != nil {
		s.minionAssist(m.boss.minions, n, attacker)
	}
}

// minionAssist is Mobius MinionList.onAssist: a master out of combat hates
// the attacker by 1; each living minion hates it by 10 when the master was
// attacked, or by 1 when a minion was and it is out of combat itself; × 10
// for a raid master.
func (s *Server) minionAssist(l *minionList, caller *Npc, attacker Creature) {
	master := l.master
	if !master.Dead && !s.InCombat(master) {
		s.addDamageHate(master, attacker, 0, minionBaseAggro)
	}
	callerIsMaster := caller == master
	aggro := int64(minionBaseAggro)
	if callerIsMaster {
		aggro = minionMasterAggro
	}
	if creatureRaid(master) {
		aggro *= minionRaidAggroFactor
	}
	for _, m := range slices.Clone(l.spawned) {
		if !m.Dead && (callerIsMaster || !s.InCombat(m)) {
			s.addDamageHate(m, attacker, 0, aggro)
		}
	}
}

// minionsMasterTeleported is Mobius MinionList.onMasterTeleported: the
// living minions that can move are teleported to the spawn ring around
// their teleported master.
func (s *Server) minionsMasterTeleported(n *Npc) {
	l := n.boss.minions
	if l == nil {
		return
	}
	mp := n.Position()
	for _, m := range slices.Clone(l.spawned) {
		if m.Dead || npcMovementDisabled(m) {
			continue
		}
		pt := npcai.SpawnPoint(s.rnd, npcai.Point{X: mp.X, Y: mp.Y, Z: mp.Z}, n.Template.Collision.Radius)
		s.TeleportNpc(m, world.Position{X: pt.X, Y: pt.Y, Z: mp.Z})
	}
}

// minionsLeashed sends a master's minions home with it when the master was
// pulled too far (AttackableAI.thinkAttack with AggroDistanceCheck): full
// HP and MP, no attack, no enemies, a walk to the master's spawn.
func (s *Server) minionsLeashed(n *Npc, home world.Position) {
	l := n.boss.minions
	if l == nil {
		return
	}
	for _, m := range slices.Clone(l.spawned) {
		m.SetCurrentHPMP(float64(m.MaxHP()), float64(m.MaxMP()))
		s.AbortAttack(m)
		m.ClearAggroList()
		m.clearAttackBy()
		if m.ai.attached {
			s.npcSetMoveTo(m, home)
		} else {
			s.TeleportNpc(m, home)
		}
	}
}

// minionFollow is the follow branch of Mobius thinkActive: a minion of a
// living leader walks or runs as the leader does, and when it is farther
// than 200 (500 for a raid minion) from the leader in 2D it walks to a
// point of the ring around the leader (inner radius 30); otherwise it may
// buff itself on 1 think in 30. It reports whether the branch took the
// think (Mobius returns either way; the wandering is an else branch).
func (s *Server) minionFollow(n *Npc) bool {
	leader := n.boss.leader
	if leader == nil || leader.Dead {
		return false
	}
	if leader.Running() != n.Running() {
		s.SetNpcRunning(n, leader.Running())
	}
	offset := npcai.FollowOffset
	if n.boss.raidMinion {
		offset = npcai.RaidFollowOffset
	}
	lp := leader.Position()
	if n.Position().Distance2D(lp) > float64(offset) {
		pt := npcai.RingPoint(s.rnd, npcai.Point{X: lp.X, Y: lp.Y, Z: lp.Z}, npcai.FollowMinRadius, offset)
		s.NpcMoveTo(n, world.Position{X: pt.X, Y: pt.Y, Z: pt.Z})
		return true
	}
	if s.rnd.Get(aiRandomWalkRate) == 0 {
		s.npcDo(n, npcai.Buff(aiNpcOf(n))) // ai_npcai.go: a buff on itself
	}
	return true
}

// minionLeash is Mobius AttackableAI.checkMinionLeash, at the start of a
// think: every 120 s (the first check 120 s after the first think) a raid
// minion that stands still outside its living master's territory, or boss
// zone, forgets its enemies and is teleported within 100 of the master.
func (s *Server) minionLeash(n *Npc) {
	b := &n.boss
	if !b.raidMinion || n.Dead || n.Move != nil {
		return
	}
	now := s.Now()
	if b.leashAt.IsZero() {
		b.leashAt = now.Add(minionLeashInterval)
		return
	}
	if b.leashAt.After(now) {
		return
	}
	b.leashAt = now.Add(minionLeashInterval)
	master := b.leader
	if master == nil || master.Dead || s.insideMasterTerritory(n, master) {
		return
	}
	n.ClearAggroList()
	n.clearAttackBy()
	s.teleportNpcAround(n, master.Position(), minionLeashOffset)
}

// insideMasterTerritory is Mobius isInsideMasterTerritory: inside the
// master's spawn territory, else inside the boss zone the master is in;
// true when the master has neither.
func (s *Server) insideMasterTerritory(n, master *Npc) bool {
	pos := n.Position()
	if sp := master.spawn; sp != nil && sp.territory != nil {
		return territoryContains3D(sp.territory, pos)
	}
	if z := s.zoneAt(master.Position(), "BossZone"); z != nil {
		return z.Shape.Contains(int(pos.X), int(pos.Y), int(pos.Z))
	}
	return true
}

// territoryContains3D is Mobius NpcSpawnTerritory.isInsideZone(x, y, z):
// inside the shape and outside every banned area.
func territoryContains3D(t *data.SpawnTerritory, pos world.Position) bool {
	x, y, z := int(pos.X), int(pos.Y), int(pos.Z)
	if !t.Shape.Contains(x, y, z) {
		return false
	}
	for _, b := range t.Banned {
		if b.Contains(x, y, z) {
			return false
		}
	}
	return true
}

// teleportNpcAround is Mobius teleToLocation(loc, randomOffset) for an NPC:
// x and y move by Rnd.get(-offset, offset) each, drawn again (up to 100
// times) while the point is closer to loc than twice the NPC's collision
// radius (at most offset) or cannot be seen or reached from it; then the
// geodata's valid location, unless the NPC flies. The heading is loc's.
func (s *Server) teleportNpcAround(n *Npc, loc world.Position, offset int) {
	z := loc.Z
	if !npcFlying(n) {
		z = s.geo.Height(loc.X, loc.Y, loc.Z)
	}
	x, y := loc.X, loc.Y
	if offset > 0 {
		draw := func() {
			x = loc.X + int32(s.rnd.Between(-offset, offset)) //nolint:gosec // small offset
			y = loc.Y + int32(s.rnd.Between(-offset, offset)) //nolint:gosec // small offset
		}
		draw()
		collision := math.Min(n.Template.Collision.Radius*2, float64(offset))
		from := geo.Point{X: loc.X, Y: loc.Y, Z: loc.Z}
		for count := 0; count < 100; count++ {
			to := geo.Point{X: x, Y: y, Z: z}
			dx, dy, dz := float64(loc.X-x), float64(loc.Y-y), float64(loc.Z-z)
			if math.Sqrt(dx*dx+dy*dy+dz*dz) >= collision && s.geo.CanSeeTarget(from, to) && s.geo.CanMoveToTarget(from, to) {
				break
			}
			draw()
		}
		if !npcFlying(n) {
			valid := s.geo.ValidLocation(geo.Point{X: loc.X, Y: loc.Y, Z: z}, geo.Point{X: x, Y: y, Z: z})
			x, y, z = valid.X, valid.Y, valid.Z
		}
	}
	s.TeleportNpc(n, world.Position{X: x, Y: y, Z: z, Heading: loc.Heading})
}
