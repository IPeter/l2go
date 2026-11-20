package npcai

import (
	"fmt"
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// Scope is an AI skill list, Mobius AISkillScope.
type Scope uint8

// The scopes, in Mobius order.
const (
	ScopeBuff Scope = iota
	ScopeDebuff
	ScopeNegative
	ScopeAttack
	ScopeImmobilize
	ScopeHeal
	ScopeRes
	// ScopeCOT holds the skills Mobius files as "COT" (debuffs, mute, fear).
	ScopeCOT
	ScopeUniversal
	ScopeLongRange
	ScopeShortRange
	ScopeGeneral
	ScopeSuicide
	NumScopes
)

var scopeNames = [NumScopes]string{
	"BUFF", "DEBUFF", "NEGATIVE", "ATTACK", "IMMOBILIZE", "HEAL", "RES", "COT",
	"UNIVERSAL", "LONG_RANGE", "SHORT_RANGE", "GENERAL", "SUICIDE",
}

// String returns the Mobius name of the scope.
func (s Scope) String() string {
	if s < NumScopes {
		return scopeNames[s]
	}
	return fmt.Sprintf("Scope(%d)", uint8(s))
}

// AIType is the template's AI type, Mobius AIType.
type AIType uint8

// The AI types.
const (
	AIFighter AIType = iota
	AIArcher
	AIBalanced
	AIMage
	AIHealer
	AICorpse
)

var aiTypes = map[string]AIType{
	"FIGHTER": AIFighter, "ARCHER": AIArcher, "BALANCED": AIBalanced,
	"MAGE": AIMage, "HEALER": AIHealer, "CORPSE": AICorpse,
}

// ShortRange is the cast range up to which a skill is a short range skill
// (NpcData), and the distance within which thinkAttack uses one.
const ShortRange = 150

// genericClan is the clan every clan helps (NpcData.getGenericClanId).
const genericClan = "ALL"

// Profile is what the AI uses of an NPC template: the AI skill lists, the
// Skill0N_* gates, the skill chances, the AI type and the clans. Build one
// per template at load with NewProfile; it is then read-only and safe for
// concurrent use.
type Profile struct {
	// NpcID is the template's ID.
	NpcID int
	// AIType is the template's AI type.
	AIType AIType
	// MinSkillChance and MaxSkillChance bound hasSkillChance (7 and 15
	// unless the template says otherwise).
	MinSkillChance, MaxSkillChance int
	// Chaos is the template's isChaos.
	Chaos bool
	// Clans are the template's clans; nil when it has none.
	Clans []string

	lists [NumScopes][]*skills.Skill
	gates [maxGates]gate
}

// SkillSource finds a skill level as Mobius SkillData.getSkill does: a level
// above the highest gives the highest. *skills.Catalog is one.
type SkillSource interface {
	SkillOrMax(id, level int) (*skills.Skill, bool)
}

// NewProfile builds the profile of t: its AI skill lists from the skills of
// its skill list (NpcData lines 610-708) and its parameter gates. A skill
// that src does not have is left out, as Mobius leaves it out, and named in
// the warnings; so is a gate parameter that is not a number (it then keeps
// its default).
func NewProfile(t *data.NpcTemplate, src SkillSource) (*Profile, []string) {
	var warnings []string
	p := &Profile{
		NpcID:          t.ID,
		AIType:         aiTypes[t.AI.Type],
		MinSkillChance: t.AI.MinSkillChance,
		MaxSkillChance: t.AI.MaxSkillChance,
		Chaos:          t.AI.IsChaos,
	}
	if len(t.AI.Clans) > 0 {
		p.Clans = slices.Clone(t.AI.Clans)
	}
	// Mobius puts the skills in a HashMap<Integer, Skill> (a later level
	// of the same ID replaces the earlier one in place) and builds the
	// lists while iterating it.
	byID := make(map[int]*skills.Skill, len(t.Skills))
	ids := make([]int, 0, len(t.Skills))
	for _, ns := range t.Skills {
		sk, ok := src.SkillOrMax(ns.ID, ns.Level)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("NPC %d: skill %d level %d not found", t.ID, ns.ID, ns.Level))
			continue
		}
		byID[sk.ID] = sk
		ids = append(ids, sk.ID)
	}
	for _, id := range javaHashMapOrder(ids) {
		sk := byID[id]
		if sk.IsPassive() {
			continue
		}
		for _, s := range ScopesOf(sk) {
			p.lists[s] = append(p.lists[s], sk)
		}
	}
	warnings = append(warnings, p.buildGates(t)...)
	return p, warnings
}

// Skills returns a skill list. The slice must not be changed.
func (p *Profile) Skills(s Scope) []*skills.Skill {
	if p == nil || s >= NumScopes {
		return nil
	}
	return p.lists[s]
}

// IsClan is Mobius NpcTemplate.isClan(Set): the profile has clans and is in
// the generic clan ALL or shares one of clans.
func (p *Profile) IsClan(clans []string) bool {
	if len(p.Clans) == 0 || len(clans) == 0 {
		return false
	}
	if slices.Contains(p.Clans, genericClan) {
		return true
	}
	for _, c := range clans {
		if slices.Contains(p.Clans, c) {
			return true
		}
	}
	return false
}

// ScopesOf returns the AI skill lists an active skill joins (NpcData): a
// suicide skill joins SUICIDE only; any other skill joins GENERAL and the
// lists of the first of these that it is:
//
//   - continuous, not a debuff: BUFF;
//   - continuous debuff: DEBUFF, COT and short or long range;
//   - DISPEL or DISPEL_BY_SLOT: NEGATIVE and short or long range;
//   - HEAL: HEAL;
//   - a damage type (PHYSICAL_ATTACK, PHYSICAL_ATTACK_HP_LINK,
//     MAGICAL_ATTACK, DEATH_LINK, HP_DRAIN): ATTACK, UNIVERSAL and short or
//     long range;
//   - SLEEP: IMMOBILIZE;
//   - STUN or ROOT: IMMOBILIZE and short or long range;
//   - MUTE or FEAR: COT and short or long range;
//   - PARALYZE: IMMOBILIZE and short or long range;
//   - DMG_OVER_TIME or DMG_OVER_TIME_PERCENT: short or long range;
//   - RESURRECTION: RES;
//   - anything else: UNIVERSAL.
//
// Short range means a cast range of at most 150 (no cast range, -1,
// included).
func ScopesOf(sk *skills.Skill) []Scope {
	if sk.SuicideAttack {
		return []Scope{ScopeSuicide}
	}
	rng := ScopeLongRange
	if sk.CastRange <= ShortRange {
		rng = ScopeShortRange
	}
	out := []Scope{ScopeGeneral}
	switch {
	case sk.IsContinuous() && !sk.IsDebuff:
		out = append(out, ScopeBuff)
	case sk.IsContinuous():
		out = append(out, ScopeDebuff, ScopeCOT, rng)
	case sk.HasEffectType(skills.TypeDispel, skills.TypeDispelBySlot):
		out = append(out, ScopeNegative, rng)
	case sk.HasEffectType(skills.TypeHeal):
		out = append(out, ScopeHeal)
	case sk.HasEffectType(skills.TypePhysicalAttack, skills.TypePhysicalAttackHPLink, skills.TypeMagicalAttack,
		skills.TypeDeathLink, skills.TypeHPDrain):
		out = append(out, ScopeAttack, ScopeUniversal, rng)
	case sk.HasEffectType(skills.TypeSleep):
		out = append(out, ScopeImmobilize)
	case sk.HasEffectType(skills.TypeStun, skills.TypeRoot):
		out = append(out, ScopeImmobilize, rng)
	case sk.HasEffectType(skills.TypeMute, skills.TypeFear):
		out = append(out, ScopeCOT, rng)
	case sk.HasEffectType(skills.TypeParalyze):
		out = append(out, ScopeImmobilize, rng)
	case sk.HasEffectType(skills.TypeDmgOverTime, skills.TypeDmgOverTimePercent):
		out = append(out, rng)
	case sk.HasEffectType(skills.TypeResurrection):
		out = append(out, ScopeRes)
	default:
		out = append(out, ScopeUniversal)
	}
	return out
}

// Java HashMap constants.
const (
	hashMapInitialCapacity = 16
	hashMapTreeifyBin      = 8
	hashMapMinTreeify      = 64
)

// javaHashMapOrder returns the distinct keys of a java.util.HashMap<Integer,
// V> filled with keys in order, in the order the map iterates them: by
// bucket (the spread hash masked by the final capacity), and in each bucket
// by first insertion. It follows the map's growth: the table doubles when
// the size passes 3/4 of the capacity, and when a bucket of 8 grows while
// the capacity is below 64 (treeifyBin resizes instead). It does not follow
// a bucket that turns into a tree (8 keys in one bucket of a table of 64 or
// more), which no NPC skill list comes near.
func javaHashMapOrder(keys []int) []int {
	capacity := hashMapInitialCapacity
	threshold := capacity * 3 / 4
	out := make([]int, 0, len(keys))
	for _, k := range keys {
		if slices.Contains(out, k) {
			continue
		}
		bucket := spread(k) & (capacity - 1)
		n := 0
		for _, o := range out {
			if spread(o)&(capacity-1) == bucket {
				n++
			}
		}
		out = append(out, k)
		if n >= hashMapTreeifyBin && capacity < hashMapMinTreeify {
			capacity *= 2
			threshold *= 2
		}
		if len(out) > threshold {
			capacity *= 2
			threshold *= 2
		}
	}
	mask := capacity - 1
	slices.SortStableFunc(out, func(a, b int) int { return spread(a)&mask - spread(b)&mask })
	return out
}

// spread is HashMap.hash of an Integer: h ^ (h >>> 16).
func spread(k int) int {
	h := uint32(int32(k)) //nolint:gosec // Java's int hash code
	return int(h ^ h>>16)
}
