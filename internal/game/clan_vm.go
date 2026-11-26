package game

import (
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
)

// The clan life cycle (decisions T12 and T14): the village master's clan
// commands (Mobius VillageMaster.onBypassFeedback), creation
// (ClanTable.createClan), level-up (Clan.levelUpClan, changeLevel),
// dissolution and recovery, the leader change (Clan.setNewLeader, applied by
// the Wednesday daily reset) and the removal of a clan
// (ClanTable.destroyClan).

// villageMasterCommand runs a village master command for p at n: args are
// the words after the command (Mobius cmdParams, cmdParams2).
type villageMasterCommand func(s *Server, p *Player, n *Npc, args []string)

// villageMasterCommands are the commands registered with
// registerVillageMasterCommand; setupClan copies them into the server.
var villageMasterCommands = map[string]villageMasterCommand{}

// registerVillageMasterCommand adds a village master command (case does not
// matter, as Mobius equalsIgnoreCase). Call it from an init function: the
// units, clan skills (Phase 7b) and alliances (Phase 8) add theirs this way.
func registerVillageMasterCommand(name string, cmd villageMasterCommand) {
	villageMasterCommands[strings.ToLower(name)] = cmd
}

func init() {
	registerVillageMasterCommand("create_clan", func(s *Server, p *Player, _ *Npc, args []string) {
		switch {
		case len(args) == 0 || args[0] == "":
			p.SendSystemMessage(clanMsgCreateClanName)
		case len(args) > 1 || !s.clanNameMatchesTemplate(args[0]):
			p.SendSystemMessage(clanMsgNameInvalid)
		default:
			s.CreateClan(p, args[0])
		}
	})
	registerVillageMasterCommand("dissolve_clan", func(s *Server, p *Player, _ *Npc, _ []string) { s.dissolveClan(p) })
	registerVillageMasterCommand("recover_clan", func(s *Server, p *Player, _ *Npc, _ []string) { s.recoverClan(p) })
	registerVillageMasterCommand("change_clan_leader", (*Server).vmChangeClanLeader)
	registerVillageMasterCommand("cancel_clan_leader_change", (*Server).vmCancelLeaderChange)
	registerVillageMasterCommand("increase_clan_level", func(s *Server, p *Player, _ *Npc, _ []string) {
		if p.clan.clan == nil {
			return // Mobius throws on the missing clan
		}
		if s.LevelUpClan(p) {
			s.broadcastClanLevelUpEffect(p)
		}
	})
}

// villageMasterCommand handles the clan commands of a village master's
// pages (npc_<objectId>_<command> ...), before the NPC's other bypasses. It
// reports whether the command was one of them.
func (s *Server) villageMasterCommand(p *Player, n *Npc, command string) bool {
	words := vmSplit(command)
	cmd, ok := s.clan.vmCommands[strings.ToLower(words[0])]
	if !ok {
		return false
	}
	cmd(s, p, n, words[1:])
	return true
}

// vmSplit is Java's String.split(" "): empty words between spaces stay,
// trailing empty words go.
func vmSplit(s string) []string {
	words := strings.Split(s, " ")
	for len(words) > 1 && words[len(words)-1] == "" {
		words = words[:len(words)-1]
	}
	return words
}

// clanNameMatchesTemplate is VillageMaster.isValidName: ClanNameTemplate
// matches the whole name.
func (s *Server) clanNameMatchesTemplate(name string) bool {
	re := s.clan.nameTemplate
	return re == nil || re.MatchString(name)
}

// ----- creation -----

// CreateClan creates a clan led by p (Mobius ClanTable.createClan): level
// 10, no clan, no create penalty, a name of 2 to 16 letters or digits that
// no clan has. The clan and its leader are written jointly. It returns the
// clan, or nil with the reason told to p. Loop-only.
func (s *Server) CreateClan(p *Player, name string) *Clan {
	f := &s.clan
	switch {
	case p.Level() < social.MinLevelToCreateClan:
		p.SendSystemMessage(clanMsgCreateCriteria)
		return nil
	case p.char.ClanID != 0 || p.clan.clan != nil:
		p.SendSystemMessage(clanMsgFailedToCreate)
		return nil
	case s.Now().UnixMilli() < p.char.ClanCreateExpiry:
		p.SendSystemMessage(clanMsgWait10Days)
		return nil
	}
	switch social.CheckClanName(name, nil) {
	case social.NameInvalid:
		p.SendSystemMessage(clanMsgNameInvalid)
		return nil
	case social.NameBadLength:
		p.SendSystemMessage(clanMsgNameLength)
		return nil
	}
	if s.ClanByName(name) != nil {
		p.SendSystemMessage(clanMsgS1AlreadyExists, serverpackets.TextParam(name))
		return nil
	}
	var id int32
	ok := false
	if s.items.pool != nil {
		id, ok = s.items.pool.TryNext()
	}
	if !ok {
		s.log.Error("game: no object ID for a new clan", append(p.logArgs(), "clan", name)...)
		p.SendSystemMessage(clanMsgFailedToCreate)
		return nil
	}
	c := &Clan{ID: id, Name: name, LeaderID: p.ObjectID(), byID: make(map[int32]*ClanMember)}
	f.clans[id] = c
	f.byName[social.NameKey(name)] = c
	p.char.ClanID = id
	// Mobius leaves the new leader's grade at 0 until the next login, which
	// gives it grade 1; L2Go gives it grade 1 at once.
	p.char.PowerGrade = social.RankLeader
	m := &ClanMember{data: storage.ClanMember{ObjectID: p.ObjectID()}}
	c.addMember(m)
	m.setPlayer(p)
	p.clan.clan = c
	p.clan.privs = social.AllPrivileges
	s.refreshPledgeClass(p)
	s.SaveJoint([]*Player{p}, storage.Batch{Clans: []storage.Clan{c.document()}})
	s.log.Info("clan created", append(p.logArgs(), "clan", name, "clanId", id)...)

	p.Send(serverpackets.NewPledgeShowInfoUpdatePacket(clanInfo(c)))
	s.sendClanWindow(p, c)
	p.UpdateUserInfo()
	p.Send(memberUpdatePacket(m))
	p.SendSystemMessage(clanMsgClanCreated)
	return c
}

// ----- level-up -----

// LevelUpClan raises p's clan by one level (Mobius Clan.levelUpClan): only
// the leader, not during a dissolution; the cost of the level comes from
// the leader (SP, adena, an item) or the clan (reputation, with enough
// members). On success the leader gets UserInfo and ItemList and the clan
// changes level (ChangeClanLevel); the leader and the clan are written
// jointly. Loop-only.
func (s *Server) LevelUpClan(p *Player) bool {
	c := p.clan.clan
	if c == nil || !p.IsClanLeader() {
		p.SendSystemMessage(clanMsgNotAuthorized)
		return false
	}
	if s.Now().UnixMilli() < c.DissolvingExpiry {
		p.SendSystemMessage(clanMsgNoLevelUpDissolving)
		return false
	}
	cost, ok := s.clan.rules.LevelUpCost(c.Level)
	if !ok {
		return false
	}
	increase := false
	switch {
	case cost.Adena > 0:
		// Levels 0 and 1: SP and adena (the adena message comes first).
		if int64(p.char.SP) >= cost.SP && p.inv.Adena() >= cost.Adena && s.ReduceAdena(p, cost.Adena, true) {
			s.takeClanLevelSP(p, cost.SP)
			increase = true
		}
	case cost.ItemID != 0:
		// Levels 2 to 4: SP and an item.
		if int64(p.char.SP) >= cost.SP && p.inv.ByItemID(cost.ItemID) != nil && s.DestroyItemByItemID(p, cost.ItemID, 1, false) {
			s.takeClanLevelSP(p, cost.SP)
			p.SendSystemMessage(clanMsgS1HasDisappeared, serverpackets.ItemNameParam(cost.ItemID))
			increase = true
		}
	default:
		// Levels 5 to 7: reputation, with enough members of every unit.
		if c.Reputation >= cost.Reputation && c.MemberCount() >= cost.Members {
			s.setClanReputation(c, -cost.Reputation)
			p.SendSystemMessage(clanMsgReputationDeducted, serverpackets.NumberParam(int32(cost.Reputation))) //nolint:gosec // config values
			increase = true
		}
	}
	if !increase {
		p.SendSystemMessage(clanMsgLevelUpConditions)
		return false
	}
	p.UpdateUserInfo()
	s.SendItemList(p, false)
	s.changeClanLevel(c, c.Level+1)
	s.SaveJoint([]*Player{p}, storage.Batch{Clans: []storage.Clan{c.document()}})
	s.log.Info("clan level up", append(p.logArgs(), "clan", c.Name, "level", c.Level)...)
	return true
}

// takeClanLevelSP takes the SP of a level-up from the leader with "Your SP
// has decreased by $s1.".
func (s *Server) takeClanLevelSP(p *Player, sp int64) {
	s.removeSP(p, sp)
	p.SendSystemMessage(clanMsgSPDecreased, serverpackets.NumberParam(int32(sp))) //nolint:gosec // at most 2,500,000
}

// broadcastClanLevelUpEffect shows the clan level-up effect on the leader
// to it and the players around (VillageMaster: MagicSkillUse and
// MagicSkillLaunched of skill 5103).
func (s *Server) broadcastClanLevelUpEffect(p *Player) {
	pos := p.Position()
	use := serverpackets.NewMagicSkillUsePacket(serverpackets.MagicSkillUse{
		CasterID: p.ObjectID(), TargetID: p.ObjectID(),
		SkillID: classChangeSkill, SkillLevel: 1,
		X: pos.X, Y: pos.Y, Z: pos.Z,
		TargetX: pos.X, TargetY: pos.Y, TargetZ: pos.Z,
	})
	launched := serverpackets.NewMagicSkillLaunchedPacket(serverpackets.MagicSkillLaunched{
		CasterID: p.ObjectID(), SkillID: classChangeSkill, SkillLevel: 1, TargetIDs: []int32{p.ObjectID()},
	})
	if p.InWorld() {
		s.world.Broadcast(p, use, true)
		s.world.Broadcast(p, launched, true)
		return
	}
	p.Send(use)
	p.Send(launched)
}

// ChangeClanLevel sets c's level (Mobius Clan.changeLevel, which //pledge
// setlevel uses too): the rank is recalculated, the leader online is told
// above level 4 that the clan gains reputation now, and the members online
// get "Your clan's skill level has increased." and PledgeShowInfoUpdate.
// The clan is saved. Loop-only.
func (s *Server) ChangeClanLevel(c *Clan, level int) {
	s.changeClanLevel(c, level)
	s.saveClan(c)
}

func (s *Server) changeClanLevel(c *Clan, level int) {
	c.Level = level
	c.Rank = s.clanRank(c)
	if l := c.Leader(); l != nil && l.Online() && level > 4 {
		// Siege skills come with sieges (M11).
		l.player.SendSystemMessage(clanMsgReputationAbove5)
	}
	broadcastToClan(c, clanSysMsg(clanMsgSkillLevelIncreased))
	s.broadcastClanInfo(c)
}

// ----- dissolution -----

// dissolveClan asks for the dissolution of p's clan (Mobius
// VillageMaster.dissolveClan): the leader of a clan outside an alliance,
// not at war, without a castle or clan hall, not in a siege zone and not
// dissolving yet. The clan goes in DaysToDissolve days and the leader takes
// the exp loss of a death.
func (s *Server) dissolveClan(p *Player) {
	c := p.clan.clan
	switch {
	case c == nil || !p.IsClanLeader():
		p.SendSystemMessage(clanMsgNotAuthorized)
		return
	case c.AllyID != 0:
		p.SendSystemMessage(clanMsgDissolveInAlliance)
		return
	case c.AtWar():
		p.SendSystemMessage(clanMsgDissolveAtWar)
		return
	case c.CastleID != 0 || c.HideoutID != 0:
		p.SendSystemMessage(clanMsgDissolveResidence)
		return
	case p.Zones.Inside(ZoneSiege):
		// No siege registrations exist before M11; the zone check stays.
		p.SendSystemMessage(clanMsgDissolveSiege)
		return
	}
	now := s.Now().UnixMilli()
	if c.DissolvingExpiry > now {
		p.SendSystemMessage(clanMsgAlreadyDissolving)
		return
	}
	c.DissolvingExpiry = social.AfterDays(now, s.clan.days.DissolveClan)
	// The clan leader takes the exp penalty of a full death.
	s.deathExpPenalty(p, nil)
	p.UpdateUserInfo()
	s.SaveJoint([]*Player{p}, storage.Batch{Clans: []storage.Clan{c.document()}})
	s.scheduleClanRemoval(c)
	s.log.Info("clan dissolution requested", append(p.logArgs(), "clan", c.Name, "expiry", time.UnixMilli(c.DissolvingExpiry))...)
}

// recoverClan cancels the dissolution of p's clan (Mobius
// VillageMaster.recoverClan).
func (s *Server) recoverClan(p *Player) {
	c := p.clan.clan
	if c == nil || !p.IsClanLeader() {
		p.SendSystemMessage(clanMsgNotAuthorized)
		return
	}
	c.DissolvingExpiry = 0
	if id, ok := s.clan.dissolve[c.ID]; ok {
		s.loop.Cancel(id)
		delete(s.clan.dissolve, c.ID)
	}
	s.saveClan(c)
}

// scheduleClanRemoval arms the removal of a dissolving clan (Mobius
// ClanTable.scheduleRemoveClan): at its expiry, at least 5 minutes from
// now; the clan goes if it is still dissolving then.
func (s *Server) scheduleClanRemoval(c *Clan) {
	if id, ok := s.clan.dissolve[c.ID]; ok {
		s.loop.Cancel(id)
	}
	id := c.ID
	delay := social.DissolveDelay(c.DissolvingExpiry, s.Now().UnixMilli())
	s.clan.dissolve[id] = s.loop.After(delay, func() {
		delete(s.clan.dissolve, id)
		if c := s.clan.clans[id]; c != nil && c.DissolvingExpiry != 0 {
			s.DestroyClan(c)
		}
	})
}

// DestroyClan removes c (Mobius ClanTable.destroyClan): its members online
// are told "Clan has dispersed.", every member leaves it (without a join
// penalty; the leader gets the create penalty), and the clan, its
// warehouse and the wars on it are deleted, together with the members'
// changes in one write. Loop-only.
func (s *Server) DestroyClan(c *Clan) {
	f := &s.clan
	if f.clans[c.ID] != c {
		return
	}
	broadcastToClan(c, clanSysMsg(clanMsgClanDispersed))
	var b storage.Batch
	var online []*Player
	for _, m := range append([]*ClanMember(nil), c.members...) {
		if p := m.Player(); p != nil {
			online = append(online, p)
		}
		b.Patches = append(b.Patches, s.removeClanMember(c, m.ObjectID(), 0)...)
	}
	delete(f.clans, c.ID)
	if f.byName[social.NameKey(c.Name)] == c {
		delete(f.byName, social.NameKey(c.Name))
	}
	if id, ok := f.dissolve[c.ID]; ok {
		s.loop.Cancel(id)
		delete(f.dissolve, c.ID)
	}
	for _, o := range f.clans {
		o.WarsDeclared = clanDeleteID(o.WarsDeclared, c.ID)
		o.Attackers = clanDeleteID(o.Attackers, c.ID)
	}
	b.DeletedClans = []int32{c.ID}
	s.SaveJoint(online, b)
	s.log.Info("clan destroyed", "clan", c.Name, "clanId", c.ID)
}

// clanDeleteID removes id from list.
func clanDeleteID(list []int32, id int32) []int32 {
	out := list[:0]
	for _, v := range list {
		if v != id {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ----- leader change -----

// vmChangeClanLeader is "change_clan_leader <name>": the leader names a
// member online, not in the academy, as the next leader; it takes over at
// the next Wednesday reset, or at once with LeaderInstantActivation.
func (s *Server) vmChangeClanLeader(p *Player, n *Npc, args []string) {
	if len(args) == 0 || args[0] == "" {
		return
	}
	name := args[0]
	if !p.IsClanLeader() {
		p.SendSystemMessage(clanMsgNotAuthorized)
		return
	}
	if strings.EqualFold(p.Name(), name) {
		return
	}
	c := p.clan.clan
	m := c.MemberByName(name)
	switch {
	case m == nil:
		p.SendSystemMessage(clanMsgS1DoesNotExist, serverpackets.TextParam(name))
		return
	case !m.Online():
		p.SendSystemMessage(clanMsgPlayerNotCurrentlyOnline)
		return
	case m.player.IsAcademyMember():
		p.SendSystemMessage(clanMsgNotForAcademyMember)
		return
	}
	if s.gs.Clan.LeaderInstantActivation {
		s.SetNewClanLeader(c, m)
		return
	}
	page := "9000-07-in-progress.htm"
	if c.NewLeaderID == 0 {
		c.NewLeaderID = m.ObjectID()
		s.saveClan(c)
		page = "9000-07-success.htm"
	}
	s.sendPage(p, s.pageFile(p, n.ID(), "scripts/village_master/ClanMaster/"+page).replaceObjectID(n.ID()))
}

// vmCancelLeaderChange is "cancel_clan_leader_change".
func (s *Server) vmCancelLeaderChange(p *Player, n *Npc, _ []string) {
	if !p.IsClanLeader() {
		p.SendSystemMessage(clanMsgNotAuthorized)
		return
	}
	c := p.clan.clan
	if c.NewLeaderID == 0 {
		s.sendPage(p, newPage(n.ID(), "<html><body>You don't have clan leader delegation applications submitted yet!</body></html>"))
		return
	}
	c.NewLeaderID = 0
	s.saveClan(c)
	s.sendPage(p, s.pageFile(p, n.ID(), "scripts/village_master/ClanMaster/9000-07-canceled.htm").replaceObjectID(n.ID()))
}

// SetNewClanLeader makes m c's leader (Mobius Clan.setNewLeader): the old
// leader loses its privileges, the new one gets them all, a pending change
// is dropped, both get their pledge class and UserInfo, the members online
// get the new clan window and "Clan lord privileges have been transferred
// to $s1.". The clan and the two leaders are written jointly, an offline
// leader with a patch of its grade. Loop-only.
func (s *Server) SetNewClanLeader(c *Clan, m *ClanMember) {
	old := c.Leader()
	c.LeaderID = m.ObjectID()
	c.NewLeaderID = 0
	var players []*Player
	var b storage.Batch
	if old != nil {
		if q := old.Player(); q != nil {
			if q.mount.flying { // mount.go: Mobius gets the old lord off a wyvern
				s.dismount(q)
			}
			q.clan.privs = 0
			s.refreshPledgeClass(q)
			q.BroadcastUserInfo()
			players = append(players, q)
		}
	}
	if q := m.Player(); q != nil {
		s.refreshPledgeClass(q)
		q.clan.privs = social.AllPrivileges
		q.BroadcastUserInfo()
		players = append(players, q)
	}
	b.Clans = []storage.Clan{c.document()}
	s.SaveJoint(players, b)
	s.broadcastClanStatus(c)
	broadcastToClan(c, clanSysMsg(clanMsgLeaderTransferred, serverpackets.TextParam(m.Name())))
	oldName := ""
	if old != nil {
		oldName = old.Name()
	}
	s.log.Info("clan leader changed", "clan", c.Name, "leader", m.Name(), "exLeader", oldName)
}

// ----- the daily reset -----

// dailyResetVar is the global variable of the last daily reset (Mobius
// GlobalVariablesManager.DAILY_TASK_RESET), Unix milliseconds.
const dailyResetVar = "DAILY_TASK_RESET"

// startDailyReset runs a missed reset (Mobius DailyResetManager's start-up
// check) and schedules the next one at 06:30 of the loop clock's time
// zone.
func (s *Server) startDailyReset() {
	now := s.Now()
	last, _ := s.quest.globals.Long(dailyResetVar, 0)
	if social.DailyResetMissed(now, last) {
		s.log.Info("daily reset missed while the server was down: running it now")
		s.dailyReset()
	}
	s.scheduleDailyReset()
}

func (s *Server) scheduleDailyReset() {
	now := s.Now()
	next := social.NextDailyReset(now)
	if !next.After(now) {
		next = social.NextDailyReset(now.Add(time.Minute))
	}
	s.clan.resetTimer = s.loop.After(next.Sub(now), func() {
		s.dailyReset()
		s.scheduleDailyReset()
	})
}

// dailyReset is DailyResetManager.onReset: the time is stored and, on
// Wednesdays, the pending clan leader changes are applied.
func (s *Server) dailyReset() {
	now := s.Now()
	s.setGlobalVar(dailyResetVar, strconv.FormatInt(now.UnixMilli(), 10))
	if social.LeaderChangeDay(now) {
		s.applyPendingClanLeaders()
	}
}

// applyPendingClanLeaders is DailyResetManager.clanLeaderApply.
func (s *Server) applyPendingClanLeaders() {
	for _, c := range sortedClans(s.clan.clans) {
		if c.NewLeaderID == 0 {
			continue
		}
		if m := c.member(c.NewLeaderID); m != nil {
			s.SetNewClanLeader(c, m)
		}
	}
	s.log.Info("clan leaders have been updated")
}
