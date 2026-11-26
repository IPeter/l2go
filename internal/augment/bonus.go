package augment

import (
	"fmt"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
)

// optionOwnerBase puts the owners of option modifiers above every item
// object ID (an int32) and the other item-kind owners.
const optionOwnerBase = int64(1) << 40

// OptionOwner returns the owner of an option's stat modifiers. Mobius uses
// the shared Options object as the owner (removeStatsOwner(this)), so two
// worn items with the same option lose both their modifiers when one of
// them comes off; the owner per option ID keeps that.
func OptionOwner(optionID int) stats.Owner {
	return stats.Owner{Kind: stats.OwnerItem, ID: optionOwnerBase + int64(optionID)}
}

// Modifiers returns the stat modifiers of an option (Options.getStatFuncs
// with FuncTemplate's orders), owned by OptionOwner.
func Modifiers(o *data.AugmentOption) ([]stats.Modifier, error) {
	mods := make([]stats.Modifier, 0, len(o.Funcs))
	for _, f := range o.Funcs {
		st, ok := stats.ParseStat(f.Stat.String())
		if !ok {
			return nil, fmt.Errorf("augment: option %d: stat %s is not supported", o.ID, f.Stat)
		}
		op, ok := stats.ParseOp(f.Op.String())
		if !ok {
			return nil, fmt.Errorf("augment: option %d: operation %s is not supported", o.ID, f.Op)
		}
		mods = append(mods, stats.Modifier{Stat: st, Op: op, Order: f.Order, Value: f.Value, Owner: OptionOwner(o.ID)})
	}
	return mods, nil
}

// Trigger kinds of TriggersOnHit and TriggersOnSkill.
const (
	triggerAttack   = data.AugmentTriggerAttack
	triggerMagic    = data.AugmentTriggerMagic
	triggerCritical = data.AugmentTriggerCritical
)

// FiresOnHit reports whether a trigger is tried on an attack hit
// (Creature.onHitTimer): attack skills on a normal hit, critical skills on
// a critical one.
func FiresOnHit(t data.AugmentTrigger, crit bool) bool {
	return (!crit && t.Type == triggerAttack) || (t.Type == triggerCritical && crit)
}

// FiresOnSkill reports whether a trigger is tried when a skill lands
// (Creature.callSkill, for skills that are not static): magic skills on a
// magic skill, attack skills on a physical one.
func FiresOnSkill(t data.AugmentTrigger, magic, physical bool) bool {
	return (magic && t.Type == triggerMagic) || (physical && t.Type == triggerAttack)
}

// Rolled reports whether a trigger fires: Rnd.get(100) < chance. Call it
// only after FiresOnHit or FiresOnSkill, as Mobius draws only then.
func Rolled(r Rand, t data.AugmentTrigger) bool { return float64(r.Get(100)) < t.Chance }
