package npcai

import (
	"fmt"
	"math"
	"strconv"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
)

// The Skill0N_* parameter gates (AttackableAI.allowsParameterizedCast).
const (
	// maxGates is the number of Skill0N slots (MAX_PARAMETERIZED_SKILL_SLOTS).
	maxGates = 6
	// GateScale is the scale of a slot's probability (SKILL_PROBABILITY_SCALE):
	// 1,000 is 10 %.
	GateScale = 10000
)

// gate is one Skill0N slot of a template's parameters, read at load.
type gate struct {
	// skillID is the slot's Skill0N_ID; 0 for an empty slot.
	skillID int
	// probability is Skill0N_Probablity, else Skill0N_Probability, else
	// Skill0N_Prob (retail AI classes spell it three ways), out of
	// GateScale; 0 for none.
	probability int
	// checkDist is Skill0N_Check_Dist == 1; distMin and distMax are
	// Skill0N_Dist_Min (0) and Skill0N_Dist_Max (Integer.MAX_VALUE).
	checkDist        bool
	distMin, distMax int
	// highHP is Skill0N_HighHP; hpTarget is Skill0N_HPTarget (1: the
	// caster's HP, else the target's).
	highHP, hpTarget int
	// attackHint is Skill0N_AttackSplash == 1, Skill0N_MainAttack == 1 or
	// Skill0N_Target > 0: without a probability, a slot with no distance,
	// HP or attack hint never allows its skill.
	attackHint bool
}

// buildGates reads the six slots of t's parameters.
func (p *Profile) buildGates(t *data.NpcTemplate) []string {
	var warnings []string
	param := func(name string, def int) int {
		v, ok := t.Parameters.Params[name]
		if !ok {
			return def
		}
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("NPC %d: parameter %s = %q is not an int", t.ID, name, v))
			return def
		}
		return int(n)
	}
	for i := range maxGates {
		prefix := "Skill0" + strconv.Itoa(i+1) + "_"
		sk, ok := t.Parameters.Skills[prefix+"ID"]
		if !ok {
			continue
		}
		g := &p.gates[i]
		g.skillID = sk.ID
		g.probability = param(prefix+"Probablity", param(prefix+"Probability", param(prefix+"Prob", 0)))
		g.checkDist = param(prefix+"Check_Dist", 0) == 1
		g.distMin = param(prefix+"Dist_Min", 0)
		g.distMax = param(prefix+"Dist_Max", math.MaxInt32)
		g.highHP = param(prefix+"HighHP", 0)
		g.hpTarget = param(prefix+"HPTarget", 0)
		g.attackHint = param(prefix+"AttackSplash", 0) == 1 || param(prefix+"MainAttack", 0) == 1 || param(prefix+"Target", 0) > 0
	}
	return warnings
}

// gateOf returns the first slot that names the skill, nil when none does
// (findParameterizedSkillSlot).
func (p *Profile) gateOf(id int) *gate {
	for i := range p.gates {
		if p.gates[i].skillID != 0 && p.gates[i].skillID == id {
			return &p.gates[i]
		}
	}
	return nil
}

// AllowsCast is AttackableAI.allowsParameterizedCast: whether the Skill0N
// slot that names sk lets caster use it on target (nil for none). A skill
// in no slot is always allowed. Otherwise, in order:
//
//   - with a probability, Get(10000) must fall below it (one draw); without
//     one, the slot needs a distance check, a HighHP or an attack hint;
//   - with Check_Dist, the 2D distance to target, as an int, must lie in
//     [Dist_Min, Dist_Max];
//   - with HighHP, the HP percent of the caster (HPTarget 1) or the target
//     must be above it (no target: refused).
func (p *Profile) AllowsCast(r rnd.Source, caster Npc, sk *skills.Skill, target Creature) bool {
	if p == nil || sk == nil {
		return true
	}
	g := p.gateOf(sk.ID)
	if g == nil {
		return true
	}
	if g.probability == 0 {
		if !g.checkDist && g.highHP == 0 && !g.attackHint {
			return false
		}
	} else if r.Get(GateScale) >= g.probability {
		return false
	}
	if g.checkDist && target != nil {
		d := int(dist2D(caster, target))
		if d < g.distMin || d > g.distMax {
			return false
		}
	}
	if g.highHP > 0 {
		var src Creature = target
		if g.hpTarget == 1 {
			src = caster
		}
		if src == nil {
			return false
		}
		if src.CurrentHP()*100.0/float64(src.MaxHP()) <= float64(g.highHP) {
			return false
		}
	}
	return true
}
