package game

import (
	"math"
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/world"
)

// The thinks of the monster AI (Mobius AttackableAI.thinkActive and
// thinkAttack), without skills, minions, raids and fake players. Skill use
// comes with Milestone 7: until then monsters only swing.

// npcThinkActive is Mobius AttackableAI.thinkActive:
//   - the post-spawn counter moves towards 0; from 0 an aggressive NPC
//     hates the players in its aggro range, and attacks the most hated;
//   - at full HP and MP it forgets its attackers on 1 think in 500;
//   - away from home with no target it walks back;
//   - otherwise it wanders on 1 think in 30.
func (s *Server) npcThinkActive(n *Npc) {
	if !s.world.NeighborsActive(n) {
		return
	}
	ai := &n.ai
	switch {
	case ai.globalAggro < 0:
		ai.globalAggro++
	case ai.globalAggro > 0:
		ai.globalAggro--
	}
	if ai.globalAggro >= 0 {
		scan := s.ai.scan[:0]
		for v := range s.world.VisiblePlayers(n) {
			if p, ok := v.(*Player); ok {
				scan = append(scan, p)
			}
		}
		for _, target := range scan {
			if s.isAggressiveTowards(n, target) && n.Hating(target) == 0 {
				s.addDamageHate(n, target, 0, 0)
			}
		}
		clear(scan)
		s.ai.scan = scan[:0]

		if hated := n.MostHated(); hated != nil {
			if n.Hating(hated)+int64(ai.globalAggro) > 0 {
				if !n.Running() {
					s.SetNpcRunning(n, true)
				}
				s.npcSetAttack(n, hated)
			}
			return
		}
	}

	if n.CurHP == float64(n.MaxHP()) && n.CurMP == float64(n.MaxMP()) && len(ai.attackBy) > 0 && s.rnd.Get(aiForgetRate) == 0 {
		n.ClearAggroList()
		n.clearAttackBy()
	}

	home, terr, hasHome := s.npcHome(n)
	pos := n.Position()
	drift := s.settings.NPCs.MaxDriftRange
	if hasHome {
		away := false
		if terr != nil {
			away = !territoryContainsXY(terr, pos.X, pos.Y)
		} else {
			away = pos.Distance2D(home) > float64(drift)
		}
		if away && s.targetGone(ai.target) {
			s.SetNpcRunning(n, false)
			s.returnHome(n)
			return
		}
	}
	if p, ok := ai.target.(*Player); ok && p.IsDead() {
		return // Mobius: do not leave a dead player
	}
	if !hasHome || s.rnd.Get(aiRandomWalkRate) != 0 || !s.aiClassOf(n).randomWalk {
		return
	}
	s.randomWalk(n, home, terr)
}

// targetGone is Mobius's "no target" test of the walk home: no target, an
// invisible one or a dead player (AttackablesCampPlayerCorpses off).
func (s *Server) targetGone(t world.Object) bool {
	if t == nil || creatureInvisible(t) {
		return true
	}
	p, ok := t.(*Player)
	return ok && !s.settings.NPCs.AttackablesCampPlayerCorpses && p.IsDead()
}

// returnHome is Mobius Attackable.returnHome: the NPC forgets its enemies
// and walks to its spawn point.
func (s *Server) returnHome(n *Npc) {
	n.ClearAggroList()
	if home, _, ok := s.npcHome(n); ok {
		s.npcSetMoveTo(n, home)
	}
}

// randomWalk is the wandering of thinkActive: a territory NPC tries up to 3
// points within 300 of where it stands that are inside the territory; any
// other walks to a point within 300 of its spawn, or to the spawn when it
// is farther.
func (s *Server) randomWalk(n *Npc, home world.Position, terr *data.SpawnTerritory) {
	drift := s.settings.NPCs.MaxDriftRange
	pos := n.Position()
	if terr != nil {
		for range aiWanderAttempts {
			dx := s.rnd.Get(drift * 2)
			dy := s.rnd.Between(dx, drift*2)
			dy = int(math.Sqrt(float64(dy*dy - dx*dx)))
			x := int32(dx) + pos.X - int32(drift) //nolint:gosec // within twice the drift range, a world distance
			y := int32(dy) + pos.Y - int32(drift) //nolint:gosec // as above
			if !territoryContainsXY(terr, x, y) {
				continue
			}
			to := world.Position{X: x, Y: y, Z: pos.Z}
			if !npcFlying(n) {
				to = s.validLocation(pos, to)
			}
			if territoryContainsXY(terr, to.X, to.Y) {
				s.NpcMoveTo(n, to)
				break
			}
		}
		return
	}
	to := home
	if within2D(pos, home, drift) {
		dx := s.rnd.Get(drift * 2)
		dy := s.rnd.Between(dx, drift*2)
		dy = int(math.Sqrt(float64(dy*dy - dx*dx)))
		to.X = int32(dx) + home.X - int32(drift) //nolint:gosec // within twice the drift range, a world distance
		to.Y = int32(dy) + home.Y - int32(drift) //nolint:gosec // as above
		to.Z = pos.Z
	}
	if !npcFlying(n) {
		to = s.validLocation(pos, to)
	}
	if home.Distance2D(to) <= float64(drift) {
		s.NpcMoveTo(n, to)
	}
}

// validLocation is Mobius GeoEngine.getValidLocation from from to to.
func (s *Server) validLocation(from, to world.Position) world.Position {
	p := s.geo.ValidLocation(geo.Point{X: from.X, Y: from.Y, Z: from.Z}, geo.Point{X: to.X, Y: to.Y, Z: to.Z})
	return world.Position{X: p.X, Y: p.Y, Z: p.Z}
}

// canSee is Mobius GeoEngine.canSeeTarget between two objects.
func (s *Server) canSee(a, b world.Object) bool {
	pa, pb := a.Position(), b.Position()
	return s.geo.CanSeeTarget(geo.Point{X: pa.X, Y: pa.Y, Z: pa.Z}, geo.Point{X: pb.X, Y: pb.Y, Z: pb.Z})
}

// isAggressiveTowards is Mobius AttackableAI.isAggressiveTowards for the
// creatures that exist: only players can draw aggression (NPCs are never
// auto-attackable by monsters, and guards leave monsters alone by default).
// A player draws it when it is alive, closer than the aggro range (3D), and
// to a guard or a friendly mob has karma, to a monster that is aggressive
// stands outside a peace zone or AltMobAgroInPeaceZone is on; and the NPC
// sees it.
func (s *Server) isAggressiveTowards(n *Npc, target Creature) bool {
	p, ok := target.(*Player)
	if !ok || p.state != stateInGame {
		return false
	}
	if p.Teleporting() && p.IsGM() {
		return false // an invulnerable GM
	}
	if p.IsDead() || !within3D(n.Position(), p.Position(), s.npcAggroRange(n)) {
		return false
	}
	c := s.aiClassOf(n)
	switch {
	case c.guard, c.friendly:
		return p.Karma() > 0 && s.canSee(n, p)
	case !s.settings.NPCs.MobAggroInPeaceZone && p.Zones.Inside(ZonePeace) && p.Zones.Inside(ZoneNoPvP):
		return false
	}
	return c.aggressive && s.canSee(n, p)
}

// npcThinkAttack is Mobius AttackableAI.thinkAttack: fight the most hated
// enemy; give up after 2 min without action (teleporting home if it was in
// combat or no player sees it); walk towards an enemy it cannot see; step
// around a crowded target; archers step back; chase an enemy out of reach
// with MoveToPawn; swing at one in reach.
func (s *Server) npcThinkAttack(n *Npc) {
	ai := &n.ai
	c := s.aiClassOf(n)
	if c.monster && s.settings.NPCs.AggroDistanceCheck && s.hasStrayed(n) {
		s.leash(n)
		return
	}
	most := n.MostHated()
	if most == nil {
		s.npcSetActive(n)
		return
	}
	if ai.attackTarget != most {
		ai.attackTarget = most
	}
	if ai.target != world.Object(most) {
		ai.target = most
	}
	if npcMovementDisabled(n) {
		s.movementDisabledThink(n)
		return
	}
	if most.IsDead() {
		n.StopHating(most)
		return
	}
	if ai.attackTimeout < s.gameTicks() {
		s.giveUp(n)
		return
	}
	if !s.canSee(n, most) {
		if n.Position().Distance3D(most.Position()) < aiUnseenChaseRange {
			s.NpcMoveTo(n, most.Position())
		}
		return
	}

	pos, tpos := n.Position(), most.Position()
	collision := npcCollision(n)
	combined := collision + creatureCollision(most)
	if s.rnd.Get(100) <= aiCrowdChance && s.crowded(n, most, collision) {
		nx := int32(combined) + int32(s.rnd.Get(40)) //nolint:gosec // below 40
		if s.rnd.Bool() {
			nx = tpos.X + nx
		} else {
			nx = tpos.X - nx
		}
		ny := int32(combined) + int32(s.rnd.Get(40)) //nolint:gosec // below 40
		if s.rnd.Bool() {
			ny = tpos.Y + ny
		} else {
			ny = tpos.Y - ny
		}
		if !within2D(pos, world.Position{X: nx, Y: ny}, int(collision)) {
			s.NpcMoveTo(n, s.validLocation(pos, world.Position{X: nx, Y: ny, Z: pos.Z + 30}))
		}
		return
	}
	if c.archer && s.rnd.Get(100) < aiArcherChance {
		if pos.Distance2D(tpos) <= float64(aiArcherNear+combined) {
			to := world.Position{X: pos.X, Y: pos.Y, Z: pos.Z + 30}
			if tpos.X < pos.X {
				to.X += aiArcherStep
			} else {
				to.X -= aiArcherStep
			}
			if tpos.Y < pos.Y {
				to.Y += aiArcherStep
			} else {
				to.Y -= aiArcherStep
			}
			if s.geo.CanMoveToTarget(geo.Point{X: pos.X, Y: pos.Y, Z: pos.Z}, geo.Point{X: to.X, Y: to.Y, Z: to.Z}) {
				s.npcSetMoveTo(n, to)
			}
			return
		}
	}

	rng := int32(n.Stats.PhysicalAttackRange()) + combined //nolint:gosec // a weapon's reach
	if n.Move != nil {
		rng *= 2
	}
	if c.archer {
		rng = aiArcherRange + combined
	}
	if pos.Distance2D(tpos) > float64(rng) {
		if s.checkTarget(n, most) {
			s.NpcMoveToPawn(n, most, rng)
			return
		}
		most = s.targetReconsider(n)
		if most == nil {
			return
		}
		ai.attackTarget, ai.target = most, most
	}
	s.npcDoAttack(n, ai.attackTarget)
}

// crowded reports whether another attackable stands within collision (2D)
// of n (Mobius's check before stepping around the target).
func (s *Server) crowded(n *Npc, target Creature, collision int32) bool {
	pos := n.Position()
	for o := range s.world.Visible(n) {
		other, ok := o.(*Npc)
		if !ok || other.class == nil || !other.class.attackable || Creature(other) == target {
			continue
		}
		if within2D(pos, other.Position(), int(collision)) {
			return true
		}
	}
	return false
}

// giveUp ends a fight that timed out (Mobius thinkAttack): ACTIVE, no
// target, no enemies, walking; a monster with a spawn teleports home when
// it was in combat or no player sees it.
func (s *Server) giveUp(n *Npc) {
	wasInCombat := s.InAttackStance(n)
	s.npcSetActive(n)
	n.ai.target = nil
	n.ClearAggroList()
	n.clearAttackBy()
	s.SetNpcRunning(n, false)
	home, _, ok := s.npcHome(n)
	if !ok || !s.aiClassOf(n).monster {
		return
	}
	if wasInCombat || !s.playerSees(n) {
		s.TeleportNpc(n, home)
	}
}

// playerSees reports whether a player is in n's 3×3×3 regions (Mobius
// World.getFirstVisibleObject(npc, Player.class)).
func (s *Server) playerSees(n *Npc) bool {
	for range s.world.VisiblePlayers(n) {
		return true
	}
	return false
}

// hasStrayed is Mobius hasStrayedFromSpawn with AggroDistanceCheck on.
func (s *Server) hasStrayed(n *Npc) bool {
	home, terr, ok := s.npcHome(n)
	if !ok {
		return false
	}
	pos := n.Position()
	out := pos.Distance2D(home) > float64(s.settings.NPCs.AggroDistanceCheckRange)
	if terr != nil {
		return out && !territoryContainsXY(terr, pos.X, pos.Y)
	}
	return out
}

// leash sends a monster pulled too far home with full HP and MP (Mobius
// thinkAttack with AggroDistanceCheck and its RestoreLife default).
func (s *Server) leash(n *Npc) {
	home, _, _ := s.npcHome(n)
	n.SetCurrentHPMP(float64(n.MaxHP()), float64(n.MaxMP()))
	s.AbortAttack(n)
	n.ClearAggroList()
	n.clearAttackBy()
	s.npcSetMoveTo(n, home)
}

// movementDisabledThink is Mobius movementDisable without skills: swing at
// a target in reach and sight, or look for another.
func (s *Server) movementDisabledThink(n *Npc) {
	target := n.ai.attackTarget
	if target == nil {
		return
	}
	dist := n.Position().Distance2D(target.Position())
	s.rnd.Get(100) // Mobius draws for its skill chances
	//nolint:gosec // a weapon's reach
	reach := int32(n.Stats.PhysicalAttackRange()) + npcCollision(n) + creatureCollision(target)
	if dist <= float64(reach) && s.canSee(n, target) {
		s.npcDoAttack(n, target)
		return
	}
	if t := s.targetReconsider(n); t != nil {
		n.ai.attackTarget, n.ai.target = t, t
	}
}

// checkTarget is Mobius AttackableAI.checkTarget: a living creature the NPC
// may attack (players; NPCs never, as monsters and guards leave each other
// alone by default), in reach and sight when the NPC cannot move.
func (s *Server) checkTarget(n *Npc, target Creature) bool {
	if target == nil || target.IsDead() {
		return false
	}
	if npcMovementDisabled(n) {
		reach := int(n.Stats.PhysicalAttackRange()) + int(npcCollision(n)) + int(creatureCollision(target))
		if !within2D(n.Position(), target.Position(), reach) || !s.canSee(n, target) {
			return false
		}
	}
	p, ok := target.(*Player)
	return ok && p.state == stateInGame
}

// targetReconsider is Mobius targetReconsider(false): the most hated
// attacker that passes checkTarget (hate 0 included), or for an aggressive
// NPC the first player in its aggro range that does.
func (s *Server) targetReconsider(n *Npc) Creature {
	var best Creature
	bestHate := int64(math.MinInt64)
	for i := range n.ai.hate {
		a := &n.ai.hate[i]
		if s.checkTarget(n, a.Attacker) && a.Hate > bestHate {
			best, bestHate = a.Attacker, a.Hate
		}
	}
	if best != nil || !s.aiClassOf(n).aggressive {
		return best
	}
	r := s.npcAggroRange(n)
	for v := range s.world.VisiblePlayersInRange(n, int32(r)) { //nolint:gosec // aggro ranges are small template values
		if p, ok := v.(*Player); ok && s.checkTarget(n, p) {
			return p
		}
	}
	return nil
}

// isClan is Mobius NpcTemplate.isClan(Set): both have clans, and they share
// one or the first is in the generic clan ALL.
func isClan(a, b *aiClass) bool {
	if len(a.clans) == 0 || len(b.clans) == 0 {
		return false
	}
	if slices.Contains(a.clans, "ALL") {
		return true
	}
	for _, c := range b.clans {
		if slices.Contains(a.clans, c) {
			return true
		}
	}
	return false
}

// callClanHelp is the call for help of Mobius notifyActionAttacked: the
// NPCs of n's clan within its clan help range plus its collision radius
// that have an AI, live, stand within 600 in height of the attacker, spawned
// at least 7 s ago, are IDLE or ACTIVE and do not ignore n come for the
// attacker, if they see it.
func (s *Server) callClanHelp(n *Npc, attacker Creature) {
	c := s.aiClassOf(n)
	r := int32(c.clanHelpRange) + npcCollision(n) //nolint:gosec // clan help ranges are small template values
	for _, other := range s.clanCandidates(n, r) {
		if !other.ai.attached || other.Dead || abs32(attacker.Position().Z-other.Position().Z) > aiClanHelpMaxDZ {
			continue
		}
		if s.Now().Sub(other.ai.spawnTime) < aiClanHelpMinAge {
			continue
		}
		if in := other.ai.intention; in != NpcIdle && in != NpcActive {
			continue
		}
		oc := s.aiClassOf(other)
		if !isClan(c, oc) || slices.Contains(oc.ignoreIDs, n.Template.ID) {
			continue
		}
		if _, ok := attacker.(*Player); ok {
			if s.canSee(other, attacker) {
				s.notifyAggression(other, attacker, 1)
			}
		} else if other.ai.intention != NpcAttack && s.canSee(other, attacker) {
			s.addDamageHate(other, attacker, 0, n.Hating(attacker))
			s.npcSetAttack(other, attacker)
		}
	}
}

// callClanOnKill is the clan call of Mobius Creature.doDie: the NPCs of the
// dead attackable's clan within its clan help range come for the killer.
func (s *Server) callClanOnKill(n *Npc, killer *Player) {
	c := s.aiClassOf(n)
	if len(c.clans) == 0 {
		return
	}
	for _, other := range s.clanCandidates(n, int32(c.clanHelpRange)) { //nolint:gosec // clan help ranges are small template values
		if other.Dead || !other.ai.attached || abs32(killer.Position().Z-other.Position().Z) > aiClanHelpMaxDZ {
			continue
		}
		if s.Now().Sub(other.ai.spawnTime) < aiClanHelpMinAge {
			continue
		}
		if in := other.ai.intention; in != NpcIdle && in != NpcActive {
			continue
		}
		if !isClan(c, s.aiClassOf(other)) {
			continue
		}
		s.notifyAggression(other, killer, 1)
	}
}

// clanCandidates returns the attackables within r of n (3D), collected
// before anyone acts.
func (s *Server) clanCandidates(n *Npc, r int32) []*Npc {
	var out []*Npc
	for o := range s.world.VisibleInRange(n, r) {
		if other, ok := o.(*Npc); ok && other.class != nil && other.class.attackable {
			out = append(out, other)
		}
	}
	return out
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
