package bbs

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// The mail board's rules (Mobius MailBoard and communitybbs/BB/Mail).
const (
	// MaxRecipients is how many recipients a player who is no GM may name.
	MaxRecipients = 5
	// DailyLimit is how many mails a player may send in 24 hours, counted in
	// its sent box.
	DailyLimit = 10
	// InboxLimit is how many mails an inbox holds; mail to a full inbox is
	// not sent (GMs ignore it).
	InboxLimit = 100
	// PerPage is how many mails a list page shows.
	PerPage = 10
	// SubjectWidth is how much of a subject a list shows, in UTF-16 units.
	SubjectWidth = 30
	// NoSubject replaces an empty subject.
	NoSubject = "(no subject)"
)

// DateFormat is how mail dates are shown (Mobius "yyyy-MM-dd HH:mm").
const DateFormat = "2006-01-02 15:04"

// MailType is the folder a mail is in (Mobius MailType).
type MailType int

// The folders, in Mobius's order (MailType.VALUES).
const (
	Inbox MailType = iota
	SentBox
	Archive
	TempArchive
	numMailTypes
)

var mailTypes = [numMailTypes]struct {
	name, description, bypass string
}{
	{"INBOX", "Inbox", `<a action="bypass _bbsmail">Inbox</a>`},
	{"SENTBOX", "Sent Box", `<a action="bypass _bbsmail;sentbox">Sent Box</a>`},
	{"ARCHIVE", "Mail Archive", `<a action="bypass _bbsmail;archive">Mail Archive</a>`},
	{"TEMPARCHIVE", "Temporary Mail Archive", `<a action="bypass _bbsmail;temp_archive">Temporary Mail Archive</a>`},
}

// String is the Java enum name ("INBOX"), which the list's page links
// carry.
func (t MailType) String() string {
	if t < 0 || t >= numMailTypes {
		return "MailType(" + strconv.Itoa(int(t)) + ")"
	}
	return mailTypes[t].name
}

// Description is the folder's title ("Sent Box").
func (t MailType) Description() string { return mailTypes[t].description }

// Bypass is the link that opens the folder. The temporary archive's link
// names an action the board does not know, as in Mobius.
func (t MailType) Bypass() string { return mailTypes[t].bypass }

// Location is how the folder is stored (Mobius bbs_mail.location).
func (t MailType) Location() string { return strings.ToLower(t.String()) }

// ParseMailType returns the folder of a name, ignoring case (Mobius
// MailType.valueOf(name.toUpperCase())).
func ParseMailType(name string) (MailType, bool) {
	upper := strings.ToUpper(name)
	for t := range numMailTypes {
		if mailTypes[t].name == upper {
			return t, true
		}
	}
	return 0, false
}

// MailTypeAt returns the folder with Mobius's ordinal i (MailType.VALUES[i]).
func MailTypeAt(i int) (MailType, bool) {
	if i < 0 || i >= int(numMailTypes) {
		return 0, false
	}
	return MailType(i), true
}

// Mail is one mail in one box: the recipient's copy in its inbox (or where
// it moved it) or the sender's copy in its sent box.
type Mail struct {
	ID       int32
	Receiver int32 // the owner of the box
	Sender   int32
	// SenderName is the sender's name when it sent the mail. Mobius looks
	// the name up by ID at every display.
	SenderName string
	Type       MailType
	// Recipients are the recipient names as the sender typed them, joined
	// with ";".
	Recipients string
	Subject    string
	Message    string
	Sent       time.Time
	Unread     bool
}

// Mailboxes are the mail boxes of every character (Mobius MailBoard._mails),
// newest mail first. Loop-only.
type Mailboxes struct {
	boxes  map[int32][]*Mail
	lastID int32
}

// NewMailboxes returns the boxes holding mails, as loaded at start; new
// mail IDs follow the highest one (Mobius _lastMailId).
func NewMailboxes(mails []Mail) *Mailboxes {
	m := &Mailboxes{boxes: make(map[int32][]*Mail)}
	for i := range mails {
		mail := mails[i]
		m.Add(&mail)
		m.lastID = max(m.lastID, mail.ID)
	}
	return m
}

// NextID returns a new mail ID (Mobius getNewMailId).
func (m *Mailboxes) NextID() int32 {
	m.lastID++
	return m.lastID
}

// Add puts a mail into its owner's box. A box lists the newest mail first,
// and of mails sent at the same time the higher ID first (Mobius keeps a
// loaded box sorted by date and a new one in hash order).
func (m *Mailboxes) Add(mail *Mail) {
	box := append(m.boxes[mail.Receiver], mail)
	slices.SortStableFunc(box, func(a, b *Mail) int {
		if c := b.Sent.Compare(a.Sent); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})
	m.boxes[mail.Receiver] = box
}

// Box returns owner's mails, newest first. The slice must not be modified.
func (m *Mailboxes) Box(owner int32) []*Mail { return m.boxes[owner] }

// Get returns the mail with the ID in owner's box, or nil (Mobius getMail).
func (m *Mailboxes) Get(owner, id int32) *Mail {
	for _, mail := range m.boxes[owner] {
		if mail.ID == id {
			return mail
		}
	}
	return nil
}

// Delete removes the mail with the ID from owner's box and reports whether
// it was there.
func (m *Mailboxes) Delete(owner, id int32) bool {
	box := m.boxes[owner]
	i := slices.IndexFunc(box, func(mail *Mail) bool { return mail.ID == id })
	if i < 0 {
		return false
	}
	m.boxes[owner] = slices.Delete(box, i, i+1)
	return true
}

// HasUnread reports whether owner has an unread mail in any folder (Mobius
// checkIfUnreadMail).
func (m *Mailboxes) HasUnread(owner int32) bool {
	return slices.ContainsFunc(m.boxes[owner], func(mail *Mail) bool { return mail.Unread })
}

// SentSince counts owner's sent box mails sent after since (Mobius's daily
// limit check).
func (m *Mailboxes) SentSince(owner int32, since time.Time) int {
	n := 0
	for _, mail := range m.boxes[owner] {
		if mail.Type == SentBox && mail.Sent.After(since) {
			n++
		}
	}
	return n
}

// InboxFull reports whether owner's inbox holds InboxLimit mails.
func (m *Mailboxes) InboxFull(owner int32) bool {
	return m.Count(owner, Inbox, "", "") >= InboxLimit
}

// Search is a mail search: SearchTarget "Title" (any case) searches the
// subjects, anything else the sender names; both ignore case. An empty
// target or keyword is no search.
type Search struct {
	Target, Keyword string
}

// active reports whether s filters anything.
func (s Search) active() bool { return s.Target != "" && s.Keyword != "" }

// matches reports whether a mail passes the search.
func (s Search) matches(mail *Mail) bool {
	if !s.active() {
		return true
	}
	keyword := strings.ToLower(s.Keyword)
	if strings.EqualFold(s.Target, "title") {
		return strings.Contains(strings.ToLower(mail.Subject), keyword)
	}
	return strings.Contains(strings.ToLower(mail.SenderName), keyword)
}

// Count counts owner's mails in a folder that pass the search (Mobius
// getMailCount).
func (m *Mailboxes) Count(owner int32, t MailType, target, keyword string) int {
	s := Search{Target: target, Keyword: keyword}
	n := 0
	for _, mail := range m.boxes[owner] {
		if mail.Type == t && s.matches(mail) {
			n++
		}
	}
	return n
}

// PagesCount is the number of list pages for n mails, at least 1 (Mobius
// getPagesCount).
func PagesCount(n int) int {
	if n < 1 {
		return 1
	}
	return (n + PerPage - 1) / PerPage
}

// ListPage is one page of a folder.
type ListPage struct {
	Type    MailType
	Search  Search
	Page    int // clamped to 1..MaxPage
	MaxPage int
	Mails   []*Mail
}

// List returns page of owner's folder t with the search (Mobius
// showMailList): the page is clamped to the pages there are.
func (m *Mailboxes) List(owner int32, t MailType, s Search, page int) ListPage {
	lp := ListPage{Type: t, Search: s, MaxPage: PagesCount(m.Count(owner, t, s.Target, s.Keyword))}
	lp.Page = max(min(page, lp.MaxPage), 1)
	first := (lp.Page - 1) * PerPage
	index := 0
	for _, mail := range m.boxes[owner] {
		if mail.Type != t || !s.matches(mail) {
			continue
		}
		if index >= first && index < first+PerPage {
			lp.Mails = append(lp.Mails, mail)
		}
		index++
	}
	return lp
}

// Rows are the list rows of the page (Mobius showMailList's %maillist%).
func (lp ListPage) Rows(format func(time.Time) string) string {
	var b strings.Builder
	for _, mail := range lp.Mails {
		b.WriteString(`<table width=610><tr><td width=5></td><td width=150>`)
		b.WriteString(mail.SenderName)
		b.WriteString(`</td><td width=300><a action="bypass _bbsmail;view;`)
		b.WriteString(strconv.Itoa(int(mail.ID)))
		b.WriteString(`">`)
		if mail.Unread {
			b.WriteString(`<font color="LEVEL">`)
		}
		b.WriteString(Abbreviate(mail.Subject, SubjectWidth))
		if mail.Unread {
			b.WriteString(`</font>`)
		}
		b.WriteString(`</a></td><td width=150>`)
		b.WriteString(format(mail.Sent))
		b.WriteString(`</td><td width=5></td></tr></table><img src="L2UI.Squaregray" width=610 height=1>`)
	}
	return b.String()
}

// Pager is the page links under the list (Mobius showMailList's
// %maillistlength%): previous, the page numbers (21 at most around the
// current one), next.
func (lp ListPage) Pager() string {
	var b strings.Builder
	search := ""
	if lp.Search.active() {
		search = ";" + lp.Search.Target + ";" + lp.Search.Keyword
	}
	link := func(page int) string {
		return `bypass _bbsmail;` + lp.Type.String() + ";" + strconv.Itoa(page) + search
	}
	number := func(i int) {
		if i == lp.Page {
			b.WriteString("<td> " + strconv.Itoa(i) + " </td>")
			return
		}
		b.WriteString(`<td><a action="` + link(i) + `"> ` + strconv.Itoa(i) + " </a></td>")
	}
	page, maxPage := lp.Page, lp.MaxPage
	prev := page - 1
	if page == 1 {
		prev = page
	}
	b.WriteString(`<td><table><tr><td></td></tr><tr><td><button action="` + link(prev) + `" back="l2ui_ch3.prev1_down" fore="l2ui_ch3.prev1" width=16 height=16></td></tr></table></td>`)
	switch {
	case maxPage <= 21:
		for i := 1; i <= maxPage; i++ {
			number(i)
		}
	case page <= 11:
		for i := 1; i <= 10+page; i++ {
			number(i)
		}
	case maxPage-page > 10:
		for i := page - 10; i <= page+10; i++ {
			number(i)
		}
	default:
		for i := page - 10; i <= maxPage; i++ {
			number(i)
		}
	}
	next := page + 1
	if page == maxPage {
		next = page
	}
	b.WriteString(`<td><table><tr><td></td></tr><tr><td><button action="` + link(next) + `" back="l2ui_ch3.next1_down" fore="l2ui_ch3.next1" width=16 height=16></td></tr></table></td>`)
	return b.String()
}

// FolderTab is a folder's entry in the list header: its title when shown,
// else its link, with its mail count (Mobius %inbox%, %sentbox%,
// %archive%).
func FolderTab(t, shown MailType, count int) string {
	label := t.Bypass()
	if t == shown {
		label = t.Description()
	}
	return label + "&nbsp;(" + strconv.Itoa(count) + ")"
}

// Recipients splits the recipients a sender typed (Mobius
// recipients.trim().split(";")).
func Recipients(s string) []string { return SplitJava(JavaTrim(s), ";") }

// Subject is the subject a mail is sent with: NoSubject for an empty one.
func Subject(s string) string {
	if s == "" {
		return NoSubject
	}
	return s
}

// Message is the text a mail is sent with: line feeds become "<br1>".
func Message(s string) string { return strings.ReplaceAll(s, "\n", "<br1>") }

// escapeView is Mobius's escaping of a shown subject and text: "<", ">"
// and the double quote become entities. A message's "<br1>" line breaks
// are escaped too, as in Mobius.
var escapeView = strings.NewReplacer("<", "&lt;", ">", "&gt;", `"`, "&quot;")

// ViewTitle is a subject as the mail view shows it.
func ViewTitle(s string) string { return escapeView.Replace(s) }

// ViewMessage is a text as the mail view shows it (Mobius replaces CR LF
// with "<br>" first, then escapes that too).
func ViewMessage(s string) string {
	return escapeView.Replace(strings.ReplaceAll(s, "\r\n", "<br>"))
}

// ViewLink is the folder path of the mail view (%maillink%).
func ViewLink(mail *Mail) string { return mail.Type.Bypass() + "&nbsp;&gt;&nbsp;" + mail.Subject }

// ReplyLink is the folder path of the reply page (%maillink%).
func ReplyLink(mail *Mail) string {
	return mail.Type.Bypass() + `&nbsp;&gt;&nbsp;<a action="bypass _bbsmail;view;` + strconv.Itoa(int(mail.ID)) + `">` + mail.Subject + "</a>&nbsp;&gt;&nbsp;"
}
