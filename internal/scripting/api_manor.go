package scripting

import lua "github.com/yuin/gopher-lua"

// The manor module (Plan 14, M11k): what the manor managers' and the
// chamberlains' scripts ask of Mobius CastleManorManager. The manor's
// windows (the manor_menu_select links) are the game's.

// Manor is the castle manor of the server.
type Manor interface {
	// ManorEnabled is Mobius GeneralConfig.ALLOW_MANOR.
	ManorEnabled() bool
	// ManorID is the NPC's manor_id template parameter, -1 without one.
	ManorID(npc NpcRef) int
	// IsManorLord is ManorManager.onFirstTalk's test: the player is no GM,
	// leads its clan, and the clan owns the castle of the NPC's manor_id.
	IsManorLord(npc NpcRef, p PlayerRef) bool
	// SeedIncome is what the castle's seeds of the running period brought
	// so far (the chamberlain's vault page).
	SeedIncome(castleID int) int64
}

// manorNpcArg returns argument n, an Npc.
func manorNpcArg(L *lua.LState, n int, fn string) NpcRef {
	npc := optNpcArg(L, n, fn)
	if npc == nil {
		L.ArgError(n, fn+": an NPC expected, got nil")
	}
	return npc
}

// manorFuncs is the manor module.
func (v *vm) manorFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"is_enabled": func(L *lua.LState) int {
			L.Push(lua.LBool(worldAs[Manor](v, L, "manor.is_enabled").ManorEnabled()))
			return 1
		},
		"manor_id": func(L *lua.LState) int {
			n := manorNpcArg(L, 1, "manor.manor_id")
			L.Push(lua.LNumber(worldAs[Manor](v, L, "manor.manor_id").ManorID(n)))
			return 1
		},
		"is_lord": func(L *lua.LState) int {
			n := manorNpcArg(L, 1, "manor.is_lord")
			p := checkPlayerArg(L, 2, "manor.is_lord")
			L.Push(lua.LBool(worldAs[Manor](v, L, "manor.is_lord").IsManorLord(n, p)))
			return 1
		},
		"seed_income": func(L *lua.LState) int {
			id := int(checkInt(L, 1, "manor.seed_income"))
			L.Push(lua.LNumber(worldAs[Manor](v, L, "manor.seed_income").SeedIncome(id)))
			return 1
		},
	}
}
