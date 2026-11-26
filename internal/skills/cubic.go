package skills

import (
	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
)

// The cubic formulas (Plan 14 M11f): a cubic acts for its owner with its own
// power instead of the owner's M.Atk.

// CubicMagicDamage is Formulas.calcMagicDam(Cubic, ...): 91 × (power + the
// skill's power) / M.Def (a perfect block deals 1); with MagicFailures a
// failed MagicSuccess of the owner is rolled again (a success within 9
// levels of the skill's magic level halves the damage, "Your attack has
// failed."; otherwise 1, "S1 has resisted your S2."), a player target is
// told it resisted; else a magic critical hit triples it. Then the element
// bonus and, against an attackable, the owner's PvE magic factor.
func CubicMagicDamage(env *Env, owner, target Creature, sk *Skill, power int, shld stats.ShieldResult, mcrit bool) float64 {
	mDef := target.MDef(owner, sk)
	switch shld {
	case stats.ShieldSucceed:
		mDef += float64(target.ShieldDef())
	case stats.ShieldPerfect:
		return 1
	}
	pvp, pve := isPlayable(target), isAttackable(target)
	damage := 91 * ((float64(power) + sk.PowerFor(owner, target, pvp, pve)) / mDef)
	drain := sk.HasEffectType(TypeHPDrain)
	switch {
	case env.Config.MagicFailures && !MagicSuccess(env, owner, target, sk):
		if MagicSuccess(env, owner, target, sk) && target.Level()-sk.MagicLevel <= 9 {
			if drain {
				send(owner, MsgDrainHalfSuccessful)
			} else {
				send(owner, MsgYourAttackHasFailed)
			}
			damage /= 2
		} else {
			send(owner, MsgS1HasResistedYourS2, sp.TextParam(target.Name()), SkillName(sk))
			damage = 1
		}
		if isPlayer(target) {
			if drain {
				send(target, MsgYouResistedS1sDrain, sp.TextParam(owner.Name()))
			} else {
				send(target, MsgYouResistedS1sMagic, sp.TextParam(owner.Name()))
			}
		}
	case mcrit:
		damage *= 3
	}
	damage *= AttributeBonus(owner, target, sk)
	if isAttackable(target) {
		damage *= calc(owner, stats.PveMagicalDmg, 1)
	}
	return damage
}

// ApplyCubic is Cubic.applyCubicEffects: nothing on another invulnerable
// target of a negative skill (or from a GM owner who may not deal damage);
// the GENERAL, PVE or PVP and CHANNELING effects with the owner as caster,
// the continuous ones of a toggle or continuous skill joining the target's
// list without a success roll (the cubic rolled already).
func (s *Skill) ApplyCubic(env *Env, owner, target Creature) {
	if owner == nil || target == nil {
		return
	}
	if !same(owner, target) && s.HasNegativeEffect() {
		if target.IsInvul() {
			return
		}
		if p := asPlayer(owner); p != nil && p.IsGM() && !p.CanGiveDamage() {
			return
		}
	}
	add := s.IsToggle() || s.IsContinuous()
	info := NewBuffInfo(env, owner, target, s)
	s.ApplyEffectScope(data.ScopeGeneral, info, true, add)
	if scope, ok := pvpOrPveScope(owner, target); ok {
		s.ApplyEffectScope(scope, info, true, add)
	}
	s.ApplyEffectScope(data.ScopeChanneling, info, true, add)
	if add {
		target.Effects().Add(info)
	}
}

// CubicSkillSuccess is Formulas.calcCubicSkillSuccess: a debuff of power -1
// lands; debuff immunity, a perfect block or a reflection stop it; without
// an activate rate (or one over 99) or a basic property it lands; otherwise
// (activate rate, or the power) / the target's stat bonus × the owner's
// trait, level and element bonuses, within the skill's chances, against
// Get(100).
func CubicSkillSuccess(env *Env, owner, target Creature, sk *Skill, shld stats.ShieldResult) bool {
	if sk.IsDebuff {
		if sk.Power == -1 {
			return true
		}
		if target.Calc(stats.DebuffImmunity, 0, nil, sk) > 0 {
			return false
		}
	}
	if shld == stats.ShieldPerfect {
		return false
	}
	if BuffDebuffReflection(env, target, sk) {
		return false
	}
	rate := sk.ActivateRate
	if rate == -1 || rate > 99 || sk.BasicProperty == data.BaseNone {
		return true
	}
	base := float64(rate)
	if rate <= 0 {
		base = sk.Power
	}
	r := base / target.StatBonus(sk.BasicProperty)
	r *= GeneralTraitBonus(owner, target, sk.Trait, false)
	r *= LvlBonusMod(owner, target, sk)
	r *= AttributeBonus(owner, target, sk)
	final := clamp(r, float64(sk.MinChanceOr(&env.Config)), float64(sk.MaxChanceOr(&env.Config)))
	return float64(env.Rand.Get(100)) < final
}
