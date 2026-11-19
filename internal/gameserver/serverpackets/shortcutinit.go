package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpShortcutInit is the opcode of ShortcutInit (Mobius SHORT_CUT_INIT).
const OpShortcutInit = 0x45

// shortcutsPerPage is the number of slots on one page of the shortcut bar.
const shortcutsPerPage = 12

// ShortcutType is what a shortcut points to (the ordinals of Mobius
// ShortcutType).
type ShortcutType int32

// Shortcut types.
const (
	ShortcutNone   ShortcutType = 0
	ShortcutItem   ShortcutType = 1
	ShortcutSkill  ShortcutType = 2
	ShortcutAction ShortcutType = 3
	ShortcutMacro  ShortcutType = 4
	ShortcutRecipe ShortcutType = 5
)

// Shortcut is one slot of the shortcut bar.
type Shortcut struct {
	Type ShortcutType
	Slot int32 // 0-11 on the page
	Page int32 // 0-9
	// ID is the item object ID, skill ID, action ID, macro ID or recipe
	// ID, depending on Type.
	ID    int32
	Level int32 // the skill level of a skill shortcut
}

// NewShortcutInitPacket builds ShortcutInit (Mobius ShortcutInit.writeImpl),
// the whole shortcut bar. An empty bar is valid.
func NewShortcutInitPacket(shortcuts []Shortcut) []byte {
	w := packets.NewWriter(5 + len(shortcuts)*34)
	w.WriteUint8(OpShortcutInit)
	putCount32(w, len(shortcuts))
	for _, sc := range shortcuts {
		w.WriteInt32(int32(sc.Type))
		w.WriteInt32(sc.Slot + sc.Page*shortcutsPerPage)
		switch sc.Type {
		case ShortcutItem:
			w.WriteInt32(sc.ID)
			w.WriteInt32(1)
			w.WriteInt32(-1)
			w.WriteInt32(0)
			w.WriteInt32(0)
			w.WriteUint16(0)
			w.WriteUint16(0)
		case ShortcutSkill:
			w.WriteInt32(sc.ID)
			w.WriteInt32(sc.Level)
			w.WriteUint8(0) // C5 addition
			w.WriteInt32(1)
		case ShortcutAction, ShortcutMacro, ShortcutRecipe:
			w.WriteInt32(sc.ID)
			w.WriteInt32(1)
		}
		// ShortcutNone has nothing after the slot, as in Mobius.
	}
	return w.Bytes()
}
