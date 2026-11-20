package skills

import (
	"math"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
)

// The skill formulas of Mobius Formulas.java and Creature.beginCast, with
// Mobius's rounding points and draw order. Class balance and champion
// multipliers are left out: their defaults are 1 (Plan 04 W16).

// meleeAttackRange is Formulas.MELEE_ATTACK_RANGE: skills with a longer
// cast range are not counter-attacked.
const meleeAttackRange = 40

// More system messages of the formulas.
const (
	MsgShieldDefenseSucceeded sp.SystemMessageID = 111  // Your shield defense has succeeded.
	MsgExcellentShieldDefense sp.SystemMessageID = 1281 // Your excellent shield defense was a success!
)

// calc runs a stat of c without target and skill (calcStat(stat, init)).
func calc(c Creature, st stats.Stat, init float64) float64 { return c.Calc(st, init, nil, nil) }

// CastTime is the cast time of a skill in milliseconds, as Creature.beginCast
// computes it: hit time plus cool time; for a non-static skill (and not a
// channeling skill with a channeling skill ID) scaled by the casting speed,
// magic × 333 / M.Atk.Spd, the rest × 300 / P.Atk.Spd, truncated; magic with
// a charged spiritshot × 0.7. A magic skill of more than 550 ms base takes
// at least 550 ms; a non-static skill of 500 ms base or more at least 500.
func CastTime(c Creature, sk *Skill) int {
	base := sk.HitTime + sk.CoolTime
	t := base
	if !sk.IsChanneling() || sk.ChannelingSkillID == 0 {
		if !sk.IsStatic() {
			t = atkSpdTime(c, sk, float64(t))
		}
		if sk.IsMagic() && (c.ChargedShot(Spiritshot) || c.ChargedShot(BlessedSpiritshot)) {
			t = javaInt(mul(0.7, float64(t)))
		}
	}
	switch {
	case sk.IsMagic() && base > 550 && t < 550:
		t = 550
	case !sk.IsStatic() && base >= 500 && t < 500:
		t = 500
	}
	return t
}

// atkSpdTime is Formulas.calcAtkSpd.
func atkSpdTime(c Creature, sk *Skill, skillTime float64) int {
	if sk.IsMagic() {
		return javaInt(mul(skillTime/float64(c.MAtkSpd()), 333))
	}
	return javaInt(mul(skillTime/c.PAtkSpd(), 300))
}

// ReuseDelay is the reuse of a skill in milliseconds before the skill
// mastery roll (Creature.beginCast): static and staticReuse skills as the
// data says; magic × mReuse, then × (333f / M.Atk.Spd) in Java floats;
// physical × pReuse and dances × dReuse, then × 333 / P.Atk.Spd in doubles.
// Each step is truncated to an int.
func ReuseDelay(c Creature, sk *Skill) int {
	r := sk.ReuseDelay
	switch {
	case sk.StaticReuse || sk.IsStatic():
		return r
	case sk.IsMagic():
		r = javaInt(mul(float64(r), calc(c, stats.MReuse, 1)))
		f := float32(333) / float32(c.MAtkSpd())
		return javaIntF(float32(float32(r) * f))
	case sk.IsPhysical():
		r = javaInt(mul(float64(r), calc(c, stats.PReuse, 1)))
	default:
		r = javaInt(mul(float64(r), calc(c, stats.DReuse, 1)))
	}
	return javaInt(mul(float64(r), 333/c.PAtkSpd()))
}

// MPConsume is CreatureStat.getMpConsume: the MP a skill takes at the hit.
// Each active dance raises a dance's cost by half its base cost, rounded
// up (DanceConsumeAdditionalMP); then the mpConsume stat and the kind's
// rate (dance, magic or physical), truncated.
func MPConsume(env *Env, c Creature, sk *Skill) int {
	if sk == nil {
		return 1
	}
	mp := float64(sk.MPConsume)
	next := math.Ceil(float64(sk.MPConsume) / 2.)
	if sk.IsDance() && env.Config.DanceConsumeAdditionalMP {
		if l := c.Effects(); l != nil && l.DanceCount() > 0 {
			mp += mul(float64(l.DanceCount()), next)
		}
	}
	mp = c.Calc(stats.MPConsume, mp, nil, sk)
	switch {
	case sk.IsDance():
		return javaInt(calc(c, stats.DanceMPConsumeRate, mp))
	case sk.IsMagic():
		return javaInt(calc(c, stats.MagicalMPConsumeRate, mp))
	}
	return javaInt(calc(c, stats.PhysicalMPConsumeRate, mp))
}

// MPInitialConsume is CreatureStat.getMpInitialConsume: the MP a skill
// takes when the cast starts.
func MPInitialConsume(c Creature, sk *Skill) int {
	if sk == nil {
		return 1
	}
	return javaInt(c.Calc(stats.MPConsume, float64(sk.MPInitialConsume), nil, sk))
}

// SkillMastery is Formulas.calcSkillMastery: a player's roll to double a
// buff's time or reset a reuse. Static skills and creatures without the
// skillCritical stat never roll; otherwise it draws Get(100) once against
// the STR (skillCritical 1) or INT (4) bonus times skillCriticalProbability.
func SkillMastery(env *Env, c Creature, sk *Skill) bool {
	if !isPlayer(c) || sk.IsStatic() {
		return false
	}
	val := javaInt(calc(c, stats.SkillCritical, 0))
	if val == 0 {
		return false
	}
	chance := 0.0
	switch val {
	case 1:
		chance = c.StatBonus(data.BaseSTR)
	case 4:
		chance = c.StatBonus(data.BaseINT)
	}
	chance = mul(chance, calc(c, stats.SkillCriticalProbability, 1))
	return float64(env.Rand.Get(100)) < chance
}

// AbnormalTime is Formulas.calcEffectAbnormalTime: the skill's abnormal
// time in seconds, -1 for passives and toggles; half for a herb on a
// servitor; doubled by a skill mastery roll (SkillMastery draws); with
// DebuffDurationUsesResists, debuffs shortened by the target's resistances
// down to half.
func AbnormalTime(env *Env, caster, target Creature, sk *Skill) int {
	t := sk.AbnormalTime
	if sk.IsPassive() || sk.IsToggle() {
		t = -1
	}
	if target != nil && isServitor(target) && sk.AbnormalInstant {
		t /= 2
	}
	if caster != nil && SkillMastery(env, caster, sk) {
		t *= 2
	}
	if caster != nil && target != nil && sk.IsDebuff && env.Config.DebuffDurationUsesResists && t > 0 {
		statMod := target.StatBonus(sk.BasicProperty)
		resMod := GeneralTraitBonus(caster, target, sk.Trait, false)
		lvlMod := LvlBonusMod(caster, target, sk)
		elemMod := AttributeBonus(caster, target, sk)
		v := mul(mul(mul(float64(t), resMod), lvlMod), elemMod) / statMod
		t = javaInt(math.Ceil(clamp(v, mul(float64(t), 0.5), float64(t))))
	}
	return t
}

// LvlBonusMod is Formulas.calcLvlBonusMod.
func LvlBonusMod(attacker, target Creature, sk *Skill) float64 {
	lvl := attacker.Level()
	if sk.MagicLevel > 0 {
		lvl = sk.MagicLevel
	}
	skillMod := 1 + float64(sk.LvlBonusRate)/100.
	lvlMod := 1 + float64(lvl-target.Level())/100.
	return mul(skillMod, lvlMod)
}

// GeneralTraitBonus is Formulas.calcGeneralTraitBonus: 1 without a trait;
// 0 against a target invulnerable to it; for a weakness trait 1 unless both
// sides have it; for an effect trait 1 when resistance is ignored; then
// (attack − defence) + 1 within 0.05 and 2. Other traits give 1.
func GeneralTraitBonus(attacker, target Creature, tt data.TraitType, ignoreResistance bool) float64 {
	if tt == data.TraitNone {
		return 1
	}
	tr := target.Traits()
	if tr.Invul(tt) {
		return 0
	}
	at := attacker.Traits()
	switch tt.Kind() {
	case 2:
		if !at.HasAttack(tt) || !tr.HasDefence(tt) {
			return 1
		}
	case 3:
		if ignoreResistance {
			return 1
		}
	default:
		return 1
	}
	return clamp(float64(at.Attack(tt)-tr.Defence(tt))+1.0, 0.05, 2.0)
}

// weaponTraits maps weapon types to their traits (WeaponType.getTraitType).
var weaponTraits = [data.NumWeaponTypes]data.TraitType{
	data.WeaponSword:      data.TraitSword,
	data.WeaponBlunt:      data.TraitBlunt,
	data.WeaponDagger:     data.TraitDagger,
	data.WeaponBow:        data.TraitBow,
	data.WeaponPole:       data.TraitPole,
	data.WeaponNone:       data.TraitNone,
	data.WeaponDual:       data.TraitDual,
	data.WeaponEtc:        data.TraitEtc,
	data.WeaponFist:       data.TraitFist,
	data.WeaponDualfist:   data.TraitDualfist,
	data.WeaponFishingrod: data.TraitNone,
	data.WeaponFlag:       data.TraitNone,
	data.WeaponOwnthing:   data.TraitNone,
	data.WeaponDualdagger: data.TraitDualdagger,
}

// WeaponTrait returns the trait of a weapon type.
func WeaponTrait(w data.WeaponType) data.TraitType {
	if int(w) < len(weaponTraits) {
		return weaponTraits[w]
	}
	return data.TraitNone
}

// AttackTraitBonus is Formulas.calcAttackTraitBonus: the weapon trait bonus
// times the weakness bonuses, within 0.05 and 2 (see stats.AttackTraitBonus).
func AttackTraitBonus(attacker, target Creature) float64 {
	weapon := stats.WeaponTraitBonus(target.Traits().Defence(WeaponTrait(attacker.AttackType())))
	var weak []float64
	for tt := data.TraitType(0); tt < data.NumTraitTypes; tt++ {
		if tt.Kind() == 2 {
			weak = append(weak, GeneralTraitBonus(attacker, target, tt, true))
		}
	}
	return stats.AttackTraitBonus(weapon, weak...)
}

// The element stats in Mobius element order: fire, water, wind, earth,
// holy, dark.
var (
	elementPowers = [...]stats.Stat{stats.FirePower, stats.WaterPower, stats.WindPower, stats.EarthPower, stats.HolyPower, stats.DarkPower}
	elementRes    = [...]stats.Stat{stats.FireRes, stats.WaterRes, stats.WindRes, stats.EarthRes, stats.HolyRes, stats.DarkRes}
)

// attackElement is CreatureStat.getAttackElement: the strongest element
// power above 0 (the first on a tie), -2 for none.
func attackElement(c Creature) int {
	elem, best := -2, 0
	for i, st := range elementPowers {
		if v := javaInt(calc(c, st, 0)); v > best {
			elem, best = i, v
		}
	}
	return elem
}

// elementValue is getAttackElementValue or getDefenseElementValue: 0 for no
// element (the templates' base values are 0 in the Interlude data).
func elementValue(c Creature, table *[6]stats.Stat, elem int) int {
	if elem < 0 || elem >= len(table) {
		return 0
	}
	return javaInt(calc(c, table[elem], 0))
}

// AttributeBonus is Formulas.calcAttributeBonus: the skill's element (with
// its element power) or the attacker's strongest one, against the target's
// resistance to it. With diff their difference: 1.025 + sqrt(diff³ / 2) ×
// 0.0001 up to 1.25, 0.975 − sqrt(|diff|³ / 2) × 0.0001 down to 0.75, 1 at
// 0.
func AttributeBonus(attacker, target Creature, sk *Skill) float64 {
	var att, def int
	if sk != nil && sk.Element != -1 {
		att = elementValue(attacker, &elementPowers, sk.Element) + sk.ElementPower
		def = elementValue(target, &elementRes, sk.Element)
	} else {
		e := attackElement(attacker)
		att = elementValue(attacker, &elementPowers, e)
		def = elementValue(target, &elementRes, e)
	}
	diff := att - def
	switch {
	case diff > 0:
		return math.Min(1.025+mul(math.Sqrt(pow(float64(diff), 3)/2), 0.0001), 1.25)
	case diff < 0:
		return math.Max(0.975-mul(math.Sqrt(pow(float64(-diff), 3)/2), 0.0001), 0.75)
	}
	return 1
}

// EffectSuccess is Formulas.calcEffectSuccess: whether the continuous
// effects of a skill land. Doors never get them. A debuff on a target with
// debuff immunity is resisted. Without an activate rate (or one over 99)
// or a basic property they land; otherwise the chance is
// ((magic level − target level + 3) × lvlBonusRate + activateRate + 30 −
// the target's base stat) × element × trait × M.Atk factor × buff/debuff
// vulnerability, within the skill's chances (0 when the trait gives 0),
// against Get(100). A resisted effect tells the attacker "S1 has resisted
// your S2.".
func EffectSuccess(env *Env, attacker, target Creature, sk *Skill) bool {
	if isDoor(target) {
		return false
	}
	resisted := func() bool {
		send(attacker, MsgS1HasResistedYourS2, sp.TextParam(target.Name()), SkillName(sk))
		return false
	}
	if sk.IsDebuff && target.Calc(stats.DebuffImmunity, 0, attacker, sk) > 0 {
		return resisted()
	}
	rate := sk.ActivateRate
	if rate == -1 || rate > 99 || sk.BasicProperty == data.BaseNone {
		return true
	}
	magicLevel := sk.MagicLevel
	if magicLevel <= -1 {
		magicLevel = target.Level() + 3
	}
	targetStat := target.BaseStat(sk.BasicProperty)
	baseMod := float64(((magicLevel-target.Level())+3)*sk.LvlBonusRate+rate) + 30.0 - float64(targetStat)
	elementMod := AttributeBonus(attacker, target, sk)
	traitMod := GeneralTraitBonus(attacker, target, sk.Trait, false)
	vuln := stats.BuffVuln
	if sk.IsDebuff {
		vuln = stats.DebuffVuln
	}
	buffDebuffMod := 1 + calc(target, vuln, 1)/100
	mAtkMod := 1.0
	if sk.IsMagic() {
		mAtk := attacker.MAtk(nil, nil)
		val := 0.0
		if attacker.ChargedShot(BlessedSpiritshot) {
			val = mul(mAtk, 3.0)
		}
		val += mAtk
		mAtkMod = mul(math.Sqrt(val)/target.MDef(nil, nil), 11.0)
	}
	r := mul(mul(mul(mul(baseMod, elementMod), traitMod), mAtkMod), buffDebuffMod)
	final := 0.0
	if traitMod > 0 {
		final = clamp(r, float64(sk.MinChanceOr(&env.Config)), float64(sk.MaxChanceOr(&env.Config)))
	}
	if final <= float64(env.Rand.Get(100)) {
		return resisted()
	}
	return true
}

// MagicSuccess is Formulas.calcMagicSuccess: a skill of power -1 always
// succeeds; otherwise the rate is 100 − round(1.3^(target level − level) ×
// magicSuccessRes), where level is the skill's magic level
// (MagicSuccessBySkillMagicLevel, when above 0) or the attacker's, against
// Get(100).
func MagicSuccess(env *Env, attacker, target Creature, sk *Skill) bool {
	if sk.Power == -1 {
		return true
	}
	lvl := attacker.Level()
	if env.Config.MagicSuccessBySkillMagicLevel && sk.MagicLevel > 0 {
		lvl = sk.MagicLevel
	}
	lvlMod := pow(1.3, target.Level()-lvl)
	resMod := target.Calc(stats.MagicSuccRes, 1, nil, sk)
	rate := 100 - javaRoundF(float32(mul(lvlMod, resMod)))
	return env.Rand.Get(100) < rate
}

// BuffDebuffReflection is Formulas.calcBuffDebuffReflection: a debuff with
// an activate rate bounces when the target's reflect stat beats Get(100).
func BuffDebuffReflection(env *Env, target Creature, sk *Skill) bool {
	if !sk.IsDebuff || sk.ActivateRate == -1 {
		return false
	}
	st := stats.ReflectSkillPhysic
	if sk.IsMagic() {
		st = stats.ReflectSkillMagic
	}
	return target.Calc(st, 0, nil, sk) > float64(env.Rand.Get(100))
}

// PhysicalSkillEvasion is Formulas.calcPhysicalSkillEvasion: a physical
// non-debuff skill is dodged when Get(100) is under the target's
// pSkillEvas; both players are told.
func PhysicalSkillEvasion(env *Env, attacker, target Creature, sk *Skill) bool {
	if sk.IsMagic() || sk.IsDebuff {
		return false
	}
	if float64(env.Rand.Get(100)) < target.Calc(stats.PSkillEvas, 0, nil, sk) {
		if isPlayer(attacker) {
			send(attacker, MsgS1DodgesTheAttack, sp.TextParam(target.Name()))
		}
		if isPlayer(target) {
			send(target, MsgYouDodgeS1sAttack, sp.TextParam(attacker.Name()))
		}
		return true
	}
	return false
}

// ShieldUse is Formulas.calcShldUse for a skill: no block for skills that
// ignore shields or a target without a shield; the rate is shieldRate × the
// DEX bonus, × 1.3 against a bow, and the attacker must stand within the
// shield angle. Get(100) above 100 − PerfectShieldBlockRate is a perfect
// block, else a second Get(100) under the rate a block. A blocking player
// is told when sendMsg is set.
func ShieldUse(env *Env, attacker, target Creature, sk *Skill, sendMsg bool) stats.ShieldResult {
	if sk != nil && sk.IgnoreShield {
		return stats.ShieldFailed
	}
	if !target.HasShield() {
		return stats.ShieldFailed
	}
	rate := mul(target.Calc(stats.RShld, 0, attacker, nil), target.StatBonus(data.BaseDEX))
	if rate <= 1e-6 {
		return stats.ShieldFailed
	}
	side := javaInt(calc(target, stats.ShieldDefAngle, 0)) + 120
	if side < 360 && math.Abs(angleFrom(target.Location(), attacker.Location())-float64(target.Heading())/headingUnits) > float64(side/2) {
		return stats.ShieldFailed
	}
	if w, ok := attacker.ActiveWeapon(); ok && w.Type == data.WeaponBow {
		rate = mul(rate, 1.3)
	}
	res := stats.ShieldFailed
	switch {
	case rate > 0 && 100-env.Config.PerfectShieldBlockRate < env.Rand.Get(100):
		res = stats.ShieldPerfect
	case rate > float64(env.Rand.Get(100)):
		res = stats.ShieldSucceed
	}
	if sendMsg && isPlayer(target) {
		switch res {
		case stats.ShieldSucceed:
			send(target, MsgShieldDefenseSucceeded)
		case stats.ShieldPerfect:
			send(target, MsgExcellentShieldDefense)
		}
	}
	return res
}

// SkillCrit is Formulas.calcCrit for a skill: baseCritRate × 10 × the STR
// bonus, through the target's defCritRate, plus its defCritRateAdd,
// against Get(1000).
func SkillCrit(env *Env, attacker, target Creature, sk *Skill) bool {
	rate := mul(float64(sk.BaseCritRate*10), attacker.StatBonus(data.BaseSTR))
	return calc(target, stats.DefCritRate, rate)+calc(target, stats.DefCritRateAdd, 0) > float64(env.Rand.Get(1000))
}

// MCrit is Formulas.calcMCrit: a magic critical hit when the rate beats
// Get(1000).
func MCrit(env *Env, rate float64) bool { return rate > float64(env.Rand.Get(1000)) }

// AtkBreak is Formulas.calcAtkBreak: whether damage breaks the target's
// cast (CancelCastByHit) or bow attack (CancelBowByHit). The rate is 15 +
// sqrt(13 × damage) − (MEN bonus × 100 − 100) through the target's cancel
// stat, within 1 and 99, against Get(100); nothing is drawn when the target
// cannot break.
func AtkBreak(env *Env, target Creature, dmg float64) bool {
	st := target.BreakState()
	if st.Channeling {
		return false
	}
	init := 0.0
	if env.Config.CancelCastByHit && st.Casting {
		init = 15
	}
	if env.Config.CancelBowByHit && st.BowAttacking {
		init = 15
	}
	if target.IsRaid() || target.IsInvul() || st.Raid || st.Invul || init <= 0 {
		return false
	}
	init += math.Sqrt(mul(13, dmg))
	init -= mul(target.StatBonus(data.BaseMEN), 100) - 100
	rate := clamp(calc(target, stats.Cancel, init), 1, 99)
	return float64(env.Rand.Get(100)) < rate
}

// PhysSkillDamage is Formulas.calcPhysDam for a skill: (P.Atk × soulshot +
// power) × 76 × position / P.Def (a critical hit: × 2 × critDmg ×
// critDmgPos × the target's defCritDamage, plus critDmgAdd × 77 / P.Def and
// the target's defCritDamageAdd), then the trait bonus, the random damage
// (twice with RandomizePhysicalSkillDamage), the PvP, skill power, element
// and PvE factors. A perfect block deals 1 without drawing.
func PhysSkillDamage(env *Env, attacker, target Creature, sk *Skill, shld stats.ShieldResult, crit, ss bool) float64 {
	pvp := isPlayable(attacker) && isPlayable(target)
	pve := isPlayable(attacker) && isAttackable(target)
	prox := 1.1
	switch {
	case isBehind(attacker, target):
		prox = 1.2
	case isInFrontOf(attacker, target):
		prox = 1
	}
	damage := attacker.PAtk(target)
	defence := target.PDef(attacker)
	if pvp {
		defence = mul(defence, calc(target, stats.PvpPhysSkillsDef, 1))
	}
	switch shld {
	case stats.ShieldSucceed:
		if !env.Config.ShieldBlocks {
			defence += float64(target.ShieldDef())
		}
	case stats.ShieldPerfect:
		return 1
	}
	boost := 1.0
	if ss {
		boost = 2
	}
	damage = mul(damage, boost) + sk.PowerFor(attacker, target, pvp, pve)
	if crit {
		mod := mul(mul(mul(2, attacker.Calc(stats.CritDmg, 1, target, sk)), attacker.Calc(stats.CritDmgPos, 1, target, sk)), target.Calc(stats.DefCritDamage, 1, target, nil))
		damage = mul(mod, mul(mul(76, damage), prox)/defence)
		damage += mul(attacker.Calc(stats.CritDmgAdd, 0, target, sk), 77) / defence
		damage += target.Calc(stats.DefCritDamageAdd, 0, target, sk)
	} else {
		damage = mul(mul(76, damage), prox) / defence
	}
	damage = mul(damage, AttackTraitBonus(attacker, target))
	if env.Config.RandomizePhysicalSkillDamage {
		damage = mul(damage, attacker.RandomDamageMultiplier(env.Rand))
	}
	damage = mul(damage, attacker.RandomDamageMultiplier(env.Rand))
	if shld != stats.ShieldFailed && env.Config.ShieldBlocks {
		damage -= float64(target.ShieldDef())
		if damage < 0 {
			damage = 0
		}
	}
	if damage > 0 && damage < 1 {
		damage = 1
	} else if damage < 0 {
		damage = 0
	}
	if pvp {
		damage = mul(damage, calc(attacker, stats.PvpPhysSkillsDmg, 1))
	}
	damage = calc(attacker, stats.PhysicalSkillPower, damage)
	damage = mul(damage, AttributeBonus(attacker, target, sk))
	if isAttackable(target) {
		if w, ok := attacker.ActiveWeapon(); ok && w.Type == data.WeaponBow {
			damage = mul(damage, calc(attacker, stats.PveBowSkillsDmg, 1))
		} else {
			damage = mul(damage, calc(attacker, stats.PvePhysDmg, 1))
		}
	}
	return damage
}

// MagicDamage is Formulas.calcMagicDam: 91 × sqrt(M.Atk × shots) / M.Def ×
// power. With MagicFailures a failed MagicSuccess roll is rolled again for
// a player: a success within 9 levels halves the damage ("Your attack has
// failed."), anything else makes it 1 ("S1 has resisted your S2."); a player
// target is told it resisted. A magic critical hit multiplies by 2.5
// between players, else 3, and mCritPower. Then the random damage, the PvP,
// element and PvE factors. A perfect block deals 1 without drawing.
func MagicDamage(env *Env, attacker, target Creature, sk *Skill, shld stats.ShieldResult, sps, bss, mcrit bool) float64 {
	mDef := target.MDef(attacker, sk)
	switch shld {
	case stats.ShieldSucceed:
		mDef += float64(target.ShieldDef())
	case stats.ShieldPerfect:
		return 1
	}
	mAtk := attacker.MAtk(target, sk)
	pvp := isPlayable(attacker) && isPlayable(target)
	pve := isPlayable(attacker) && isAttackable(target)
	if pvp {
		st := stats.PvpPhysSkillsDef
		if sk.IsMagic() {
			st = stats.PvpMagicalDef
		}
		mDef = mul(mDef, calc(target, st, 1))
	}
	switch {
	case bss:
		mAtk = mul(mAtk, 4)
	case sps:
		mAtk = mul(mAtk, 2)
	}
	damage := mul(mul(91, math.Sqrt(mAtk))/mDef, sk.PowerFor(attacker, target, pvp, pve))
	drain := sk.HasEffectType(TypeHPDrain)
	switch {
	case env.Config.MagicFailures && !MagicSuccess(env, attacker, target, sk):
		if isPlayer(attacker) {
			if MagicSuccess(env, attacker, target, sk) && target.Level()-attacker.Level() <= 9 {
				if drain {
					send(attacker, MsgDrainHalfSuccessful)
				} else {
					send(attacker, MsgYourAttackHasFailed)
				}
				damage /= 2
			} else {
				send(attacker, MsgS1HasResistedYourS2, sp.TextParam(target.Name()), SkillName(sk))
				damage = 1
			}
		}
		if isPlayer(target) {
			if drain {
				send(target, MsgYouResistedS1sDrain, sp.TextParam(attacker.Name()))
			} else {
				send(target, MsgYouResistedS1sMagic, sp.TextParam(attacker.Name()))
			}
		}
	case mcrit:
		if isPlayer(attacker) && isPlayer(target) {
			damage = mul(damage, 2.5)
		} else {
			damage = mul(damage, 3)
		}
		damage = mul(damage, calc(attacker, stats.MCritPower, 1))
	}
	if env.Config.RandomizeMagicalSkillDamage {
		damage = mul(damage, attacker.RandomDamageMultiplier(env.Rand))
	}
	if pvp {
		st := stats.PvpPhysSkillsDmg
		if sk.IsMagic() {
			st = stats.PvpMagicalDmg
		}
		damage = mul(damage, calc(attacker, st, 1))
	}
	damage = mul(damage, AttributeBonus(attacker, target, sk))
	if isAttackable(target) {
		damage = mul(damage, calc(attacker, stats.PveMagicalDmg, 1))
	}
	return damage
}

// DamageReflected is Formulas.calcDamageReflected: a melee physical skill
// is counter-attacked when Get(100) is under the target's
// vengeanceSkillPhysicalDamage. The attacker takes P.Atk × 10 × 70 / P.Def,
// times the weapon trait, skill trait and element bonuses, twice on a
// critical hit; both players are told.
func DamageReflected(env *Env, attacker, target Creature, sk *Skill, crit bool) {
	if sk.IsMagic() || sk.CastRange > meleeAttackRange {
		return
	}
	chance := target.Calc(stats.VengeancePdam, 0, target, sk)
	if !(float64(env.Rand.Get(100)) < chance) {
		return
	}
	if isPlayer(target) {
		send(target, MsgYouCounterAttackS1sAttack, sp.TextParam(attacker.Name()))
	}
	if isPlayer(attacker) {
		send(attacker, MsgS1IsPerformingCounterAttack, sp.TextParam(target.Name()))
	}
	dmg := mul(mul(target.PAtk(attacker), 10.0), 70.0) / attacker.PDef(target)
	dmg = mul(dmg, stats.WeaponTraitBonus(target.Traits().Defence(WeaponTrait(attacker.AttackType()))))
	dmg = mul(dmg, GeneralTraitBonus(attacker, target, sk.Trait, false))
	dmg = mul(dmg, AttributeBonus(attacker, target, sk))
	attacker.ReduceCurrentHP(dmg, target, sk)
	if crit {
		attacker.ReduceCurrentHP(dmg, target, sk)
	}
}
