package storage

import "context"

// LocClanWarehouse is the location of an item in a clan's warehouse. Such an
// item has the clan ID as its owner (Plan 12, data model).
const LocClanWarehouse ItemLocation = "CLANWH"

// Clan is one clan (decision T1): the Mobius clan_data row with its side
// tables. Alliances have no record of their own (decision T2): every member
// clan carries the alliance fields, and the alliance ID is the ID of the
// leader clan. Times are Unix milliseconds, 0 for none (decision T14).
type Clan struct {
	ID          int32 // from the object ID counter (Mobius clan_id)
	Name        string
	Level       int32
	Reputation  int32 // Mobius reputation_score
	LeaderID    int32 // the leader's character object ID
	NewLeaderID int32 // a leader change waiting for the weekly reset, 0 for none
	// CrestID and CrestLargeID are crest IDs (CrestStore), 0 for none.
	CrestID, CrestLargeID int32

	AllyID      int32 // the alliance leader clan's ID, 0 outside an alliance
	AllyName    string
	AllyCrestID int32
	AllyPenalty AllyPenalty
	// MemberPenaltyExpiry is when the clan may accept members again after it
	// dismissed one (Mobius char_penalty_expiry_time).
	MemberPenaltyExpiry int64
	// DissolvingExpiry is when a dissolution requested by the leader takes
	// effect (Mobius dissolving_expiry_time).
	DissolvingExpiry int64

	// CastleID and HideoutID stay 0 until Milestone 11.
	CastleID, HideoutID int32

	// Ranks are the privileges of the ranks (Mobius clan_privs), Units the
	// academy, Royal Guard and Knight units (clan_subpledges), Skills the
	// clan skills (clan_skills). The stores keep the order and return nil
	// for none.
	Ranks  []ClanRank
	Units  []ClanUnit
	Skills []ClanSkill
	// Notice is the clan notice shown at login (Mobius clan_notices).
	Notice ClanNotice
	// WarsDeclared are the IDs of the clans this clan declared war on (Mobius
	// clan_wars rows with this clan as clan1), nil for none.
	WarsDeclared []int32
}

// Alliance penalty types of AllyPenalty.Type (the Mobius Clan constants).
const (
	AllyPenaltyNone          int32 = 0
	AllyPenaltyClanLeaved    int32 = 1
	AllyPenaltyClanDismissed int32 = 2
	AllyPenaltyDismissClan   int32 = 3
	AllyPenaltyDissolveAlly  int32 = 4
)

// AllyPenalty is a clan's alliance penalty (Mobius ally_penalty_expiry_time
// and ally_penalty_type).
type AllyPenalty struct {
	Expiry int64 // Unix milliseconds, 0 for none
	Type   int32 // AllyPenaltyClanLeaved, ...
}

// ClanRank is the privilege mask of one rank (a Mobius clan_privs row; its
// party column is always 0).
type ClanRank struct {
	Rank  int32 // 1-9
	Privs int32 // the ClanPrivilege bit mask
}

// Unit types of ClanUnit.Type and Character.PledgeType (Mobius Clan
// SUBUNIT_*).
const (
	UnitMain         int32 = 0
	UnitAcademy      int32 = -1
	UnitRoyalGuard1  int32 = 100
	UnitRoyalGuard2  int32 = 200
	UnitKnight1      int32 = 1001
	UnitKnight2      int32 = 1002
	UnitKnight3      int32 = 2001
	UnitKnight4      int32 = 2002
	ClanSkillAllUnit int32 = -2 // ClanSkill.Unit of a skill of the whole clan
)

// ClanUnit is one unit of a clan (a Mobius clan_subpledges row).
type ClanUnit struct {
	Type     int32 // UnitAcademy, UnitRoyalGuard1, ...
	Name     string
	LeaderID int32 // the unit leader's character object ID, 0 for none
}

// ClanSkill is one clan skill (a Mobius clan_skills row).
type ClanSkill struct {
	SkillID int32
	Level   int32
	Unit    int32 // the unit it applies to, ClanSkillAllUnit for the whole clan
}

// ClanNotice is the clan notice (a Mobius clan_notices row).
type ClanNotice struct {
	Text    string // at most 8,192 characters
	Enabled bool
}

// ClanMember is what the clan registry keeps of a member without loading
// the whole character (decision T1, Mobius Clan.restore): the clan fields
// of the character and what the member list shows.
type ClanMember struct {
	ObjectID         int32
	ClanID           int32
	Name             string
	Level            int32
	ClassID          int32
	Race             int32
	Sex              int32
	Title            string
	PowerGrade       int32
	PledgeType       int32
	Apprentice       int32
	Sponsor          int32
	LvlJoinedAcademy int32
	WantsPeace       bool
}

// ClanStore reads clans, their members and their warehouses. Clans are
// written with a Batch (BatchWriter). Names are unique ignoring case
// (NameKey).
type ClanStore interface {
	// LoadClans returns every clan in ID order, nil if there is none.
	LoadClans(ctx context.Context) ([]Clan, error)
	// GetClan returns the clan or ErrNotFound.
	GetClan(ctx context.Context, id int32) (Clan, error)
	// ClanMembers returns the characters in clan clanID, or with clanID 0
	// the characters in any clan, in clan ID and then object ID order. It
	// reads only the fields of ClanMember.
	ClanMembers(ctx context.Context, clanID int32) ([]ClanMember, error)
	// ClanItems returns the items the clan owns, its warehouse, normalized
	// and in object ID order, nil if there is none.
	ClanItems(ctx context.Context, clanID int32) ([]Item, error)
}

// Crest types of Crest.Type (Mobius CrestType).
const (
	CrestPledge      int32 = 1
	CrestPledgeLarge int32 = 2
	CrestAlly        int32 = 3
)

// Crest is a clan or alliance crest image (decision T3).
type Crest struct {
	ID   int32 // from CrestStore.NextCrestID, not the object ID space
	Type int32 // CrestPledge, CrestPledgeLarge or CrestAlly
	Data []byte
}

// CrestStore hands out crest IDs and reads crests. Crests are written with a
// Batch (BatchWriter).
type CrestStore interface {
	// NextCrestID returns a new crest ID. The IDs start at 1 and are never
	// handed out twice, also across restarts for a persistent store.
	NextCrestID(ctx context.Context) (int32, error)
	// LoadCrests returns every crest in ID order, nil if there is none.
	LoadCrests(ctx context.Context) ([]Crest, error)
}

// Relation types of Relation.Type (Mobius character_friends.relation).
const (
	RelationFriend int32 = 0
	RelationBlock  int32 = 1
)

// Relation is one row of a character's friend or block list (decision T4):
// Owner lists Target.
type Relation struct {
	Owner, Target int32
	Type          int32 // RelationFriend or RelationBlock
}

// FriendRelations returns the two relations of a friendship between a and
// b, one in each direction, as adding or removing a friend writes them.
func FriendRelations(a, b int32) []Relation {
	return []Relation{{Owner: a, Target: b, Type: RelationFriend}, {Owner: b, Target: a, Type: RelationFriend}}
}

// RelationStore reads the friend and block lists. They are written with a
// Batch (BatchWriter); deleting a character removes its relations both ways.
type RelationStore interface {
	// Relations returns the relations owner has, by type and then target.
	Relations(ctx context.Context, owner int32) ([]Relation, error)
	// RelationsTo returns the relations other characters have to target, by
	// type and then owner.
	RelationsTo(ctx context.Context, target int32) ([]Relation, error)
}

// CharacterSave is one character's part of a Batch: what
// CharacterSaver.Save writes.
type CharacterSave struct {
	Character Character // Items and Quests are not used, as in Save
	Changes   ItemChanges
}

// OwnerItemChanges are the changes to the items of an owner that is not a
// character, such as a clan's warehouse with the clan ID as owner. They mean
// what they mean in ItemChanges.
type OwnerItemChanges struct {
	Owner     int32
	Upserts   []Item
	Destroyed []int32
	Left      []int32
}

// PatchFields selects the fields a CharacterPatch writes.
type PatchFields uint32

// Fields of a CharacterPatch.
const (
	PatchClanID PatchFields = 1 << iota
	PatchTitle
	PatchPowerGrade
	PatchPledgeType
	PatchApprentice
	PatchSponsor
	PatchLvlJoinedAcademy
	PatchWantsPeace
	PatchClanJoinExpiry
	PatchClanCreateExpiry
	// PatchKarma and PatchPKKills give back an offline holder's karma and
	// PK count when its cursed weapon ends (cursedweapons.go).
	PatchKarma
	PatchPKKills

	// PatchClanMembership is every clan field and the title: what Mobius
	// Clan.removeClanMember writes for a member who leaves or is dismissed.
	PatchClanMembership = PatchClanID | PatchTitle | PatchPowerGrade | PatchPledgeType | PatchApprentice |
		PatchSponsor | PatchLvlJoinedAcademy | PatchWantsPeace | PatchClanJoinExpiry | PatchClanCreateExpiry
)

// PatchValues are the values a CharacterPatch can write; see Character.
type PatchValues struct {
	ClanID           int32
	Title            string
	PowerGrade       int32
	PledgeType       int32
	Apprentice       int32
	Sponsor          int32
	LvlJoinedAcademy int32
	WantsPeace       bool
	ClanJoinExpiry   int64
	ClanCreateExpiry int64
	Karma            int32
	PKKills          int32
}

// CharacterPatch changes some fields of a character that may be offline
// (decision T6): a dismissal, a leader change, a rank or unit change.
type CharacterPatch struct {
	ObjectID int32
	Fields   PatchFields // which of Values are written
	Values   PatchValues
}

// Apply writes the patch's fields into c.
func (p CharacterPatch) Apply(c *Character) {
	v := p.Values
	set := func(f PatchFields) bool { return p.Fields&f != 0 }
	if set(PatchClanID) {
		c.ClanID = v.ClanID
	}
	if set(PatchTitle) {
		c.Title = v.Title
	}
	if set(PatchPowerGrade) {
		c.PowerGrade = v.PowerGrade
	}
	if set(PatchPledgeType) {
		c.PledgeType = v.PledgeType
	}
	if set(PatchApprentice) {
		c.Apprentice = v.Apprentice
	}
	if set(PatchSponsor) {
		c.Sponsor = v.Sponsor
	}
	if set(PatchLvlJoinedAcademy) {
		c.LvlJoinedAcademy = v.LvlJoinedAcademy
	}
	if set(PatchWantsPeace) {
		c.WantsPeace = v.WantsPeace
	}
	if set(PatchClanJoinExpiry) {
		c.ClanJoinExpiry = v.ClanJoinExpiry
	}
	if set(PatchClanCreateExpiry) {
		c.ClanCreateExpiry = v.ClanCreateExpiry
	}
	if set(PatchKarma) {
		c.Karma = v.Karma
	}
	if set(PatchPKKills) {
		c.PKKills = v.PKKills
	}
}

// AllyCrest sets the alliance crest of every clan of an alliance (decision
// T2, Mobius Clan.changeAllyCrest).
type AllyCrest struct {
	AllyID  int32
	CrestID int32
}

// RelationChanges are added and removed relations.
type RelationChanges struct {
	Added   []Relation // adding a stored relation changes nothing
	Removed []Relation // removing a missing relation changes nothing
}

// Batch is a set of writes the stores apply in one transaction: all of it
// or none of it (decisions T5 and T6). It is the joint save of a trade, a
// store purchase or a clan warehouse move (the pending changes of every
// owner in the exchange), and it carries clan, crest and relation writes
// and patches to offline characters.
//
// The parts apply in this order: the character saves in their order, the
// other owners' item changes, the clan deletes and then the clan documents,
// the alliance crests, the crest deletes and then the crests, the removed
// and then the added relations, the patches in their order, so a patch
// wins over a save of the same character in the same batch, and last the
// cursed weapon deletes and rows.
type Batch struct {
	Saves []CharacterSave
	Items []OwnerItemChanges

	// DeletedClans are removed with their warehouse items, and with their
	// IDs taken out of the other clans' WarsDeclared (Mobius
	// ClanTable.destroyClan).
	DeletedClans []int32
	// Clans are stored whole: inserted, or replacing the stored clan with
	// the same ID.
	Clans      []Clan
	AllyCrests []AllyCrest

	DeletedCrests []int32
	Crests        []Crest // inserted, or replacing the crest with the same ID

	Relations RelationChanges

	// Patches to characters that do not exist are skipped.
	Patches []CharacterPatch

	// DeletedCursedWeapons are the item IDs of cursed weapon rows to remove;
	// CursedWeapons are stored whole, replacing the row of the same item ID
	// (cursedweapons.go). They apply after the patches, deletes first.
	DeletedCursedWeapons []int32
	CursedWeapons        []CursedWeapon

	// Castles are stored whole, replacing the castle of the same ID
	// (residence.go). They apply last.
	Castles []Castle

	// ClanHalls are stored whole, replacing the hall of the same ID;
	// DeletedAuctions are removed, then Auctions stored whole, replacing the
	// auction of the same ID (clanhalls.go). They apply after the castles.
	ClanHalls       []ClanHall
	DeletedAuctions []int32
	Auctions        []Auction

	// ManorCastles are stored whole, replacing the lists of the same castle;
	// ManorState, when set, replaces the manor's mode (manor.go). They apply
	// last.
	ManorCastles []ManorCastle
	ManorState   *ManorState
}

// Empty reports whether b writes nothing.
func (b *Batch) Empty() bool {
	return len(b.Saves) == 0 && len(b.Items) == 0 && len(b.DeletedClans) == 0 && len(b.Clans) == 0 &&
		len(b.AllyCrests) == 0 && len(b.DeletedCrests) == 0 && len(b.Crests) == 0 &&
		len(b.Relations.Added) == 0 && len(b.Relations.Removed) == 0 && len(b.Patches) == 0 &&
		len(b.DeletedCursedWeapons) == 0 && len(b.CursedWeapons) == 0 && len(b.Castles) == 0 &&
		len(b.ClanHalls) == 0 && len(b.DeletedAuctions) == 0 && len(b.Auctions) == 0 &&
		len(b.ManorCastles) == 0 && b.ManorState == nil
}

// BatchWriter writes a Batch in one transaction. Both stores implement it
// on their character store, next to CharacterSaver.
type BatchWriter interface {
	// WriteBatch applies b. It returns ErrNotFound if a saved character
	// does not exist and ErrExists if a clan name is taken by another clan,
	// ignoring case, and then writes nothing.
	WriteBatch(ctx context.Context, b Batch) error
}
