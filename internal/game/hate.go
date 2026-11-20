package game

import (
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
)

// The hate list of an attackable (Mobius Attackable._aggroList and
// AggroInfo) and its attack-by list (Creature.getAttackByList). The hate
// decides whom a monster attacks; the damage is what rewards share out.

// aggroLimit caps hate and damage (Mobius AggroInfo.LIMIT).
const aggroLimit int64 = 1_000_000_000_000_000

// AggroInfo is one attacker on an NPC's hate list (Mobius AggroInfo): the
// damage it did and the hate the NPC feels for it.
type AggroInfo struct {
	Attacker Creature
	Damage   int64
	Hate     int64
}

// addHate adds to the hate, capped above (Mobius AggroInfo.addHate).
func (a *AggroInfo) addHate(v int64) { a.Hate = aggroAdd(a.Hate, v) }

// addDamage adds to the damage, capped above (Mobius AggroInfo.addDamage).
func (a *AggroInfo) addDamage(v int64) { a.Damage = aggroAdd(a.Damage, v) }

func aggroAdd(cur, v int64) int64 {
	switch {
	case v == 0:
		return cur
	case v > 0 && cur >= aggroLimit-v:
		return aggroLimit
	}
	return cur + v
}

// Aggro returns a copy of the NPC's hate list, in the order the attackers
// joined it. Rewards read the damage from it. Loop-only.
func (n *Npc) Aggro() []AggroInfo {
	out := make([]AggroInfo, len(n.ai.hate))
	copy(out, n.ai.hate)
	return out
}

// AttackBy returns a copy of the NPC's attack-by list: every creature that
// hit it since it spawned (Mobius getAttackByList). Loop-only.
func (n *Npc) AttackBy() []Creature {
	out := make([]Creature, len(n.ai.attackBy))
	copy(out, n.ai.attackBy)
	return out
}

// AddAttackBy adds c to the attack-by list unless it is there (Mobius
// Attackable.addAttackerToAttackByList). ReduceHP does it for every hit;
// the attack code calls it for skills. Loop-only.
func (n *Npc) AddAttackBy(c Creature) {
	if c == nil || c == Creature(n) || n.inAttackBy(c) {
		return
	}
	n.ai.attackBy = append(n.ai.attackBy, c)
}

func (n *Npc) inAttackBy(c Creature) bool {
	for _, a := range n.ai.attackBy {
		if a == c {
			return true
		}
	}
	return false
}

// aggroOf returns the entry of c on the hate list, or nil.
func (n *Npc) aggroOf(c Creature) *AggroInfo {
	for i := range n.ai.hate {
		if n.ai.hate[i].Attacker == c {
			return &n.ai.hate[i]
		}
	}
	return nil
}

// removeAggro takes c off the hate list, keeping the order of the others.
func (n *Npc) removeAggro(c Creature) {
	for i := range n.ai.hate {
		if n.ai.hate[i].Attacker == c {
			n.ai.hate = slices.Delete(n.ai.hate, i, i+1)
			return
		}
	}
}

// InAggroList reports whether c is on the hate list (Mobius isInAggroList).
func (n *Npc) InAggroList(c Creature) bool { return c != nil && n.aggroOf(c) != nil }

// ClearAggroList empties the hate list (Mobius clearAggroList).
func (n *Npc) ClearAggroList() {
	clear(n.ai.hate)
	n.ai.hate = n.ai.hate[:0]
}

// clearAttackBy empties the attack-by list.
func (n *Npc) clearAttackBy() {
	clear(n.ai.attackBy)
	n.ai.attackBy = n.ai.attackBy[:0]
}

// Hating returns the hate of the NPC for c (Mobius getHating). Like
// Mobius, it also cleans the list: a player that is invisible,
// invulnerable or spawn protected, and anyone out of the world or
// invisible, leaves it; a dead attacker's hate drops to 0. Loop-only.
func (n *Npc) Hating(c Creature) int64 {
	if len(n.ai.hate) == 0 || c == nil {
		return 0
	}
	a := n.aggroOf(c)
	if a == nil {
		return 0
	}
	if p, ok := c.(*Player); ok && (p.Invisible() || p.Teleporting() || p.admin.invul || (n.srv != nil && n.srv.SpawnProtected(p))) {
		n.removeAggro(c)
		return 0
	}
	if !creatureInWorld(c) || creatureInvisible(c) {
		n.removeAggro(c)
		return 0
	}
	if c.IsDead() {
		a.Hate = 0
		return 0
	}
	return a.Hate
}

// StopHating sets the hate for c to 0, keeping it on the list (Mobius
// stopHating). Loop-only.
func (n *Npc) StopHating(c Creature) {
	if a := n.aggroOf(c); a != nil {
		a.Hate = 0
	}
}

// MostHated returns the attacker the NPC hates most, or nil (Mobius
// getMostHated). Hate checks drop the hate of attackers that are dead,
// gone or outside the NPC's 3×3×3 regions (AggroInfo.checkHate). The first
// of equal hates wins. Loop-only.
func (n *Npc) MostHated() Creature {
	if len(n.ai.hate) == 0 || n.Dead {
		return nil
	}
	var most Creature
	var maxHate int64
	for i := range n.ai.hate {
		a := &n.ai.hate[i]
		if n.checkHate(a) > maxHate {
			most, maxHate = a.Attacker, a.Hate
		}
	}
	return most
}

// checkHate is Mobius AggroInfo.checkHate.
func (n *Npc) checkHate(a *AggroInfo) int64 {
	c := a.Attacker
	if c == nil || c.IsDead() || !creatureInWorld(c) {
		a.Hate = 0
		return 0
	}
	if !inSurroundingRegion(n, c) {
		a.Hate = 0
	}
	return a.Hate
}

// AddDamageHate adds damage and hate for attacker (Mobius
// Attackable.addDamageHate): a player that gets no hate gets 1 and wakes
// the NPC, anyone else gets 1 for 0; any hate wakes an idle NPC. Loop-only.
func (n *Npc) AddDamageHate(attacker Creature, damage, aggro int64) {
	if n.srv == nil {
		return
	}
	n.srv.addDamageHate(n, attacker, damage, aggro)
}

func (s *Server) addDamageHate(n *Npc, attacker Creature, damage, aggro int64) {
	if attacker == nil || attacker == Creature(n) {
		return
	}
	a := n.aggroOf(attacker)
	if a == nil {
		n.ai.hate = append(n.ai.hate, AggroInfo{Attacker: attacker})
		a = &n.ai.hate[len(n.ai.hate)-1]
	}
	a.addDamage(damage)
	a.addHate(aggro)
	_, isPlayer := attacker.(*Player)
	switch {
	case isPlayer && aggro == 0:
		s.addDamageHate(n, attacker, 0, 1)
		if s.aiOf(n).intention == NpcIdle {
			s.npcSetActive(n)
		}
	case !isPlayer && aggro == 0:
		aggro = 1
		a.addHate(1)
	}
	if aggro != 0 && s.aiOf(n).intention == NpcIdle {
		s.npcSetActive(n)
	}
}

// npcDamaged is what an attack on n does before its HP go down (Mobius
// Attackable.reduceCurrentHp, addDamage and NpcStatus.reduceHp): the AI of
// an attackable hears of the attack and hates the attacker by the damage,
// and the attacker joins the attack-by list.
func (s *Server) npcDamaged(n *Npc, attacker Creature, value float64) {
	if n.class == nil || !n.class.attackable {
		return
	}
	damage := int(javaInt32(value))
	if !n.Dead {
		s.notifyAttacked(n, attacker)
		hate := stats.DamageHate(damage, n.Level(), s.hateRatio(n, attacker))
		s.addDamageHate(n, attacker, int64(damage), int64(hate))
		s.minionsAssist(n, attacker) // minion.go: Attackable.reduceCurrentHp's onAssist
	}
	if !n.Dead {
		n.AddAttackBy(attacker)
	}
}

// hateRatio is Mobius Attackable.getHateRatio: the template's
// SetHateGroup, SetHateOccupation and SetHateRace parameters raise the hate
// for players of a class category, a class or a race.
func (s *Server) hateRatio(n *Npc, attacker Creature) float64 {
	p, ok := attacker.(*Player)
	if !ok {
		return 1
	}
	c := s.aiClassOf(n)
	percent := 0
	if c.hateGroupOK && s.items.categories != nil && s.items.categories.InCategory(c.hateGroup, p.ClassID()) {
		percent += c.hateGroupRatio
	}
	if c.hateOccupation >= 0 && p.ClassID() == c.hateOccupation {
		percent += c.hateOccupationRatio
	}
	if c.hateRaceOK && p.Race() == int(c.hateRace) {
		percent += c.hateRaceRatio
	}
	return stats.HateRatio(percent)
}

// hateParams reads the hate parameters of a template.
func (c *aiClass) hateParams(t *data.NpcTemplate) {
	c.hateOccupation = -1
	p := t.Parameters.Params
	if len(p) == 0 {
		return
	}
	if v, ok := p["SetHateGroup"]; ok {
		c.hateGroup, c.hateGroupOK = data.ParseCategoryType(v)
	}
	c.hateGroupRatio = npcParamInt(p, "SetHateGroupRatio", 0)
	c.hateOccupation = npcParamInt(p, "SetHateOccupation", -1)
	c.hateOccupationRatio = npcParamInt(p, "SetHateOccupationRatio", 0)
	if v, ok := p["SetHateRace"]; ok {
		c.hateRace, c.hateRaceOK = data.ParseRace(v)
	}
	c.hateRaceRatio = npcParamInt(p, "SetHateRaceRatio", 0)
}
