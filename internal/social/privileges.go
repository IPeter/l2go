package social

import "strconv"

// Access is Mobius ClanAccess: one clan privilege. Its value is the enum's
// ordinal and its bit in a Privileges mask is 1 << value. The order is
// Mobius's and must stay: the client shows the bits.
type Access int

// The clan privileges, in Mobius order.
const (
	AccessNone Access = iota
	AccessInviteMember
	AccessAssignTitle
	AccessWarehouse
	AccessModifyRanks
	AccessWarDeclaration
	AccessRemoveMember
	AccessChangeCrest
	AccessDismissMentee
	AccessMemberFame
	AccessAirship
	AccessHallOpenDoor
	AccessHallFunctions
	AccessHallAuction
	AccessHallBanish
	AccessHallManageFunctions
	AccessCastleOpenDoor
	AccessCastleManor
	AccessCastleSiege
	AccessCastleFunctions
	AccessCastleBanish
	AccessCastleVault
	AccessCastleMercenaries
	AccessCastleManageFunctions

	accessCount // the number of privileges (24)
)

var accessNames = [...]string{
	"NONE", "INVITE_MEMBER", "ASSIGN_TITLE", "ACCESS_WAREHOUSE", "MODIFY_RANKS",
	"WAR_DECLARATION", "REMOVE_MEMBER", "CHANGE_CREST", "DISMISS_MENTEE", "MEMBER_FAME",
	"ACCESS_AIRSHIP", "HALL_OPEN_DOOR", "HALL_FUNCTIONS", "HALL_AUCTION", "HALL_BANISH",
	"HALL_MANAGE_FUNCTIONS", "CASTLE_OPEN_DOOR", "CASTLE_MANOR", "CASTLE_SIEGE",
	"CASTLE_FUNCTIONS", "CASTLE_BANISH", "CASTLE_VAULT", "CASTLE_MERCENARIES",
	"CASTLE_MANAGE_FUNCTIONS",
}

// String returns Mobius's name of a.
func (a Access) String() string {
	if a < 0 || a >= accessCount {
		return "Access(" + strconv.Itoa(int(a)) + ")"
	}
	return accessNames[a]
}

// Mask is ClanAccess.getMask: the privilege's bit.
func (a Access) Mask() Privileges { return 1 << a }

// Privileges is a ClanPrivileges mask, as stored per rank and sent to the
// client.
type Privileges int32

// AllPrivileges is ClanPrivileges.getCompleteMask: every bit of the 24
// privileges (0xFFFFFF). The clan leader has it.
const AllPrivileges Privileges = 1<<accessCount - 1

// Rank9Allowed are the only privileges rank 9 (the academy rank) may hold
// (RequestPledgePower): warehouse, clan hall door and castle door.
const Rank9Allowed = Privileges(1<<AccessWarehouse | 1<<AccessHallOpenDoor | 1<<AccessCastleOpenDoor)

// Has is ClanPrivileges.hasMinimumPrivileges: every given privilege is
// set.
func (p Privileges) Has(a Access, more ...Access) bool {
	if p&a.Mask() == 0 {
		return false
	}
	for _, b := range more {
		if p&b.Mask() == 0 {
			return false
		}
	}
	return true
}

// With returns p with the given privileges added (addMinimumPrivileges).
func (p Privileges) With(a ...Access) Privileges {
	for _, b := range a {
		p |= b.Mask()
	}
	return p
}

// Without returns p with the given privileges removed
// (removeMinimumPrivileges).
func (p Privileges) Without(a ...Access) Privileges {
	for _, b := range a {
		p &^= b.Mask()
	}
	return p
}

// RankPrivileges is the mask RequestPledgePower (action 2) stores for rank:
// rank 9 keeps only Rank9Allowed, other ranks keep privs as sent (Mobius
// stores any bits the client sends).
func RankPrivileges(rank int, privs Privileges) Privileges {
	if rank == RankAcademy {
		return privs & Rank9Allowed
	}
	return privs
}

// RankTable is a clan's privileges per rank: index rank − 1 for ranks 1
// to 9 (Clan._privs, initialised to none).
type RankTable [Ranks]Privileges

// Of is Clan.getRankPrivs: the privileges of rank, none for a rank outside
// 1–9.
func (t *RankTable) Of(rank int) Privileges {
	if rank < 1 || rank > Ranks {
		return 0
	}
	return t[rank-1]
}

// Set stores privs for rank (Clan.setRankPrivs); ranks outside 1–9 are
// ignored.
func (t *RankTable) Set(rank int, privs Privileges) {
	if rank >= 1 && rank <= Ranks {
		t[rank-1] = privs
	}
}

// MemberAccess is what Player.restore gives a clan member at login: the
// leader gets every privilege and grade 1; others keep their grade (5 when
// stored as 0) and get that rank's privileges.
func MemberAccess(leader bool, grade int, ranks *RankTable) (Privileges, int) {
	if leader {
		return AllPrivileges, RankLeader
	}
	if grade == 0 {
		grade = RankNewMember
	}
	return ranks.Of(grade), grade
}
