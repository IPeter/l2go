package game

import (
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/world"
)

// A player's casts: RequestMagicSkillUse, useMagic with its queue and
// checks, the CAST intention (PlayerAI) and Esc.

// handleRequestMagicSkillUse handles RequestMagicSkillUse (0x2F), Mobius
// RequestMagicSkillUse: a dead player gets ActionFailed; one faking death
// may only use Fake Death again ("You cannot move while sitting." and
// ActionFailed otherwise); an unknown skill gets ActionFailed. The request
// ends the spawn protection; a continuous self buff stops the player where
// it stands; then useMagic. Mounts and the karma teleport rule do not exist
// yet (AltKarmaPlayerCanTeleport is on).
func (s *Server) handleRequestMagicSkillUse(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestMagicSkillUse(body)
	if err != nil {
		s.log.Debug("game: bad RequestMagicSkillUse", append(p.logArgs(), "err", err)...)
		return
	}
	if p.dead {
		p.SendActionFailed()
		return
	}
	if p.skill.fakeDeath {
		if req.SkillID != skillFakeDeath {
			p.SendSystemMessage(sysMsgCannotMoveSitting)
			p.SendActionFailed()
			return
		}
		if fd, ok := s.Skill(skillFakeDeath, 1); ok && s.skillDisabled(&p.skill.skillBase, fd) {
			p.SendActionFailed()
			return
		}
	}
	sk := p.KnownSkill(int(req.SkillID))
	if sk == nil {
		p.SendActionFailed()
		return
	}
	s.onActionRequest(p)
	if sk.IsContinuous() && !sk.IsDebuff && sk.TargetType == data.TargetSelf {
		s.castStopForSelfBuff(p)
	}
	s.useMagic(p, sk, req.CtrlPressed, req.ShiftPressed)
}

// castStopForSelfBuff is RequestMagicSkillUse's setIntentionMoveTo(the
// player's own location) before a continuous self buff (PlayerAI): a
// sitting player gets ActionFailed; during a swing or a cast the move waits
// (ActionFailed); otherwise the player stops where it stands, as a walk of
// no length does (MOVE_TO arrives at once: IDLE, StopMove when it walked,
// ActionFailed).
func (s *Server) castStopForSelfBuff(p *Player) {
	if p.Sitting {
		p.SendActionFailed()
		return
	}
	if !s.playerIntendMove(p, p.Position()) {
		return
	}
	s.AbortAttack(p)
	if !(p.stats.MoveSpeed() > 0) || p.skill.IsMovementDisabled() {
		p.SendActionFailed()
		return
	}
	s.playerIntendIdle(p)
	p.SendActionFailed()
}

// UseMagic makes p use sk as its skill button does (Mobius Player.useMagic):
// ctrl forces a skill on a target that may not be attacked otherwise, shift
// casts without walking into range. During a cast another skill is queued
// for its end. It reports whether the cast intention started. Item skills
// (Phase 7.6) use it for skills that are not cast simultaneously.
// Loop-only.
func (s *Server) UseMagic(p *Player, sk *skills.Skill, ctrl, shift bool) bool {
	if p == nil || sk == nil || p.skill == nil {
		return false
	}
	return s.useMagic(p, sk, ctrl, shift)
}

// useMagic is Mobius Player.useMagic: a passive gets ActionFailed; during a
// cast the same skill or a disabled one gets ActionFailed and another one
// is queued (ActionFailed too). Otherwise the player casts: the skill is
// current, the queue empties, checkUseMagicConditions decides, and the cast
// intention starts with the player as the target of self and aura skills,
// or the first target of the list.
func (s *Server) useMagic(p *Player, sk *skills.Skill, ctrl, shift bool) bool {
	if sk.IsPassive() {
		p.SendActionFailed()
		return false
	}
	st := &p.skill.cast
	if st.casting {
		switch {
		case st.current != nil && sk.ID == st.current.skill.ID:
			p.SendActionFailed()
			return false
		case s.skillDisabled(&p.skill.skillBase, sk):
			p.SendActionFailed()
			return false
		}
		st.queued = &skillUse{skill: sk, ctrl: ctrl, shift: shift}
		p.SendActionFailed()
		return false
	}
	st.casting = true
	st.current = &skillUse{skill: sk, ctrl: ctrl, shift: shift}
	st.queued = nil
	if !s.checkUseMagicConditions(p, sk, ctrl, shift) {
		st.casting = false
		return false
	}
	var target Creature
	switch sk.TargetType {
	case data.TargetAura, data.TargetFrontAura, data.TargetBehindAura, data.TargetGround, data.TargetSelf,
		data.TargetAuraCorpseMob, data.TargetCommandChannel, data.TargetAuraFriendly:
		target = p
	default:
		target = s.gameCreature(sk.FirstTarget(s.skills.env, p.skill))
	}
	s.playerIntendCast(p, sk, target)
	return true
}

// checkUseMagicConditions is Mobius Player.checkUseMagicConditions, with
// what Mobius sends for each refusal:
//
//   - Confused, afraid, paralyzed, stunned, asleep or dead: ActionFailed.
//   - Formal wear: its system message.
//   - Sitting: "You cannot move while sitting." and ActionFailed.
//   - A toggle that is on goes off (ActionFailed).
//   - Faking death: ActionFailed.
//   - No target (the player for self, aura and party skills): ActionFailed.
//   - Under reuse: the time left as text; disabled otherwise: "S1 is not
//     available at this time: being prepared for reuse.".
//   - The skill's conditions: their messages and ActionFailed.
//   - Negative skills: a peace zone ("You may not attack this target in a
//     peaceful zone." and ActionFailed), a target that cannot be attacked
//     without Ctrl (ActionFailed), out of range with Shift ("Your target is
//     out of range." and ActionFailed).
//   - A positive skill on a monster without Ctrl: ActionFailed.
//   - A player target the PvP rules forbid: "That is the incorrect target."
//     and ActionFailed.
//   - No line of sight for a skill with a cast range: "Cannot see target."
//     and ActionFailed.
//
// Mounts, cursed weapons, fishing, observers, doors, duels, the Olympiad and
// sieges do not exist yet.
func (s *Server) checkUseMagicConditions(p *Player, sk *skills.Skill, ctrl, shift bool) bool {
	a := p.skill
	el := a.el
	if el.IsAffected(skills.FlagConfused) || el.IsAffected(skills.FlagFear) || allSkillsDisabled(&a.skillBase) || p.dead {
		p.SendActionFailed()
		return false
	}
	if p.wearsFormalWear() {
		p.SendSystemMessage(sysMsgFormalWearNoCasting)
		return false
	}
	if p.Sitting {
		p.SendSystemMessage(sysMsgCannotMoveSitting)
		p.SendActionFailed()
		return false
	}
	if sk.IsToggle() && el.IsAffectedBySkill(sk.ID) {
		el.StopSkill(skills.FinishRemoved, sk.ID)
		p.SendActionFailed()
		return false
	}
	if a.fakeDeath {
		p.SendActionFailed()
		return false
	}

	var obj world.Object
	tt := sk.TargetType
	switch tt {
	case data.TargetAura, data.TargetFrontAura, data.TargetBehindAura, data.TargetParty, data.TargetClan,
		data.TargetPartyClan, data.TargetGround, data.TargetSelf, data.TargetAreaSummon, data.TargetAuraCorpseMob,
		data.TargetCommandChannel, data.TargetAuraFriendly:
		obj = p
	case data.TargetPet, data.TargetServitor, data.TargetSummon:
		// Summons come with Milestone 11.
	default:
		obj = p.Target
	}
	if obj == nil {
		p.SendActionFailed()
		return false
	}
	target, _ := obj.(Creature)

	if s.skillDisabled(&a.skillBase, sk) {
		if left := a.reuseLeft(sk, s.Now()); left > 0 {
			p.SendMessage(castReuseText(sk, left))
		} else {
			p.SendSystemMessage(sysMsgNotReadyForReuse, skills.SkillName(sk))
		}
		return false
	}
	if !sk.CheckCondition(s.skills.env, a, s.skillCreatureOf(obj), false) {
		p.SendActionFailed()
		return false
	}
	if sk.HasNegativeEffect() {
		if s.insidePeaceZone(p, target) {
			p.SendSystemMessage(sysMsgPeacefulZone)
			p.SendActionFailed()
			return false
		}
		if target == nil && !p.IsGM() {
			// Only creatures can be attacked (WorldObject.canBeAttacked).
			p.SendActionFailed()
			return false
		}
		if !castAutoAttackable(p, target) && !ctrl && !castSkipsForceCheck(tt) {
			p.SendActionFailed()
			return false
		}
		if shift && sk.CastRange > 0 && !inRadius2D(p.Position(), obj.Position(), sk.CastRange+attackCollision(p)) {
			p.SendSystemMessage(sysMsgTargetOutOfRange)
			p.SendActionFailed()
			return false
		}
	}
	if n, ok := target.(*Npc); ok && sk.EffectPoint > 0 && n.IsMonster() && !ctrl {
		p.SendActionFailed()
		return false
	}
	if !castSkipsPvPCheck(tt) {
		if tp, ok := target.(*Player); ok && !p.IsGM() && !s.checkPvPSkill(p, tp, sk) {
			p.SendSystemMessage(sysMsgIncorrectTarget)
			p.SendActionFailed()
			return false
		}
	}
	if sk.CastRange > 0 && !s.canSee(p, obj) {
		p.SendSystemMessage(sysMsgCannotSeeTarget)
		p.SendActionFailed()
		return false
	}
	return true
}

// castSkipsForceCheck lists the target types a negative skill may use
// without Ctrl on a target that cannot be attacked.
func castSkipsForceCheck(t data.TargetType) bool {
	switch t {
	case data.TargetAura, data.TargetFrontAura, data.TargetBehindAura, data.TargetAuraCorpseMob, data.TargetClan,
		data.TargetParty, data.TargetSelf, data.TargetGround, data.TargetAreaSummon, data.TargetUnlockable,
		data.TargetAuraFriendly:
		return true
	}
	return false
}

// castSkipsPvPCheck lists the target types whose PvP rules the target list
// checks itself.
func castSkipsPvPCheck(t data.TargetType) bool {
	switch t {
	case data.TargetParty, data.TargetClan, data.TargetPartyClan, data.TargetAura, data.TargetFrontAura,
		data.TargetBehindAura, data.TargetAreaSummon, data.TargetGround, data.TargetSelf:
		return true
	}
	return false
}

// castAutoAttackable is Mobius target.isAutoAttackable(player).
func castAutoAttackable(p *Player, t Creature) bool {
	switch v := t.(type) {
	case *Npc:
		return v.autoAttackableBy(p)
	case *Player:
		return v.autoAttackableBy(p)
	}
	return false
}

// castReuseText is Mobius's reuse message: the hours, minutes and seconds
// left of the skill's reuse.
func castReuseText(sk *skills.Skill, left time.Duration) string {
	secs := int(left / time.Second)
	h, m, sec := secs/3600, (secs%3600)/60, secs%60
	switch {
	case h > 0:
		return "There are " + strconv.Itoa(h) + " hour(s), " + strconv.Itoa(m) + " minute(s), and " +
			strconv.Itoa(sec) + " second(s) remaining in " + sk.Name + "'s re-use time."
	case m > 0:
		return "There are " + strconv.Itoa(m) + " minute(s), " + strconv.Itoa(sec) + " second(s) remaining in " +
			sk.Name + "'s re-use time."
	}
	return "There are " + strconv.Itoa(sec) + " second(s) remaining in " + sk.Name + "'s re-use time."
}

// checkPvPSkill is Mobius Player.checkPvpSkill for a player target: a
// debuff or negative skill may not hit p itself or a player in a peace
// zone; a PvP-only skill needs a flagged or karma target outside PvP and
// siege zones; in a common arena anything goes; clan mates and peaceful
// players (no flag, no karma) take only a damage skill with an effect
// range, forced with Ctrl on the current target; others must be flagged or
// have karma. Members of clans at mutual war always may; alliance members
// count as clan mates. Parties, duels, the Olympiad and sieges do not exist
// yet.
func (s *Server) checkPvPSkill(p, target *Player, sk *skills.Skill) bool {
	if sk == nil || target == nil {
		return false
	}
	if !sk.IsDebuff && !sk.HasEffectType(skills.TypeStealAbnormal) && !sk.HasNegativeEffect() {
		return true
	}
	if target == p {
		return false
	}
	if target.Zones.Inside(ZonePeace) {
		return false
	}
	if sk.PvPOnly && target.PvPFlag == 0 && target.Karma() == 0 && !p.Zones.Inside(ZonePvP) && !p.Zones.Inside(ZoneSiege) {
		return false
	}
	st := &p.skill.cast
	ctrl := st.current != nil && st.current.ctrl
	forced := sk.EffectRange > 0 && ctrl && p.Target == world.Object(target) && sk.IsDamage()
	if p.Zones.Inside(ZonePvP) && target.Zones.Inside(ZonePvP) {
		return true
	}
	if p.clan.clan != nil && target.clan.clan != nil {
		if MutualWar(p.clan.clan, target.clan.clan) {
			return true // clanwar.go
		}
		if sameClanOrAlly(p, target) {
			return forced
		}
	}
	if target.PvPFlag == 0 && target.Karma() == 0 {
		return forced
	}
	return target.PvPFlag > 0 || target.Karma() > 0
}

// playerIntendCast is the CAST intention (PlayerAI and CreatureAI
// setIntentionCast): the intention the cast interrupts is saved (not for
// toggles); the follow stops; a resting player refuses magic; a swing in
// progress ends first, then the intention changes and thinks.
func (s *Server) playerIntendCast(p *Player, sk *skills.Skill, target Creature) {
	ai := s.playerAIOf(p)
	st := &p.skill.cast
	if !sk.IsToggle() && ai.intention != playerCasting {
		st.saved = castSaved{set: true, intention: ai.intention, target: ai.target}
	}
	s.playerStopFollow(p)
	if p.Sitting && sk.IsMagic() {
		p.SendActionFailed()
		st.casting = false
		return
	}
	if d := s.attackLeft(p); d > 0 {
		s.castLater(st, d, func() {
			if !p.InGame() || !p.InWorld() {
				return
			}
			if s.AttackingNow(p) {
				s.AbortAttack(p)
			}
			s.playerChangeToCast(p, sk, target)
		})
		return
	}
	s.playerChangeToCast(p, sk, target)
}

// attackLeft is how long p's attack still runs (Mobius setIntentionCast's
// wait for the bow lock or the swing's end).
func (s *Server) attackLeft(p *Player) time.Duration {
	st := s.attack.states[p]
	if st == nil {
		return 0
	}
	if tick := s.gameTicks(); st.bowLock > tick {
		return time.Duration((st.bowLock-tick)*walkTickMillis) * time.Millisecond
	}
	if now := s.Now(); st.end.After(now) {
		return st.end.Sub(now)
	}
	return 0
}

// playerChangeToCast is changeIntentionToCast: the cast target and skill,
// the intention CAST, and a think.
func (s *Server) playerChangeToCast(p *Player, sk *skills.Skill, target Creature) {
	ai := s.playerAIOf(p)
	st := &p.skill.cast
	st.skill, st.target = sk, target
	ai.intention = playerCasting
	s.playerThink(p)
}

// playerThinkCast is PlayerAI.thinkCast: a lost target ends the intention;
// a target out of reach (the skill's cast range through mAtkRange, plus
// both collision radii) is walked up to first; a skill with a hit time
// stops a walk; then the cast, with the cast target as the player's target
// for its time.
func (s *Server) playerThinkCast(p *Player, ai *playerAI) {
	st := &p.skill.cast
	sk, target := st.skill, st.target
	if sk == nil {
		return
	}
	if s.castTargetLost(p, sk, target) {
		if sk.HasNegativeEffect() && ai.target != nil {
			st.target = nil
		}
		st.casting = false
		return
	}
	if target != nil && s.playerMaybeMoveToPawn(p, ai, target, s.magicalAttackRange(p, sk)) {
		st.casting = false
		return
	}
	if sk.HitTime > 50 && !sk.SimultaneousCast {
		s.playerStopMoving(p)
	}
	cur := p.Target
	if target != nil && cur != nil && cur != world.Object(target) {
		s.SetTarget(p, target)
		s.castBegin(p, sk, false)
		s.SetTarget(p, cur)
		return
	}
	s.castBegin(p, sk, false)
}

// castTargetLost is CreatureAI.checkTargetLost for a cast: no target, or for
// a negative area skill a target out of sight (out of the walk's reach
// while walking), ends the intention.
func (s *Server) castTargetLost(p *Player, sk *skills.Skill, target Creature) bool {
	if target == nil {
		s.playerIntendIdle(p)
		return true
	}
	if sk.HasNegativeEffect() && sk.AffectRange > 0 {
		from, to := geoPoint(p.Position()), geoPoint(target.Position())
		lost := !s.geo.CanSeeTarget(from, to)
		if p.Move != nil {
			lost = !s.geo.CanMoveToTarget(from, to)
		}
		if lost {
			s.playerIntendIdle(p)
			return true
		}
	}
	return false
}

// magicalAttackRange is Mobius getMagicalAttackRange(skill): the cast range
// through the mAtkRange modifiers, cut to an int.
func (s *Server) magicalAttackRange(p *Player, sk *skills.Skill) int {
	return int(javaInt32(p.skill.Calc(stats.MAtkRange, float64(sk.CastRange), nil, sk)))
}

// castNextIsMove reports whether p waits to walk somewhere after its cast
// (PlayerAI.getNextIntention() == MOVE_TO).
func (s *Server) castNextIsMove(p *Player) bool {
	if ai := s.attack.players[p]; ai != nil && ai.savedMove != nil {
		return true
	}
	st := &p.skill.cast
	return st.saved.set && st.saved.intention == playerMovingTo
}

// castFinishedAI is PlayerAI.notifyActionFinishCasting: while the intention
// is CAST, a move clicked during the cast replays, else the intention the
// cast interrupted (an attack goes on); anything else, or nothing saved,
// makes the player idle. A saved walk without a destination makes the
// player idle too (Mobius would keep the CAST intention).
func (s *Server) castFinishedAI(p *Player) {
	ai := s.attack.players[p]
	if ai == nil || ai.intention != playerCasting {
		return
	}
	st := &p.skill.cast
	saved := st.saved
	st.saved = castSaved{}
	if dest := ai.savedMove; dest != nil {
		ai.savedMove = nil
		switch {
		case p.Sitting:
			p.SendActionFailed()
		case s.playerIntendMove(p, *dest):
			m := s.motionOf(p)
			if s.startWalk(p, m, *dest, 0) {
				s.broadcastWalk(p, m, false)
			}
		}
		return
	}
	if saved.set && saved.intention == playerAttacking && saved.target != nil {
		s.playerIntendAttack(p, saved.target)
		return
	}
	s.playerIntendIdle(p)
}

// playerCastBusy reports a player that may not walk or attack now (Mobius
// isAllSkillsDisabled or isCastingNow).
func (s *Server) playerCastBusy(p *Player) bool {
	if p.skill == nil {
		return false
	}
	return p.skill.cast.casting || allSkillsDisabled(&p.skill.skillBase)
}

// castTargetCanceled is RequestTargetCanceld (Esc): a locked target stays
// ("Failed to disable attack target."); with unselect 0 a cast that can
// still be aborted aborts, otherwise the target goes.
func (s *Server) castTargetCanceled(p *Player, unselect uint16) {
	if p.skill != nil && p.skill.locked != nil {
		p.SendSystemMessage(sysMsgCannotDisableTarget)
		return
	}
	if unselect == 0 && s.CastingNow(p) && s.CanAbortCast(p) {
		s.abortCast(p)
		return
	}
	if p.Target != nil {
		s.SetTarget(p, nil)
	}
}
