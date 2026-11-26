package scripttest

import "github.com/IPeter/l2go/internal/scripting"

// The hero table of the fake world (scripting.Heroes): a test makes a
// player an unclaimed hero, scripts claim; a claimed hero is a Player with
// Hero set.

var _ scripting.Heroes = (*World)(nil)

// heroWorld is the hero state of a World.
type heroWorld struct {
	unclaimed map[int32]bool
	// claims are the object IDs of the players that claimed, in order.
	claims []int32
}

// SetUnclaimedHero makes a player a hero of the last period's end that has
// not claimed the status.
func (w *World) SetUnclaimedHero(p *Player) {
	if w.heroes.unclaimed == nil {
		w.heroes.unclaimed = make(map[int32]bool)
	}
	w.heroes.unclaimed[p.ObjectID] = true
}

// HeroClaims returns the object IDs of the players that claimed the hero
// status, in order.
func (w *World) HeroClaims() []int32 { return w.heroes.claims }

// IsHero reads Player.Hero.
func (w *World) IsHero(p scripting.PlayerRef) bool {
	pl := playerOf(p)
	return pl != nil && pl.Hero
}

// IsUnclaimedHero reports whether SetUnclaimedHero made the player one.
func (w *World) IsUnclaimedHero(p scripting.PlayerRef) bool {
	pl := playerOf(p)
	return pl != nil && w.heroes.unclaimed[pl.ObjectID]
}

// ClaimHero sets Player.Hero and records the claim.
func (w *World) ClaimHero(p scripting.PlayerRef) {
	pl := playerOf(p)
	if pl == nil {
		return
	}
	delete(w.heroes.unclaimed, pl.ObjectID)
	pl.Hero = true
	w.heroes.claims = append(w.heroes.claims, pl.ObjectID)
}
