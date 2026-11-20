package game

import (
	"cmp"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// Loading and saving skills, effects and reuse (K12, K13), and the effects
// at death, teleport and leave.
//
//   - Character selection (loadSkills, Mobius Player.load): the saved
//     skills of the active class in skill ID order (the order Mobius's
//     table gives them), the auto-get skills (rewardSkills), then the HP,
//     MP and CP within the maxima the passives give.
//   - EnterWorld (skillsEnter, Mobius onPlayerEnter → restoreEffects): the
//     reuse stamps still running, then the saved effects with their time
//     left and the player as their caster, without "flow through you".
//   - Saving (Player.Snapshot → storeSkills): the stored skills, and the
//     effects and reuse as Mobius storeEffect writes them.
//   - Leaving: the effects to save are frozen, then every effect ends
//     quietly and the timers stop (Mobius deleteMe).

// classIndex is the class index of the active class: 0 until subclasses
// exist (decision X3).
const classIndex = 0

// storedSkill is a row of character_skills of the active class.
type storedSkill struct{ id, level int32 }

// storeSkill adds or updates the saved row of a skill (Mobius storeSkill).
func (a *skillPlayer) storeSkill(id, level int) {
	for i := range a.stored {
		if a.stored[i].id == int32(id) { //nolint:gosec // skill IDs are small
			a.stored[i].level = int32(level) //nolint:gosec // skill levels are small
			return
		}
	}
	a.stored = append(a.stored, storedSkill{id: int32(id), level: int32(level)}) //nolint:gosec // small
}

// unstoreSkill deletes the saved row of a skill; it reports whether there
// was one.
func (a *skillPlayer) unstoreSkill(id int) bool {
	n := len(a.stored)
	a.stored = slices.DeleteFunc(a.stored, func(r storedSkill) bool { return r.id == int32(id) }) //nolint:gosec // small
	return len(a.stored) != n
}

// loadSkills gives a selected player its skills, as Mobius Player.load
// does before it sets the HP: the saved skills of the active class (a level
// the data does not have is skipped but kept saved, as Mobius keeps its
// row), then the auto-get skills (with AutoLearnSkills every skill its class
// can have at its level, without packets); then the HP, MP and CP of the character
// within the maxima the passives give. It runs on the loop, before
// CharSelected shows them.
func (s *Server) loadSkills(p *Player) {
	a := p.skill
	rows := slices.Clone(p.char.Skills)
	slices.SortStableFunc(rows, func(x, y storage.CharacterSkill) int { return cmp.Compare(x.ID, y.ID) })
	for _, r := range rows {
		if r.ClassIndex != classIndex {
			continue
		}
		a.stored = append(a.stored, storedSkill{id: r.ID, level: r.Level})
		if sk, ok := s.Skill(int(r.ID), int(r.Level)); ok {
			a.addSkill(sk)
		} else if s.skills.catalog != nil {
			s.log.Warn("game: skipping a saved skill the data does not have", append(p.logArgs(), "skill", r.ID, "level", r.Level)...)
		}
	}
	if s.settings.AutoLearnSkills {
		s.learnAvailableSkills(p, false, false, false) // rewardSkills (skill_known.go)
	} else {
		s.giveAutoGetSkills(p)
	}
	if !p.IsGM() {
		s.checkPlayerSkills(p) // the rest of rewardSkills (skill_learn.go)
	}
	v := p.stats.View()
	p.lastView = v
	p.CurHP = clampVital(p.char.CurHP, v.MaxHP)
	p.CurMP = clampVital(p.char.CurMP, v.MaxMP)
	p.CurCP = clampVital(p.char.CurCP, v.MaxCP)
}

// storeSkills puts what a save keeps of p's skills into c: the rows of
// character_skills (other classes' rows as loaded), and those of
// character_skills_save (the frozen ones of a leaving player, else the live
// effects and reuse).
func (p *Player) storeSkills(c *storage.Character) {
	a := p.skill
	if a == nil {
		return
	}
	var rows []storage.CharacterSkill
	for _, r := range p.char.Skills {
		if r.ClassIndex != classIndex {
			rows = append(rows, r)
		}
	}
	for _, r := range a.stored {
		rows = append(rows, storage.CharacterSkill{ClassIndex: classIndex, ID: r.id, Level: r.level})
	}
	c.Skills = rows
	saved := a.saved
	if saved == nil && p.state == stateInGame && a.entered {
		saved = p.srv.effectsToSave(p)
	}
	if !a.entered {
		// Not restored yet: keep what was loaded.
		c.SavedSkills = p.char.SavedSkills
		return
	}
	var out []storage.SavedSkill
	for _, r := range p.char.SavedSkills {
		if r.ClassIndex != classIndex {
			out = append(out, r)
		}
	}
	c.SavedSkills = append(out, saved...)
}

// effectsToSave is Mobius storeEffect(true): the effects to keep
// (EffectList.ToStore) with their seconds left and the reuse of their skill
// when it still runs, then the other skills under reuse.
func (s *Server) effectsToSave(p *Player) []storage.SavedSkill {
	a := p.skill
	now := s.Now()
	var out []storage.SavedSkill
	seen := make(map[int]bool)
	for _, info := range a.el.ToStore() {
		sk := info.Skill()
		h := sk.ReuseHash()
		seen[h] = true
		row := storage.SavedSkill{
			ClassIndex:    classIndex,
			SkillID:       int32(sk.ID),       //nolint:gosec // skill IDs are small
			SkillLevel:    int32(sk.Level),    //nolint:gosec // skill levels are small
			RemainingTime: int32(info.Time()), //nolint:gosec // seconds of an effect
			RestoreType:   storage.RestoreEffect,
		}
		if r, ok := a.reuse.byHash[h]; ok && r.end.After(now) {
			row.ReuseDelay, row.ReuseEnd = r.delay.Milliseconds(), r.end.UnixMilli()
		}
		out = append(out, row)
	}
	for _, h := range a.reuse.order.keys() {
		r := a.reuse.byHash[h]
		if seen[h] || !r.end.After(now) {
			continue
		}
		seen[h] = true
		out = append(out, storage.SavedSkill{
			ClassIndex:    classIndex,
			SkillID:       int32(r.id),    //nolint:gosec // skill IDs are small
			SkillLevel:    int32(r.level), //nolint:gosec // skill levels are small
			RemainingTime: -1,
			ReuseDelay:    r.delay.Milliseconds(),
			ReuseEnd:      r.end.UnixMilli(),
			RestoreType:   storage.RestoreReuse,
		})
	}
	return out
}

// skillsEnter is the OnPlayerEnter hook of the skills, Mobius
// restoreEffects in onPlayerEnter: the reuse stamps that still run more
// than 10 ms, then each saved effect with its time left (applyEffects with
// the player as caster and target). The rows are used up, as Mobius deletes
// them. Afterwards the player has entered the world (HasEnteredWorld), so
// later effects tell it they flow through it.
func (s *Server) skillsEnter(p *Player) {
	a := p.skill
	env := s.skills.env
	now := s.Now()
	var keep []storage.SavedSkill
	for _, r := range p.char.SavedSkills {
		if r.ClassIndex != classIndex {
			keep = append(keep, r)
			continue
		}
		sk, ok := s.Skill(int(r.SkillID), int(r.SkillLevel))
		if !ok {
			continue
		}
		end := time.UnixMilli(r.ReuseEnd)
		if end.Sub(now) > 10*time.Millisecond {
			a.addReuse(sk, time.Duration(r.ReuseDelay)*time.Millisecond, end)
		}
		if r.RestoreType != storage.RestoreEffect {
			continue
		}
		sk.ApplyEffects(env, a, a, false, false, false, int(r.RemainingTime))
	}
	p.char.SavedSkills = keep
	a.entered = true
}

// skillsLeave is the OnPlayerLeave hook of the skills: the effects to save
// are frozen for the snapshot leave takes next, a cast stops, and the icon
// timer and the buff clock stop. The effects end once the player is out
// (Mobius deleteMe: stopAllEffectsWithoutExclusions without updates), so
// their end messages reach no client.
func (s *Server) skillsLeave(p *Player) {
	a := p.skill
	if a.entered {
		a.saved = s.effectsToSave(p)
		if a.saved == nil {
			a.saved = []storage.SavedSkill{}
		}
	}
	if a.listTimer != 0 {
		s.loop.Cancel(a.listTimer)
		a.listTimer = 0
	}
	a.AbortCast()
	a.el.CancelTimers()
	s.loop.After(0, func() {
		a.el.StopAllWithoutExclusions(false, false)
		a.el.CancelTimers()
	})
}

// skillsTeleport is the OnTeleport hook of the skills: a teleport stops a
// cast (Mobius teleToLocation: abortCast).
func (s *Server) skillsTeleport(p *Player) { p.skill.AbortCast() }

// skillsDie is the OnDie hook of the skills (Mobius doDie): a cast stops; a
// player keeps only the effects that last through death; an attackable that
// respawns loses its effects but not its passives, one that does not loses
// everything; another NPC keeps the effects that last through death.
func (s *Server) skillsDie(victim, _ Creature) {
	b := s.skillBaseOf(victim)
	if b == nil {
		return
	}
	b.AbortCast()
	if b.p != nil {
		if a, ok := b.self.(*skillPlayer); ok {
			a.fakeDeath = false
		}
		b.el.StopAllExceptStayAfterDeath()
		b.p.BroadcastUserInfo() // Player.stopAllEffectsExceptThoseThatLastThroughDeath
		return
	}
	n := b.n
	switch {
	case n.class != nil && n.class.attackable && n.spawn != nil:
		b.el.StopAll()
	case n.class != nil && n.class.attackable:
		b.el.StopAllWithoutExclusions(true, true)
	default:
		b.el.StopAllExceptStayAfterDeath()
	}
}

// npcSkillsGone ends the effects of an NPC that leaves the world: one that
// respawns keeps its passives (Mobius initializeNpc stops the others), one
// gone for good loses everything and its timers stop (Mobius deleteMe).
func (s *Server) npcSkillsGone(n *Npc, forGood bool) {
	if n.skill == nil {
		return
	}
	el := n.skill.el
	if !forGood {
		el.StopAll()
		return
	}
	el.StopAllWithoutExclusions(false, false)
	el.CancelTimers()
}

// skillsHit is the OnHit hook of the skills: a hit ends the target's
// effects that end on damage (Mobius reduceCurrentHp → stopEffectsOnDamage).
func (s *Server) skillsHit(target, _ Creature, _ int) {
	if b := s.skillBaseOf(target); b != nil {
		b.el.StopOnDamage(true)
	}
}

// skillsAttacked is the OnAttacked hook of the skills: the target's damage
// listeners hear of the hit (OnCreatureDamageReceived).
func (s *Server) skillsAttacked(target, attacker Creature, damage int) {
	if b := s.skillBaseOf(target); b != nil && len(b.listeners) > 0 {
		b.NotifyDamageReceived(float64(damage), s.skillCreatureOf(attacker), nil, false, false)
	}
}
