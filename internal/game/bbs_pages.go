package game

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/IPeter/l2go/internal/bbs"
)

// The home, favorites, memo and homepage boards (Mobius HomeBoard,
// FavoriteBoard, MemoBoard and HomepageBoard).

// homeBoard handles "_bbshome" and "_bbstop" (Mobius HomeBoard without the
// custom board): the home page with the favorite, region and clan counts,
// or "_bbstop;<page>.html", a page of the CommunityBoard folder.
func (s *Server) homeBoard(p *Player, command string) {
	cfg := s.gs.CommunityBoard
	if cfg.KarmaDisabled && p.Karma() > 0 {
		p.SendMessage("Players with Karma cannot use the Community Board.")
		return
	}
	if cfg.PeaceOnly && !p.Zones.Inside(ZonePeace) {
		p.SendMessage("Community Board cannot be used out of peace zone.")
		return
	}
	switch {
	case command == "_bbshome" || command == "_bbstop":
		s.rememberBoardPage(p, "Home", command)
		html, ok := s.boardFile(p, "home.html")
		if !ok {
			return
		}
		html = strings.ReplaceAll(html, "%fav_count%", strconv.Itoa(s.board.favs.Count(p.ObjectID())))
		html = strings.ReplaceAll(html, "%region_count%", "0") // a TODO in Mobius
		html = strings.ReplaceAll(html, "%clan_count%", strconv.Itoa(len(s.Clans())))
		s.sendBoard(p, html)
	case strings.HasPrefix(command, "_bbstop;"):
		page := strings.ReplaceAll(command, "_bbstop;", "")
		if page != "" && strings.HasSuffix(page, ".html") {
			if html, ok := s.boardFile(p, page); ok {
				s.sendBoard(p, html)
			}
		}
	}
}

// favoriteBoard handles the favorites (Mobius FavoriteBoard):
// "_bbsgetfav" lists them, "bbs_add_fav" (the tab's button) bookmarks the
// last page p opened that may be bookmarked, "_bbsdelfav_<id>" deletes one.
func (s *Server) favoriteBoard(p *Player, command string) {
	f := &s.board
	switch {
	case strings.HasPrefix(command, "_bbsgetfav"):
		s.showFavorites(p)
	case strings.HasPrefix(command, "bbs_add_fav"):
		st := s.boardState(p)
		if !st.hasBypass {
			return
		}
		remembered := st.bypass
		st.bypass, st.hasBypass = "", false
		title, bypass, ok := bbs.SplitBypass(remembered)
		if !ok {
			s.log.Warn("game: cannot add a favorite: not a valid bypass", append(p.logArgs(), "bypass", remembered)...)
			return
		}
		if utf8.RuneCountInString(title) > bbs.MaxFavoriteTitle || utf8.RuneCountInString(bypass) > bbs.MaxFavoriteBypass {
			s.log.Warn("game: cannot add a favorite: too long", append(p.logArgs(), "title", title, "bypass", bypass)...)
			return
		}
		s.saveFavorite(f.favs.Add(p.ObjectID(), title, bypass, s.Now()))
		s.showFavorites(p)
	case strings.HasPrefix(command, "_bbsdelfav_"):
		text := strings.ReplaceAll(command, "_bbsdelfav_", "")
		id, ok := parseJavaInt(text)
		if !bbs.IsNumeric(text) || !ok {
			s.log.Warn("game: cannot delete a favorite: not a valid ID", append(p.logArgs(), "id", text)...)
			return
		}
		f.favs.Delete(p.ObjectID(), id)
		s.deleteFavoriteRow(p.ObjectID(), id)
		s.showFavorites(p)
	}
}

// showFavorites shows p its favorites, newest first.
func (s *Server) showFavorites(p *Player) {
	row, ok := s.boardFile(p, "favorite/favorite_list.html")
	if !ok {
		return
	}
	date := s.boardDate(bbs.FavoriteDateFormat)
	var list strings.Builder
	for _, fav := range s.board.favs.List(p.ObjectID()) {
		link := strings.ReplaceAll(row, "%fav_bypass%", fav.Bypass)
		link = strings.ReplaceAll(link, "%fav_title%", fav.Title)
		link = strings.ReplaceAll(link, "%fav_add_date%", date(fav.Added))
		link = strings.ReplaceAll(link, "%fav_id%", strconv.Itoa(int(fav.ID)))
		list.WriteString(link)
	}
	html, ok := s.boardFile(p, "favorite/favorite.html")
	if !ok {
		return
	}
	s.sendBoard(p, strings.ReplaceAll(html, "%fav_list%", list.String()))
}

// memoBoard handles "_bbsmemo" and "_bbstopics" (Mobius MemoBoard): the
// memo page. Writing memos is a TODO in Mobius.
func (s *Server) memoBoard(p *Player, command string) {
	s.rememberBoardPage(p, "Memo Command", command)
	if html, ok := s.boardFile(p, "memo/memo.html"); ok {
		s.sendBoard(p, html)
	}
}

// homepageBoard handles "_bbslink" (Mobius HomepageBoard).
func (s *Server) homepageBoard(p *Player) {
	if html, ok := s.boardFile(p, "homepage.html"); ok {
		s.sendBoard(p, html)
	}
}
