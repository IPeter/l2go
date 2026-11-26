package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/summon"
	"github.com/IPeter/l2go/internal/world"
)

// Cubics (Mobius Cubic, SummonCubic, CubicMastery and the cubic tasks
// CubicAction, CubicHeal and CubicDisappear): a cubic skill gives a player
// a cubic for its template's duration; while the player or its summon is in
// its combat stance the cubic acts every delay on its own timer (a life
// cubic heals from the start), at most maxCount times; it uses its skills
// on the owner's target or heals the weakest of the owner's party. A player
// keeps one cubic, more with Cubic Mastery; death, an observer mode, a
// logout and a subclass switch clear them. UserInfo and CharInfo show them.

// cubic is one of a player's cubics (Mobius Cubic).
type cubic struct {
	owner *Player
	id    int
	tmpl  *data.CubicTemplate
	// power is the cubic's M.Atk (cubicPower or the template's), delay the
	// period of its action in milliseconds, chance its skill chance in 100,
	// maxCount the actions it has (-1 for no limit).
	power, delay, chance, maxCount int
	givenByOther                   bool
	skills                         []cubicSkill
	target                         Creature
	count                          int
	action, disappear              world.TimerID
}

// cubicSkill is a skill of a cubic with its data.
type cubicSkill struct {
	d  *data.CubicSkill
	sk *skills.Skill
}

// cubicMaxMagicRange is Cubic.MAX_MAGIC_RANGE: the reach of a cubic.
const cubicMaxMagicRange = 900

// cubicLifeID is Cubic.LIFE_CUBIC, which never waits for a stance; and
// cubicHealSkill Cubic.SKILL_CUBIC_HEAL.
const (
	cubicLifeID    = 3
	cubicHealSkill = 4051
)

// sumMsgAttackFailed is "Your attack has failed.".
const sumMsgAttackFailed serverpackets.SystemMessageID = 158

// setupCubics registers nothing of its own: cubics hang off the summon
// feature's hooks (summonDied, summonLeave) and the attack stance.
func (s *Server) setupCubics() error { return nil }

// CubicIDs returns the IDs of p's cubics in the order they came, for
// UserInfo and CharInfo. Loop-only.
func (s *Server) CubicIDs(p *Player) []int16 {
	o := s.summons.owners[p]
	if o == nil || len(o.cubics) == 0 {
		return nil
	}
	out := make([]int16, 0, len(o.cubics))
	for _, c := range o.cubics {
		out = append(out, int16(c.id)) //nolint:gosec // cubic IDs are small
	}
	return out
}

// summonCubic is Mobius SummonCubic.onStart for a living player p: the
// template of the cubic at the skill's level; a cubic of the same ID is
// replaced; otherwise the oldest go until another fits (Cubic Mastery); the
// new cubic starts and the players around see it (UserInfo, CharInfo).
func (s *Server) summonCubic(p *Player, givenByOther bool, sk *skills.Skill, prm skills.CubicParams) {
	if s.mounted(p) || s.summons.data == nil || prm.ID < 0 {
		return
	}
	level := skills.CubicTemplateLevel(prm.Level, sk.Level)
	tmpl, ok := s.summons.data.Cubic(prm.ID, level)
	if !ok {
		s.log.Warn("game: missing cubic template", "cubic", prm.ID, "level", level, "skill", sk.ID)
		return
	}
	o := s.summonOwnerOf(p)
	if s.cubicOf(p, prm.ID) != nil {
		s.removeCubic(p, prm.ID)
	} else {
		allowed := max(1, o.maxCubics)
		for len(o.cubics) >= allowed {
			s.removeCubic(p, o.cubics[0].id)
		}
	}
	c := &cubic{owner: p, id: prm.ID, tmpl: tmpl, chance: prm.SkillChance, givenByOther: givenByOther}
	c.power = prm.Power
	if c.power <= 0 {
		c.power = int(tmpl.Power)
	}
	c.delay = prm.Delay
	if c.delay <= 0 {
		c.delay = tmpl.Delay
	}
	c.delay *= 1000
	c.maxCount = prm.MaxCount
	if c.maxCount <= -1 {
		c.maxCount = tmpl.MaxCount
	}
	duration := prm.Duration
	if duration <= 0 {
		duration = tmpl.Duration
	}
	for i := range tmpl.Skills {
		cs := &tmpl.Skills[i]
		if csk, ok := s.Skill(cs.ID, cs.Level); ok {
			c.skills = append(c.skills, cubicSkill{d: cs, sk: csk})
		}
	}
	o.cubics = append(o.cubics, c)
	c.disappear = s.loop.After(time.Duration(duration)*time.Second, func() {
		c.disappear = 0
		s.removeCubic(p, c.id)
		p.BroadcastUserInfo()
	})
	if tmpl.TargetType == data.CubicTargetHeal {
		s.cubicDoAction(c)
	}
	p.BroadcastUserInfo()
}

// cubicOf returns p's cubic of the ID, nil without one.
func (s *Server) cubicOf(p *Player, id int) *cubic {
	o := s.summons.owners[p]
	if o == nil {
		return nil
	}
	for _, c := range o.cubics {
		if c.id == id {
			return c
		}
	}
	return nil
}

// removeCubic is Player.removeCubic: the cubic stops and goes.
func (s *Server) removeCubic(p *Player, id int) {
	o := s.summons.owners[p]
	if o == nil {
		return
	}
	i := slices.IndexFunc(o.cubics, func(c *cubic) bool { return c.id == id })
	if i < 0 {
		return
	}
	c := o.cubics[i]
	o.cubics = slices.Delete(o.cubics, i, i+1)
	s.cubicStopAction(c)
	if c.disappear != 0 {
		s.loop.Cancel(c.disappear)
		c.disappear = 0
	}
}

// cubicsStop is Player.stopCubics: every cubic goes; show sends UserInfo.
func (s *Server) cubicsStop(p *Player, show bool) {
	o := s.summons.owners[p]
	if o == nil || len(o.cubics) == 0 {
		return
	}
	for _, c := range slices.Clone(o.cubics) {
		s.removeCubic(p, c.id)
	}
	if show && p.InGame() {
		p.BroadcastUserInfo()
	}
}

// StopCubics clears p's cubics, as a subclass switch does (Mobius
// setActiveClass's stopCubics). Loop-only.
func (s *Server) StopCubics(p *Player) { s.cubicsStop(p, true) }

// cubicsOwnerDied clears a dead player's cubics (the cubic tasks remove
// them at their next run in Mobius).
func (s *Server) cubicsOwnerDied(p *Player) { s.cubicsStop(p, true) }

// cubicsAct is the cubic part of AttackStanceTaskManager.addAttackStanceTask:
// a playable entering its stance makes its owner's cubics act, but the life
// cubic.
func (s *Server) cubicsAct(c Creature) {
	p := s.actingPlayer(c)
	if p == nil {
		return
	}
	o := s.summons.owners[p]
	if o == nil {
		return
	}
	for _, cb := range o.cubics {
		if cb.id != cubicLifeID {
			s.cubicDoAction(cb)
		}
	}
}

// cubicDoAction is Cubic.doAction: the action task starts, its first run at
// once, unless it runs.
func (s *Server) cubicDoAction(c *cubic) {
	if c.action != 0 || c.tmpl == nil || len(c.skills) == 0 {
		return
	}
	run := func() { s.cubicActionTick(c) }
	if c.tmpl.TargetType == data.CubicTargetHeal {
		run = func() { s.cubicHealTick(c) }
	}
	c.action = s.loop.Every(time.Duration(c.delay)*time.Millisecond, run)
	run()
}

// cubicStopAction is Cubic.stopAction.
func (s *Server) cubicStopAction(c *cubic) {
	c.target = nil
	if c.action != 0 {
		s.loop.Cancel(c.action)
		c.action = 0
	}
}

// cubicSpent ends a cubic that used its actions: it stops, and a use-up
// cubic goes.
func (s *Server) cubicSpent(c *cubic) bool {
	if c.maxCount <= -1 || c.count < c.maxCount {
		return false
	}
	s.cubicStopAction(c)
	if c.tmpl.UseUp > 0 {
		s.removeCubic(c.owner, c.id)
		c.owner.BroadcastUserInfo()
	}
	return true
}

// cubicOwnerGone is the start of each cubic task: a dead or offline owner
// loses the cubic.
func (s *Server) cubicOwnerGone(c *cubic) bool {
	p := c.owner
	if p.IsDead() || !p.InGame() {
		s.removeCubic(p, c.id)
		if p.InGame() {
			p.BroadcastUserInfo()
		}
		return true
	}
	return false
}

// cubicActionTick is Mobius CubicAction.run: without the owner's (or its
// summon's) stance the cubic stops; a spent one stops (and goes); a
// priority skill that finds a target is used; otherwise on chance in 100 a
// skill by the selection mode.
func (s *Server) cubicActionTick(c *cubic) {
	if s.cubicOwnerGone(c) {
		return
	}
	p := c.owner
	if !s.InAttackStance(p) {
		sm := s.SummonOf(p)
		if sm == nil || !s.InAttackStance(sm.npc) {
			s.cubicStopAction(c)
			return
		}
	}
	if s.cubicSpent(c) {
		return
	}
	for i := range c.skills {
		if c.skills[i].d.Priority && s.cubicTryUse(c, &c.skills[i]) {
			c.count++
			return
		}
	}
	if s.rnd.Get(100) >= c.chance {
		return
	}
	if cs := s.cubicPickSkill(c); cs != nil && s.cubicTryUse(c, cs) {
		c.count++
	}
}

// cubicPickSkill is Cubic.getCubicSkill: by the template's selection mode,
// among the skills that are not priority ones.
func (s *Server) cubicPickSkill(c *cubic) *cubicSkill {
	var normal []*cubicSkill
	for i := range c.skills {
		if !c.skills[i].d.Priority {
			normal = append(normal, &c.skills[i])
		}
	}
	if len(normal) == 0 {
		return nil
	}
	rates := make([]int, len(normal))
	for i, cs := range normal {
		rates[i] = cs.d.TriggerRate
	}
	i := summon.PickCubicSkill(s.rnd, c.tmpl.SelectionMode, rates)
	if i < 0 {
		return nil
	}
	return normal[i]
}

// cubicTryUse is CubicAction.tryUseSkill: a living target and the skill
// lands.
func (s *Server) cubicTryUse(c *cubic, cs *cubicSkill) bool {
	target := s.cubicResolveTarget(c, cs)
	if target == nil || target.IsDead() {
		return false
	}
	s.cubicActivate(c, cs.sk, target)
	return true
}

// cubicResolveTarget is CubicAction.resolveTarget: by the template's target
// type (or the skill's for BY_SKILL): the weakest to heal, the owner, or the
// owner's valid enemy within the cubic's reach.
func (s *Server) cubicResolveTarget(c *cubic, cs *cubicSkill) Creature {
	typ := c.tmpl.TargetType
	if typ == data.CubicTargetBySkill {
		typ = cs.d.TargetType
	}
	switch typ {
	case data.CubicTargetHeal:
		c.target = s.cubicHealTarget(c)
	case data.CubicTargetMaster:
		c.target = c.owner
	default:
		c.target = s.cubicEnemy(c)
		if c.target != nil && !cubicInRange(c.owner, c.target) {
			c.target = nil
		}
	}
	return c.target
}

// cubicInRange is Cubic.isInCubicRange: within 900 (3D).
func cubicInRange(owner, target Creature) bool {
	if owner == nil || target == nil {
		return false
	}
	a, b := owner.Position(), target.Position()
	dx, dy, dz := int64(a.X-b.X), int64(a.Y-b.Y), int64(a.Z-b.Z)
	return dx*dx+dy*dy+dz*dz <= cubicMaxMagicRange*cubicMaxMagicRange
}

// cubicEnemy is Cubic.getCubicTarget without events, duels and the
// Olympiad: the owner's target, not itself or its summon; a living
// attackable that hates the owner or its summon; or, while the owner is
// flagged outside a peace zone or in a PvP zone, a flagged (or PvP zone)
// player outside a peace zone that is not of its party, clan or alliance.
func (s *Server) cubicEnemy(c *cubic) Creature {
	p := c.owner
	t, ok := p.Target.(Creature)
	if !ok || t == nil || t == Creature(p) {
		return nil
	}
	sm := s.SummonOf(p)
	if sm != nil && t == Creature(sm.npc) {
		return nil
	}
	if n, ok := t.(*Npc); ok && n.class != nil && n.class.attackable {
		if !n.Dead && (n.InAggroList(p) || (sm != nil && n.InAggroList(sm.npc))) {
			return t
		}
	}
	if !((p.PvPFlag > 0 && !p.Zones.Inside(ZonePeace)) || p.Zones.Inside(ZonePvP)) || t.IsDead() {
		return nil
	}
	enemy := s.actingPlayer(t)
	if enemy == nil || enemy == p {
		return nil
	}
	if pt := p.party.party; pt != nil && slices.Contains(pt.members, enemy) {
		return nil
	}
	if p.char.ClanID != 0 && !p.Zones.Inside(ZonePvP) {
		if enemy.char.ClanID == p.char.ClanID {
			return nil
		}
		if p.AllyID() > 0 && enemy.AllyID() == p.AllyID() {
			return nil
		}
	}
	if (enemy.PvPFlag == 0 && !enemy.Zones.Inside(ZonePvP)) || enemy.Zones.Inside(ZonePeace) || !enemy.InWorld() {
		return nil
	}
	return enemy
}

// cubicHealTarget is Cubic.cubicTargetForHeal: in a party the living
// member (or member's summon) within reach with the lowest HP ratio below
// full; alone the owner when hurt, or its summon when weaker.
func (s *Server) cubicHealTarget(c *cubic) Creature {
	p := c.owner
	var target Creature
	left := 1.0
	ratio := func(x Creature) float64 { return x.CurrentHP() / float64(x.MaxHP()) }
	if pt := p.party.party; pt != nil {
		for _, m := range pt.members {
			if !m.IsDead() && cubicInRange(p, m) && m.CurrentHP() < float64(m.MaxHP()) && left > ratio(m) {
				left, target = ratio(m), m
			}
			sm := s.SummonOf(m)
			if sm == nil || sm.npc.Dead || !cubicInRange(p, sm.npc) {
				continue
			}
			if sm.npc.CurrentHP() < float64(sm.npc.MaxHP()) && left > ratio(sm.npc) {
				left, target = ratio(sm.npc), sm.npc
			}
		}
		return target
	}
	if p.CurrentHP() < float64(p.MaxHP()) {
		left, target = ratio(p), p
	}
	if sm := s.SummonOf(p); sm != nil && !sm.npc.Dead && sm.npc.CurrentHP() < float64(sm.npc.MaxHP()) &&
		left > ratio(sm.npc) && cubicInRange(p, sm.npc) {
		target = sm.npc
	}
	return target
}

// cubicHealTick is Mobius CubicHeal.run: a spent cubic stops (and goes);
// the weakest gets Cubic Heal when it misses more HP than the skill's
// power, shown from the owner.
func (s *Server) cubicHealTick(c *cubic) {
	if s.cubicOwnerGone(c) || s.cubicSpent(c) {
		return
	}
	target := s.cubicHealTarget(c)
	c.target = target
	if target == nil || target.IsDead() {
		return
	}
	var heal *cubicSkill
	for i := range c.skills {
		if c.skills[i].sk.ID == cubicHealSkill {
			heal = &c.skills[i]
			break
		}
	}
	if heal == nil || float64(target.MaxHP())-target.CurrentHP() <= heal.sk.Power {
		return
	}
	owner := c.owner
	if t := s.skillCreatureOf(target); t != nil {
		heal.sk.Activate(s.skills.env, owner.skill, []skills.Creature{t})
	}
	s.world.Broadcast(owner, castUsePacket(owner, target, heal.sk.ID, heal.sk.Level, 0, 0), true)
	c.count++
}

// cubicActivate is CubicAction.activateSkill: MagicSkillUse from the owner,
// then by the skill's kind the continuous effects, magic damage, drain,
// disablers, damage over time, aggression, or the plain activation.
func (s *Server) cubicActivate(c *cubic, sk *skills.Skill, target Creature) {
	owner := c.owner
	s.world.Broadcast(owner, castUsePacket(owner, target, sk.ID, sk.Level, 0, 0), true)
	t := s.skillCreatureOf(target)
	if t == nil {
		return
	}
	env := s.skills.env
	switch {
	case sk.IsContinuous():
		s.cubicContinuous(c, sk, t)
	case sk.HasEffectType(skills.TypeMagicalAttack):
		s.cubicMagicDamage(c, sk, t, false)
	case sk.HasEffectType(skills.TypeHPDrain):
		s.cubicMagicDamage(c, sk, t, true)
	case sk.HasEffectType(skills.TypeStun, skills.TypeRoot, skills.TypeParalyze, skills.TypeAggression):
		s.cubicDisabler(c, sk, t)
	case sk.HasEffectType(skills.TypeDmgOverTime, skills.TypeDmgOverTimePercent):
		s.cubicContinuous(c, sk, t)
	default:
		sk.Activate(env, owner.skill, []skills.Creature{t})
	}
}

// cubicContinuous is Cubic.useCubicContinuous: a harmful skill must pass
// the cubic's roll ("Your attack has failed."), then its effects land.
func (s *Server) cubicContinuous(c *cubic, sk *skills.Skill, t skills.Creature) {
	if t.IsDead() {
		return
	}
	env := s.skills.env
	if sk.HasNegativeEffect() && !skills.CubicSkillSuccess(env, c.owner.skill, t, sk, stats.ShieldFailed) {
		c.owner.SendSystemMessage(sumMsgAttackFailed)
		return
	}
	sk.ApplyCubic(env, c.owner.skill, t)
}

// cubicMagicDamage is Cubic.useCubicMdam and useCubicDrain: the cubic's
// magic damage with the owner's critical rate and the target's shield; a
// drain gives the owner 40 % back; the target may break its attack or cast,
// and takes the damage from the owner, who is told.
func (s *Server) cubicMagicDamage(c *cubic, sk *skills.Skill, t skills.Creature, drain bool) {
	owner := c.owner.skill
	if t.IsAlikeDead() {
		if !drain && skills.IsPlayer(t) && t.AsPlayer().IsFakeDeath() && s.skills.env.Config.FakeDeathDamageStand {
			t.StopFakeDeath(true)
		} else {
			return
		}
	}
	env := s.skills.env
	mcrit := skills.MCrit(env, float64(owner.MCriticalHit(t, sk)))
	shld := skills.ShieldUse(env, owner, t, sk, true)
	damage := float64(int(skills.CubicMagicDamage(env, owner, t, sk, c.power, shld, mcrit)))
	if drain {
		p := c.owner
		p.SetCurrentHP(min(p.CurHP+0.4*damage, float64(p.MaxHP())))
		if damage > 0 && !t.IsDead() {
			t.ReduceCurrentHP(damage, owner, sk)
			if !t.IsRaid() && skills.AtkBreak(env, t, damage) {
				t.BreakAttack()
				t.BreakCast()
			}
			owner.SendDamageMessage(t, int(damage), mcrit, false, false)
		}
		return
	}
	if damage <= 0 {
		return
	}
	if !t.IsRaid() && skills.AtkBreak(env, t, damage) {
		t.BreakAttack()
		t.BreakCast()
	}
	if t.Calc(stats.VengeanceMdam, 0, t, sk) > float64(env.Rand.Get(100)) {
		return
	}
	owner.SendDamageMessage(t, int(damage), mcrit, false, false)
	t.ReduceCurrentHP(damage, owner, sk)
}

// cubicDisabler is Cubic.useCubicDisabler: a stun, root or paralysis that
// passes the cubic's roll lands; an aggression skill that passes makes an
// attackable hate the owner by 150 × power / (level + 7), and lands.
func (s *Server) cubicDisabler(c *cubic, sk *skills.Skill, t skills.Creature) {
	if t.IsDead() {
		return
	}
	env := s.skills.env
	owner := c.owner.skill
	if sk.HasEffectType(skills.TypeStun, skills.TypeParalyze, skills.TypeRoot) && skills.CubicSkillSuccess(env, owner, t, sk, stats.ShieldFailed) {
		sk.ApplyCubic(env, owner, t)
	}
	if sk.HasEffectType(skills.TypeAggression) && skills.CubicSkillSuccess(env, owner, t, sk, stats.ShieldFailed) {
		if n, ok := s.gameCreature(t).(*Npc); ok && n.class != nil && n.class.attackable {
			s.NotifyAggression(n, c.owner, int(150*sk.Power/float64(n.Level()+7)))
		}
		sk.ApplyCubic(env, owner, t)
	}
}
