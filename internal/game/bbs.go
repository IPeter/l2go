package game

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IPeter/l2go/internal/bbs"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
)

// The community board (Alt+B, Plan 14 M11o): Mobius CommunityBoardHandler
// with the eight standard boards of dp/scripts/handlers/bypass/
// communityboard (home, favorites, region, clan, memo, mail, friends,
// homepage). RequestShowBoard opens the default page, the board's links
// come as RequestBypassToServer (routed here by bypass.go), its "Write"
// buttons as RequestBBSwrite. Pages go out as ShowBoard parts
// (bbs.Parts), and their links are cached in the community board scope.
// The drop search board and the custom board stay off (decision X8).
func init() { registerFeature("bbs", setupBoard) }

// System messages of the community board (Mobius SystemMessageId).
const (
	bbsMsgInvalidTarget      serverpackets.SystemMessageID = 109  // Invalid target.
	bbsMsgOnlyClanLeader     serverpackets.SystemMessageID = 236  // Only the clan leader is enabled.
	bbsMsgOffline            serverpackets.SystemMessageID = 938  // The community server is currently offline.
	bbsMsgBlockingEverything serverpackets.SystemMessageID = 940  // $s1 is blocking everything.
	bbsMsgNoCommunity        serverpackets.SystemMessageID = 1050 // There are no communities in my clan. ...
	bbsMsgMailboxFull        serverpackets.SystemMessageID = 1205 // Your mailbox is full. There is a 100 message limit.
	bbsMsgBlockedYou         serverpackets.SystemMessageID = 1228 // $s1 has blocked you. You cannot send mail to $s1 .
	bbsMsgDailyLimit         serverpackets.SystemMessageID = 1229 // No more messages may be sent at this time. ...
	bbsMsgFiveRecipients     serverpackets.SystemMessageID = 1230 // You are limited to five recipients at a time.
	bbsMsgSentMail           serverpackets.SystemMessageID = 1231 // You've sent mail.
	bbsMsgNotSent            serverpackets.SystemMessageID = 1232 // The message was not sent.
	bbsMsgGotMail            serverpackets.SystemMessageID = 1233 // You've got mail.
	bbsMsgNoMailToGM         serverpackets.SystemMessageID = 1370 // Your message to $s1 did not reach it's recipient. ...
	bbsMsgNoticeSaved        serverpackets.SystemMessageID = 1556 // Your clan notice has been saved.
)

// boardFeature is the community board's state (Server.board). Loop-only,
// except store, which New sets.
type boardFeature struct {
	store  storage.BBSStore // nil keeps mails and favorites in memory only
	writer *boardWriter
	mail   *bbs.Mailboxes
	favs   *bbs.Favorites
	// players are the board states of the players in the game.
	players map[int32]*boardPlayer
	// lookups counts the recipient lookups running off the loop (tests
	// wait for them).
	lookups sync.WaitGroup
}

// boardPlayer is a player's board state: the last page it may bookmark
// (Mobius CommunityBoardHandler._bypasses), its selections on the friend
// and block lists (Player._selectedFriendList, _selectedBlocksList) and
// the mail list page it saw last (Player._mailPosition).
type boardPlayer struct {
	bypass     string // "title&bypass"
	hasBypass  bool
	selFriends []int32
	selBlocks  []int32
	mailPos    int
}

func setupBoard(s *Server) error {
	f := &s.board
	f.players = make(map[int32]*boardPlayer)
	f.writer = newBoardWriter(s.log, s.background)
	if err := s.loadBoard(); err != nil { // bbs_store.go
		return err
	}
	if err := s.Register(clientpackets.OpRequestShowBoard, s.handleRequestShowBoard); err != nil {
		return err
	}
	if err := s.Register(clientpackets.OpRequestBBSwrite, s.handleRequestBBSwrite); err != nil {
		return err
	}
	s.OnPlayerEnter(s.boardPlayerEntered)
	s.OnPlayerLeave(func(p *Player) { delete(f.players, p.ObjectID()) })
	return nil
}

// boardState returns p's board state.
func (s *Server) boardState(p *Player) *boardPlayer {
	st := s.board.players[p.ObjectID()]
	if st == nil {
		st = &boardPlayer{}
		s.board.players[p.ObjectID()] = st
	}
	return st
}

// rememberBoardPage records the page p is on for "add to favorites"
// (Mobius CommunityBoardHandler.addBypass).
func (s *Server) rememberBoardPage(p *Player, title, command string) {
	st := s.boardState(p)
	st.bypass, st.hasBypass = title+"&"+command, true
}

// boardPlayerEntered tells a player with unread mail at login (Mobius
// MailBoard.onPlayerLogin).
func (s *Server) boardPlayerEntered(p *Player) {
	if s.gs.CommunityBoard.Enabled && s.board.mail.HasUnread(p.ObjectID()) {
		p.SendSystemMessage(bbsMsgGotMail)
		p.Send(serverpackets.NewExMailArrivedPacket())
	}
}

// isBoardBypass reports whether a bypass belongs to a board (Mobius
// CommunityBoardHandler.isCommunityBoardCommand).
func isBoardBypass(command string) bool {
	_, ok := bbs.Route(command)
	return ok
}

// handleRequestShowBoard handles RequestShowBoard (0x57), Alt+B: the
// default page (Mobius BBSDefault).
func (s *Server) handleRequestShowBoard(p *Player, body []byte) {
	if _, err := clientpackets.ParseRequestShowBoard(body); err != nil {
		s.log.Debug("game: bad RequestShowBoard", append(p.logArgs(), "err", err)...)
		return
	}
	s.boardBypass(p, s.gs.CommunityBoard.Default)
}

// boardBypass runs a board command (Mobius
// CommunityBoardHandler.handleParseCommand).
func (s *Server) boardBypass(p *Player, command string) {
	if !s.gs.CommunityBoard.Enabled {
		p.SendSystemMessage(bbsMsgOffline)
		return
	}
	board, ok := bbs.Route(command)
	if !ok {
		s.log.Warn("game: no board for the command", append(p.logArgs(), "command", command)...)
		return
	}
	switch board {
	case bbs.BoardHome:
		s.homeBoard(p, command) // bbs_pages.go
	case bbs.BoardFavorite:
		s.favoriteBoard(p, command) // bbs_pages.go
	case bbs.BoardRegion:
		s.regionBoard(p, command) // bbs_region.go
	case bbs.BoardClan:
		s.clanBoard(p, command) // bbs_clan.go
	case bbs.BoardMemo:
		s.memoBoard(p, command) // bbs_pages.go
	case bbs.BoardMail:
		s.mailBoard(p, command) // bbs_mail.go
	case bbs.BoardFriends:
		s.friendsBoard(p, command) // bbs_friends.go
	case bbs.BoardHomepage:
		s.homepageBoard(p) // bbs_pages.go
	}
}

// handleRequestBBSwrite handles RequestBBSwrite (0x22), a "Write" button
// (Mobius CommunityBoardHandler.handleWriteCommand).
func (s *Server) handleRequestBBSwrite(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestBBSwrite(body)
	if err != nil {
		s.log.Debug("game: bad RequestBBSwrite", append(p.logArgs(), "err", err)...)
		return
	}
	if !s.gs.CommunityBoard.Enabled {
		p.SendSystemMessage(bbsMsgOffline)
		return
	}
	board, ok := bbs.WriteTarget(req.URL)
	if !ok {
		s.sendBoard(p, bbs.NotImplementedPage(req.URL))
		return
	}
	switch board {
	case bbs.BoardClan:
		s.clanBoardWrite(p, req) // bbs_clan.go
	case bbs.BoardMail:
		s.mailBoardWrite(p, req) // bbs_mail.go
	case bbs.BoardRegion, bbs.BoardMemo:
		// A TODO in Mobius: nothing happens.
	default:
		s.log.Warn("game: the board takes no writes", append(p.logArgs(), "url", req.URL)...)
	}
}

// sendBoard shows p a board page (Mobius HtmlUtil.sendCBHtml): its links go
// to the community board scope of the link cache, not bound to an object,
// and the page goes out as the ShowBoard parts 101, 102 and 103.
func (s *Server) sendBoard(p *Player, html string) {
	p.HTML.Set(HTMLScopeCommunityBoard, 0, html)
	for i, part := range bbs.Parts(html) {
		p.Send(serverpackets.NewShowBoardPacket(serverpackets.BoardContent(bbs.PartIDs[i], part)))
	}
}

// sendBoardEdit shows p a board page whole and fills its multiedit field
// with text (Mobius HtmlUtil.sendCBHtml with fillMultiEdit and
// fillMultiEditContent).
func (s *Server) sendBoardEdit(p *Player, html, text string) {
	p.HTML.Set(HTMLScopeCommunityBoard, 0, html)
	p.Send(serverpackets.NewShowBoardPacket(serverpackets.BoardContent(bbs.EditPageID, html)))
	fill := []string{"0", "0", "0", "0", "0", "0", p.Name(), strconv.Itoa(int(p.ObjectID())), p.Account(), "9", " ", " ",
		strings.ReplaceAll(text, "<br>", "\n"), "0", "0", "0", "0"}
	p.Send(serverpackets.NewShowBoardPacket(serverpackets.BoardFillContent(fill)))
}

// boardFile reads a page of the HTML folder's CommunityBoard folder (Mobius
// HtmCache.getHtm with "data/html/CommunityBoard/"); a missing one is
// logged.
func (s *Server) boardFile(p *Player, rel string) (string, bool) {
	content, ok := s.htmlFile(p, "CommunityBoard/"+rel)
	if !ok {
		s.log.Warn("game: missing community board page", append(p.logArgs(), "page", htmlPrefix+"CommunityBoard/"+rel)...)
	}
	return content, ok
}

// boardDate formats a time in the server's zone.
func (s *Server) boardDate(layout string) func(t time.Time) string {
	return func(t time.Time) string { return s.calendar.In(t).Format(layout) }
}

// parseJavaInt parses a number like the Java Integer.parseInt.
func parseJavaInt(s string) (int32, bool) {
	v, err := strconv.ParseInt(s, 10, 32)
	return int32(v), err == nil
}
