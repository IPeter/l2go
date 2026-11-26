package scripting

import lua "github.com/yuin/gopher-lua"

// PlayerHenna is the part of a PlayerRef behind the symbol maker's windows
// (M11c): Mobius scripts send HennaEquipList and HennaRemoveList with
// player.sendPacket. A PlayerRef without it cannot open them (the methods
// raise an error).
type PlayerHenna interface {
	// ShowHennaEquipList opens the "draw a symbol" window (Mobius
	// HennaEquipList).
	ShowHennaEquipList()
	// ShowHennaRemoveList opens the "delete a symbol" window (Mobius
	// HennaRemoveList).
	ShowHennaRemoveList()
}

// playerHennaMethods are player:show_henna_equip_list() and
// player:show_henna_remove_list().
func playerHennaMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"show_henna_equip_list": func(L *lua.LState) int {
			playerAs[PlayerHenna](L, "show_henna_equip_list", "symbol windows").ShowHennaEquipList()
			return 0
		},
		"show_henna_remove_list": func(L *lua.LState) int {
			playerAs[PlayerHenna](L, "show_henna_remove_list", "symbol windows").ShowHennaRemoveList()
			return 0
		},
	}
}
