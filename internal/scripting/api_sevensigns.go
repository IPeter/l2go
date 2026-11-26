package scripting

import lua "github.com/yuin/gopher-lua"

// The sevensigns module (Plan 14, M11l): Mobius SevenSigns for the priests'
// dialogs (SignsPriest, DawnPriest, DuskPriest) and the castle
// chamberlain's Dawn ticket sale, with the item helpers of Mobius Player
// those dialogs use.

// SevenSignsConfig is the part of the Seven Signs config the dialogs read
// (Mobius FeatureConfig).
type SevenSignsConfig struct {
	MaxPlayerContrib   int
	JoinDawnFee        int
	ManorsAgreementID  int
	DawnTicketQuantity int
	DawnTicketPrice    int
	DawnTicketBundle   int
	CastleForDawn      bool
	CastleForDusk      bool
}

// SevenSigns is the Seven Signs of the server (Mobius SevenSigns). Cabals
// are 0 none, 1 Dusk, 2 Dawn; seals 1 avarice, 2 gnosis, 3 strife; periods
// 0 recruiting, 1 competition, 2 results, 3 seal validation.
type SevenSigns interface {
	SevenSignsPeriod() int
	SevenSignsCycle() int
	SevenSignsSealOwner(seal int) int
	// SevenSignsHighestCabal is getCabalHighestScore.
	SevenSignsHighestCabal() int
	SevenSignsPlayerCabal(p PlayerRef) int
	SevenSignsPlayerSeal(p PlayerRef) int
	// SevenSignsContribution is getPlayerContribScore.
	SevenSignsContribution(p PlayerRef) int
	// SevenSignsJoin is setPlayerInfo, saved.
	SevenSignsJoin(p PlayerRef, cabal, seal int)
	// SevenSignsContribute is addPlayerStoneContrib, saved: the score the
	// stones are worth, -1 above the limit.
	SevenSignsContribute(p PlayerRef, blue, green, red int) int
	// SevenSignsTakeReward is getAncientAdenaReward(id, true), saved.
	SevenSignsTakeReward(p PlayerRef) int
	SevenSignsConfig() SevenSignsConfig
	// SevenSignsDestroyItem is Player.destroyItemByItemId.
	SevenSignsDestroyItem(p PlayerRef, itemID int, count int64, sendMessage bool) bool
	// SevenSignsAddItem is Player.addItem (addAdena, addAncientAdena).
	SevenSignsAddItem(p PlayerRef, itemID int, count int64, sendMessage bool) bool
	// SevenSignsReduceAdena is Player.reduceAdena(..., true).
	SevenSignsReduceAdena(p PlayerRef, count int64) bool
	// SevenSignsSendDisappeared sends "$s2 $s1 has disappeared." with the
	// item's name and the count, as the priests do after taking stones.
	SevenSignsSendDisappeared(p PlayerRef, itemID int, count int64)
	// SevenSignsHasRoom is the inventory's validateCapacity(slots).
	SevenSignsHasRoom(p PlayerRef, slots int) bool
	// SevenSignsClanCastle is the castle ID of the player's clan, 0 for none.
	SevenSignsClanCastle(p PlayerRef) int
	// SevenSignsTicketCount and SevenSignsSetTicketCount are a castle's
	// Dawn ticket sales (Castle.getTicketBuyCount, setTicketBuyCount, saved).
	SevenSignsTicketCount(castleID int) int
	SevenSignsSetTicketCount(castleID, n int)
}

// sevenSignsFuncs is the sevensigns module.
func (v *vm) sevenSignsFuncs() map[string]lua.LGFunction {
	ss := func(L *lua.LState, fn string) SevenSigns { return worldAs[SevenSigns](v, L, "sevensigns."+fn) }
	num := func(name string, f func(w SevenSigns) int) lua.LGFunction {
		return func(L *lua.LState) int {
			L.Push(lua.LNumber(f(ss(L, name))))
			return 1
		}
	}
	playerNum := func(name string, f func(w SevenSigns, p PlayerRef) int) lua.LGFunction {
		return func(L *lua.LState) int {
			p := checkPlayerArg(L, 1, "sevensigns."+name)
			L.Push(lua.LNumber(f(ss(L, name), p)))
			return 1
		}
	}
	return map[string]lua.LGFunction{
		"period":        num("period", func(w SevenSigns) int { return w.SevenSignsPeriod() }),
		"cycle":         num("cycle", func(w SevenSigns) int { return w.SevenSignsCycle() }),
		"highest_cabal": num("highest_cabal", func(w SevenSigns) int { return w.SevenSignsHighestCabal() }),
		"seal_owner": func(L *lua.LState) int {
			seal := int(checkInt(L, 1, "sevensigns.seal_owner"))
			L.Push(lua.LNumber(ss(L, "seal_owner").SevenSignsSealOwner(seal)))
			return 1
		},
		"cabal":        playerNum("cabal", func(w SevenSigns, p PlayerRef) int { return w.SevenSignsPlayerCabal(p) }),
		"seal":         playerNum("seal", func(w SevenSigns, p PlayerRef) int { return w.SevenSignsPlayerSeal(p) }),
		"contribution": playerNum("contribution", func(w SevenSigns, p PlayerRef) int { return w.SevenSignsContribution(p) }),
		"take_reward":  playerNum("take_reward", func(w SevenSigns, p PlayerRef) int { return w.SevenSignsTakeReward(p) }),
		"clan_castle":  playerNum("clan_castle", func(w SevenSigns, p PlayerRef) int { return w.SevenSignsClanCastle(p) }),
		"join": func(L *lua.LState) int {
			p := checkPlayerArg(L, 1, "sevensigns.join")
			cabal := int(checkInt(L, 2, "sevensigns.join"))
			seal := int(checkInt(L, 3, "sevensigns.join"))
			ss(L, "join").SevenSignsJoin(p, cabal, seal)
			return 0
		},
		"contribute": func(L *lua.LState) int {
			p := checkPlayerArg(L, 1, "sevensigns.contribute")
			blue := int(checkInt(L, 2, "sevensigns.contribute"))
			green := int(checkInt(L, 3, "sevensigns.contribute"))
			red := int(checkInt(L, 4, "sevensigns.contribute"))
			L.Push(lua.LNumber(ss(L, "contribute").SevenSignsContribute(p, blue, green, red)))
			return 1
		},
		"config": func(L *lua.LState) int {
			c := ss(L, "config").SevenSignsConfig()
			t := L.NewTable()
			t.RawSetString("max_player_contrib", lua.LNumber(c.MaxPlayerContrib))
			t.RawSetString("join_dawn_fee", lua.LNumber(c.JoinDawnFee))
			t.RawSetString("manors_agreement_id", lua.LNumber(c.ManorsAgreementID))
			t.RawSetString("dawn_ticket_quantity", lua.LNumber(c.DawnTicketQuantity))
			t.RawSetString("dawn_ticket_price", lua.LNumber(c.DawnTicketPrice))
			t.RawSetString("dawn_ticket_bundle", lua.LNumber(c.DawnTicketBundle))
			t.RawSetString("castle_for_dawn", lua.LBool(c.CastleForDawn))
			t.RawSetString("castle_for_dusk", lua.LBool(c.CastleForDusk))
			L.Push(t)
			return 1
		},
		"destroy_item": func(L *lua.LState) int {
			p := checkPlayerArg(L, 1, "sevensigns.destroy_item")
			id := int(checkInt(L, 2, "sevensigns.destroy_item"))
			n := checkInt(L, 3, "sevensigns.destroy_item")
			L.Push(lua.LBool(ss(L, "destroy_item").SevenSignsDestroyItem(p, id, n, L.ToBool(4))))
			return 1
		},
		"add_item": func(L *lua.LState) int {
			p := checkPlayerArg(L, 1, "sevensigns.add_item")
			id := int(checkInt(L, 2, "sevensigns.add_item"))
			n := checkInt(L, 3, "sevensigns.add_item")
			L.Push(lua.LBool(ss(L, "add_item").SevenSignsAddItem(p, id, n, L.ToBool(4))))
			return 1
		},
		"reduce_adena": func(L *lua.LState) int {
			p := checkPlayerArg(L, 1, "sevensigns.reduce_adena")
			n := checkInt(L, 2, "sevensigns.reduce_adena")
			L.Push(lua.LBool(ss(L, "reduce_adena").SevenSignsReduceAdena(p, n)))
			return 1
		},
		"send_disappeared": func(L *lua.LState) int {
			p := checkPlayerArg(L, 1, "sevensigns.send_disappeared")
			id := int(checkInt(L, 2, "sevensigns.send_disappeared"))
			n := checkInt(L, 3, "sevensigns.send_disappeared")
			ss(L, "send_disappeared").SevenSignsSendDisappeared(p, id, n)
			return 0
		},
		"has_room": func(L *lua.LState) int {
			p := checkPlayerArg(L, 1, "sevensigns.has_room")
			n := int(checkInt(L, 2, "sevensigns.has_room"))
			L.Push(lua.LBool(ss(L, "has_room").SevenSignsHasRoom(p, n)))
			return 1
		},
		"ticket_count": func(L *lua.LState) int {
			id := int(checkInt(L, 1, "sevensigns.ticket_count"))
			L.Push(lua.LNumber(ss(L, "ticket_count").SevenSignsTicketCount(id)))
			return 1
		},
		"set_ticket_count": func(L *lua.LState) int {
			id := int(checkInt(L, 1, "sevensigns.set_ticket_count"))
			n := int(checkInt(L, 2, "sevensigns.set_ticket_count"))
			ss(L, "set_ticket_count").SevenSignsSetTicketCount(id, n)
			return 0
		},
	}
}
