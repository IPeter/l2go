package game

import (
	"strings"

	"github.com/IPeter/l2go/internal/bbs"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
)

// The clan board (Mobius ClanBoard): the clan list with its search, a
// clan's home, the clan notice and the clan mail form. Pages come from
// bbs.

// boardClan is what the clan pages show of a clan.
func boardClan(c *Clan) bbs.ClanInfo {
	return bbs.ClanInfo{ID: c.ID, Name: c.Name, LeaderName: c.LeaderName(), Level: c.Level, Members: c.MemberCount(), AllyName: c.AllyName}
}

// boardClans returns every clan in the registry's order.
func (s *Server) boardClans() []bbs.ClanInfo {
	clans := s.Clans()
	out := make([]bbs.ClanInfo, len(clans))
	for i, c := range clans {
		out[i] = boardClan(c)
	}
	return out
}

// clanBoard handles the clan board's bypasses (Mobius ClanBoard.onCommand).
func (s *Server) clanBoard(p *Player, command string) {
	switch {
	case command == "_bbsclan":
		s.rememberBoardPage(p, "Clan", command)
		if c := p.Clan(); c == nil || c.Level < bbs.ClanMinLevel {
			s.clanList(p, 1)
		} else {
			s.clanHome(p, c.ID)
		}
	case strings.HasPrefix(command, "_bbsclan_clanlist"):
		s.rememberBoardPage(p, "Clan List", command)
		switch {
		case command == "_bbsclan_clanlist":
			s.clanList(p, 1)
		case strings.HasPrefix(command, "_bbsclan_clanlist;"):
			index, ok := boardArg(command)
			if !ok {
				s.log.Warn("game: invalid clan list bypass", append(p.logArgs(), "bypass", command)...)
				index = 1
			}
			s.clanList(p, int(index))
		}
	case strings.HasPrefix(command, "_bbsclan_clanhome"):
		s.rememberBoardPage(p, "Clan Home", command)
		switch {
		case command == "_bbsclan_clanhome":
			s.ownClanHome(p)
		case strings.HasPrefix(command, "_bbsclan_clanhome;"):
			if id, ok := boardArg(command); ok {
				s.clanHome(p, id)
			} else {
				s.log.Warn("game: invalid clan home bypass", append(p.logArgs(), "bypass", command)...)
				s.ownClanHome(p)
			}
		}
	case strings.HasPrefix(command, "_bbsclan_clannotice_edit;"):
		s.rememberBoardPage(p, "Clan Edit", command)
		s.clanNotice(p, p.ClanID())
	case strings.HasPrefix(command, "_bbsclan_clannotice_enable"), strings.HasPrefix(command, "_bbsclan_clannotice_disable"):
		enable := strings.HasPrefix(command, "_bbsclan_clannotice_enable")
		title := "Clan Notice Disable"
		if enable {
			title = "Clan Notice Enable"
		}
		s.rememberBoardPage(p, title, command)
		// Mobius lets any member switch the notice; only its leader is
		// offered the link, and only the leader may (decision X10).
		if c := p.Clan(); c != nil && p.IsClanLeader() && c.Notice.Enabled != enable {
			c.Notice.Enabled = enable
			s.saveClan(c)
		}
		s.clanNotice(p, p.ClanID())
	case strings.HasPrefix(command, "_bbsclan;mail"):
		s.rememberBoardPage(p, "Clan Mail", command)
		if c := p.Clan(); c != nil {
			s.clanMailForm(p, c.ID)
		}
	default:
		s.sendBoard(p, bbs.NeedsDevelopmentPage(command))
	}
}

// boardArg is the number after the first ";" of a bypass (Mobius
// Integer.parseInt(command.split(";")[1])).
func boardArg(command string) (int32, bool) {
	parts := bbs.SplitJava(command, ";")
	if len(parts) < 2 {
		return 0, false
	}
	return parseJavaInt(parts[1])
}

// clanList shows page index of the clan list.
func (s *Server) clanList(p *Player, index int) {
	var mine int32
	if c := p.Clan(); c != nil {
		mine = c.ID
	}
	s.sendBoard(p, bbs.ClanListPage(s.boardClans(), index, mine))
}

// ownClanHome shows the home of p's clan; without a clan nothing happens
// (Mobius fails on it).
func (s *Server) ownClanHome(p *Player) {
	if c := p.Clan(); c != nil {
		s.clanHome(p, c.ID)
	}
}

// clanHome shows a clan's home; a clan below level 2 has none, and p gets
// the clan list.
func (s *Server) clanHome(p *Player, id int32) {
	c := s.Clan(id)
	if c == nil {
		return
	}
	if c.Level < bbs.ClanMinLevel {
		p.SendSystemMessage(bbsMsgNoCommunity)
		s.clanBoard(p, "_bbsclan_clanlist")
		return
	}
	s.sendBoard(p, bbs.ClanHomePage(boardClan(c), p.ObjectID() == c.LeaderID))
}

// clanNotice shows the clan notice page: the leader gets the notice to
// edit, a member reads it.
func (s *Server) clanNotice(p *Player, id int32) {
	c := s.Clan(id)
	if c == nil {
		return
	}
	if c.Level < bbs.ClanMinLevel {
		p.SendSystemMessage(bbsMsgNoCommunity)
		s.clanBoard(p, "_bbsclan_clanlist")
		return
	}
	notice := bbs.NoticeText(c.Notice.Text)
	if p.IsClanLeader() {
		s.sendBoardEdit(p, bbs.ClanNoticeEditPage(c.ID, c.Notice.Enabled), notice)
		return
	}
	s.sendBoard(p, bbs.ClanNoticeViewPage(c.ID, c.Notice.Enabled, notice))
}

// clanMailForm shows the clan mail form to the clan's leader.
func (s *Server) clanMailForm(p *Player, id int32) {
	c := s.Clan(id)
	if c == nil {
		return
	}
	if p.ClanID() != id || !p.IsClanLeader() {
		p.SendSystemMessage(bbsMsgOnlyClanLeader)
		s.clanList(p, 1)
		return
	}
	s.sendBoard(p, bbs.ClanMailPage(c.ID, c.Name))
}

// clanBoardWrite handles the clan board's "Write" buttons (Mobius
// ClanBoard.writeCommunityBoardCommand): "Set _" saves the notice, "mail"
// mails the clan, "-1 0" searches the clans.
func (s *Server) clanBoardWrite(p *Player, req clientpackets.RequestBBSwrite) {
	c := p.Clan()
	switch {
	case strings.EqualFold(req.Arg1, "Set") && req.Arg2 == "_":
		// Mobius saves any member's notice; only the leader is offered the
		// form, and only the leader may (decision X10).
		if c != nil && p.IsClanLeader() {
			c.Notice.Text = bbs.NoticeToStore(req.Arg3)
			s.saveClan(c)
			p.SendSystemMessage(bbsMsgNoticeSaved)
		}
	case strings.EqualFold(req.Arg1, "mail"):
		id, ok := parseJavaInt(req.Arg2)
		if !ok || id != p.ClanID() || c == nil {
			return
		}
		names := make([]string, 0, c.MemberCount())
		for _, m := range c.Members() {
			names = append(names, m.Name())
		}
		s.sendMail(p, strings.Join(names, ";"), req.Arg4, req.Arg5, func() { s.clanHome(p, p.ClanID()) })
	case req.Arg1 == "-1" && req.Arg2 == "0":
		var mine int32
		if c != nil {
			mine = c.ID
		}
		s.sendBoard(p, bbs.ClanSearchPage(bbs.SearchClans(s.boardClans(), req.Arg4, req.Arg3), mine, req.Arg4, req.Arg3))
	}
}
