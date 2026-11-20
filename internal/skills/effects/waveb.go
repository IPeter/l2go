package effects

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
)

// What the Wave B handlers share: their system messages and the Mobius
// Creature methods several of them call.

// The system messages of the Wave B handlers (Mobius SystemMessageId).
const (
	msgObtainedS2S1           sp.SystemMessageID = 29   // You have obtained $s2 $s1.
	msgObtainedS1             sp.SystemMessageID = 30   // You have obtained $s1.
	msgSkillRemovedLackOfMP   sp.SystemMessageID = 140  // Your skill was removed due to a lack of MP.
	msgObtainedEnchantedS1S2  sp.SystemMessageID = 369  // You have obtained a +$s1 $s2.
	msgSkillCanceledLackOfHP  sp.SystemMessageID = 610  // Your skill has been canceled due to lack of HP.
	msgS2MPDrainedByS1        sp.SystemMessageID = 970  // $s2's MP has been drained by $s1.
	msgLethalStrike           sp.SystemMessageID = 1667 // Lethal Strike!
	msgLethalStrikeSuccessful sp.SystemMessageID = 1668 // Your lethal strike was successful!
	msgNothingFoundInside     sp.SystemMessageID = 1669 // There was nothing found inside of that.
	msgOpponentMPReducedByS1  sp.SystemMessageID = 1867 // Your opponent's MP was reduced by $s1.
)

// dispelSlot is one abnormal type a slot dispel removes, with the highest
// abnormal level it removes (negative for any).
type dispelSlot struct {
	abnormal data.AbnormalType
	level    int
}

// parseDispelSlots reads the dispel parameter, "TYPE,level;TYPE,level", as
// the Mobius constructors do into an EnumMap: an unknown type is NONE, a
// repeated type keeps its last level, and the slots go in the enum's order.
// withLevel false gives every slot the level Short.MAX_VALUE.
func parseDispelSlots(s string, withLevel bool) ([]dispelSlot, error) {
	if s == "" {
		return nil, nil
	}
	levels := make(map[data.AbnormalType]int)
	for _, part := range strings.Split(s, ";") {
		f := strings.Split(part, ",")
		t, ok := data.ParseAbnormalType(f[0])
		if !ok {
			t = data.AbnormalNone
		}
		lvl := math.MaxInt16
		if withLevel {
			if len(f) < 2 {
				return nil, fmt.Errorf("dispel %q: %q has no level", s, part)
			}
			n, err := strconv.ParseInt(f[1], 10, 16)
			if err != nil {
				return nil, fmt.Errorf("dispel %q: %w", s, err)
			}
			lvl = int(n)
		}
		levels[t] = lvl
	}
	slots := make([]dispelSlot, 0, len(levels))
	for t, lvl := range levels {
		slots = append(slots, dispelSlot{abnormal: t, level: lvl})
	}
	slices.SortFunc(slots, func(a, b dispelSlot) int { return int(a.abnormal) - int(b.abnormal) })
	return slots, nil
}

// drainMP is the tick of the MP-draining effects (ManaDamOverTime,
// SilentMove, MpConsumePerLevel): nothing for the dead; an amount over the
// MP ends the effect with "Your skill was removed due to a lack of MP." (for
// ManaDamOverTime only on a toggle, so that the MP just runs out); else the
// MP goes down (reduceCurrentMp, not below 0). A toggle keeps going.
func drainMP(info *skills.BuffInfo, amount float64, stopWhenShort bool) bool {
	effected, sk := info.Effected(), info.Skill()
	if effected.IsDead() {
		return false
	}
	if amount > effected.CurrentMP() && (stopWhenShort || sk.IsToggle()) {
		skills.Send(effected, msgSkillRemovedLackOfMP)
		return false
	}
	effected.SetCurrentMP(max(effected.CurrentMP()-amount, 0))
	return sk.IsToggle()
}

// isMonsterB reports a monster (Mobius isMonster).
func isMonsterB(c skills.Creature) bool {
	return c != nil && c.InstanceType().Is(data.InstanceMonster)
}

// startStunning is Creature.startStunning: the attack, cast and movement
// stop, the AI is told, a non-summon goes idle, and the visual effect is
// broadcast.
func startStunning(c skills.Creature) {
	c.AbortAttack()
	c.AbortCast()
	c.StopMove()
	c.NotifyAI(skills.AIStunned, nil)
	if !isSummon(c) {
		c.NotifyAI(skills.AIIdle, nil)
	}
	c.UpdateAbnormalEffect()
}

// startParalyze is Creature.startParalyze.
func startParalyze(c skills.Creature) {
	c.AbortAttack()
	c.AbortCast()
	c.StopMove()
	c.NotifyAI(skills.AIParalyzed, nil)
}

// thinkUnlessPlayer is the onExit of the control effects: an NPC's AI
// thinks again (notifyActionThink).
func thinkUnlessPlayer(c skills.Creature) {
	if !isPlayer(c) {
		c.NotifyAI(skills.AIThink, nil)
	}
}
