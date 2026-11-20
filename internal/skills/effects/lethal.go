package effects

import (
	"math"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// lethal is Mobius Lethal, the lethal strikes of blows: with the element and
// trait bonuses as mult, one roll of Get(1000000) / 10000 under fullLethal ×
// mult leaves a player at 1 CP and 1 HP and a monster or summon at 1 HP
// ("Lethal Strike!"); under (full + halfLethal) × mult it takes a player's
// CP to 1 and halves a monster's or summon's HP ("Half-Kill!"). Nothing
// happens from a GM who may not deal damage, with a magic level more than 6
// under the target's level, or on a target that is not lethalable or is
// invulnerable.
type lethal struct {
	skills.Base
	full, half int
}

func init() {
	register(data.EffectLethal, func(def *data.SkillEffect) (skills.Effect, error) {
		full, err := def.Params.Int("fullLethal", 0)
		if err != nil {
			return nil, err
		}
		half, err := def.Params.Int("halfLethal", 0)
		if err != nil {
			return nil, err
		}
		return &lethal{Base: skills.NewBase(def), full: full, half: half}, nil
	})
}

func (*lethal) Instant() bool { return true }

func (e *lethal) OnStart(info *skills.BuffInfo) {
	env, effector, effected, sk := info.Env(), info.Effector(), info.Effected(), info.Skill()
	if isPlayer(effector) && !effector.AsPlayer().CanGiveDamage() {
		return
	}
	if sk.MagicLevel < effected.Level()-6 {
		return
	}
	if !effected.IsLethalable() || effected.IsInvul() {
		return
	}
	attr := skills.AttributeBonus(effector, effected, sk)
	trait := skills.GeneralTraitBonus(effector, effected, sk.Trait, false)
	mult := math.Max(0, attr*trait)
	fullChance := lethalClamp(float64(float64(e.full)*mult), 0, 100)
	halfChance := lethalClamp(float64(float64(e.half)*mult), 0, 100)
	roll := float64(env.Rand.Get(1000000)) / 10000.0
	halfSum := lethalClamp(fullChance+halfChance, 0, 100)
	monsterOrSummon := isMonsterB(effected) || isSummon(effected)
	if roll < fullChance {
		switch {
		case isPlayer(effected):
			effected.NotifyDamageReceived(effected.CurrentHP()-1, effector, sk, true, false)
			effected.SetCurrentCP(1)
			effected.SetCurrentHP(1)
			skills.Send(effected, msgLethalStrike)
		case monsterOrSummon:
			effected.NotifyDamageReceived(effected.CurrentHP()-1, effector, sk, true, false)
			effected.SetCurrentHP(1)
		}
		skills.Send(effector, msgLethalStrikeSuccessful)
		return
	}
	if roll < halfSum {
		switch {
		case isPlayer(effected):
			effected.SetCurrentCP(1)
			effected.SendText("Half-Kill!")
			effected.SendText("Your CP was drained because you were hit with a Half-Kill skill.")
		case monsterOrSummon:
			newHP := float64(effected.CurrentHP() * 0.5)
			effected.NotifyDamageReceived(effected.CurrentHP()-newHP, effector, sk, true, false)
			effected.SetCurrentHP(newHP)
		}
		if isPlayer(effector) {
			effector.SendText("Half-Kill!")
		}
	}
}

// lethalClamp is Java's Math.clamp(double, double, double).
func lethalClamp(v, lo, hi float64) float64 { return math.Min(hi, math.Max(v, lo)) }
