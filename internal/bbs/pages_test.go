package bbs

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func testClans(n int) []ClanInfo {
	var out []ClanInfo
	for i := range n {
		out = append(out, ClanInfo{ID: int32(100 + i), Name: "Clan" + strconv.Itoa(i), LeaderName: "Lead" + strconv.Itoa(i), Level: 2, Members: i + 1})
	}
	return out
}

func TestClanListPage(t *testing.T) {
	clans := testClans(12)
	page := ClanListPage(clans, 1, 0)
	if !strings.Contains(page, `<img src="L2UI.SquareBlank" width="0" height="0">&nbsp;&nbsp;`) || strings.Contains(page, "[GO TO MY CLAN]") {
		t.Error("a player without a clan has no link to it")
	}
	if strings.Count(page, "_bbsclan_clanhome;") != 10 || !strings.Contains(page, ">Clan9</a>") || strings.Contains(page, ">Clan10</a>") {
		t.Error("page 1 does not list the first ten clans")
	}
	if !strings.Contains(page, `<td><button action="" back="l2ui_ch3.prev1_down"`) ||
		!strings.Contains(page, `<td> 1 </td><td><a action="bypass _bbsclan_clanlist;2"> 2 </a></td>`) ||
		!strings.Contains(page, `<td><button action="bypass _bbsclan_clanlist;2" back="l2ui_ch3.next1_down"`) {
		t.Errorf("page 1 links:\n%s", page)
	}
	page = ClanListPage(clans, 2, 101)
	if !strings.Contains(page, `<a action="bypass _bbsclan_clanhome;101">[GO TO MY CLAN]</a>`) ||
		strings.Count(page, ">Clan") != 2 || strings.Count(page, ">Clan1") != 2 || // Clan10 and Clan11
		!strings.Contains(page, `<td><button action="" back="l2ui_ch3.next1_down"`) {
		t.Errorf("page 2:\n%s", page)
	}
	if !strings.HasPrefix(page, `<html><body><br><center><table><tr><td FIXWIDTH=15>&nbsp;</td>`) || !strings.HasSuffix(page, `<br><br></center></body></html>`) {
		t.Error("page frame")
	}
}

func TestClanPages(t *testing.T) {
	c := ClanInfo{ID: 7, Name: "Knights", LeaderName: "Alice", Level: 3, Members: 2, AllyName: "Order"}
	home := ClanHomePage(c, true)
	for _, want := range []string{
		`<a action="bypass _bbsclan_clanhome;7;announce">[Clan Announcement]</a>`,
		`<a action="bypass _bbsclan;mail">[Clan Mail]</a>`,
		`<a action="bypass _bbsclan_clannotice_edit;7;cnotice">[Clan Notice]</a>`,
		`CLAN NAME</td><td fixWIDTH=195 align=left>Knights</td>`,
		`CLAN LEVEL</td><td fixWIDTH=195 align=left height=16>3</td>`,
		`CLAN MEMBERS</td><td fixWIDTH=195 align=left height=16>2</td>`,
		`CLAN LEADER</td><td fixWIDTH=195 align=left height=16>Alice</td>`,
		`ALLIANCE</td><td fixWIDTH=195 align=left height=16>Order</td>`,
	} {
		if !strings.Contains(home, want) {
			t.Errorf("leader's clan home lacks %s", want)
		}
	}
	if member := ClanHomePage(c, false); strings.Contains(member, "[Clan Notice]") || strings.Contains(member, "[Clan Mail]") {
		t.Error("a member sees the leader's buttons")
	}
	edit := ClanNoticeEditPage(7, true)
	if !strings.Contains(edit, `&$227;&nbsp;&nbsp;&nbsp;/&nbsp;&nbsp;&nbsp;<a action="bypass _bbsclan_clannotice_disable">&$228;</a>`) ||
		!strings.Contains(edit, `action="Write Notice Set _ Content Content Content"`) || !strings.Contains(edit, `[Clan Notice]</a>&nbsp;&nbsp;`) {
		t.Errorf("notice edit page:\n%s", edit)
	}
	if off := ClanNoticeEditPage(7, false); !strings.Contains(off, `<a action="bypass _bbsclan_clannotice_enable">&$227;</a>`) {
		t.Error("the disabled notice has no enable link")
	}
	view := ClanNoticeViewPage(7, true, "Hello")
	if !strings.Contains(view, `cannot change the clan notice`) || !strings.Contains(view, `<td FIXWIDTH=630 align=left>Hello</td>`) {
		t.Errorf("notice view page:\n%s", view)
	}
	if strings.Contains(ClanNoticeViewPage(7, false, "Hello"), "Hello") {
		t.Error("a disabled notice is shown")
	}
	if NoticeText(`<a ACTION="BYPASS x">`) != "" || NoticeText("action only") != "action only" {
		t.Error("NoticeText")
	}
	if got := NoticeToStore(strings.Repeat("x", 9000)); len(got) != MaxNoticeLength {
		t.Errorf("NoticeToStore keeps %d", len(got))
	}
	mail := ClanMailPage(7, "Knights")
	if !strings.Contains(mail, `Knights Clan Members</td>`) || !strings.Contains(mail, `action="Write CMail mail 7 Title Title Content"`) {
		t.Errorf("clan mail page:\n%s", mail)
	}
	if got := NeedsDevelopmentPage("_bbsclan_x"); got != "<html><body><br><br><center>Command _bbsclan_x needs development.</center><br><br></body></html>" {
		t.Errorf("NeedsDevelopmentPage = %q", got)
	}
}

func TestClanSearch(t *testing.T) {
	clans := testClans(12)
	if got := SearchClans(clans, "clan name", "CLAN1"); len(got) != 3 {
		t.Errorf("name search found %d, want 3 (Clan1, Clan10, Clan11)", len(got))
	}
	if got := SearchClans(clans, "Clan Ruler", "lead2"); len(got) != 1 || got[0].ID != 102 {
		t.Errorf("leader search found %+v", got)
	}
	if SearchClans(clans, "Clan Name", "") != nil || SearchClans(clans, "Other", "Clan") != nil {
		t.Error("an empty keyword or another type finds clans")
	}
	page := ClanSearchPage(nil, 0, "Clan Ruler", "zed")
	if !strings.Contains(page, `<br><br><font>No rulers found matching the name: </font>zed<br><br>`) ||
		!strings.Contains(page, `<a action="bypass _bbsclan_clanhome;0">[GO TO MY CLAN]</a>`) ||
		!strings.Contains(page, `&nbsp;&gt; <a action="bypass _bbsclan_clanlist"> Search </a>`) {
		t.Errorf("empty search page:\n%s", page)
	}
	page = ClanSearchPage(clans[:1], 100, "Clan Name", "Clan0")
	if !strings.Contains(page, `<tr><td FIXWIDTH=5></td><td FIXWIDTH=200 align=center><a action="bypass _bbsclan_clanhome;100">Clan0</a>`) {
		t.Errorf("search page:\n%s", page)
	}
}

func TestRegion(t *testing.T) {
	for percent, want := range map[int]string{0: "0.0%", 10: "10.0%", 15: "15.0%", 25: "25.0%", 7: "7.000000000000001%", 29: "28.999999999999996%"} {
		if got := TaxText(percent); got != want {
			t.Errorf("TaxText(%d) = %q, want %q", percent, got, want)
		}
	}
	if ClanLink(0, "x") != "NPC" || ClanLink(5, "K") != `<a action="bypass _bbsclan_clanhome;5">K</a>` {
		t.Error("ClanLink")
	}
	empty := HallsList(nil)
	if !strings.HasSuffix(empty, `There are no clan halls in this region.</td></tr></table>`) {
		t.Errorf("no halls: %s", empty)
	}
	// Dion's halls 31-33 come in HashMap order: 32, 33, 31.
	list := HallsList([]Hall{{ID: 31, Name: "Atramental"}, {ID: 32, Name: "Scarlet", OwnerID: 9, OwnerName: "Knights", OwnerLeader: "Alice"}, {ID: 33, Name: "Viridian"}})
	if a, b, c := strings.Index(list, "Scarlet"), strings.Index(list, "Viridian"), strings.Index(list, "Atramental"); a >= b || b >= c {
		t.Errorf("hall order: %s", list)
	}
	if !strings.Contains(list, `<td width=200>Scarlet</td><td width=200><a action="bypass _bbsclan_clanhome;9">Knights</a></td><td width=200>Alice</td>`) ||
		!strings.Contains(list, `<td width=200>Viridian</td><td width=200>None</td><td width=200>None</td>`) {
		t.Errorf("halls: %s", list)
	}
}

func TestFavorites(t *testing.T) {
	f := NewFavorites([]Favorite{{ID: 4, Player: 1, Title: "Home", Bypass: "_bbshome", Added: mailStart}})
	added := f.Add(1, "Region", "_bbsloc", mailStart.Add(time.Second))
	if added.ID != 5 || f.Count(1) != 2 || f.Count(2) != 0 {
		t.Errorf("Add = %+v, count %d", added, f.Count(1))
	}
	f.Add(1, "Clan", "_bbsclan", mailStart.Add(time.Second))
	var titles []string
	for _, fav := range f.List(1) {
		titles = append(titles, fav.Title)
	}
	if !slices.Equal(titles, []string{"Clan", "Region", "Home"}) {
		t.Errorf("List order %q", titles)
	}
	if f.Delete(2, 4) || !f.Delete(1, 4) || f.Count(1) != 2 {
		t.Error("Delete")
	}
	if title, bypass, ok := SplitBypass(" Region> & _bbsloc;1 "); !ok || title != "Region>" || bypass != "_bbsloc;1" {
		t.Errorf("SplitBypass = %q %q %v", title, bypass, ok)
	}
	if _, _, ok := SplitBypass("x"); ok {
		t.Error("SplitBypass without &")
	}
}

func TestListRows(t *testing.T) {
	entries := []ListEntry{{ID: 1, Name: "Ann", Online: true}, {ID: 2, Name: "Bob"}, {ID: 3, Name: "Cid"}}
	if got := ListRows("_friend", entries, []int32{2}, false); got != `<a action="bypass _friend;select;1">[Select]</a>&nbsp;Ann (on)<br1><a action="bypass _friend;select;3">[Select]</a>&nbsp;Cid (off)<br1>` {
		t.Errorf("unselected rows %q", got)
	}
	if got := ListRows("_block", entries, []int32{3, 2, 9}, true); got != `<a action="bypass _block;deselect;3">[Deselect]</a>&nbsp;Cid (off)<br1><a action="bypass _block;deselect;2">[Deselect]</a>&nbsp;Bob (off)<br1>` {
		t.Errorf("selected rows %q", got)
	}
}
