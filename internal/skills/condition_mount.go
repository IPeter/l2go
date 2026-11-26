package skills

import "github.com/IPeter/l2go/internal/data"

// PlayerMount is the part of a Player behind ConditionPlayerIsRidingStrider.
// A Player without it rides nothing.
type PlayerMount interface {
	// RidingStrider reports a player on a strider (Mobius getMountType() ==
	// STRIDER).
	RidingStrider() bool
}

func init() {
	// ConditionPlayerIsRidingStrider: a player whose riding a strider is the
	// value (Strider Siege Assault).
	leafConditions[data.CondPlayerIsridingstrider] = func(c *data.Condition) Cond {
		want := c.Bool
		return func(cc ConditionContext) bool {
			p := asPlayer(cc.Caster)
			if p == nil {
				return false
			}
			m, ok := p.(PlayerMount)
			return (ok && m.RidingStrider()) == want
		}
	}
}
