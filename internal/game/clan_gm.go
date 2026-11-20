package game

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The clan GM commands (decision T16: Mobius AdminPledge and AdminClan),
// /clanpenalty (user command 100, Mobius ClanPenalty) and the clan rule of
// character deletion (GameClient.markToDeleteChar).

const (
	// userCmdClanPenalty is /clanpenalty.
	userCmdClanPenalty = 100
	// gmPledgeRepLevel is the clan level //pledge rep needs (Mobius
	// AdminPledge.REP_POINTS_REWARD_LEVEL).
	gmPledgeRepLevel = 5
)

// registerClanCommands registers the GM and user commands.
func (s *Server) registerClanCommands() error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("pledge", admin, s.gmPledge),
		s.RegisterGMCommand("clan_info", admin, s.gmClanInfo),
		s.RegisterGMCommand("clan_changeleader", admin, s.gmClanChangeLeader),
		s.RegisterGMCommand("clan_show_pending", admin, s.gmClanShowPending),
		s.RegisterGMCommand("clan_force_pending", admin, s.gmClanForcePending),
		s.RegisterUserCommand(userCmdClanPenalty, s.userClanPenalty),
	)
}

// gmPledgeMainPage is AdminPledge.showMainPage.
func (s *Server) gmPledgeMainPage(p *Player) { s.ShowAdminPage(p, "game_menu.htm") }

// gmPledge is //pledge <create name|dismiss|info [clan]|setlevel n|rep n>
// for the targeted player, or the GM itself (Mobius AdminPledge). Every
// action but create needs a clan leader.
func (s *Server) gmPledge(p *Player, args []string) {
	t, ok := gmTargetPlayer(p)
	if !ok {
		t = p
	}
	if len(args) == 0 {
		gmSysMessage(p, "Not allowed Action on Clan")
		s.gmPledgeMainPage(p)
		return
	}
	action := args[0]
	if action != "create" && !t.IsClanLeader() {
		p.SendSystemMessage(clanMsgS1IsNotClanLeader, serverpackets.TextParam(t.Name()))
		s.gmPledgeMainPage(p)
		return
	}
	param := ""
	if len(args) > 1 {
		param = args[1]
	}
	switch action {
	case "create":
		if param == "" {
			gmSysMessage(p, "Please, enter clan name.")
			s.gmPledgeMainPage(p)
			return
		}
		cet := t.char.ClanCreateExpiry
		t.char.ClanCreateExpiry = 0
		if s.CreateClan(t, param) != nil {
			gmSysMessage(p, "Clan "+param+" created. Leader: "+t.Name())
			return
		}
		t.char.ClanCreateExpiry = cet
		gmSysMessage(p, "There was a problem while creating the clan.")
		s.gmPledgeMainPage(p)
	case "dismiss":
		s.DestroyClan(t.clan.clan)
		if t.clan.clan == nil {
			gmSysMessage(p, "Clan disbanded.")
			return
		}
		gmSysMessage(p, "There was a problem while destroying the clan.")
		s.gmPledgeMainPage(p)
	case "info":
		c := t.clan.clan
		if param != "" {
			c = s.ClanByName(param)
		}
		if c == nil {
			return // Mobius throws on an unknown clan
		}
		members := make([]serverpackets.PledgeMember, 0, len(c.members))
		for _, m := range c.members {
			members = append(members, pledgeMember(m))
		}
		p.Send(serverpackets.NewGMViewPledgeInfoPacket(t.Name(), clanInfo(c), members))
	case "setlevel":
		c := t.clan.clan
		level, err := strconv.Atoi(param)
		if param == "" || err != nil || level < 0 || level > social.MaxClanLevel {
			if param != "" {
				gmSysMessage(p, "Level incorrect.")
			}
			gmSysMessage(p, "Usage: //pledge <setlevel|rep> <number>")
			s.gmPledgeMainPage(p)
			return
		}
		s.ChangeClanLevel(c, level)
		gmSysMessage(p, fmt.Sprintf("You set level %d for clan %s", level, c.Name))
	case "rep":
		c := t.clan.clan
		points, err := strconv.Atoi(param)
		if param == "" || err != nil {
			if param != "" {
				gmSysMessage(p, "Points incorrect.")
			}
			gmSysMessage(p, "Usage: //pledge <setlevel|rep> <number>")
			s.gmPledgeMainPage(p)
			return
		}
		if c.Level < gmPledgeRepLevel {
			gmSysMessage(p, "Only clans of level 5 or above may receive reputation points.")
			s.gmPledgeMainPage(p)
			return
		}
		s.AddClanReputation(c, points)
		verb, prep := "add ", "to "
		if points <= 0 {
			verb, prep = "remove ", "from "
		}
		gmSysMessage(p, fmt.Sprintf("You %s%d points %s%s's reputation. Their current score is %d", verb, clanAbs(points), prep, c.Name, c.Reputation))
		s.gmPledgeMainPage(p)
	default:
		gmSysMessage(p, "Clan Action not allowed...")
		s.gmPledgeMainPage(p)
	}
}

func clanAbs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// gmClanPlayer is AdminClan.getPlayer: the player named (or with the object
// ID given) in args, or the GM's target.
func (s *Server) gmClanPlayer(p *Player, args []string) *Player {
	if len(args) == 0 {
		t, ok := gmTargetPlayer(p)
		if !ok {
			p.SendSystemMessage(clanMsgInvalidTarget)
			return nil
		}
		return t
	}
	if bypassIsNumeric(args[0]) {
		id, err := strconv.ParseInt(args[0], 10, 32)
		t, ok := s.Player(world.ObjectID(id))
		if err != nil || !ok {
			p.SendSystemMessage(clanMsgPlayerNotOnline)
			return nil
		}
		return t
	}
	t, ok := s.PlayerByName(args[0])
	if !ok {
		p.SendSystemMessage(clanMsgIncorrectName)
		return nil
	}
	return t
}

// gmClanInfo is //clan_info [player] (Mobius AdminClan): the clan of a
// player on the claninfo page.
func (s *Server) gmClanInfo(p *Player, args []string) {
	t := s.gmClanPlayer(p, args)
	if t == nil {
		return
	}
	c := t.clan.clan
	if c == nil {
		p.SendSystemMessage(clanMsgTargetMustBeMember)
		return
	}
	ally := "Not in ally"
	if c.AllyID > 0 {
		ally = c.AllyName
	}
	pg := s.adminPageFile(p, "claninfo.htm")
	for _, r := range [][2]string{
		{"%clan_name%", c.Name},
		{"%clan_leader%", c.LeaderName()},
		{"%clan_level%", strconv.Itoa(c.Level)},
		{"%clan_has_castle%", "No"},   // castles come in M11
		{"%clan_has_clanhall%", "No"}, // clan halls come in M11
		{"%clan_points%", strconv.Itoa(c.Reputation)},
		{"%clan_players_count%", strconv.Itoa(c.MemberCount())},
		{"%clan_ally%", ally},
		{"%current_player_objectId%", strconv.Itoa(int(t.ObjectID()))},
		{"%current_player_name%", t.Name()},
	} {
		pg.replace(r[0], r[1])
	}
	s.sendPage(p, pg)
}

// gmClanChangeLeader is //clan_changeleader [player]: the player leads its
// clan at once.
func (s *Server) gmClanChangeLeader(p *Player, args []string) {
	t := s.gmClanPlayer(p, args)
	if t == nil {
		return
	}
	c := t.clan.clan
	if c == nil {
		p.SendSystemMessage(clanMsgTargetMustBeMember)
		return
	}
	m := c.member(t.ObjectID())
	if m == nil {
		return
	}
	if t.IsAcademyMember() {
		t.SendSystemMessage(clanMsgNotForAcademyMember)
		return
	}
	s.SetNewClanLeader(c, m)
}

// gmClanShowPending is //clan_show_pending: the pending leader changes.
func (s *Server) gmClanShowPending(p *Player, _ []string) {
	var b strings.Builder
	for _, c := range s.Clans() {
		if c.NewLeaderID == 0 {
			continue
		}
		name := ""
		if m := c.member(c.NewLeaderID); m != nil {
			name = m.Name()
		}
		fmt.Fprintf(&b, "<tr><td>%s</td><td>%s</td><td><a action=\"bypass -h admin_clan_force_pending %d\">Force</a></td></tr>", c.Name, name, c.ID)
	}
	s.sendPage(p, s.adminPageFile(p, "clanchanges.htm").replace("%data%", b.String()))
}

// gmClanForcePending is //clan_force_pending <clanId>: the clan's pending
// leader change now.
func (s *Server) gmClanForcePending(p *Player, args []string) {
	if len(args) == 0 || !bypassIsNumeric(args[0]) {
		return
	}
	id, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return
	}
	c := s.Clan(int32(id))
	if c == nil {
		return
	}
	m := c.member(c.NewLeaderID)
	if m == nil {
		return
	}
	s.SetNewClanLeader(c, m)
	gmSysMessage(p, "Task have been forcely executed.")
}

// userClanPenalty is /clanpenalty (Mobius ClanPenalty): the clan penalties
// of p and its clan with their dates.
func (s *Server) userClanPenalty(p *Player) {
	now := s.Now()
	nowMs := now.UnixMilli()
	date := func(ms int64) string { return time.UnixMilli(ms).In(now.Location()).Format("2006-01-02") }
	var b strings.Builder
	b.WriteString("<html><body><center><table width=270 border=0 bgcolor=111111><tr><td width=170>Penalty</td><td width=100 align=center>Expiration Date</td></tr></table><table width=270 border=0><tr>")
	penalty := false
	if social.Active(p.char.ClanJoinExpiry, nowMs) {
		b.WriteString("<td width=170>Unable to join a clan.</td><td width=100 align=center>" + date(p.char.ClanJoinExpiry) + "</td>")
		penalty = true
	}
	if social.Active(p.char.ClanCreateExpiry, nowMs) {
		b.WriteString("<td width=170>Unable to create a clan.</td><td width=100 align=center>" + date(p.char.ClanCreateExpiry) + "</td>")
		penalty = true
	}
	if c := p.clan.clan; c != nil && social.Active(c.MemberPenaltyExpiry, nowMs) {
		b.WriteString("<td width=170>Unable to invite a clan member.</td><td width=100 align=center>" + date(c.MemberPenaltyExpiry) + "</td>")
		penalty = true
	}
	if !penalty {
		b.WriteString("<td width=170>No penalty is imposed.</td><td width=100 align=center></td>")
	}
	b.WriteString("</tr></table><img src=\"L2UI.SquareWhite\" width=270 height=1></center></body></html>")
	s.sendPage(p, newPage(0, b.String()))
}

// ----- character deletion -----

// ClanDeletion is what the clan rule says about deleting a character
// (Mobius GameClient.markToDeleteChar).
type ClanDeletion int

// The answers of CharacterClanDeletion.
const (
	// ClanDeletionAllowed: no clan, or a clan that no longer exists.
	ClanDeletionAllowed ClanDeletion = iota
	// ClanDeletionMember: clan members may not be deleted
	// (CharDeleteFail REASON_YOU_MAY_NOT_DELETE_CLAN_MEMBER).
	ClanDeletionMember
	// ClanDeletionLeader: clan leaders may not be deleted
	// (CharDeleteFail REASON_CLAN_LEADERS_MAY_NOT_BE_DELETED).
	ClanDeletionLeader
)

// ClanDeletionChecker is the optional part of Service the session layer
// asks before it marks a character for deletion.
type ClanDeletionChecker interface {
	// CharacterClanDeletion reports whether the character with the object
	// ID and stored clan ID may be deleted.
	CharacterClanDeletion(ctx context.Context, objectID, clanID int32) (ClanDeletion, error)
}

var _ ClanDeletionChecker = (*Server)(nil)

// CharacterClanDeletion implements ClanDeletionChecker on the loop: a
// member of an existing clan may not be deleted, its leader neither.
func (s *Server) CharacterClanDeletion(ctx context.Context, objectID, clanID int32) (ClanDeletion, error) {
	if clanID == 0 {
		return ClanDeletionAllowed, nil
	}
	answer := ClanDeletionAllowed
	err := s.loop.Call(ctx, func() {
		c := s.clan.clans[clanID]
		switch {
		case c == nil:
		case c.LeaderID == objectID:
			answer = ClanDeletionLeader
		default:
			answer = ClanDeletionMember
		}
	})
	return answer, err
}
