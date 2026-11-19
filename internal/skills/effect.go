package skills

import (
	"fmt"
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
)

// Effect is an effect handler of a skill level, Mobius AbstractEffect. The
// handlers live in internal/skills/effects; each embeds Base, which gives
// the Mobius defaults, and overrides what its Mobius class overrides.
//
// An instant effect (Instant) runs once when the skill lands: CalcSuccess
// is its own roll, OnStart its action. A continuous effect is collected into
// a BuffInfo when CanStart allows it; OnStart runs when the BuffInfo is
// added to the target's effect list, OnActionTime every Ticks effect ticks,
// OnExit when it ends. Its stat functions (Funcs) become stats.Modifier
// values for as long as it lasts.
//
// Effects are built once per skill level by the Catalog and shared by every
// creature: they keep no per-creature state.
type Effect interface {
	// Name is the handler's name.
	Name() data.EffectName
	// Ticks is the effect's period in effect ticks, 0 if it does not tick.
	Ticks() int
	// Funcs are the stat functions the effect adds while it lasts.
	Funcs() []StatFunc
	// Type is the Mobius effect type.
	Type() EffectType
	// Flags are the effect flags it sets while it lasts (stunned, ...).
	Flags() EffectFlags
	// Instant reports an instant effect.
	Instant() bool
	// CalcSuccess is an instant effect's own success roll.
	CalcSuccess(info *BuffInfo) bool
	// CanStart decides whether a continuous effect joins its BuffInfo.
	CanStart(info *BuffInfo) bool
	// OnStart runs an instant effect, or starts a continuous one.
	OnStart(info *BuffInfo)
	// OnActionTime runs on each tick; for a toggle, false ends the toggle.
	OnActionTime(info *BuffInfo) bool
	// OnExit ends a continuous effect.
	OnExit(info *BuffInfo)

	base() *Base
}

// Base holds what every effect has: its name, ticks and stat functions. It
// implements every Effect method with the AbstractEffect default; embed it
// and override the rest.
type Base struct {
	name  data.EffectName
	ticks int
	funcs []StatFunc
	// passive holds the unconditional functions as the modifiers of a
	// passive skill (owner OwnerSkill, the skill's ID), built once by the
	// Catalog and shared by every creature with the skill (K8).
	passive []stats.Modifier
}

// NewBase returns the base of an effect: its name and ticks from the data.
// The Catalog adds the compiled stat functions.
func NewBase(def *data.SkillEffect) Base {
	return Base{name: def.Name, ticks: def.Ticks}
}

func (b *Base) base() *Base { return b }

// Name returns the handler's name.
func (b *Base) Name() data.EffectName { return b.name }

// Ticks returns the effect's period in effect ticks.
func (b *Base) Ticks() int { return b.ticks }

// TicksMultiplier is Mobius getTicksMultiplier: ticks × EffectTickRatio /
// 1000, computed as a Java float.
func (b *Base) TicksMultiplier(env *Env) float64 {
	return float64(float32(b.ticks*env.Config.EffectTickRatio) / 1000)
}

// Funcs returns the compiled stat functions.
func (b *Base) Funcs() []StatFunc { return b.funcs }

// Type returns TypeNone.
func (b *Base) Type() EffectType { return TypeNone }

// Flags returns the mask of FlagNone, as AbstractEffect does.
func (b *Base) Flags() EffectFlags { return FlagNone.Mask() }

// Instant returns false.
func (b *Base) Instant() bool { return false }

// CalcSuccess returns true.
func (b *Base) CalcSuccess(*BuffInfo) bool { return true }

// CanStart returns true.
func (b *Base) CanStart(*BuffInfo) bool { return true }

// OnStart does nothing.
func (b *Base) OnStart(*BuffInfo) {}

// OnActionTime returns false.
func (b *Base) OnActionTime(*BuffInfo) bool { return false }

// OnExit does nothing.
func (b *Base) OnExit(*BuffInfo) {}

// EffectType is the type of an effect, Mobius EffectType. Skills ask for
// it (Skill.HasEffectType) to tell damage skills, heals, potions and the
// like apart.
type EffectType uint8

// The effect types, in Mobius order.
const (
	TypeAggression EffectType = iota
	TypeBuff
	TypeCharmOfLuck
	TypeChatBlock
	TypeCPHeal
	TypeDebuff
	TypeDispel
	TypeDispelBySlot
	TypeDmgOverTime
	TypeDmgOverTimePercent
	TypeDeathLink
	TypeFakeDeath
	TypeFear
	TypeFishing
	TypeFishingStart
	TypeHate
	TypeHeal
	TypeHPDrain
	TypeMagicalAttack
	TypeManaHealByLevel
	TypeManaHealPercent
	TypeMute
	TypeNoblesseBlessing
	TypeNone
	TypeParalyze
	TypePhysicalAttack
	TypePhysicalAttackHPLink
	TypeRebalanceHP
	TypeRefuelAirship
	TypeRelaxing
	TypeResurrection
	TypeResurrectionSpecial
	TypeRoot
	TypeSleep
	TypeStealAbnormal
	TypeStun
	TypeSummon
	TypeSummonPet
	TypeSummonNpc
	TypeTeleport
	TypeTeleportToTarget
	numEffectTypes
)

var effectTypeNames = [numEffectTypes]string{
	"AGGRESSION", "BUFF", "CHARM_OF_LUCK", "CHAT_BLOCK", "CPHEAL", "DEBUFF", "DISPEL", "DISPEL_BY_SLOT",
	"DMG_OVER_TIME", "DMG_OVER_TIME_PERCENT", "DEATH_LINK", "FAKE_DEATH", "FEAR", "FISHING", "FISHING_START",
	"HATE", "HEAL", "HP_DRAIN", "MAGICAL_ATTACK", "MANAHEAL_BY_LEVEL", "MANAHEAL_PERCENT", "MUTE",
	"NOBLESSE_BLESSING", "NONE", "PARALYZE", "PHYSICAL_ATTACK", "PHYSICAL_ATTACK_HP_LINK", "REBALANCE_HP",
	"REFUEL_AIRSHIP", "RELAXING", "RESURRECTION", "RESURRECTION_SPECIAL", "ROOT", "SLEEP", "STEAL_ABNORMAL",
	"STUN", "SUMMON", "SUMMON_PET", "SUMMON_NPC", "TELEPORT", "TELEPORT_TO_TARGET",
}

// String returns the Mobius name of t.
func (t EffectType) String() string {
	if t < numEffectTypes {
		return effectTypeNames[t]
	}
	return fmt.Sprintf("EffectType(%d)", uint8(t))
}

// effectTypeSet is a set of effect types.
type effectTypeSet uint64

func (s effectTypeSet) has(t EffectType) bool { return t < numEffectTypes && s&(1<<t) != 0 }

// EffectFlag is a state an effect puts its creature in, Mobius EffectFlag.
type EffectFlag uint8

// The effect flags, in Mobius order.
const (
	FlagNone EffectFlag = iota
	FlagResurrectionSpecial
	FlagNoblessBlessing
	FlagSilentMove
	FlagProtectionBlessing
	FlagRelaxing
	FlagFear
	FlagConfused
	FlagMuted
	FlagPhysicalMuted
	FlagPhysicalAttackMuted
	FlagPassive
	FlagDisarmed
	FlagRooted
	FlagSleep
	FlagStunned
	FlagBetrayed
	FlagInvul
	FlagParalyzed
	FlagBlockResurrection
	FlagServitorShare
	FlagPolearmSingleTarget
)

// Mask returns the bit of f (Mobius getMask, 1 << ordinal).
func (f EffectFlag) Mask() EffectFlags { return 1 << f }

// EffectFlags is a set of effect flags.
type EffectFlags uint32

// Has reports whether f is in the set.
func (fs EffectFlags) Has(f EffectFlag) bool { return fs&f.Mask() != 0 }

// StatFunc is a compiled stat function of an effect, Mobius FuncTemplate:
// Op applies Value to Stat at Order, if Cond (when not nil) holds for the
// creature whose stat it is, the other creature and the skill of the
// calculation.
type StatFunc struct {
	Stat  stats.Stat
	Op    stats.Op
	Order int
	Value float64
	Cond  Cond
}

// statOps maps the data's stat functions to the stat engine's operations.
var statOps = map[data.StatOp]stats.Op{
	data.OpAdd:       stats.OpAdd,
	data.OpDiv:       stats.OpDiv,
	data.OpEnchant:   stats.OpEnchant,
	data.OpEnchantHP: stats.OpEnchantHP,
	data.OpMul:       stats.OpMul,
	data.OpSet:       stats.OpSet,
	data.OpShare:     stats.OpShare,
	data.OpSub:       stats.OpSub,
}

// compileFunc compiles a stat function of the data. Condition kinds
// without a port are added to unported.
func compileFunc(f *data.StatFunc, unported map[data.ConditionKind]bool) (StatFunc, error) {
	op, ok := statOps[f.Op]
	st := stats.Stat(f.Stat)
	if !ok || !f.Stat.Valid() || !st.Valid() {
		return StatFunc{}, fmt.Errorf("stat function %s %s: unknown stat or operation", f.Op, f.Stat)
	}
	sf := StatFunc{Stat: st, Op: op, Order: f.Order, Value: f.Value}
	if sf.Order < 0 { // FuncTemplate takes the operation's order then
		sf.Order = op.DefaultOrder()
	}
	if f.Cond != nil {
		sf.Cond = compileCondition(f.Cond, unported)
	}
	return sf, nil
}

// modifier returns the modifier of the function on holder's stats, owned by
// owner. A condition sees holder as the caster, and the calculation's other
// creature and skill (stats.Context) as the target and skill.
func (f *StatFunc) modifier(env *Env, owner stats.Owner, holder Creature) stats.Modifier {
	m := stats.Modifier{Stat: f.Stat, Op: f.Op, Order: f.Order, Value: f.Value, Owner: owner}
	if f.Cond != nil {
		cond := f.Cond
		m.Cond = stats.CondFunc(func(ctx stats.Context) bool {
			target, _ := ctx.Target.(Creature)
			sk, _ := ctx.Skill.(*Skill)
			return cond(ConditionContext{Env: env, Caster: holder, Target: target, Skill: sk})
		})
	}
	return m
}

// sharedModifiers returns the modifiers of the functions, owned by owner,
// when none has a condition: then every creature can share them. It
// returns nil when a function has a condition, whose closure needs its
// creature.
func sharedModifiers(funcs []StatFunc, owner stats.Owner) []stats.Modifier {
	if len(funcs) == 0 {
		return nil
	}
	mods := make([]stats.Modifier, len(funcs))
	for i := range funcs {
		f := &funcs[i]
		if f.Cond != nil {
			return nil
		}
		mods[i] = stats.Modifier{Stat: f.Stat, Op: f.Op, Order: f.Order, Value: f.Value, Owner: owner}
	}
	return mods
}

// Factory builds the effect of an <effect> of a skill level, parsing its
// parameters once into typed values (K2). It must not keep def.
type Factory func(def *data.SkillEffect) (Effect, error)

// Registry maps effect names to their factories. Names without a factory
// load as stubs that refuse the cast.
type Registry struct {
	factories map[data.EffectName]Factory
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[data.EffectName]Factory)}
}

// Register sets the factory of an effect name, replacing any earlier one.
func (r *Registry) Register(name data.EffectName, f Factory) {
	if f != nil {
		r.factories[name] = f
	}
}

// Ported reports whether the name has a factory.
func (r *Registry) Ported(name data.EffectName) bool {
	_, ok := r.factories[name]
	return ok
}

// Names returns the names with a factory, sorted.
func (r *Registry) Names() []data.EffectName {
	names := make([]data.EffectName, 0, len(r.factories))
	for n := range r.factories {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// stub stands for an effect without a port, or one whose parameters did not
// parse. It is an instant effect that never runs, so it never joins a
// BuffInfo; the skill that has it refuses to be used.
type stub struct {
	Base
	reason string
}

func (s *stub) Instant() bool              { return true }
func (s *stub) CalcSuccess(*BuffInfo) bool { return false }

// StubReason returns why e is a stub ("effect X is not ported", ...), or ""
// for a real effect.
func StubReason(e Effect) string {
	if s, ok := e.(*stub); ok {
		return s.reason
	}
	return ""
}

// build makes the effect of def: the factory's, or a stub. ported is false
// for a stub.
func (r *Registry) build(def *data.SkillEffect) (e Effect, ported bool, err error) {
	f, ok := r.factories[def.Name]
	if !ok {
		return &stub{Base: NewBase(def), reason: fmt.Sprintf("effect %s is not ported yet", def.Name)}, false, nil
	}
	e, err = f(def)
	if err != nil {
		return &stub{Base: NewBase(def), reason: fmt.Sprintf("effect %s: %v", def.Name, err)}, true, err
	}
	if e == nil {
		return &stub{Base: NewBase(def), reason: fmt.Sprintf("effect %s: no effect built", def.Name)}, true, fmt.Errorf("effect %s: the factory built nothing", def.Name)
	}
	return e, true, nil
}
