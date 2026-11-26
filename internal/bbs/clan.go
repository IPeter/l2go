package bbs

import (
	"strconv"
	"strings"
)

// The clan board's pages, which Mobius builds in code (ClanBoard).

// ClanMinLevel is the clan level from which a clan has a community (Mobius
// "clan.getLevel() < 2").
const ClanMinLevel = 2

// ClansPerPage is how many clans a page of the clan list shows.
const ClansPerPage = 10

// MaxNoticeLength is the longest clan notice, in UTF-16 units; a longer one
// is cut (Mobius Clan.MAX_NOTICE_LENGTH).
const MaxNoticeLength = 8192

// ClanInfo is what the clan pages show of a clan.
type ClanInfo struct {
	ID         int32
	Name       string
	LeaderName string
	Level      int
	Members    int
	// AllyName is empty without an alliance.
	AllyName string
}

func itoa(v int) string   { return strconv.Itoa(v) }
func id32(v int32) string { return strconv.Itoa(int(v)) }

func clanHomeLink(id int32) string {
	return `<a action="bypass _bbsclan_clanhome;` + id32(id) + `">`
}

// clanRow is one clan of a list (clanList and generateSearchHtml).
func clanRow(b *strings.Builder, c ClanInfo, space string) {
	b.WriteString(`<img src="L2UI.SquareBlank" width="610" height="3"><table border=0 cellspacing=0 cellpadding=0 width=610><tr>` + space + `<td FIXWIDTH=5></td><td FIXWIDTH=200 align=center>`)
	b.WriteString(clanHomeLink(c.ID))
	b.WriteString(c.Name)
	b.WriteString(`</a></td><td FIXWIDTH=200 align=center>`)
	b.WriteString(c.LeaderName)
	b.WriteString(`</td><td FIXWIDTH=100 align=center>`)
	b.WriteString(itoa(c.Level))
	b.WriteString(`</td><td FIXWIDTH=100 align=center>`)
	b.WriteString(itoa(c.Members))
	b.WriteString(`</td><td FIXWIDTH=5></td></tr><tr><td height=5></td></tr></table><img src="L2UI.SquareBlank" width="610" height="3"><img src="L2UI.SquareGray" width="610" height="1">`)
}

// ClanListPage is the clan list (Mobius ClanBoard.clanList): page index of
// clans (all clans, in the registry's order), with a link to the viewer's
// clan when myClan is not 0.
func ClanListPage(clans []ClanInfo, index int, myClan int32) string {
	index = max(index, 1)
	var b strings.Builder
	b.WriteString(`<html><body><br><center><table><tr><td FIXWIDTH=15>&nbsp;</td><td width=630 height=30 align=left><a action="bypass _bbshome">Home</a>&nbsp;>&nbsp;&amp;$809;</td></tr></table>`)
	b.WriteString(`<table border=0 cellspacing=0 cellpadding=0 width=625 bgcolor=000000>`)
	b.WriteString(`<tr><td height=10></td></tr>`)
	b.WriteString(`<tr><td fixWIDTH=5></td><td fixWIDTH=630>`)
	if myClan != 0 {
		b.WriteString(`<a action="bypass _bbsclan_clanhome;` + id32(myClan))
		b.WriteString(`">[GO TO MY CLAN]</a>&nbsp;&nbsp;`)
	} else {
		b.WriteString(`<img src="L2UI.SquareBlank" width="0" height="0">&nbsp;&nbsp;`)
	}
	b.WriteString(`</td><td fixWIDTH=5></td></tr>`)
	b.WriteString(`<tr><td height=10></td></tr>`)
	b.WriteString(`</table><br>`)
	b.WriteString(`<table border=0 cellspacing=0 cellpadding=2 bgcolor=000000 width=625>`)
	b.WriteString(`<tr><td FIXWIDTH=5></td><td FIXWIDTH=200 align=center>CLAN NAME</td><td FIXWIDTH=200 align=center>CLAN LEADER</td>`)
	b.WriteString(`<td FIXWIDTH=100 align=center>CLAN LEVEL</td><td FIXWIDTH=100 align=center>CLAN MEMBERS</td><td FIXWIDTH=5></td></tr>`)
	b.WriteString(`</table><img src="L2UI.Squareblank" width="1" height="5">`)
	start := (index - 1) * ClansPerPage
	for i := start; i < start+ClansPerPage && i < len(clans); i++ {
		clanRow(&b, clans[i], " ")
	}
	b.WriteString(`<img src="L2UI.SquareBlank" width="610" height="2"><table cellpadding=0 cellspacing=2 border=0><tr>`)
	if index == 1 {
		b.WriteString(`<td><button action="" back="l2ui_ch3.prev1_down" fore="l2ui_ch3.prev1" width=16 height=16 ></td>`)
	} else {
		b.WriteString(`<td><button action="bypass _bbsclan_clanlist;` + itoa(index-1) + `" back="l2ui_ch3.prev1_down" fore="l2ui_ch3.prev1" width=16 height=16 ></td>`)
	}
	pages := (len(clans) + ClansPerPage - 1) / ClansPerPage
	for i := 1; i <= pages; i++ {
		if i == index {
			b.WriteString("<td> " + itoa(i) + " </td>")
		} else {
			b.WriteString(`<td><a action="bypass _bbsclan_clanlist;` + itoa(i) + `"> ` + itoa(i) + " </a></td>")
		}
	}
	if index == pages {
		b.WriteString(`<td><button action="" back="l2ui_ch3.next1_down" fore="l2ui_ch3.next1" width=16 height=16 ></td>`)
	} else {
		b.WriteString(`<td><button action="bypass _bbsclan_clanlist;` + itoa(index+1) + `" back="l2ui_ch3.next1_down" fore="l2ui_ch3.next1" width=16 height=16 ></td>`)
	}
	b.WriteString(`</tr></table><table border=0 cellspacing=0 cellpadding=0><tr><td width=610><img src="sek.cbui141" width="610" height="1"></td></tr></table><table border=0><tr><td><combobox width=75 var=keyword list="Clan Name;Clan Ruler"></td><td><edit var = "Search" width=130 height=11 length="16"></td>` +
		`<td><button value="&$420;" action="Write Search -1 0 Search keyword keyword" back="l2ui_ch3.smallbutton2_down" width=65 height=20 fore="l2ui_ch3.smallbutton2"> </td> </tr></table><br><br></center></body></html>`)
	return b.String()
}

// clanLeaderLinks are the clan home's buttons for the leader (and the
// notice page's, which ends without the notice link).
func clanLeaderLinks(b *strings.Builder, id int32) {
	b.WriteString(`<a action="bypass _bbsclan_clanhome;` + id32(id) + `;announce">[Clan Announcement]</a>&nbsp;&nbsp;`)
	b.WriteString(`<a action="bypass _bbsclan_clanhome;` + id32(id) + `;cbb">[Clan Bulletin Board]</a>&nbsp;&nbsp;`)
	b.WriteString(`<a action="bypass _bbsclan_clanhome;` + id32(id) + `;cbb">[Clan Management]</a>&nbsp;&nbsp;`)
	b.WriteString(`<a action="bypass _bbsclan;mail">[Clan Mail]</a>&nbsp;&nbsp;`)
}

// ClanHomePage is a clan's home (Mobius ClanBoard.clanHome): leader shows
// the leader's buttons (the viewer leads that clan).
func ClanHomePage(c ClanInfo, leader bool) string {
	var b strings.Builder
	b.WriteString(`<html><body><br><table><tr><td FIXWIDTH=15>&nbsp;</td><td width=630 height=30 align=left><a action="bypass _bbshome">Home</a> &nbsp;&gt; <a action="bypass _bbsclan_clanlist">&nbsp;&amp;$809;</a>&nbsp;&gt;`)
	b.WriteString(`&amp;$802; </a></td></tr></table>`)
	b.WriteString(`<table border=0 cellspacing=0 cellpadding=0 width=625 bgcolor=000000>`)
	b.WriteString(`<tr><td height=10></td></tr>`)
	b.WriteString(`<tr><td fixWIDTH=5></td><td fixwidth=630>`)
	if leader {
		clanLeaderLinks(&b, c.ID)
		b.WriteString(`<a action="bypass _bbsclan_clannotice_edit;` + id32(c.ID) + `;cnotice">[Clan Notice]</a>&nbsp;&nbsp;`)
	} else {
		b.WriteString(`<img src="L2UI.SquareBlank" width="0" height="0">&nbsp;&nbsp;`)
	}
	b.WriteString(`</td><td fixWIDTH=5></td></tr>`)
	b.WriteString(`<tr><td height=10></td></tr></table>`)
	b.WriteString(`<table border=0 cellspacing=0 cellpadding=0 width=610><tr><td height=10></td></tr>`)
	b.WriteString(`<tr><td fixWIDTH=5></td><td fixwidth=290 valign=top></td>`)
	b.WriteString(`<td fixWIDTH=5></td><td fixWIDTH=5 align=center valign=top><img src="l2ui.squaregray" width=2 height=128></td>`)
	b.WriteString(`<td fixWIDTH=5></td><td fixwidth=295><table border=0 cellspacing=0 cellpadding=0 width=295>`)
	b.WriteString(`<tr><td fixWIDTH=100 align=left>CLAN NAME</td><td fixWIDTH=195 align=left>`)
	b.WriteString(c.Name)
	b.WriteString(`</td></tr>`)
	b.WriteString(`<tr><td height=7></td></tr>`)
	b.WriteString(`<tr><td fixWIDTH=100 align=left>CLAN LEVEL</td><td fixWIDTH=195 align=left height=16>`)
	b.WriteString(itoa(c.Level))
	b.WriteString(`</td></tr>`)
	b.WriteString(`<tr><td height=7></td></tr>`)
	b.WriteString(`<tr><td fixWIDTH=100 align=left>CLAN MEMBERS</td><td fixWIDTH=195 align=left height=16>`)
	b.WriteString(itoa(c.Members))
	b.WriteString(`</td></tr>`)
	b.WriteString(`<tr><td height=7></td></tr>`)
	b.WriteString(`<tr><td fixWIDTH=100 align=left>CLAN LEADER</td><td fixWIDTH=195 align=left height=16>`)
	b.WriteString(c.LeaderName)
	b.WriteString(`</td></tr>`)
	b.WriteString(`<tr><td height=7></td></tr>`)
	b.WriteString(`<tr><td fixWIDTH=100 align=left>ALLIANCE</td><td fixWIDTH=195 align=left height=16>`)
	b.WriteString(c.AllyName)
	b.WriteString(`</td></tr></table></td><td fixWIDTH=5></td></tr>`)
	b.WriteString(`<tr><td height=10></td></tr></table>`)
	b.WriteString(`<img src="L2UI.squareblank" width="1" height="5"><img src="L2UI.squaregray" width="610" height="1"><br></center><br> <br></body></html>`)
	return b.String()
}

// clanNoticeHead is the start of the clan notice page.
func clanNoticeHead(b *strings.Builder, id int32) {
	b.WriteString(`<html><body><br><table><tr><td FIXWIDTH=15>&nbsp;</td><td width=630 height=30 align=left><a action="bypass _bbshome">Home</a>&nbsp;&gt;<a action="bypass _bbsclan_clanlist">&nbsp;&amp;$809;</a>&nbsp;&gt;<a action="bypass _bbsclan_clanhome;`)
	b.WriteString(id32(id))
	b.WriteString(`">&amp;$802;</a></td></tr></table>`)
	b.WriteString(`<table border=0 cellspacing=0 cellpadding=0 width=625 bgcolor=000000>`)
	b.WriteString(`<tr><td height=10></td></tr>`)
	b.WriteString(`<tr><td fixWIDTH=5></td><td fixWIDTH=630>`)
	clanLeaderLinks(b, id)
	b.WriteString(`[Clan Notice]</a>&nbsp;&nbsp;`)
	b.WriteString(`</td><td fixWIDTH=5></td></tr>`)
	b.WriteString(`<tr><td height=10></td></tr>`)
	b.WriteString(`</table>`)
}

// ClanNoticeEditPage is the clan notice page of the leader (Mobius
// ClanBoard.clanNotice): the notice on or off, and a multiedit field the
// server fills with the notice right after (EditPageID).
func ClanNoticeEditPage(id int32, enabled bool) string {
	var b strings.Builder
	clanNoticeHead(&b, id)
	b.WriteString(`<br><br><center><table width=610 border=0 cellspacing=0 cellpadding=0><tr><td fixwidth=610><font color="AAAAAA">The Clan Notice function allows the clan leader to send messages through a pop-up window to clan members at login.</font> </td></tr><tr><td height=20></td></tr>`)
	if enabled {
		b.WriteString(`<tr><td fixwidth=610> Clan Notice Function:&nbsp;&nbsp;&$227;&nbsp;&nbsp;&nbsp;/&nbsp;&nbsp;&nbsp;<a action="bypass _bbsclan_clannotice_disable">&$228;</a>`)
	} else {
		b.WriteString(`<tr><td fixwidth=610> Clan Notice Function:&nbsp;&nbsp;<a action="bypass _bbsclan_clannotice_enable">&$227;</a>&nbsp;&nbsp;&nbsp;/&nbsp;&nbsp;&nbsp;&$228;`)
	}
	b.WriteString(`</td></tr></table><img src="L2UI.Squaregray" width="610" height="1"><br> <br><table width=610 border=0 cellspacing=2 cellpadding=0><tr><td>Edit Notice: </td></tr><tr><td height=5></td></tr><tr><td><MultiEdit var ="Content" width=610 height=100></td></tr></table><br><table width=610 border=0 cellspacing=0 cellpadding=0><tr><td height=5></td></tr><tr><td align=center FIXWIDTH=65><button value="&$140;" action="Write Notice Set _ Content Content Content" back="l2ui_ch3.smallbutton2_down" width=65 height=20 fore="l2ui_ch3.smallbutton2" ></td><td align=center FIXWIDTH=45></td><td align=center FIXWIDTH=500></td></tr></table></center></body></html>`)
	return b.String()
}

// ClanNoticeViewPage is the clan notice page of a member who is not the
// leader: the notice, when it is on.
func ClanNoticeViewPage(id int32, enabled bool, notice string) string {
	var b strings.Builder
	clanNoticeHead(&b, id)
	b.WriteString(`<img src="L2UI.squareblank" width="1" height="10"><center><table border=0 cellspacing=0 cellpadding=0><tr><td>You are not your clan's leader, and therefore cannot change the clan notice</td></tr></table>`)
	if enabled {
		b.WriteString(`<table border=0 cellspacing=0 cellpadding=0><tr><td>The current clan notice:</td></tr><tr><td fixwidth=5></td><td FIXWIDTH=630 align=left>` + notice + `</td><td fixqqwidth=5></td></tr></table>`)
	}
	b.WriteString(`</center></body></html>`)
	return b.String()
}

// NoticeText is a stored notice as the board and the login pop-up show it
// (Mobius Clan.getNotice): empty when it holds both "action" and "bypass"
// in any case, against links in the notice.
func NoticeText(notice string) string {
	lower := strings.ToLower(notice)
	if strings.Contains(lower, "action") && strings.Contains(lower, "bypass") {
		return ""
	}
	return notice
}

// NoticeToStore is a notice as it is saved: cut to MaxNoticeLength (Mobius
// Clan.storeNotice).
func NoticeToStore(notice string) string { return Abbreviate(notice, MaxNoticeLength) }

// SearchClans returns the clans whose name ("Clan Name") or leader name
// ("Clan Ruler", either ignoring case) contains the keyword, ignoring case
// (Mobius ClanBoard.filterClans). An empty type or keyword finds none.
func SearchClans(clans []ClanInfo, searchType, keyword string) []ClanInfo {
	if searchType == "" || keyword == "" {
		return nil
	}
	byName := strings.EqualFold(searchType, "Clan Name")
	byLeader := strings.EqualFold(searchType, "Clan Ruler")
	key := strings.ToLower(keyword)
	var out []ClanInfo
	for _, c := range clans {
		if (byName && strings.Contains(strings.ToLower(c.Name), key)) ||
			(byLeader && c.LeaderName != "" && strings.Contains(strings.ToLower(c.LeaderName), key)) {
			out = append(out, c)
		}
	}
	return out
}

// ClanSearchPage is the result of a clan search (Mobius
// ClanBoard.generateSearchHtml); myClan is the viewer's clan ID, 0 for none.
func ClanSearchPage(found []ClanInfo, myClan int32, searchType, keyword string) string {
	var b strings.Builder
	b.WriteString(`<html><body><br><br><center><br1><br1><table border=0 cellspacing=0 cellpadding=0><tr><td FIXWIDTH=15>&nbsp;</td><td width=610 height=30 align=left>`)
	b.WriteString(`<a action="bypass _bbshome">Home</a> &nbsp;&gt; <a action="bypass _bbsclan_clanlist">&nbsp;&amp;$809;</a>`)
	if keyword != "" {
		b.WriteString(`&nbsp;&gt; <a action="bypass _bbsclan_clanlist"> Search </a>`)
	}
	b.WriteString(`</td></tr></table><table border=0 cellspacing=0 cellpadding=0 width=625 bgcolor=000000><tr><td height=10></td></tr><tr><td fixWIDTH=5></td><td fixWIDTH=630><a action="bypass _bbsclan_clanhome;`)
	b.WriteString(id32(myClan))
	b.WriteString(`">[GO TO MY CLAN]</a>&nbsp;&nbsp;</td><td fixWIDTH=5></td></tr><tr><td height=10></td></tr></table><br><table border=0 cellspacing=0 cellpadding=2 bgcolor=000000 width=625><tr><td FIXWIDTH=5></td><td FIXWIDTH=200 align=center>CLAN NAME</td><td FIXWIDTH=200 align=center>CLAN LEADER</td><td FIXWIDTH=100 align=center>CLAN LEVEL</td><td FIXWIDTH=100 align=center>CLAN MEMBERS</td><td FIXWIDTH=5></td></tr></table><img src="L2UI.Squareblank" width="1" height="5">`)
	if len(found) > 0 {
		for _, c := range found {
			clanRow(&b, c, "")
		}
	} else {
		b.WriteString(`<br><br><font>No `)
		if strings.EqualFold(searchType, "Clan Name") {
			b.WriteString("clans")
		} else {
			b.WriteString("rulers")
		}
		b.WriteString(` found matching the name: </font>`)
		b.WriteString(keyword)
		b.WriteString(`<br><br>`)
	}
	b.WriteString(`<img src="L2UI.SquareBlank" width="610" height="2"><table cellpadding=0 cellspacing=2 border=0><tr><td><combobox width=75 var=keyword list="Clan Name;Clan Ruler"></td><td><edit var = "Search" width=130 height=11 length="16"></td><td><button value="Search" action="Write Search -1 0 Search keyword keyword" back="l2ui_ch3.smallbutton2_down" width=65 height=20 fore="l2ui_ch3.smallbutton2"> </td></tr></table><br><br></center></body></html>`)
	return b.String()
}

// ClanMailPage is the clan mail form of a clan leader (Mobius
// ClanBoard.generateClanMailHtml).
func ClanMailPage(id int32, name string) string {
	var b strings.Builder
	b.WriteString(`<html><body><br>`)
	b.WriteString(`<table width=610>`)
	b.WriteString(`<tr>`)
	b.WriteString(`<td width=15>&nbsp;</td>`)
	b.WriteString(`<td width=595 height=30>`)
	b.WriteString(`<a action="bypass _bbshome">Home</a>&nbsp;>&nbsp;`)
	b.WriteString(`<a action="bypass _bbsclan_clanlist">&nbsp;&amp;$809;</a>&nbsp;>&nbsp;`)
	b.WriteString(`<a action="bypass _bbsclan_clanhome;`)
	b.WriteString(id32(id))
	b.WriteString(`">&$802;</a>`)
	b.WriteString(`</td>`)
	b.WriteString(`</tr>`)
	b.WriteString(`</table>`)
	b.WriteString(`<img src="L2UI.squaregray" width=610 height=1><br>`)
	b.WriteString(`<table width=610>`)
	b.WriteString(`<tr>`)
	b.WriteString(`<td align=right width=60 height=24>Recipient:</td>`)
	b.WriteString(`<td width=540 height=24>`)
	b.WriteString(name)
	b.WriteString(` Clan Members</td>`)
	b.WriteString(`</tr><tr>`)
	b.WriteString(`<td align=right height=24>&$413;:</td>`)
	b.WriteString(`<td><edit var="Title" width=540 height=13 length=128></td>`)
	b.WriteString(`</tr><tr>`)
	b.WriteString(`<td valign=top align=right height=24>&$427;:</td>`)
	b.WriteString(`<td><MultiEdit var="Content" width=540 height=313></td>`)
	b.WriteString(`</tr>`)
	b.WriteString(`</table><br>`)
	b.WriteString(`<img src="L2UI.squaregray" width=610 height=1><br>`)
	b.WriteString(`<table width=610>`)
	b.WriteString(`<tr>`)
	b.WriteString(`<td width=465 align=center></td>`)
	b.WriteString(`<td width=70 align=center>`)
	b.WriteString(`<button value="&$1078;" action="Write CMail mail ` + id32(id) + ` Title Title Content" back="l2ui_ch3.smallbutton2_down" width=65 height=20 fore="l2ui_ch3.smallbutton2"></td>`)
	b.WriteString(`<td width=70 align=center>`)
	b.WriteString(`<button value="&$141;" action="bypass _bbsclan_clanlist" back="l2ui_ch3.smallbutton2_down" width=65 height=20 fore="l2ui_ch3.smallbutton2"></td>`)
	b.WriteString(`<td width=5></td>`)
	b.WriteString(`</tr>`)
	b.WriteString(`</table>`)
	b.WriteString(`</body></html>`)
	return b.String()
}

// NeedsDevelopmentPage is the page for a clan board command Mobius does not
// handle.
func NeedsDevelopmentPage(command string) string {
	return "<html><body><br><br><center>Command " + command + " needs development.</center><br><br></body></html>"
}
