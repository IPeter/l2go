package main

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// Modes of the bots (-mode).
const (
	// modeWander walks around the start point, chats and talks to NPCs.
	modeWander = "wander"
	// modeHunt hunts the monsters around the start point, casts an attack
	// skill now and then, holds buffs (-buffs), dies and restarts.
	modeHunt = "hunt"
)

// Character classes of the bots (-class).
const (
	classFighter = "fighter"
	classMystic  = "mystic"
)

// skillWindStrike is the attack skill a new Human Mystic knows.
const skillWindStrike int32 = 1177

// huntBuffs are the buffs a hunting bot holds with -buffs, given with the
// GM command //buff at their highest level: ten buffs of different stack
// groups (Might, Shield, Wind Walk, Bless the Body, Bless the Soul, Mental
// Shield, Acumen, Empower, Focus, Haste).
var huntBuffs = []int32{1068, 1040, 1204, 1045, 1048, 1035, 1085, 1059, 1077, 1086}

// Timing of a hunting bot.
const (
	// restartDelay is how long a dead bot lies before it restarts in the
	// village, as a player reads the window first.
	restartDelay = 2 * time.Second
	// castTimeout bounds a cast: a refused cast (no MP, reuse) has no
	// answer of its own, and the bot should not stand still for -timeout.
	castTimeout = 5 * time.Second
	// buffRetry is how long a bot waits after its GM buffs failed (no GM
	// rights) before it tries again.
	buffRetry = time.Minute
	// minCastMP is the share of its MP a bot keeps before it casts.
	minCastMP = 0.25
)

// hunt is one action of a hunting bot: restart when dead, refresh missing
// buffs, otherwise fight the monster it hunts (cast the attack skill when
// its interval has passed, else attack) or walk around the start point
// when no monster is in reach.
func (b *bot) hunt(ctx context.Context, gc *testclient.GameConn) error {
	w := gc.World()
	me := w.Me()
	if me.Dead {
		return b.restart(ctx, gc)
	}
	now := time.Now()
	b.checkKill(w, now)
	if b.cfg.buffs > 0 && now.After(b.buffRetryAt) && len(missingBuffs(w.Effects(), huntBuffs[:b.cfg.buffs])) > 0 {
		return b.buff(ctx, gc)
	}
	target, ok := b.prey(w, now)
	if !ok {
		return b.roam(ctx, gc)
	}
	if b.cfg.skill != 0 && b.cfg.cast > 0 && now.Sub(b.lastCast) >= b.cfg.cast && enoughMana(me) {
		return b.cast(ctx, gc, target)
	}
	if b.attacking != target.ObjectID {
		return b.attack(ctx, gc, target)
	}
	return nil
}

// checkKill ends the fight with the hunted monster when it died (a kill)
// or left sight.
func (b *bot) checkKill(w *testclient.World, now time.Time) {
	if b.target == 0 {
		return
	}
	o, ok := w.Object(b.target)
	if ok && !o.Dead {
		return
	}
	if ok {
		b.stats.Count("kill")
		b.stats.Observe("kill", now.Sub(b.fightStart))
	}
	b.target, b.attacking = 0, 0
}

// prey returns the monster the bot hunts: the current one, or the nearest
// living monster within -hunt-range of the start point.
func (b *bot) prey(w *testclient.World, now time.Time) (testclient.Object, bool) {
	if b.target != 0 {
		if o, ok := w.Object(b.target); ok {
			return o, true
		}
	}
	o, ok := nearestMonster(w.Me().Pos, b.start, w.Npcs(), b.cfg.huntRange, b.avoid)
	if !ok {
		return o, false
	}
	b.target, b.attacking, b.fightStart = o.ObjectID, 0, now
	return o, true
}

// nearestMonster returns the living attackable monster of objs nearest to
// me among those within maxDist of home, leaving out avoid (a monster the
// bot could not attack). Ties go to the lower object ID.
func nearestMonster(me, home testclient.Position, objs []testclient.Object, maxDist float64, avoid int32) (testclient.Object, bool) {
	var (
		best     testclient.Object
		bestDist = math.Inf(1)
		found    bool
	)
	for _, o := range objs {
		if o.Kind != testclient.KindNpc || !o.Attackable || o.Dead || o.ObjectID == avoid || o.Pos.DistanceTo(home) > maxDist {
			continue
		}
		d := o.Pos.DistanceTo(me)
		if !found || d < bestDist || (d == bestDist && o.ObjectID < best.ObjectID) {
			best, bestDist, found = o, d, true
		}
	}
	return best, found
}

// missingBuffs returns the buffs of want the effects do not have.
func missingBuffs(effects []testclient.Effect, want []int32) []int32 {
	var missing []int32
	for _, id := range want {
		has := false
		for _, e := range effects {
			if e.SkillID == id {
				has = true
				break
			}
		}
		if !has {
			missing = append(missing, id)
		}
	}
	return missing
}

// enoughMana reports whether the character has the MP to cast and keep a
// reserve.
func enoughMana(me testclient.Me) bool {
	return me.MaxMP > 0 && float64(me.MP) >= minCastMP*float64(me.MaxMP)
}

// attack attacks the hunted monster and waits for the first swing that hits
// it ("attack": AttackRequest to the first Attack, the walk included). A
// monster the bot cannot attack is avoided next time.
func (b *bot) attack(ctx context.Context, gc *testclient.GameConn, target testclient.Object) error {
	t0 := time.Now()
	if _, err := gc.Attack(ctx, target.ObjectID); err != nil {
		b.avoid, b.target, b.attacking = target.ObjectID, 0, 0
		return staged("attack", err)
	}
	b.stats.Observe("attack", time.Since(t0))
	b.attacking = target.ObjectID
	return nil
}

// cast casts the attack skill on the hunted monster and waits until it
// takes effect ("cast": RequestMagicSkillUse to MagicSkillLaunched, the
// cast time included). The bot attacks again after it.
func (b *bot) cast(ctx context.Context, gc *testclient.GameConn, target testclient.Object) error {
	b.lastCast = time.Now()
	b.attacking = 0
	cctx, cancel := context.WithTimeout(ctx, castTimeout)
	defer cancel()
	if _, err := gc.CastOn(cctx, target.ObjectID, b.cfg.skill); err != nil {
		return staged("cast", err)
	}
	b.stats.Observe("cast", time.Since(b.lastCast))
	b.stats.Count("cast")
	return nil
}

// buff gives the bot its missing buffs with //buff on itself and waits for
// their icons ("buff"). Without GM rights nothing comes; the bot then tries
// again after buffRetry.
func (b *bot) buff(ctx context.Context, gc *testclient.GameConn) error {
	t0 := time.Now()
	want := huntBuffs[:b.cfg.buffs]
	if err := gc.TargetSelf(ctx); err != nil {
		return staged("buff", err)
	}
	b.attacking = 0
	for _, id := range missingBuffs(gc.World().Effects(), want) {
		if err := gc.GMCommand(ctx, "buff "+strconv.Itoa(int(id))); err != nil {
			return staged("buff", err)
		}
	}
	if err := pollUntil(ctx, func() bool { return len(missingBuffs(gc.World().Effects(), want)) == 0 }); err != nil {
		b.buffRetryAt = time.Now().Add(buffRetry)
		return staged("buff", err)
	}
	b.stats.Observe("buff", time.Since(t0))
	return nil
}

// restart gets the dead bot up in the village ("restart": RequestRestartPoint
// to the UserInfo after the teleport). It then hunts around its start point
// again, walking back first.
func (b *bot) restart(ctx context.Context, gc *testclient.GameConn) error {
	b.stats.Count("death")
	b.target, b.attacking = 0, 0
	if err := sleep(ctx, restartDelay); err != nil {
		return err
	}
	t0 := time.Now()
	if _, err := gc.RestartPoint(ctx, testclient.RestartVillage); err != nil {
		return staged("restart", err)
	}
	b.stats.Observe("restart", time.Since(t0))
	return nil
}

// roam walks to a random point around the start point when no monster is
// in reach ("move"). After a restart in the village this is the walk back.
func (b *bot) roam(ctx context.Context, gc *testclient.GameConn) error {
	me := gc.World().Me()
	if me.Moving && me.Dest.DistanceTo(b.start) <= b.cfg.radius {
		return nil // still on the way
	}
	dest := randomPoint(b.rng, b.start, b.cfg.radius)
	t0 := time.Now()
	if _, err := gc.MoveTo(ctx, dest.X, dest.Y, dest.Z); err != nil {
		return staged("move", err)
	}
	b.stats.Observe("move", time.Since(t0))
	return nil
}

// pollInterval is how often pollUntil looks at the world model.
const pollInterval = 100 * time.Millisecond

// pollUntil calls done every pollInterval until it is true or ctx ends.
func pollUntil(ctx context.Context, done func() bool) error {
	t := time.NewTicker(pollInterval)
	defer t.Stop()
	for !done() {
		select {
		case <-t.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
