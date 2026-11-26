package scripting

import lua "github.com/yuin/gopher-lua"

// The hero module (Plan 14, M11h): Mobius Hero's isHero, isUnclaimedHero
// and claimHero, which the Monument of Heroes script asks.

// Heroes is the hero table of the server (Mobius Hero).
type Heroes interface {
	// IsHero is Mobius Hero.isHero: p is a hero of the running cycle that
	// claimed the status.
	IsHero(p PlayerRef) bool
	// IsUnclaimedHero is Mobius Hero.isUnclaimedHero: p became a hero at
	// the last period's end and has not claimed the status.
	IsUnclaimedHero(p PlayerRef) bool
	// ClaimHero is Mobius Hero.claimHero: p takes the hero status (the
	// skills, the aura, the clan reputation, the animation).
	ClaimHero(p PlayerRef)
}

// heroFuncs is the hero module.
func (v *vm) heroFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"is_hero": func(L *lua.LState) int {
			p := checkPlayerArg(L, 1, "hero.is_hero")
			L.Push(lua.LBool(worldAs[Heroes](v, L, "hero.is_hero").IsHero(p)))
			return 1
		},
		"is_unclaimed": func(L *lua.LState) int {
			p := checkPlayerArg(L, 1, "hero.is_unclaimed")
			L.Push(lua.LBool(worldAs[Heroes](v, L, "hero.is_unclaimed").IsUnclaimedHero(p)))
			return 1
		},
		"claim": func(L *lua.LState) int {
			p := checkPlayerArg(L, 1, "hero.claim")
			worldAs[Heroes](v, L, "hero.claim").ClaimHero(p)
			return 0
		},
	}
}
