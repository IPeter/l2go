package skills

import (
	"cmp"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
)

// Catalog holds every skill level of the data, compiled (K1, K2). Build it
// once at startup with NewCatalog; it is then read-only and safe for
// concurrent use.
type Catalog struct {
	byID   map[int]*catalogSkill
	report Report
}

// catalogSkill is one skill: its template and its compiled levels, parallel
// to the template's AllLevels.
type catalogSkill struct {
	tmpl   *data.SkillTemplate
	levels []Skill
}

// Report is what a Catalog found while compiling: the effect names in use
// and their ports, the stubs, and the condition kinds without a port.
// Startup logs it (Report.Log).
type Report struct {
	// Skills and Levels count the compiled skills and levels.
	Skills, Levels int
	// EffectSkills counts the skills that use each effect name, in any level
	// or scope; Ported tells which of them have a port.
	EffectSkills map[data.EffectName]int
	Ported       map[data.EffectName]bool
	// StubSkills counts the skills with a level that has a stub effect.
	StubSkills int
	// RefusedLevels counts the levels that refuse to be used.
	RefusedLevels int
	// UnportedConditions counts the skills that use each condition kind
	// without a port.
	UnportedConditions map[data.ConditionKind]int
	// Problems lists effects whose parameters did not parse.
	Problems []string
}

// NewCatalog compiles every level of every skill of d with the effect
// factories of reg. Effects without a factory, or whose parameters do not
// parse, load as stubs, and a skill with one refuses to be used; so does a
// skill with a precondition of a kind without a port. Both are counted in
// the Report, never fatal.
func NewCatalog(d *data.Skills, reg *Registry) (*Catalog, error) {
	if d == nil || reg == nil {
		return nil, fmt.Errorf("skills: %w: a catalog needs the skill data and a registry", ErrInvalid)
	}
	c := &Catalog{
		byID: make(map[int]*catalogSkill, len(d.Skills)),
		report: Report{
			EffectSkills:       make(map[data.EffectName]int),
			Ported:             make(map[data.EffectName]bool),
			UnportedConditions: make(map[data.ConditionKind]int),
		},
	}
	for i := range d.Skills {
		t := &d.Skills[i]
		c.byID[t.ID] = c.compile(t, reg)
	}
	return c, nil
}

// compile compiles the levels of one skill.
func (c *Catalog) compile(t *data.SkillTemplate, reg *Registry) *catalogSkill {
	levels := t.AllLevels()
	cs := &catalogSkill{tmpl: t, levels: make([]Skill, len(levels))}
	// The data shares an effect block between the levels that do not
	// depend on the level; so do the compiled effects.
	shared := make(map[*data.SkillEffect][]Effect)
	names := make(map[data.EffectName]bool)
	kinds := make(map[data.ConditionKind]bool)
	stubbed := false
	for i := range levels {
		ds := &levels[i]
		sk := &cs.levels[i]
		sk.Skill = ds
		sk.visual = visualMasksOf(ds.AbnormalVisualEffects)
		var refusals []string
		for j := range ds.Conditions {
			pc := &ds.Conditions[j]
			un := make(map[data.ConditionKind]bool)
			cond := compileCondition(&pc.Cond, un)
			if len(un) > 0 {
				// A tree with a kind without a port fails as a whole, so a
				// <not> around the kind cannot let it pass.
				cond = never
				for k := range un {
					kinds[k] = true
					refusals = append(refusals, "condition "+k.String()+" is not ported yet")
				}
			}
			sk.preconds = append(sk.preconds, Precondition{Cond: cond, MsgID: pc.MsgID, AddName: pc.AddName, Msg: pc.Msg})
		}
		for scope := range ds.Effects {
			defs := ds.Effects[scope]
			if len(defs) == 0 {
				continue
			}
			effs, ok := shared[&defs[0]]
			if !ok {
				effs = c.compileEffects(t, defs, reg, kinds)
				shared[&defs[0]] = effs
			}
			sk.effects[scope] = effs
			for _, e := range effs {
				names[e.Name()] = true
				sk.types |= 1 << e.Type()
				if reason := StubReason(e); reason != "" {
					refusals = append(refusals, reason)
					stubbed = true
				}
				if ds.IsPassive() {
					if b := e.base(); b.passive == nil {
						b.passive = sharedModifiers(b.funcs, passiveOwner(ds.ID))
					}
				}
			}
		}
		if len(refusals) > 0 {
			slices.Sort(refusals)
			sk.refusal = strings.Join(slices.Compact(refusals), "; ")
			c.report.RefusedLevels++
		}
	}
	c.report.Skills++
	c.report.Levels += len(levels)
	for n := range names {
		c.report.EffectSkills[n]++
		c.report.Ported[n] = reg.Ported(n)
	}
	for k := range kinds {
		c.report.UnportedConditions[k]++
	}
	if stubbed {
		c.report.StubSkills++
	}
	return cs
}

// compileEffects builds the effects of a block and compiles their stat
// functions.
func (c *Catalog) compileEffects(t *data.SkillTemplate, defs []data.SkillEffect, reg *Registry, kinds map[data.ConditionKind]bool) []Effect {
	effs := make([]Effect, len(defs))
	for j := range defs {
		def := &defs[j]
		e, _, err := reg.build(def)
		if err != nil {
			c.report.Problems = append(c.report.Problems, fmt.Sprintf("skill %d (%s): %v", t.ID, t.Name, err))
		}
		b := e.base()
		for k := range def.Funcs {
			un := make(map[data.ConditionKind]bool)
			sf, err := compileFunc(&def.Funcs[k], un)
			if err != nil {
				c.report.Problems = append(c.report.Problems, fmt.Sprintf("skill %d (%s): effect %s: %v", t.ID, t.Name, def.Name, err))
				continue
			}
			if len(un) > 0 {
				sf.Cond = never // the function never applies
				for kind := range un {
					kinds[kind] = true
				}
			}
			b.funcs = append(b.funcs, sf)
		}
		effs[j] = e
	}
	return effs
}

// passiveOwner is the owner of a passive skill's modifiers: the skill ID,
// so a new level replaces the old one's (K8).
func passiveOwner(id int) stats.Owner { return stats.Owner{Kind: stats.OwnerSkill, ID: int64(id)} }

// Skill returns one level of a skill: a normal level or an enchant level
// (101-130, 141-170).
func (c *Catalog) Skill(id, level int) (*Skill, bool) {
	cs, ok := c.byID[id]
	if !ok {
		return nil, false
	}
	i := levelIndex(cs.tmpl, level)
	if i < 0 || i >= len(cs.levels) {
		return nil, false
	}
	return &cs.levels[i], true
}

// SkillOrMax returns a level of a skill as Mobius SkillData.getSkill does:
// a level above the highest normal level gives the highest one.
func (c *Catalog) SkillOrMax(id, level int) (*Skill, bool) {
	if sk, ok := c.Skill(id, level); ok {
		return sk, true
	}
	if cs, ok := c.byID[id]; ok && level > cs.tmpl.Levels {
		return c.Skill(id, cs.tmpl.Levels)
	}
	return nil, false
}

// MaxLevel returns the highest normal level of a skill, 0 if there is no
// such skill.
func (c *Catalog) MaxLevel(id int) int {
	if cs, ok := c.byID[id]; ok {
		return cs.tmpl.Levels
	}
	return 0
}

// Report returns what the catalog found while compiling.
func (c *Catalog) Report() *Report { return &c.report }

// levelIndex returns the index of level in the template's AllLevels, -1 if
// it has no such level (data.SkillTemplate.Level's rule).
func levelIndex(t *data.SkillTemplate, level int) int {
	if level >= 1 && level <= t.Levels {
		return level - 1
	}
	i := t.Levels
	for r, route := range t.Enchants {
		if base := data.EnchantLevel(r + 1); level >= base && level < base+route.Levels {
			return i + level - base
		}
		i += route.Levels
	}
	return -1
}

// Stubs returns the effect names in use without a port, the most used
// first.
func (r *Report) Stubs() []data.EffectName {
	var names []data.EffectName
	for n, ported := range r.Ported {
		if !ported {
			names = append(names, n)
		}
	}
	slices.SortFunc(names, func(a, b data.EffectName) int {
		if c := cmp.Compare(r.EffectSkills[b], r.EffectSkills[a]); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})
	return names
}

// Log writes the report: one summary line, one line per stub name and per
// condition kind without a port, and the problems.
func (r *Report) Log(log *slog.Logger) {
	if log == nil {
		log = slog.Default()
	}
	stubs := r.Stubs()
	log.Info("skills loaded", "skills", r.Skills, "levels", r.Levels,
		"effect names", len(r.EffectSkills), "stub names", len(stubs),
		"skills with stubs", r.StubSkills, "refused levels", r.RefusedLevels,
		"condition kinds without a port", len(r.UnportedConditions))
	for _, n := range stubs {
		log.Debug("effect without a port", "effect", n.String(), "skills", r.EffectSkills[n])
	}
	kinds := make([]data.ConditionKind, 0, len(r.UnportedConditions))
	for k := range r.UnportedConditions {
		kinds = append(kinds, k)
	}
	slices.Sort(kinds)
	for _, k := range kinds {
		log.Debug("condition without a port", "kind", k.String(), "skills", r.UnportedConditions[k])
	}
	for _, p := range r.Problems {
		log.Warn("skill effect problem", "problem", p)
	}
}
