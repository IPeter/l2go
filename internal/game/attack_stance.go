package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
)

// The combat stance (Mobius AttackStanceTaskManager and AbstractAI's
// clientStartAutoAttack/clientStopAutoAttack): a creature that attacks or is
// attacked shows its combat stance (AutoAttackStart) and keeps it until 15 s
// after its last blow; a scan every second ends it with AutoAttackStop. A
// player in its stance cannot restart or log out (attack_logout.go).

const (
	// attackStanceTime is how long a stance lasts after its last start
	// (Mobius COMBAT_TIME); the scan ends it once more time passed.
	attackStanceTime = 15 * time.Second
	// attackStanceScanPeriod is how often the scan runs.
	attackStanceScanPeriod = time.Second
)

// StartAutoAttack shows c's combat stance (Mobius clientStartAutoAttack): the
// first time c and the players around get AutoAttackStart; every call
// restarts the 15 s of the stance. NPCs that are not attackable never enter
// it. Loop-only.
func (s *Server) StartAutoAttack(c Creature) {
	if n, ok := c.(*Npc); ok && (n.class == nil || !n.class.attackable) && n.summon == nil { // summons are playables (summon.go)
		return
	}
	f := &s.attack
	if !f.autoAttacking[c] {
		if attackerInWorld(c) {
			s.world.Broadcast(c, serverpackets.NewAutoAttackStartPacket(int32(c.ID())), true)
		}
		f.autoAttacking[c] = true
	}
	s.addStance(c)
}

// StopAutoAttack is Mobius clientStopAutoAttack: a player keeps its stance
// until the scan ends it (a player that shows the stance without one gets
// one); another creature leaves it at once with AutoAttackStop. Loop-only.
func (s *Server) StopAutoAttack(c Creature) {
	f := &s.attack
	if _, ok := c.(*Player); ok {
		if !s.InAttackStance(c) && f.autoAttacking[c] {
			s.addStance(c)
		}
		return
	}
	if f.autoAttacking[c] {
		if attackerInWorld(c) {
			s.world.Broadcast(c, serverpackets.NewAutoAttackStopPacket(int32(c.ID())), true)
		}
		delete(f.autoAttacking, c)
	}
}

// InCombat is Mobius isInCombat, the in-combat flag of CharInfo and
// NpcInfo: c has an attack target or shows its stance. An NPC's attack
// target is its AI's, so for NPCs only the stance counts. Loop-only.
func (s *Server) InCombat(c Creature) bool {
	if s.attack.autoAttacking[c] {
		return true
	}
	p, ok := c.(*Player)
	return ok && s.AttackTarget(p) != nil
}

// InAttackStance reports whether c is in its combat stance (Mobius
// hasAttackStanceTask): within 15 to 16 s of its last blow. Loop-only.
func (s *Server) InAttackStance(c Creature) bool {
	_, ok := s.attack.stance[c]
	return ok
}

// addStance starts or restarts c's stance (Mobius addAttackStanceTask).
func (s *Server) addStance(c Creature) {
	s.cubicsAct(c) // cubic.go: a playable's cubics start acting
	f := &s.attack
	if _, ok := f.stance[c]; !ok {
		f.stanceOrder = append(f.stanceOrder, c)
	}
	f.stance[c] = s.Now()
}

// dropStance forgets c's stance and stance display without packets (c left
// the game).
func (s *Server) dropStance(c Creature) {
	f := &s.attack
	delete(f.stance, c)
	delete(f.autoAttacking, c)
}

// scanStances is the stance scan (Mobius AttackStanceTaskManager.run, every
// second): a stance that started more than 15 s ago ends, and c and the
// players around get AutoAttackStop, in the order the stances began.
func (s *Server) scanStances() {
	f := &s.attack
	if len(f.stanceOrder) == 0 {
		return
	}
	now := s.Now()
	f.stanceOrder = slices.DeleteFunc(f.stanceOrder, func(c Creature) bool {
		started, ok := f.stance[c]
		if !ok {
			return true
		}
		if now.Sub(started) <= attackStanceTime {
			return false
		}
		delete(f.stance, c)
		delete(f.autoAttacking, c)
		if attackerInWorld(c) {
			s.world.Broadcast(c, serverpackets.NewAutoAttackStopPacket(int32(c.ID())), true)
		}
		return true
	})
}
