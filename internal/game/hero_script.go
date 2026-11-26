package game

import "github.com/IPeter/l2go/internal/scripting"

// The hero module of the scripts (scripting.Heroes): Mobius Hero for the
// Monument of Heroes.

var _ scripting.Heroes = (*ScriptOutput)(nil)

// IsHero is Mobius Hero.isHero.
func (o *ScriptOutput) IsHero(p scripting.PlayerRef) bool {
	s, pl, ok := o.player(p)
	return ok && s.olympiad.heroes.IsHero(pl.ObjectID())
}

// IsUnclaimedHero is Mobius Hero.isUnclaimedHero.
func (o *ScriptOutput) IsUnclaimedHero(p scripting.PlayerRef) bool {
	s, pl, ok := o.player(p)
	return ok && s.olympiad.heroes.IsUnclaimed(pl.ObjectID())
}

// ClaimHero is Mobius Hero.claimHero.
func (o *ScriptOutput) ClaimHero(p scripting.PlayerRef) {
	if s, pl, ok := o.player(p); ok {
		s.claimHero(pl)
	}
}
