package game

import (
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/world"
)

// Casting (Plan 10 Phase 7.3), ported from Mobius Creature (doCast,
// beginCast, checkDoCastConditions, onMagicLaunchedTimer, onMagicHitTimer,
// onMagicFinalizer, abortCast, breakCast, makeTriggerCast), Player
// (useMagic, checkUseMagicConditions, checkPvpSkill) and the CAST intention
// of PlayerAI:
//
//   - cast.go: the cast of any creature: its state, the timers of one cast
//     (K7), the numbers (K10), interruption and the API below.
//   - cast_call.go: callSkill, where a cast lands (K4, K11): the effects, raid
//     curses, PvP flags, hate and the monsters that see the spell.
//   - cast_player.go: RequestMagicSkillUse, useMagic and its checks, the cast
//     intention (walking into range), Esc (RequestTargetCanceld) and the PvP
//     rules of skills.
//
// The API for item skills, scripts, the newbie buffer and the monster AI
// (all loop-only):
//
//   - UseMagic is a player's Mobius useMagic: the checks, the queue and the
//     cast intention, as a skill button does.
//   - DoCast and DoSimultaneousCast are Mobius doCast(skill) and
//     doSimultaneousCast(skill): any creature casts at its current target.
//     NpcCast sets an NPC's target first (Mobius npc.setTarget(player);
//     npc.doCast(skill), as SupportMagic does). DoCastAt is doCast(skill,
//     target, targets), for an AI that picked its targets itself.
//   - CastingNow, CanAbortCast, AbortCast and BreakCast are Mobius
//     isCastingNow, canAbortCast, abortCast and breakCast.
//   - SkillDisabled and DisableSkill are isSkillDisabled and disableSkill.
//   - SetShotRecharger plugs the shots in (Player.rechargeShots, Phase 7.6);
//     OnSkillFinished hooks run where Mobius notifies the scripts that a
//     skill finished.
func init() { registerFeature("cast", setupCast) }

const (
	// castLaunchLead is how long before the end of its cast time a skill is
	// launched (Mobius MagicUseTask: "400 ms before" for the client
	// animation); the hit comes castLaunchLead after the launch.
	castLaunchLead = 400 * time.Millisecond
	// castSimultaneousRetry is when a simultaneous cast that found the slot
	// busy tries again (Mobius queues herbs and potions 100 ms later).
	castSimultaneousRetry = 100 * time.Millisecond
	// castNextAttackDelay is how long after a nextActionAttack skill the
	// player attacks (Mobius waits for the skill's landing animation).
	castNextAttackDelay = 333 * time.Millisecond
	// castSightTime is the cast time from which the launch checks the line
	// of sight (Mobius mut.getSkillTime() > 550).
	castSightTime = 550
	// castEscapeAffectRange is the affect range over which a skill without a
	// cast range checks its targets' range at the launch.
	castEscapeAffectRange = 80
	// castInterruptTicks is how many game ticks before its end a cast stops
	// being interruptible (Mobius _castInterruptTime = -2 + ...).
	castInterruptTicks = 2
	// castSkillSeeRange is the range of the monsters that see a player's
	// spell (Mobius callSkill: forEachVisibleObjectInRange 1000).
	castSkillSeeRange = 1000
)

// Skills Mobius treats by ID while casting.
const (
	skillFakeDeath  = 60   // CommonSkill.FAKE_DEATH
	skillLure       = 51   // never puts its caster on the attack-by list
	skillTemptation = 511  // nor does this one
	skillFishing    = 1312 // its message comes from fishing
	skillWolfCollar = 2046 // "Summoning your pet..."
	skillRaidCurse  = 4215 // CommonSkill.RAID_CURSE (silence)
	skillRaidCurse2 = 4515 // CommonSkill.RAID_CURSE2 (petrification)
)

// System messages of casting (Mobius SystemMessageId).
const (
	sysMsgTargetOutOfRange    serverpackets.SystemMessageID = 22   // "Your target is out of range."
	sysMsgNotEnoughHP         serverpackets.SystemMessageID = 23   // "Not enough HP."
	sysMsgCastingInterrupted  serverpackets.SystemMessageID = 27   // "Your casting has been interrupted."
	sysMsgCannotMoveSitting   serverpackets.SystemMessageID = 31   // "You cannot move while sitting."
	sysMsgNotReadyForReuse    serverpackets.SystemMessageID = 48   // "$s1 is not available at this time: being prepared for reuse."
	sysMsgIncorrectTarget     serverpackets.SystemMessageID = 144  // "That is the incorrect target."
	sysMsgRejectResurrection  serverpackets.SystemMessageID = 356  // "Reject resurrection."
	sysMsgSummoningYourPet    serverpackets.SystemMessageID = 547  // "Summoning your pet..."
	sysMsgCannotDisableTarget serverpackets.SystemMessageID = 563  // "Failed to disable attack target."
	sysMsgServitorCosts       serverpackets.SystemMessageID = 1197 // "Summoning a servitor costs $s2 $s1."
	sysMsgFormalWearNoCasting serverpackets.SystemMessageID = 1604 // "While dressed in formal wear, you can't use items that require all skills and casting operations."
)

// sysMsgCastDistanceTooFar is "The distance is too far and so the casting
// has been stopped.".
const sysMsgCastDistanceTooFar = skills.MsgDistanceTooFarCastStopped

// Texts Mobius sends with sendMessage while casting.
const (
	castTextPeaceZone   = "A malicious skill cannot be used in a peace zone."
	castTextWeaponOnly  = "That weapon cannot use any other skill except the weapon's skill."
	castTextNotEnoughIt = "There are not enough necessary items to use the skill."
)

// castState is a creature's cast state (Mobius Creature _isCastingNow,
// _skillCast, _castInterruptTime, _disabledSkills, ... and, for players,
// _currentSkill, _queuedSkill and the CAST intention of PlayerAI). It lives
// in the creature's skill adapter (skillBase.cast). Loop-only.
type castState struct {
	// casting and simCasting are the normal and the simultaneous slot
	// (isCastingNow, isCastingSimultaneouslyNow); timer and simTimer their
	// pending phase.
	casting, simCasting bool
	timer, simTimer     world.TimerID
	// interrupt is the game tick until which the cast may be interrupted
	// (_castInterruptTime); last the skill of the normal slot
	// (_lastSkillCast).
	interrupt int64
	last      *skills.Skill
	// disabled holds the short disables by reuse hash (_disabledSkills): the
	// time they end.
	disabled map[int]time.Time
	// later are the other timers of the casts: the retries of simultaneous
	// casts and the attack after a cast; they end when the creature leaves.
	later []world.TimerID

	// current and queued are a player's skill in use and the skill pressed
	// meanwhile (_currentSkill, _queuedSkill).
	current, queued *skillUse
	// skill and target are the CAST intention (AbstractAI _skill and
	// getCastTarget); saved the intention the cast interrupted
	// (PlayerAI._savedIntention and its attack target).
	skill  *skills.Skill
	target Creature
	saved  castSaved
}

// skillUse is a skill button press (Mobius SkillUseHolder): Ctrl forces it,
// Shift casts without walking.
type skillUse struct {
	skill       *skills.Skill
	ctrl, shift bool
}

// castSaved is the intention a cast interrupted (PlayerAI
// saveCurrentIntentionForCast).
type castSaved struct {
	set       bool
	intention playerIntention
	target    Creature
}

// magicUse is one cast on its way (Mobius MagicUseTask): the skill, its
// targets (filtered at the launch), its cast time in milliseconds, its slot
// and how often it hit.
type magicUse struct {
	skill   *skills.Skill
	targets []skills.Creature
	time    int
	sim     bool
	count   int
}

// ShotRecharger charges a player's automatic shots for a skill (Mobius
// Player.rechargeShots(physical, magic)): soulshots for physical attack
// skills, spiritshots for magic ones.
type ShotRecharger func(p *Player, physical, magic bool)

// SkillFinishedHook runs when a creature's cast finished (Mobius
// notifyQuestEventSkillFinished): the skill and its first target, nil for
// none.
type SkillFinishedHook func(c Creature, sk *skills.Skill, target Creature)

// castFeature is the server side of casting: the SkillCaster the creature
// adapters ask, and the hooks of other features.
type castFeature struct {
	s        *Server
	shots    ShotRecharger
	finished []SkillFinishedHook
}

var _ SkillCaster = (*castFeature)(nil)

// castFeatureOf returns the server's cast feature, made and plugged into
// the adapters on first use, so that features set up before or after this
// one share it.
func (s *Server) castFeatureOf() *castFeature {
	if f, ok := s.skills.caster.(*castFeature); ok {
		return f
	}
	f := &castFeature{s: s}
	s.SetSkillCaster(f)
	return f
}

// setupCast registers RequestMagicSkillUse and the clean-up of a leaving
// player.
func setupCast(s *Server) error {
	s.castFeatureOf()
	s.OnPlayerLeave(s.castLeave)
	return s.Register(clientpackets.OpRequestMagicSkillUse, s.handleRequestMagicSkillUse)
}

// SetShotRecharger sets how a player's automatic shots are charged at a
// skill's hit and after a timed cast (Phase 7.6). Call it from a feature's
// setup.
func (s *Server) SetShotRecharger(fn ShotRecharger) { s.castFeatureOf().shots = fn }

// OnSkillFinished adds fn to the functions run when a cast finished. Call it
// from a feature's setup.
func (s *Server) OnSkillFinished(fn SkillFinishedHook) {
	f := s.castFeatureOf()
	f.finished = append(f.finished, fn)
}

// castOf returns c's cast state, nil for a creature without a skill
// adapter.
func (s *Server) castOf(c Creature) *castState {
	if b := s.skillBaseOf(c); b != nil {
		return &b.cast
	}
	return nil
}

// CastingNow reports a cast in the normal slot (Mobius isCastingNow).
// Loop-only.
func (s *Server) CastingNow(c Creature) bool {
	st := s.castOf(c)
	return st != nil && st.casting
}

// CastingSimultaneouslyNow reports a cast in the simultaneous slot (Mobius
// isCastingSimultaneouslyNow). Loop-only.
func (s *Server) CastingSimultaneouslyNow(c Creature) bool {
	st := s.castOf(c)
	return st != nil && st.simCasting
}

// CanAbortCast reports a cast still before its interrupt time, 200 ms before
// its end (Mobius canAbortCast). Loop-only.
func (s *Server) CanAbortCast(c Creature) bool {
	st := s.castOf(c)
	return st != nil && st.interrupt > s.gameTicks()
}

// DoCast makes c cast sk at its current target (Mobius doCast(skill)): the
// cast conditions, the target list, then the cast with its timers. It is
// the NPC entry point; players use UseMagic, which walks into range first.
// Loop-only.
func (s *Server) DoCast(c Creature, sk *skills.Skill) { s.castBegin(c, sk, false) }

// DoSimultaneousCast is DoCast in the simultaneous slot (Mobius
// doSimultaneousCast): potions and herbs, which do not stop another cast.
// Loop-only.
func (s *Server) DoSimultaneousCast(c Creature, sk *skills.Skill) { s.castBegin(c, sk, true) }

// NpcCast makes n target target and cast sk at it (Mobius
// npc.setTarget(target); npc.doCast(skill)), as the newbie buffer and NPC
// scripts do. Loop-only.
func (s *Server) NpcCast(n *Npc, target Creature, sk *skills.Skill) {
	if n == nil || n.skill == nil {
		return
	}
	n.skill.SetTarget(s.skillCreatureOf(target))
	s.castBegin(n, sk, false)
}

// DoCastAt is Mobius doCast(skill, target, targets) for an AI that chose its
// target and targets: the cast conditions, then the cast (simultaneous for a
// simultaneous skill). Loop-only.
func (s *Server) DoCastAt(c Creature, sk *skills.Skill, target Creature, targets []Creature) {
	b := s.skillBaseOf(c)
	if b == nil || sk == nil {
		return
	}
	if s.monsterDropsCast(c, sk) { // Monster.doCast(skill, target, targets), ai_cast.go
		b.cast.clear(false)
		b.cast.clear(true)
		return
	}
	sim := sk.SimultaneousCast
	if !s.checkDoCastConditions(c, b, sk) {
		b.cast.clear(sim)
		return
	}
	b.el.StopOnAction()
	list := make([]skills.Creature, 0, len(targets))
	for _, t := range targets {
		if sc := s.skillCreatureOf(t); sc != nil {
			list = append(list, sc)
		}
	}
	s.castStart(c, b, sk, sim, s.skillCreatureOf(target), list)
}

// AbortCast stops c's cast (Mobius abortCast): the timers end, a player's
// intention goes back to what the cast interrupted, everyone around sees
// MagicSkillCanceled and a player gets ActionFailed. Nothing happens when c
// does not cast. Loop-only.
func (s *Server) AbortCast(c Creature) { s.abortCast(c) }

// BreakCast interrupts c's cast as damage does (Mobius breakCast): only a
// magic or static skill before its interrupt time; a player is told "Your
// casting has been interrupted.". Loop-only.
func (s *Server) BreakCast(c Creature) {
	st := s.castOf(c)
	if st == nil || !st.casting || st.interrupt <= s.gameTicks() || st.last == nil || (!st.last.IsMagic() && !st.last.IsStatic()) {
		return
	}
	s.abortCast(c)
	if p, ok := c.(*Player); ok {
		p.SendSystemMessage(sysMsgCastingInterrupted)
	}
}

// SkillDisabled reports whether c cannot use sk now (Mobius isSkillDisabled):
// all its skills are disabled (stunned, asleep, paralyzed), sk is under
// reuse or disabled for a moment. Loop-only.
func (s *Server) SkillDisabled(c Creature, sk *skills.Skill) bool {
	b := s.skillBaseOf(c)
	return b != nil && s.skillDisabled(b, sk)
}

// DisableSkill disables sk for c for d, for good when d is not positive
// (Mobius disableSkill). Loop-only.
func (s *Server) DisableSkill(c Creature, sk *skills.Skill, d time.Duration) {
	if b := s.skillBaseOf(c); b != nil && sk != nil {
		s.disableSkill(b, sk, d)
	}
}

// EnableSkill ends a DisableSkill (Mobius enableSkill). Loop-only.
func (s *Server) EnableSkill(c Creature, sk *skills.Skill) {
	if st := s.castOf(c); st != nil && sk != nil {
		delete(st.disabled, sk.ReuseHash())
	}
}

func (s *Server) disableSkill(b *skillBase, sk *skills.Skill, d time.Duration) {
	st := &b.cast
	if st.disabled == nil {
		st.disabled = make(map[int]time.Time)
	}
	until := time.Unix(1<<62, 0)
	if d > 0 {
		until = s.Now().Add(d)
	}
	st.disabled[sk.ReuseHash()] = until
}

func (s *Server) skillDisabled(b *skillBase, sk *skills.Skill) bool {
	if sk == nil {
		return false
	}
	if allSkillsDisabled(b) {
		return true
	}
	now := s.Now()
	if b.reuseLeft(sk, now) > 0 {
		return true
	}
	st := &b.cast
	until, ok := st.disabled[sk.ReuseHash()]
	if !ok {
		return false
	}
	if until.Before(now) {
		delete(st.disabled, sk.ReuseHash())
		return false
	}
	return true
}

// allSkillsDisabled is Mobius isAllSkillsDisabled: stunned, asleep or
// paralyzed (also by //para).
func allSkillsDisabled(b *skillBase) bool {
	return b.admin.paralyzed || b.el.IsAffected(skills.FlagStunned) || b.el.IsAffected(skills.FlagSleep) || b.el.IsAffected(skills.FlagParalyzed)
}

// clear ends the slot's cast flag (setCastingNow(false) or
// setCastingSimultaneouslyNow(false)).
func (st *castState) clear(sim bool) {
	if sim {
		st.simCasting = false
	} else {
		st.casting = false
	}
}

// castLater runs fn after d as one of c's cast timers.
func (s *Server) castLater(st *castState, d time.Duration, fn func()) {
	var id world.TimerID
	id = s.loop.After(d, func() {
		for i, x := range st.later {
			if x == id {
				st.later = append(st.later[:i], st.later[i+1:]...)
				break
			}
		}
		fn()
	})
	st.later = append(st.later, id)
}

// castSchedule runs the next phase of a cast after d in its slot, replacing
// what the slot held (Mobius _skillCast and _skillCast2).
func (s *Server) castSchedule(st *castState, sim bool, d time.Duration, fn func()) {
	slot := &st.timer
	if sim {
		slot = &st.simTimer
	}
	if *slot != 0 {
		s.loop.Cancel(*slot)
	}
	var id world.TimerID
	id = s.loop.After(d, func() {
		if *slot == id {
			*slot = 0
		}
		fn()
	})
	*slot = id
}

// castBegin is Mobius beginCast(skill, simultaneous): an attackable on the
// move does not cast; the cast conditions; a simultaneous skill takes the
// simultaneous slot; effects that end on an action end; then the targets
// of the skill type and the target the packets name.
func (s *Server) castBegin(c Creature, sk *skills.Skill, sim bool) {
	b := s.skillBaseOf(c)
	if b == nil || sk == nil || !attackerInWorld(c) {
		return
	}
	st := &b.cast
	p, _ := c.(*Player)
	if n, ok := c.(*Npc); ok && n.class != nil && n.class.attackable && n.Move != nil {
		return
	}
	if !s.checkDoCastConditions(c, b, sk) {
		st.clear(sim)
		if p != nil {
			s.playerIntendIdle(p)
		}
		return
	}
	if sk.SimultaneousCast {
		sim = true
	}
	b.el.StopOnAction()
	targets := sk.Targets(s.skills.env, b.self, false, b.Target())
	var target skills.Creature
	switch sk.TargetType {
	case data.TargetAreaSummon:
		// The caster's summon: summons come with Milestone 11.
	case data.TargetAura, data.TargetAuraCorpseMob, data.TargetFrontAura, data.TargetBehindAura,
		data.TargetGround, data.TargetAuraFriendly:
		target = b.self
	default:
		if len(targets) == 0 {
			st.clear(sim)
			if p != nil {
				p.SendActionFailed()
				s.playerIntendIdle(p)
			}
			return
		}
		doit := castOnFirstTarget(sk.TargetType) ||
			(sk.IsContinuous() && !sk.IsDebuff) || sk.HasEffectType(skills.TypeCPHeal, skills.TypeHeal)
		if doit {
			target = targets[0]
		} else {
			target = b.Target()
		}
	}
	s.castStart(c, b, sk, sim, target, targets)
}

// castOnFirstTarget reports the target types whose packets name the first
// target of the list (beginCast's "doit" types).
func castOnFirstTarget(t data.TargetType) bool {
	switch t {
	case data.TargetSelf, data.TargetPet, data.TargetServitor, data.TargetSummon, data.TargetOwnerPet,
		data.TargetParty, data.TargetClan, data.TargetPartyClan, data.TargetCommandChannel:
		return true
	}
	return false
}

// castStart is Mobius beginCast(skill, simultaneously, target, targets),
// the start of the cast at t0 (K10): the cast time; a busy simultaneous
// slot retries in 100 ms; the interrupt time; the MP of the start with its
// StatusUpdate; the reuse stamp or the short disable after the mastery
// roll; the caster faces the target; the consumed item; MagicSkillUse and
// MagicSkillLaunched to everyone around (toggles send neither, GROUND skills
// no MagicSkillLaunched); "You use S1."; the START effects; SetupGauge; and
// the launch max(0, T − 400) ms later, at once for a cast time of 0.
func (s *Server) castStart(c Creature, b *skillBase, sk *skills.Skill, sim bool, target skills.Creature, targets []skills.Creature) {
	st := &b.cast
	p, _ := c.(*Player)
	if target == nil {
		st.clear(sim)
		if p != nil {
			p.SendActionFailed()
			s.playerIntendIdle(p)
		}
		return
	}
	if sk.HasEffectType(skills.TypeResurrection) &&
		(b.el.IsAffected(skills.FlagBlockResurrection) || target.Effects().IsAffected(skills.FlagBlockResurrection)) {
		skills.Send(b.self, sysMsgRejectResurrection)
		skills.Send(target, sysMsgRejectResurrection)
		st.clear(sim)
		if p != nil {
			s.playerIntendIdle(p)
			p.SendActionFailed()
		}
		return
	}
	env := s.skills.env
	skillTime := skills.CastTime(b.self, sk)
	if st.simCasting && sim {
		s.castLater(st, castSimultaneousRetry, func() {
			if attackerInWorld(c) && !c.IsDead() {
				s.castStart(c, b, sk, sim, target, targets)
			}
		})
		return
	}
	if sim {
		st.simCasting = true
	} else {
		st.casting = true
		st.interrupt = -castInterruptTicks + s.gameTicks() + int64(skillTime)/walkTickMillis
		st.last = sk
	}
	reuse := skills.ReuseDelay(b.self, sk)
	if mp := skills.MPInitialConsume(b.self, sk); mp > 0 {
		s.castReduceMP(c, float64(mp))
		if p != nil {
			p.Send(castStatusPacket(p, serverpackets.StatusValue{Attr: serverpackets.StatusCurMP, Value: int32(p.CurMP)}))
		}
	}
	if reuse > 10 {
		if skills.SkillMastery(env, b.self, sk) {
			reuse = 100
			if p != nil {
				p.SendSystemMessage(skills.MsgSkillReadyToBeUsedAgain)
			}
		}
		if reuse > 1000 {
			s.AddSkillReuse(c, sk, time.Duration(reuse)*time.Millisecond)
		} else {
			s.disableSkill(b, sk, time.Duration(reuse)*time.Millisecond)
		}
	}
	if gt := s.gameCreature(target); gt != nil && gt != c {
		a, t := c.Position(), gt.Position()
		setCreatureHeading(c, int32(castHeading(a, t))) //nolint:gosec // 0-65535
	}
	if p != nil && sk.ItemConsumeID > 0 &&
		!s.DestroyItemByItemID(p, int32(sk.ItemConsumeID), int64(sk.ItemConsumeCount), true) { //nolint:gosec // item IDs are small
		p.SendSystemMessage(sysMsgIncorrectCount)
		s.abortCast(c)
		return
	}
	if !sk.IsToggle() {
		if gt := s.gameCreature(target); gt != nil {
			s.world.Broadcast(c, castUsePacket(c, gt, sk.DisplayID, sk.DisplayLevel, skillTime, reuse), true)
		}
		if sk.TargetType != data.TargetGround && (!sk.IsChanneling() || sk.ChannelingSkillID == 0) {
			s.world.Broadcast(c, castLaunchedPacket(c, sk.DisplayID, sk.DisplayLevel, targets), true)
		}
	}
	if p != nil && !sk.AbnormalInstant {
		switch sk.ID {
		case skillFishing:
		case skillWolfCollar:
			p.SendSystemMessage(sysMsgSummoningYourPet)
		default:
			p.SendSystemMessage(skills.MsgUseS1, skills.SkillName(sk))
		}
	}
	if sk.HasEffects(data.ScopeStart) {
		sk.ApplyEffectScope(data.ScopeStart, skills.NewBuffInfo(env, b.self, target, sk), true, false)
	}
	mu := &magicUse{skill: sk, targets: targets, time: skillTime, sim: sim}
	if skillTime <= 0 {
		mu.time = 0
		s.castLaunched(c, b, mu)
		return
	}
	if p != nil && !sim {
		p.Send(serverpackets.NewSetupGaugePacket(serverpackets.SetupGauge{
			Color: serverpackets.GaugeBlue, Time: int32(skillTime), MaxTime: int32(skillTime), //nolint:gosec // a cast time
		}))
	}
	lead := max(0, time.Duration(skillTime)*time.Millisecond-castLaunchLead)
	s.castSchedule(st, sim, lead, func() { s.castLaunched(c, b, mu) })
}

// castHeading returns the client heading from a to b.
func castHeading(a, b world.Position) int {
	return stats.HeadingTo(int(a.X), int(a.Y), int(b.X), int(b.Y))
}

// castReduceMP takes MP from a caster (Mobius reduceMp).
func (s *Server) castReduceMP(c Creature, mp float64) {
	switch v := c.(type) {
	case *Player:
		v.ReduceMP(mp)
	case *Npc:
		v.ReduceMP(mp)
	}
}

// castStatusPacket is the StatusUpdate of the values a cast changed, which
// Mobius sends the caster at once.
func castStatusPacket(p *Player, values ...serverpackets.StatusValue) []byte {
	return serverpackets.NewStatusUpdatePacket(serverpackets.StatusUpdate{ObjectID: p.ObjectID(), Values: values})
}

// castUsePacket is MagicSkillUse of c casting at t.
func castUsePacket(c, t Creature, id, level, hitTime, reuse int) []byte {
	cp, tp := c.Position(), t.Position()
	return serverpackets.NewMagicSkillUsePacket(serverpackets.MagicSkillUse{
		CasterID:   int32(c.ID()), //nolint:gosec // object IDs fit
		TargetID:   int32(t.ID()), //nolint:gosec // object IDs fit
		SkillID:    int32(id),     //nolint:gosec // skill IDs are small
		SkillLevel: int32(level),  //nolint:gosec // skill levels are small
		HitTime:    time.Duration(hitTime) * time.Millisecond,
		ReuseDelay: time.Duration(reuse) * time.Millisecond,
		X:          cp.X, Y: cp.Y, Z: cp.Z,
		TargetX: tp.X, TargetY: tp.Y, TargetZ: tp.Z,
	})
}

// castLaunchedPacket is MagicSkillLaunched of c on targets.
func castLaunchedPacket(c Creature, id, level int, targets []skills.Creature) []byte {
	ids := make([]int32, 0, len(targets))
	for _, t := range targets {
		if t != nil {
			ids = append(ids, t.ObjectID())
		}
	}
	return serverpackets.NewMagicSkillLaunchedPacket(serverpackets.MagicSkillLaunched{
		CasterID:   int32(c.ID()), //nolint:gosec // object IDs fit
		SkillID:    int32(id),     //nolint:gosec // skill IDs are small
		SkillLevel: int32(level),  //nolint:gosec // skill levels are small
		TargetIDs:  ids,
	})
}

// castLaunched is Mobius onMagicLaunchedTimer, max(0, T − 400) ms into the
// cast: only aura skills go on without targets; targets beyond the effect
// range (plus the caster's collision), out of sight (casts over 550 ms;
// party heals see through walls) or, for negative skills, in a peace zone
// are dropped, and with none left the cast aborts with the reason's
// message. A cast that ended meanwhile, or a dead caster (static skills
// excepted), cancels; otherwise the hit comes 400 ms later.
func (s *Server) castLaunched(c Creature, b *skillBase, mu *magicUse) {
	st := &b.cast
	sk := mu.skill
	if !attackerInWorld(c) {
		s.abortCast(c)
		return
	}
	if len(mu.targets) == 0 {
		switch sk.TargetType {
		case data.TargetAura, data.TargetFrontAura, data.TargetBehindAura, data.TargetAuraCorpseMob, data.TargetAuraFriendly:
		default:
			s.abortCast(c)
			return
		}
	}
	escape := 0
	if sk.EffectRange > escape {
		escape = sk.EffectRange
	} else if sk.CastRange < 0 && sk.AffectRange > castEscapeAffectRange {
		escape = sk.AffectRange
	}
	p, _ := c.(*Player)
	if len(mu.targets) > 0 && escape > 0 {
		var skipRange, skipSight, skipPeace int
		kept := make([]skills.Creature, 0, len(mu.targets))
		pos := c.Position()
		for _, t := range mu.targets {
			gt := s.gameCreature(t)
			if gt == nil {
				continue
			}
			tp := gt.Position()
			if !inRadius3D(pos, tp, escape+attackCollision(c)) {
				skipRange++
				continue
			}
			if (sk.TargetType != data.TargetParty || !sk.HasEffectType(skills.TypeHeal)) &&
				mu.time > castSightTime && !s.canSee(c, gt) {
				skipSight++
				continue
			}
			if sk.HasNegativeEffect() && p != nil && s.insidePeaceZone(p, gt) {
				skipPeace++
				continue
			}
			kept = append(kept, t)
		}
		if len(kept) == 0 {
			if p != nil {
				switch {
				case skipRange > 0:
					p.SendSystemMessage(sysMsgCastDistanceTooFar)
				case skipSight > 0:
					p.SendSystemMessage(sysMsgCannotSeeTarget)
				case skipPeace > 0:
					p.SendMessage(castTextPeaceZone)
				}
			}
			s.abortCast(c)
			return
		}
		mu.targets = kept
	}
	if (mu.sim && !st.simCasting) || (!mu.sim && !st.casting) || (b.IsAlikeDead() && !sk.IsStatic()) {
		s.castNotifyCancel(c)
		return
	}
	if mu.time == 0 {
		s.castHit(c, b, mu)
		return
	}
	s.castSchedule(st, mu.sim, castLaunchLead, func() { s.castHit(c, b, mu) })
}

// castHit is Mobius onMagicHitTimer, at max(T, 400): a playable's skill
// moves the hate of attackable targets by its effect point; the shots
// recharge; the MP of the hit (abort when short), the HP and the charges
// go, with the caster's StatusUpdate; then callSkill, and the finalizer
// right after.
func (s *Server) castHit(c Creature, b *skillBase, mu *magicUse) {
	sk := mu.skill
	if !attackerInWorld(c) {
		s.abortCast(c)
		return
	}
	p, _ := c.(*Player)
	if p != nil {
		for _, t := range mu.targets {
			a := t.AsAttackable()
			switch {
			case a == nil:
			case sk.EffectPoint > 0:
				a.ReduceHate(b.self, int64(sk.EffectPoint))
			case sk.EffectPoint < 0:
				a.AddDamageHate(b.self, 0, int64(-sk.EffectPoint))
			}
		}
	}
	s.castRechargeShots(c, sk.UseSoulShot(), sk.UseSpiritShot())
	var values []serverpackets.StatusValue
	if mp := float64(skills.MPConsume(s.skills.env, b.self, sk)); mp > 0 {
		if mp > b.CurrentMP() {
			skills.Send(b.self, sysMsgNotEnoughMP)
			s.abortCast(c)
			return
		}
		s.castReduceMP(c, mp)
		values = append(values, serverpackets.StatusValue{Attr: serverpackets.StatusCurMP, Value: int32(b.CurrentMP())})
	}
	if sk.HPConsume > 0 {
		hp := float64(sk.HPConsume)
		if hp >= b.CurrentHP() {
			skills.Send(b.self, sysMsgNotEnoughHP)
			s.abortCast(c)
			return
		}
		c.ReduceHP(hp, c)
		values = append(values, serverpackets.StatusValue{Attr: serverpackets.StatusCurHP, Value: int32(b.CurrentHP())})
	}
	if p != nil && len(values) > 0 {
		p.Send(castStatusPacket(p, values...))
	}
	if p != nil && sk.ChargeConsume > 0 {
		s.castDecreaseCharges(p, sk.ChargeConsume)
	}
	s.callSkill(c, b, sk, mu.targets)
	if mu.time > 0 {
		mu.count++
		s.castSchedule(&b.cast, mu.sim, 0, func() { s.castFinish(c, b, mu) })
		return
	}
	s.castFinish(c, b, mu)
}

// castRechargeShots charges a player's automatic shots
// (Player.rechargeShots) through the ShotRecharger.
func (s *Server) castRechargeShots(c Creature, physical, magic bool) {
	p, ok := c.(*Player)
	if !ok {
		return
	}
	if fn := s.castFeatureOf().shots; fn != nil {
		s.safely("ShotRecharger", func() { fn(p, physical, magic) })
	}
}

// castDecreaseCharges takes n force charges (Mobius decreaseCharges):
// nothing when p has fewer; EtcStatusUpdate shows the rest.
func (s *Server) castDecreaseCharges(p *Player, n int) bool {
	a := p.skill
	if a.charges < n {
		return false
	}
	a.charges -= n
	if p.InGame() {
		p.Send(etcStatusPacket(p))
	}
	return true
}

// castFinish is Mobius onMagicFinalizer: the simultaneous slot just ends.
// Otherwise the cast ends (shots recharge after a timed cast); a
// nextActionAttack skill whose target is still the caster's target makes a
// player attack it 333 ms later (unless Shift was held, or the player waits
// to walk somewhere, which delays the rest) and an NPC at once.
func (s *Server) castFinish(c Creature, b *skillBase, mu *magicUse) {
	st := &b.cast
	if mu.sim {
		st.simCasting = false
		return
	}
	st.interrupt = 0
	if mu.count > 0 {
		s.castRechargeShots(c, mu.skill.UseSoulShot(), mu.skill.UseSpiritShot())
	}
	st.casting, st.simCasting = false, false
	sk := mu.skill
	var target skills.Creature
	if len(mu.targets) > 0 {
		target = mu.targets[0]
	}
	if cur := b.Target(); sk.NextActionAttack && cur != nil && !castSame(cur, b.self) && castSame(cur, target) {
		gt := s.gameCreature(target)
		switch v := c.(type) {
		case *Player:
			if !s.castNextIsMove(v) {
				if st.current == nil || !st.current.shift {
					s.castLater(st, castNextAttackDelay, func() {
						if v.InGame() && v.InWorld() && !castCreatureDisabled(b) && !s.AttackingNow(v) && !st.casting && !st.simCasting {
							s.playerIntendAttack(v, gt)
						}
					})
				}
			} else {
				s.castLater(st, castNextAttackDelay, func() { s.castComplete(c, b, sk, target) })
				return
			}
		case *Npc:
			if gt != nil && v.ai.attached {
				s.npcSetAttack(v, gt)
			}
		}
	}
	s.castComplete(c, b, sk, target)
}

// castSame reports whether a and b are the same creature.
func castSame(a, b skills.Creature) bool {
	return a != nil && b != nil && a.ObjectID() == b.ObjectID()
}

// castCreatureDisabled is Mobius isDisabled (without the AI switch): dead,
// faking death, physically unable to attack, stunned, asleep or paralyzed.
func castCreatureDisabled(b *skillBase) bool {
	return b.IsAlikeDead() || b.el.IsAffected(skills.FlagPhysicalAttackMuted) || allSkillsDisabled(b)
}

// castComplete is Mobius completeMagicFinalizer: a negative skill starts the
// caster's combat stance; a player's intention goes back to what the cast
// interrupted; the finished hooks run; a player's queued skill is used next.
func (s *Server) castComplete(c Creature, b *skillBase, sk *skills.Skill, target skills.Creature) {
	if sk.HasNegativeEffect() && sk.TargetType != data.TargetUnlockable {
		s.StartAutoAttack(c)
	}
	p, isPlayer := c.(*Player)
	if isPlayer && p.InGame() {
		s.castFinishedAI(p)
	}
	gt := s.gameCreature(target)
	for _, fn := range s.castFeatureOf().finished {
		s.safely("OnSkillFinished hook", func() { fn(c, sk, gt) })
	}
	if !isPlayer {
		return
	}
	st := &b.cast
	q := st.queued
	st.current = nil
	if q != nil {
		st.queued = nil
		s.castLater(st, 0, func() {
			if p.InGame() && p.InWorld() {
				s.useMagic(p, q.skill, q.ctrl, q.shift)
			}
		})
	}
}

// abortCast is Mobius abortCast.
func (s *Server) abortCast(c Creature) {
	st := s.castOf(c)
	if st == nil || (!st.casting && !st.simCasting) {
		return
	}
	for _, id := range []*world.TimerID{&st.timer, &st.simTimer} {
		if *id != 0 {
			s.loop.Cancel(*id)
			*id = 0
		}
	}
	st.casting, st.simCasting = false, false
	st.interrupt = 0
	p, isPlayer := c.(*Player)
	if isPlayer && p.InGame() {
		s.castFinishedAI(p)
	}
	if attackerInWorld(c) {
		s.world.Broadcast(c, serverpackets.NewMagicSkillCanceledPacket(int32(c.ID())), true) //nolint:gosec // object IDs fit
	}
	if isPlayer {
		p.SendActionFailed()
	}
}

// castNotifyCancel is Mobius notifyActionCancel at a launch that found the
// cast over: the saved intention is dropped, the cast aborts, the follow
// stops, AutoAttackStop goes out without a stance, and the intention
// thinks again.
func (s *Server) castNotifyCancel(c Creature) {
	if p, ok := c.(*Player); ok {
		if st := s.castOf(p); st != nil {
			st.saved = castSaved{}
		}
		if ai := s.attack.players[p]; ai != nil {
			ai.savedMove = nil
		}
	}
	s.abortCast(c)
	switch v := c.(type) {
	case *Player:
		s.playerStopFollow(v)
		if !s.InAttackStance(v) && v.InWorld() {
			s.world.Broadcast(v, serverpackets.NewAutoAttackStopPacket(v.ObjectID()), true)
		}
		s.playerThink(v)
	case *Npc:
		if !s.InAttackStance(v) && v.InWorld() {
			s.world.Broadcast(v, serverpackets.NewAutoAttackStopPacket(int32(v.ID())), true) //nolint:gosec // object IDs fit
		}
		s.npcAttackEvent(v, NpcAttackCancelled)
	}
}

// castLeave is the OnPlayerLeave hook: the cast, its timers, the queue and
// the intention end without packets.
func (s *Server) castLeave(p *Player) {
	st := s.castOf(p)
	if st == nil {
		return
	}
	for _, id := range append([]world.TimerID{st.timer, st.simTimer}, st.later...) {
		if id != 0 {
			s.loop.Cancel(id)
		}
	}
	*st = castState{disabled: st.disabled}
}

// checkDoCastConditions is Mobius checkDoCastConditions: a skill without a
// port (K2) or disabled; MP for the start and the hit ("Not enough MP."),
// HP over the HP it takes ("Not enough HP."); a muted caster for magic, a
// physically muted one for the rest (static skills excepted); a weapon that
// uses only its own skills; the item the skill consumes. A player gets
// ActionFailed where Mobius sends it.
func (s *Server) checkDoCastConditions(c Creature, b *skillBase, sk *skills.Skill) bool {
	p, _ := c.(*Player)
	failed := func() bool {
		if p != nil {
			p.SendActionFailed()
		}
		return false
	}
	if sk == nil {
		return failed()
	}
	if why := sk.Refusal(); why != "" {
		s.log.Info("game: a skill without a port was used", "skill", sk.String(), "objectId", c.ID(), "why", why)
		return failed()
	}
	if s.skillDisabled(b, sk) {
		return failed()
	}
	env := s.skills.env
	if b.CurrentMP() < float64(skills.MPConsume(env, b.self, sk)+skills.MPInitialConsume(b.self, sk)) {
		skills.Send(b.self, sysMsgNotEnoughMP)
		return failed()
	}
	if b.CurrentHP() <= float64(sk.HPConsume) {
		skills.Send(b.self, sysMsgNotEnoughHP)
		return failed()
	}
	if !sk.IsStatic() {
		if sk.IsMagic() {
			if b.el.IsAffected(skills.FlagMuted) {
				return failed()
			}
		} else if b.el.IsAffected(skills.FlagPhysicalMuted) {
			return failed()
		}
	}
	if w := s.activeWeaponItem(c); w != nil && w.Weapon.WeaponSkillsOnly && (p == nil || !p.IsGM()) && len(w.Skills) > 0 {
		found := false
		for _, ws := range w.Skills {
			if ws.ID == sk.ID {
				found = true
			}
		}
		if !found {
			if p != nil {
				p.SendMessage(castTextWeaponOnly)
			}
			return false
		}
	}
	if sk.ItemConsumeID > 0 && p != nil && p.inv.Inventory != nil {
		it := p.inv.ByItemID(int32(sk.ItemConsumeID)) //nolint:gosec // item IDs are small
		if it == nil || it.Count() < int64(sk.ItemConsumeCount) {
			if sk.HasEffectType(skills.TypeSummon) {
				p.SendSystemMessage(sysMsgServitorCosts, s.itemNameParam(int32(sk.ItemConsumeID)), //nolint:gosec // item IDs are small
					serverpackets.NumberParam(int32(sk.ItemConsumeCount))) //nolint:gosec // small counts
			} else {
				p.SendMessage(castTextNotEnoughIt)
			}
			return false
		}
	}
	return true
}

// The SkillCaster methods, which the creature adapters call.

// IsCastingNow is Mobius isCastingNow.
func (f *castFeature) IsCastingNow(c skills.Creature) bool {
	return f.s.CastingNow(f.s.gameCreature(c))
}

// CanAbortCast is Mobius canAbortCast.
func (f *castFeature) CanAbortCast(c skills.Creature) bool {
	return f.s.CanAbortCast(f.s.gameCreature(c))
}

// AbortCast is Mobius abortCast.
func (f *castFeature) AbortCast(c skills.Creature) {
	if gc := f.s.gameCreature(c); gc != nil {
		f.s.abortCast(gc)
	}
}

// BreakCast is Mobius breakCast.
func (f *castFeature) BreakCast(c skills.Creature) {
	if gc := f.s.gameCreature(c); gc != nil {
		f.s.BreakCast(gc)
	}
}

// TriggerCast is Mobius makeTriggerCast: the skill's conditions, its reuse
// (a disable for the skill's reuse delay), the target list (or target
// alone), MagicSkillUse and MagicSkillLaunched without times, then the
// effects at once.
func (f *castFeature) TriggerCast(c skills.Creature, sk *skills.Skill, target skills.Creature, ignoreTargetType bool) {
	s := f.s
	gc := s.gameCreature(c)
	b := s.skillBaseOf(gc)
	if b == nil || sk == nil || !attackerInWorld(gc) {
		return
	}
	env := s.skills.env
	if !sk.CheckCondition(env, c, target, false) || s.skillDisabled(b, sk) {
		return
	}
	if sk.ReuseDelay > 0 {
		s.disableSkill(b, sk, time.Duration(sk.ReuseDelay)*time.Millisecond)
	}
	var targets []skills.Creature
	if ignoreTargetType {
		if target == nil {
			return
		}
		targets = []skills.Creature{target}
	} else {
		targets = sk.Targets(env, c, false, target)
	}
	if len(targets) == 0 {
		return
	}
	skillTarget := target
	for _, t := range targets {
		if t != nil {
			skillTarget = t
			break
		}
	}
	if gt := s.gameCreature(skillTarget); gt != nil {
		s.world.Broadcast(gc, castUsePacket(gc, gt, sk.DisplayID, sk.Level, 0, 0), true)
	}
	s.world.Broadcast(gc, castLaunchedPacket(gc, sk.DisplayID, sk.Level, targets), true)
	sk.Activate(env, c, targets)
}
