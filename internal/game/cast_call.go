package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/world"
)

// raidCurseLevelGap is how many levels above a raid boss a creature may be
// before the boss curses it (Mobius getLevel() > raid.getLevel() + 8).
const raidCurseLevelGap = 8

// callSkill is Mobius Creature.callSkill, where a cast lands (K4, K11):
//
//   - A toggle that is already on does nothing; no targets, nothing.
//   - A caster 9 levels above a raid boss among the targets, or helping a
//     creature that fights such a boss, is cursed by it instead
//     (DisableRaidCurse is off).
//   - The skill's effects land (Skill.Activate: reflection, the SELF
//     effects, the shots spent).
//   - A player's negative skill on a raid boss flags it
//     (FlagPlayerOnRaidAttack); on a playable it starts the target's stance
//     and flags the caster (UpdatePvPStatus); an attackable gets the caster
//     on its attack-by list (not for Lure and Temptation). A positive skill
//     on a flagged or karma player, or on an attackable, flags the caster.
//   - Attackables within 1,000 of a player that fight one of the targets
//     hate the healer by the effect point; a raid boss in combat curses a
//     player 9 levels above it.
//   - Negative skills (but hate effects) tell the targets' AI of the attack.
//
// The option trigger skills of augmentations fire per target (augment.go).
// Over-hit (a skill marks the attackable), the weapon's trigger skills,
// events and the skill-see scripts are not ported here.
func (s *Server) callSkill(c Creature, b *skillBase, sk *skills.Skill, targets []skills.Creature) {
	if sk.IsToggle() && b.el.IsAffectedBySkill(sk.ID) {
		return
	}
	if len(targets) == 0 {
		return
	}
	for _, t := range targets {
		gt := s.gameCreature(t)
		if gt == nil {
			continue
		}
		if s.castRaidCurse(c, b, sk, gt) {
			return
		}
		s.augmentSkillTriggers(c, sk, gt) // augment.go: the option trigger skills
	}
	env := s.skills.env
	sk.Activate(env, b.self, targets)

	p, _ := c.(*Player)
	if p != nil && sk.EffectPoint <= 0 && sk.HasNegativeEffect() {
		for _, t := range targets {
			if gt := s.gameCreature(t); gt != nil && creatureRaid(gt) {
				s.UpdatePvPStatusNoTarget(p)
				break
			}
		}
	}
	if p != nil {
		for _, t := range targets {
			gt := s.gameCreature(t)
			if gt == nil {
				continue
			}
			s.castFlagAndNotify(p, sk, t, gt)
		}
		s.castSkillSeen(p, sk, targets)
	}
	if sk.HasNegativeEffect() && !sk.HasEffectType(skills.TypeHate) {
		for _, t := range targets {
			if gt := s.gameCreature(t); gt != nil {
				s.NotifyAttacked(gt, c)
			}
		}
	}
}

// castFlagAndNotify is the part of callSkill that flags a player caster
// and tells each target of the attack.
func (s *Server) castFlagAndNotify(p *Player, sk *skills.Skill, t skills.Creature, gt Creature) {
	if sk.EffectPoint > 0 {
		switch v := gt.(type) {
		case *Player:
			if v != p && (v.PvPFlag > 0 || v.Karma() > 0) {
				s.UpdatePvPStatusNoTarget(p)
			}
		case *Npc:
			if skills.IsAttackable(t) {
				s.UpdatePvPStatusNoTarget(p)
			}
		}
		return
	}
	switch {
	case skills.IsPlayable(t) && sk.HasNegativeEffect():
		if gt != Creature(p) {
			if tp, ok := gt.(*Player); ok {
				s.StartAutoAttack(tp)
			}
			if sk.EffectPoint != 0 || sk.AffectRange <= 0 {
				s.UpdatePvPStatus(p, gt)
			}
		}
	case skills.IsAttackable(t):
		if n, ok := gt.(*Npc); ok && sk.ID != skillLure && sk.ID != skillTemptation {
			n.AddAttackBy(p)
		}
	}
	if sk.HasNegativeEffect() && !sk.HasEffectType(skills.TypeHate) && sk.AbnormalType != data.AbnormalTurnPassive {
		s.NotifyAttacked(gt, p)
	}
}

// castSkillSeen is callSkill's "mobs in range 1000 see spell": an
// attackable that fights one of the targets of a positive skill (or is one
// of them) hates p by effectPoint × 150 / (its level + 7) × its hate ratio;
// a raid boss in combat curses a player more than 8 levels above it for any
// skill but a static one.
func (s *Server) castSkillSeen(p *Player, sk *skills.Skill, targets []skills.Creature) {
	if !p.InWorld() {
		return
	}
	var npcs []*Npc
	for o := range s.world.VisibleInRange(p, castSkillSeeRange) {
		if n, ok := o.(*Npc); ok && n.class != nil && n.class.attackable {
			npcs = append(npcs, n)
		}
	}
	for _, n := range npcs {
		if ep := sk.EffectPoint; ep > 0 && n.ai.attached && n.ai.intention == NpcAttack {
			for _, t := range targets {
				gt := s.gameCreature(t)
				if gt == nil {
					continue
				}
				if (n.ai.target != nil && n.ai.target == world.Object(gt)) || gt == Creature(n) {
					aggro := int64(float64((ep*150)/(n.Level()+7)) * s.hateRatio(n, p))
					n.AddDamageHate(p, 0, aggro)
				}
			}
		}
		if !sk.IsStatic() && s.raidCurses(n) && s.InCombat(n) && p.Level()-n.Level() > raidCurseLevelGap {
			id := skillRaidCurse
			if sk.HasNegativeEffect() {
				id = skillRaidCurse2
			}
			s.applyRaidCurse(p, n, id)
		}
	}
}

// castRaidCurse is callSkill's raid curse check for one target: a caster
// more than 8 levels above a raid boss target, or above the raid boss a
// target of its positive skill fights (when the target is on the boss's
// attack-by list), is cursed: silence for magic, petrification otherwise.
// It reports whether the cast stops there.
func (s *Server) castRaidCurse(c Creature, b *skillBase, sk *skills.Skill, gt Creature) bool {
	level := b.Level()
	cursed := s.raidCurses(gt) && level > creatureLevel(gt)+raidCurseLevelGap
	if !cursed && !sk.HasNegativeEffect() {
		for _, r := range s.castFightTargets(gt) {
			n, ok := r.(*Npc)
			if ok && s.raidCurses(n) && n.inAttackBy(gt) && level > n.Level()+raidCurseLevelGap {
				cursed = true
				break
			}
		}
	}
	if !cursed {
		return false
	}
	id := skillRaidCurse2
	if sk.IsMagic() {
		id = skillRaidCurse
	}
	s.applyRaidCurse(c, gt, id)
	return true
}

// castFightTargets returns the attack target and the cast target of c's
// AI (Mobius getAI().getAttackTarget() and getCastTarget()).
func (s *Server) castFightTargets(c Creature) []Creature {
	var out []Creature
	switch v := c.(type) {
	case *Player:
		if t := s.AttackTarget(v); t != nil {
			out = append(out, t)
		}
		if ai := s.attack.players[v]; ai != nil && ai.intention == playerCasting {
			if t := v.skill.cast.target; t != nil {
				out = append(out, t)
			}
		}
	case *Npc:
		if v.ai.attackTarget != nil {
			out = append(out, v.ai.attackTarget)
		}
	}
	return out
}

// creatureLevel returns a creature's level.
func creatureLevel(c Creature) int {
	switch v := c.(type) {
	case *Player:
		return v.Level()
	case *Npc:
		return v.Level()
	}
	return 0
}

// applyRaidCurse curses c from raid (Mobius: abortAttack, abortCast,
// setIntentionIdle, then the curse skill's effects with the boss as their
// caster).
func (s *Server) applyRaidCurse(c, raid Creature, id int) {
	curse, ok := s.Skill(id, 1)
	if !ok {
		s.log.Warn("game: the raid curse skill is missing", "skillId", id)
		return
	}
	s.AbortAttack(c)
	s.abortCast(c)
	if p, ok := c.(*Player); ok {
		s.playerIntendIdle(p)
	}
	from, to := s.skillCreatureOf(raid), s.skillCreatureOf(c)
	if from != nil && to != nil {
		curse.ApplyTo(s.skills.env, from, to)
	}
}
