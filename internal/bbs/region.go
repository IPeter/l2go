package bbs

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
)

// The region board (Mobius RegionBoard): one region per castle.

// RegionNames are the client string IDs of the region names, in castle
// order (Mobius RegionBoard.REGIONS): the list shows "&$<id>;".
var RegionNames = [9]int{1049, 1052, 1053, 1057, 1060, 1059, 1248, 1247, 1056}

// SiegeDateFormat is how the region page shows the next siege (Mobius
// "dd-MM-yyyy HH:mm").
const SiegeDateFormat = "02-01-2006 15:04"

// TaxText is a castle's tax as the region pages show it: the Java double
// taxPercent / 100.0 * 100 and "%" ("0.0%", "7.000000000000001%").
func TaxText(percent int) string {
	v := float64(percent) / 100.0 * 100
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.ContainsAny(s, ".") {
		s += ".0"
	}
	return s + "%"
}

// ClanLink is the region pages' link to a clan's home, or "NPC" without an
// owner (ownerID 0).
func ClanLink(ownerID int32, name string) string {
	if ownerID == 0 {
		return "NPC"
	}
	return `<a action="bypass _bbsclan_clanhome;` + strconv.Itoa(int(ownerID)) + `">` + name + "</a>"
}

// Hall is a clan hall the region page lists.
type Hall struct {
	ID   int32
	Name string
	// OwnerID is the owning clan, 0 for none; OwnerName and OwnerLeader are
	// that clan's name and leader name.
	OwnerID                int32
	OwnerName, OwnerLeader string
}

// HallsList is the clan hall table of a region page (Mobius
// RegionBoard's %hallsList%). Mobius keeps the halls of a region in a
// java.util.HashMap by ID and lists them in its order, which for the
// default 16 buckets and these small IDs is by ID modulo 16; so does
// HallsList.
func HallsList(halls []Hall) string {
	var b strings.Builder
	b.WriteString(`<br><br><table width=625 bgcolor=000000><tr><td width=5></td><td width=200>Clan Hall Name</td><td width=200>Owning Clan</td><td width=200>Clan Leader Name</td><td width=5></td></tr></table>`)
	if len(halls) == 0 {
		b.WriteString(`<br><table width=610><tr><td align="center">There are no clan halls in this region.</td></tr></table>`)
		return b.String()
	}
	ordered := slices.Clone(halls)
	slices.SortStableFunc(ordered, func(a, b Hall) int { return cmp.Compare(hashSlot(a.ID), hashSlot(b.ID)) })
	for _, h := range ordered {
		owner, leader := "None", "None"
		if h.OwnerID != 0 {
			owner = `<a action="bypass _bbsclan_clanhome;` + strconv.Itoa(int(h.OwnerID)) + `">` + h.OwnerName + "</a>"
			leader = h.OwnerLeader
		}
		b.WriteString(`<table><tr><td width=5></td><td width=200>` + h.Name + `</td><td width=200>` + owner + `</td><td width=200>` + leader + `</td><td width=5></td></tr></table><br1><img src="L2UI.Squaregray" width=630 height=1><br1>`)
	}
	return b.String()
}

// hashSlot is the bucket of an Integer key in a java.util.HashMap of 16
// buckets.
func hashSlot(id int32) uint32 {
	h := uint32(id) //nolint:gosec // the bits of the Java int
	return (h ^ h>>16) & 15
}
