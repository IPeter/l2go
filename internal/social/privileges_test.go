package social

import "testing"

func TestAccessBits(t *testing.T) {
	tests := []struct {
		a    Access
		mask Privileges
		name string
	}{
		{AccessNone, 0x1, "NONE"},
		{AccessInviteMember, 0x2, "INVITE_MEMBER"},
		{AccessAssignTitle, 0x4, "ASSIGN_TITLE"},
		{AccessWarehouse, 0x8, "ACCESS_WAREHOUSE"},
		{AccessModifyRanks, 0x10, "MODIFY_RANKS"},
		{AccessWarDeclaration, 0x20, "WAR_DECLARATION"},
		{AccessRemoveMember, 0x40, "REMOVE_MEMBER"},
		{AccessChangeCrest, 0x80, "CHANGE_CREST"},
		{AccessHallOpenDoor, 0x800, "HALL_OPEN_DOOR"},
		{AccessCastleOpenDoor, 0x10000, "CASTLE_OPEN_DOOR"},
		{AccessCastleManageFunctions, 0x800000, "CASTLE_MANAGE_FUNCTIONS"},
	}
	for _, tt := range tests {
		if tt.a.Mask() != tt.mask || tt.a.String() != tt.name {
			t.Errorf("%v: mask %#x name %s, want %#x %s", int(tt.a), tt.a.Mask(), tt.a, tt.mask, tt.name)
		}
	}
	if AllPrivileges != 0xFFFFFF {
		t.Errorf("AllPrivileges = %#x", AllPrivileges)
	}
	if Access(24).String() != "Access(24)" {
		t.Error("Access.String out of range")
	}
}

func TestRankPrivileges(t *testing.T) {
	tests := []struct {
		rank int
		in   Privileges
		want Privileges
	}{
		{9, AllPrivileges, 0x10808},
		{9, AccessInviteMember.Mask() | AccessWarehouse.Mask(), AccessWarehouse.Mask()},
		{9, 0, 0},
		{8, AllPrivileges, AllPrivileges},
		{5, AccessInviteMember.Mask(), AccessInviteMember.Mask()},
	}
	for _, tt := range tests {
		if got := RankPrivileges(tt.rank, tt.in); got != tt.want {
			t.Errorf("RankPrivileges(%d, %#x) = %#x, want %#x", tt.rank, tt.in, got, tt.want)
		}
	}
	if Rank9Allowed != 0x10808 {
		t.Errorf("Rank9Allowed = %#x", Rank9Allowed)
	}
}

func TestPrivilegesHas(t *testing.T) {
	p := Privileges(0).With(AccessInviteMember, AccessAssignTitle)
	if !p.Has(AccessInviteMember) || !p.Has(AccessInviteMember, AccessAssignTitle) {
		t.Error("Has missed a set bit")
	}
	if p.Has(AccessWarehouse) || p.Has(AccessInviteMember, AccessWarehouse) {
		t.Error("Has found an unset bit")
	}
	if p.Without(AccessInviteMember) != AccessAssignTitle.Mask() {
		t.Error("Without")
	}
	if !AllPrivileges.Has(AccessCastleManageFunctions, AccessNone) {
		t.Error("AllPrivileges lacks a bit")
	}
}

func TestMemberAccess(t *testing.T) {
	var ranks RankTable
	ranks.Set(3, AccessInviteMember.Mask())
	ranks.Set(5, AccessWarehouse.Mask())
	ranks.Set(10, AllPrivileges) // ignored
	tests := []struct {
		leader    bool
		grade     int
		wantPrivs Privileges
		wantGrade int
	}{
		{true, 5, AllPrivileges, 1},
		{true, 0, AllPrivileges, 1},
		{false, 3, AccessInviteMember.Mask(), 3},
		{false, 0, AccessWarehouse.Mask(), 5},
		{false, 9, 0, 9},
	}
	for _, tt := range tests {
		p, g := MemberAccess(tt.leader, tt.grade, &ranks)
		if p != tt.wantPrivs || g != tt.wantGrade {
			t.Errorf("MemberAccess(%v, %d) = %#x %d, want %#x %d", tt.leader, tt.grade, p, g, tt.wantPrivs, tt.wantGrade)
		}
	}
	if ranks.Of(0) != 0 || ranks.Of(10) != 0 {
		t.Error("rank outside 1–9 has privileges")
	}
}

func TestRelation(t *testing.T) {
	member := RelationInfo{InClan: true}
	academy := RelationInfo{InClan: true, Unit: UnitAcademy}
	tests := []struct {
		name              string
		p, target         RelationInfo
		targetWar, ownWar bool
		want              int32
	}{
		{"nothing", RelationInfo{}, RelationInfo{}, false, false, 0},
		{"flag karma leader", RelationInfo{PvPFlag: true, Karma: 5, ClanLeader: true}, RelationInfo{}, false, false, 0x86},
		{"one-sided", member, member, true, false, 0x10000},
		{"mutual", member, member, true, true, 0x18000},
		{"only own declaration", member, member, false, true, 0},
		{"academy self", academy, member, true, true, 0},
		{"academy target", member, academy, true, true, 0},
		{"target without clan", member, RelationInfo{}, true, true, 0},
		{"leader at war", RelationInfo{InClan: true, ClanLeader: true}, member, true, true, 0x18080},
		{"siege attacker vs defender", RelationInfo{SiegeState: 1}, RelationInfo{SiegeState: 2}, false, false, 0x1600},
		{"siege same side defender", RelationInfo{SiegeState: 2}, RelationInfo{SiegeState: 2}, false, false, 0xA00},
	}
	for _, tt := range tests {
		if got := Relation(tt.p, tt.target, tt.targetWar, tt.ownWar); got != tt.want {
			t.Errorf("%s: Relation = %#x, want %#x", tt.name, got, tt.want)
		}
	}
	for _, tt := range []struct {
		leader bool
		siege  int
		want   int32
	}{{false, 0, 0}, {true, 0, 0x40}, {true, 1, 0x1C0}, {false, 2, 0x80}} {
		if got := UserInfoRelation(tt.leader, tt.siege); got != tt.want {
			t.Errorf("UserInfoRelation(%v, %d) = %#x, want %#x", tt.leader, tt.siege, got, tt.want)
		}
	}
}

func TestWarRules(t *testing.T) {
	m := WarMember{InClan: true}
	peace := WarMember{InClan: true, WantsPeace: true}
	acad := WarMember{InClan: true, Unit: UnitAcademy, Academy: true}
	if !WarAutoAttackable(m, m, true) || WarAutoAttackable(m, m, false) {
		t.Error("WarAutoAttackable mutual war")
	}
	if WarAutoAttackable(peace, m, true) || WarAutoAttackable(m, peace, true) {
		t.Error("WarAutoAttackable with peace")
	}
	if WarAutoAttackable(acad, m, true) || !WarAutoAttackable(m, acad, true) {
		t.Error("WarAutoAttackable academy: only the target's academy state counts")
	}
	if !WarPvPKill(m, m, true) || WarPvPKill(m, acad, true) || WarPvPKill(acad, m, true) || WarPvPKill(m, m, false) {
		t.Error("WarPvPKill")
	}
	if !AtWar(m, m, true) || AtWar(m, m, false) || AtWar(acad, m, true) || AtWar(m, WarMember{}, true) {
		t.Error("AtWar")
	}
}
