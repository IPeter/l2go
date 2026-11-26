package game

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/bbs"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
)

// The mail board (Mobius MailBoard): the folders with their search and
// pages, reading, replying, deleting and archiving mail, and sending it to
// up to five players, who get "You've got mail." and ExMailArrived.

// mailDay is the window of the daily sending limit.
const mailDay = 24 * time.Hour

// mailBoard handles "_bbsmail..." and "_maillist_0_1_0_" (Mobius
// MailBoard.onCommand).
func (s *Server) mailBoard(p *Player, command string) {
	s.rememberBoardPage(p, "Mail Command", command)
	if command == "_bbsmail" || command == "_maillist_0_1_0_" {
		s.showMailList(p, 1, bbs.Inbox, bbs.Search{})
		return
	}
	if !strings.HasPrefix(command, "_bbsmail") {
		return
	}
	tokens := bbs.Tokens(command, ';')
	if len(tokens) < 2 {
		return // Mobius fails on the missing token
	}
	action, args := strings.ToLower(tokens[1]), tokens[2:]
	// mailArg is the mail ID after the action, -1 without one; ok is false
	// for one that is no number (Mobius logs the IllegalArgumentException).
	mailArg := func() (int32, bool) {
		if len(args) == 0 {
			return -1, true
		}
		return parseJavaInt(args[0])
	}
	box := s.board.mail
	switch action {
	case "inbox", "sentbox", "archive", "temparchive":
		page := int32(1)
		if len(args) > 0 {
			var ok bool
			if page, ok = parseJavaInt(args[0]); !ok {
				s.log.Warn("game: invalid mail list bypass", append(p.logArgs(), "bypass", command)...)
				return
			}
		}
		var search bbs.Search
		if len(args) > 1 {
			search.Target = args[1]
		}
		if len(args) > 2 {
			search.Keyword = args[2]
		}
		t, _ := bbs.ParseMailType(action)
		s.showMailList(p, int(page), t, search)
	case "write":
		if html, ok := s.boardFile(p, "mail/mail-write.html"); ok {
			s.sendBoard(p, html)
		}
	case "view", "reply", "del", "store":
		id, ok := mailArg()
		if !ok {
			s.log.Warn("game: invalid mail bypass", append(p.logArgs(), "bypass", command)...)
			return
		}
		mail := box.Get(p.ObjectID(), id)
		switch action {
		case "view":
			if mail == nil {
				s.showLastMailList(p)
				return
			}
			s.showMailView(p, mail)
			if mail.Unread {
				mail.Unread = false
				s.saveMails(mail)
			}
		case "reply":
			if mail == nil {
				s.showLastMailList(p)
				return
			}
			s.showMailReply(p, mail)
		case "del":
			if mail != nil {
				box.Delete(p.ObjectID(), mail.ID)
				s.deleteMailRow(mail.ID)
			}
			s.showLastMailList(p)
		case "store":
			if mail != nil {
				mail.Type = bbs.Archive
				s.saveMails(mail)
			}
			s.showMailList(p, 1, bbs.Archive, bbs.Search{})
		}
	default:
		s.log.Warn("game: unknown mail action", append(p.logArgs(), "action", action)...)
	}
}

// mailBoardWrite handles the mail board's "Write" buttons (Mobius
// MailBoard.writeCommunityBoardCommand): "Send" sends a mail and shows the
// sent box, "Search;<folder>" searches a folder. The friends board's "mail"
// button does nothing, as in Mobius.
func (s *Server) mailBoardWrite(p *Player, req clientpackets.RequestBBSwrite) {
	switch {
	case req.Arg1 == "Send":
		s.sendMail(p, req.Arg3, req.Arg4, req.Arg5, func() { s.showMailList(p, 1, bbs.SentBox, bbs.Search{}) })
	case strings.HasPrefix(req.Arg1, "Search"):
		tokens := bbs.Tokens(req.Arg1, ';')
		if len(tokens) < 2 {
			return
		}
		t, ok := bbs.ParseMailType(tokens[1])
		if !ok {
			return
		}
		s.showMailList(p, 1, t, bbs.Search{Target: req.Arg4, Keyword: req.Arg5})
	}
}

// showMailList shows a page of one of p's folders (Mobius showMailList)
// and remembers the page.
func (s *Server) showMailList(p *Player, page int, t bbs.MailType, search bbs.Search) {
	box := s.board.mail
	id := p.ObjectID()
	lp := box.List(id, t, search, page)
	s.boardState(p).mailPos = lp.Page
	html, ok := s.boardFile(p, "mail/mail.html")
	if !ok {
		return
	}
	html = strings.ReplaceAll(html, "%inbox%", bbs.FolderTab(bbs.Inbox, t, box.Count(id, bbs.Inbox, "", "")))
	html = strings.ReplaceAll(html, "%sentbox%", bbs.FolderTab(bbs.SentBox, t, box.Count(id, bbs.SentBox, "", "")))
	html = strings.ReplaceAll(html, "%archive%", bbs.FolderTab(bbs.Archive, t, box.Count(id, bbs.Archive, "", "")))
	html = strings.ReplaceAll(html, "%temparchive%", strconv.Itoa(box.Count(id, bbs.TempArchive, "", "")))
	html = strings.ReplaceAll(html, "%type%", t.Description())
	html = strings.ReplaceAll(html, "%htype%", strings.ToLower(t.String()))
	html = strings.ReplaceAll(html, "%maillist%", lp.Rows(s.boardDate(bbs.DateFormat)))
	html = strings.ReplaceAll(html, "%maillistlength%", lp.Pager())
	s.sendBoard(p, html)
}

// showLastMailList shows the mail list page p saw last (Mobius
// showLastForum): the page is kept, the folder is the inbox, as Mobius
// keeps only the page in the position it reads the folder from.
func (s *Server) showLastMailList(p *Player) {
	pos := s.boardState(p).mailPos
	if t, ok := bbs.MailTypeAt(pos / 1000); ok {
		s.showMailList(p, pos%1000, t, bbs.Search{})
	}
}

// showMailView shows a mail (Mobius showMailView).
func (s *Server) showMailView(p *Player, mail *bbs.Mail) {
	html, ok := s.boardFile(p, "mail/mail-show.html")
	if !ok {
		return
	}
	html = strings.ReplaceAll(html, "%maillink%", bbs.ViewLink(mail))
	html = strings.ReplaceAll(html, "%writer%", mail.SenderName)
	html = strings.ReplaceAll(html, "%sentDate%", s.boardDate(bbs.DateFormat)(mail.Sent))
	html = strings.ReplaceAll(html, "%receiver%", mail.Recipients)
	html = strings.ReplaceAll(html, "%delDate%", "Unknown")
	html = strings.ReplaceAll(html, "%title%", bbs.ViewTitle(mail.Subject))
	html = strings.ReplaceAll(html, "%mes%", bbs.ViewMessage(mail.Message))
	html = strings.ReplaceAll(html, "%mailId%", strconv.Itoa(int(mail.ID)))
	s.sendBoard(p, html)
}

// showMailReply shows the reply form of a mail (Mobius showWriteView with a
// mail): to the sender, or to the same recipients for p's own mail.
func (s *Server) showMailReply(p *Player, mail *bbs.Mail) {
	html, ok := s.boardFile(p, "mail/mail-reply.html")
	if !ok {
		return
	}
	to := mail.SenderName
	if mail.Sender == p.ObjectID() {
		to = mail.Recipients
	}
	html = strings.ReplaceAll(html, "%maillink%", bbs.ReplyLink(mail))
	html = strings.ReplaceAll(html, "%recipients%", to)
	html = strings.ReplaceAll(html, "%mailId%", strconv.Itoa(int(mail.ID)))
	s.sendBoard(p, html)
}

// mailRecipient is a recipient name as found among the characters (Mobius
// CharInfoTable.getIdByName and getAccessLevelById).
type mailRecipient struct {
	name   string
	id     int32
	access storage.AccessLevel
	found  bool
}

// lookupRecipients finds the characters named, in the game or in the
// store, and runs then on the loop with them, in order, unless p left
// meanwhile. Names of players in the game are found at once; the others
// are read off the loop.
func (s *Server) lookupRecipients(p *Player, names []string, then func([]mailRecipient)) {
	out := make([]mailRecipient, len(names))
	var missing []int
	for i, name := range names {
		out[i].name = name
		if name == "" {
			continue // Mobius getIdByName answers -1
		}
		if q, ok := s.PlayerByName(name); ok {
			out[i].id, out[i].access, out[i].found = q.ObjectID(), q.AccessLevel(), true
			continue
		}
		missing = append(missing, i)
	}
	if len(missing) == 0 {
		then(out)
		return
	}
	logArgs := p.logArgs()
	s.board.lookups.Add(1)
	go func() {
		defer s.board.lookups.Done()
		ctx, cancel := context.WithTimeout(s.background(), relationLoadTimeout)
		defer cancel()
		for _, i := range missing {
			c, err := s.chars.GetByName(ctx, out[i].name)
			switch {
			case err == nil:
				out[i].id, out[i].access, out[i].found = c.ObjectID, c.AccessLevel, true
			case !errors.Is(err, storage.ErrNotFound):
				s.log.Warn("game: cannot look up a mail recipient", append(logArgs, "name", out[i].name, "err", err)...)
			}
		}
		s.loop.Post(func() {
			if p.state == stateInGame {
				then(out)
			}
		})
	}()
}

// sendMail sends a mail from p (Mobius MailBoard.sendMail): at most 10 a
// day and 5 recipients (GMs any number); each recipient gets a copy in its
// inbox unless it does not exist, is p, or, for a sender who is no GM, is a
// GM, refuses all messages, blocks p or has a full inbox. If anyone got
// one, p gets a copy in its sent box. then runs afterwards in every case
// (Mobius shows a page after sendMail), unless p left.
func (s *Server) sendMail(p *Player, recipients, subject, message string, then func()) {
	box := s.board.mail
	now := s.Now()
	if box.SentSince(p.ObjectID(), now.Add(-mailDay)) >= bbs.DailyLimit {
		p.SendSystemMessage(bbsMsgDailyLimit)
		then()
		return
	}
	names := bbs.Recipients(recipients)
	if len(names) > bbs.MaxRecipients && !p.IsGM() {
		p.SendSystemMessage(bbsMsgFiveRecipients)
		then()
		return
	}
	subject, message = bbs.Subject(subject), bbs.Message(message)
	s.lookupRecipients(p, names, func(found []mailRecipient) {
		var sent []*bbs.Mail
		for _, r := range found {
			if !r.found || r.id <= 0 || r.id == p.ObjectID() {
				p.SendSystemMessage(bbsMsgInvalidTarget)
				continue
			}
			to, online := s.onlinePlayer(r.id)
			if !p.IsGM() {
				if r.access > storage.AccessLevelPlayer {
					p.SendSystemMessage(bbsMsgNoMailToGM, textParam(r.name))
					continue
				}
				if online && to.MessageRefusal() {
					p.SendSystemMessage(bbsMsgBlockingEverything, textParam(r.name))
					continue
				}
				if online && to.HasBlocked(p.ObjectID()) {
					p.SendSystemMessage(bbsMsgBlockedYou, textParam(r.name))
					continue
				}
				if box.InboxFull(r.id) {
					p.SendSystemMessage(bbsMsgNotSent)
					if online {
						to.SendSystemMessage(bbsMsgMailboxFull)
					}
					continue
				}
			}
			mail := &bbs.Mail{ID: box.NextID(), Receiver: r.id, Sender: p.ObjectID(), SenderName: p.Name(), Type: bbs.Inbox,
				Recipients: recipients, Subject: subject, Message: message, Sent: now, Unread: true}
			box.Add(mail)
			sent = append(sent, mail)
			if online {
				to.SendSystemMessage(bbsMsgGotMail)
				to.Send(serverpackets.NewExMailArrivedPacket())
			}
		}
		if len(sent) > 0 {
			own := &bbs.Mail{ID: box.NextID(), Receiver: p.ObjectID(), Sender: p.ObjectID(), SenderName: p.Name(), Type: bbs.SentBox,
				Recipients: recipients, Subject: subject, Message: message, Sent: now}
			box.Add(own)
			sent = append(sent, own)
			p.SendSystemMessage(bbsMsgSentMail)
		}
		s.saveMails(sent...)
		then()
	})
}
