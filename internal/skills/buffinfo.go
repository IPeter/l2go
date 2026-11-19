package skills

import (
	"github.com/IPeter/l2go/internal/stats"
)

// FinishType says how a BuffInfo ends, which picks its message (Mobius
// SkillFinishType).
type FinishType uint8

// The finish types.
const (
	// FinishNormal: the effect ran out ("S1 has worn off."); END effects
	// run.
	FinishNormal FinishType = iota
	// FinishRemoved: the effect was taken away ("The effect of S1 has been
	// removed."); END effects do not run.
	FinishRemoved
	// FinishSilent: no message; END effects run.
	FinishSilent
)

// BuffInfo is the set of continuous effects one skill put on one creature,
// Mobius BuffInfo: who cast it, on whom, its effects, its abnormal time and
// start tick. A BuffInfo with continuous effects lives in the effected's
// EffectList until it ends; instant effects get a BuffInfo too, as their
// context, that never joins a list.
type BuffInfo struct {
	env      *Env
	effector Creature
	effected Creature
	skill    *Skill
	effects  []Effect
	// tasks are the tick timers, parallel to effects (0 for none).
	tasks []TimerID

	abnormalTime int
	periodStart  int64
	finish       FinishType
	inUse        bool

	// owner owns the effects' modifiers: the skill ID for a passive, a
	// number of the effect list for anything else.
	owner stats.Owner
	// clockCount is what the buff clock counted (BuffFinishTask); onClock
	// tells whether the buff is on the clock.
	clockCount int
	onClock    bool
	list       *EffectList
}

// NewBuffInfo is the BuffInfo constructor: it computes the abnormal time
// (AbnormalTime, which rolls skill mastery for players that have it) and
// takes the start tick from the scheduler.
func NewBuffInfo(env *Env, effector, effected Creature, sk *Skill) *BuffInfo {
	return &BuffInfo{
		env:          env,
		effector:     effector,
		effected:     effected,
		skill:        sk,
		abnormalTime: AbnormalTime(env, effector, effected, sk),
		periodStart:  env.Sched.GameTicks(),
		inUse:        true,
	}
}

// Env returns the environment the BuffInfo was made in.
func (b *BuffInfo) Env() *Env { return b.env }

// Effector returns the creature that caused the effects.
func (b *BuffInfo) Effector() Creature { return b.effector }

// Effected returns the creature the effects act on.
func (b *BuffInfo) Effected() Creature { return b.effected }

// Skill returns the skill of the effects.
func (b *BuffInfo) Skill() *Skill { return b.skill }

// Effects returns the continuous effects. The slice must not be changed.
func (b *BuffInfo) Effects() []Effect { return b.effects }

// AbnormalTime returns the duration in seconds; -1 (or less, after a
// mastery roll) for passives and toggles.
func (b *BuffInfo) AbnormalTime() int { return b.abnormalTime }

// SetAbnormalTime sets the duration in seconds (a restored buff's time
// left, a custom time).
func (b *BuffInfo) SetAbnormalTime(seconds int) { b.abnormalTime = seconds }

// PeriodStartTicks returns the game tick the BuffInfo was made at.
func (b *BuffInfo) PeriodStartTicks() int64 { return b.periodStart }

// Time returns the seconds left, as the icons show them: the abnormal time
// less the whole seconds of game ticks since the start (BuffInfo.getTime).
func (b *BuffInfo) Time() int {
	return b.abnormalTime - int((b.env.Sched.GameTicks()-b.periodStart)/10)
}

// IsRemoved reports a BuffInfo that was taken away (FinishRemoved).
func (b *BuffInfo) IsRemoved() bool { return b.finish == FinishRemoved }

// Finish returns how the BuffInfo ended (FinishNormal until it does).
func (b *BuffInfo) Finish() FinishType { return b.finish }

// InUse reports whether the effects act: false while a herb hides the
// buff, which keeps its time running.
func (b *BuffInfo) InUse() bool { return b.inUse }

// addEffect adds a continuous effect.
func (b *BuffInfo) addEffect(e Effect) { b.effects = append(b.effects, e) }

// initializeEffects starts the effects once the BuffInfo is in the list
// (BuffInfo.initializeEffects): the "flow through you" message, the buff
// clock, OnStart, the tick timers, the stats and the visual effects. A dead
// target gets nothing, unless the skill is passive.
func (b *BuffInfo) initializeEffects() {
	if b.effected == nil || b.skill == nil {
		return
	}
	sk := b.skill
	if isPlayer(b.effected) && !sk.IsPassive() {
		if p := b.effected.AsPlayer(); p.HasEnteredWorld() || b.env.Config.ShowEffectMessagesOnLogin {
			send(b.effected, MsgEffectsOfS1FlowThroughYou, SkillName(sk))
		}
	}
	if b.abnormalTime > 0 && b.list != nil {
		b.list.clockAdd(b)
	}
	b.tasks = make([]TimerID, len(b.effects))
	update := false
	for i, e := range b.effects {
		if e.Instant() || (b.effected.IsDead() && !sk.IsPassive()) {
			continue
		}
		e.OnStart(b)
		// A target the start killed gets no continuous effect: it would
		// tick forever.
		if b.effected.IsDead() && !sk.IsPassive() {
			continue
		}
		if e.Ticks() > 0 {
			b.tasks[i] = b.env.Sched.Every(b.env.tickPeriod(e.Ticks()), func() { b.onTick(i) })
		}
		b.addEffectStats(e)
		update = true
	}
	if update {
		b.addVisuals()
	}
}

// onTick runs effect i's action (BuffInfo.onTick). A hidden buff's effects
// do not act. A toggle ends when the action asks to stop.
func (b *BuffInfo) onTick(i int) {
	e := b.effects[i]
	forever := false
	if b.inUse {
		forever = e.OnActionTime(b)
	}
	if !forever && b.skill.IsToggle() {
		if id := b.tasks[i]; id != 0 {
			b.env.Sched.Cancel(id)
			b.tasks[i] = 0
		}
		b.effected.Effects().StopSkill(FinishRemoved, b.skill.ID)
	}
}

// stopAllEffects ends the effects without removing the BuffInfo from its
// list (BuffInfo.stopAllEffects): it leaves the buff clock and finishes.
func (b *BuffInfo) stopAllEffects(ft FinishType, broadcast bool) {
	b.finish = ft
	if b.list != nil {
		b.list.clockRemove(b)
	}
	b.finishEffects(broadcast)
}

// finishEffects cancels the ticks, removes the stats, runs OnExit, removes
// the visual effects, sends the end message and resets the short buff
// (BuffInfo.finishEffects). A toggle is "aborted", a removed effect
// "removed", any other non-passive effect "worn off".
func (b *BuffInfo) finishEffects(broadcast bool) {
	for i, id := range b.tasks {
		if id != 0 {
			b.env.Sched.Cancel(id)
			b.tasks[i] = 0
		}
	}
	b.removeStats()
	for _, e := range b.effects {
		if !e.Instant() {
			e.OnExit(b)
		}
	}
	b.removeVisuals(broadcast)
	var msg = MsgS1HasWornOff
	switch {
	case b.finish == FinishSilent:
		msg = 0
	case b.skill.IsToggle():
		msg = MsgS1HasBeenAborted
	case b.finish == FinishRemoved:
		msg = MsgEffectOfS1Removed
	case b.skill.IsPassive():
		msg = 0
	}
	if msg != 0 {
		send(b.effected, msg, SkillName(b.skill))
	}
	if l := b.effected.Effects(); l != nil && l.shortBuff == b {
		l.shortBuffUpdate(nil)
	}
}

// addEffectStats adds an effect's stat functions as modifiers of the
// effected.
func (b *BuffInfo) addEffectStats(e Effect) {
	base := e.base()
	if len(base.funcs) == 0 || b.owner == (stats.Owner{}) {
		return
	}
	mods := base.passive
	if mods == nil || !b.skill.IsPassive() {
		mods = make([]stats.Modifier, len(base.funcs))
		for i := range base.funcs {
			mods[i] = base.funcs[i].modifier(b.env, b.owner, b.effected)
		}
	}
	if err := b.effected.AddModifiers(mods...); err != nil {
		b.env.log().Warn("skills: cannot add the stats of an effect", "skill", b.skill.String(), "effect", e.Name().String(), "err", err)
	}
}

// addStats adds the stats of every effect (BuffInfo.addStats), when a herb
// gives a hidden buff back.
func (b *BuffInfo) addStats() {
	for _, e := range b.effects {
		b.addEffectStats(e)
	}
}

// removeStats removes the stats of every effect (BuffInfo.removeStats).
func (b *BuffInfo) removeStats() {
	if b.owner != (stats.Owner{}) {
		b.effected.RemoveOwner(b.owner)
	}
}

// addVisuals starts the skill's abnormal visual effects on the effected
// (event effects on players only) and broadcasts them.
func (b *BuffInfo) addVisuals() {
	v := b.skill.visual
	l := b.effected.Effects()
	l.visual.Normal |= v.Normal
	if isPlayer(b.effected) {
		l.visual.Event |= v.Event
	}
	l.visual.Special |= v.Special
	b.effected.UpdateAbnormalEffect()
}

// removeVisuals stops the skill's abnormal visual effects, broadcasting
// when asked.
func (b *BuffInfo) removeVisuals(broadcast bool) {
	v := b.skill.visual
	l := b.effected.Effects()
	l.visual.Normal &^= v.Normal
	if isPlayer(b.effected) {
		l.visual.Event &^= v.Event
	}
	l.visual.Special &^= v.Special
	if broadcast {
		b.effected.UpdateAbnormalEffect()
	}
}
