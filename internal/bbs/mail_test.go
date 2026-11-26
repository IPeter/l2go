package bbs

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

var mailStart = time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)

func TestMailTypes(t *testing.T) {
	for _, tt := range []struct {
		t                         MailType
		name, location, desc, byp string
	}{
		{Inbox, "INBOX", "inbox", "Inbox", `<a action="bypass _bbsmail">Inbox</a>`},
		{SentBox, "SENTBOX", "sentbox", "Sent Box", `<a action="bypass _bbsmail;sentbox">Sent Box</a>`},
		{Archive, "ARCHIVE", "archive", "Mail Archive", `<a action="bypass _bbsmail;archive">Mail Archive</a>`},
		{TempArchive, "TEMPARCHIVE", "temparchive", "Temporary Mail Archive", `<a action="bypass _bbsmail;temp_archive">Temporary Mail Archive</a>`},
	} {
		if tt.t.String() != tt.name || tt.t.Location() != tt.location || tt.t.Description() != tt.desc || tt.t.Bypass() != tt.byp {
			t.Errorf("%d: %q %q %q %q", tt.t, tt.t.String(), tt.t.Location(), tt.t.Description(), tt.t.Bypass())
		}
		if got, ok := ParseMailType(strings.ToLower(tt.name)); !ok || got != tt.t {
			t.Errorf("ParseMailType(%q) = %v, %v", strings.ToLower(tt.name), got, ok)
		}
	}
	if _, ok := ParseMailType("temp_archive"); ok {
		t.Error("temp_archive is no folder name (Mobius's link to it does nothing)")
	}
	if got, ok := MailTypeAt(1); !ok || got != SentBox {
		t.Error("MailTypeAt(1)")
	}
	if _, ok := MailTypeAt(4); ok {
		t.Error("MailTypeAt(4)")
	}
}

// fill gives owner n inbox mails from "Writer<i>", one minute apart, the
// newest last.
func fill(m *Mailboxes, owner int32, n int) {
	for i := range n {
		m.Add(&Mail{ID: m.NextID(), Receiver: owner, Sender: 9, SenderName: "Writer" + strconv.Itoa(i),
			Type: Inbox, Subject: "Subject " + strconv.Itoa(i), Sent: mailStart.Add(time.Duration(i) * time.Minute), Unread: i%2 == 0})
	}
}

func TestMailboxes(t *testing.T) {
	m := NewMailboxes([]Mail{
		{ID: 7, Receiver: 1, Type: Inbox, Sent: mailStart},
		{ID: 3, Receiver: 1, Type: SentBox, Sent: mailStart.Add(time.Hour)},
		{ID: 5, Receiver: 2, Type: Archive, Sent: mailStart},
	})
	if id := m.NextID(); id != 8 {
		t.Errorf("NextID = %d, want 8 (after the highest loaded)", id)
	}
	if ids := mailIDs(m.Box(1)); !slices.Equal(ids, []int32{3, 7}) {
		t.Errorf("box order %v, want newest first", ids)
	}
	if m.Get(1, 5) != nil || m.Get(2, 5) == nil {
		t.Error("Get looks into the owner's box only")
	}
	if m.HasUnread(1) {
		t.Error("HasUnread with no unread mail")
	}
	m.Add(&Mail{ID: 8, Receiver: 1, Type: Inbox, Sent: mailStart.Add(-time.Hour), Unread: true})
	if !m.HasUnread(1) {
		t.Error("no unread mail")
	}
	if !m.Delete(1, 8) || m.Delete(1, 8) || m.HasUnread(1) {
		t.Error("Delete")
	}
	if n := m.SentSince(1, mailStart); n != 1 {
		t.Errorf("SentSince = %d, want 1", n)
	}
	if n := m.SentSince(1, mailStart.Add(time.Hour)); n != 0 {
		t.Errorf("SentSince(at the sending) = %d, want 0 (after, not at)", n)
	}
}

func mailIDs(ms []*Mail) []int32 {
	var ids []int32
	for _, m := range ms {
		ids = append(ids, m.ID)
	}
	return ids
}

func TestMailList(t *testing.T) {
	m := NewMailboxes(nil)
	fill(m, 1, 25)
	lp := m.List(1, Inbox, Search{}, 0)
	if lp.Page != 1 || lp.MaxPage != 3 || len(lp.Mails) != 10 || lp.Mails[0].SenderName != "Writer24" {
		t.Errorf("page 0: page %d of %d, %d mails, first %q", lp.Page, lp.MaxPage, len(lp.Mails), lp.Mails[0].SenderName)
	}
	lp = m.List(1, Inbox, Search{}, 9)
	if lp.Page != 3 || len(lp.Mails) != 5 || lp.Mails[4].SenderName != "Writer0" {
		t.Errorf("page 9: page %d, %d mails", lp.Page, len(lp.Mails))
	}
	if lp := m.List(1, SentBox, Search{}, 1); lp.MaxPage != 1 || len(lp.Mails) != 0 {
		t.Errorf("empty folder: %+v", lp)
	}
	// Searches ignore case; "Title" searches subjects, anything else senders.
	if n := m.Count(1, Inbox, "title", "SUBJECT 1"); n != 11 { // 1, 10-19
		t.Errorf("title search count %d, want 11", n)
	}
	if n := m.Count(1, Inbox, "Writer", "writer2"); n != 6 { // 2, 20-24
		t.Errorf("writer search count %d, want 6", n)
	}
	if n := m.Count(1, Inbox, "", "x"); n != 25 {
		t.Errorf("a search without a target counts all: %d", n)
	}
	lp = m.List(1, Inbox, Search{Target: "Title", Keyword: "subject 2"}, 1)
	if len(lp.Mails) != 6 || lp.MaxPage != 1 {
		t.Errorf("search page: %d mails of %d pages", len(lp.Mails), lp.MaxPage)
	}
	if m.InboxFull(1) {
		t.Error("25 mails fill the inbox")
	}
	fill(m, 1, 75)
	if !m.InboxFull(1) {
		t.Error("100 mails do not fill the inbox")
	}
}

func TestMailRows(t *testing.T) {
	m := NewMailboxes(nil)
	m.Add(&Mail{ID: 4, Receiver: 1, SenderName: "Bob", Type: Inbox, Subject: strings.Repeat("s", 35), Sent: mailStart, Unread: true})
	m.Add(&Mail{ID: 2, Receiver: 1, SenderName: "Ann", Type: Inbox, Subject: "Hi", Sent: mailStart.Add(-time.Hour)})
	rows := m.List(1, Inbox, Search{}, 1).Rows(func(t time.Time) string { return t.Format(DateFormat) })
	want := `<table width=610><tr><td width=5></td><td width=150>Bob</td><td width=300><a action="bypass _bbsmail;view;4"><font color="LEVEL">` + strings.Repeat("s", 30) +
		`</font></a></td><td width=150>2026-11-26 12:00</td><td width=5></td></tr></table><img src="L2UI.Squaregray" width=610 height=1>` +
		`<table width=610><tr><td width=5></td><td width=150>Ann</td><td width=300><a action="bypass _bbsmail;view;2">Hi</a></td><td width=150>2026-11-26 11:00</td><td width=5></td></tr></table><img src="L2UI.Squaregray" width=610 height=1>`
	if rows != want {
		t.Errorf("Rows =\n%s\nwant\n%s", rows, want)
	}
}

func TestMailPager(t *testing.T) {
	prev := func(page int) string {
		return `<td><table><tr><td></td></tr><tr><td><button action="bypass _bbsmail;INBOX;` + strconv.Itoa(page) + `" back="l2ui_ch3.prev1_down" fore="l2ui_ch3.prev1" width=16 height=16></td></tr></table></td>`
	}
	next := func(page int) string {
		return `<td><table><tr><td></td></tr><tr><td><button action="bypass _bbsmail;INBOX;` + strconv.Itoa(page) + `" back="l2ui_ch3.next1_down" fore="l2ui_ch3.next1" width=16 height=16></td></tr></table></td>`
	}
	got := ListPage{Type: Inbox, Page: 1, MaxPage: 2}.Pager()
	want := prev(1) + `<td> 1 </td><td><a action="bypass _bbsmail;INBOX;2"> 2 </a></td>` + next(2)
	if got != want {
		t.Errorf("Pager =\n%s\nwant\n%s", got, want)
	}
	// With a search the links carry it.
	got = ListPage{Type: SentBox, Page: 2, MaxPage: 2, Search: Search{Target: "Title", Keyword: "x"}}.Pager()
	if !strings.Contains(got, `"bypass _bbsmail;SENTBOX;1;Title;x"`) || !strings.Contains(got, `<td> 2 </td>`) {
		t.Errorf("search Pager = %s", got)
	}
	// More than 21 pages: 10 around the page.
	count := func(lp ListPage) (first, last int) {
		first, last = -1, -1
		for i := 1; i <= lp.MaxPage; i++ {
			s := strconv.Itoa(i)
			if strings.Contains(lp.Pager(), "> "+s+" </") || strings.Contains(lp.Pager(), "<td> "+s+" </td>") {
				if first < 0 {
					first = i
				}
				last = i
			}
		}
		return first, last
	}
	for _, tt := range []struct{ page, first, last int }{{5, 1, 15}, {11, 1, 21}, {12, 2, 22}, {25, 15, 30}, {30, 20, 30}} {
		if f, l := count(ListPage{Type: Inbox, Page: tt.page, MaxPage: 30}); f != tt.first || l != tt.last {
			t.Errorf("page %d of 30 links %d-%d, want %d-%d", tt.page, f, l, tt.first, tt.last)
		}
	}
}

func TestMailTexts(t *testing.T) {
	if got := Recipients("  Bob;Ann; "); !slices.Equal(got, []string{"Bob", "Ann"}) {
		t.Errorf("Recipients = %q", got)
	}
	if Subject("") != NoSubject || Subject("x") != "x" {
		t.Error("Subject")
	}
	if got := Message("a\nb\r\nc"); got != "a<br1>b\r<br1>c" {
		t.Errorf("Message = %q", got)
	}
	// Mobius escapes after it turns CR LF into <br>, so line breaks show as
	// text; kept for parity.
	if got := ViewMessage("a<br1>\"b\"\r\nc"); got != "a&lt;br1&gt;&quot;b&quot;&lt;br&gt;c" {
		t.Errorf("ViewMessage = %q", got)
	}
	if got := ViewTitle("<x>"); got != "&lt;x&gt;" {
		t.Errorf("ViewTitle = %q", got)
	}
	mail := &Mail{ID: 3, Type: Archive, Subject: "Hi"}
	if got := ViewLink(mail); got != `<a action="bypass _bbsmail;archive">Mail Archive</a>&nbsp;&gt;&nbsp;Hi` {
		t.Errorf("ViewLink = %q", got)
	}
	if got := ReplyLink(mail); got != `<a action="bypass _bbsmail;archive">Mail Archive</a>&nbsp;&gt;&nbsp;<a action="bypass _bbsmail;view;3">Hi</a>&nbsp;&gt;&nbsp;` {
		t.Errorf("ReplyLink = %q", got)
	}
	if got := FolderTab(Inbox, Inbox, 3); got != "Inbox&nbsp;(3)" {
		t.Errorf("FolderTab shown = %q", got)
	}
	if got := FolderTab(SentBox, Inbox, 0); got != `<a action="bypass _bbsmail;sentbox">Sent Box</a>&nbsp;(0)` {
		t.Errorf("FolderTab = %q", got)
	}
	if PagesCount(0) != 1 || PagesCount(10) != 1 || PagesCount(11) != 2 {
		t.Error("PagesCount")
	}
}
