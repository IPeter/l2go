package social

// MaxClanLevel is the highest clan level Interlude reaches (Clan.levelUpClan
// stops at 8).
const MaxClanLevel = 8

// MinLevelToCreateClan is the character level needed to found a clan
// (ClanTable.createClan).
const MinLevelToCreateClan = 10

// The clan units (pledge types, Clan.SUBUNIT_*).
const (
	UnitMain    = 0
	UnitAcademy = -1
	UnitRoyal1  = 100
	UnitRoyal2  = 200
	UnitKnight1 = 1001
	UnitKnight2 = 1002
	UnitKnight3 = 2001
	UnitKnight4 = 2002
)

// Items a clan level-up consumes (Clan.levelUpClan).
const (
	ItemBloodMark         = 1419 // level 2 → 3
	ItemAllianceManifesto = 3874 // level 3 → 4
	ItemSealOfAspiration  = 3870 // level 4 → 5
)

// Clan ranks (power grades) are 1 to 9.
const (
	// RankLeader is the leader's grade (Player.restore).
	RankLeader = 1
	// RankNewMember is a new main-clan member's grade, also given at login
	// to members stored with grade 0 (RequestAnswerJoinPledge,
	// Player.restore).
	RankNewMember = 5
	// RankAcademy is an academy member's grade.
	RankAcademy = 9
	// Ranks is the number of ranks.
	Ranks = 9
)

// ClanRules is the clan configuration the rules need (Feature.ini 309–385,
// Player.ini 584–637).
type ClanRules struct {
	// LevelCost[i] and LevelMembers[i] are the reputation cost and the
	// member count needed to reach level 6+i (ClanLevel6Cost..8,
	// ClanLevel6Requirement..8).
	LevelCost    [3]int
	LevelMembers [3]int
	// RoyalGuardCost and KnightUnitCost are the reputation a Royal Guard or
	// Order of Knights unit costs (CreateRoyalGuardCost,
	// CreateKnightUnitCost).
	RoyalGuardCost, KnightUnitCost int
	// ReputationPerKill is ReputationScorePerKill (clan war kills).
	ReputationPerKill int
	// AcademyMinPoints and AcademyMaxPoints are CompleteAcademyMinPoints
	// and CompleteAcademyMaxPoints (graduation reputation).
	AcademyMinPoints, AcademyMaxPoints int
}

// DefaultClanRules returns Mobius's shipped values.
func DefaultClanRules() ClanRules {
	return ClanRules{
		LevelCost:         [3]int{5000, 10000, 20000},
		LevelMembers:      [3]int{30, 50, 80},
		RoyalGuardCost:    5000,
		KnightUnitCost:    10000,
		ReputationPerKill: 1,
		AcademyMinPoints:  190,
		AcademyMaxPoints:  650,
	}
}

// LevelUpCost is what raising a clan by one level takes from the leader
// and the clan (Clan.levelUpClan).
type LevelUpCost struct {
	// SP and Adena are taken from the leader.
	SP, Adena int64
	// ItemID is one item taken from the leader, or 0.
	ItemID int32
	// Reputation is taken from the clan.
	Reputation int
	// Members is the clan's member count needed (all units, academy
	// included); nothing is taken.
	Members int
}

// LevelUpCost returns the cost of raising a clan from level to level+1, or
// false at MaxClanLevel and above (Mobius answers false without a message).
func (r ClanRules) LevelUpCost(level int) (LevelUpCost, bool) {
	switch level {
	case 0:
		return LevelUpCost{SP: 20000, Adena: 650000}, true
	case 1:
		return LevelUpCost{SP: 100000, Adena: 2500000}, true
	case 2:
		return LevelUpCost{SP: 350000, ItemID: ItemBloodMark}, true
	case 3:
		return LevelUpCost{SP: 1000000, ItemID: ItemAllianceManifesto}, true
	case 4:
		return LevelUpCost{SP: 2500000, ItemID: ItemSealOfAspiration}, true
	case 5, 6, 7:
		i := level - 5
		return LevelUpCost{Reputation: r.LevelCost[i], Members: r.LevelMembers[i]}, true
	}
	return LevelUpCost{}, false
}

// LevelUpHave is what the leader and the clan have, for LevelUpCost.Met.
type LevelUpHave struct {
	// SP and Adena are the leader's.
	SP, Adena int64
	// HasItem is true when the leader holds the cost's item.
	HasItem bool
	// Reputation and Members are the clan's.
	Reputation, Members int
}

// Met reports whether have covers c, with Mobius's comparisons (each at
// least the cost).
func (c LevelUpCost) Met(have LevelUpHave) bool {
	return have.SP >= c.SP && have.Adena >= c.Adena &&
		(c.ItemID == 0 || have.HasItem) &&
		have.Reputation >= c.Reputation && have.Members >= c.Members
}

// MaxMembers is Clan.getMaxNrOfMembers: how many members a unit of a clan
// of clanLevel holds. Main clan 10/15/20/30 at levels 0–3 and 40 above,
// academy 20, Royal Guard 20 (30 at level 11), Knights 10 (25 at 9–11);
// 0 for an unknown unit.
func MaxMembers(clanLevel, unit int) int {
	switch unit {
	case UnitMain:
		switch clanLevel {
		case 0:
			return 10
		case 1:
			return 15
		case 2:
			return 20
		case 3:
			return 30
		}
		return 40
	case UnitAcademy:
		return 20
	case UnitRoyal1, UnitRoyal2:
		if clanLevel == 11 {
			return 30
		}
		return 20
	case UnitKnight1, UnitKnight2, UnitKnight3, UnitKnight4:
		if clanLevel >= 9 && clanLevel <= 11 {
			return 25
		}
		return 10
	}
	return 0
}

// IsRoyalGuard reports whether unit is a Royal Guard unit.
func IsRoyalGuard(unit int) bool { return unit == UnitRoyal1 || unit == UnitRoyal2 }

// IsKnights reports whether unit is an Order of Knights unit.
func IsKnights(unit int) bool {
	return unit == UnitKnight1 || unit == UnitKnight2 || unit == UnitKnight3 || unit == UnitKnight4
}

// UnitMinClanLevel is the clan level a village master asks before creating
// a unit of the kind of unit (VillageMaster: academy 5, Royal Guard 6,
// Knights 7), or 0 for the main clan.
func UnitMinClanLevel(unit int) int {
	switch {
	case unit == UnitAcademy:
		return 5
	case IsRoyalGuard(unit):
		return 6
	case IsKnights(unit):
		return 7
	}
	return 0
}

// UnitCost is the reputation a new unit costs (Clan.createSubPledge):
// none for the academy, RoyalGuardCost for units below UnitKnight1,
// KnightUnitCost above UnitRoyal2.
func (r ClanRules) UnitCost(unit int) int {
	switch {
	case unit == UnitAcademy:
		return 0
	case unit < UnitKnight1:
		return r.RoyalGuardCost
	}
	return r.KnightUnitCost
}

// AvailableUnit is Clan.getAvailablePledgeTypes: the unit ID a request for
// unit creates, given the units the clan has, or 0 when every unit of that
// kind exists. A Royal Guard request moves on from UnitRoyal1 to
// UnitRoyal2, a Knights request from UnitKnight1 through UnitKnight4.
func AvailableUnit(unit int, exists func(unit int) bool) int {
	for exists(unit) {
		switch unit {
		case UnitRoyal1:
			unit = UnitRoyal2
		case UnitKnight1:
			unit = UnitKnight2
		case UnitKnight2:
			unit = UnitKnight3
		case UnitKnight3:
			unit = UnitKnight4
		case UnitAcademy, UnitRoyal2, UnitKnight4:
			return 0
		default:
			return unit
		}
	}
	return unit
}

// PledgeMember is a character as ClanMember.calculatePledgeClass sees it.
type PledgeMember struct {
	// InClan is false for characters without a clan.
	InClan bool
	// ClanLevel is the clan's level.
	ClanLevel int
	// Leader is true for the clan leader.
	Leader bool
	// Unit is the character's pledge type (UnitMain, UnitAcademy, ...).
	Unit int
	// LeadsUnit is the unit the character is captain of
	// (Clan.getLeaderSubPledge), or 0.
	LeadsUnit int
	// Noble and Hero raise the class to at least 5 and 8.
	Noble, Hero bool
}

// pledgeRow is one clan level's classes in calculatePledgeClass.
type pledgeRow struct {
	academy, royal, knight, leader, royalCaptain, knightCaptain, member int
}

// pledgeRows are calculatePledgeClass's classes for clan levels 6 to 11. At
// level 6 knights have no case (0) and knight captains fall to the default
// member class.
var pledgeRows = map[int]pledgeRow{
	6:  {academy: 1, royal: 2, knight: 0, leader: 5, royalCaptain: 4, knightCaptain: 3, member: 3},
	7:  {academy: 1, royal: 3, knight: 2, leader: 7, royalCaptain: 6, knightCaptain: 5, member: 4},
	8:  {academy: 1, royal: 4, knight: 3, leader: 8, royalCaptain: 7, knightCaptain: 6, member: 5},
	9:  {academy: 1, royal: 5, knight: 4, leader: 9, royalCaptain: 8, knightCaptain: 7, member: 6},
	10: {academy: 1, royal: 6, knight: 5, leader: 10, royalCaptain: 9, knightCaptain: 8, member: 7},
	11: {academy: 1, royal: 7, knight: 6, leader: 11, royalCaptain: 10, knightCaptain: 9, member: 8},
}

// PledgeClass is ClanMember.calculatePledgeClass: the social class shown in
// UserInfo and CharInfo. Without a clan 0. Clan levels 0–3 give 1; level 4
// gives the leader 3 and other members 0 (as in Mobius); level 5 gives the
// leader 4 and everyone else 2; levels 6+ go by unit and captaincy.
// Nobles get at least 5, heroes at least 8.
func PledgeClass(m PledgeMember) int {
	c := 0
	if m.InClan {
		switch m.ClanLevel {
		case 4:
			if m.Leader {
				c = 3
			}
		case 5:
			c = 2
			if m.Leader {
				c = 4
			}
		default:
			row, ok := pledgeRows[m.ClanLevel]
			if !ok {
				c = 1
				break
			}
			switch {
			case m.Unit == UnitAcademy:
				c = row.academy
			case IsRoyalGuard(m.Unit):
				c = row.royal
			case IsKnights(m.Unit):
				c = row.knight
			case m.Unit == UnitMain:
				switch {
				case m.Leader:
					c = row.leader
				case IsRoyalGuard(m.LeadsUnit):
					c = row.royalCaptain
				case IsKnights(m.LeadsUnit):
					c = row.knightCaptain
				default:
					c = row.member
				}
			}
		}
	}
	if m.Noble && c < 5 {
		c = 5
	}
	if m.Hero && c < 8 {
		c = 8
	}
	return c
}

// CanJoinAcademy is Clan.checkClanJoinCondition's academy rule: level 40 or
// below and before the second class transfer (classLevel 0 or 1, the
// class's PlayerClass.level()).
func CanJoinAcademy(level, classLevel int) bool { return level <= 40 && classLevel < 2 }

// AcademyGraduationPoints is the reputation a clan gains when an academy
// member who joined at lvlJoined graduates (Player.setClassId): max points
// up to level 16, min points from 39, max − (lvlJoined − 16) × 20 between.
func (r ClanRules) AcademyGraduationPoints(lvlJoined int) int {
	switch {
	case lvlJoined <= 16:
		return r.AcademyMaxPoints
	case lvlJoined >= 39:
		return r.AcademyMinPoints
	}
	return r.AcademyMaxPoints - (lvlJoined-16)*20
}
