package stats

import "time"

// SwingTiming is when the parts of one auto-attack swing happen, counted
// from the moment it starts (Creature.doAttack). Mobius schedules in whole
// milliseconds, so every duration is a whole number of them.
type SwingTiming struct {
	// Interval is calculateTimeBetweenAttacks: (int)(500000 / attack
	// speed) ms.
	Interval time.Duration
	// Reuse is calculateReuseTime: (int)(reuseDelay × 333 / attack speed)
	// ms for a weapon with a reuse delay (bows), else 0.
	Reuse time.Duration
	// Hits holds when each hit lands (the HitTask delays); NumHits says how
	// many are used. With t = Interval/2 (integer halves): simple and pole
	// hit at t, dual at t/2 and t, a bow at Interval.
	Hits    [2]time.Duration
	NumHits int
	// End is when the creature stops attacking (_attackEndTime,
	// isAttackingNow): Interval, for bows Interval/2 + Reuse/2.
	End time.Duration
	// Ready is when the AI may act again and the next swing may start
	// (notifyActionReadyToAct): Interval + Reuse.
	Ready time.Duration
	// Gauge is the bow's SetupGauge time players see, Interval + Reuse; 0
	// for other shapes.
	Gauge time.Duration
	// BowLockTicks is how many game ticks (100 ms) after the current one
	// the bow cannot shoot again, (Interval + Reuse) / 100
	// (_disableBowAttackEndTime); 0 for other shapes.
	BowLockTicks int
}

// SwingTimingOf returns the timing of a swing at attack speed pAtkSpd
// (Combatant.PAtkSpd, so capped for players) with a weapon of reuseDelay
// (0 without one).
func SwingTimingOf(pAtkSpd float64, shape AttackShape, reuseDelay int) SwingTiming {
	timeAtk := int(javaInt(500000 / pAtkSpd))
	timeToHit := timeAtk / 2
	reuse := 0
	if reuseDelay != 0 {
		reuse = int(javaInt(float64(int32(reuseDelay)*333) / pAtkSpd)) //nolint:gosec // a weapon's reuse delay, in Java int arithmetic
	}
	ms := func(v int) time.Duration { return time.Duration(v) * time.Millisecond }
	t := SwingTiming{
		Interval: ms(timeAtk),
		Reuse:    ms(reuse),
		NumHits:  1,
		End:      ms(timeAtk),
		Ready:    ms(timeAtk + reuse),
	}
	switch shape {
	case ShapeBow:
		t.Hits[0] = ms(timeAtk)
		t.End = ms(timeToHit + reuse/2)
		t.Gauge = ms(timeAtk + reuse)
		t.BowLockTicks = (timeAtk + reuse) / 100
	case ShapeDual:
		t.Hits = [2]time.Duration{ms(timeToHit / 2), ms(timeToHit)}
		t.NumHits = 2
	default:
		t.Hits[0] = ms(timeToHit)
	}
	return t
}

// Hit is one rolled hit of a swing, as the Attack packet and the hit timer
// need it.
type Hit struct {
	Miss   bool
	Crit   bool
	Shield ShieldResult
	// Damage is what the hit takes from the target: (int) PhysDamage, then
	// halved (integer division) for dual hits, scaled for pole splash and
	// bow distance. 0 on a miss.
	Damage int
}

// SwingResult is a whole rolled swing: its timing and its hits.
type SwingResult struct {
	Timing SwingTiming
	// Hits holds Timing.NumHits hits; Hits[i] lands at Timing.Hits[i].
	Hits [2]Hit
}

// Landed reports whether at least one hit did not miss (doAttack's hitted):
// a swing that only missed aborts the attack, one that hit discharges the
// soulshot.
func (r *SwingResult) Landed() bool {
	for i := range r.Timing.NumHits {
		if !r.Hits[i].Miss {
			return true
		}
	}
	return false
}

// AutoAttack rolls one swing of an auto-attack as Creature.doAttack does
// after its checks, with the attack shape and reuse delay of the attacker.
// The draws, in order:
//
//   - simple, pole (main target) and bow: HitMiss, then for a hit
//     ShieldUse, Crit and PhysDamage;
//   - dual: HitMiss for both hits first, then ShieldUse, Crit and
//     PhysDamage for each hit that did not miss, the first hit first.
//
// A bow's canUseRangeWeapon (arrows, MP, its reduced-MP roll) comes before
// and is the caller's. With DistanceBowDamage a bow hit's damage is
// (int)(damage × (distance / 4000 + 0.8)).
func (c *CombatRules) AutoAttack(rnd Rand, s *Strike) SwingResult {
	a := &s.Attacker
	res := SwingResult{Timing: SwingTimingOf(a.Stats.PAtkSpd(), a.Attack, a.ReuseDelay)}
	switch a.Attack {
	case ShapeDual:
		miss1 := c.HitMiss(rnd, s)
		miss2 := c.HitMiss(rnd, s)
		res.Hits[0] = c.rollHit(rnd, s, miss1)
		res.Hits[0].Damage /= 2
		res.Hits[1] = c.rollHit(rnd, s, miss2)
		res.Hits[1].Damage /= 2
	case ShapeBow:
		h := c.rollHit(rnd, s, c.HitMiss(rnd, s))
		if !h.Miss && c.cfg.DistanceBowDamage {
			h.Damage = int(javaInt(float64(h.Damage) * (distance3D(a, &s.Target)/4000 + 0.8)))
		}
		res.Hits[0] = h
	default:
		res.Hits[0] = c.SimpleHit(rnd, s, 100)
	}
	return res
}

// SimpleHit rolls one hit as Creature.doAttackHitSimple does: HitMiss, then
// for a hit ShieldUse, Crit and PhysDamage. percent scales the damage of a
// pole's extra targets: (int)(damage × percent / 100), 85 for the first,
// divided by 1.15 for each next one; 100 keeps it.
func (c *CombatRules) SimpleHit(rnd Rand, s *Strike, percent float64) Hit {
	h := c.rollHit(rnd, s, c.HitMiss(rnd, s))
	if !h.Miss && percent != 100 {
		h.Damage = int(javaInt(mul(float64(h.Damage), percent) / 100))
	}
	return h
}

// rollHit rolls the shield, the critical hit and the damage of a hit that
// did not miss.
func (c *CombatRules) rollHit(rnd Rand, s *Strike, miss bool) Hit {
	if miss {
		return Hit{Miss: true}
	}
	shld := c.ShieldUse(rnd, s)
	crit := c.Crit(rnd, s)
	return Hit{
		Crit:   crit,
		Shield: shld,
		Damage: int(javaInt(c.PhysDamage(rnd, s, shld, crit))),
	}
}
