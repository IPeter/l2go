package bbs

import (
	"slices"
	"strconv"
	"strings"
)

// The friends board's lists (Mobius FriendsBoard).

// The confirmation boxes of "Delete All" (%deleteMSG%).
const (
	FriendDeleteAllBox = "<br>\n<table><tr><td width=10></td><td>Are you sure you want to delete all friends from your Friends List?</td><td width=20></td><td><button value=\"OK\" action=\"bypass _friend;delall\" back=\"l2ui_ch3.smallbutton2_down\" width=65 height=20 fore=\"l2ui_ch3.smallbutton2\"></td><td><button value=\"Cancel\" action=\"bypass _friendlist\" back=\"l2ui_ch3.smallbutton2_down\" width=65 height=20 fore=\"l2ui_ch3.smallbutton2\"></td></tr></table>"
	BlockDeleteAllBox  = "<br>\n<table><tr><td width=10></td><td>Are you sure you want to delete all players from your Block List?</td><td width=20></td><td><button value=\"OK\" action=\"bypass _block;delall\" back=\"l2ui_ch3.smallbutton2_down\" width=65 height=20 fore=\"l2ui_ch3.smallbutton2\"></td><td><button value=\"Cancel\" action=\"bypass _friendblocklist\" back=\"l2ui_ch3.smallbutton2_down\" width=65 height=20 fore=\"l2ui_ch3.smallbutton2\"></td></tr></table>"
)

// ListEntry is a character on a friend or block list.
type ListEntry struct {
	ID     int32
	Name   string
	Online bool
}

// ListRows are the rows of a friend or block list page: kind is "_friend"
// or "_block"; with selected false the entries not in sel get a [Select]
// link, with selected true the entries of sel (in sel's order) a [Deselect]
// link (Mobius showFriendsList, showBlockList).
func ListRows(kind string, entries []ListEntry, sel []int32, selected bool) string {
	var b strings.Builder
	row := func(e ListEntry, action, label string) {
		b.WriteString(`<a action="bypass ` + kind + ";" + action + ";" + strconv.Itoa(int(e.ID)) + `">` + label + `</a>&nbsp;`)
		b.WriteString(e.Name)
		b.WriteByte(' ')
		if e.Online {
			b.WriteString("(on)")
		} else {
			b.WriteString("(off)")
		}
		b.WriteString("<br1>")
	}
	if !selected {
		for _, e := range entries {
			if !slices.Contains(sel, e.ID) {
				row(e, "select", "[Select]")
			}
		}
		return b.String()
	}
	for _, id := range sel {
		i := slices.IndexFunc(entries, func(e ListEntry) bool { return e.ID == id })
		if i >= 0 {
			row(entries[i], "deselect", "[Deselect]")
		}
	}
	return b.String()
}
