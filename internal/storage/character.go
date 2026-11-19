package storage

import (
	"context"
	"strings"
	"time"
)

// FirstObjectID is the first object ID an IDAllocator hands out. It matches
// L2J, which keeps lower IDs for static objects such as NPCs and doors.
const FirstObjectID int32 = 0x10000000

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

// Item is one item a character owns.
type Item struct {
	ObjectID int32 // unique across characters and items, from an IDAllocator
	ItemID   int32 // item template ID
	Count    int64
	Enchant  int32         // enchant level
	Slot     PaperdollSlot // PaperdollNone if not equipped
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

	CurHP, MaxHP float64
	CurMP, MaxMP float64
	CurCP, MaxCP float64

	X, Y, Z int32
	Heading int32

	AccessLevel AccessLevel
	DeleteTime  time.Time // when the character will be deleted; zero if not deleting
	CreatedAt   time.Time
	LastAccess  time.Time // zero if the character never entered the world

	// Items are stored inside the character, so creating a character with
	// its starting items is one atomic write: a standalone MongoDB (as in
	// docker compose) has no transactions. Items may move to their own
	// collection later.
	Items []Item
}

// NameKey returns the form of a character name used to compare names. Names
// that differ only in letter case have the same key.
func NameKey(name string) string {
	return strings.ToLower(name)
}

// CharacterStore stores player characters. Names are unique and compared
// case-insensitively (see NameKey). Account names are compared exactly.
type CharacterStore interface {
	// Create stores c with its items. The caller assigns the object IDs of c
	// and its items from an IDAllocator. If c.CreatedAt is zero, Create sets
	// it to the current time. Create returns ErrExists if the name is taken,
	// ignoring case, or if the object ID is taken.
	Create(ctx context.Context, c *Character) error
	// ListByAccount returns the account's characters, oldest first.
	ListByAccount(ctx context.Context, account string) ([]Character, error)
	// CountByAccount returns how many characters the account has.
	CountByAccount(ctx context.Context, account string) (int, error)
	// NameExists reports whether a character has this name, ignoring case.
	NameExists(ctx context.Context, name string) (bool, error)
	// Get returns the character with this object ID or ErrNotFound.
	Get(ctx context.Context, objectID int32) (Character, error)
}

// IDAllocator hands out object IDs for characters and items. IDs are unique
// across both, and across restarts for persistent implementations. They
// start at FirstObjectID.
type IDAllocator interface {
	// NextID returns one new ID.
	NextID(ctx context.Context) (int32, error)
	// NextIDs returns n new IDs, for example for a character and its items.
	// n must be at least 1.
	NextIDs(ctx context.Context, n int) ([]int32, error)
}
