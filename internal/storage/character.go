package storage

import (
	"context"
	"strings"
	"time"
)

// FirstObjectID is the first object ID an IDAllocator hands out. It matches
// L2J, which keeps lower IDs for static objects such as NPCs and doors.
const FirstObjectID int32 = 0x10000000

// FirstRuntimeObjectID is the first ID of the runtime range: NPCs and other
// objects that are never stored take IDs from here up (decision W3, the
// world package's FirstRuntimeID). Persistent IDs stay below it.
const FirstRuntimeObjectID int32 = 0x60000000

// PaperdollSlot is the equipment slot an item is worn in. The values match
// the PAPERDOLL_* indexes of the Mobius Inventory class.
//
// The zero value is PaperdollUnder, so items that are only carried must have
// their slot set to PaperdollNone explicitly.
type PaperdollSlot int8

// Paperdoll slots.
const (
	PaperdollNone    PaperdollSlot = -1 // in the inventory, not equipped
	PaperdollUnder   PaperdollSlot = 0
	PaperdollLEar    PaperdollSlot = 1
	PaperdollREar    PaperdollSlot = 2
	PaperdollNeck    PaperdollSlot = 3
	PaperdollLFinger PaperdollSlot = 4
	PaperdollRFinger PaperdollSlot = 5
	PaperdollHead    PaperdollSlot = 6
	PaperdollRHand   PaperdollSlot = 7
	PaperdollLHand   PaperdollSlot = 8
	PaperdollGloves  PaperdollSlot = 9
	PaperdollChest   PaperdollSlot = 10
	PaperdollLegs    PaperdollSlot = 11
	PaperdollFeet    PaperdollSlot = 12
	PaperdollCloak   PaperdollSlot = 13
	PaperdollFace    PaperdollSlot = 14
	PaperdollHair    PaperdollSlot = 15
	PaperdollHair2   PaperdollSlot = 16
)

// PaperdollTotalSlots is the number of equipment slots.
const PaperdollTotalSlots = 17

// Equipped reports whether s is one of the equipment slots.
func (s PaperdollSlot) Equipped() bool {
	return s >= 0 && s < PaperdollTotalSlots
}

// ItemLocation is where an item is kept. The values are the names of the
// Mobius ItemLocation enum, which its items table stores in the loc column.
type ItemLocation string

// Item locations. Others (CLANWH, FREIGHT, PET, PET_EQUIP) come with the
// features that use them; the stores keep any value.
const (
	LocInventory ItemLocation = "INVENTORY" // carried, not worn
	LocPaperdoll ItemLocation = "PAPERDOLL" // worn in the paperdoll slot Item.Slot
	LocWarehouse ItemLocation = "WAREHOUSE" // in the owner's private warehouse
)

// NoMana is Item.Mana for an item without mana, which is every item except
// shadow items (Mobius mana_left -1).
const NoMana int32 = -1

// Item is one item a character owns.
//
// For an item the character carries, Slot decides whether it is worn: the
// item is in the paperdoll (LocPaperdoll) when Slot is a paperdoll slot and
// in the inventory (LocInventory) otherwise, whichever of the two Loc names.
// An empty Loc also means a carried item, so code that only knows about the
// paperdoll can keep setting Slot alone. Items kept elsewhere, such as in the
// warehouse, have no slot. Normalized makes Loc and Slot agree, and the stores
// return items normalized.
type Item struct {
	ObjectID int32 // unique across characters and items, from an IDAllocator
	ItemID   int32 // item template ID
	Count    int64
	Enchant  int32         // enchant level
	Slot     PaperdollSlot // the paperdoll slot if worn, else PaperdollNone (Mobius loc_data)
	Loc      ItemLocation  // where the item is kept (Mobius loc); see above
	// Mana is the mana a shadow item has left, in minutes, or NoMana. The
	// zero value is not NoMana: whoever creates an item sets it.
	Mana        int32
	CustomType1 int32 // Mobius custom_type1, usually 0
	CustomType2 int32 // Mobius custom_type2, usually 0
}

// Carried reports whether the character carries the item, in the inventory
// or worn, as opposed to keeping it somewhere else such as the warehouse.
func (it Item) Carried() bool {
	switch it.Loc {
	case "", LocInventory, LocPaperdoll:
		return true
	}
	return false
}

// Normalized returns it with Loc and Slot in agreement, as the stores keep
// it: a carried item gets LocPaperdoll if Slot is a paperdoll slot, and
// LocInventory with PaperdollNone otherwise; an item kept elsewhere keeps its
// Loc and gets PaperdollNone.
func (it Item) Normalized() Item {
	switch {
	case !it.Carried():
		it.Slot = PaperdollNone
	case it.Slot.Equipped():
		it.Loc = LocPaperdoll
	default:
		it.Loc, it.Slot = LocInventory, PaperdollNone
	}
	return it
}

// Character is a player character.
//
// Times are kept with millisecond precision, in UTC.
type Character struct {
	ObjectID    int32 // unique across characters and items, from an IDAllocator
	Account     string
	Name        string
	Race        int32 // 0 human, 1 elf, 2 dark elf, 3 orc, 4 dwarf
	Sex         int32 // 0 male, 1 female
	ClassID     int32
	BaseClassID int32
	HairStyle   int32
	HairColor   int32
	Face        int32
	Level       int32
	Exp         int64
	SP          int32
	Karma       int32
	PvPKills    int32
	PKKills     int32
	ClanID      int32 // 0 if not in a clan

	// DeathPenaltyLevel is the level of the Death Penalty (skill 5076), 0
	// without one (Mobius character_skills_save of 5076).
	DeathPenaltyLevel int32
	// ExpBeforeDeath is the exp the character had before its last death, 0
	// when it has not died since its last revive (Mobius expBeforeDeath).
	ExpBeforeDeath int64

	CurHP, MaxHP float64
	CurMP, MaxMP float64
	CurCP, MaxCP float64

	X, Y, Z int32
	Heading int32

	AccessLevel AccessLevel
	DeleteTime  time.Time // when the character will be deleted; zero if not deleting
	CreatedAt   time.Time
	LastAccess  time.Time // creation or last save; zero if never set

	// Items are all the character's items: the carried ones and those in
	// its warehouse. The stores keep items apart from the character (one
	// MongoDB document per item) and write a character together with its
	// items in one transaction. Get and ListByAccount return the items
	// normalized (Item.Normalized) and in object ID order, which is the
	// order they were created in.
	Items []Item

	// Vars are the character's variables (Mobius PlayerVariables, the
	// character_variables table, decision Q15): string values by name. The
	// stores keep them in the character document and return nil when there
	// are none.
	Vars map[string]string

	// Quests are the character's quest documents (decision Q4), one per
	// quest. The stores keep them apart from the character, write them in
	// the same transaction as the character and its items, and return them
	// in name order, nil when there are none.
	Quests []Quest
}

// Quest is the stored state of one quest of a character: the rows of the
// L2J character_quests table for (charId, name), "<state>" included, as a
// map from var to value. The quest package gives these rows their meaning.
type Quest struct {
	Name string            // the quest's script name, e.g. "Q00401_PathOfTheWarrior"
	Vars map[string]string // var → value; the stores return nil for none
}

// QuestChanges are the changes to a character's quests since its last
// save: the full documents of the quests that changed and the names of the
// quests that no longer have one.
type QuestChanges struct {
	// Upserts replace the stored documents of these quests, or add them.
	// Of a name listed twice, the last one counts.
	Upserts []Quest
	// Deleted are quests whose documents are deleted. An upsert of the
	// same name wins.
	Deleted []string
}

// Empty reports whether ch changes no quest.
func (ch QuestChanges) Empty() bool {
	return len(ch.Upserts) == 0 && len(ch.Deleted) == 0
}

// Merge returns the changes of ch followed by those of newer as one set of
// changes with the meaning of CharacterSaver.Save: per quest, the newer
// change wins. Neither argument is changed; the documents are shared.
func (ch QuestChanges) Merge(newer QuestChanges) QuestChanges {
	if newer.Empty() {
		return ch
	}
	if ch.Empty() {
		return newer
	}
	replaced := make(map[string]bool, len(newer.Upserts)+len(newer.Deleted))
	for _, q := range newer.Upserts {
		replaced[q.Name] = true
	}
	for _, name := range newer.Deleted {
		replaced[name] = true
	}
	var out QuestChanges
	for _, q := range ch.Upserts {
		if !replaced[q.Name] {
			out.Upserts = append(out.Upserts, q)
		}
	}
	for _, name := range ch.Deleted {
		if !replaced[name] {
			out.Deleted = append(out.Deleted, name)
		}
	}
	upserted := make(map[string]bool, len(newer.Upserts))
	for _, q := range newer.Upserts {
		upserted[q.Name] = true
	}
	out.Upserts = append(out.Upserts, newer.Upserts...)
	for _, name := range newer.Deleted {
		if !upserted[name] {
			out.Deleted = append(out.Deleted, name)
		}
	}
	return out
}

// NameKey returns the form of a character name used to compare names. Names
// that differ only in letter case have the same key.
func NameKey(name string) string {
	return strings.ToLower(name)
}

// CharacterStore stores player characters. Names are unique and compared
// case-insensitively (see NameKey). Account names are compared exactly.
//
// Create, Update and Delete write the character, its items and its quests in
// one transaction: either all of it is stored or none of it.
type CharacterStore interface {
	// Create stores c with its items, variables and quests. The caller
	// assigns the object IDs of c and its items from an IDAllocator. If
	// c.CreatedAt is zero, Create sets it to the current time. Create
	// returns ErrExists if the name is taken, ignoring case, if the object
	// ID is taken, or if an item's object ID is taken or listed twice. Of a
	// quest listed twice, the last one counts.
	Create(ctx context.Context, c *Character) error
	// ListByAccount returns the account's characters, oldest first, each
	// with all its items, variables and quests.
	ListByAccount(ctx context.Context, account string) ([]Character, error)
	// CountByAccount returns how many characters the account has.
	CountByAccount(ctx context.Context, account string) (int, error)
	// NameExists reports whether a character has this name, ignoring case.
	NameExists(ctx context.Context, name string) (bool, error)
	// Get returns the character with this object ID, with all its items,
	// variables and quests, or ErrNotFound. This is the load at character
	// select.
	Get(ctx context.Context, objectID int32) (Character, error)

	// Update saves the state of the character with c.ObjectID: position and
	// heading, HP/MP/CP, level, exp and SP, class, sex and appearance, karma
	// and kill counts, death penalty level and exp before death, clan, delete
	// time, last access, access level and variables. The
	// identity fields (account, name, race and creation time) are kept as
	// stored. c.Items replaces the character's items: those it owned that
	// c.Items does not list are deleted, and the listed ones are stored with
	// c as their owner, also when another character owned them before. Of
	// an object ID listed twice, the last one counts. c.Quests replaces the
	// character's quests the same way. Update returns ErrNotFound if there
	// is no such character.
	Update(ctx context.Context, c *Character) error
	// Delete removes the character, all its items, wherever they are kept,
	// and its quests. Its name becomes free. It returns ErrNotFound if there
	// is no such character.
	Delete(ctx context.Context, objectID int32) error
	// SetDeleteTime sets when the character will be deleted. The zero time
	// clears it (the character is restored). It returns ErrNotFound if
	// there is no such character.
	SetDeleteTime(ctx context.Context, objectID int32, t time.Time) error
}

// ItemChanges are the changes to a character's items since its last save
// (decision I2). The game keeps them per inventory, so a save writes only
// what changed.
type ItemChanges struct {
	// Upserts are the new and changed items. They are stored with the
	// character as their owner, also when another character owned them
	// before: the receiving side of a transfer.
	Upserts []Item
	// Destroyed lists items that no longer exist: sold, used up, merged
	// into a stack, expired. They are deleted whoever owns them.
	Destroyed []int32
	// Left lists items that left the character: dropped or given away.
	// They are deleted only while the character still owns them, so an
	// item another character saved first stays with that character.
	Left []int32

	// Quests are the changes to the character's quests (decision Q4),
	// written in the same transaction as the items so that quest state and
	// quest items never diverge. Whoever merges ItemChanges merges these
	// with QuestChanges.Merge.
	Quests QuestChanges
}

// CharacterSaver saves a character with only the changes to its items and
// quests, the saving side of decisions I2 and Q4. Both stores implement it
// next to CharacterStore.
type CharacterSaver interface {
	// Save stores what Update stores of c, except its items and quests:
	// c.Items and c.Quests are not used. Instead it applies changes to the
	// character's items and changes.Quests to its quests. Deletes come
	// first, so an object ID in changes.Upserts wins over the same ID in
	// Destroyed or Left, and of an object ID listed twice in Upserts the
	// last one counts; quests follow the same rules by name. Everything is
	// one transaction. Save returns ErrNotFound if there is no such
	// character.
	Save(ctx context.Context, c *Character, changes ItemChanges) error
}

// VarChanges are changes to a set of string variables, such as the global
// variables.
type VarChanges struct {
	Set     map[string]string // new and changed variables
	Deleted []string          // removed variables; Set wins for the same name
}

// Empty reports whether ch changes nothing.
func (ch VarChanges) Empty() bool {
	return len(ch.Set) == 0 && len(ch.Deleted) == 0
}

// GlobalVariableStore stores the server's global variables (Mobius
// GlobalVariablesManager, the global_variables table, decision Q15): string
// values by name.
type GlobalVariableStore interface {
	// LoadGlobalVariables returns all global variables, nil if there are
	// none.
	LoadGlobalVariables(ctx context.Context) (map[string]string, error)
	// SaveGlobalVariables applies changes in one write: deletes first, then
	// the set variables.
	SaveGlobalVariables(ctx context.Context, changes VarChanges) error
}

// Deleting reports whether the character is marked for deletion.
func (c *Character) Deleting() bool { return !c.DeleteTime.IsZero() }

// DeleteExpired reports whether the character is marked for deletion and
// its delete time has passed at now (Mobius CharSelectionInfo deletes it
// then).
func (c *Character) DeleteExpired(now time.Time) bool {
	return c.Deleting() && now.After(c.DeleteTime)
}

// IDAllocator hands out object IDs for characters and items. IDs are unique
// across both, and across restarts for persistent implementations. They
// start at FirstObjectID and stay below FirstRuntimeObjectID. The world loop
// takes IDs from an IDPool instead, which never waits for the allocator.
type IDAllocator interface {
	// NextID returns one new ID.
	NextID(ctx context.Context) (int32, error)
	// NextIDs returns n new IDs, for example for a character and its items.
	// n must be at least 1.
	NextIDs(ctx context.Context, n int) ([]int32, error)
}
