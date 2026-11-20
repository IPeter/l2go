package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
)

// energyDamage is Mobius EnergyDamage, the force skills of players (Sonic
// Blaster, Force Blaster): its own roll is the physical skill evasion. The
// damage is (P.Atk + power) × soulshot × (1 + 0.2 × (charges before the
// cast − 1)) × 70 with a bow, else 77, over P.Def (with the shield unless
// ignoreShieldDefence; a perfect block deals 1), times the weapon trait,
// element and skill trait bonuses, and the PvP factors against a player.
// STR bonus × criticalChance over 100 × nextDouble doubles it and adds
// critDmgAdd and the target's defCritDamageAdd. Damage tells the attacker,
// hurts the target and may be counter-attacked.
type energyDamage struct {
	skills.Base
	power          float64
	criticalChance int
	ignoreShield   bool
}

func init() {
	register(data.EffectEnergyDamage, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		crit, err := def.Params.Int("criticalChance", 0)
		if err != nil {
			return nil, err
		}
		ignore, err := def.Params.Bool("ignoreShieldDefence", false)
		if err != nil {
			return nil, err
		}
		return &energyDamage{Base: skills.NewBase(def), power: power, criticalChance: crit, ignoreShield: ignore}, nil
	})
}

func (*energyDamage) Type() skills.EffectType { return skills.TypePhysicalAttack }

func (*energyDamage) Instant() bool { return true }

func (*energyDamage) CalcSuccess(info *skills.BuffInfo) bool {
	return !skills.PhysicalSkillEvasion(info.Env(), info.Effector(), info.Effected(), info.Skill())
}

func (e *energyDamage) OnStart(info *skills.BuffInfo) {
	env, attacker, effected, sk := info.Env(), info.Effector(), info.Effected(), info.Skill()
	if !isPlayer(attacker) {
		return
	}
	attack := attacker.PAtk(effected)
	defence := effected.PDef(attacker)
	if !e.ignoreShield {
		switch skills.ShieldUse(env, attacker, effected, sk, true) {
		case stats.ShieldSucceed:
			defence += float64(effected.ShieldDef())
		case stats.ShieldPerfect:
			defence = -1
		}
	}
	damage := 1.0
	critical := false
	if defence != -1 {
		weaponTrait := stats.WeaponTraitBonus(effected.Traits().Defence(skills.WeaponTrait(attacker.AttackType())))
		mult := weaponTrait * skills.AttributeBonus(attacker, effected, sk) * skills.GeneralTraitBonus(attacker, effected, sk.Trait, true)
		ssBoost := 1.0
		if sk.UseSoulShot() && attacker.ChargedShot(skills.Soulshot) {
			ssBoost = 2
		}
		weaponBoost := 77.0
		if w, ok := attacker.ActiveWeapon(); ok && w.Type == data.WeaponBow {
			weaponBoost = 70
		}
		// The charges were spent before the effects run: add them back.
		chargesBoost := float64(float64(attacker.AsPlayer().Charges()+sk.ChargeConsume-1)*0.2) + 1
		attack += e.power
		attack *= ssBoost
		attack *= chargesBoost
		attack *= weaponBoost
		if isPlayer(effected) {
			defence *= effected.Calc(stats.PvpPhysSkillsDef, 1.0, nil, nil)
		}
		damage = attack / defence
		damage *= mult
		if isPlayer(effected) {
			damage *= attacker.Calc(stats.PvpPhysSkillsDmg, 1.0, nil, nil)
			damage = attacker.Calc(stats.PhysicalSkillPower, damage, nil, nil)
		}
		critical = float64(attacker.StatBonus(data.BaseSTR)*float64(e.criticalChance)) > float64(env.Rand.Float64()*100)
		if critical {
			damage *= 2
			damage += attacker.Calc(stats.CritDmgAdd, 0, effected, sk)
			damage += effected.Calc(stats.DefCritDamageAdd, 0, effected, sk)
		}
	}
	if damage > 0 {
		attacker.SendDamageMessage(effected, int(javaInt(damage)), false, critical, false)
		effected.ReduceCurrentHP(damage, attacker, sk)
		effected.NotifyDamageReceived(damage, attacker, sk, critical, false)
		skills.DamageReflected(env, attacker, effected, sk, critical)
	}
}
