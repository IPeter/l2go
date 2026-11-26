package scripting

import (
	lua "github.com/yuin/gopher-lua"
)

// The castle side of the scripts (Plan 14 M11i): the castle an NPC belongs
// to (Mobius Npc.getCastle) with its owner, tax, treasury, siege and doors,
// the clan privileges of a player (Mobius Player.hasAccess(ClanAccess)), and
// the merchant window and the map-region shout the castle NPCs use.

// NpcCastle is the part of an NpcRef behind npc:castle() and
// npc:is_my_lord(player). An NpcRef without it belongs to no castle.
type NpcCastle interface {
	// ScriptCastle returns the castle the NPC belongs to (Mobius
	// Npc.getCastle), nil without one.
	ScriptCastle() CastleRef
}

// PlayerClanPrivileges is the part of a PlayerRef behind
// player:has_clan_privilege(name). A PlayerRef without it has none.
type PlayerClanPrivileges interface {
	// HasClanPrivilege reports whether the player holds the clan privilege
	// named as Mobius's ClanAccess ("CASTLE_VAULT", ...); ok is false for a
	// name ClanAccess does not have.
	HasClanPrivilege(name string) (has, ok bool)
}

// CastleFunction is a castle function the lord rented (Mobius
// Castle.CastleFunction). EndDay, EndMonth (1-12) and EndYear are the
// date of EndTime in the server's time zone.
type CastleFunction struct {
	Level   int
	Lease   int
	Rate    int64 // ms between two payments
	EndTime int64 // ms since the epoch

	EndDay, EndMonth, EndYear int
}

// CastleRef is a castle as the scripts see it (Mobius Castle and its
// Siege). Its methods run on the world loop, while a handler runs.
type CastleRef interface {
	CastleID() int
	CastleName() string
	// OwnerID is the owner clan's ID, 0 without an owner; OwnerClan the
	// clan, nil without one.
	OwnerID() int32
	OwnerClan() ClanRef
	TaxPercent() int
	// SetTaxPercent sets and saves the tax (Mobius setTaxPercent).
	SetTaxPercent(percent int)
	// MaxTax is the highest tax the lord may set for the Seal of Strife's
	// owner (the chamberlain's getTaxLimit).
	MaxTax() int
	Treasury() int64
	// DepositToTreasury moves amount adena from the player to the treasury,
	// saved together; false (nothing moves) when the player has less or the
	// treasury refuses it.
	DepositToTreasury(p PlayerRef, amount int64) bool
	// WithdrawFromTreasury moves amount adena from the treasury to the
	// player, saved together; false (nothing moves) when the treasury has
	// less or refuses it.
	WithdrawFromTreasury(p PlayerRef, amount int64) bool
	// SiegeDate is the next siege's start in seconds since the epoch, 0
	// when none is set.
	SiegeDate() int64
	IsSiegeInProgress() bool
	// IsRegistrationOver is Mobius Siege.isRegistrationOver.
	IsRegistrationOver() bool
	// IsZoneActive is Mobius getZone().isActive(): the siege zone is a
	// battlefield.
	IsZoneActive() bool
	// ControlTowerCount is the number of control towers standing (Mobius
	// Siege.getControlTowerCount).
	ControlTowerCount() int
	// Function returns the castle function of the type (Mobius
	// getFunction: 1 teleport, 2 HP, 3 MP, 4 exp, 5 support), false when
	// it is not rented.
	Function(kind int) (CastleFunction, bool)
	// SealOwner is Mobius SevenSigns.getSealOwner(seal) (1 avarice, 2
	// gnosis, 3 strife): 0 none, 1 Dusk, 2 Dawn.
	SealOwner(seal int) int
	// SevenSignsPeriod is Mobius SevenSigns.getCurrentPeriod: 0 recruiting,
	// 1 competition, 2 results, 3 seal validation.
	SevenSignsPeriod() int
	// OpenCloseDoor is Mobius Castle.openCloseDoor: a door of the castle
	// opens or closes when the player is of the owner clan.
	OpenCloseDoor(p PlayerRef, doorID int, open bool)
	// BanishForeigners sends everyone in the castle who is not of the owner
	// clan out (Mobius banishForeigners).
	BanishForeigners()
	// OustAllPlayers sends everyone in the castle's teleport zone to its
	// spawn (Mobius oustAllPlayers).
	OustAllPlayers()
	// ShowSiegeInfo opens the siege window for the player (Mobius
	// getSiege().listRegisterClan: SiegeInfo).
	ShowSiegeInfo(p PlayerRef)
}

// CastleNpcActions is the part of a QuestWorld behind npc:show_buy_window
// and npc:broadcast_say_to_region.
type CastleNpcActions interface {
	// ShowBuyWindow opens buylist listID of the merchant NPC (Mobius
	// Merchant.showBuyWindow(player, listId, applyTax)); applyTax only
	// changes the tax the window shows. Nothing happens for an NPC that is
	// not a merchant or a list that does not name it.
	ShowBuyWindow(npc NpcRef, p PlayerRef, listID int, applyTax bool)
	// NpcSayToMapRegion sends the NPC's text in the chat type to every
	// player in the NPC's map region (Mobius NpcSay to the players whose
	// MapRegionData.getMapRegionLocId is the NPC's).
	NpcSayToMapRegion(npc NpcRef, chatType int, text string)
}

// npcCastleOf returns the castle of n, nil without one.
func npcCastleOf(n NpcRef) CastleRef {
	if nc, ok := n.(NpcCastle); ok {
		return nc.ScriptCastle()
	}
	return nil
}

// playerCastleMethods are player:has_clan_privilege(name) and
// player:clan_id().
func playerCastleMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"has_clan_privilege": func(L *lua.LState) int {
			p := checkPlayer(L, "has_clan_privilege")
			name := L.CheckString(2)
			pp, ok := p.(PlayerClanPrivileges)
			if !ok {
				L.Push(lua.LFalse)
				return 1
			}
			has, known := pp.HasClanPrivilege(name)
			if !known {
				L.ArgError(2, "player:has_clan_privilege(): unknown privilege "+name+" (a Mobius ClanAccess name, e.g. CASTLE_VAULT)")
				return 0
			}
			L.Push(lua.LBool(has))
			return 1
		},
		"clan_id": func(L *lua.LState) int {
			id := int32(0)
			if c := playerClanOf(checkPlayer(L, "clan_id")); c != nil {
				id = c.ClanID()
			}
			L.Push(lua.LNumber(id))
			return 1
		},
	}
}

// npcCastleMethods are npc:castle(), npc:is_my_lord(player),
// npc:show_buy_window(player, list_id, apply_tax) and
// npc:broadcast_say_to_region(chat_type, text).
func (v *vm) npcCastleMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"castle": func(L *lua.LState) int {
			c := npcCastleOf(checkNpc(L, "castle"))
			if c == nil {
				L.Push(lua.LNil)
				return 1
			}
			L.Push(v.newCastle(L, c))
			return 1
		},
		"is_my_lord": func(L *lua.LState) int {
			n := checkNpc(L, "is_my_lord")
			p := checkPlayerArg(L, 2, "npc:is_my_lord")
			L.Push(lua.LBool(isMyLord(n, p)))
			return 1
		},
		"show_buy_window": func(L *lua.LState) int {
			const fn = "npc:show_buy_window"
			n := checkNpc(L, "show_buy_window")
			p := checkPlayerArg(L, 2, fn)
			list := int(checkInt(L, 3, fn))
			tax := L.Get(4) == lua.LNil || L.ToBool(4)
			worldAs[CastleNpcActions](v, L, fn).ShowBuyWindow(n, p, list, tax)
			return 0
		},
		"broadcast_say_to_region": func(L *lua.LState) int {
			const fn = "npc:broadcast_say_to_region"
			n := checkNpc(L, "broadcast_say_to_region")
			chat := int(checkInt(L, 2, fn))
			text := L.CheckString(3)
			worldAs[CastleNpcActions](v, L, fn).NpcSayToMapRegion(n, chat, text)
			return 0
		},
	}
}

// isMyLord is Mobius Npc.isMyLord: the player leads the clan that owns the
// NPC's castle.
func isMyLord(n NpcRef, p PlayerRef) bool {
	pc, ok := p.(PlayerClan)
	if !ok || !pc.IsClanLeader() {
		return false
	}
	clan := pc.ScriptClan()
	c := npcCastleOf(n)
	return clan != nil && c != nil && c.OwnerID() != 0 && c.OwnerID() == clan.ClanID()
}

// newCastle builds the Lua castle object: a table of methods, called with
// ":" (the table itself is ignored), like the clan object:
//
//	castle:id(), name(), owner_id(), owner() (a Clan or nil),
//	tax_percent(), set_tax_percent(n), max_tax(), treasury(),
//	deposit(player, n), withdraw(player, n), siege_date(),
//	is_siege_in_progress(), is_registration_over(), is_zone_active(),
//	control_tower_count(), get_function(type), seal_owner(seal),
//	seven_signs_period(), open_close_door(player, door_id, open),
//	door_upgrade_level(door_id), set_door_upgrade(door_id, level),
//	banish_foreigners(), oust_all_players(), show_siege_info(player)
func (v *vm) newCastle(L *lua.LState, c CastleRef) lua.LValue {
	t := L.NewTable()
	fn := func(name string, f lua.LGFunction) { t.RawSetString(name, L.NewFunction(f)) }
	num := func(name string, get func() int64) {
		fn(name, func(L *lua.LState) int {
			L.Push(lua.LNumber(get()))
			return 1
		})
	}
	flag := func(name string, get func() bool) {
		fn(name, func(L *lua.LState) int {
			L.Push(lua.LBool(get()))
			return 1
		})
	}
	num("id", func() int64 { return int64(c.CastleID()) })
	fn("name", func(L *lua.LState) int {
		L.Push(lua.LString(c.CastleName()))
		return 1
	})
	num("owner_id", func() int64 { return int64(c.OwnerID()) })
	fn("owner", func(L *lua.LState) int {
		if o := c.OwnerClan(); o != nil {
			L.Push(v.newClan(L, o))
		} else {
			L.Push(lua.LNil)
		}
		return 1
	})
	num("tax_percent", func() int64 { return int64(c.TaxPercent()) })
	fn("set_tax_percent", func(L *lua.LState) int {
		const name = "castle:set_tax_percent"
		p := checkInt(L, 2, name)
		if p < 0 || p > 100 {
			L.ArgError(2, name+": the tax must be 0 to 100")
			return 0
		}
		c.SetTaxPercent(int(p))
		return 0
	})
	num("max_tax", func() int64 { return int64(c.MaxTax()) })
	num("treasury", c.Treasury)
	move := func(name string, do func(PlayerRef, int64) bool) {
		fn(name, func(L *lua.LState) int {
			full := "castle:" + name
			p := checkPlayerArg(L, 2, full)
			amount := checkInt(L, 3, full)
			L.Push(lua.LBool(amount > 0 && do(p, amount)))
			return 1
		})
	}
	move("deposit", c.DepositToTreasury)
	move("withdraw", c.WithdrawFromTreasury)
	num("siege_date", c.SiegeDate)
	flag("is_siege_in_progress", c.IsSiegeInProgress)
	flag("is_registration_over", c.IsRegistrationOver)
	flag("is_zone_active", c.IsZoneActive)
	num("control_tower_count", func() int64 { return int64(c.ControlTowerCount()) })
	fn("get_function", func(L *lua.LState) int {
		f, ok := c.Function(int(checkInt(L, 2, "castle:get_function")))
		if !ok {
			L.Push(lua.LNil)
			return 1
		}
		ft := L.NewTable()
		for k, val := range map[string]int64{
			"level": int64(f.Level), "lease": int64(f.Lease), "rate": f.Rate, "end_time": f.EndTime,
			"end_day": int64(f.EndDay), "end_month": int64(f.EndMonth), "end_year": int64(f.EndYear),
		} {
			ft.RawSetString(k, lua.LNumber(val))
		}
		L.Push(ft)
		return 1
	})
	fn("seal_owner", func(L *lua.LState) int {
		L.Push(lua.LNumber(c.SealOwner(int(checkInt(L, 2, "castle:seal_owner")))))
		return 1
	})
	num("seven_signs_period", func() int64 { return int64(c.SevenSignsPeriod()) })
	fn("open_close_door", func(L *lua.LState) int {
		const name = "castle:open_close_door"
		p := checkPlayerArg(L, 2, name)
		door := int(checkInt(L, 3, name))
		c.OpenCloseDoor(p, door, L.ToBool(4))
		return 0
	})
	fn("banish_foreigners", func(L *lua.LState) int {
		c.BanishForeigners()
		return 0
	})
	fn("oust_all_players", func(L *lua.LState) int {
		c.OustAllPlayers()
		return 0
	})
	fn("show_siege_info", func(L *lua.LState) int {
		c.ShowSiegeInfo(checkPlayerArg(L, 2, "castle:show_siege_info"))
		return 0
	})
	castleDoorMethods(L, t, c) // api_castle_doors.go
	return t
}
