package serverpackets

import (
	"math"
	"time"

	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
)

// selectionSlots is the order of the equipment lists in CharSelectionInfo.
// Mobius writes the right hand a second time where older clients had the
// two-handed slot, and leaves out the face slot.
var selectionSlots = [storage.PaperdollTotalSlots]storage.PaperdollSlot{
	storage.PaperdollUnder,
	storage.PaperdollREar,
	storage.PaperdollLEar,
	storage.PaperdollNeck,
	storage.PaperdollRFinger,
	storage.PaperdollLFinger,
	storage.PaperdollHead,
	storage.PaperdollRHand,
	storage.PaperdollLHand,
	storage.PaperdollGloves,
	storage.PaperdollChest,
	storage.PaperdollLegs,
	storage.PaperdollFeet,
	storage.PaperdollCloak,
	storage.PaperdollRHand,
	storage.PaperdollHair,
	storage.PaperdollHair2,
}

// maxEnchantEffect caps the weapon enchant level shown in the list.
const maxEnchantEffect = 127

// NewCharSelectionInfoPacket builds CharSelectionInfo, the character list
// of an account, in the order of chars. sessionID is the PlayOK1 session
// key, which Mobius sends as the account ID. now is used for the time left
// until a character is deleted.
//
// The character played or created last (the latest LastAccess) is marked
// active. A list where no character has a LastAccess has no active one.
func NewCharSelectionInfoPacket(account string, sessionID int32, chars []storage.Character, now time.Time) []byte {
	w := packets.NewWriter(5 + len(chars)*400)
	w.WriteUint8(OpCharSelectionInfo)
	w.WriteInt32(int32(len(chars))) //nolint:gosec // at most a few characters per account
	active := activeIndex(chars)
	for i, c := range chars {
		writeCharacter(w, account, sessionID, c, i == active, now)
	}
	return w.Bytes()
}

// activeIndex returns the index of the character with the latest
// LastAccess, or -1 if none has one. Ties go to the first, as in Mobius.
func activeIndex(chars []storage.Character) int {
	active := -1
	var last time.Time
	for i, c := range chars {
		if c.LastAccess.After(last) && !c.LastAccess.IsZero() {
			last = c.LastAccess
			active = i
		}
	}
	return active
}

func writeCharacter(w *packets.Writer, account string, sessionID int32, c storage.Character, active bool, now time.Time) {
	var objectIDs, itemIDs [storage.PaperdollTotalSlots]int32
	var enchant int32
	for _, it := range c.Items {
		if !it.Slot.Equipped() {
			continue
		}
		objectIDs[it.Slot] = it.ObjectID
		itemIDs[it.Slot] = it.ItemID
		if it.Slot == storage.PaperdollRHand {
			enchant = it.Enchant
		}
	}
	baseClass := c.BaseClassID
	if baseClass == 0 && c.ClassID > 0 {
		baseClass = c.ClassID
	}

	w.WriteString(c.Name)
	w.WriteInt32(c.ObjectID)
	w.WriteString(account)
	w.WriteInt32(sessionID)
	w.WriteInt32(c.ClanID)
	w.WriteInt32(0) // builder level
	w.WriteInt32(c.Sex)
	w.WriteInt32(c.Race)
	w.WriteInt32(baseClass)
	w.WriteInt32(1) // game server name
	w.WriteInt32(0) // x
	w.WriteInt32(0) // y
	w.WriteInt32(0) // z
	w.WriteFloat64(c.CurHP)
	w.WriteFloat64(c.CurMP)
	w.WriteInt32(c.SP)
	w.WriteUint64(uint64(c.Exp)) //nolint:gosec // same bits as the Java long
	w.WriteInt32(c.Level)
	w.WriteInt32(c.Karma)
	for range 9 {
		w.WriteInt32(0)
	}
	for _, slot := range selectionSlots {
		w.WriteInt32(objectIDs[slot])
	}
	for _, slot := range selectionSlots {
		w.WriteInt32(itemIDs[slot])
	}
	w.WriteInt32(c.HairStyle)
	w.WriteInt32(c.HairColor)
	w.WriteInt32(c.Face)
	w.WriteFloat64(c.MaxHP)
	w.WriteFloat64(c.MaxMP)
	w.WriteInt32(deleteTimer(c, now))
	w.WriteInt32(c.ClassID)
	if active {
		w.WriteInt32(1)
	} else {
		w.WriteInt32(0)
	}
	w.WriteUint8(uint8(min(max(enchant, 0), maxEnchantEffect))) //nolint:gosec // clamped to 0-127
	w.WriteInt32(0)                                             // augmentation ID: augmentations are not stored yet
}

// deleteTimer returns the delete timer field: -1 for a banned character,
// the seconds left until deletion, or 0 when it is not being deleted.
// Mobius removes characters whose time has passed before listing them; L2Go
// does not delete yet and shows them as not being deleted.
func deleteTimer(c storage.Character, now time.Time) int32 {
	if c.AccessLevel < storage.AccessLevelPlayer {
		return -1
	}
	if c.DeleteTime.IsZero() {
		return 0
	}
	secs := int64(c.DeleteTime.Sub(now) / time.Second)
	return int32(min(max(secs, 0), math.MaxInt32)) //nolint:gosec // clamped to the int32 range
}
