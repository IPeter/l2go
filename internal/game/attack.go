package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The attack feature (attack*.go) ports Mobius's auto-attack (Plan 09,
// Phase 6.4):
//
//   - attack.go: one swing of any creature (Creature.doAttack), its hits
//     (onHitTimer) and the hooks other features build on.
//   - attack_stance.go: the combat stance (AutoAttackStart/Stop and
//     AttackStanceTaskManager).
//   - attack_player.go: the player's intentions IDLE, ATTACK and MOVE_TO
//     (PlayerAI), the attack follow and the click handlers' attack paths.
//   - attack_logout.go: no restart or logout in combat (decision F14).
//
// The API for the monster AI, death and skills:
//
//   - DoAttack swings a creature's weapon at a target once (any attacker);
//     the AI of an NPC calls it when its target is in reach, and hears of
//     the swing's end through OnNpcAttackEvent (NpcAttackReady), as Mobius's
//     notifyActionReadyToAct.
//   - AttackingNow, AbortAttack and BreakAttack are Mobius's
//     isAttackingNow, abortAttack and breakAttack.
//   - StartAutoAttack, StopAutoAttack and InAttackStance are Mobius's
//     clientStartAutoAttack, clientStopAutoAttack and hasAttackStanceTask.
//   - OnHit runs when a landing hit is about to take HP (Mobius
//     Attackable.addDamage: hate); OnAttacked after it took them (Mobius
//     notifyActionAttacked in onHitTimer).
//   - OnPlayerSwing runs where Mobius flags a player for PvP.
//   - AttackIntention and IdleIntention set a player's intention.
func init() { registerFeature("attack", setupAttack) }

const (
	// attackBowShotTicks is how long a bow shot locks the bow, in game
	// ticks, until the swing sets the real lock (Mobius canUseRangeWeapon:
	// 5 × TICKS_PER_SECOND).
	attackBowShotTicks = 5 * walkTicksPerSecond
	// attackBowRetry is when a bow that could not shoot tries again (Mobius
	// schedules notifyActionReadyToAct after 1,000 ms).
	attackBowRetry = time.Second
	// attackNoWeapon is what a player hears when its weapon cannot attack
	// (Mobius doAttack).
	attackNoWeapon = "That weapon cannot perform any attacks."
)

// System messages of the attack (Mobius SystemMessageId).
const (
	sysMsgNotEnoughMP     serverpackets.SystemMessageID = 24   // "Not enough MP."
	sysMsgYouHitForDamage serverpackets.SystemMessageID = 35   // "You hit for $s1 damage."
	sysMsgNockArrow       serverpackets.SystemMessageID = 41   // "You carefully nock an arrow..."
	sysMsgYouMissed       serverpackets.SystemMessageID = 43   // "You have missed."
	sysMsgCriticalHit     serverpackets.SystemMessageID = 44   // "Critical hit!"
	sysMsgPeacefulZone    serverpackets.SystemMessageID = 85   // "You may not attack this target in a peaceful zone."
	sysMsgShieldDefence   serverpackets.SystemMessageID = 111  // "Your shield defense has succeeded."
	sysMsgOutOfArrows     serverpackets.SystemMessageID = 112  // "You have run out of arrows."
	sysMsgAttackFailed    serverpackets.SystemMessageID = 158  // "Your attack has failed."
	sysMsgCannotSeeTarget serverpackets.SystemMessageID = 181  // "Cannot see target."
	sysMsgExcellentShield serverpackets.SystemMessageID = 1281 // "Your excellent shield defense was a success!"
	sysMsgFishingPole     serverpackets.SystemMessageID = 1472 // "You look oddly at the fishing pole in disbelief..."
)

// AttackOutcome is what DoAttack did.
type AttackOutcome uint8

const (
	// AttackNone: nothing happened. The attacker swings already, the
	// target cannot be targeted, the weapon cannot attack, an NPC's bow is
	// still locked or a player ran out of arrows (its intention is then
	// IDLE).
	AttackNone AttackOutcome = iota
	// AttackSwung: the swing started; its hits and its ready time follow.
	AttackSwung
	// AttackRefused: Mobius's doAttack refused the swing and set the
	// attacker's intention ACTIVE: the target is dead (for an NPC or a
	// player attacker) or out of the surrounding regions, a peace zone is in
	// the way, or there is no line of sight. The attacker got ActionFailed;
	// an NPC's AI hears AttackRefused.
	AttackRefused
	// AttackBowWait: a player's bow cannot shoot yet (bow lock, MP); the
	// ready time comes in one second.
	AttackBowWait
)

// AttackEvent is what the attack tells an NPC's AI (OnNpcAttackEvent).
type AttackEvent uint8

const (
	// NpcAttackReady: the swing's ready time came (Mobius
	// notifyActionReadyToAct, timeAtk + reuse after the swing).
	NpcAttackReady AttackEvent = iota
	// NpcAttackCancelled: a hit found the attacker dead or its target dead
	// or gone (Mobius notifyActionCancel). An AutoAttackStop went out when
	// the NPC had no stance.
	NpcAttackCancelled
	// NpcAttackRefused: DoAttack refused the swing (AttackRefused); Mobius
	// sets the intention ACTIVE.
	NpcAttackRefused
)

// SwingHook runs when a player swings at a target outside a PvP zone, after
// its stance started: Mobius updatePvPStatus(target) there.
type SwingHook func(p *Player, target Creature)

// HitHook runs for a hit of attacker that takes damage HP from target.
type HitHook func(target, attacker Creature, damage int)

// NpcAttackHook tells an NPC's AI about its attack (AttackEvent).
type NpcAttackHook func(n *Npc, ev AttackEvent)

// TraitBonus returns Mobius calcAttackTraitBonus of attacker against target,
// ok false for none (×1). The traits come with the passive skills (F10, M7).
type TraitBonus func(attacker, target Creature) (bonus float64, ok bool)

// ArrowSupply is the inventory side of a player's bow shot.
type ArrowSupply interface {
	// EquipArrows reports whether the player has arrows for its bow in the
	// left hand, equipping them first if needed (Mobius
	// checkAndEquipArrows).
	EquipArrows(p *Player) bool
	// UseArrow takes one arrow (Mobius reduceArrowCount).
	UseArrow(p *Player)
}

// attackFeature is the attack feature's state on the Server (Server.attack).
// Loop-only.
type attackFeature struct {
	// states holds the swing state of the creatures that attacked.
	states map[Creature]*attackState
	// players holds the intentions of the players (PlayerAI).
	players map[*Player]*playerAI
	// following lists the players with an attack follow, in the order it
	// started (Mobius ATTACK_FOLLOW_CREATURES).
	following []*Player

	// stance holds when each creature's stance last started, and
	// stanceOrder the creatures in the order they entered it (Mobius
	// CREATURE_ATTACK_STANCES). autoAttacking holds the creatures that
	// showed AutoAttackStart (Mobius _clientAutoAttacking).
	stance        map[Creature]time.Time
	stanceOrder   []Creature
	autoAttacking map[Creature]bool

	swing     []SwingHook
	hit       []HitHook
	attacked  []HitHook
	npcEvents []NpcAttackHook

	traits TraitBonus
	arrows ArrowSupply
	// rnd is the source of the combat rolls: the server's (Server.Rand).
	// Tests give the attack a scripted one of its own, so that spawns and
	// the monster AI keep their draws.
	rnd rnd.Source
}

// attackState is one creature's swing state (Mobius _attackEndTime,
// _disableBowAttackEndTime and the hit and ready tasks).
type attackState struct {
	end     time.Time // the swing lasts until then (isAttackingNow)
	bowLock int64     // the game tick from which the bow shoots again
	timers  []world.TimerID
}

// setupAttack sets up the attack feature: the stance scan, the attack
// follow, the click handlers' hooks and the clean-up.
func setupAttack(s *Server) error {
	f := &s.attack
	f.states = make(map[Creature]*attackState)
	f.players = make(map[*Player]*playerAI)
	f.stance = make(map[Creature]time.Time)
	f.autoAttacking = make(map[Creature]bool)
	f.rnd = s.rnd
	if f.arrows == nil {
		f.arrows = inventoryArrows{s}
	}
	s.loop.Every(attackStanceScanPeriod, s.scanStances)
	s.loop.Every(attackFollowPeriod, s.playerFollowTick)
	s.OnPlayerLeave(s.attackLeave)
	s.OnPlayerArrived(s.attackArrived)
	s.OnForget(s.attackForget)
	s.OnTeleport(func(p *Player, _ world.Position) { s.attackTeleport(p) })
	s.OnDie(s.attackDie)
	return nil
}

// OnPlayerSwing adds fn to the functions run when a player swings at a
// target outside a PvP zone (SwingHook). Call it from a feature's setup.
func (s *Server) OnPlayerSwing(fn SwingHook) { s.attack.swing = append(s.attack.swing, fn) }

// OnHit adds fn to the functions run when a hit that landed is about to take
// HP: the target is alive, its HP not yet reduced (Mobius
// Attackable.reduceCurrentHp → addDamage: hate and notifyActionAttacked).
// Call it from a feature's setup.
func (s *Server) OnHit(fn HitHook) { s.attack.hit = append(s.attack.hit, fn) }

// OnAttacked adds fn to the functions run after a hit took HP (Mobius
// onHitTimer: target.getAI().notifyActionAttacked(attacker)); the target may
// be dead by then. Call it from a feature's setup.
func (s *Server) OnAttacked(fn HitHook) { s.attack.attacked = append(s.attack.attacked, fn) }

// OnNpcAttackEvent adds fn to the functions told about an NPC's attack
// (AttackEvent): the monster AI thinks on NpcAttackReady. Call it from a
// feature's setup.
func (s *Server) OnNpcAttackEvent(fn NpcAttackHook) {
	s.attack.npcEvents = append(s.attack.npcEvents, fn)
}

// SetTraitBonus sets where the attack trait bonus of a swing comes from;
// without one every swing deals damage as if no traits applied. Call it
// from a feature's setup.
func (s *Server) SetTraitBonus(fn TraitBonus) { s.attack.traits = fn }

// SetArrowSupply replaces the arrows of bow shots (by default the player's
// inventory). Call it from a feature's setup.
func (s *Server) SetArrowSupply(a ArrowSupply) { s.attack.arrows = a }

// DoAttack swings c's weapon at target once (Mobius Creature.doAttack) and
// reports what happened. It does not check the range: the caller walks c up
// to target first. The hits land later on timers, and the ready time comes
// timeAtk + reuse after the swing: a player's intention thinks again then,
// an NPC's AI hears NpcAttackReady. Loop-only.
func (s *Server) DoAttack(c, target Creature) AttackOutcome { return s.doAttack(c, target) }

// AttackingNow reports whether c is in the middle of a swing (Mobius
// isAttackingNow). Loop-only.
func (s *Server) AttackingNow(c Creature) bool {
	st := s.attack.states[c]
	return st != nil && st.end.After(s.Now())
}

// AbortAttack sends ActionFailed to a player in the middle of a swing
// (Mobius abortAttack; the hits still land). Loop-only.
func (s *Server) AbortAttack(c Creature) {
	if p, ok := c.(*Player); ok && s.AttackingNow(c) {
		p.SendActionFailed()
	}
}

// BreakAttack interrupts c's swing (Mobius breakAttack): a player gets
// ActionFailed and "Your attack has failed.". Loop-only.
func (s *Server) BreakAttack(c Creature) {
	if !s.AttackingNow(c) {
		return
	}
	s.AbortAttack(c)
	if p, ok := c.(*Player); ok {
		p.SendSystemMessage(sysMsgAttackFailed)
	}
}

// attackStateOf returns c's swing state, made on first use.
func (s *Server) attackStateOf(c Creature) *attackState {
	st := s.attack.states[c]
	if st == nil {
		st = &attackState{}
		s.attack.states[c] = st
	}
	return st
}

// attackTimer runs fn after d as one of c's attack tasks, which end when c
// leaves the game.
func (s *Server) attackTimer(st *attackState, d time.Duration, fn func()) {
	var id world.TimerID
	id = s.loop.After(d, func() {
		st.timers = slices.DeleteFunc(st.timers, func(x world.TimerID) bool { return x == id })
		fn()
	})
	st.timers = append(st.timers, id)
}

// readyAfter tells c it is ready to act again after d (Mobius
// notifyActionReadyToAct on a timer).
func (s *Server) readyAfter(c Creature, st *attackState, d time.Duration) {
	s.attackTimer(st, d, func() { s.readyToAct(c) })
}

// readyToAct is Mobius notifyActionReadyToAct: a player's intention thinks,
// an NPC's AI hears NpcAttackReady. Nothing happens to a creature that
// left.
func (s *Server) readyToAct(c Creature) {
	switch v := c.(type) {
	case *Player:
		if v.state == stateInGame && v.InWorld() {
			s.playerReadyToAct(v)
		}
	case *Npc:
		if v.InWorld() {
			s.npcAttackEvent(v, NpcAttackReady)
		}
	}
}

// npcAttackEvent tells the AI of n about its attack.
func (s *Server) npcAttackEvent(n *Npc, ev AttackEvent) {
	for _, fn := range s.attack.npcEvents {
		s.safely("OnNpcAttackEvent hook", func() { fn(n, ev) })
	}
}

// doAttack is Mobius Creature.doAttack.
func (s *Server) doAttack(a, t Creature) AttackOutcome {
	if a == nil || t == nil || !attackerInWorld(a) || adminParalyzed(a) {
		return AttackNone // paralyzed by //para: Mobius isAttackingDisabled
	}
	now := s.Now()
	st := s.attackStateOf(a)
	if st.end.After(now) || !creatureTargetable(t) {
		return AttackNone
	}
	p, _ := a.(*Player)
	if !a.IsDead() {
		_, npc := a.(*Npc)
		if (npc && t.IsDead()) || !s.surrounding(a, t) || (p != nil && t.IsDead()) {
			return s.refuseAttack(a)
		}
	}
	w := s.activeWeaponItem(a)
	if w != nil && !w.Weapon.AttackWeapon && (p == nil || !p.IsGM()) {
		if p != nil {
			if w.Weapon.Type == data.WeaponFishingrod {
				p.SendSystemMessage(sysMsgFishingPole)
			} else {
				p.SendMessage(attackNoWeapon)
			}
			p.SendActionFailed()
		}
		return AttackNone
	}
	// Mobius checks the peace zones of player attackers only here; other
	// attackers are not playable, so no zone stops them (F13).
	if p != nil && s.insidePeaceZone(p, t) {
		return s.refuseAttack(a)
	}
	if !s.canSee(a, t) {
		if p != nil {
			p.SendSystemMessage(sysMsgCannotSeeTarget)
		}
		return s.refuseAttack(a)
	}
	s.stopCreature(a)

	af := s.fighter(a, w)
	tf := s.fighter(t, s.activeWeaponItem(t))
	heading := int32(stats.HeadingTo(af.X, af.Y, tf.X, tf.Y)) //nolint:gosec // 0-65535
	setCreatureHeading(a, heading)
	af.Heading = int(heading)
	bow := af.Attack == stats.ShapeBow
	if bow {
		if out, ok := s.canUseRangeWeapon(a, w, st); !ok {
			return out
		}
	}

	// The Attack shows the soulshot charged at the start (shot.go).
	soulshot, grade := s.attackSoulshot(a)
	strike := &stats.Strike{Attacker: af, Target: tf, Soulshot: soulshot, Night: s.world.GameTime().IsNight()}
	if s.attack.traits != nil {
		if b, ok := s.attack.traits(a, t); ok {
			strike.SetTrait(b)
		}
	}
	res := s.combat.AutoAttack(s.attack.rnd, strike)
	timing := res.Timing
	st.end = now.Add(timing.End)
	hits := res.Hits[:timing.NumHits]

	// calcShldUse tells a blocking player at once.
	if tp, ok := t.(*Player); ok {
		for _, h := range hits {
			switch h.Shield {
			case stats.ShieldSucceed:
				tp.SendSystemMessage(sysMsgShieldDefence)
			case stats.ShieldPerfect:
				tp.SendSystemMessage(sysMsgExcellentShield)
			}
		}
	}
	if bow {
		if p != nil {
			s.attack.arrows.UseArrow(p)
			p.SendSystemMessage(sysMsgNockArrow)
			p.Send(serverpackets.NewSetupGaugePacket(serverpackets.SetupGauge{
				Color:   serverpackets.GaugeRed,
				Time:    int32(timing.Gauge / time.Millisecond), //nolint:gosec // a few seconds
				MaxTime: int32(timing.Gauge / time.Millisecond), //nolint:gosec // a few seconds
			}))
		}
		st.bowLock = gameTick(now) + int64(timing.BowLockTicks)
	}
	for i, h := range hits {
		s.attackTimer(st, timing.Hits[i], func() { s.onHitTimer(a, t, h, i == 0) })
	}

	// Mobius flags the attacker outside PvP zones and starts its stance.
	if p != nil && !p.Zones.Inside(ZonePvP) && Creature(p) != t {
		s.addStance(p)
		for _, fn := range s.attack.swing {
			s.safely("OnPlayerSwing hook", func() { fn(p, t) })
		}
	}
	if !res.Landed() {
		s.AbortAttack(a)
	} else {
		s.spendSoulshot(a) // shot.go: a landed swing spends the soulshot
	}
	s.world.Broadcast(a, s.attackPacket(a, t, hits, soulshot, grade), true)
	s.readyAfter(a, st, timing.Ready)
	return AttackSwung
}

// refuseAttack ends a refused swing as Mobius does: the intention becomes
// ACTIVE and a player gets ActionFailed.
func (s *Server) refuseAttack(a Creature) AttackOutcome {
	switch v := a.(type) {
	case *Player:
		s.playerIntendIdle(v)
		v.SendActionFailed()
	case *Npc:
		s.npcAttackEvent(v, NpcAttackRefused)
	}
	return AttackRefused
}

// canUseRangeWeapon is Mobius canUseRangeWeapon: a player needs arrows, MP
// and an unlocked bow, and pays the MP; an NPC needs an unlocked bow. It
// reports false with what DoAttack returns when the bow cannot shoot.
func (s *Server) canUseRangeWeapon(a Creature, w *data.Item, st *attackState) (AttackOutcome, bool) {
	tick := gameTick(s.Now())
	switch v := a.(type) {
	case *Player:
		if w == nil || w.Weapon.Type != data.WeaponBow {
			return AttackNone, false
		}
		if !s.attack.arrows.EquipArrows(v) {
			s.playerIntendIdle(v)
			v.SendActionFailed()
			v.SendSystemMessage(sysMsgOutOfArrows)
			return AttackNone, false
		}
		if st.bowLock > tick {
			s.readyAfter(v, st, attackBowRetry)
			v.SendActionFailed()
			return AttackBowWait, false
		}
		mp := w.Weapon.MPConsume
		if w.Weapon.ReducedMPConsume > 0 && s.attack.rnd.Get(100) < w.Weapon.ReducedMPConsumeChance {
			mp = w.Weapon.ReducedMPConsume
		}
		mp = int(javaInt32(v.stats.Calc(stats.BowMPConsumeRate, float64(mp))))
		if v.CurMP < float64(mp) {
			s.readyAfter(v, st, attackBowRetry)
			v.SendSystemMessage(sysMsgNotEnoughMP)
			v.SendActionFailed()
			return AttackBowWait, false
		}
		if mp > 0 {
			v.ReduceMP(float64(mp))
		}
		st.bowLock = tick + attackBowShotTicks
	case *Npc:
		if st.bowLock > tick {
			return AttackNone, false
		}
	}
	return AttackSwung, true
}

// attackPacket is the Attack of a's swing at t with its hits, and the
// soulshot with the weapon's grade when one was charged.
func (s *Server) attackPacket(a, t Creature, hits []stats.Hit, soulshot bool, grade int32) []byte {
	ap, tp := a.Position(), t.Position()
	pkt := serverpackets.Attack{
		AttackerID: int32(a.ID()),
		X:          ap.X, Y: ap.Y, Z: ap.Z,
		TargetX: tp.X, TargetY: tp.Y, TargetZ: tp.Z,
		Soulshot: soulshot, Grade: grade,
		Hits: make([]serverpackets.AttackHit, len(hits)),
	}
	for i, h := range hits {
		pkt.Hits[i] = serverpackets.AttackHit{
			TargetID: int32(t.ID()),
			Damage:   int32(h.Damage), //nolint:gosec // hit damage fits
			Miss:     h.Miss,
			Critical: h.Crit,
			Shield:   h.Shield != stats.ShieldFailed,
		}
	}
	return serverpackets.NewAttackPacket(pkt)
}

// onHitTimer is Mobius Creature.onHitTimer: a hit of a's swing lands on t.
// Reflected and absorbed damage (which skip bows), damage listeners,
// trigger skills come with the skills (M7): until then the
// REFLECT_DAMAGE_PERCENT and ABSORB_DAMAGE_PERCENT stats are 0. The first
// hit of a swing (recharge) charges the automatic soulshots again at its
// end, and when the target died before it (shot.go).
func (s *Server) onHitTimer(a, t Creature, h stats.Hit, recharge bool) {
	if !attackerInWorld(a) {
		return
	}
	if t == nil || a.IsDead() {
		s.notifyCancel(a)
		return
	}
	if recharge {
		defer s.RechargeShots(a, true, false)
	}
	_, npc := a.(*Npc)
	if (npc && t.IsDead()) || t.IsDead() || !s.surrounding(a, t) {
		s.notifyCancel(a)
		if p, ok := a.(*Player); ok {
			p.SendActionFailed()
		}
		return
	}
	if p, ok := a.(*Player); ok {
		sendDamageMessage(p, t, h.Damage, h.Crit, h.Miss)
	}
	if s.raidCurses(t) && creatureLevel(a) > creatureLevel(t)+raidCurseLevelGap {
		// raid.go: the raid boss petrifies an attacker more than 8 levels
		// above it, and the hit does no damage.
		s.applyRaidCurse(a, t, skillRaidCurse2)
		h.Damage = 0
	}
	if tp, ok := t.(*Player); ok {
		s.StartAutoAttack(tp)
	}
	if h.Miss || h.Damage <= 0 {
		return
	}
	for _, fn := range s.attack.hit {
		s.safely("OnHit hook", func() { fn(t, a, h.Damage) })
	}
	t.ReduceHP(float64(h.Damage), a)
	for _, fn := range s.attack.attacked {
		s.safely("OnAttacked hook", func() { fn(t, a, h.Damage) })
	}
	s.StartAutoAttack(a)
	if !creatureRaid(t) {
		if tc, st := s.breakTarget(t); tc != nil && s.combat.AttackBreak(s.attack.rnd, tc, st, float64(h.Damage)) {
			s.BreakAttack(t)
			s.BreakCast(t)
		}
	}
	s.augmentHitTriggers(a, t, h.Crit) // augment.go: the option trigger skills
}

// notifyCancel is Mobius notifyActionCancel for an attacker whose hit could
// not land: a player's saved move is dropped, the follow stops, an
// AutoAttackStop goes out without a stance and the intention thinks again;
// an NPC's AI hears NpcAttackCancelled after the same AutoAttackStop.
func (s *Server) notifyCancel(a Creature) {
	if !attackerInWorld(a) {
		return
	}
	if p, ok := a.(*Player); ok {
		ai := s.playerAIOf(p)
		ai.savedMove = nil
		s.playerStopFollow(p)
	}
	if !s.InAttackStance(a) {
		s.world.Broadcast(a, serverpackets.NewAutoAttackStopPacket(int32(a.ID())), true)
	}
	switch v := a.(type) {
	case *Player:
		s.playerThink(v)
	case *Npc:
		s.npcAttackEvent(v, NpcAttackCancelled)
	}
}

// sendDamageMessage is Mobius Player.sendDamageMessage: "You have missed.",
// or "Critical hit!" and "You hit for $s1 damage.".
func sendDamageMessage(p *Player, target Creature, damage int, crit, miss bool) {
	if miss {
		p.SendSystemMessage(sysMsgYouMissed)
		return
	}
	if crit {
		p.SendSystemMessage(sysMsgCriticalHit)
	}
	if Creature(p) != target {
		p.SendSystemMessage(sysMsgYouHitForDamage, serverpackets.NumberParam(int32(damage))) //nolint:gosec // hit damage fits
	}
}

// breakTarget returns what Formulas.calcAtkBreak reads of t.
func (s *Server) breakTarget(t Creature) (stats.Combatant, stats.BreakState) {
	var st stats.BreakState
	st.Casting = s.CastingNow(t)
	switch v := t.(type) {
	case *Player:
		w := s.activeWeaponItem(v)
		st.BowAttacking = s.AttackingNow(v) && w != nil && w.Weapon.Type == data.WeaponBow
		return v.stats, st
	case *Npc:
		w := s.activeWeaponItem(v)
		st.BowAttacking = s.AttackingNow(v) && w != nil && w.Weapon.Type == data.WeaponBow
		st.Raid = creatureRaid(v)
		return v.Stats, st
	}
	return nil, st
}

// fighter builds the stats side of c for a swing: its stats, active weapon
// w (nil for none), shield and position.
func (s *Server) fighter(c Creature, w *data.Item) stats.Fighter {
	pos := c.Position()
	f := stats.Fighter{Ref: s.statsRef(c), X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z), Heading: int(pos.Heading)}
	var sw *stats.Weapon
	if w != nil {
		sw = &stats.Weapon{ItemID: w.ID, Type: w.Weapon.Type.String(), ReuseDelay: w.ReuseDelay}
	}
	switch v := c.(type) {
	case *Player:
		f.Stats, f.Playable = v.stats, true
		active := stats.PlayerWeapon(v.ClassID(), sw)
		f.Arm(&active, data.WeaponFist.String(), true)
		f.Shield = playerHasShield(v)
	case *Npc:
		f.Stats = v.Stats
		f.Attackable = v.class != nil && v.class.attackable
		f.Arm(sw, v.Template.Stats.Attack.Type.String(), false)
	case *Door:
		f.Stats = v.combatStats() // door_combat.go
	}
	return f
}

// activeWeaponItem returns the item template of c's active weapon, nil for
// none: a player's right-hand weapon (without one Mobius gives it its class
// fists, which stats.PlayerWeapon adds); an NPC's template right-hand item
// when it is a weapon (Mobius Npc.getActiveWeaponItem).
func (s *Server) activeWeaponItem(c Creature) *data.Item {
	switch v := c.(type) {
	case *Player:
		if v.inv.Inventory == nil {
			return nil
		}
		if it := v.inv.Paperdoll(storage.PaperdollRHand); it != nil && it.Template().Weapon != nil {
			return it.Template()
		}
	case *Npc:
		if id := v.Template.Equipment.RHand; id > 0 {
			if t, ok := s.data.ItemTemplate(id); ok && t.Weapon != nil {
				return t
			}
		}
	}
	return nil
}

// playerHasShield reports whether p's left hand holds armor: a shield
// (Formulas.calcShldUse; Interlude has no sigils).
func playerHasShield(p *Player) bool {
	if p.inv.Inventory == nil {
		return false
	}
	it := p.inv.Paperdoll(storage.PaperdollLHand)
	return it != nil && it.Template().Armor != nil
}

// inventoryArrows is the default ArrowSupply: the player's inventory.
type inventoryArrows struct{ s *Server }

// EquipArrows is Mobius checkAndEquipArrows: arrows of the bow's grade go
// into an empty left hand; whatever the left hand then holds counts.
func (a inventoryArrows) EquipArrows(p *Player) bool {
	if p.inv.Inventory == nil {
		return false
	}
	a.s.checkAndEquipArrows(p)
	return p.inv.Paperdoll(storage.PaperdollLHand) != nil
}

// UseArrow is Mobius reduceArrowCount: one arrow of the left hand goes; the
// last one leaves the hand empty (the client gets its ItemList).
func (a inventoryArrows) UseArrow(p *Player) {
	if p.inv.Inventory == nil {
		return
	}
	if it := p.inv.Paperdoll(storage.PaperdollLHand); it != nil {
		a.s.DestroyItem(p, it, 1, false)
	}
}

// stopCreature stops a moving creature where it stands with StopMove
// (Mobius doAttack: stopMove(getLocation())).
func (s *Server) stopCreature(c Creature) {
	switch v := c.(type) {
	case *Player:
		if v.Move != nil {
			s.StopPlayer(v)
		}
	case *Npc:
		if v.Move != nil {
			s.stopNpc(v, nil)
		}
	}
}

// surrounding reports whether b is in a's region or one next to it (Mobius
// isInSurroundingRegion).
func (s *Server) surrounding(a, b Creature) bool {
	ka, okA := creatureRegion(a)
	kb, okB := creatureRegion(b)
	return okA && okB && ka.Adjacent(kb)
}

func creatureRegion(c Creature) (world.RegionKey, bool) {
	if r, ok := c.(interface {
		Region() (world.RegionKey, bool)
	}); ok {
		return r.Region()
	}
	return world.RegionKey{}, false
}

// attackerInWorld reports whether c is in the world.
func attackerInWorld(c Creature) bool {
	b, ok := c.(interface{ InWorld() bool })
	return ok && b.InWorld()
}

// creatureTargetable is Mobius isTargetable.
func creatureTargetable(c Creature) bool {
	switch v := c.(type) {
	case *Npc:
		return v.Template.Status.Targetable
	case *Door:
		return v.Targetable()
	}
	return true
}

// creatureRaid is Mobius isRaid: raid and grand bosses, and the minions of
// a raid master (setIsRaidMinion sets _isRaid too).
func creatureRaid(c Creature) bool {
	n, ok := c.(*Npc)
	return ok && ((n.class != nil && n.class.raid) || n.boss.raidMinion)
}

// creatureMoving is Mobius isMoving.
func creatureMoving(c Creature) bool {
	switch v := c.(type) {
	case *Player:
		return v.Move != nil
	case *Npc:
		return v.Move != nil
	}
	return false
}

// attackCollision is the template collision radius Mobius's range checks
// use (getTemplate().getCollisionRadius(), an int; a player's male one).
func attackCollision(c Creature) int {
	switch v := c.(type) {
	case *Player:
		return int(walkCollision(v))
	case *Npc:
		return int(v.Template.Collision.Radius)
	case *Door:
		return v.collisionRadius()
	}
	return 0
}

func setCreatureHeading(c Creature, heading int32) {
	if b, ok := c.(interface{ SetHeading(h int32) }); ok {
		b.SetHeading(heading)
	}
}

func geoPoint(p world.Position) geo.Point { return geo.Point{X: p.X, Y: p.Y, Z: p.Z} }
