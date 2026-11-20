package main

import (
	"cmp"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// The passives domain extracts the stat changes of passive skills (Plan 09
// decision F10), loaded by data.LoadPassives.
func init() { registerDomain("passives", genPassives) }

// skippedCondFunc names a Buff stat function with a condition in the report
// of skipped effects.
const skippedCondFunc = "Buff with a conditional function"

// genPassives adds skills/passives.json, one skill per line: the passive
// skills NPC templates name in their <skills>, plus Weight Penalty (4270)
// and Death Penalty (5076). It reports the effects it leaves out.
func genPassives(o options) error {
	quiet := o
	quiet.log = io.Discard
	templates, err := readSkills(quiet)
	if err != nil {
		return fmt.Errorf("skills: %w", err)
	}
	skills, err := data.NewSkills(data.SkillsData{Skills: templates})
	if err != nil {
		return fmt.Errorf("skills: %w", err)
	}
	npcs, err := readNpcs(quiet)
	if err != nil {
		return fmt.Errorf("NPCs: %w", err)
	}
	passives, err := buildPassives(skills, npcs)
	if err != nil {
		return fmt.Errorf("passives: %w", err)
	}
	if _, err := data.NewPassives(passives); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.PassivesFile, passives); err != nil {
		return err
	}

	levels := 0
	skipped := make(map[string][]int) // effect -> skill IDs
	for _, p := range passives {
		levels += len(p.Levels)
		for _, name := range p.Skipped {
			skipped[name] = append(skipped[name], p.ID)
		}
	}
	fmt.Fprintf(o.log, "passives: %d passive skills with %d levels\n", len(passives), levels)
	if len(skipped) == 0 {
		fmt.Fprintln(o.log, "passives: no effects skipped")
	}
	for _, name := range slices.Sorted(maps.Keys(skipped)) {
		ids := skipped[name]
		fmt.Fprintf(o.log, "passives: skipped %s in %d skills: %s\n", name, len(ids), joinInts(ids, 30))
	}
	return nil
}

// buildPassives returns the passive skills NPCs name, plus 4270 and 5076,
// sorted by ID, with every normal level. Skills NPCs name that do not exist
// are left out (the skills domain warns about them), and so are the ones
// that are not passive or have no effects.
func buildPassives(skills *data.Skills, npcs []data.NpcTemplate) ([]data.PassiveSkill, error) {
	ids := map[int]bool{data.SkillWeightPenalty: true, data.SkillDeathPenalty: true}
	for id := range ids {
		t, ok := skills.Template(id)
		if !ok {
			return nil, fmt.Errorf("skill %d does not exist", id)
		}
		if !t.AllLevels()[0].IsPassive() {
			return nil, fmt.Errorf("skill %d (%q) is not passive", id, t.Name)
		}
	}
	for i := range npcs {
		for _, sk := range npcs[i].Skills {
			if t, ok := skills.Template(sk.ID); ok && t.AllLevels()[0].IsPassive() {
				ids[sk.ID] = true
			}
		}
	}
	var out []data.PassiveSkill
	for _, id := range slices.Sorted(maps.Keys(ids)) {
		t, _ := skills.Template(id)
		p, err := passiveSkill(t)
		if err != nil {
			return nil, fmt.Errorf("skill %d (%q): %w", id, t.Name, err)
		}
		if passiveEmpty(p) {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// passiveEmpty reports whether a passive skill has no effects at all, such
// as the markers whose effects Mobius does not implement (Greater Resist
// Bow Attack); they are left out.
func passiveEmpty(p data.PassiveSkill) bool {
	if len(p.Skipped) > 0 {
		return false
	}
	for _, l := range p.Levels {
		if len(l.Funcs)+len(l.DefenceTraits)+len(l.AttackTraits) > 0 {
			return false
		}
	}
	return true
}

// passiveSkill extracts the normal levels of a passive skill: the <effects>
// block, which Mobius applies when the skill is added (the PASSIVE scope).
// The other blocks of a passive skill are never applied.
func passiveSkill(t *data.SkillTemplate) (data.PassiveSkill, error) {
	p := data.PassiveSkill{ID: t.ID, Name: t.Name}
	skipped := make(map[string]bool)
	for level := 1; level <= t.Levels; level++ {
		sk, _ := t.Level(level)
		if !sk.IsPassive() {
			return p, fmt.Errorf("level %d is not passive", level)
		}
		var pl data.PassiveLevel
		for _, e := range sk.Effects[data.ScopePassive] {
			switch e.Name {
			case data.EffectBuff:
				for _, f := range e.Funcs {
					if f.Cond != nil {
						skipped[skippedCondFunc] = true
						continue
					}
					pl.Funcs = append(pl.Funcs, data.PassiveFunc{Stat: f.Stat, Op: f.Op, Order: f.Order, Value: f.Value})
				}
			case data.EffectDefenceTrait, data.EffectAttackTrait:
				defence := e.Name == data.EffectDefenceTrait
				traits, err := passiveTraits(e.Params, defence)
				if err != nil {
					return p, fmt.Errorf("level %d: %s: %w", level, e.Name, err)
				}
				if defence {
					pl.DefenceTraits = addTraits(pl.DefenceTraits, traits)
				} else {
					pl.AttackTraits = addTraits(pl.AttackTraits, traits)
				}
			default:
				skipped[e.Name.String()] = true
			}
		}
		p.Levels = append(p.Levels, pl)
	}
	p.Skipped = slices.Sorted(maps.Keys(skipped))
	return p, nil
}

// passiveTraits reads the parameters of a DefenceTrait or AttackTrait
// effect, as their Mobius constructors do: each parameter is a trait name
// and a Java float. DefenceTrait leaves out a value of 0. A name or value
// Mobius cannot read fails.
func passiveTraits(params data.EffectParams, defence bool) ([]data.PassiveTrait, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("no parameters")
	}
	var traits []data.PassiveTrait
	for _, prm := range params {
		tt, ok := data.ParseTraitType(prm.Name)
		if !ok {
			return nil, fmt.Errorf("unknown trait %q", prm.Name)
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(prm.Value), 32)
		if err != nil {
			return nil, fmt.Errorf("trait %s: %q is not a float", prm.Name, prm.Value)
		}
		if defence && v == 0 {
			continue
		}
		traits = append(traits, data.PassiveTrait{Trait: tt, Value: float32(v)})
	}
	return traits, nil
}

// addTraits adds the traits of another effect of the same kind on one
// level, keeping them sorted by trait. Mobius applies each effect, so a
// trait two effects name is kept twice.
func addTraits(have, add []data.PassiveTrait) []data.PassiveTrait {
	all := append(have, add...)
	slices.SortStableFunc(all, func(a, b data.PassiveTrait) int { return cmp.Compare(a.Trait, b.Trait) })
	return all
}
