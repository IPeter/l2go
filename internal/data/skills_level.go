package data

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

// BaseNone is the basic property of a skill without one (Mobius
// BaseStat.NONE, the ordinal after MEN). Its stat bonus is 1.
const BaseNone BaseStat = 6

// Skill is one level of a skill, resolved from its SkillTemplate as Mobius
// DocumentSkill and the Skill constructor build it: every value is typed
// and holds the Mobius default when the XML has none. Times are in
// milliseconds unless noted.
type Skill struct {
	ID    int
	Level int
	// DisplayID and DisplayLevel are what the client shows (ID and Level
	// by default).
	DisplayID    int
	DisplayLevel int
	Name         string
	Icon         string

	OperateType SkillOperateType
	TargetType  TargetType
	// Magic is Mobius isMagic: 0 physical, 1 magic, 2 static, 3 dance.
	Magic int
	// Trait is the attack trait of the skill (TraitNone if none).
	Trait TraitType
	// BasicProperty is the base stat whose bonus scales the success of its
	// effects (BaseNone if none).
	BasicProperty BaseStat
	MagicLevel    int

	MPConsume        int
	MPInitialConsume int
	// MPPerChanneling is the MP a channeling skill takes per tick
	// (MPConsume by default).
	MPPerChanneling  int
	HPConsume        int
	ItemConsumeID    int
	ItemConsumeCount int
	ChargeConsume    int

	// CastRange and EffectRange are -1 when not set.
	CastRange   int
	EffectRange int
	AffectRange int
	// AffectLimit is the "min-max" affect limit (zero when not set).
	AffectLimit [2]int

	HitTime     int
	CoolTime    int
	ReuseDelay  int
	StaticReuse bool

	// AbnormalType, AbnormalLevel and AbnormalTime (in seconds) describe
	// the effect a buff or debuff leaves.
	AbnormalType    AbnormalType
	AbnormalLevel   int
	AbnormalTime    int
	AbnormalInstant bool
	// AbnormalVisualEffects are the effects the client shows while it lasts.
	AbnormalVisualEffects []AbnormalVisualEffect

	// Power, PvPPower and PvEPower hold Mobius's float values (PvPPower and
	// PvEPower default to Power).
	Power    float64
	PvPPower float64
	PvEPower float64
	// ActivateRate is the base success chance of its effects, -1 for none.
	ActivateRate int
	LvlBonusRate int
	// MinChance and MaxChance bound the success chance; -1 is the
	// configured MinAbnormalStateSuccessRate or MaxAbnormalStateSuccessRate.
	MinChance    int
	MaxChance    int
	BlowChance   int
	BaseCritRate int
	// Element is the attack attribute (0 fire ... 5 dark), -1 for none.
	Element      int
	ElementPower int
	Attribute    string
	EffectPoint  int

	IsDebuff                     bool
	IsRecoveryHerb               bool
	IgnoreShield                 bool
	NextActionAttack             bool
	RemovedOnAnyActionExceptMove bool
	RemovedOnDamage              bool
	BlockedInOlympiad            bool
	OverHit                      bool
	SuicideAttack                bool
	DirectHPDamage               bool
	StayAfterDeath               bool
	// StayOnSubclassChange and CanBeDispeled are true by default.
	StayOnSubclassChange bool
	CanBeDispeled        bool
	ExcludedFromCheck    bool
	SimultaneousCast     bool
	PvPOnly              bool

	Feed           int
	MinPledgeClass int
	// ChannelingSkillID is the skill a channeling skill casts each tick;
	// the interval and first delay are in milliseconds.
	ChannelingSkillID          int
	ChannelingTickInterval     int
	ChannelingTickInitialDelay int

	// Conditions are what the caster needs to use the skill, Mobius's
	// preconditions: each is tested in turn, and the first that fails
	// sends its message.
	Conditions []SkillCondition
	// Effects are the effects by scope (ScopeGeneral, ScopeSelf, ...); a
	// passive skill has its <effects> as ScopePassive.
	Effects [NumEffectScopes][]SkillEffect
}

// SkillCondition is a resolved <conditions> block.
type SkillCondition struct {
	Cond    Condition
	MsgID   int
	AddName bool
	Msg     string
}

// SkillEffect is a resolved <effect>.
type SkillEffect struct {
	Name EffectName
	// Ticks is the period of a periodic effect, in Mobius effect ticks.
	Ticks  int
	Params EffectParams
	// Funcs are the stat functions of a Buff or Debuff (and of the other
	// effects that have them).
	Funcs []StatFunc
	// Items are the item groups of a RestorationRandom effect.
	Items []RestorationGroup
}

// StatFunc is a resolved stat function, Mobius FuncTemplate: Op applies
// Value to Stat at Order, if Cond (when not nil) holds.
type StatFunc struct {
	Stat  Stat
	Op    StatOp
	Order int
	Value float64
	Cond  *Condition
}

// EffectParam is an effect parameter.
type EffectParam struct {
	Name, Value string
}

// EffectParams are the parameters of an effect, sorted by name.
type EffectParams []EffectParam

// Get returns the value of the parameter name.
func (ps EffectParams) Get(name string) (string, bool) {
	i, ok := slices.BinarySearchFunc(ps, name, func(p EffectParam, n string) int { return strings.Compare(p.Name, n) })
	if !ok {
		return "", false
	}
	return ps[i].Value, true
}

// String returns the parameter name, or def if the effect has none.
func (ps EffectParams) String(name, def string) string {
	if v, ok := ps.Get(name); ok {
		return v
	}
	return def
}

// Int returns the parameter name as Mobius StatSet.getInt reads it, or def
// if the effect has none.
func (ps EffectParams) Int(name string, def int) (int, error) {
	v, ok := ps.Get(name)
	if !ok {
		return def, nil
	}
	n, err := parseJavaInt(v)
	if err != nil {
		return def, fmt.Errorf("parameter %s: %w", name, err)
	}
	return n, nil
}

// Float returns the parameter name as a double (Mobius getDouble), or def
// if the effect has none.
func (ps EffectParams) Float(name string, def float64) (float64, error) {
	v, ok := ps.Get(name)
	if !ok {
		return def, nil
	}
	f, err := parseJavaDouble(v)
	if err != nil {
		return def, fmt.Errorf("parameter %s: %w", name, err)
	}
	return f, nil
}

// Bool returns the parameter name as a boolean, or def if the effect has
// none.
func (ps EffectParams) Bool(name string, def bool) (bool, error) {
	v, ok := ps.Get(name)
	if !ok {
		return def, nil
	}
	b, err := parseJavaBool(v)
	if err != nil {
		return def, fmt.Errorf("parameter %s: %w", name, err)
	}
	return b, nil
}

// IsActive, IsPassive, IsToggle, IsContinuous, IsSelfContinuous and
// IsChanneling classify the operate type as Mobius SkillOperateType does.
func (o SkillOperateType) IsActive() bool { return o.Valid() && o != OperateP && o != OperateT }

// IsPassive reports a passive skill.
func (o SkillOperateType) IsPassive() bool { return o == OperateP }

// IsToggle reports a toggle skill.
func (o SkillOperateType) IsToggle() bool { return o == OperateT }

// IsContinuous reports a skill whose effects last (A2, A4, DA2).
func (o SkillOperateType) IsContinuous() bool {
	return o == OperateA2 || o == OperateA4 || o == OperateDA2
}

// IsSelfContinuous reports A3.
func (o SkillOperateType) IsSelfContinuous() bool { return o == OperateA3 }

// IsChanneling reports a channeling skill (CA1, CA5).
func (o SkillOperateType) IsChanneling() bool { return o == OperateCA1 || o == OperateCA5 }

// IsPassive reports a passive skill.
func (s *Skill) IsPassive() bool { return s.OperateType.IsPassive() }

// IsToggle reports a toggle skill.
func (s *Skill) IsToggle() bool { return s.OperateType.IsToggle() }

// IsPhysical, IsMagic, IsStatic and IsDance tell the kind of skill (Magic).
func (s *Skill) IsPhysical() bool { return s.Magic == 0 }

// IsMagic reports a magic skill.
func (s *Skill) IsMagic() bool { return s.Magic == 1 }

// IsStatic reports a static skill.
func (s *Skill) IsStatic() bool { return s.Magic == 2 }

// IsDance reports a dance or song.
func (s *Skill) IsDance() bool { return s.Magic == 3 }

// HasEffect reports whether the level has an effect of the given name in
// any scope.
func (s *Skill) HasEffect(name EffectName) bool {
	for _, effects := range s.Effects {
		for i := range effects {
			if effects[i].Name == name {
				return true
			}
		}
	}
	return false
}

// skillField sets one value of a Skill from its StatSet text.
type skillField func(s *Skill, v string) error

func intField(f func(s *Skill) *int) skillField {
	return func(s *Skill, v string) error {
		n, err := parseJavaInt(v)
		*f(s) = n
		return err
	}
}

func boolField(f func(s *Skill) *bool) skillField {
	return func(s *Skill, v string) error {
		b, err := parseJavaBool(v)
		*f(s) = b
		return err
	}
}

// floatField reads a Java float, kept with float32 precision.
func floatField(f func(s *Skill) *float64) skillField {
	return func(s *Skill, v string) error {
		x, err := parseJavaFloat(v)
		*f(s) = x
		return err
	}
}

// skillFields are the values the Mobius Skill constructor reads, by
// StatSet name.
var skillFields = map[string]skillField{
	"displayId":    intField(func(s *Skill) *int { return &s.DisplayID }),
	"displayLevel": intField(func(s *Skill) *int { return &s.DisplayLevel }),
	"icon":         func(s *Skill, v string) error { s.Icon = v; return nil },
	"operateType": func(s *Skill, v string) error {
		t, ok := ParseSkillOperateType(v)
		if !ok {
			return fmt.Errorf("unknown operate type %q", v)
		}
		s.OperateType = t
		return nil
	},
	"targetType": func(s *Skill, v string) error {
		t, ok := ParseTargetType(v)
		if !ok {
			return fmt.Errorf("unknown target type %q", v)
		}
		s.TargetType = t
		return nil
	},
	"isMagic": intField(func(s *Skill) *int { return &s.Magic }),
	"trait": func(s *Skill, v string) error {
		t, ok := ParseTraitType(v)
		if !ok {
			return fmt.Errorf("unknown trait %q", v)
		}
		s.Trait = t
		return nil
	},
	"basicProperty": func(s *Skill, v string) error {
		b, ok := parseBaseStat(v)
		if !ok {
			return fmt.Errorf("unknown basic property %q", v)
		}
		s.BasicProperty = b
		return nil
	},
	"magicLevel":       intField(func(s *Skill) *int { return &s.MagicLevel }),
	"staticReuse":      boolField(func(s *Skill) *bool { return &s.StaticReuse }),
	"mpConsume":        intField(func(s *Skill) *int { return &s.MPConsume }),
	"mpInitialConsume": intField(func(s *Skill) *int { return &s.MPInitialConsume }),
	"mpPerChanneling":  intField(func(s *Skill) *int { return &s.MPPerChanneling }),
	"hpConsume":        intField(func(s *Skill) *int { return &s.HPConsume }),
	"itemConsumeCount": intField(func(s *Skill) *int { return &s.ItemConsumeCount }),
	"itemConsumeId":    intField(func(s *Skill) *int { return &s.ItemConsumeID }),
	"chargeConsume":    intField(func(s *Skill) *int { return &s.ChargeConsume }),
	"castRange":        intField(func(s *Skill) *int { return &s.CastRange }),
	"effectRange":      intField(func(s *Skill) *int { return &s.EffectRange }),
	"affectRange":      intField(func(s *Skill) *int { return &s.AffectRange }),
	"affectLimit": func(s *Skill, v string) error {
		lo, hi, ok := strings.Cut(v, "-")
		a, err1 := parseJavaInt(lo)
		b, err2 := parseJavaInt(hi)
		if !ok || err1 != nil || err2 != nil {
			return fmt.Errorf("affect limit %q: want min-max", v)
		}
		s.AffectLimit = [2]int{a, b}
		return nil
	},
	"hitTime":         intField(func(s *Skill) *int { return &s.HitTime }),
	"coolTime":        intField(func(s *Skill) *int { return &s.CoolTime }),
	"reuseDelay":      intField(func(s *Skill) *int { return &s.ReuseDelay }),
	"abnormalLevel":   intField(func(s *Skill) *int { return &s.AbnormalLevel }),
	"abnormalTime":    intField(func(s *Skill) *int { return &s.AbnormalTime }),
	"abnormalInstant": boolField(func(s *Skill) *bool { return &s.AbnormalInstant }),
	"abnormalType": func(s *Skill, v string) error {
		t, ok := ParseAbnormalType(v)
		if !ok {
			return fmt.Errorf("unknown abnormal type %q", v)
		}
		s.AbnormalType = t
		return nil
	},
	"abnormalVisualEffect": func(s *Skill, v string) error {
		s.AbnormalVisualEffects = nil
		for _, name := range strings.Split(v, ";") {
			e, ok := ParseAbnormalVisualEffect(name)
			if !ok {
				return fmt.Errorf("unknown abnormal visual effect %q", name)
			}
			s.AbnormalVisualEffects = append(s.AbnormalVisualEffects, e)
		}
		return nil
	},
	"power":        floatField(func(s *Skill) *float64 { return &s.Power }),
	"pvpPower":     floatField(func(s *Skill) *float64 { return &s.PvPPower }),
	"pvePower":     floatField(func(s *Skill) *float64 { return &s.PvEPower }),
	"activateRate": intField(func(s *Skill) *int { return &s.ActivateRate }),
	"lvlBonusRate": intField(func(s *Skill) *int { return &s.LvlBonusRate }),
	"minChance":    intField(func(s *Skill) *int { return &s.MinChance }),
	"maxChance":    intField(func(s *Skill) *int { return &s.MaxChance }),
	"blowChance":   intField(func(s *Skill) *int { return &s.BlowChance }),
	"baseCritRate": intField(func(s *Skill) *int { return &s.BaseCritRate }),
	"element": func(s *Skill, v string) error {
		n, err := strconv.ParseInt(v, 10, 8) // Java Byte.parseByte
		if err != nil {
			return fmt.Errorf("%q is not a byte", v)
		}
		s.Element = int(n)
		return nil
	},
	"elementPower":                 intField(func(s *Skill) *int { return &s.ElementPower }),
	"attribute":                    func(s *Skill, v string) error { s.Attribute = v; return nil },
	"effectPoint":                  intField(func(s *Skill) *int { return &s.EffectPoint }),
	"isDebuff":                     boolField(func(s *Skill) *bool { return &s.IsDebuff }),
	"isRecoveryHerb":               boolField(func(s *Skill) *bool { return &s.IsRecoveryHerb }),
	"ignoreShld":                   boolField(func(s *Skill) *bool { return &s.IgnoreShield }),
	"nextActionAttack":             boolField(func(s *Skill) *bool { return &s.NextActionAttack }),
	"removedOnAnyActionExceptMove": boolField(func(s *Skill) *bool { return &s.RemovedOnAnyActionExceptMove }),
	"removedOnDamage":              boolField(func(s *Skill) *bool { return &s.RemovedOnDamage }),
	"blockedInOlympiad":            boolField(func(s *Skill) *bool { return &s.BlockedInOlympiad }),
	"overHit":                      boolField(func(s *Skill) *bool { return &s.OverHit }),
	"isSuicideAttack":              boolField(func(s *Skill) *bool { return &s.SuicideAttack }),
	"dmgDirectlyToHp":              boolField(func(s *Skill) *bool { return &s.DirectHPDamage }),
	"stayAfterDeath":               boolField(func(s *Skill) *bool { return &s.StayAfterDeath }),
	"stayOnSubclassChange":         boolField(func(s *Skill) *bool { return &s.StayOnSubclassChange }),
	"canBeDispeled":                boolField(func(s *Skill) *bool { return &s.CanBeDispeled }),
	"excludedFromCheck":            boolField(func(s *Skill) *bool { return &s.ExcludedFromCheck }),
	"simultaneousCast":             boolField(func(s *Skill) *bool { return &s.SimultaneousCast }),
	"isPvPOnly":                    boolField(func(s *Skill) *bool { return &s.PvPOnly }),
	"feed":                         intField(func(s *Skill) *int { return &s.Feed }),
	"minPledgeClass":               intField(func(s *Skill) *int { return &s.MinPledgeClass }),
	"channelingSkillId":            intField(func(s *Skill) *int { return &s.ChannelingSkillID }),
	"channelingTickInterval":       intField(func(s *Skill) *int { return &s.ChannelingTickInterval }),
	"channelingTickInitialDelay":   intField(func(s *Skill) *int { return &s.ChannelingTickInitialDelay }),
}

// SkillValueNames returns the names of the skill values Mobius reads (the
// keys of SkillTemplate.Values), sorted.
func SkillValueNames() []string { return sortedNames(skillFields) }

// newSkill builds a level from its resolved values, with the defaults of
// the Mobius Skill constructor. operateType is required.
func newSkill(id, level int, name string, vals map[string]string) (Skill, error) {
	s := Skill{
		ID: id, Level: level, DisplayID: id, DisplayLevel: level, Name: name,
		Icon:          "icon.skill0000",
		TargetType:    TargetSelf,
		Trait:         TraitNone,
		BasicProperty: BaseNone,
		AbnormalType:  AbnormalNone,
		CastRange:     -1, EffectRange: -1, ActivateRate: -1,
		MinChance: -1, MaxChance: -1, Element: -1,
		StayOnSubclassChange: true, CanBeDispeled: true,
		ChannelingTickInterval: 2,
	}
	if _, ok := vals["operateType"]; !ok {
		return s, errors.New("no operateType")
	}
	for _, n := range sortedNames(vals) {
		set, ok := skillFields[n]
		if !ok {
			return s, fmt.Errorf("unknown value %q", n)
		}
		if err := set(&s, vals[n]); err != nil {
			return s, fmt.Errorf("%s: %w", n, err)
		}
	}
	if _, ok := vals["mpPerChanneling"]; !ok {
		s.MPPerChanneling = s.MPConsume
	}
	if _, ok := vals["pvpPower"]; !ok {
		s.PvPPower = s.Power
	}
	if _, ok := vals["pvePower"]; !ok {
		s.PvEPower = s.Power
	}
	if _, ok := vals["channelingTickInitialDelay"]; !ok {
		s.ChannelingTickInitialDelay = s.ChannelingTickInterval
	}
	s.ChannelingTickInterval *= 1000
	s.ChannelingTickInitialDelay *= 1000
	return s, nil
}

// parseBaseStat returns the BaseStat of a Mobius BaseStat name.
func parseBaseStat(name string) (BaseStat, bool) {
	for b := BaseSTR; b <= BaseMEN; b++ {
		if b.String() == name {
			return b, true
		}
	}
	return BaseNone, name == "NONE"
}

// tableIndex resolves #table references of a template at one table index:
// Mobius's "current level" while it parses a level.
type tableIndex struct {
	t   *SkillTemplate
	idx int
}

// value returns raw, or its table's entry if it is a #table reference.
func (ti tableIndex) value(raw string) (string, error) {
	if !strings.HasPrefix(raw, "#") {
		return raw, nil
	}
	col, ok := ti.t.Tables[raw]
	if !ok {
		return "", fmt.Errorf("table %s does not exist", raw)
	}
	if ti.idx < 0 || ti.idx >= len(col) {
		return "", fmt.Errorf("table %s has %d values, entry %d is needed", raw, len(col), ti.idx+1)
	}
	return col[ti.idx], nil
}

// usesTable reports whether a raw value is a #table reference.
func usesTable(raw string) bool { return strings.HasPrefix(raw, "#") }

// blockKey names an effect block of a template: its scope, and the route
// it comes from (0 for the skill's own).
type blockKey struct {
	scope EffectScope
	route int
}

// resolver resolves the levels of one template, sharing the effect blocks
// that do not depend on the level between them.
type resolver struct {
	t      *SkillTemplate
	shared map[blockKey][]SkillEffect
	// names and routeNames are the value names of the skill and of each
	// route, sorted, so a problem is always reported for the same value.
	names      []string
	routeNames [][]string
}

// resolve builds every level of t into t.levels: the normal levels, then
// the enchant routes' levels. It fails on the first level that does not
// resolve: a missing table or entry, a value that does not parse, an
// unknown name.
func (t *SkillTemplate) resolve() error {
	n := t.Levels
	for _, r := range t.Enchants {
		if r.Levels < 0 {
			return fmt.Errorf("enchant route with %d levels", r.Levels)
		}
		n += r.Levels
	}
	if t.Effects[ScopePassive] != nil {
		return errors.New("effects: passive effects are the GENERAL block of a passive skill")
	}
	rv := resolver{t: t, shared: make(map[blockKey][]SkillEffect), names: sortedNames(t.Values)}
	for _, route := range t.Enchants {
		rv.routeNames = append(rv.routeNames, sortedNames(route.Values))
	}
	t.levels = make([]Skill, 0, n)
	for level := 1; level <= t.Levels; level++ {
		s, err := rv.level(level, -1, level-1)
		if err != nil {
			return fmt.Errorf("level %d: %w", level, err)
		}
		t.levels = append(t.levels, s)
	}
	for r, route := range t.Enchants {
		if route.Effects[ScopePassive] != nil {
			return fmt.Errorf("enchant route %d: passive effects are the GENERAL block of a passive skill", r+1)
		}
		for i := range route.Levels {
			level := enchantRouteBase[r] + i
			s, err := rv.level(level, r, i)
			if err != nil {
				return fmt.Errorf("level %d: %w", level, err)
			}
			t.levels = append(t.levels, s)
		}
	}
	return nil
}

// level resolves one level: a normal level (route -1, idx the level's table
// index) or an enchant level of route r (idx its index in the route).
func (rv *resolver) level(level, r, idx int) (Skill, error) {
	t := rv.t
	own := tableIndex{t, idx}
	skill := own // the table index of the skill's own values and blocks
	var route *EnchantRoute
	if r >= 0 {
		route = &t.Enchants[r]
		skill = tableIndex{t, t.Levels - 1}
	}
	vals := make(map[string]string, len(t.Values)+4)
	for _, name := range rv.names {
		v, err := skill.value(t.Values[name])
		if err != nil {
			return Skill{}, fmt.Errorf("value %s: %w", name, err)
		}
		vals[name] = v
	}
	if route != nil {
		for _, name := range rv.routeNames[r] {
			v, err := own.value(route.Values[name])
			if err != nil {
				return Skill{}, fmt.Errorf("enchant route %d value %s: %w", r+1, name, err)
			}
			vals[name] = v
		}
	}
	s, err := newSkill(t.ID, level, t.Name, vals)
	if err != nil {
		return s, err
	}

	conds, condIdx := t.Conditions, skill
	if route != nil && route.Conditions != nil {
		conds, condIdx = route.Conditions, own
	}
	for i := range conds {
		c, err := conds[i].Cond.resolve(&condIdx)
		if err != nil {
			return s, fmt.Errorf("conditions %d: %w", i+1, err)
		}
		s.Conditions = append(s.Conditions, SkillCondition{Cond: c, MsgID: conds[i].MsgID, AddName: conds[i].AddName, Msg: conds[i].Msg})
	}

	for scope := range EffectScope(NumEffectScopes) {
		defs, ti, key := t.Effects[scope], skill, blockKey{scope, 0}
		present := defs != nil
		if route != nil {
			if rd, ok := route.Effects[scope]; ok {
				defs, ti, key, present = rd, own, blockKey{scope, r + 1}, true
			} else if r == 1 && scope == ScopeStart {
				// Mobius DocumentSkill does not give route 2 the
				// skill's <startEffects>.
				present = false
			}
		}
		if !present || len(defs) == 0 {
			continue
		}
		effects, err := rv.effects(key, defs, ti, level)
		if err != nil {
			return s, fmt.Errorf("%s effects: %w", scope, err)
		}
		if scope == ScopeGeneral && s.IsPassive() {
			scope = ScopePassive
		}
		s.Effects[scope] = effects
	}
	return s, nil
}

// effects resolves an effect block at a table index. A block that uses no
// table and no per-level items is resolved once and shared.
func (rv *resolver) effects(key blockKey, defs []EffectDef, ti tableIndex, level int) ([]SkillEffect, error) {
	fixed := true
	for i := range defs {
		if defs[i].dependsOnLevel() {
			fixed = false
			break
		}
	}
	if fixed {
		if effects, ok := rv.shared[key]; ok {
			return effects, nil
		}
	}
	effects := make([]SkillEffect, len(defs))
	for i := range defs {
		e, err := defs[i].resolve(ti, level)
		if err != nil {
			return nil, fmt.Errorf("effect %d (%s): %w", i+1, defs[i].Name, err)
		}
		effects[i] = e
	}
	if fixed {
		rv.shared[key] = effects
	}
	return effects, nil
}

// dependsOnLevel reports whether the effect uses a table or per-level
// items.
func (e *EffectDef) dependsOnLevel() bool {
	if usesTable(e.Name) || usesTable(e.Ticks) || (e.Items != nil && e.Items.ByLevel != nil) {
		return true
	}
	for _, v := range e.Params {
		if usesTable(v) {
			return true
		}
	}
	for _, f := range e.Funcs {
		if usesTable(f.Value) {
			return true
		}
	}
	return false
}

// resolve builds the effect at a table index for the given skill level.
func (e *EffectDef) resolve(ti tableIndex, level int) (SkillEffect, error) {
	name, err := ti.value(e.Name)
	if err != nil {
		return SkillEffect{}, fmt.Errorf("name: %w", err)
	}
	en, ok := ParseEffectName(name)
	if !ok {
		return SkillEffect{}, fmt.Errorf("unknown effect %q", name)
	}
	r := SkillEffect{Name: en}
	if e.Ticks != "" {
		v, err := ti.value(e.Ticks)
		if err != nil {
			return r, fmt.Errorf("ticks: %w", err)
		}
		if r.Ticks, err = parseJavaInt(v); err != nil {
			return r, fmt.Errorf("ticks: %w", err)
		}
	}
	if len(e.Params) > 0 {
		r.Params = make(EffectParams, 0, len(e.Params))
		for _, name := range sortedNames(e.Params) {
			v, err := ti.value(e.Params[name])
			if err != nil {
				return r, fmt.Errorf("parameter %s: %w", name, err)
			}
			r.Params = append(r.Params, EffectParam{Name: name, Value: v})
		}
	}
	for i, f := range e.Funcs {
		sf, err := f.resolve(ti)
		if err != nil {
			return r, fmt.Errorf("function %d (%s %s): %w", i+1, f.Op, f.Stat, err)
		}
		r.Funcs = append(r.Funcs, sf)
	}
	if e.Items != nil {
		if en != EffectRestorationRandom {
			return r, errors.New("items on an effect other than RestorationRandom")
		}
		groups := e.Items.All
		if e.Items.ByLevel != nil {
			if e.Items.All != nil {
				return r, errors.New("items for all levels and per level")
			}
			groups = e.Items.ByLevel[level]
		}
		for _, g := range groups {
			if len(g.Items) == 0 {
				return r, errors.New("an item group without items")
			}
			for _, it := range g.Items {
				if it.ID <= 0 || it.Count < 0 {
					return r, fmt.Errorf("item %d count %d: bad numbers", it.ID, it.Count)
				}
			}
			g.Chance = float64(float32(g.Chance))
			r.Items = append(r.Items, g)
		}
	}
	return r, nil
}

// resolve builds the stat function at a table index. Its condition never
// uses tables: Mobius parses it with the effect, not the skill, as its
// template, and fails on a table reference there.
func (f *StatFuncDef) resolve(ti tableIndex) (StatFunc, error) {
	if !f.Stat.Valid() || !f.Op.Valid() {
		return StatFunc{}, errors.New("invalid stat or function")
	}
	sf := StatFunc{Stat: f.Stat, Op: f.Op, Order: f.Order}
	if f.Value != "" {
		v, err := ti.value(f.Value)
		if err != nil {
			return sf, err
		}
		if sf.Value, err = parseJavaDouble(v); err != nil {
			return sf, err
		}
	}
	if f.Cond != nil {
		c, err := f.Cond.resolve(nil)
		if err != nil {
			return sf, fmt.Errorf("condition: %w", err)
		}
		sf.Cond = &c
	}
	return sf, nil
}

// parseJavaInt parses a decimal int as Java Integer.parseInt does.
func parseJavaInt(s string) (int, error) {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%q is not an int", s)
	}
	return int(n), nil
}

// parseJavaDouble parses a decimal number as Java Double.parseDouble does
// for the forms the data uses; NaN and infinities are refused.
func parseJavaDouble(s string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("%q is not a number", s)
	}
	return f, nil
}

// parseJavaFloat parses a number as Java Float.parseFloat does and returns
// it with float32 precision, as Mobius keeps it.
func parseJavaFloat(s string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 32)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("%q is not a number", s)
	}
	return float64(float32(f)), nil
}

// parseJavaBool parses true or false in any case. Java's
// Boolean.parseBoolean reads anything else as false; the loader refuses it.
func parseJavaBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("%q is not true or false", s)
}

// parseJavaDecode parses an int as Java Integer.decode does: an optional
// sign, then decimal digits, 0x, 0X or # and hex digits, or 0 and octal.
func parseJavaDecode(s string) (int, error) {
	t, sign := s, ""
	if t != "" && (t[0] == '-' || t[0] == '+') {
		sign, t = t[:1], t[1:]
	}
	base := 10
	switch {
	case strings.HasPrefix(t, "0x"), strings.HasPrefix(t, "0X"):
		base, t = 16, t[2:]
	case strings.HasPrefix(t, "#"):
		base, t = 16, t[1:]
	case len(t) > 1 && t[0] == '0':
		base, t = 8, t[1:]
	}
	if t == "" || t[0] == '-' || t[0] == '+' {
		return 0, fmt.Errorf("%q is not an int", s)
	}
	n, err := strconv.ParseInt(sign+t, base, 32)
	if err != nil {
		return 0, fmt.Errorf("%q is not an int", s)
	}
	return int(n), nil
}
