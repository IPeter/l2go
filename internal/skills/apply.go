package skills

import "github.com/IPeter/l2go/internal/data"

// ApplyEffectScope runs the effects of a scope on info's effected
// (Skill.applyEffectScope): an instant effect runs when applyInstant is set
// and its own roll (CalcSuccess) succeeds; a continuous effect joins info
// when addContinuous is set and CanStart allows it.
func (s *Skill) ApplyEffectScope(scope data.EffectScope, info *BuffInfo, applyInstant, addContinuous bool) {
	for _, e := range s.ScopeEffects(scope) {
		if e.Instant() {
			if applyInstant && e.CalcSuccess(info) {
				e.OnStart(info)
			}
		} else if addContinuous && e.CanStart(info) {
			info.addEffect(e)
		}
	}
}

// pvpOrPveScope is the extra scope between effector and effected: PVE from
// a playable on an attackable, PVP between playables, none otherwise.
func pvpOrPveScope(effector, effected Creature) (data.EffectScope, bool) {
	switch {
	case isPlayable(effector) && isAttackable(effected):
		return data.ScopePVE, true
	case isPlayable(effector) && isPlayable(effected):
		return data.ScopePVP, true
	}
	return 0, false
}

// ApplyEffects is Skill.applyEffects(effector, effected, self, passive,
// instant, abnormalTime) (K4):
//
//   - A negative skill does nothing to another creature that is
//     invulnerable, nor from a GM who may not deal damage.
//   - Continuous effects land for toggles, and for continuous skills when
//     EffectSuccess allows it (its roll and message).
//   - Unless self or passive: the GENERAL effects, then PVE or PVP by the
//     pair, on effected; the continuous ones join one BuffInfo that goes to
//     effected's list. A positive abnormalTime replaces the abnormal time.
//   - self: the SELF effects on the effector (A3 skills count as
//     continuous); its BuffInfo joins the list only for skills with a BUFF
//     effect, as Mobius does.
//   - passive: the PASSIVE effects on the effector, always added.
func (s *Skill) ApplyEffects(env *Env, effector, effected Creature, self, passive, instant bool, abnormalTime int) {
	if effected == nil || effector == nil {
		return
	}
	if !same(effector, effected) && s.HasNegativeEffect() {
		if effected.IsInvul() {
			return
		}
		if p := asPlayer(effector); isPlayer(effector) && p.IsGM() && !p.CanGiveDamage() {
			return
		}
	}
	addContinuous := !passive && (s.IsToggle() || (s.OperateType.IsContinuous() && EffectSuccess(env, effector, effected, s)))
	if !self && !passive {
		info := NewBuffInfo(env, effector, effected, s)
		if addContinuous && abnormalTime > 0 {
			info.SetAbnormalTime(abnormalTime)
		}
		s.ApplyEffectScope(data.ScopeGeneral, info, instant, addContinuous)
		if scope, ok := pvpOrPveScope(effector, effected); ok {
			s.ApplyEffectScope(scope, info, instant, addContinuous)
		}
		if addContinuous {
			effected.Effects().Add(info)
		}
	}
	if self {
		addContinuous = !passive && (s.IsToggle() || (s.IsContinuous() && EffectSuccess(env, effector, effector, s)))
		info := NewBuffInfo(env, effector, effector, s)
		if addContinuous && abnormalTime > 0 {
			info.SetAbnormalTime(abnormalTime)
		}
		s.ApplyEffectScope(data.ScopeSelf, info, instant, addContinuous)
		if addContinuous && s.HasEffectType(TypeBuff) {
			effector.Effects().Add(info)
		}
	}
	if passive {
		info := NewBuffInfo(env, effector, effector, s)
		s.ApplyEffectScope(data.ScopePassive, info, false, true)
		effector.Effects().Add(info)
	}
}

// ApplyTo is applyEffects(effector, effected): the GENERAL (and PVE or PVP)
// effects, instant ones included, with the skill's abnormal time.
func (s *Skill) ApplyTo(env *Env, effector, effected Creature) {
	s.ApplyEffects(env, effector, effected, false, false, true, 0)
}

// AddPassive starts a passive skill's effects on c, as Creature.addSkill
// does for a passive (applyEffects with passive set).
func (s *Skill) AddPassive(env *Env, c Creature) {
	s.ApplyEffects(env, c, c, false, true, false, 0)
}

// Activate is Skill.activateSkill for a creature (not a cubic): the effects
// on each target, a reflected debuff landing its continuous effects on the
// caster instead; then the SELF effects (replacing the caster's earlier
// ones of the skill); then the shots the skill used are spent, and a
// suicide skill kills the caster.
func (s *Skill) Activate(env *Env, caster Creature, targets []Creature) {
	for _, target := range targets {
		if target == nil {
			continue
		}
		if BuffDebuffReflection(env, target, s) {
			s.ApplyEffects(env, target, caster, false, false, false, 0)
			info := NewBuffInfo(env, caster, target, s)
			s.ApplyEffectScope(data.ScopeGeneral, info, true, false)
			if scope, ok := pvpOrPveScope(caster, target); ok {
				s.ApplyEffectScope(scope, info, true, false)
			}
		} else {
			s.ApplyTo(env, caster, target)
		}
	}
	if s.HasEffects(data.ScopeSelf) {
		if caster.Effects().IsAffectedBySkill(s.ID) {
			caster.Effects().StopSkill(FinishRemoved, s.ID)
		}
		s.ApplyEffects(env, caster, caster, true, false, true, 0)
	}
	switch {
	case s.UseSpiritShot():
		if caster.ChargedShot(BlessedSpiritshot) {
			caster.SetChargedShot(BlessedSpiritshot, false)
		} else {
			caster.SetChargedShot(Spiritshot, false)
		}
	case s.UseSoulShot():
		caster.SetChargedShot(Soulshot, false)
	}
	if s.SuicideAttack {
		caster.DoDie(caster)
	}
}
