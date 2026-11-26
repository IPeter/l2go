package game

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The clan registry (Plan 12 Phase 7, decision T1): every clan with its
// members, loaded before the loop starts, as Mobius ClanTable. The clan
// document is written through the saver (SaveJoint) whenever it changes;
// membership lives on the characters, written with their saves or, for an
// offline member, with a storage.CharacterPatch.
//
// The files: clan.go (registry, player side, packets), clan_vm.go (village
// master commands, dissolution, leader change, daily reset), clan_members.go
// (invites, withdrawal, dismissal, online state, chat), clan_ranks.go (ranks,
// privileges, titles, member info, unit moves), clan_gm.go (GM and user
// commands, character deletion).
//
// For the follow-up work (crests, clan skills, academy, warehouse, units,
// alliances, wars): Clan carries the crest IDs, Units, Skills, the alliance
// fields and WarsDeclared/Attackers; change them on the loop and call
// saveClan (or SaveJoint with c.document() for a joint write); the window
// helpers (sendClanWindow, broadcastClanStatus, broadcastClanInfo) and
// registerVillageMasterCommand are the hooks.
func init() { registerFeature("clan", setupClan) }

// clanLoadTimeout bounds loading the clans at startup.
const clanLoadTimeout = time.Minute

// System messages of the clans (Mobius SystemMessageId).
const (
	clanMsgS1DoesNotExist           serverpackets.SystemMessageID = 2
	clanMsgCannotAskYourself        serverpackets.SystemMessageID = 4
	clanMsgS1AlreadyExists          serverpackets.SystemMessageID = 5
	clanMsgS1IsNotClanLeader        serverpackets.SystemMessageID = 9
	clanMsgS1WorkingWithAnotherClan serverpackets.SystemMessageID = 10
	clanMsgInvalidTarget            serverpackets.SystemMessageID = 109
	clanMsgPlayerNotOnline          serverpackets.SystemMessageID = 145
	clanMsgWrongTarget              serverpackets.SystemMessageID = 152
	clanMsgPlayerNotCurrentlyOnline serverpackets.SystemMessageID = 161
	clanMsgWaitingForReply          serverpackets.SystemMessageID = 164
	clanMsgCreateClanName           serverpackets.SystemMessageID = 188
	clanMsgClanCreated              serverpackets.SystemMessageID = 189
	clanMsgFailedToCreate           serverpackets.SystemMessageID = 190
	clanMsgMemberExpelled           serverpackets.SystemMessageID = 191
	clanMsgClanDispersed            serverpackets.SystemMessageID = 193
	clanMsgEnteredClan              serverpackets.SystemMessageID = 195
	clanMsgYouWithdrew              serverpackets.SystemMessageID = 197
	clanMsgRecentlyDismissed        serverpackets.SystemMessageID = 199
	clanMsgIncorrectName            serverpackets.SystemMessageID = 204
	clanMsgNotClanMember            serverpackets.SystemMessageID = 212
	clanMsgTitleChanged             serverpackets.SystemMessageID = 214
	clanMsgS1JoinedClan             serverpackets.SystemMessageID = 222
	clanMsgS1Withdrew               serverpackets.SystemMessageID = 223
	clanMsgS1DidNotRespond          serverpackets.SystemMessageID = 224
	clanMsgYouDidNotRespond         serverpackets.SystemMessageID = 225
	clanMsgCreateCriteria           serverpackets.SystemMessageID = 229
	clanMsgWait10Days               serverpackets.SystemMessageID = 230
	clanMsgClanWaitAfterDismiss     serverpackets.SystemMessageID = 231
	clanMsgWaitAfterLeaving         serverpackets.SystemMessageID = 232
	clanMsgUnitFull                 serverpackets.SystemMessageID = 233
	clanMsgTargetMustBeMember       serverpackets.SystemMessageID = 234
	clanMsgLeaderCannotWithdraw     serverpackets.SystemMessageID = 239
	clanMsgNameInvalid              serverpackets.SystemMessageID = 261
	clanMsgNameLength               serverpackets.SystemMessageID = 262
	clanMsgAlreadyDissolving        serverpackets.SystemMessageID = 263
	clanMsgDissolveAtWar            serverpackets.SystemMessageID = 264
	clanMsgDissolveSiege            serverpackets.SystemMessageID = 265
	clanMsgDissolveResidence        serverpackets.SystemMessageID = 266
	clanMsgCannotDismissYourself    serverpackets.SystemMessageID = 269
	clanMsgTitleNeedsLevel3         serverpackets.SystemMessageID = 271
	clanMsgSkillLevelIncreased      serverpackets.SystemMessageID = 274
	clanMsgS1HasDisappeared         serverpackets.SystemMessageID = 302
	clanMsgMemberLoggedIn           serverpackets.SystemMessageID = 304
	clanMsgExpelSucceeded           serverpackets.SystemMessageID = 309
	clanMsgSPDecreased              serverpackets.SystemMessageID = 538
	clanMsgNoLevelUpDissolving      serverpackets.SystemMessageID = 551
	clanMsgDissolveInAlliance       serverpackets.SystemMessageID = 554
	clanMsgS1JoinPenalty            serverpackets.SystemMessageID = 760
	clanMsgNotAuthorized            serverpackets.SystemMessageID = 794
	clanMsgNoLeaveInCombat          serverpackets.SystemMessageID = 1116
	clanMsgNoDismissInCombat        serverpackets.SystemMessageID = 1117
	clanMsgAcademyRequirements      serverpackets.SystemMessageID = 1734
	clanMsgS1NotForAcademy          serverpackets.SystemMessageID = 1735
	clanMsgNotForAcademyMember      serverpackets.SystemMessageID = 1754
	clanMsgApprenticeLoggedIn       serverpackets.SystemMessageID = 1756
	clanMsgSponsorLoggedIn          serverpackets.SystemMessageID = 1758
	clanMsgReputationAbove5         serverpackets.SystemMessageID = 1771
	clanMsgReputationDeducted       serverpackets.SystemMessageID = 1787
	clanMsgSkillsDeactivated        serverpackets.SystemMessageID = 1789
	clanMsgLevelUpConditions        serverpackets.SystemMessageID = 1790
	clanMsgLeaderTransferred        serverpackets.SystemMessageID = 1798
	clanMsgS1IsFull                 serverpackets.SystemMessageID = 1835
	clanMsgSkillsActivated          serverpackets.SystemMessageID = 1862
)

// clanFeature is the clan registry on the Server (Server.clan). Loop-only,
// except store, which New sets.
type clanFeature struct {
	// store reads the clans and their members at startup; nil starts
	// without clans (Options.Clans).
	store storage.ClanStore

	rules        social.ClanRules
	days         social.ClanDays
	nameTemplate *regexp.Regexp

	clans  map[int32]*Clan  // by ID
	byName map[string]*Clan // by social.NameKey(name)

	// dissolve are the pending dissolution timers by clan ID
	// (ClanTable.scheduleRemoveClan).
	dissolve map[int32]world.TimerID
	// resetTimer is the next daily reset (DailyResetManager).
	resetTimer world.TimerID
	// vmCommands are the village master commands (VillageMaster
	// onBypassFeedback), by lower-case name.
	vmCommands map[string]villageMasterCommand
}

// Clan is one clan of the registry (Mobius Clan). Its fields are the stored
// clan (storage.Clan) and what is derived from it; change them on the loop
// and write them with saveClan. Members are reached with Members and Member.
type Clan struct {
	ID                    int32
	Name                  string
	Level                 int
	Reputation            int
	Rank                  int // ClanTable.getClanRank, recalculated every 20 minutes
	LeaderID              int32
	NewLeaderID           int32 // a leader change waiting for the Wednesday reset, or 0
	CrestID, CrestLargeID int32

	AllyID      int32
	AllyName    string
	AllyCrestID int32
	AllyPenalty storage.AllyPenalty
	// MemberPenaltyExpiry is when the clan may accept members again after a
	// dismissal (Mobius _charPenaltyExpiryTime), Unix milliseconds.
	MemberPenaltyExpiry int64
	// DissolvingExpiry is when a requested dissolution happens, Unix
	// milliseconds, 0 for none.
	DissolvingExpiry    int64
	CastleID, HideoutID int32

	Ranks  social.RankTable
	Units  []storage.ClanUnit
	Skills []storage.ClanSkill
	Notice storage.ClanNotice

	// WarsDeclared are the clans this clan declared war on (Mobius
	// _atWarWith, stored); Attackers the clans that declared war on it
	// (Mobius _atWarAttackers, derived at load).
	WarsDeclared []int32
	Attackers    []int32

	members []*ClanMember // in the order they were loaded or joined
	byID    map[int32]*ClanMember
}

// ClanMember is a member of a clan (Mobius ClanMember): the stored fields
// while offline, the player's own while online.
type ClanMember struct {
	clan   *Clan
	data   storage.ClanMember
	player *Player // while in the game
}

// playerClan is a player's clan side (Player.clan): its clan, privileges
// and pledge class (Mobius _clan, _clanPrivileges, _pledgeClass).
type playerClan struct {
	clan        *Clan
	privs       social.Privileges
	pledgeClass int
}

// setupClan reads the clan rules, loads the clans and starts the rank and
// daily-reset timers.
func setupClan(s *Server) error {
	f := &s.clan
	cfg := s.gs.Clan
	f.rules = social.ClanRules{
		LevelCost:         [3]int{cfg.Level6Cost, cfg.Level7Cost, cfg.Level8Cost},
		LevelMembers:      [3]int{cfg.Level6Members, cfg.Level7Members, cfg.Level8Members},
		RoyalGuardCost:    cfg.RoyalGuardCost,
		KnightUnitCost:    cfg.KnightUnitCost,
		ReputationPerKill: cfg.ReputationPerKill,
		AcademyMinPoints:  cfg.AcademyMinPoints,
		AcademyMaxPoints:  cfg.AcademyMaxPoints,
	}
	ally := s.gs.Ally
	f.days = social.ClanDays{
		JoinClan:                cfg.DaysBeforeJoin,
		CreateClan:              cfg.DaysBeforeCreate,
		DissolveClan:            cfg.DaysToDissolve,
		AllyJoinWhenLeft:        ally.DaysBeforeJoinWhenLeft,
		AllyJoinWhenDismissed:   ally.DaysBeforeJoinWhenDismissed,
		AllyAcceptWhenDismissed: ally.DaysBeforeAcceptWhenDismissed,
		AllyCreateWhenDissolved: ally.DaysBeforeCreateWhenDissolved,
	}
	if cfg.NameTemplate != "" {
		re, err := cfg.NamePattern()
		if err != nil {
			return fmt.Errorf("clan.nameTemplate: %w", err)
		}
		f.nameTemplate = re
	}
	f.clans = make(map[int32]*Clan)
	f.byName = make(map[string]*Clan)
	f.dissolve = make(map[int32]world.TimerID)
	if f.vmCommands == nil {
		f.vmCommands = make(map[string]villageMasterCommand)
	}
	for name, cmd := range villageMasterCommands {
		f.vmCommands[name] = cmd
	}
	if f.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), clanLoadTimeout)
		defer cancel()
		if err := s.LoadClans(ctx); err != nil {
			return err
		}
	}
	s.loop.Every(social.ClanRankInterval, s.updateClanRanks)
	s.loop.After(0, s.startDailyReset) // once the loop runs: the quest globals are loaded then
	s.OnPlayerEnter(s.clanPlayerEntered)
	s.OnPlayerLeave(s.clanPlayerLeft)
	return errors.Join(s.registerClanPackets(), s.registerClanCommands())
}

// LoadClans replaces the registry with the clans of the clan store and
// their members (Mobius ClanTable): alliances whose leader clan is gone are
// dropped (allianceCheck), the wars are restored (restoreClanWars), pending
// dissolutions are re-armed (scheduleRemoveClan) and the ranks calculated.
// New calls it with Options.Clans; call it before Run. It does nothing
// without a clan store.
func (s *Server) LoadClans(ctx context.Context) error {
	f := &s.clan
	if f.store == nil {
		return nil
	}
	list, err := f.store.LoadClans(ctx)
	if err != nil {
		return fmt.Errorf("game: load the clans: %w", err)
	}
	members, err := f.store.ClanMembers(ctx, 0)
	if err != nil {
		return fmt.Errorf("game: load the clan members: %w", err)
	}
	clear(f.clans)
	clear(f.byName)
	nowMs := s.Now().UnixMilli()
	for _, doc := range list {
		c := clanFromDocument(doc, f.days, nowMs)
		f.clans[c.ID] = c
		f.byName[social.NameKey(c.Name)] = c
	}
	for _, m := range members {
		if c := f.clans[m.ClanID]; c != nil {
			c.addMember(&ClanMember{clan: c, data: m})
		}
	}
	var fixed []*Clan
	for _, c := range sortedClans(f.clans) {
		// restoreClanWars: wars on clans that are gone are left out.
		c.WarsDeclared = slices.DeleteFunc(c.WarsDeclared, func(id int32) bool { return f.clans[id] == nil })
		for _, id := range c.WarsDeclared {
			if t := f.clans[id]; !slices.Contains(t.Attackers, c.ID) {
				t.Attackers = append(t.Attackers, c.ID)
			}
		}
		// allianceCheck.
		if c.AllyID != 0 && c.AllyID != c.ID && f.clans[c.AllyID] == nil {
			s.log.Info("game: removed the alliance of a clan whose alliance leader is gone", "clan", c.Name, "allyId", c.AllyID)
			c.AllyID, c.AllyName, c.AllyCrestID = 0, "", 0
			fixed = append(fixed, c)
		}
		if c.member(c.LeaderID) == nil {
			s.log.Warn("game: clan without its leader", "clan", c.Name, "leaderId", c.LeaderID)
		}
		if c.DissolvingExpiry != 0 {
			s.scheduleClanRemoval(c)
		}
	}
	if len(fixed) > 0 {
		b := storage.Batch{}
		for _, c := range fixed {
			b.Clans = append(b.Clans, c.document())
		}
		s.SaveJoint(nil, b)
	}
	s.updateClanRanks()
	s.log.Info("clans loaded", "clans", len(f.clans), "members", len(members))
	return nil
}

// clanFromDocument builds a clan of the registry from its stored document,
// with Mobius Clan.restore's clean-up of expired penalties.
func clanFromDocument(doc storage.Clan, days social.ClanDays, nowMs int64) *Clan {
	c := &Clan{
		ID:                  doc.ID,
		Name:                doc.Name,
		Level:               int(doc.Level),
		Reputation:          int(doc.Reputation),
		LeaderID:            doc.LeaderID,
		NewLeaderID:         doc.NewLeaderID,
		CrestID:             doc.CrestID,
		CrestLargeID:        doc.CrestLargeID,
		AllyID:              doc.AllyID,
		AllyName:            doc.AllyName,
		AllyCrestID:         doc.AllyCrestID,
		MemberPenaltyExpiry: days.RestoredMemberPenalty(doc.MemberPenaltyExpiry, nowMs),
		DissolvingExpiry:    doc.DissolvingExpiry,
		CastleID:            doc.CastleID,
		HideoutID:           doc.HideoutID,
		Units:               slices.Clone(doc.Units),
		Skills:              slices.Clone(doc.Skills),
		Notice:              doc.Notice,
		WarsDeclared:        slices.Clone(doc.WarsDeclared),
		byID:                make(map[int32]*ClanMember),
	}
	expiry, typ := social.RestoredAllyPenalty(doc.AllyPenalty.Expiry, int(doc.AllyPenalty.Type), nowMs)
	c.AllyPenalty = storage.AllyPenalty{Expiry: expiry, Type: int32(typ)} //nolint:gosec // penalty types are small
	for _, r := range doc.Ranks {
		c.Ranks.Set(int(r.Rank), social.Privileges(r.Privs))
	}
	return c
}

// document is the clan as it is stored.
func (c *Clan) document() storage.Clan {
	doc := storage.Clan{
		ID:                  c.ID,
		Name:                c.Name,
		Level:               int32(c.Level),      //nolint:gosec // clan levels are small
		Reputation:          int32(c.Reputation), //nolint:gosec // capped at 100,000,000
		LeaderID:            c.LeaderID,
		NewLeaderID:         c.NewLeaderID,
		CrestID:             c.CrestID,
		CrestLargeID:        c.CrestLargeID,
		AllyID:              c.AllyID,
		AllyName:            c.AllyName,
		AllyCrestID:         c.AllyCrestID,
		AllyPenalty:         c.AllyPenalty,
		MemberPenaltyExpiry: c.MemberPenaltyExpiry,
		DissolvingExpiry:    c.DissolvingExpiry,
		CastleID:            c.CastleID,
		HideoutID:           c.HideoutID,
		Units:               slices.Clone(c.Units),
		Skills:              slices.Clone(c.Skills),
		Notice:              c.Notice,
		WarsDeclared:        slices.Clone(c.WarsDeclared),
	}
	for rank := 1; rank <= social.Ranks; rank++ {
		doc.Ranks = append(doc.Ranks, storage.ClanRank{Rank: int32(rank), Privs: int32(c.Ranks.Of(rank))}) //nolint:gosec // small
	}
	return doc
}

// sortedClans returns the clans in ID order.
func sortedClans(m map[int32]*Clan) []*Clan {
	list := make([]*Clan, 0, len(m))
	for _, c := range m {
		list = append(list, c)
	}
	slices.SortFunc(list, func(a, b *Clan) int { return int(a.ID) - int(b.ID) })
	return list
}

// saveClan writes c's document through the saver (decision T6). Loop-only.
func (s *Server) saveClan(c *Clan) {
	s.SaveJoint(nil, storage.Batch{Clans: []storage.Clan{c.document()}})
}

// Clan returns the clan with the ID, or nil. Loop-only.
func (s *Server) Clan(id int32) *Clan { return s.clan.clans[id] }

// ClanByName returns the clan with the name, ignoring case (Mobius
// getClanByName), or nil. Loop-only.
func (s *Server) ClanByName(name string) *Clan { return s.clan.byName[social.NameKey(name)] }

// Clans returns the clans in ID order. Loop-only.
func (s *Server) Clans() []*Clan { return sortedClans(s.clan.clans) }

// ----- members -----

func (c *Clan) addMember(m *ClanMember) {
	m.clan = c
	m.data.ClanID = c.ID
	if c.byID == nil {
		c.byID = make(map[int32]*ClanMember)
	}
	if _, ok := c.byID[m.data.ObjectID]; !ok {
		c.members = append(c.members, m)
	}
	c.byID[m.data.ObjectID] = m
}

func (c *Clan) removeMember(id int32) *ClanMember {
	m := c.byID[id]
	if m == nil {
		return nil
	}
	delete(c.byID, id)
	c.members = slices.DeleteFunc(c.members, func(x *ClanMember) bool { return x == m })
	return m
}

// member returns the member with the object ID, or nil.
func (c *Clan) member(id int32) *ClanMember {
	if id == 0 {
		return nil
	}
	return c.byID[id]
}

// Member returns the member with the object ID, or nil (Mobius
// getClanMember(int)).
func (c *Clan) Member(id int32) *ClanMember { return c.member(id) }

// MemberByName returns the member with the name, compared exactly as Mobius
// getClanMember(String) does, or nil.
func (c *Clan) MemberByName(name string) *ClanMember {
	for _, m := range c.members {
		if m.Name() == name {
			return m
		}
	}
	return nil
}

// Members returns the members in order. Don't change the slice.
func (c *Clan) Members() []*ClanMember { return c.members }

// MemberCount is the number of members of every unit (Mobius
// getMembersCount).
func (c *Clan) MemberCount() int { return len(c.members) }

// UnitMemberCount is the number of members in the unit (Mobius
// getSubPledgeMembersCount).
func (c *Clan) UnitMemberCount(unit int) int {
	n := 0
	for _, m := range c.members {
		if m.PledgeType() == unit {
			n++
		}
	}
	return n
}

// Leader returns the leader, or nil.
func (c *Clan) Leader() *ClanMember { return c.member(c.LeaderID) }

// LeaderName is the leader's name, empty without one (Mobius
// getLeaderName).
func (c *Clan) LeaderName() string {
	if l := c.Leader(); l != nil {
		return l.Name()
	}
	return ""
}

// AtWar reports whether the clan declared war on any clan (Mobius isAtWar).
func (c *Clan) AtWar() bool { return len(c.WarsDeclared) > 0 }

// AtWarWith reports whether the clan declared war on the clan id (Mobius
// isAtWarWith).
func (c *Clan) AtWarWith(id int32) bool { return slices.Contains(c.WarsDeclared, id) }

// Unit returns the unit of the type, or nil (Mobius getSubPledge).
func (c *Clan) Unit(unit int) *storage.ClanUnit {
	for i := range c.Units {
		if int(c.Units[i].Type) == unit {
			return &c.Units[i]
		}
	}
	return nil
}

// UnitLedBy returns the unit the member leads, or 0 (Mobius
// getLeaderSubPledge).
func (c *Clan) UnitLedBy(id int32) int {
	unit := 0
	for _, u := range c.Units {
		if u.LeaderID != 0 && u.LeaderID == id {
			unit = int(u.Type)
		}
	}
	return unit
}

// OnlineMembers yields the players of the members in the game.
func (c *Clan) OnlineMembers() []*Player {
	var list []*Player
	for _, m := range c.members {
		if m.Online() {
			list = append(list, m.player)
		}
	}
	return list
}

// ObjectID is the member's object ID.
func (m *ClanMember) ObjectID() int32 { return m.data.ObjectID }

// Clan is the member's clan.
func (m *ClanMember) Clan() *Clan { return m.clan }

// Player is the member's player while it is in the game, or nil.
func (m *ClanMember) Player() *Player { return m.player }

// Online reports whether the member is in the game (Mobius isOnline).
func (m *ClanMember) Online() bool { return m.player != nil && m.player.state == stateInGame }

// Name is the member's name.
func (m *ClanMember) Name() string {
	if p := m.player; p != nil {
		return p.char.Name
	}
	return m.data.Name
}

// Level is the member's level.
func (m *ClanMember) Level() int {
	if p := m.player; p != nil {
		return p.Level()
	}
	return int(m.data.Level)
}

// ClassID is the member's class.
func (m *ClanMember) ClassID() int32 {
	if p := m.player; p != nil {
		return p.char.ClassID
	}
	return m.data.ClassID
}

// Race is the member's race.
func (m *ClanMember) Race() int32 {
	if p := m.player; p != nil {
		return p.char.Race
	}
	return m.data.Race
}

// Female reports whether the member is female.
func (m *ClanMember) Female() bool {
	if p := m.player; p != nil {
		return p.Female()
	}
	return m.data.Sex == 1
}

// Title is the member's title.
func (m *ClanMember) Title() string {
	if p := m.player; p != nil {
		return p.char.Title
	}
	return m.data.Title
}

// PowerGrade is the member's rank.
func (m *ClanMember) PowerGrade() int {
	if p := m.player; p != nil {
		return int(p.char.PowerGrade)
	}
	return int(m.data.PowerGrade)
}

// PledgeType is the member's unit.
func (m *ClanMember) PledgeType() int {
	if p := m.player; p != nil {
		return int(p.char.PledgeType)
	}
	return int(m.data.PledgeType)
}

// Sponsor is the member's sponsor (academy), or 0.
func (m *ClanMember) Sponsor() int32 {
	if p := m.player; p != nil {
		return p.char.Sponsor
	}
	return m.data.Sponsor
}

// Apprentice is the member's apprentice, or 0.
func (m *ClanMember) Apprentice() int32 {
	if p := m.player; p != nil {
		return p.char.Apprentice
	}
	return m.data.Apprentice
}

// ApprenticeOrSponsorName is Mobius getApprenticeOrSponsorName.
func (m *ClanMember) ApprenticeOrSponsorName() string {
	id := m.Apprentice()
	if id == 0 {
		id = m.Sponsor()
	}
	if id == 0 {
		return ""
	}
	if o := m.clan.member(id); o != nil {
		return o.Name()
	}
	return "Error"
}

// setPlayer is Mobius ClanMember.setPlayer: a member leaving keeps its
// values for the offline list.
func (m *ClanMember) setPlayer(p *Player) {
	if p == nil && m.player != nil {
		c := &m.player.char
		m.data.Name, m.data.Level, m.data.ClassID = c.Name, int32(m.player.Level()), c.ClassID //nolint:gosec // levels are small
		m.data.Race, m.data.Sex, m.data.Title = c.Race, c.Sex, c.Title
		m.data.PowerGrade, m.data.PledgeType = c.PowerGrade, c.PledgeType
		m.data.Apprentice, m.data.Sponsor = c.Apprentice, c.Sponsor
		m.data.LvlJoinedAcademy, m.data.WantsPeace = c.LvlJoinedAcademy, c.WantsPeace
	}
	m.player = p
}

// patch changes stored fields of an offline member: in the registry and
// as a CharacterPatch for the saver.
func (m *ClanMember) patch(fields storage.PatchFields, v storage.PatchValues) storage.CharacterPatch {
	pt := storage.CharacterPatch{ObjectID: m.data.ObjectID, Fields: fields, Values: v}
	c := storage.Character{
		ClanID: m.data.ClanID, Title: m.data.Title, PowerGrade: m.data.PowerGrade, PledgeType: m.data.PledgeType,
		Apprentice: m.data.Apprentice, Sponsor: m.data.Sponsor, LvlJoinedAcademy: m.data.LvlJoinedAcademy,
		WantsPeace: m.data.WantsPeace,
	}
	pt.Apply(&c)
	m.data.ClanID, m.data.Title, m.data.PowerGrade, m.data.PledgeType = c.ClanID, c.Title, c.PowerGrade, c.PledgeType
	m.data.Apprentice, m.data.Sponsor, m.data.LvlJoinedAcademy, m.data.WantsPeace = c.Apprentice, c.Sponsor, c.LvlJoinedAcademy, c.WantsPeace
	return pt
}

// ----- the player's side -----

// Clan returns the player's clan, or nil. Loop-only.
func (p *Player) Clan() *Clan { return p.clan.clan }

// ClanID returns the player's clan ID, 0 without a clan.
func (p *Player) ClanID() int32 {
	if c := p.clan.clan; c != nil {
		return c.ID
	}
	return 0
}

// IsClanLeader reports whether the player leads its clan (Mobius
// isClanLeader).
func (p *Player) IsClanLeader() bool {
	c := p.clan.clan
	return c != nil && c.LeaderID == p.ObjectID()
}

// ClanPrivileges returns the player's clan privileges.
func (p *Player) ClanPrivileges() social.Privileges { return p.clan.privs }

// HasClanAccess reports whether the player has the clan privilege (Mobius
// hasAccess).
func (p *Player) HasClanAccess(a social.Access) bool { return p.clan.privs.Has(a) }

// PowerGrade returns the player's clan rank, 0 without a clan.
func (p *Player) PowerGrade() int { return int(p.char.PowerGrade) }

// PledgeType returns the player's clan unit.
func (p *Player) PledgeType() int { return int(p.char.PledgeType) }

// PledgeClass returns the social class UserInfo and CharInfo show (Mobius
// getPledgeClass), set at login and when the membership changes.
func (p *Player) PledgeClass() int { return p.clan.pledgeClass }

// IsAcademyMember reports whether the player joined the academy (Mobius
// isAcademyMember: lvlJoinedAcademy > 0).
func (p *Player) IsAcademyMember() bool { return p.char.LvlJoinedAcademy > 0 }

// refreshPledgeClass recalculates p's pledge class (Mobius
// setPledgeClass(ClanMember.calculatePledgeClass(player))). Nobles and
// heroes call it when they change.
func (s *Server) refreshPledgeClass(p *Player) {
	m := social.PledgeMember{Noble: p.IsNoble(), Hero: p.IsHero()}
	if c := p.clan.clan; c != nil {
		m.InClan = true
		m.ClanLevel = c.Level
		m.Leader = c.LeaderID == p.ObjectID()
		m.Unit = p.PledgeType()
		m.LeadsUnit = c.UnitLedBy(p.ObjectID())
	}
	p.clan.pledgeClass = social.PledgeClass(m)
}

// restoreClan binds a selected character to its clan (Mobius
// Player.restore): a member gets its rank's privileges (grade 5 when stored
// as 0), the leader every privilege and grade 1; a character whose clan is
// gone, or that is no member of it any more, leaves it (Player.setClan
// with null). Loop-only.
func (s *Server) restoreClan(p *Player) {
	p.clan = playerClan{}
	if id := p.char.ClanID; id != 0 {
		c := s.clan.clans[id]
		if c != nil && c.member(p.ObjectID()) != nil {
			p.clan.clan = c
			privs, grade := social.MemberAccess(c.LeaderID == p.ObjectID(), int(p.char.PowerGrade), &c.Ranks)
			p.clan.privs = privs
			p.char.PowerGrade = int32(grade) //nolint:gosec // 1-9
		} else {
			clearClanFields(p)
		}
	}
	s.refreshPledgeClass(p)
}

// clearClanFields is Mobius Player.setClan(null).
func clearClanFields(p *Player) {
	p.clan.clan = nil
	p.clan.privs = 0
	c := &p.char
	c.Title = ""
	c.ClanID, c.PledgeType, c.PowerGrade = 0, 0, 0
	c.LvlJoinedAcademy, c.Apprentice, c.Sponsor = 0, 0, 0
}

// clanView is what UserInfo and CharInfo show of a player's clan (Mobius
// getClanId, getClanCrestId, getAllyId, ...).
type clanView struct {
	clanID, crestID, crestLargeID, allyID, allyCrestID int32
	userRelation, privileges, pledgeClass, pledgeType  int32
}

func (p *Player) clanView() clanView {
	v := clanView{pledgeClass: int32(p.clan.pledgeClass), pledgeType: p.char.PledgeType} //nolint:gosec // small
	c := p.clan.clan
	if c == nil {
		return v
	}
	v.clanID, v.crestID, v.crestLargeID = c.ID, c.CrestID, c.CrestLargeID
	v.allyID, v.allyCrestID = c.AllyID, c.AllyCrestID
	v.userRelation = social.UserInfoRelation(c.LeaderID == p.ObjectID(), p.srv.siegeState(p)) // siege_zone.go
	v.privileges = int32(p.clan.privs)
	return v
}

// clanRelationBits are the clan bits of p's relation to v (Mobius
// Player.getRelation): the leader bit and the war bits.
func (s *Server) clanRelationBits(p, v *Player) int32 {
	pc, vc := p.clan.clan, v.clan.clan
	info := func(q *Player) social.RelationInfo {
		return social.RelationInfo{ClanLeader: q.IsClanLeader(), InClan: q.clan.clan != nil, Unit: q.PledgeType()}
	}
	targetAtWar := pc != nil && vc != nil && vc.AtWarWith(pc.ID)
	atWar := pc != nil && vc != nil && pc.AtWarWith(vc.ID)
	return social.Relation(info(p), info(v), targetAtWar, atWar)
}

// ----- packets -----

// clanInfo is the clan as the clan window packets show it.
func clanInfo(c *Clan) serverpackets.ClanInfo {
	return serverpackets.ClanInfo{
		ID:          c.ID,
		Name:        c.Name,
		LeaderName:  c.LeaderName(),
		CrestID:     c.CrestID,
		Level:       int32(c.Level), //nolint:gosec // small
		CastleID:    c.CastleID,
		HideoutID:   c.HideoutID,
		Rank:        int32(c.Rank),       //nolint:gosec // small
		Reputation:  int32(c.Reputation), //nolint:gosec // capped
		AllyID:      c.AllyID,
		AllyName:    c.AllyName,
		AllyCrestID: c.AllyCrestID,
		AtWar:       c.AtWar(),
	}
}

// pledgeMember is a member as the member list packets show it.
func pledgeMember(m *ClanMember) serverpackets.PledgeMember {
	return serverpackets.PledgeMember{
		Name:       m.Name(),
		Level:      int32(m.Level()), //nolint:gosec // small
		ClassID:    m.ClassID(),
		Female:     m.Female(),
		Race:       m.Race(),
		ObjectID:   m.ObjectID(),
		Online:     m.Online(),
		PledgeType: int32(m.PledgeType()), //nolint:gosec // small
		HasSponsor: m.Sponsor() != 0,
	}
}

// memberUpdatePacket is PledgeShowMemberListUpdate of a member (its
// sponsor flag only for academy members).
func memberUpdatePacket(m *ClanMember) []byte {
	pm := pledgeMember(m)
	pm.HasSponsor = m.PledgeType() == social.UnitAcademy && m.Sponsor() != 0
	return serverpackets.NewPledgeShowMemberListUpdatePacket(pm)
}

// sendClanWindow sends p the clan window (Mobius PledgeShowMemberListAll
// with what it sends from inside writeImpl): the main clan's members, a
// PledgeReceiveSubPledgeCreated per unit, a PledgeShowMemberListAdd per
// unit member, then UserInfo.
func (s *Server) sendClanWindow(p *Player, c *Clan) {
	var main []serverpackets.PledgeMember
	for _, m := range c.members {
		if m.PledgeType() == social.UnitMain {
			main = append(main, pledgeMember(m))
		}
	}
	p.Send(serverpackets.NewPledgeShowMemberListAllPacket(clanInfo(c), main))
	for _, u := range c.Units {
		leader := ""
		if u.Type != storage.UnitAcademy && u.LeaderID != 0 {
			if m := c.member(u.LeaderID); m != nil {
				leader = m.Name()
			}
		}
		p.Send(serverpackets.NewPledgeReceiveSubPledgeCreatedPacket(serverpackets.SubPledge{ID: u.Type, Name: u.Name, LeaderName: leader}))
	}
	for _, m := range c.members {
		if m.PledgeType() != social.UnitMain {
			p.Send(serverpackets.NewPledgeShowMemberListAddPacket(pledgeMember(m)))
		}
	}
	p.UpdateUserInfo()
}

// broadcastClanStatus sends every member online an empty clan window and
// the new one (Mobius broadcastClanStatus).
func (s *Server) broadcastClanStatus(c *Clan) {
	for _, q := range c.OnlineMembers() {
		q.Send(serverpackets.NewPledgeShowMemberListDeleteAllPacket())
		s.sendClanWindow(q, c)
	}
}

// broadcastToClan sends body to the members online (Mobius
// broadcastToOnlineMembers).
func broadcastToClan(c *Clan, body []byte) {
	for _, q := range c.OnlineMembers() {
		q.Send(body)
	}
}

// broadcastToOtherClanMembers sends body to the members online but p
// (Mobius broadcastToOtherOnlineMembers).
func broadcastToOtherClanMembers(c *Clan, body []byte, p *Player) {
	for _, q := range c.OnlineMembers() {
		if q != p {
			q.Send(body)
		}
	}
}

// broadcastClanInfo sends the members online PledgeShowInfoUpdate.
func (s *Server) broadcastClanInfo(c *Clan) {
	broadcastToClan(c, serverpackets.NewPledgeShowInfoUpdatePacket(clanInfo(c)))
}

// clanSysMsg builds a system message.
func clanSysMsg(id serverpackets.SystemMessageID, params ...serverpackets.SystemMessageParam) []byte {
	return serverpackets.NewSystemMessagePacket(serverpackets.SystemMessage{ID: id, Params: params})
}

// ----- reputation and ranks -----

// AddClanReputation changes c's reputation by delta (Mobius
// addReputationScore / takeReputationScore): capped at ±100,000,000; the
// members online are told when clan skills switch off or on, and get
// PledgeShowInfoUpdate. The clan is saved. Loop-only.
func (s *Server) AddClanReputation(c *Clan, delta int) {
	s.setClanReputation(c, delta)
	s.saveClan(c)
}

// setClanReputation is Mobius setReputationScore without the save.
func (s *Server) setClanReputation(c *Clan, delta int) {
	score, off, on := social.ReputationChange(c.Reputation, delta)
	switch {
	case off:
		broadcastToClan(c, clanSysMsg(clanMsgSkillsDeactivated))
		s.clanSkillsStatus(c, true)
	case on:
		broadcastToClan(c, clanSysMsg(clanMsgSkillsActivated))
		s.clanSkillsStatus(c, false)
	}
	c.Reputation = score
	s.broadcastClanInfo(c)
}

// clanSkillsHook switches the clan skills of the members online off or on
// (Mobius skillsStatus); the clan skills (Phase 7b) set it.
var clanSkillsHook func(s *Server, p *Player, disable bool)

func (s *Server) clanSkillsStatus(c *Clan, disable bool) {
	if clanSkillsHook == nil {
		return
	}
	for _, q := range c.OnlineMembers() {
		clanSkillsHook(s, q, disable)
	}
}

// updateClanRanks recalculates the rank of every clan (Mobius
// ClanTable.updateClanRanks, every 20 minutes).
func (s *Server) updateClanRanks() {
	list := sortedClans(s.clan.clans)
	standings := make([]social.ClanStanding, len(list))
	for i, c := range list {
		standings[i] = social.ClanStanding{Level: c.Level, Reputation: c.Reputation}
	}
	for i, r := range social.ClanRanks(standings) {
		list[i].Rank = r
	}
}

// clanRank is ClanTable.getClanRank for one clan.
func (s *Server) clanRank(c *Clan) int {
	list := sortedClans(s.clan.clans)
	standings := make([]social.ClanStanding, len(list))
	at := 0
	for i, o := range list {
		standings[i] = social.ClanStanding{Level: o.Level, Reputation: o.Reputation}
		if o == c {
			at = i
		}
	}
	return social.ClanRank(standings, at)
}

// clanClassLevel is the class's transfer level (PlayerClass.level).
func clanClassLevel(classID int32) int { return data.PlayerClass(classID).Level() }
