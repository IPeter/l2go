package skills

import (
	"strconv"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
)

// Skill is one level of a skill as the server uses it: the resolved data
// (embedded, read-only) with its compiled preconditions and effects and the
// values Mobius derives from them. The Catalog builds every Skill; they are
// shared and must not be changed.
type Skill struct {
	*data.Skill

	preconds []Precondition
	effects  [data.NumEffectScopes][]Effect
	types    effectTypeSet
	visual   VisualMasks
	// refusal says why the skill cannot be used (a stub effect, a
	// condition without a port); empty when it can.
	refusal string
}

// VisualMasks are the abnormal visual effect bits of a skill or creature,
// split as the packets send them: the normal mask, the special one and the
// event one (Mobius AbnormalVisualEffect types 0, 1 and 2).
type VisualMasks struct {
	Normal, Special, Event uint32
}

func visualMasksOf(aves []data.AbnormalVisualEffect) VisualMasks {
	var v VisualMasks
	for _, a := range aves {
		switch {
		case a.IsEvent():
			v.Event |= a.Mask()
		case a.IsSpecial():
			v.Special |= a.Mask()
		default:
			v.Normal |= a.Mask()
		}
	}
	return v
}

// Visual returns the skill's abnormal visual effects as masks.
func (s *Skill) Visual() VisualMasks { return s.visual }

// Refusal returns why the skill cannot be used, or "" when it can. A skill
// with an effect that has no port, or a precondition of a kind without a
// port, refuses: the cast gets ActionFailed and one log line (K2).
func (s *Skill) Refusal() string { return s.refusal }

// ScopeEffects returns the effects of a scope. The slice must not be
// changed.
func (s *Skill) ScopeEffects(scope data.EffectScope) []Effect {
	if int(scope) >= len(s.effects) {
		return nil
	}
	return s.effects[scope]
}

// HasEffects reports whether the scope has effects (Mobius hasEffects).
func (s *Skill) HasEffects(scope data.EffectScope) bool { return len(s.ScopeEffects(scope)) > 0 }

// HasEffectType reports whether any effect of any scope has one of the
// types (Mobius hasEffectType).
func (s *Skill) HasEffectType(types ...EffectType) bool {
	for _, t := range types {
		if s.types.has(t) {
			return true
		}
	}
	return false
}

// IsActive reports an active skill (not passive or toggle).
func (s *Skill) IsActive() bool { return s.OperateType.IsActive() }

// IsContinuous is Mobius Skill.isContinuous: A2, A4, DA2 or A3.
func (s *Skill) IsContinuous() bool {
	return s.OperateType.IsContinuous() || s.OperateType.IsSelfContinuous()
}

// IsSelfContinuous reports A3.
func (s *Skill) IsSelfContinuous() bool { return s.OperateType.IsSelfContinuous() }

// IsChanneling reports a channeling skill.
func (s *Skill) IsChanneling() bool { return s.OperateType.IsChanneling() }

// HasNegativeEffect reports an offensive skill: a negative effect point and
// a target other than the caster.
func (s *Skill) HasNegativeEffect() bool {
	return s.EffectPoint < 0 && s.TargetType != data.TargetSelf
}

// IsDamage reports a skill that deals damage.
func (s *Skill) IsDamage() bool {
	return s.HasEffectType(TypeMagicalAttack, TypeHPDrain, TypePhysicalAttack, TypePhysicalAttackHPLink)
}

// UseSoulShot reports a skill that a soulshot strengthens (physical attack
// effects).
func (s *Skill) UseSoulShot() bool {
	return s.HasEffectType(TypePhysicalAttack, TypePhysicalAttackHPLink)
}

// UseSpiritShot reports a skill that a spiritshot strengthens: magic skills.
func (s *Skill) UseSpiritShot() bool { return s.IsMagic() }

// IsHealingPotion reports an HP_RECOVER skill: a potion, shown as the
// short buff and not counted in the buff limit.
func (s *Skill) IsHealingPotion() bool { return s.AbnormalType == data.AbnormalHPRecover }

// Is7Signs reports the Seven Signs skills (4361-4366), which the buff
// limit leaves alone.
func (s *Skill) Is7Signs() bool { return s.ID > 4360 && s.ID < 4367 }

// IsAOE reports the area target types.
func (s *Skill) IsAOE() bool {
	switch s.TargetType {
	case data.TargetArea, data.TargetAura, data.TargetBehindArea, data.TargetBehindAura,
		data.TargetFrontArea, data.TargetFrontAura:
		return true
	}
	return false
}

// ReuseHash is the key of the skill's reuse stamp (SkillData.getSkillHashCode):
// reuse is kept per ID and level, so a new level is ready at once.
func (s *Skill) ReuseHash() int { return s.ID*1021 + s.Level }

// PowerFor is Skill.getPower(caster, target, isPvP, isPvE): the PvE, PvP
// or plain power, scaled by the caster's (DEATH_LINK) or target's
// (PHYSICAL_ATTACK_HP_LINK) missing HP.
func (s *Skill) PowerFor(caster, target Creature, pvp, pve bool) float64 {
	p := s.Power
	switch {
	case pve:
		p = s.PvEPower
	case pvp:
		p = s.PvPPower
	}
	if caster == nil {
		return p
	}
	switch {
	case s.HasEffectType(TypeDeathLink):
		return mul(p, -(mul(caster.CurrentHP(), 2)/float64(caster.MaxHP()))+2)
	case s.HasEffectType(TypePhysicalAttackHPLink) && target != nil:
		return mul(p, -(mul(target.CurrentHP(), 2)/float64(target.MaxHP()))+2)
	}
	return p
}

// RollAffectLimit is Skill.getAffectLimit: the first number of the affect
// limit plus Get(the second); it draws only when the second is positive.
func (s *Skill) RollAffectLimit(r stats.Rand) int {
	return s.AffectLimit[0] + r.Get(s.AffectLimit[1])
}

// MinChanceOr returns the skill's minimum landing chance, or the configured
// one when it has none; MaxChanceOr the maximum.
func (s *Skill) MinChanceOr(cfg *Config) int {
	if s.MinChance == -1 {
		return cfg.MinAbnormalStateSuccessRate
	}
	return s.MinChance
}

// MaxChanceOr returns the skill's maximum landing chance, or the configured
// one.
func (s *Skill) MaxChanceOr(cfg *Config) int {
	if s.MaxChance == -1 {
		return cfg.MaxAbnormalStateSuccessRate
	}
	return s.MaxChance
}

// String returns "Skill Name(id,level)", as Mobius toString.
func (s *Skill) String() string {
	return "Skill " + s.Name + "(" + strconv.Itoa(s.ID) + "," + strconv.Itoa(s.Level) + ")"
}
