package social

// The RelationChanged bits (RelationChanged.RELATION_*).
const (
	RelationPvPFlag   int32 = 0x00002
	RelationHasKarma  int32 = 0x00004
	RelationLeader    int32 = 0x00080
	RelationInSiege   int32 = 0x00200
	RelationAttacker  int32 = 0x00400
	RelationAlly      int32 = 0x00800 // blue siege icon
	RelationEnemy     int32 = 0x01000 // red siege icon
	RelationMutualWar int32 = 0x08000 // double fist
	RelationOneSided  int32 = 0x10000 // single fist
)

// RelationInfo is a player as Player.getRelation sees it.
type RelationInfo struct {
	// PvPFlag is true while the player is flagged.
	PvPFlag bool
	// Karma is the player's karma.
	Karma int
	// ClanLeader is true for a clan leader.
	ClanLeader bool
	// SiegeState is 0 outside sieges, 1 attacker, 2 defender (M11).
	SiegeState int
	// InClan is true for a clan member.
	InClan bool
	// Unit is the player's pledge type (UnitAcademy excludes war bits).
	Unit int
}

// Relation is Player.getRelation: the relation of p as shown to target.
// targetClanAtWar is true when target's clan has declared war on p's clan,
// clanAtWar when p's clan has declared war on target's. The war bits need
// both in a clan and neither in the academy (by pledge type, as Mobius
// checks here): one-sided when target's clan declared, mutual when both
// did.
func Relation(p, target RelationInfo, targetClanAtWar, clanAtWar bool) int32 {
	var r int32
	if p.PvPFlag {
		r |= RelationPvPFlag
	}
	if p.Karma > 0 {
		r |= RelationHasKarma
	}
	if p.ClanLeader {
		r |= RelationLeader
	}
	if p.SiegeState != 0 {
		r |= RelationInSiege
		if p.SiegeState != target.SiegeState {
			r |= RelationEnemy
		} else {
			r |= RelationAlly
		}
		if p.SiegeState == 1 {
			r |= RelationAttacker
		}
	}
	if p.InClan && target.InClan && p.Unit != UnitAcademy && target.Unit != UnitAcademy && targetClanAtWar {
		r |= RelationOneSided
		if clanAtWar {
			r |= RelationMutualWar
		}
	}
	return r
}

// UserInfoRelation is the relation field of a player's own UserInfo
// (UserInfo constructor): 0x40 for a clan leader, 0x180 as siege attacker,
// 0x80 as defender.
func UserInfoRelation(clanLeader bool, siegeState int) int32 {
	var r int32
	if clanLeader {
		r = 0x40
	}
	switch siegeState {
	case 1:
		r |= 0x180
	case 2:
		r |= 0x80
	}
	return r
}

// WarMember is a player as the clan war combat rules see it.
type WarMember struct {
	// InClan is true for a clan member.
	InClan bool
	// Unit is the player's pledge type.
	Unit int
	// Academy is Player.isAcademyMember: joined the academy at some level
	// (lvlJoinedAcademy > 0).
	Academy bool
	// WantsPeace is the member's peace request in a war.
	WantsPeace bool
}

// WarAutoAttackable is the war rule of Player.isAutoAttackable: target can
// be attacked by attacker without Ctrl when both clans have declared war on
// each other, neither asked for peace and the target is not an academy
// member (isAcademyMember; the attacker's academy state is not checked, as
// in Mobius).
func WarAutoAttackable(target, attacker WarMember, mutualWar bool) bool {
	return target.InClan && attacker.InClan && mutualWar &&
		!target.WantsPeace && !attacker.WantsPeace && !target.Academy
}

// WarPvPKill is Player.onPlayerKill's war rule: killing a member of a clan
// at mutual war counts as PvP, not PK, unless either side is in the academy
// (by pledge type).
func WarPvPKill(killer, victim WarMember, mutualWar bool) bool {
	return killer.InClan && victim.InClan && mutualWar &&
		killer.Unit != UnitAcademy && victim.Unit != UnitAcademy
}

// AtWar is Player.isAtWarWith: p's clan has declared war on target's
// (clanAtWar), and neither is an academy member (isAcademyMember). A player
// killed by someone it is AtWar with loses a quarter of the death exp
// (calculateDeathExpPenalty(killer, victim.isAtWarWith(killer))).
func AtWar(p, target WarMember, clanAtWar bool) bool {
	return p.InClan && target.InClan && !p.Academy && !target.Academy && clanAtWar
}
